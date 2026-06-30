package workflow

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"sync"
	"time"

	"fastdp-orbit/backend/engine/orchestrator"
	"fastdp-orbit/backend/models/machine"
	"fastdp-orbit/backend/models/workflow"

	"gorm.io/gorm"
)

// StageExecutionService 单阶段执行服务
type StageExecutionService struct {
	db      *gorm.DB
	running map[uint]context.CancelFunc
	mu      sync.RWMutex

	// 依赖注入（用于执行任务）
	executeTaskFunc func(ctx context.Context, taskExec *workflow.StageTaskExecution, task *workflow.StageTask, m *machine.Machine, hooks []workflow.WorkflowHook, params map[string]string, globalVars map[string]interface{}, groupVars map[string]interface{}, groupsMap map[string]interface{}, loopItem interface{}, registeredVars map[string]map[string]map[string]interface{}, registeredVarsMu *sync.RWMutex)

	// SSE 事件回调
	emitTaskFunc func(executionID uint, taskRef int, taskName string, status string, host string, output string, errStr string, trace string, errorCode int32, changed bool, duration int64)
	emitExecFunc func(executionID uint, status string)
}

// NewStageExecutionService 创建单阶段执行服务
func NewStageExecutionService(db *gorm.DB) *StageExecutionService {
	return &StageExecutionService{
		db:      db,
		running: make(map[uint]context.CancelFunc),
	}
}

// SetExecuteTaskFunc 设置任务执行函数（注入 orchestrator 的 ExecuteTaskForStage 逻辑）
func (s *StageExecutionService) SetExecuteTaskFunc(fn func(ctx context.Context, taskExec *workflow.StageTaskExecution, task *workflow.StageTask, m *machine.Machine, hooks []workflow.WorkflowHook, params map[string]string, globalVars map[string]interface{}, groupVars map[string]interface{}, groupsMap map[string]interface{}, loopItem interface{}, registeredVars map[string]map[string]map[string]interface{}, registeredVarsMu *sync.RWMutex)) {
	s.executeTaskFunc = fn
}

// SetEmitFuncs 设置 SSE 事件回调
func (s *StageExecutionService) SetEmitFuncs(
	emitTask func(executionID uint, taskRef int, taskName string, status string, host string, output string, errStr string, trace string, errorCode int32, changed bool, duration int64),
	emitExec func(executionID uint, status string),
) {
	s.emitTaskFunc = emitTask
	s.emitExecFunc = emitExec
}

// ListStageExecutions 获取阶段的执行历史
func (s *StageExecutionService) ListStageExecutions(stageTemplateID uint) ([]workflow.StageExecution, error) {
	var executions []workflow.StageExecution
	if err := s.db.Where("stage_template_id = ?", stageTemplateID).
		Order("created_at DESC").
		Find(&executions).Error; err != nil {
		return nil, err
	}
	if executions == nil {
		executions = []workflow.StageExecution{}
	}
	return executions, nil
}

// GetStageExecution 获取执行详情（含任务记录）
func (s *StageExecutionService) GetStageExecution(id uint) (*workflow.StageExecution, error) {
	var exec workflow.StageExecution
	if err := s.db.Preload("TaskExecutions", func(db *gorm.DB) *gorm.DB {
		return db.Order("id ASC")
	}).First(&exec, id).Error; err != nil {
		return nil, err
	}
	return &exec, nil
}

// DeleteStageExecution 删除执行记录
func (s *StageExecutionService) DeleteStageExecution(id uint) error {
	var exec workflow.StageExecution
	if err := s.db.First(&exec, id).Error; err != nil {
		return fmt.Errorf("执行记录不存在")
	}
	s.db.Where("stage_execution_id = ?", id).Delete(&workflow.StageTaskExecution{})
	return s.db.Delete(&exec).Error
}

// CancelStageExecution 取消执行
func (s *StageExecutionService) CancelStageExecution(id uint) error {
	var exec workflow.StageExecution
	if err := s.db.First(&exec, id).Error; err != nil {
		return fmt.Errorf("执行记录不存在")
	}
	if exec.Status != "running" {
		return fmt.Errorf("执行状态为 %s，无法取消", exec.Status)
	}

	s.mu.RLock()
	cancel, ok := s.running[id]
	s.mu.RUnlock()
	if ok {
		cancel()
	}

	now := time.Now()
	s.db.Model(&exec).Updates(map[string]interface{}{
		"status":      "cancelled",
		"error":       "用户取消",
		"finished_at": now,
	})
	s.db.Model(&workflow.StageTaskExecution{}).
		Where("stage_execution_id = ? AND status IN ?", id, []string{"pending", "running"}).
		Updates(map[string]interface{}{"status": "skipped", "error": "用户取消"})

	if s.emitExecFunc != nil {
		s.emitExecFunc(id, "cancelled")
	}
	return nil
}

