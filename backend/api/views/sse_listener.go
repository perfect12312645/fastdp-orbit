package views

// SSEListener 适配 orchestrator.EventListener，桥接到 SSE Hub
type SSEListener struct{}

func (l *SSEListener) OnExecutionStatus(executionID uint, status string, errMsg string) {
	BroadcastExecutionStatus(executionID, status, errMsg)
}

func (l *SSEListener) OnGroupStatus(executionID uint, groupID uint, status string) {
	BroadcastGroupStatus(executionID, groupID, status)
}

func (l *SSEListener) OnStageStatus(executionID uint, stageID uint, status string) {
	BroadcastStageStatus(executionID, stageID, status)
}

func (l *SSEListener) OnTaskStatus(executionID uint, taskID uint, status string, host string, output string, errStr string, duration int64) {
	BroadcastTaskStatus(executionID, taskID, status, host, output, errStr, duration)
}
