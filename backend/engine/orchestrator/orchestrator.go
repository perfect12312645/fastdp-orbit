package orchestrator

import (
	"context"
	"encoding/json"
	"fastdp-orbit/backend/models/workflow"
	"fastdp-orbit/backend/pkg/logger"
	agentpb "fastdp-orbit/backend/proto/agent"
	"fastdp-orbit/backend/server/grpc"
	"fmt"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

// Orchestrator 工作流执行引擎
type Orchestrator struct {
	mu       sync.RWMutex
	db       *gorm.DB
	pool     *grpc.AgentConnPool
	running  map[uint]*cancelContext // executionID -> cancel
}

type cancelContext struct {
	cancel context.CancelFunc
}

// NewOrchestrator 创建执行引擎
func NewOrchestrator(db *gorm.DB, pool *grpc.AgentConnPool) *Orchestrator {
	return &Orchestrator{
		db:      db,
		pool:    pool,
		running: make(map[uint]*cancelContext),
	}
}

// Execute 启动工作流执行
func (o *Orchestrator) Execute(execution *workflow.WorkflowExecution) error {
	ctx, cancel := context.WithCancel(context.Background())

	o.mu.Lock()
	o.running[execution.ID] = &cancelContext{cancel: cancel}
	o.mu.Unlock()

	// 异步执行
	go o.run(ctx, execution)

	return nil
}

// Pause 暂停执行（当前阶段完成后停止）
func (o *Orchestrator) Pause(executionID uint) error {
	o.mu.RLock()
	cc, ok := o.running[executionID]
	o.mu.RUnlock()

	if !ok {
		return fmt.Errorf("执行实例 %d 不在运行中", executionID)
	}

	// 更新状态为 paused
	o.db.Model(&workflow.WorkflowExecution{}).Where("id = ?", executionID).
		Update("status", "paused").Update("error", "用户暂停")

	// 取消 context（当前 stage 完成后退出循环）
	cc.cancel()
	return nil
}

// Cancel 终止执行（立即停止）
func (o *Orchestrator) Cancel(executionID uint) error {
	o.mu.RLock()
	cc, ok := o.running[executionID]
	o.mu.RUnlock()

	if !ok {
		return fmt.Errorf("执行实例 %d 不在运行中", executionID)
	}

	cc.cancel()

	// 更新状态为 cancelled
	o.db.Model(&workflow.WorkflowExecution{}).Where("id = ?", executionID).
		Update("status", "cancelled").Update("error", "用户终止")
	o.db.Model(&workflow.WorkflowStageExecution{}).Where("execution_id = ? AND status IN ?", executionID, []string{"pending", "running"}).
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

// run 执行工作流主循环
func (o *Orchestrator) run(ctx context.Context, execution *workflow.WorkflowExecution) {
	defer func() {
		o.mu.Lock()
		delete(o.running, execution.ID)
		o.mu.Unlock()
	}()

	logger.Info("工作流开始执行",
		zap.Uint("execution_id", execution.ID),
		zap.Uint("workflow_id", execution.WorkflowID),
	)

	// 加载 workflow 定义（含 stages 和 tasks）
	var wf workflow.Workflow
	if err := o.db.Preload("Stages.Tasks").First(&wf, execution.WorkflowID).Error; err != nil {
		o.failExecution(execution, fmt.Sprintf("加载工作流定义失败: %v", err))
		return
	}

	// 按 Order 排序 stages（GORM 默认按 ID，手动排序更可靠）
	stages := wf.Stages
	sortStages(stages)

	now := time.Now()
	execution.StartedAt = now
	o.db.Model(execution).Update("started_at", now)
	o.db.Model(execution).Update("status", "running")

	for _, stage := range stages {
		// 检查 context 是否已取消（暂停/终止）
		if ctx.Err() != nil {
			logger.Info("工作流执行被中断", zap.Uint("execution_id", execution.ID))
			return
		}

		// 创建阶段执行记录
		stageExec := &workflow.WorkflowStageExecution{
			ExecutionID: execution.ID,
			StageID:     stage.ID,
			Status:      "running",
			StartedAt:   &now,
		}
		o.db.Create(stageExec)

		// 执行阶段内的所有任务
		stageFailed := false
		tasks := stage.Tasks
		sortTasks(tasks)

		for _, task := range tasks {
			// 检查 context
			if ctx.Err() != nil {
				o.db.Model(stageExec).Updates(map[string]interface{}{
					"status":      "skipped",
					"finished_at": time.Now(),
				})
				return
			}

			// 创建任务执行记录
			taskExec := &workflow.WorkflowTaskExecution{
				StageExecutionID: stageExec.ID,
				TaskID:           task.ID,
				Status:           "running",
			}
			o.db.Create(taskExec)

			// 执行任务
			o.executeTask(ctx, taskExec, &task)

			if taskExec.Status == "failed" {
				stageFailed = true
				o.db.Model(stageExec).Updates(map[string]interface{}{
					"status":      "failed",
					"error":       taskExec.Error,
					"finished_at": time.Now(),
				})
				break
			}
		}

		if stageFailed {
			o.failExecution(execution, fmt.Sprintf("阶段 [%s] 执行失败", stage.Name))
			return
		}

		// 阶段成功
		finishedAt := time.Now()
		o.db.Model(stageExec).Updates(map[string]interface{}{
			"status":      "success",
			"finished_at": finishedAt,
		})
	}

	// 全部成功
	finishedAt := time.Now()
	o.db.Model(execution).Updates(map[string]interface{}{
		"status":      "success",
		"finished_at": finishedAt,
	})

	logger.Info("工作流执行完成", zap.Uint("execution_id", execution.ID))
}

// executeTask 执行单个任务（通过 gRPC 调用 Agent）
func (o *Orchestrator) executeTask(ctx context.Context, taskExec *workflow.WorkflowTaskExecution, task *workflow.WorkflowTask) {
	startTime := time.Now()
	now := time.Now()
	o.db.Model(taskExec).Update("started_at", now)

	// 检查 when 条件（简化实现：仅检查 OSName 包含/不包含）
	if task.When != "" && !evaluateWhen(task.When) {
		logger.Info("任务条件不满足，跳过", zap.Uint("task_id", task.ID), zap.String("when", task.When))
		taskExec.Status = "skipped"
		taskExec.Output = "条件不满足，跳过执行"
		taskExec.DurationMs = 0
		o.db.Save(taskExec)
		return
	}

	// 获取 Agent 连接
	conn, err := o.pool.GetConn(task.Host)
	if err != nil {
		taskExec.Status = "failed"
		taskExec.Error = fmt.Sprintf("连接Agent失败: %v", err)
		taskExec.DurationMs = time.Since(startTime).Milliseconds()
		o.db.Save(taskExec)
		return
	}

	// 解析 Params JSON
	params := make(map[string]string)
	if task.Params != "" {
		if err := json.Unmarshal([]byte(task.Params), &params); err != nil {
			// 解析失败时作为单个 command 参数
			params["command"] = task.Params
		}
	}

	// 调用 Agent Exec RPC
	client := agentpb.NewAgentServiceClient(conn)
	reqCtx, cancel := context.WithTimeout(ctx, time.Duration(task.Timeout)*time.Second)
	defer cancel()

	resp, err := client.Exec(reqCtx, &agentpb.ExecRequest{
		MachineId:  task.Host,
		Module:     task.Module,
		Parameters: params,
		TaskId:     fmt.Sprintf("task-%d", task.ID),
		Timeout:    int32(task.Timeout),
	})

	duration := time.Since(startTime)
	taskExec.DurationMs = duration.Milliseconds()

	if err != nil {
		taskExec.Status = "failed"
		taskExec.Error = fmt.Sprintf("gRPC调用失败: %v", err)
		o.db.Save(taskExec)
		return
	}

	// 解析响应
	taskExec.Output = resp.Stdout
	if !resp.Success {
		taskExec.Status = "failed"
		taskExec.Error = resp.Stderr
		if resp.Error != nil {
			taskExec.Error = resp.Error.Message
		}
	} else {
		taskExec.Status = "success"
	}

	// 执行后置钩子（hooks）
	if task.Hooks != "" && taskExec.Status == "success" {
		o.executeHooks(ctx, task.Host, task.Hooks)
	}

	o.db.Save(taskExec)
}

// executeHooks 执行后置钩子（逗号分隔的钩子名）
func (o *Orchestrator) executeHooks(ctx context.Context, host string, hooks string) {
	// 后置钩子需要调用 Agent 执行预定义的脚本
	// 暂时仅记录日志，后续实现钩子脚本库
	logger.Info("执行后置钩子", zap.String("host", host), zap.String("hooks", hooks))
}

// evaluateWhen 简化 when 条件评估（支持 contains/!contains 操作符）
func evaluateWhen(when string) bool {
	// 简单实现：解析 "{{.Machine.OSName}} !contains ubuntu" 格式
	// 后续可扩展为完整 Go 模板表达式
	if strings.Contains(when, "!contains") {
		return !strings.Contains(when, "ubuntu")
	}
	if strings.Contains(when, "contains") {
		return strings.Contains(when, "ubuntu")
	}
	return true
}

// failExecution 标记执行失败
func (o *Orchestrator) failExecution(execution *workflow.WorkflowExecution, errMsg string) {
	now := time.Now()
	o.db.Model(execution).Updates(map[string]interface{}{
		"status":      "failed",
		"error":       errMsg,
		"finished_at": now,
	})

	// 将所有 pending 的阶段标记为 skipped
	o.db.Model(&workflow.WorkflowStageExecution{}).
		Where("execution_id = ? AND status = ?", execution.ID, "pending").
		Update("status", "skipped")

	logger.Error("工作流执行失败",
		zap.Uint("execution_id", execution.ID),
		zap.String("error", errMsg),
	)
}

func sortStages(stages []workflow.WorkflowStage) {
	for i := 1; i < len(stages); i++ {
		for j := i; j > 0 && stages[j].Order < stages[j-1].Order; j-- {
			stages[j], stages[j-1] = stages[j-1], stages[j]
		}
	}
}

func sortTasks(tasks []workflow.WorkflowTask) {
	for i := 1; i < len(tasks); i++ {
		for j := i; j > 0 && tasks[j].Order < tasks[j-1].Order; j-- {
			tasks[j], tasks[j-1] = tasks[j-1], tasks[j]
		}
	}
}