// ExecuteStage 执行单阶段
func (s *StageExecutionService) ExecuteStage(stageTemplateID uint, machineGroupID uint) (*workflow.StageExecution, error) {
	var stageTemplate workflow.StageTemplate
	if err := s.db.First(&stageTemplate, stageTemplateID).Error; err != nil {
		return nil, fmt.Errorf("阶段模板不存在: %v", err)
	}

	var tasks []workflow.StageTask
	if err := json.Unmarshal([]byte(stageTemplate.Tasks), &tasks); err != nil {
		return nil, fmt.Errorf("解析任务失败: %v", err)
	}

	groupID := machineGroupID
	if groupID == 0 {
		groupID = stageTemplate.MachineGroupID
	}
	if groupID == 0 {
		return nil, fmt.Errorf("未指定机器分组")
	}

	var machineGroup machine.MachineGroup
	if err := s.db.Preload("Machines").First(&machineGroup, groupID).Error; err != nil {
		return nil, fmt.Errorf("机器分组不存在: %v", err)
	}
	if len(machineGroup.Machines) == 0 {
		return nil, fmt.Errorf("机器分组内无机器")
	}

	now := time.Now()
	exec := &workflow.StageExecution{
		StageTemplateID:  stageTemplateID,
		StageName:        stageTemplate.Name,
		MachineGroupID:   groupID,
		MachineGroupName: machineGroup.Name,
		Status:           "running",
		Trigger:          "manual",
		StartedAt:        &now,
	}
	if err := s.db.Create(exec).Error; err != nil {
		return nil, fmt.Errorf("创建执行记录失败: %v", err)
	}

	go s.runStage(exec, tasks, &machineGroup)

	return exec, nil
}

