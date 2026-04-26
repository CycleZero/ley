package task

import (
	"time"
)

// TaskStatus 任务状态
type TaskStatus int

const (
	TaskStatusPending   TaskStatus = iota // 等待执行
	TaskStatusRunning                     // 执行中
	TaskStatusSuccess                     // 执行成功
	TaskStatusFailed                      // 执行失败
	TaskStatusCancelled                   // 已取消
)

func (s TaskStatus) String() string {
	switch s {
	case TaskStatusPending:
		return "pending"
	case TaskStatusRunning:
		return "running"
	case TaskStatusSuccess:
		return "success"
	case TaskStatusFailed:
		return "failed"
	case TaskStatusCancelled:
		return "cancelled"
	default:
		return "unknown"
	}
}

// TaskFunc 任务函数类型
type TaskFunc func() error

// Task 任务定义
type Task struct {
	ID         string                 // 任务唯一标识
	Name       string                 // 任务名称
	TaskFunc   TaskFunc               // 任务执行函数
	Status     TaskStatus             // 任务状态
	Priority   int                    // 优先级 (0-100, 越高优先级越高)
	MaxRetries int                    // 最大重试次数
	RetryCount int                    // 当前重试次数
	Timeout    time.Duration          // 任务超时时间
	CreatedAt  time.Time              // 创建时间
	StartedAt  *time.Time             // 开始执行时间
	FinishedAt *time.Time             // 完成时间
	Error      error                  // 错误信息
	Metadata   map[string]interface{} // 元数据
	result     interface{}            // 执行结果 (内部使用)
}

// NewTask 创建新任务
func NewTask(name string, fn TaskFunc, opts ...TaskOption) *Task {
	task := &Task{
		ID:         generateTaskID(),
		Name:       name,
		TaskFunc:   fn,
		Status:     TaskStatusPending,
		Priority:   50, // 默认中等优先级
		MaxRetries: 3,
		Timeout:    30 * time.Second,
		CreatedAt:  time.Now(),
		Metadata:   make(map[string]interface{}),
	}

	for _, opt := range opts {
		opt(task)
	}

	return task
}

// TaskOption 任务配置选项
type TaskOption func(*Task)

// WithPriority 设置任务优先级
func WithPriority(priority int) TaskOption {
	return func(t *Task) {
		if priority < 0 {
			priority = 0
		}
		if priority > 100 {
			priority = 100
		}
		t.Priority = priority
	}
}

// WithMaxRetries 设置最大重试次数
func WithMaxRetries(maxRetries int) TaskOption {
	return func(t *Task) {
		if maxRetries < 0 {
			maxRetries = 0
		}
		t.MaxRetries = maxRetries
	}
}

// WithTimeout 设置任务超时时间
func WithTimeout(timeout time.Duration) TaskOption {
	return func(t *Task) {
		if timeout > 0 {
			t.Timeout = timeout
		}
	}
}

// WithMetadata 设置任务元数据
func WithMetadata(key string, value interface{}) TaskOption {
	return func(t *Task) {
		t.Metadata[key] = value
	}
}

// GetResult 获取任务执行结果
func (t *Task) GetResult() interface{} {
	return t.result
}
