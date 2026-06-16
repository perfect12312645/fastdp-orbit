package monitor

import (
	"time"
)

// Monitor handles system monitoring
type Monitor struct {
	interval time.Duration
	stopCh   chan struct{}
}

// NewMonitor creates a new monitor instance
func NewMonitor(interval time.Duration) *Monitor {
	return &Monitor{
		interval: interval,
		stopCh:   make(chan struct{}),
	}
}

// Start begins monitoring
func (m *Monitor) Start() {
	// TODO: Start monitoring goroutines
}

// Stop stops monitoring
func (m *Monitor) Stop() {
	close(m.stopCh)
}

// GetNodeMetrics returns metrics for a specific node
func (m *Monitor) GetNodeMetrics(nodeID uint) (*NodeMetrics, error) {
	// TODO: Get node metrics
	return nil, nil
}

// GetClusterOverview returns overview of the cluster
func (m *Monitor) GetClusterOverview() (*ClusterOverview, error) {
	// TODO: Get cluster overview
	return nil, nil
}

// NodeMetrics represents metrics for a single node
type NodeMetrics struct {
	CPUUsage    float64 `json:"cpu_usage"`
	MemoryUsage float64 `json:"memory_usage"`
	DiskUsage   float64 `json:"disk_usage"`
	NetworkIn   int64   `json:"network_in"`
	NetworkOut  int64   `json:"network_out"`
	GPUMetrics  []GPUMetric `json:"gpu_metrics,omitempty"`
}

// GPUMetric represents metrics for a single GPU
type GPUMetric struct {
	Index       int     `json:"index"`
	Name        string  `json:"name"`
	Usage       float64 `json:"usage"`
	MemoryUsed  int64   `json:"memory_used"`
	MemoryTotal int64   `json:"memory_total"`
	Temperature int     `json:"temperature"`
}

// ClusterOverview represents overview of the entire cluster
type ClusterOverview struct {
	TotalNodes   int     `json:"total_nodes"`
	OnlineNodes  int     `json:"online_nodes"`
	TotalCPU     int     `json:"total_cpu"`
	UsedCPU      float64 `json:"used_cpu"`
	TotalMemory  int64   `json:"total_memory"`
	UsedMemory   int64   `json:"used_memory"`
	TotalGPUs    int     `json:"total_gpus"`
	UsedGPUs     int     `json:"used_gpus"`
}
