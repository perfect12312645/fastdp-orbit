// Package orchestrator 工作流编排引擎
//
// 整体架构：
//
//	WorkflowExecution（一次工作流执行）
//	  └── StageGroup（阶段组，按 Order 从左到右顺序执行，组内支持 parallel/sequential 模式）
//	        └── Stage（阶段，组内按 Order 从上到下顺序执行）
//	              └── Task（任务，通过 gRPC 调用 Agent 执行，支持重试、条件执行、后置钩子）
//
// 执行层级关系：
//
//	Workflow 定义 → WorkflowExecution（执行实例）
//	  └── WorkflowStageGroupExecution（阶段组执行记录）
//	        └── WorkflowStageExecution（阶段执行记录）
//	              └── WorkflowTaskExecution（任务执行记录，每台机器一条）
//
// 状态流转：
//
//	Execution:  running → success / failed / paused / cancelled
//	StageGroup: pending → running → success / failed / paused / skipped
//	Stage:      pending → running → success / failed / skipped
//	Task:       pending → running → success / failed / skipped
//
// 核心设计：goroutine 常驻 + channel 阻塞模型
//
//	run() 启动后 goroutine 常驻，按顺序执行每个 StageGroup。
//	当 stage 失败或用户暂停时，goroutine 不退出，而是创建 pausing channel 阻塞在 waitForResume() 上。
//	用户点击"继续"或"重试"时，关闭 channel 唤醒 goroutine，goroutine 复用旧 ctx 继续执行。
//	剩余未执行的 stages/groups 始终保持 pending 状态，不产生 skipped 记录。
//
// 核心操作：
//
//	Execute(execution)    - 启动工作流，创建 goroutine 异步执行 run() 主循环
//	Pause(executionID)    - 暂停：创建 pausing channel，标记 execution=paused + stageGroup=paused
//	                     goroutine 检测 pausing channel 后调用 waitForResume() 阻塞等待
//	                     当前 stage 继续执行完毕（task 的 gRPC 超时由 task 自己控制）
//	                     剩余 stages/groups 保持 pending 不动，ctx 保留不销毁
//	Cancel(executionID)   - 终止：唯一销毁 ctx 的操作，关闭 pausing channel 唤醒可能阻塞的 goroutine
//	                     标记 execution=cancelled，pending 的 group/stage 标记 skipped
//	                     当前 task 的 gRPC 调用立即失败，stage 立即停止，goroutine 退出
//	                     取消后只能通过 RetryExecution 重新全量执行
//	Resume(executionID)   - 继续：关闭 pausing channel 唤醒 goroutine，重置 paused group=running
//	                     goroutine 被唤醒后复用旧 ctx，继续执行下一个 stageGroup
//	RetryStage(executionID, stageExecutionID) - 重试：关闭 pausing channel 唤醒 goroutine
//	                     删除失败 stage 的记录，重置 failed group=running
//	                     goroutine 复用旧 ctx 重新执行整个 group（已成功的 stage 自动跳过）
//	                     仅在 goroutine 存活时可用，服务重启后应使用 RetryExecution
//	RetryExecution(executionID) - 重新执行整个工作流（创建新 ctx 和 goroutine）
//	Recover()             - 服务重启恢复：扫描 DB 中 running 状态的记录，全部标记 failed
//	                     paused 状态保持不变，用户可手动 Resume
//
// Pause vs Failure vs Cancel 的关键区别：
//
//	Pause:    创建 pausing channel → executeStagesSequential/Parallel 检测到 → 返回 paused=true
//	          goroutine 标记 group=paused → waitForResume() 阻塞 → 不退出
//	          效果：当前 stage 完整执行完毕，后续 stage 不再开始，goroutine 等待唤醒
//
//	Failure:  stage 失败 → executeStagesSequential/Parallel 返回 failed=true
//	          goroutine 标记 group=failed → 创建 pausing channel → waitForResume() 阻塞
//	          效果：与 Pause 一致，goroutine 阻塞等待用户重试，ctx 保留
//
//	Cancel:   ctx 取消 → executeTask 内的 gRPC 调用立即失败（reqCtx 派生自 ctx）
//	          task 被标为 skipped → executeStage 检测 ctx → stage 标为 failed → return true
//	          goroutine 检测 IsCancelled → 直接 return 退出
//	          效果：当前 task 的 gRPC 调用立即中断，stage 立即停止，goroutine 退出
//
// 并发控制：
//
//	running   map[uint]*cancelContext  - 追踪 goroutine 的 cancel 函数
//	cancelled map[uint]bool           - 标记已被 Cancel 的 execution
//	pausing   map[uint]chan struct{}   - 暂停/失败等待通道，goroutine 阻塞于此
//	ctxMap    map[uint]context.Context - 当前有效的 context（仅 Cancel 时销毁）
//	lastOp    map[uint]time.Time       - 操作频率限制，防频繁点击
//	sync.RWMutex                       - 保护所有 map 的并发访问
//
// goroutine 生命周期：
//
//	Execute → goroutine 启动 → 执行 stages → 失败/暂停 → waitForResume() 阻塞
//	                                                            ↓
//	                                              Resume/RetryStage 关闭 channel
//	                                                            ↓
//	                                              goroutine 获取新 ctx → 继续执行
//	                                                            ↓
//	                                              全部成功 → goroutine 退出 → defer 清理 map
package orchestrator

