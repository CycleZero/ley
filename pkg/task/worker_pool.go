package task

import (
	"container/heap"
	"sync"
	"sync/atomic"
	"time"
)

// PriorityQueue 优先级队列
type PriorityQueue struct {
	tasks priorityHeap
	mu    sync.Mutex
}

// priorityHeap 实现 heap.Interface
type priorityHeap []*Task

func (ph priorityHeap) Len() int { return len(ph) }
func (ph priorityHeap) Less(i, j int) bool {
	// 优先级高的先执行，同优先级则先创建的先执行
	if ph[i].Priority == ph[j].Priority {
		return ph[i].CreatedAt.Before(ph[j].CreatedAt)
	}
	return ph[i].Priority > ph[j].Priority
}
func (ph priorityHeap) Swap(i, j int) { ph[i], ph[j] = ph[j], ph[i] }

func (ph *priorityHeap) Push(x interface{}) {
	*ph = append(*ph, x.(*Task))
}

func (ph *priorityHeap) Pop() interface{} {
	old := *ph
	n := len(old)
	if n == 0 {
		return nil
	}
	task := old[n-1]
	*ph = old[0 : n-1]
	return task
}

// PushTask 添加任务到队列
func (pq *PriorityQueue) PushTask(task *Task) {
	pq.mu.Lock()
	defer pq.mu.Unlock()
	heap.Push(&pq.tasks, task)
}

// PopTask 从队列取出任务
func (pq *PriorityQueue) PopTask() *Task {
	pq.mu.Lock()
	defer pq.mu.Unlock()
	if len(pq.tasks) == 0 {
		return nil
	}
	return heap.Pop(&pq.tasks).(*Task)
}

// PeekTask 查看队首任务（不取出）
func (pq *PriorityQueue) PeekTask() *Task {
	pq.mu.Lock()
	defer pq.mu.Unlock()
	if len(pq.tasks) == 0 {
		return nil
	}
	return pq.tasks[0]
}

// Size 获取队列大小
func (pq *PriorityQueue) Size() int {
	pq.mu.Lock()
	defer pq.mu.Unlock()
	return len(pq.tasks)
}

// WorkerPool 工作池
type WorkerPool struct {
	workers      []*Worker
	workerCount  int
	taskQueue    *PriorityQueue
	retryQueue   *PriorityQueue
	shutdownChan chan struct{}
	wg           sync.WaitGroup
	isShutdown   int32

	// 任务回调
	onTaskSuccess func(*Task)
	onTaskFailed  func(*Task)

	// 统计信息
	stats *PoolStats

	// 任务完成通知
	taskDoneChan chan struct{}
}

// PoolStats 工作池统计信息
type PoolStats struct {
	TotalTasks     int64 // 总任务数
	SuccessTasks   int64 // 成功任务数
	FailedTasks    int64 // 失败任务数
	RetryTasks     int64 // 重试任务数
	ActiveWorkers  int64 // 活跃工作器数
	QueuedTasks    int64 // 排队任务数
	AvgExecuteTime int64 // 平均执行时间 (纳秒)
	totalExecTime  int64 // 总执行时间 (内部使用)
	execCount      int64 // 执行次数 (内部使用)
}

// NewWorkerPool 创建工作池
func NewWorkerPool(workerCount int) *WorkerPool {
	if workerCount <= 0 {
		workerCount = runtimeNumCPU()
	}

	pool := &WorkerPool{
		workerCount:  workerCount,
		workers:      make([]*Worker, 0, workerCount),
		taskQueue:    &PriorityQueue{tasks: make(priorityHeap, 0)},
		retryQueue:   &PriorityQueue{tasks: make(priorityHeap, 0)},
		shutdownChan: make(chan struct{}),
		stats:        &PoolStats{},
		taskDoneChan: make(chan struct{}, 1),
	}

	// 创建工作器
	for i := 0; i < workerCount; i++ {
		worker := newWorker(i, pool)
		pool.workers = append(pool.workers, worker)
	}

	return pool
}

// Start 启动工作池
func (p *WorkerPool) Start() {
	logger().Infof("启动工作池，工作器数量：%d", p.workerCount)

	// 启动所有工作器
	for _, worker := range p.workers {
		worker.start()
	}

	// 启动任务调度器
	go p.scheduler()
}

// Stop 停止工作池
func (p *WorkerPool) Stop() {
	if !atomic.CompareAndSwapInt32(&p.isShutdown, 0, 1) {
		return // 已经关闭
	}

	logger().Info("停止工作池...")
	close(p.shutdownChan)

	// 等待所有任务完成
	p.wg.Wait()

	// 停止所有工作器
	for _, worker := range p.workers {
		worker.stop()
	}

	logger().Info("工作池已停止")
}

// Submit 提交任务
func (p *WorkerPool) Submit(task *Task) error {
	if atomic.LoadInt32(&p.isShutdown) == 1 {
		return ErrPoolShutdown
	}

	atomic.AddInt64(&p.stats.TotalTasks, 1)
	p.wg.Add(1)
	p.taskQueue.PushTask(task)

	// 通知调度器有新任务
	select {
	case p.taskDoneChan <- struct{}{}:
	default:
	}

	return nil
}

