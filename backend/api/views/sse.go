package views

import (
	"net/http"
	"strconv"
	"sync"

	"fastdp-orbit/backend/pkg/logger"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// SSEEvent SSE 事件
type SSEEvent struct {
	ExecutionID uint   `json:"execution_id"`
	Type        string `json:"type"` // execution_status, group_status, stage_status, task_status
	Status      string `json:"status"`
	GroupID     uint   `json:"group_id,omitempty"`
	StageID     uint   `json:"stage_id,omitempty"`
	TaskID      uint   `json:"task_id,omitempty"`
	Error       string `json:"error,omitempty"`
}

// SSEClient SSE 客户端
type SSEClient struct {
	ID          uint
	ExecutionID uint
	Chan        chan SSEEvent
	Done        chan struct{}
}

// SSEHub SSE 连接管理中心
type SSEHub struct {
	mu      sync.RWMutex
	clients map[uint][]*SSEClient // executionID -> clients
	nextID  uint
}

var Hub = &SSEHub{
	clients: make(map[uint][]*SSEClient),
}

// Subscribe 订阅指定 execution 的状态更新
func (h *SSEHub) Subscribe(executionID uint) *SSEClient {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.nextID++
	client := &SSEClient{
		ID:          h.nextID,
		ExecutionID: executionID,
		Chan:        make(chan SSEEvent, 64),
		Done:        make(chan struct{}),
	}
	h.clients[executionID] = append(h.clients[executionID], client)
	return client
}

// Unsubscribe 取消订阅
func (h *SSEHub) Unsubscribe(client *SSEClient) {
	h.mu.Lock()
	defer h.mu.Unlock()

	clients := h.clients[client.ExecutionID]
	for i, c := range clients {
		if c.ID == client.ID {
			h.clients[client.ExecutionID] = append(clients[:i], clients[i+1:]...)
			break
		}
	}
	close(client.Done)
}

// Broadcast 向指定 execution 的所有客户端广播事件
func (h *SSEHub) Broadcast(event SSEEvent) {
	h.mu.RLock()
	clients := h.clients[event.ExecutionID]
	h.mu.RUnlock()

	for _, client := range clients {
		select {
		case client.Chan <- event:
		default:
			// channel 满了则丢弃，避免阻塞
		}
	}
}

// HandleSSE SSE 端点处理器
func HandleSSE(c *gin.Context) {
	executionIDStr := c.Param("id")
	executionID, err := strconv.ParseUint(executionIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid execution id"})
		return
	}

	// 设置 SSE 响应头
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")

	client := Hub.Subscribe(uint(executionID))
	defer Hub.Unsubscribe(client)

	// 发送初始连接成功消息
	c.SSEvent("connected", map[string]any{
		"execution_id": executionID,
		"message":      "connected",
	})
	c.Writer.Flush()

	// 监听事件并推送给客户端
	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		return
	}

	for {
		select {
		case <-client.Done:
			return
		case <-c.Request.Context().Done():
			return
		case event := <-client.Chan:
			c.SSEvent(event.Type, event)
			flusher.Flush()
		}
	}
}

// BroadcastExecutionStatus 广播 execution 状态变更（供 orchestrator 调用）
func BroadcastExecutionStatus(executionID uint, status string, errStr string) {
	Hub.Broadcast(SSEEvent{
		ExecutionID: executionID,
		Type:        "execution_status",
		Status:      status,
		Error:       errStr,
	})
	logger.Debug("SSE broadcast execution status",
		zap.Uint("execution_id", executionID),
		zap.String("status", status),
	)
}

// BroadcastGroupStatus 广播阶段组状态变更
func BroadcastGroupStatus(executionID uint, groupID uint, status string) {
	Hub.Broadcast(SSEEvent{
		ExecutionID: executionID,
		Type:        "group_status",
		GroupID:     groupID,
		Status:      status,
	})
}

// BroadcastStageStatus 广播阶段状态变更
func BroadcastStageStatus(executionID uint, stageID uint, status string) {
	Hub.Broadcast(SSEEvent{
		ExecutionID: executionID,
		Type:        "stage_status",
		StageID:     stageID,
		Status:      status,
	})
}

// BroadcastTaskStatus 广播任务状态变更
func BroadcastTaskStatus(executionID uint, taskID uint, status string) {
	Hub.Broadcast(SSEEvent{
		ExecutionID: executionID,
		Type:        "task_status",
		TaskID:      taskID,
		Status:      status,
	})
}