import (
	"context"
	"encoding/json"
	"fastdp-orbit/backend/models/machine"
	"fastdp-orbit/backend/models/workflow"
	"fastdp-orbit/backend/pkg/logger"
	"fastdp-orbit/backend/pkg/utils"
	agentpb "fastdp-orbit/backend/proto/agent"
	"fastdp-orbit/backend/server/grpc"
	"fmt"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

// EventListener 状态变更事件回调（避免循环依赖）
type EventListener interface {
	OnExecutionStatus(executionID uint, status string, errMsg string)
	OnGroupStatus(executionID uint, groupID uint, status string)
	OnStageStatus(executionID uint, stageID uint, status string)
	OnTaskStatus(executionID uint, taskID uint, taskRef int, taskName string, status string, host string, output string, errStr string, trace string, errorCode int32, changed bool, duration int64)
}

// Orchestrator 工作流执行引擎
type Orchestrator struct {
	mu         sync.RWMutex
	db         *gorm.DB
	pool       *grpc.AgentConnPool
	listener   EventListener
	running    map[uint]*cancelContext  // executionID -> cancel
	cancelled  map[uint]bool            // executionID -> 是否被用户终止
	pausing    map[uint]chan struct{}   // executionID -> 暂停/失败等待通道，goroutine 阻塞于此
	lastOp     map[uint]time.Time       // executionID -> 上次操作时间，防频繁点击
	ctxMap     map[uint]context.Context // executionID -> 当前有效的 context（Resume/RetryStage 会更新）
	serverIP   string
	serverPort string
	protocol   string
}

type cancelContext struct {
	cancel context.CancelFunc
}

// NewOrchestrator 创建执行引擎
func NewOrchestrator(db *gorm.DB, pool *grpc.AgentConnPool, ServerListenAddr string, Protocol string) *Orchestrator {
	return &Orchestrator{
		db:         db,
		pool:       pool,
		running:    make(map[uint]*cancelContext),
		cancelled:  make(map[uint]bool),
		pausing:    make(map[uint]chan struct{}),
		lastOp:     make(map[uint]time.Time),
		ctxMap:     make(map[uint]context.Context),
		serverIP:   strings.Split(ServerListenAddr, ":")[0],
		serverPort: strings.Split(ServerListenAddr, ":")[1],
		protocol:   Protocol,
	}
}

// SetEventListener 设置事件监听器
func (o *Orchestrator) SetEventListener(listener EventListener) {
	o.listener = listener
}

// GetServerIP 获取服务器IP
func (o *Orchestrator) GetServerIP() string {
	return o.serverIP
}

// GetServerPort 获取服务器端口
func (o *Orchestrator) GetServerPort() string {
	return o.serverPort
}

// GetProtocol 获取协议
func (o *Orchestrator) GetProtocol() string {
	return o.protocol
}

// emit 安全地触发事件
func (o *Orchestrator) emit(executionID uint, status string, errMsg string) {
	if o.listener != nil {
		o.listener.OnExecutionStatus(executionID, status, errMsg)
	}
}

func (o *Orchestrator) emitGroup(executionID uint, groupID uint, status string) {
	if o.listener != nil {
		o.listener.OnGroupStatus(executionID, groupID, status)
	}
}

func (o *Orchestrator) emitStage(executionID uint, stageID uint, status string) {
	if o.listener != nil {
		o.listener.OnStageStatus(executionID, stageID, status)
	}
}

func (o *Orchestrator) emitTask(executionID uint, taskID uint, taskRef int, taskName string, status string, host string, output string, errStr string, trace string, errorCode int32, changed bool, duration int64) {
	if o.listener != nil {
		o.listener.OnTaskStatus(executionID, taskID, taskRef, taskName, status, host, output, errStr, trace, errorCode, changed, duration)
	}
}

// CreateAndExecute 创建执行记录并启动工作流
func (o *Orchestrator) CreateAndExecute(execution *workflow.WorkflowExecution) error {
	// 写入数据库获取 ID
	if err := o.db.Create(execution).Error; err != nil {
		return fmt.Errorf("创建执行记录失败: %v", err)
	}
	// 直接启动，跳过 Execute 中的 DB 重复检查（刚创建的记录必然是 running 状态）
	ctx, cancel := context.WithCancel(context.Background())
	o.mu.Lock()
	if _, exists := o.running[execution.ID]; exists {
		o.mu.Unlock()
		cancel()
		return fmt.Errorf("执行实例 %d 已在运行中，禁止重复提交", execution.ID)
	}
	o.running[execution.ID] = &cancelContext{cancel: cancel}
	o.ctxMap[execution.ID] = ctx
	o.mu.Unlock()
	go o.run(ctx, execution)
	return nil
}

// Pause 暂停执行（当前 stage 继续完成，不启动下一个 stage）
func (o *Orchestrator) Pause(executionID uint) error {
	if err := o.checkRateLimit(executionID, 1*time.Second); err != nil {
		return err
	}

	o.mu.RLock()
	_, ok := o.running[executionID]
	o.mu.RUnlock()

	if !ok {
		return fmt.Errorf("执行实例 %d 不在运行中", executionID)
	}

	// 创建暂停通道（不取消 ctx，让当前 stage 继续执行）
	o.mu.Lock()
	o.pausing[executionID] = make(chan struct{})
	o.mu.Unlock()

	// 更新 execution 状态为 paused
	o.db.Model(&workflow.WorkflowExecution{}).Where("id = ?", executionID).
		Updates(map[string]interface{}{
			"status": "paused",
			"error":  "用户暂停",
		})
	o.emit(executionID, "paused", "")

	// 标记当前运行中的 stage group 为 paused
	o.db.Model(&workflow.WorkflowStageGroupExecution{}).
		Where("execution_id = ? AND status = ?", executionID, "running").
		Update("status", "paused")

	return nil
}

// Cancel 终止执行（立即停止当前 stage）
func (o *Orchestrator) Cancel(executionID uint) error {
	if err := o.checkRateLimit(executionID, 1*time.Second); err != nil {
		return err
	}

	o.mu.RLock()
	cc, ok := o.running[executionID]
	o.mu.RUnlock()

	// 如果 goroutine 已不存在（服务重启导致），直接更新 DB 状态
	if !ok {
		var exec workflow.WorkflowExecution
		if err := o.db.First(&exec, executionID).Error; err != nil {
			return fmt.Errorf("执行实例 %d 不存在", executionID)
		}
		if exec.Status != "running" && exec.Status != "paused" {
			return fmt.Errorf("执行实例 %d 状态为 %s，无法终止", executionID, exec.Status)
		}

		now := time.Now()
		o.db.Model(&workflow.WorkflowExecution{}).Where("id = ?", executionID).
			Updates(map[string]interface{}{
				"status":      "cancelled",
				"error":       "用户终止（协程已消失）",
				"finished_at": now,
			})
		o.emit(executionID, "cancelled", "")

		// 跳过所有 pending/running 的记录
		o.db.Model(&workflow.WorkflowStageGroupExecution{}).
			Where("execution_id = ? AND status IN ?", executionID, []string{"pending", "running"}).
			Updates(map[string]interface{}{"status": "skipped", "error": "用户终止"})
		o.db.Model(&workflow.WorkflowStageExecution{}).
			Where("stage_group_execution_id IN (SELECT id FROM workflow_stage_group_executions WHERE execution_id = ?) AND status IN ?", executionID, []string{"pending", "running"}).
			Updates(map[string]interface{}{"status": "skipped", "error": "用户终止"})
		o.db.Model(&workflow.WorkflowTaskExecution{}).
			Where("stage_execution_id IN (SELECT id FROM workflow_stage_executions WHERE stage_group_execution_id IN (SELECT id FROM workflow_stage_group_executions WHERE execution_id = ?)) AND status IN ?", executionID, []string{"pending", "running"}).
			Updates(map[string]interface{}{"status": "skipped", "error": "用户终止"})

		return nil
	}

	// goroutine 存在，正常终止流程
	o.mu.Lock()
	o.cancelled[executionID] = true
	if ch, exists := o.pausing[executionID]; exists {
		close(ch)
		delete(o.pausing, executionID)
	}
	o.mu.Unlock()

	cc.cancel()

	now := time.Now()
	o.db.Model(&workflow.WorkflowExecution{}).Where("id = ?", executionID).
		Updates(map[string]interface{}{
			"status":      "cancelled",
			"error":       "用户终止",
			"finished_at": now,
		})
	o.emit(executionID, "cancelled", "")

	o.db.Model(&workflow.WorkflowStageGroupExecution{}).
		Where("execution_id = ? AND status = ?", executionID, "pending").
		Update("status", "skipped")
	o.db.Model(&workflow.WorkflowStageExecution{}).
		Where("stage_group_execution_id IN (SELECT id FROM workflow_stage_group_executions WHERE execution_id = ?) AND status = ?",
			executionID, "pending").
		Update("status", "skipped")

	return nil
}

// GetRunning 获取运行中的执行实例
func (o *Orchestrator) GetRunning(executionID uint) bool {
	o.mu.RLock()
	defer o.mu.RUnlock()
	_, ok := o.running[executionID]
	return ok
}

// IsCancelled 检查执行实例是否已被用户终止
func (o *Orchestrator) IsCancelled(executionID uint) bool {
	o.mu.RLock()
	defer o.mu.RUnlock()
	return o.cancelled[executionID]
}

// waitForResume 阻塞等待用户操作（Resume/RetryStage/Cancel）
// 返回后 goroutine 应重新获取 ctx（因为 Cancel 可能已替换 ctx）
func (o *Orchestrator) waitForResume(executionID uint) {
	o.mu.RLock()
	ch, ok := o.pausing[executionID]
	o.mu.RUnlock()
	if !ok {
		return
	}
	<-ch // 阻塞直到 channel 被关闭
}

// checkRateLimit 检查操作频率限制，防止用户频繁点击
// 返回 nil 表示允许操作，否则返回 error
func (o *Orchestrator) checkRateLimit(executionID uint, cooldown time.Duration) error {
	o.mu.Lock()
	defer o.mu.Unlock()
	if last, exists := o.lastOp[executionID]; exists {
		if time.Since(last) < cooldown {
			return fmt.Errorf("操作过于频繁，请稍后重试")
		}
	}
	o.lastOp[executionID] = time.Now()
	return nil
}

// Recover 服务重启后恢复中断的执行（将 DB 中 running 状态的记录标记为 failed，等待用户重试）
func (o *Orchestrator) Recover() {
	// 查找所有状态为 running 的 execution（重启前 goroutine 已死亡）
	var runningExecutions []workflow.WorkflowExecution
	o.db.Where("status = ?", "running").Find(&runningExecutions)

	for _, exec := range runningExecutions {
		// 标记 execution 为 failed，等待用户重试
		o.db.Model(&exec).Updates(map[string]interface{}{
			"status":      "failed",
			"error":       "服务重启导致中断",
			"finished_at": time.Now(),
		})

		// 将 running 状态的 stage_group_executions 标记为 failed
		o.db.Model(&workflow.WorkflowStageGroupExecution{}).
			Where("execution_id = ? AND status = ?", exec.ID, "running").
			Updates(map[string]interface{}{
				"status":      "failed",
				"error":       "服务重启导致中断",
				"finished_at": time.Now(),
			})

		// 将 running 状态的 stage_executions 标记为 failed
		o.db.Model(&workflow.WorkflowStageExecution{}).
			Where("stage_group_execution_id IN (SELECT id FROM workflow_stage_group_executions WHERE execution_id = ?) AND status = ?",
				exec.ID, "running").
			Updates(map[string]interface{}{
				"status":      "failed",
				"error":       "服务重启导致中断",
				"finished_at": time.Now(),
			})

		// 将 running 状态的 task_executions 标记为 failed
		o.db.Model(&workflow.WorkflowTaskExecution{}).
			Where("stage_execution_id IN (SELECT id FROM workflow_stage_executions WHERE stage_group_execution_id IN (SELECT id FROM workflow_stage_group_executions WHERE execution_id = ?)) AND status = ?",
				exec.ID, "running").
			Updates(map[string]interface{}{
				"status":      "failed",
				"error":       "服务重启导致中断",
				"finished_at": time.Now(),
			})

		logger.Warn("恢复中断的工作流执行",
			zap.Uint("execution_id", exec.ID),
			zap.Uint("workflow_id", exec.WorkflowID),
			zap.String("workflow_name", exec.Workflow.Name),
		)
	}
}

// Resume 继续执行暂停的工作流（唤醒阻塞的 goroutine，从下一个 stage 开始）
func (o *Orchestrator) Resume(executionID uint) error {
	if err := o.checkRateLimit(executionID, 1*time.Second); err != nil {
		return err
	}

	// 检查 execution 状态 = "paused"
	var execution workflow.WorkflowExecution
	if err := o.db.First(&execution, executionID).Error; err != nil {
		return fmt.Errorf("执行实例 %d 不存在", executionID)
	}
	if execution.Status != "paused" {
		return fmt.Errorf("执行实例 %d 不在暂停状态，当前状态: %s", executionID, execution.Status)
	}

	// 重置 paused 的 stage group → running
	o.db.Model(&workflow.WorkflowStageGroupExecution{}).
		Where("execution_id = ? AND status = ?", executionID, "paused").
		Updates(map[string]interface{}{
			"status":      "running",
			"error":       "",
			"finished_at": nil,
		})

	// 更新 execution 状态 = running
	o.db.Model(&execution).Updates(map[string]interface{}{
		"status":      "running",
		"error":       "",
		"finished_at": nil,
	})
	o.emit(executionID, "running", "")

	// 检查 goroutine 是否还活着
	o.mu.RLock()
	_, goroutineAlive := o.running[executionID]
	o.mu.RUnlock()

	o.mu.Lock()
	o.cancelled[executionID] = false

	if goroutineAlive {
		// goroutine 在 waitForResume 阻塞，关闭 channel 唤醒它
		// Pause 没有取消 ctx，旧 ctx 仍然有效，goroutine 通过 getCtx() 拿到旧 ctx 继续执行
		if ch, exists := o.pausing[executionID]; exists {
			close(ch)
			delete(o.pausing, executionID)
		}
		o.mu.Unlock()
	} else {
		// goroutine 已死（服务重启导致），重新启动
		newCtx, newCancel := context.WithCancel(context.Background())
		o.running[executionID] = &cancelContext{cancel: newCancel}
		o.ctxMap[executionID] = newCtx
		o.mu.Unlock()
		go o.run(newCtx, &execution)
	}

	logger.Info("工作流执行恢复", zap.Uint("execution_id", executionID))
	return nil
}

// RetryStage 重试失败的 stage（关闭 pausing channel 唤醒 goroutine，复用旧 ctx 重新执行失败 group）
// 仅在 goroutine 存活时可用（服务重启后应使用 RetryExecution）
func (o *Orchestrator) RetryStage(executionID uint, stageExecutionID uint) error {
	if err := o.checkRateLimit(executionID, 3*time.Second); err != nil {
		return err
	}

	// 检查 execution 存在且状态允许重试
	var execution workflow.WorkflowExecution
	if err := o.db.First(&execution, executionID).Error; err != nil {
		return fmt.Errorf("执行实例 %d 不存在", executionID)
	}
	if execution.Status == "cancelled" || execution.Status == "success" || execution.Status == "failed" {
		return fmt.Errorf("执行实例 %d 状态为 %s，无法重试（cancelled/success/failed 请使用 RetryExecution）", executionID, execution.Status)
	}

	// 找到失败的 stage execution
	var failedStageExec workflow.WorkflowStageExecution
	if err := o.db.First(&failedStageExec, stageExecutionID).Error; err != nil {
		return fmt.Errorf("阶段执行记录 %d 不存在", stageExecutionID)
	}
	if failedStageExec.Status != "failed" {
		return fmt.Errorf("阶段执行记录 %d 状态不是 failed，当前状态: %s", stageExecutionID, failedStageExec.Status)
	}

	// 删除失败 stage 的 task_executions 和 stage_execution 本身（executeStage 会重新创建）
	o.db.Where("stage_execution_id = ?", stageExecutionID).Delete(&workflow.WorkflowTaskExecution{})
	o.db.Delete(&failedStageExec)

	// 重置 failed 的 stage group → running（run 循环会重新执行整个 group）
	o.db.Model(&workflow.WorkflowStageGroupExecution{}).
		Where("execution_id = ? AND id = ? AND status = ?", executionID, failedStageExec.StageGroupExecutionID, "failed").
		Updates(map[string]interface{}{
			"status":      "running",
			"error":       "",
			"finished_at": nil,
		})

	// 关闭 pausing channel 唤醒 goroutine（不创建新 ctx，旧 ctx 仍然有效）
	o.mu.Lock()
	o.cancelled[executionID] = false
	if ch, exists := o.pausing[executionID]; exists {
		close(ch)
		delete(o.pausing, executionID)
	}
	o.mu.Unlock()

	logger.Info("工作流 stage 重试",
		zap.Uint("execution_id", executionID),
		zap.Uint("stage_execution_id", stageExecutionID),
	)
	return nil
}

// RetryExecution 重新执行整个工作流（从第一个失败的 stage group 开始）
func (o *Orchestrator) RetryExecution(executionID uint) error {
	if err := o.checkRateLimit(executionID, 3*time.Second); err != nil {
		return err
	}

	// 检查 execution 状态 = "failed"
	var execution workflow.WorkflowExecution
	if err := o.db.First(&execution, executionID).Error; err != nil {
		return fmt.Errorf("执行实例 %d 不存在", executionID)
	}
	if execution.Status != "failed" {
		return fmt.Errorf("执行实例 %d 状态不是 failed，当前状态: %s", executionID, execution.Status)
	}

	// 检查是否已在运行
	if _, exists := o.running[executionID]; exists {
		return fmt.Errorf("执行实例 %d 已在运行中", executionID)
	}

	// 找到第一个 failed 的 stage group execution
	var failedGroup workflow.WorkflowStageGroupExecution
	if err := o.db.Where("execution_id = ? AND status = ?", executionID, "failed").
		Order("id ASC").First(&failedGroup).Error; err != nil {
		return fmt.Errorf("执行实例 %d 没有失败的 stage group", executionID)
	}

	// 找到该 group 内第一个 failed 的 stage execution
	var failedStage workflow.WorkflowStageExecution
	if err := o.db.Where("stage_group_execution_id = ? AND status = ?", failedGroup.ID, "failed").
		Order("id ASC").First(&failedStage).Error; err != nil {
		// group 是 failed 但没有具体 failed stage，可能是 group 加载失败（FailExecution 场景）
		// 直接重置 group 即可
		o.db.Model(&failedGroup).Updates(map[string]interface{}{
			"status":      "running",
			"error":       "",
			"finished_at": nil,
		})
	} else {
		// 删除失败 stage 的 task_executions
		o.db.Where("stage_execution_id = ?", failedStage.ID).Delete(&workflow.WorkflowTaskExecution{})

		// 删除失败的 stage_execution 本身（executeStage 会重新创建）
		o.db.Delete(&failedStage)

		// 重置同 group 内后续 skipped 的 stages
		o.db.Model(&workflow.WorkflowStageExecution{}).
			Where("stage_group_execution_id = ? AND status = ?", failedStage.StageGroupExecutionID, "skipped").
			Update("status", "pending")

		// 重置 failed 的 stage group → running
		o.db.Model(&failedGroup).Updates(map[string]interface{}{
			"status":      "running",
			"error":       "",
			"finished_at": nil,
		})
	}

	// 跳过 failed group 之前所有 pending/skipped 的 stage_group（已完成的保持 success，未执行的跳过）
	o.db.Model(&workflow.WorkflowStageGroupExecution{}).
		Where("execution_id = ? AND id < ? AND status IN ?", executionID, failedGroup.ID, []string{"pending", "skipped"}).
		Update("status", "skipped")

	// 更新 execution 状态 = running
	now := time.Now()
	o.db.Model(&execution).Updates(map[string]interface{}{
		"status":      "running",
		"error":       "",
		"finished_at": nil,
		"started_at":  now,
	})

	// 发出 running 事件，让前端 SSE 更新状态
	o.emit(executionID, "running", "")
	o.emitGroup(executionID, failedGroup.GroupID, "running")

	// 启动 goroutine
	o.mu.Lock()
	o.cancelled[executionID] = false
	newCtx, newCancel := context.WithCancel(context.Background())
	o.running[executionID] = &cancelContext{cancel: newCancel}
	o.ctxMap[executionID] = newCtx
	o.mu.Unlock()

	go o.run(newCtx, &execution)

	logger.Info("工作流整体重试", zap.Uint("execution_id", executionID))
	return nil
}

// ExecuteTaskForStage 为单阶段执行服务暴露的任务执行方法
// 复用 executeTask 的核心逻辑，但写入 StageTaskExecution 表
func (o *Orchestrator) ExecuteTaskForStage(ctx context.Context, taskExec *workflow.StageTaskExecution, task *workflow.StageTask, m *machine.Machine, hooks []workflow.WorkflowHook, params map[string]string, globalVars map[string]interface{}, groupVars map[string]interface{}, groupsMap map[string]interface{}, loopItem interface{}, registeredVars map[string]map[string]map[string]interface{}, registeredVarsMu *sync.RWMutex) {
	logger.Info("任务执行开始", zap.String("task_id", fmt.Sprintf("ref-%d", task.Ref)), zap.String("machine", fmt.Sprintf("%s:%d", m.IP, m.Port)))
	host := fmt.Sprintf("%s:%d", m.IP, m.Port)

	templateVars := map[string]interface{}{
		"Machine":        MachineToMap(m),
		"GlobalVariable": globalVars,
		"Group":          groupVars,
		"Groups":         groupsMap,
		"Register":       registeredVars,
		"Server": map[string]interface{}{
			"ip":       o.serverIP,
			"port":     o.serverPort,
			"protocol": o.protocol,
		},
	}
	if loopItem != nil {
		templateVars["item"] = loopItem
	}

	// 检查 when 条件
	if task.When != "" {
		run, err := evaluateWhen(task.When, templateVars)
		if err != nil {
			taskExec.Status = "failed"
			taskExec.Error = fmt.Sprintf("when 条件解析失败: %v", err)
			return
		}
		if !run {
			taskExec.Status = "skipped"
			taskExec.Output = "条件不满足，跳过执行"
			return
		}
	}

	// 渲染参数
	renderedParams := make(map[string]string)
	for k, v := range params {
		rendered, err := RenderTemplate(v, templateVars)
		if err != nil {
			renderedParams[k] = v
		} else {
			renderedParams[k] = rendered
		}
	}

	// copy 模块自动计算 server 侧 src 文件的 MD5
	if task.Module == "copy" {
		if srcPath, ok := renderedParams["src"]; ok && srcPath != "" {
			if md5Val, err := utils.FileMD5(srcPath); err == nil {
				renderedParams["md5"] = md5Val
			} else {
				logger.Warn("copy模块自动计算MD5失败", zap.String("src", srcPath), zap.Error(err))
			}
		}
	}

	// template 模块：如果指定了 src（模板名称），从 DB 读取模板内容并渲染
	if task.Module == "template" {
		if srcVal, ok := renderedParams["src"]; ok && srcVal != "" && renderedParams["content"] == "" {
			var tpl workflow.WorkflowTemplate
			if err := o.db.Where("name = ?", srcVal).First(&tpl).Error; err == nil {
				// 渲染模板内容
				rendered, err := RenderTemplate(tpl.Content, templateVars)
				if err != nil {
					logger.Warn("template模块渲染模板失败", zap.String("src", srcVal), zap.Error(err))
					taskExec.Status = "failed"
					taskExec.Error = fmt.Sprintf("渲染模板失败: %v", err)
					return
				}
				renderedParams["content"] = rendered
				delete(renderedParams, "src") // 不传 src 给 agent
			} else {
				taskExec.Status = "failed"
				taskExec.Error = fmt.Sprintf("模板「%s」不存在", srcVal)
				return
			}
		}
	}

	maxRetries := task.Retries
	if maxRetries < 0 {
		maxRetries = 0
	}

	var lastErr string
	var hasAgentResponse bool
	var lastChanged bool
	for attempt := 0; attempt <= maxRetries; attempt++ {
		logger.Info("开始执行任务", zap.String("尝试次数", fmt.Sprintf("%d/%d", attempt+1, maxRetries+1)), zap.String("task_id", fmt.Sprintf("ref-%d", task.Ref)))

		if ctx.Err() != nil {
			taskExec.Status = "skipped"
			taskExec.Error = "执行被中断"
			return
		}

		if attempt > 0 && task.Delay > 0 {
			select {
			case <-ctx.Done():
				taskExec.Status = "skipped"
				taskExec.Error = "执行被中断"
				return
			case <-time.After(time.Duration(task.Delay) * time.Second):
			}
		}

		startTime := time.Now()

		conn, err := o.pool.GetConn(host)
		if err != nil {
			lastErr = fmt.Sprintf("连接Agent失败: %v", err)
			taskExec.DurationMs = time.Since(startTime).Milliseconds()
			continue
		}

		client := agentpb.NewAgentServiceClient(conn)
		var reqCtx context.Context
		var cancel context.CancelFunc
		if task.Timeout > 0 {
			reqCtx, cancel = context.WithTimeout(ctx, time.Duration(task.Timeout)*time.Second)
		} else {
			reqCtx, cancel = context.WithCancel(ctx)
		}

		resp, err := client.Exec(reqCtx, &agentpb.ExecRequest{
			MachineId:  host,
			Module:     task.Module,
			Parameters: renderedParams,
			TaskId:     fmt.Sprintf("ref-%d", task.Ref),
			Timeout:    int32(task.Timeout),
		})
		cancel()

		if resp != nil && resp.DurationMs > 0 {
			taskExec.DurationMs = resp.DurationMs
		} else {
			taskExec.DurationMs = time.Since(startTime).Milliseconds()
		}

		if err != nil {
			logger.Error("gRPC调用失败", zap.String("machine", host), zap.Error(err))
			lastErr = fmt.Sprintf("gRPC调用失败: %v", err)
			continue
		}

		if !resp.Success {
			logger.Error("任务执行失败", zap.String("machine", host), zap.String("task_id", fmt.Sprintf("ref-%d", task.Ref)))
			hasAgentResponse = true
			lastChanged = resp.Changed
			taskExec.Output = resp.Stdout // 失败时也保留输出（如脚本打印的错误信息）
			if resp.Error != nil {
				taskExec.ErrorCode = resp.Error.Code
				taskExec.Error = resp.Error.Message
				taskExec.Trace = resp.Error.Trace
			} else {
				taskExec.Error = "任务执行失败（无详细错误信息）"
			}
			lastErr = taskExec.Error
			continue
		}

		// 成功
		logger.Info("任务执行成功", zap.String("machine", host), zap.String("task_id", fmt.Sprintf("ref-%d", task.Ref)))
		taskExec.Status = "success"
		taskExec.Output = resp.Stdout
		taskExec.Changed = resp.Changed

		// 注册变量
		if task.Register != "" {
			registeredVarsMu.Lock()
			if registeredVars[task.Register] == nil {
				registeredVars[task.Register] = make(map[string]map[string]interface{})
			}
			registeredVars[task.Register][host] = map[string]interface{}{
				"stdout":  resp.Stdout,
				"changed": resp.Changed,
			}
			registeredVarsMu.Unlock()
		}

		// 执行后置钩子
		if task.Hooks != "" {
			shouldRunHooks := resp.Changed || taskExec.HookStatus == "failed"
			if shouldRunHooks {
				taskExec.HookStatus = "running"
				if err := o.executeHooks(ctx, host, task.Hooks, hooks, templateVars); err != nil {
					taskExec.HookStatus = "failed"
					taskExec.HookError = err.Error()
					taskExec.Status = "failed"
					taskExec.Error = fmt.Sprintf("后置钩子执行失败: %v", err)
					return
				}
				taskExec.HookStatus = "success"
			}
		}
		return
	}

	// 所有重试均失败
	logger.Error("所有重试均失败", zap.String("task_id", fmt.Sprintf("ref-%d", task.Ref)), zap.String("machine", fmt.Sprintf("%s:%d", m.IP, m.Port)), zap.String("错误", lastErr))
	taskExec.Status = "failed"
	taskExec.Error = lastErr

	if hasAgentResponse && task.IgnoreErrors {
		// 忽略错误：标记为成功，继续执行后续 task
		taskExec.Status = "success"
		taskExec.Output = fmt.Sprintf("[忽略错误] %s", lastErr)
		if task.Hooks != "" {
			shouldRun := lastChanged || taskExec.HookStatus == "failed"
			if shouldRun {
				taskExec.HookStatus = "running"
				if err := o.executeHooks(ctx, host, task.Hooks, hooks, templateVars); err != nil {
					taskExec.HookStatus = "failed"
					taskExec.HookError = err.Error()
				} else {
					taskExec.HookStatus = "success"
				}
			}
		}
	}
}

// run 执行工作流主循环
func (o *Orchestrator) run(ctx context.Context, execution *workflow.WorkflowExecution) {
	defer func() {
		o.mu.Lock()
		delete(o.running, execution.ID)
		delete(o.cancelled, execution.ID)
		delete(o.pausing, execution.ID)
		delete(o.ctxMap, execution.ID)
		o.mu.Unlock()
	}()

	// 加载 workflow 定义（含 stage_groups、stages、tasks、hooks）
	var wf workflow.Workflow
	if err := o.db.
		Preload("StageGroups.Stages.Tasks").
		Preload("Hooks").
		First(&wf, execution.WorkflowID).Error; err != nil {
		o.failExecution(execution, fmt.Sprintf("加载工作流定义失败: %v", err))
		return
	}

	logger.Info("工作流开始执行",
		zap.Uint("execution_id", execution.ID),
		zap.Uint("workflow_id", execution.WorkflowID),
		zap.String("workflow_name", wf.Name),
	)

	// 加载每个 stage 关联的机器分组和机器
	for i := range wf.StageGroups {
		for j := range wf.StageGroups[i].Stages {
			stage := &wf.StageGroups[i].Stages[j]
			if stage.MachineGroupID > 0 {
				var group machine.MachineGroup
				if err := o.db.Preload("Machines").First(&group, stage.MachineGroupID).Error; err != nil {
					o.failExecution(execution, fmt.Sprintf("加载机器分组失败: %v", err))
					return
				}
				stage.MachineGroup = &group
			}
		}
	}

	// 加载全局变量（整个工作流只查一次）
	var globalVarList []workflow.GlobalVariable
	if err := o.db.Find(&globalVarList).Error; err != nil {
		o.failExecution(execution, fmt.Sprintf("加载全局变量失败: %v", err))
		return
	}
	globalVars := make(map[string]interface{})
	for _, v := range globalVarList {
		globalVars[v.Key] = v.Value
	}

	// 加载所有机器分组（供 Groups 模板变量使用）
	var allMachineGroups []machine.MachineGroup
	if err := o.db.Preload("Machines").Find(&allMachineGroups).Error; err != nil {
		o.failExecution(execution, fmt.Sprintf("加载所有机器分组失败: %v", err))
		return
	}
	groupsMap := BuildGroupsMap(allMachineGroups)

	// 按 Order 排序 stage_groups
	groups := wf.StageGroups
	sort.Slice(groups, func(i, j int) bool {
		return groups[i].Order < groups[j].Order
	})

	now := time.Now()
	execution.StartedAt = now
	o.db.Model(execution).Updates(map[string]any{
		"started_at": now,
		"status":     "running",
	})
	o.emit(execution.ID, "running", "")

	// getCtx 从 ctxMap 获取当前有效 context（Resume/RetryStage 后会更新为新 ctx）
	getCtx := func() context.Context {
		o.mu.RLock()
		curCtx, ok := o.ctxMap[execution.ID]
		o.mu.RUnlock()
		if !ok {
			return context.Background()
		}
		return curCtx
	}

	for _, group := range groups {
		// 每次循环重新获取 context（Resume/RetryStage 后 ctx 已更新）
		curCtx := getCtx()

		// 检查 context 是否已取消（Cancel 时退出）
		if curCtx.Err() != nil {
			if o.IsCancelled(execution.ID) {
				logger.Info("工作流执行被终止", zap.Uint("execution_id", execution.ID))
			} else {
				logger.Info("工作流执行被中断", zap.Uint("execution_id", execution.ID))
			}
			return
		}

		// 跳过已成功的 stage group（Resume/RetryStage 时不需要重跑）
		var existingGroupExec workflow.WorkflowStageGroupExecution
		if err := o.db.Where("execution_id = ? AND group_id = ? AND status = ?",
			execution.ID, group.ID, "success").First(&existingGroupExec).Error; err == nil {
			continue
		}

		// 查找或创建阶段组执行记录
		var groupExec workflow.WorkflowStageGroupExecution
		result := o.db.Where("execution_id = ? AND group_id = ? AND status != ?",
			execution.ID, group.ID, "success").First(&groupExec)
		if result.Error != nil {
			now := time.Now()
			groupExec = workflow.WorkflowStageGroupExecution{
				ExecutionID: execution.ID,
				GroupID:     group.ID,
				Status:      "running",
				StartedAt:   &now,
			}
			o.db.Create(&groupExec)
		} else {
			o.db.Model(&groupExec).Updates(map[string]interface{}{
				"status":      "running",
				"error":       "",
				"finished_at": nil,
			})
			o.emitGroup(execution.ID, groupExec.GroupID, "running")
		}

		stages := group.Stages
		sort.Slice(stages, func(i, j int) bool {
			return stages[i].Order < stages[j].Order
		})

	retryGroup:
		curCtx = getCtx()
		groupFailed := false
		groupPaused := false

		if group.Mode == "parallel" {
			groupFailed, groupPaused = o.executeStagesParallel(curCtx, execution, &groupExec, stages, wf.Hooks, globalVars, groupsMap)
		} else {
			groupFailed, groupPaused = o.executeStagesSequential(curCtx, execution, &groupExec, stages, wf.Hooks, globalVars, groupsMap)
		}

		if groupFailed {
			if o.IsCancelled(execution.ID) {
				logger.Info("工作流执行被终止", zap.Uint("execution_id", execution.ID))
				return
			}

			o.db.Model(&groupExec).Updates(map[string]interface{}{
				"status":      "failed",
				"finished_at": time.Now(),
			})
			o.emitGroup(execution.ID, groupExec.GroupID, "failed")
			logger.Error("阶段组执行失败，等待用户重试",
				zap.Uint("execution_id", execution.ID),
				zap.String("group_name", group.Name),
			)

			// 创建 pausing channel，阻塞等待用户重试（与 Pause 行为一致，ctx 保留不销毁）
			o.mu.Lock()
			o.pausing[execution.ID] = make(chan struct{})
			o.mu.Unlock()

			o.waitForResume(execution.ID)

			// 重新获取 context（RetryStage 复用旧 ctx）
			curCtx = getCtx()
			if curCtx.Err() != nil {
				return
			}

			logger.Info("工作流 stage 重试，重新执行阶段组",
				zap.Uint("execution_id", execution.ID),
				zap.String("group_name", group.Name),
			)
			goto retryGroup
		}

		if groupPaused {
			o.db.Model(&groupExec).Updates(map[string]interface{}{
				"status":      "paused",
				"finished_at": time.Now(),
			})
			o.emitGroup(execution.ID, groupExec.GroupID, "paused")
			logger.Info("工作流执行暂停，等待用户继续",
				zap.Uint("execution_id", execution.ID),
				zap.String("group_name", group.Name),
			)

			o.waitForResume(execution.ID)

			curCtx = getCtx()
			if curCtx.Err() != nil {
				return
			}

			logger.Info("工作流执行恢复，继续执行",
				zap.Uint("execution_id", execution.ID),
			)
			goto retryGroup
		}

		finishedAt := time.Now()
		o.db.Model(&groupExec).Updates(map[string]interface{}{
			"status":      "success",
			"finished_at": finishedAt,
		})
		o.emitGroup(execution.ID, groupExec.GroupID, "success")
	}

	finishedAt := time.Now()
	o.db.Model(execution).Updates(map[string]interface{}{
		"status":      "success",
		"finished_at": finishedAt,
	})
	o.emit(execution.ID, "success", "")

	logger.Info("工作流执行完成", zap.Uint("execution_id", execution.ID), zap.String("workflow_name", wf.Name))
}

// executeStagesSequential 顺序执行 stages，返回 (是否有失败, 是否被暂停)
func (o *Orchestrator) executeStagesSequential(ctx context.Context, execution *workflow.WorkflowExecution, groupExec *workflow.WorkflowStageGroupExecution, stages []workflow.WorkflowStage, hooks []workflow.WorkflowHook, globalVars map[string]interface{}, groupsMap map[string]interface{}) (bool, bool) {
	for _, stage := range stages {
		// 检查 context 是否已取消（Cancel 时退出）
		if ctx.Err() != nil {
			logger.Info("工作流执行被中断", zap.Uint("execution_id", execution.ID))
			return false, true
		}

		// 检查是否被暂停（Pause 时跳过后续 stages，但不取消 ctx）
		o.mu.RLock()
		_, paused := o.pausing[execution.ID]
		o.mu.RUnlock()
		if paused {
			logger.Info("工作流执行暂停，跳过后续阶段", zap.Uint("execution_id", execution.ID))
			return false, true
		}

		// 跳过已成功的 stage（Resume/RetryStage 时不需要重跑）
		var existingStageExec workflow.WorkflowStageExecution
		if err := o.db.Where("stage_group_execution_id = ? AND stage_id = ? AND status = ?",
			groupExec.ID, stage.ID, "success").First(&existingStageExec).Error; err == nil {
			continue
		}

		stageFailed := o.executeStage(ctx, execution, groupExec, stage, hooks, globalVars, groupsMap)
		if stageFailed {
			return true, false
		}
	}
	return false, false
}

// executeStagesParallel 并行执行 stages，返回 (是否有失败, 是否被暂停)
func (o *Orchestrator) executeStagesParallel(ctx context.Context, execution *workflow.WorkflowExecution, groupExec *workflow.WorkflowStageGroupExecution, stages []workflow.WorkflowStage, hooks []workflow.WorkflowHook, globalVars map[string]interface{}, groupsMap map[string]interface{}) (bool, bool) {
	var wg sync.WaitGroup
	var mu sync.Mutex
	failed := false

	for _, stage := range stages {
		// 检查 context 是否已取消（Cancel 时退出）
		if ctx.Err() != nil {
			logger.Info("工作流执行被中断", zap.Uint("execution_id", execution.ID))
			wg.Wait()
			return false, true
		}

		// 检查是否被暂停（Pause 时跳过后续 stages）
		o.mu.RLock()
		_, paused := o.pausing[execution.ID]
		o.mu.RUnlock()
		if paused {
			logger.Info("工作流执行暂停，跳过后续阶段", zap.Uint("execution_id", execution.ID))
			wg.Wait()
			return false, true
		}

		// 跳过已成功的 stage（Resume 时不需要重跑）
		var existingStageExec workflow.WorkflowStageExecution
		if err := o.db.Where("stage_group_execution_id = ? AND stage_id = ? AND status = ?",
			groupExec.ID, stage.ID, "success").First(&existingStageExec).Error; err == nil {
			continue
		}

		wg.Add(1)
		go func(s workflow.WorkflowStage) {
			defer wg.Done()
			stageFailed := o.executeStage(ctx, execution, groupExec, s, hooks, globalVars, groupsMap)
			if stageFailed {
				mu.Lock()
				failed = true
				mu.Unlock()
			}
		}(stage)
	}

	wg.Wait()
	return failed, false
}

// executeStage 执行单个 stage，返回是否有失败
// 执行顺序：循环 task → 每个 task 并发执行所有机器
// 优化：Params 在 stage 层解析一次，taskExec 批量写入 DB
func (o *Orchestrator) executeStage(ctx context.Context, execution *workflow.WorkflowExecution, groupExec *workflow.WorkflowStageGroupExecution, stage workflow.WorkflowStage, hooks []workflow.WorkflowHook, globalVars map[string]interface{}, groupsMap map[string]interface{}) bool {
	now := time.Now()

	// 创建阶段执行记录
	stageExec := &workflow.WorkflowStageExecution{
		StageGroupExecutionID: groupExec.ID,
		StageID:               stage.ID,
		Status:                "running",
		StartedAt:             &now,
	}
	o.db.Create(stageExec)
	o.emitStage(execution.ID, stage.ID, "running")

	// 获取该阶段的机器列表
	if stage.MachineGroup == nil || len(stage.MachineGroup.Machines) == 0 {
		o.db.Model(stageExec).Updates(map[string]interface{}{
			"status":      "failed",
			"error":       "阶段未配置机器分组或分组内无机器",
			"finished_at": time.Now(),
		})
		o.emitStage(execution.ID, stage.ID, "failed")
		return true
	}

	// 构建当前阶段的 Group 变量
	groupVars := map[string]interface{}{
		"name": stage.MachineGroup.Name,
	}

	// 按 Order 排序 tasks
	tasks := stage.Tasks
	sort.Slice(tasks, func(i, j int) bool { return tasks[i].Order < tasks[j].Order })

	// 注册变量表（stage 级别，用于 register 字段，存储 task 输出供后续 task 引用）
	// 结构：registeredVars[registerName][host] = {stdout, changed}
	registeredVars := make(map[string]map[string]map[string]interface{})
	var registeredVarsMu sync.RWMutex

	// 逐个 task 执行，每个 task 内所有机器并发
	for _, task := range tasks {
		if ctx.Err() != nil {
			o.db.Model(stageExec).Updates(map[string]interface{}{
				"status":      "failed",
				"error":       "执行被终止",
				"finished_at": time.Now(),
			})
			o.emitStage(execution.ID, stage.ID, "failed")
			return true
		}

		// 在 stage 层解析一次 Params，避免每台机器重复解析
		params := make(map[string]string)
		if task.Params != "" {
			if err := json.Unmarshal([]byte(task.Params), &params); err != nil {
				logger.Warn("任务参数解析失败", zap.Uint("task_id", task.ID), zap.Error(err))
				o.db.Model(stageExec).Updates(map[string]interface{}{
					"status":      "failed",
					"error":       "任务参数解析失败",
					"finished_at": time.Now(),
				})
				o.emitStage(execution.ID, stage.ID, "failed")
				return true
			}
		}

		// loop 处理：解析 JSON 数组，逐个 item 执行
		if task.Loop != "" {
			var loopItems []interface{}
			if err := json.Unmarshal([]byte(task.Loop), &loopItems); err != nil {
				logger.Warn("任务 loop 参数解析失败", zap.Uint("task_id", task.ID), zap.Error(err))
				o.db.Model(stageExec).Updates(map[string]interface{}{
					"status":      "failed",
					"error":       "任务 loop 参数解析失败",
					"finished_at": time.Now(),
				})
				o.emitStage(execution.ID, stage.ID, "failed")
				return true
			}

			// 逐个 item 执行
			for _, item := range loopItems {
				var taskExecs sync.Map
				var failCount int32
				var wg sync.WaitGroup

				for _, m := range stage.MachineGroup.Machines {
					host := fmt.Sprintf("%s:%d", m.IP, m.Port)
					wg.Add(1)
					go func(m machine.Machine) {
						defer wg.Done()
						now := time.Now()
						taskExec := &workflow.WorkflowTaskExecution{
							StageExecutionID: stageExec.ID,
							TaskID:           task.ID,
							Host:             host,
							Status:           "running",
							StartedAt:        &now,
						}
						o.executeTask(ctx, taskExec, &task, &m, hooks, params, globalVars, groupVars, groupsMap, item, registeredVars, &registeredVarsMu)
						now2 := time.Now()
						taskExec.FinishedAt = &now2
						taskExecs.Store(host, taskExec)
						if taskExec.Status == "failed" {
							atomic.AddInt32(&failCount, 1)
						}
					}(m)
				}
				wg.Wait()

				o.db.Transaction(func(tx *gorm.DB) error {
					taskExecs.Range(func(key, value interface{}) bool {
						te := value.(*workflow.WorkflowTaskExecution)
						tx.Save(te)
						// 广播任务状态（带详细信息）
						o.emitTask(execution.ID, te.TaskID, task.Ref, task.Name, te.Status, te.Host, te.Output, te.Error, te.Trace, te.ErrorCode, te.Changed, te.DurationMs)
						return true
					})
					return nil
				})

				if ctx.Err() != nil {
					o.db.Model(stageExec).Updates(map[string]interface{}{
						"status":      "failed",
						"error":       "执行被终止",
						"finished_at": time.Now(),
					})
					o.emitStage(execution.ID, stage.ID, "failed")
					return true
				}

				// 任一迭代失败则停止后续迭代
				if failCount > 0 {
					o.db.Model(stageExec).Updates(map[string]interface{}{
						"status":      "failed",
						"error":       fmt.Sprintf("任务 [%s] 循环 %s 时在 %d 台机器上失败", task.Name, item, failCount),
						"finished_at": time.Now(),
					})
					o.emitStage(execution.ID, stage.ID, "failed")
					return true
				}
			}
		} else {
			// 无 loop：原有逻辑
			var taskExecs sync.Map
			var failCount int32
			var wg sync.WaitGroup

			for _, m := range stage.MachineGroup.Machines {
				host := fmt.Sprintf("%s:%d", m.IP, m.Port)
				wg.Add(1)
				go func(m machine.Machine) {
					defer wg.Done()
					now := time.Now()
					taskExec := &workflow.WorkflowTaskExecution{
						StageExecutionID: stageExec.ID,
						TaskID:           task.ID,
						Host:             host,
						Status:           "running",
						StartedAt:        &now,
					}
					o.executeTask(ctx, taskExec, &task, &m, hooks, params, globalVars, groupVars, groupsMap, nil, registeredVars, &registeredVarsMu)
					now2 := time.Now()
					taskExec.FinishedAt = &now2
					taskExecs.Store(host, taskExec)
					if taskExec.Status == "failed" {
						atomic.AddInt32(&failCount, 1)
					}
				}(m)
			}
			wg.Wait()

			// 批量写入所有 taskExec，一次事务
			o.db.Transaction(func(tx *gorm.DB) error {
				taskExecs.Range(func(key, value interface{}) bool {
					te := value.(*workflow.WorkflowTaskExecution)
					tx.Save(te)
					// 广播任务状态（带详细信息）
					o.emitTask(execution.ID, te.TaskID, task.Ref, task.Name, te.Status, te.Host, te.Output, te.Error, te.Trace, te.ErrorCode, te.Changed, te.DurationMs)
					return true
				})
				return nil
			})

			// 检查 context
			if ctx.Err() != nil {
				o.db.Model(stageExec).Updates(map[string]interface{}{
					"status":      "failed",
					"error":       "执行被终止",
					"finished_at": time.Now(),
				})
				o.emitStage(execution.ID, stage.ID, "failed")
				return true
			}

			// 汇总失败信息
			if failCount > 0 {
				o.db.Model(stageExec).Updates(map[string]interface{}{
					"status":      "failed",
					"error":       fmt.Sprintf("任务 [%s] 在 %d 台机器上失败", task.Name, failCount),
					"finished_at": time.Now(),
				})
				o.emitStage(execution.ID, stage.ID, "failed")
				return true
			}
		}
	}

	// 阶段成功
	o.db.Model(stageExec).Updates(map[string]interface{}{
		"status":      "success",
		"finished_at": time.Now(),
	})
	o.emitStage(execution.ID, stage.ID, "success")

	return false
}

// executeTask 执行单个任务（通过 gRPC 调用 Agent），支持重试
// 注意：此函数不做任何 DB 写入和日志输出，由调用方批量处理
func (o *Orchestrator) executeTask(ctx context.Context, taskExec *workflow.WorkflowTaskExecution, task *workflow.WorkflowTask, m *machine.Machine, hooks []workflow.WorkflowHook, params map[string]string, globalVars map[string]interface{}, groupVars map[string]interface{}, groupsMap map[string]interface{}, loopItem interface{}, registeredVars map[string]map[string]map[string]interface{}, registeredVarsMu *sync.RWMutex) {
	host := fmt.Sprintf("%s:%d", m.IP, m.Port)

	machineMap := MachineToMap(m)

	// 构建当前机器的 register 变量（per-machine，只取当前 host 的数据）
	currentRegister := make(map[string]interface{})
	registeredVarsMu.RLock()
	for regName, hostMap := range registeredVars {
		if data, ok := hostMap[host]; ok {
			currentRegister[regName] = data
		}
	}
	registeredVarsMu.RUnlock()

	// 构建完整的模板变量上下文
	templateVars := map[string]interface{}{
		"Machine":        machineMap,
		"GlobalVariable": globalVars,
		"Group":          groupVars,
		"Groups":         groupsMap,
		"Register":       currentRegister,
		"Server": map[string]interface{}{
			"ip":       o.serverIP,
			"port":     o.serverPort,
			"protocol": o.protocol,
		},
	}
	if loopItem != nil {
		templateVars["item"] = loopItem
	}

	// 检查 when 条件（兼容旧格式 .machine 和新格式 .Machine）
	if task.When != "" {
		whenVars := map[string]interface{}{
			"Machine":        machineMap,
			"GlobalVariable": globalVars,
			"Group":          groupVars,
			"Groups":         groupsMap,
			"Register":       currentRegister,
			"Server":         templateVars["Server"],
		}
		if loopItem != nil {
			whenVars["item"] = loopItem
		}
		run, err := evaluateWhen(task.When, whenVars)
		if err != nil {
			taskExec.Status = "failed"
			taskExec.Error = fmt.Sprintf("when 条件解析失败: %v", err)
			return
		}
		if !run {
			taskExec.Status = "skipped"
			taskExec.Output = "条件不满足，跳过执行"
			return
		}
	}

	// 对 params 的每个 value 做模板渲染
	renderedParams := make(map[string]string)
	for k, v := range params {
		rendered, err := RenderTemplate(v, templateVars)
		if err != nil {
			// 渲染失败时原样使用（含非法模板语法的情况，如 {{xxx}}）
			renderedParams[k] = v
		} else {
			renderedParams[k] = rendered
		}
	}

	// copy 模块自动计算 server 侧 src 文件的 MD5
	if task.Module == "copy" {
		if srcPath, ok := renderedParams["src"]; ok && srcPath != "" {
			if md5Val, err := utils.FileMD5(srcPath); err == nil {
				renderedParams["md5"] = md5Val
			} else {
				logger.Warn("copy模块自动计算MD5失败", zap.String("src", srcPath), zap.Error(err))
			}
		}
	}

	// template 模块：如果指定了 src（模板名称），从 DB 读取模板内容并渲染
	if task.Module == "template" {
		if srcVal, ok := renderedParams["src"]; ok && srcVal != "" && renderedParams["content"] == "" {
			var tpl workflow.WorkflowTemplate
			if err := o.db.Where("name = ?", srcVal).First(&tpl).Error; err == nil {
				// 渲染模板内容
				rendered, err := RenderTemplate(tpl.Content, templateVars)
				if err != nil {
					logger.Warn("template模块渲染模板失败", zap.String("src", srcVal), zap.Error(err))
					taskExec.Status = "failed"
					taskExec.Error = fmt.Sprintf("渲染模板失败: %v", err)
					return
				}
				renderedParams["content"] = rendered
				delete(renderedParams, "src") // 不传 src 给 agent
			} else {
				taskExec.Status = "failed"
				taskExec.Error = fmt.Sprintf("模板「%s」不存在", srcVal)
				return
			}
		}
	}

	maxRetries := task.Retries
	if maxRetries < 0 {
		maxRetries = 0
	}

	var lastErr string
	var hasAgentResponse bool // 是否成功连接到 agent 并拿到业务响应
	var lastChanged bool      // 最后一次业务响应的 changed 状态
	for attempt := 0; attempt <= maxRetries; attempt++ {
		logger.Info("开始执行任务", zap.String("尝试次数", fmt.Sprintf("%d/%d", attempt+1, maxRetries+1)), zap.String("task_id", fmt.Sprintf("ref-%d", task.Ref)))

		if ctx.Err() != nil {
			taskExec.Status = "skipped"
			taskExec.Error = "执行被中断"
			return
		}

		// 重试前等待
		if attempt > 0 && task.Delay > 0 {
			select {
			case <-ctx.Done():
				taskExec.Status = "skipped"
				taskExec.Error = "执行被中断"
				return
			case <-time.After(time.Duration(task.Delay) * time.Second):
			}
		}

		startTime := time.Now()

		conn, err := o.pool.GetConn(host)
		if err != nil {
			lastErr = fmt.Sprintf("连接Agent失败: %v", err)
			taskExec.DurationMs = time.Since(startTime).Milliseconds()
			continue
		}

		client := agentpb.NewAgentServiceClient(conn)
		var reqCtx context.Context
		var cancel context.CancelFunc
		if task.Timeout > 0 {
			reqCtx, cancel = context.WithTimeout(ctx, time.Duration(task.Timeout)*time.Second)
		} else {
			reqCtx, cancel = context.WithCancel(ctx)
		}

		resp, err := client.Exec(reqCtx, &agentpb.ExecRequest{
			MachineId:  host,
			Module:     task.Module,
			Parameters: renderedParams,
			TaskId:     fmt.Sprintf("ref-%d", task.Ref),
			Timeout:    int32(task.Timeout),
		})
		cancel()

		// 优先使用 agent 返回的耗时，兜底用本地计算
		if resp != nil && resp.DurationMs > 0 {
			taskExec.DurationMs = resp.DurationMs
		} else {
			taskExec.DurationMs = time.Since(startTime).Milliseconds()
		}

		if err != nil {
			lastErr = fmt.Sprintf("gRPC调用失败: %v", err)
			continue
		}

		if !resp.Success {
			// 业务失败（agent 已响应，但执行失败）
			hasAgentResponse = true
			lastChanged = resp.Changed
			taskExec.Output = resp.Stdout
			if resp.Error != nil {
				taskExec.ErrorCode = resp.Error.Code
				taskExec.Error = resp.Error.Message
				taskExec.Trace = resp.Error.Trace
			} else {
				taskExec.Error = "任务执行失败（无详细错误信息）"
			}
			lastErr = taskExec.Error
			continue
		}

		// 成功
		taskExec.Status = "success"
		taskExec.Output = resp.Stdout
		taskExec.Changed = resp.Changed

		// 注册变量（per-machine，按 host 存储，供后续 task 的 when 或 params 引用）
		if task.Register != "" {
			registeredVarsMu.Lock()
			if registeredVars[task.Register] == nil {
				registeredVars[task.Register] = make(map[string]map[string]interface{})
			}
			registeredVars[task.Register][host] = map[string]interface{}{
				"stdout":  resp.Stdout,
				"changed": resp.Changed,
			}
			registeredVarsMu.Unlock()
		}

		// 执行后置钩子：Changed=true 或上次钩子失败时触发
		if task.Hooks != "" {
			shouldRunHooks := resp.Changed || taskExec.HookStatus == "failed"
			if shouldRunHooks {
				taskExec.HookStatus = "running"
				if err := o.executeHooks(ctx, host, task.Hooks, hooks, templateVars); err != nil {
					taskExec.HookStatus = "failed"
					taskExec.HookError = err.Error()
					taskExec.Status = "failed"
					taskExec.Error = fmt.Sprintf("后置钩子执行失败: %v", err)
					return
				}
				taskExec.HookStatus = "success"
			}
		}
		return
	}

	// 所有重试均失败
	taskExec.Status = "failed"
	taskExec.Error = lastErr

	// 业务失败且忽略错误：标记为成功，继续执行后续 task
	if hasAgentResponse && task.IgnoreErrors {
		taskExec.Status = "success"
		taskExec.Output = fmt.Sprintf("[忽略错误] %s", lastErr)
		if task.Hooks != "" {
			shouldRun := lastChanged || taskExec.HookStatus == "failed"
			if shouldRun {
				taskExec.HookStatus = "running"
				if err := o.executeHooks(ctx, host, task.Hooks, hooks, templateVars); err != nil {
					taskExec.HookStatus = "failed"
					taskExec.HookError = err.Error()
				} else {
					taskExec.HookStatus = "success"
				}
			}
		}
	}
}

// executeHooks 执行后置钩子（解析 HookIDs 名称列表，从 workflow.Hooks 中匹配执行）
// 钩子失败返回 error，workflow 整体标记为 failed
func (o *Orchestrator) executeHooks(ctx context.Context, host string, hooksJSON string, allHooks []workflow.WorkflowHook, templateVars map[string]interface{}) error {
	var hookNames []string
	if err := json.Unmarshal([]byte(hooksJSON), &hookNames); err != nil {
		return fmt.Errorf("解析钩子名称失败: %w", err)
	}

	// 建立 hook name -> WorkflowHook 的映射
	hookMap := make(map[string]workflow.WorkflowHook)
	for _, h := range allHooks {
		hookMap[h.Name] = h
	}

	for _, name := range hookNames {
		hook, ok := hookMap[name]
		if !ok {
			return fmt.Errorf("钩子[%s]不存在", name)
		}

		logger.Info("执行后置钩子",
			zap.String("host", host),
			zap.String("name", hook.Name),
			zap.String("module", hook.Module),
		)

		// 解析钩子 Params 并渲染模板变量
		params := make(map[string]string)
		if hook.Params != "" {
			if err := json.Unmarshal([]byte(hook.Params), &params); err != nil {
				params["command"] = hook.Params
			}
		}
		renderedParams := make(map[string]string)
		for k, v := range params {
			rendered, err := RenderTemplate(v, templateVars)
			if err != nil {
				renderedParams[k] = v
			} else {
				renderedParams[k] = rendered
			}
		}

		// 钩子独立重试循环
		maxRetries := hook.Retries
		if maxRetries < 0 {
			maxRetries = 0
		}
		var lastErr error
		var hadAgentResponse bool
		for attempt := 0; attempt <= maxRetries; attempt++ {
			if ctx.Err() != nil {
				return fmt.Errorf("钩子[%s]执行被中断", name)
			}

			// 重试前等待
			if attempt > 0 && hook.Delay > 0 {
				select {
				case <-ctx.Done():
					return fmt.Errorf("钩子[%s]执行被中断", name)
				case <-time.After(time.Duration(hook.Delay) * time.Second):
				}
			}

			// 获取 Agent 连接
			conn, err := o.pool.GetConn(host)
			if err != nil {
				lastErr = fmt.Errorf("连接Agent失败: %v", err)
				continue
			}

			// 调用 Agent Exec
			client := agentpb.NewAgentServiceClient(conn)
			var hookCtx context.Context
			var cancel context.CancelFunc
			if hook.Timeout > 0 {
				hookCtx, cancel = context.WithTimeout(ctx, time.Duration(hook.Timeout)*time.Second)
			} else {
				hookCtx, cancel = context.WithCancel(ctx)
			}

			resp, err := client.Exec(hookCtx, &agentpb.ExecRequest{
				MachineId:  host,
				Module:     hook.Module,
				Parameters: renderedParams,
				TaskId:     fmt.Sprintf("hook-%s", hook.Name),
			})
			cancel()

			if err != nil {
				lastErr = fmt.Errorf("gRPC调用失败: %v", err)
				continue
			}

			if !resp.Success {
				hadAgentResponse = true
				if resp.Error != nil {
					lastErr = fmt.Errorf("钩子执行失败: %s", resp.Error.Message)
				} else {
					lastErr = fmt.Errorf("钩子执行失败（无详细错误信息）")
				}
				continue
			}

			// 成功
			lastErr = nil
			break
		}

		if lastErr != nil {
			if hadAgentResponse && hook.IgnoreErrors {
				// 业务失败且忽略错误：仅记录日志，继续下一个钩子
				logger.Warn("钩子执行失败（已忽略）",
					zap.String("name", hook.Name),
					zap.Error(lastErr),
				)
			} else {
				return fmt.Errorf("钩子[%s]执行失败(retries=%d): %w", name, maxRetries, lastErr)
			}
		}
	}
	return nil
}

// failExecution 标记执行失败（仅用于基础设施错误：加载 workflow 定义、机器分组失败等不可重试的错误）
func (o *Orchestrator) failExecution(execution *workflow.WorkflowExecution, errMsg string) {
	now := time.Now()
	o.db.Model(execution).Updates(map[string]interface{}{
		"status":      "failed",
		"error":       errMsg,
		"finished_at": now,
	})
	o.emit(execution.ID, "failed", errMsg)

	logger.Error("工作流执行失败",
		zap.Uint("execution_id", execution.ID),
		zap.String("error", errMsg),
	)
}

// MachineToMap 将 Machine 对象转换为模板可用的 map
func MachineToMap(m *machine.Machine) map[string]interface{} {
	disks := make([]map[string]interface{}, 0)
	for _, d := range m.Disks {
		disks = append(disks, map[string]interface{}{
			"device":   d.Device,
			"type":     d.Type,
			"total_gb": d.TotalGB,
		})
	}
	networks := make([]map[string]interface{}, 0)
	for _, n := range m.Networks {
		networks = append(networks, map[string]interface{}{
			"name":   n.Name,
			"ip":     n.IP,
			"mac":    n.MAC,
			"speed":  n.Speed,
			"status": n.Status,
		})
	}
	gpus := make([]map[string]interface{}, 0)
	for _, g := range m.GPUs {
		gpus = append(gpus, map[string]interface{}{
			"name":           g.Name,
			"count":          g.Count,
			"driver_version": g.DriverVersion,
		})
	}
	return map[string]interface{}{
		"ip":               m.IP,
		"port":             m.Port,
		"status":           m.Status,
		"hostname":         m.Hostname,
		"virtualization":   m.Virtualization,
		"os_name":          m.OSName,
		"os_version":       m.OSVersion,
		"kernel":           m.Kernel,
		"arch":             m.Arch,
		"cpu_model":        m.CPUModel,
		"cpu_cores":        m.CPUCores,
		"memory_kb":        m.MemoryKB,
		"swap_kb":          m.SwapKB,
		"gateway":          m.Gateway,
		"firewall_status":  m.FirewallStatus,
		"firewall_enabled": m.FirewallEnabled,
		"timezone":         m.Timezone,
		"disks":            disks,
		"networks":         networks,
		"gpus":             gpus,
	}
}

// BuildGroupsMap 将所有机器分组转换为模板变量格式：map[组名][]map[属性名]值
func BuildGroupsMap(groups []machine.MachineGroup) map[string]interface{} {
	result := make(map[string]interface{})
	for _, g := range groups {
		machines := make([]map[string]interface{}, 0, len(g.Machines))
		for i := range g.Machines {
			machines = append(machines, MachineToMap(&g.Machines[i]))
		}
		result[g.Name] = machines
	}
	return result
}