// SubmitBatch 批量提交任务
func (p *WorkerPool) SubmitBatch(tasks []*Task) error {
	if atomic.LoadInt32(&p.isShutdown) == 1 {
		return ErrPoolShutdown
	}

	for _, task := range tasks {
		atomic.AddInt64(&p.stats.TotalTasks, 1)
		p.wg.Add(1)
		p.taskQueue.PushTask(task)
	}

	// 通知调度器有新任务
	select {
	case p.taskDoneChan <- struct{}{}:
	default:
	}

	return nil
}

// scheduler 任务调度器
func (p *WorkerPool) scheduler() {
	for {
		select {
		case <-p.shutdownChan:
			return
		case <-p.taskDoneChan:
			p.dispatchTasks()
		case <-time.After(100 * time.Millisecond):
			// 定期检查是否有任务需要处理
			p.dispatchTasks()
		}
	}
}

// dispatchTasks 分发任务到空闲工作器
func (p *WorkerPool) dispatchTasks() {
	for _, worker := range p.workers {
		if worker.IsBusy() {
			continue
		}

		// 优先处理重试队列中的任务
		task := p.retryQueue.PopTask()
		if task == nil {
			task = p.taskQueue.PopTask()
		}

		if task == nil {
			return
		}

		worker.taskChan <- task
	}
}

// retryTask 重试任务
func (p *WorkerPool) retryTask(task *Task) {
	atomic.AddInt64(&p.stats.RetryTasks, 1)
	p.wg.Add(1) // 重试任务需要重新添加 WaitGroup
	p.retryQueue.PushTask(task)

	// 通知调度器
	select {
	case p.taskDoneChan <- struct{}{}:
	default:
	}
}

// taskDone 任务完成
func (p *WorkerPool) taskDone() {
	p.wg.Done()
}

// notifyTaskSuccess 通知任务成功
func (p *WorkerPool) notifyTaskSuccess(task *Task) {
	atomic.AddInt64(&p.stats.SuccessTasks, 1)

	// 更新统计信息
	if task.StartedAt != nil && task.FinishedAt != nil {
		execTime := task.FinishedAt.Sub(*task.StartedAt).Nanoseconds()
		atomic.AddInt64(&p.stats.totalExecTime, execTime)
		atomic.AddInt64(&p.stats.execCount, 1)
		atomic.StoreInt64(&p.stats.AvgExecuteTime,
			atomic.LoadInt64(&p.stats.totalExecTime)/atomic.LoadInt64(&p.stats.execCount))
	}

	if p.onTaskSuccess != nil {
		go p.onTaskSuccess(task)
	}
}

// notifyTaskFailed 通知任务失败
func (p *WorkerPool) notifyTaskFailed(task *Task) {
	atomic.AddInt64(&p.stats.FailedTasks, 1)

	// 更新统计信息
	if task.StartedAt != nil && task.FinishedAt != nil {
		execTime := task.FinishedAt.Sub(*task.StartedAt).Nanoseconds()
		atomic.AddInt64(&p.stats.totalExecTime, execTime)
		atomic.AddInt64(&p.stats.execCount, 1)
		atomic.StoreInt64(&p.stats.AvgExecuteTime,
			atomic.LoadInt64(&p.stats.totalExecTime)/atomic.LoadInt64(&p.stats.execCount))
	}

	if p.onTaskFailed != nil {
		go p.onTaskFailed(task)
	}
}

// OnTaskSuccess 设置任务成功回调
func (p *WorkerPool) OnTaskSuccess(fn func(*Task)) {
	p.onTaskSuccess = fn
}

// OnTaskFailed 设置任务失败回调
func (p *WorkerPool) OnTaskFailed(fn func(*Task)) {
	p.onTaskFailed = fn
}

// GetStats 获取统计信息
func (p *WorkerPool) GetStats() *PoolStats {
	stats := &PoolStats{
		TotalTasks:     atomic.LoadInt64(&p.stats.TotalTasks),
		SuccessTasks:   atomic.LoadInt64(&p.stats.SuccessTasks),
		FailedTasks:    atomic.LoadInt64(&p.stats.FailedTasks),
		RetryTasks:     atomic.LoadInt64(&p.stats.RetryTasks),
		QueuedTasks:    int64(p.taskQueue.Size()),
		AvgExecuteTime: atomic.LoadInt64(&p.stats.AvgExecuteTime),
	}

	// 计算活跃工作器数
	var activeWorkers int64
	for _, worker := range p.workers {
		if worker.IsBusy() {
			activeWorkers++
		}
	}
	stats.ActiveWorkers = activeWorkers

	return stats
}

// GetWorkerCount 获取工作器数量
func (p *WorkerPool) GetWorkerCount() int {
	return p.workerCount
}

// GetQueueSize 获取队列大小
func (p *WorkerPool) GetQueueSize() int {
	return p.taskQueue.Size()
}

// IsShutdown 检查是否已关闭
func (p *WorkerPool) IsShutdown() bool {
	return atomic.LoadInt32(&p.isShutdown) == 1
}