// runStage 执行阶段（goroutine）
func (s *StageExecutionService) runStage(exec *workflow.StageExecution, tasks []workflow.StageTask, machineGroup *machine.MachineGroup) {
	ctx, cancel := context.WithCancel(context.Background())
	s.mu.Lock()
	s.running[exec.ID] = cancel
	s.mu.Unlock()

	defer func() {
		s.mu.Lock()
		delete(s.running, exec.ID)
		s.mu.Unlock()
	}()

	if s.emitExecFunc != nil {
		s.emitExecFunc(exec.ID, "running")
	}

	// 加载全局变量
	var globalVarList []workflow.GlobalVariable
	s.db.Find(&globalVarList)
	globalVars := make(map[string]interface{})
	for _, v := range globalVarList {
		globalVars[v.Key] = v.Value
	}

	// 加载所有机器分组
	var allMachineGroups []machine.MachineGroup
	s.db.Preload("Machines").Find(&allMachineGroups)
	groupsMap := orchestrator.BuildGroupsMap(allMachineGroups)

	// 加载钩子
	var hooks []workflow.WorkflowHook
	s.db.Find(&hooks)

	groupVars := map[string]interface{}{
		"name": machineGroup.Name,
	}

	registeredVars := make(map[string]map[string]map[string]interface{})
	var registeredVarsMu sync.RWMutex

	// 按 Order 排序
	sort.Slice(tasks, func(i, j int) bool {
		return tasks[i].Order < tasks[j].Order
	})

	for _, task := range tasks {
		if ctx.Err() != nil {
			s.failExecution(exec, fmt.Sprintf("执行被终止 "))
			return
		}

		// 解析参数（失败则整个 stage 失败）
		params := make(map[string]string)
		if task.Params != "" {
			if err := json.Unmarshal([]byte(task.Params), &params); err != nil {
				s.failExecution(exec, fmt.Sprintf("参数解析失败: %v", task.Name, err))
				return
			}
		}

		// loop 处理
		if task.Loop != "" {
			var loopItems []interface{}
			if err := json.Unmarshal([]byte(task.Loop), &loopItems); err != nil {
				s.failExecution(exec, fmt.Sprintf("任务「%s」loop解析失败: %v", task.Name, err))
				return
			}

			// 每个 loop item 在所有机器上并发执行
			for _, item := range loopItems {
				if ctx.Err() != nil {
					s.failExecution(exec, fmt.Sprintf("执行被终止 "))
					return
				}
				var wg sync.WaitGroup
				var mu sync.Mutex
				itemFailed := false
				for _, m := range machineGroup.Machines {
					wg.Add(1)
					go func(m machine.Machine) {
						defer wg.Done()
						if s.execTask(ctx, exec, &task, &m, hooks, params, globalVars, groupVars, groupsMap, item, registeredVars, &registeredVarsMu) {
							mu.Lock()
							itemFailed = true
							mu.Unlock()
						}
					}(m)
				}
			wg.Wait()
			if itemFailed {
				cancel()
				s.failExecution(exec, fmt.Sprintf("任务「%s」执行失败", task.Name))
				return
			}
			// 检查是否被取消
			if ctx.Err() != nil {
				s.failExecution(exec, "执行被终止")
				return
			}
		}
		} else {
			// 普通执行：所有机器并发
			var wg sync.WaitGroup
			var mu sync.Mutex
			taskFailed := false
			for _, m := range machineGroup.Machines {
				if ctx.Err() != nil {
					s.failExecution(exec, fmt.Sprintf("执行被终止 "))
					return
				}
				wg.Add(1)
				go func(m machine.Machine) {
					defer wg.Done()
					if s.execTask(ctx, exec, &task, &m, hooks, params, globalVars, groupVars, groupsMap, nil, registeredVars, &registeredVarsMu) {
						mu.Lock()
						taskFailed = true
						mu.Unlock()
					}
				}(m)
			}
			wg.Wait()
			if taskFailed {
				cancel()
				s.failExecution(exec, fmt.Sprintf("任务「%s」执行失败", task.Name))
				return
			}
			// 检查是否被取消
			if ctx.Err() != nil {
				s.failExecution(exec, "执行被终止")
				return
			}
		}
	}

	// 所有任务成功
	now := time.Now()
	s.db.Model(exec).Updates(map[string]interface{}{
		"status":      "success",
		"finished_at": now,
	})
	if s.emitExecFunc != nil {
		s.emitExecFunc(exec.ID, "success")
	}
}

// failExecution 标记执行失败并更新状态
func (s *StageExecutionService) failExecution(exec *workflow.StageExecution, errMsg string) {
	now := time.Now()
	s.db.Model(exec).Updates(map[string]interface{}{
		"status":      "failed",
		"error":       errMsg,
		"finished_at": now,
	})
	if s.emitExecFunc != nil {
		s.emitExecFunc(exec.ID, "failed")
	}
}

// execTask 执行单个任务，返回是否失败
func (s *StageExecutionService) execTask(
	ctx context.Context,
	exec *workflow.StageExecution,
	task *workflow.StageTask,
	m *machine.Machine,
	hooks []workflow.WorkflowHook,
	params map[string]string,
	globalVars map[string]interface{},
	groupVars map[string]interface{},
	groupsMap map[string]interface{},
	loopItem interface{},
	registeredVars map[string]map[string]map[string]interface{},
	registeredVarsMu *sync.RWMutex,
) bool {
	host := fmt.Sprintf("%s:%d", m.IP, m.Port)
	taskExec := &workflow.StageTaskExecution{
		StageExecutionID: exec.ID,
		TaskRef:          task.Ref,
		TaskName:         task.Name,
		TaskModule:       task.Module,
		Host:             host,
		Status:           "running",
	}
	s.db.Create(taskExec)

	if s.executeTaskFunc != nil {
		s.executeTaskFunc(ctx, taskExec, task, m, hooks, params, globalVars, groupVars, groupsMap, loopItem, registeredVars, registeredVarsMu)
	}

	now := time.Now()
	taskExec.FinishedAt = &now
	s.db.Save(taskExec)

	if s.emitTaskFunc != nil {
		s.emitTaskFunc(exec.ID, task.Ref, task.Name, taskExec.Status, taskExec.Host, taskExec.Output, taskExec.Error, taskExec.Trace, taskExec.ErrorCode, taskExec.Changed, taskExec.DurationMs)
	}

	return taskExec.Status == "failed"
}
