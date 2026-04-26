package task

import (
	"context"
	"sync"
	"time"

	"go.uber.org/zap"
)

// TaskQueue 任务队列管理器
type TaskQueue struct {
	pool         *WorkerPool
	taskMap      sync.Map // map[taskID]*Task
	maxQueueSize int
	ctx          context.Context
	cancel       context.CancelFunc
	wg           sync.WaitGroup
}

// TaskQueueConfig 任务队列配置
type TaskQueueConfig struct {
	WorkerCount  int // 工作器数量
	MaxQueueSize int // 最大队列大小
}

// DefaultConfig 默认配置
func DefaultConfig() TaskQueueConfig {
	return TaskQueueConfig{
		WorkerCount:  0, // 0 表示自动根据 CPU 核心数
		MaxQueueSize: 10000,
	}
}

// logger 获取全局日志器
func logger() *zap.SugaredLogger {
	return zap.L().Sugar()
}

// NewTaskQueue 创建任务队列
func NewTaskQueue(config TaskQueueConfig) *TaskQueue {
	if config.WorkerCount <= 0 {
		config.WorkerCount = runtimeNumCPU()
	}
	if config.MaxQueueSize <= 0 {
		config.MaxQueueSize = 10000
	}

	ctx, cancel := context.WithCancel(context.Background())

	queue := &TaskQueue{
		pool:         NewWorkerPool(config.WorkerCount),
		maxQueueSize: config.MaxQueueSize,
		ctx:          ctx,
		cancel:       cancel,
	}

	// 设置任务回调
	queue.pool.OnTaskSuccess(func(task *Task) {
		logger().Debugf("任务执行成功：task=%s, name=%s", task.ID, task.Name)
	})

	queue.pool.OnTaskFailed(func(task *Task) {
		logger().Errorf("任务执行失败：task=%s, name=%s, error=%v",
			task.ID, task.Name, task.Error)
	})

	return queue
}

// Start 启动任务队列
func (q *TaskQueue) Start() {
	logger().Info("启动任务队列...")
	q.pool.Start()

	// 启动监控协程
	q.wg.Add(1)
	go q.monitor()
}

// Stop 停止任务队列
func (q *TaskQueue) Stop() {
	logger().Info("停止任务队列...")
	q.cancel()
	q.pool.Stop()
	q.wg.Wait()
	logger().Info("任务队列已停止")
}

// Submit 提交任务
func (q *TaskQueue) Submit(task *Task) error {
	// 检查队列是否已满
	if q.GetQueueSize() >= q.maxQueueSize {
		return ErrQueueFull
	}

	// 注册任务
	q.taskMap.Store(task.ID, task)

	// 提交到工作池
	return q.pool.Submit(task)
}

// SubmitFunc 提交任务函数（便捷方法）
func (q *TaskQueue) SubmitFunc(name string, fn TaskFunc, opts ...TaskOption) (string, error) {
	task := NewTask(name, fn, opts...)

	// 检查队列是否已满
	if q.GetQueueSize() >= q.maxQueueSize {
		return "", ErrQueueFull
	}

	q.taskMap.Store(task.ID, task)

	err := q.pool.Submit(task)
	if err != nil {
		return "", err
	}

	return task.ID, nil
}

// SubmitFuncSync 同步提交任务并等待结果
func (q *TaskQueue) SubmitFuncSync(name string, fn TaskFunc, timeout time.Duration, opts ...TaskOption) (interface{}, error) {
	task := NewTask(name, fn, opts...)
	task.Timeout = timeout

	// 创建结果通道
	resultChan := make(chan error, 1)

	// 设置完成回调
	task.TaskFunc = func() error {
		err := fn()
		resultChan <- err
		return err
	}

	// 检查队列是否已满
	if q.GetQueueSize() >= q.maxQueueSize {
		return nil, ErrQueueFull
	}

	q.taskMap.Store(task.ID, task)

	err := q.pool.Submit(task)
	if err != nil {
		return nil, err
	}

	// 等待结果
	select {
	case err := <-resultChan:
		return nil, err
	case <-time.After(timeout):
		return nil, context.DeadlineExceeded
	}
}

// GetTask 获取任务
func (q *TaskQueue) GetTask(taskID string) (*Task, bool) {
	task, ok := q.taskMap.Load(taskID)
	if !ok {
		return nil, false
	}
	return task.(*Task), true
}

// CancelTask 取消任务
func (q *TaskQueue) CancelTask(taskID string) error {
	task, ok := q.taskMap.Load(taskID)
	if !ok {
		return ErrTaskNotFound
	}

	t := task.(*Task)
	if t.Status == TaskStatusPending {
		t.Status = TaskStatusCancelled
		return nil
	}

	// 对于正在执行的任务，无法立即取消，只能等待其完成
	return nil
}

// GetQueueSize 获取队列大小
func (q *TaskQueue) GetQueueSize() int {
	return q.pool.GetQueueSize()
}

// GetStats 获取统计信息
func (q *TaskQueue) GetStats() *QueueStats {
	poolStats := q.pool.GetStats()

	return &QueueStats{
		TotalTasks:     poolStats.TotalTasks,
		SuccessTasks:   poolStats.SuccessTasks,
		FailedTasks:    poolStats.FailedTasks,
		RetryTasks:     poolStats.RetryTasks,
		ActiveWorkers:  poolStats.ActiveWorkers,
		QueuedTasks:    poolStats.QueuedTasks,
		AvgExecuteTime: poolStats.AvgExecuteTime,
		WorkerCount:    int64(q.pool.GetWorkerCount()),
		MaxQueueSize:   int64(q.maxQueueSize),
	}
}

// QueueStats 队列统计信息
type QueueStats struct {
	TotalTasks     int64 // 总任务数
	SuccessTasks   int64 // 成功任务数
	FailedTasks    int64 // 失败任务数
	RetryTasks     int64 // 重试任务数
	ActiveWorkers  int64 // 活跃工作器数
	QueuedTasks    int64 // 排队任务数
	WorkerCount    int64 // 工作器总数
	MaxQueueSize   int64 // 最大队列大小
	AvgExecuteTime int64 // 平均执行时间 (纳秒)
}

// monitor 监控协程
func (q *TaskQueue) monitor() {
	defer q.wg.Done()

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-q.ctx.Done():
			return
		case <-ticker.C:
			stats := q.GetStats()
			logger().Infof("任务队列统计：总任务=%d, 成功=%d, 失败=%d, 重试=%d, 排队=%d, 活跃工作器=%d/%d",
				stats.TotalTasks, stats.SuccessTasks, stats.FailedTasks,
				stats.RetryTasks, stats.QueuedTasks, stats.ActiveWorkers, stats.WorkerCount)
		}
	}
}
