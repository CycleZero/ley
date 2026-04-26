package task

import (
	"encoding/json"
	"time"
)

// MetricsExporter 指标导出器
type MetricsExporter struct {
	queue *TaskQueue
}

// NewMetricsExporter 创建指标导出器
func NewMetricsExporter(queue *TaskQueue) *MetricsExporter {
	return &MetricsExporter{
		queue: queue,
	}
}

// ExportMetrics 导出指标为 JSON
func (e *MetricsExporter) ExportMetrics() ([]byte, error) {
	stats := e.queue.GetStats()

	metrics := map[string]interface{}{
		"timestamp":           time.Now().Format(time.RFC3339),
		"total_tasks":         stats.TotalTasks,
		"success_tasks":       stats.SuccessTasks,
		"failed_tasks":        stats.FailedTasks,
		"retry_tasks":         stats.RetryTasks,
		"queued_tasks":        stats.QueuedTasks,
		"active_workers":      stats.ActiveWorkers,
		"worker_count":        stats.WorkerCount,
		"max_queue_size":      stats.MaxQueueSize,
		"avg_execute_time_ms": stats.AvgExecuteTime / 1e6, // 转换为毫秒
		"success_rate":        0.0,
		"failure_rate":        0.0,
		"queue_usage":         0.0,
	}

	// 计算成功率
	if stats.TotalTasks > 0 {
		metrics["success_rate"] = float64(stats.SuccessTasks) / float64(stats.TotalTasks) * 100
		metrics["failure_rate"] = float64(stats.FailedTasks) / float64(stats.TotalTasks) * 100
	}

	// 计算队列使用率
	if stats.MaxQueueSize > 0 {
		metrics["queue_usage"] = float64(stats.QueuedTasks) / float64(stats.MaxQueueSize) * 100
	}

	return json.MarshalIndent(metrics, "", "  ")
}

// GetMetricsMap 获取指标 Map
func (e *MetricsExporter) GetMetricsMap() map[string]interface{} {
	stats := e.queue.GetStats()

	metrics := make(map[string]interface{})
	metrics["timestamp"] = time.Now().Format(time.RFC3339)
	metrics["total_tasks"] = stats.TotalTasks
	metrics["success_tasks"] = stats.SuccessTasks
	metrics["failed_tasks"] = stats.FailedTasks
	metrics["retry_tasks"] = stats.RetryTasks
	metrics["queued_tasks"] = stats.QueuedTasks
	metrics["active_workers"] = stats.ActiveWorkers
	metrics["worker_count"] = stats.WorkerCount
	metrics["max_queue_size"] = stats.MaxQueueSize
	metrics["avg_execute_time_ms"] = stats.AvgExecuteTime / 1e6

	// 计算成功率
	if stats.TotalTasks > 0 {
		metrics["success_rate"] = float64(stats.SuccessTasks) / float64(stats.TotalTasks) * 100
		metrics["failure_rate"] = float64(stats.FailedTasks) / float64(stats.TotalTasks) * 100
	}

	// 计算队列使用率
	if stats.MaxQueueSize > 0 {
		metrics["queue_usage"] = float64(stats.QueuedTasks) / float64(stats.MaxQueueSize) * 100
	}

	return metrics
}

// HealthCheck 健康检查
func (e *MetricsExporter) HealthCheck() HealthStatus {
	stats := e.queue.GetStats()

	status := HealthStatus{
		Status:    "healthy",
		Timestamp: time.Now().Format(time.RFC3339),
		Checks:    make(map[string]bool),
	}

	// 检查工作器是否活跃
	status.Checks["workers_active"] = stats.ActiveWorkers > 0
	if !status.Checks["workers_active"] {
		status.Status = "unhealthy"
	}

	// 检查队列是否已满
	status.Checks["queue_not_full"] = float64(stats.QueuedTasks)/float64(stats.MaxQueueSize) < 0.9
	if !status.Checks["queue_not_full"] {
		status.Status = "warning"
	}

	// 检查失败率
	if stats.TotalTasks > 0 {
		failureRate := float64(stats.FailedTasks) / float64(stats.TotalTasks)
		status.Checks["failure_rate_ok"] = failureRate < 0.1 // 失败率低于 10%
		if !status.Checks["failure_rate_ok"] {
			if status.Status == "healthy" {
				status.Status = "warning"
			}
		}
	} else {
		status.Checks["failure_rate_ok"] = true
	}

	return status
}

// HealthStatus 健康状态
type HealthStatus struct {
	Status    string          `json:"status"`
	Timestamp string          `json:"timestamp"`
	Checks    map[string]bool `json:"checks"`
}
