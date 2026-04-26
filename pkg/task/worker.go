package task

import (
	"context"
	"fmt"
	"runtime/debug"
	"sync/atomic"
	"time"
)

// Worker 工作器
type Worker struct {
	id          int
	pool        *WorkerPool
	taskChan    chan *Task
	stopChan    chan struct{}
	isBusy      atomic.Bool
	currentTask atomic.Pointer[Task]
}

// newWorker 创建工作器
func newWorker(id int, pool *WorkerPool) *Worker {
	return &Worker{
		id:       id,
		pool:     pool,
		taskChan: make(chan *Task, 1),
		stopChan: make(chan struct{}, 1),
	}
}

// start 启动工作器
func (w *Worker) start() {
	go func() {
		for {
			select {
			case task := <-w.taskChan:
				if task == nil {
					return
				}
				w.execute(task)
			case <-w.stopChan:
				return
			}
		}
	}()
}

// stop 停止工作器
func (w *Worker) stop() {
	select {
	case w.stopChan <- struct{}{}:
	default:
	}
}

// execute 执行任务
func (w *Worker) execute(task *Task) {
	w.isBusy.Store(true)
	w.currentTask.Store(task)

	defer func() {
		w.isBusy.Store(false)
		w.currentTask.Store(nil)
		w.pool.taskDone()
	}()

	// 更新任务状态
	now := time.Now()
	task.Status = TaskStatusRunning
	task.StartedAt = &now

	// 执行任务（带超时和恢复）
	ctx, cancel := context.WithTimeout(context.Background(), task.Timeout)
	defer cancel()

	doneChan := make(chan error, 1)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				logger().Errorf("任务执行发生 panic: task=%s, error=%v, stack=%s",
					task.ID, r, string(debug.Stack()))
				doneChan <- fmt.Errorf("panic: %v", r)
			}
		}()
		doneChan <- task.TaskFunc()
	}()

	select {
	case err := <-doneChan:
		if err != nil {
			task.Error = err
			task.Status = TaskStatusFailed
			task.FinishedAt = &now

			// 检查是否需要重试
			if task.RetryCount < task.MaxRetries {
				task.RetryCount++
				logger().Infof("任务执行失败，准备重试：task=%s, retry=%d/%d, error=%v",
					task.ID, task.RetryCount, task.MaxRetries, err)
				w.pool.retryTask(task)
			} else {
				logger().Errorf("任务执行失败：task=%s, error=%v", task.ID, err)
				w.pool.notifyTaskFailed(task)
			}
		} else {
			task.Status = TaskStatusSuccess
			task.FinishedAt = &now
			w.pool.notifyTaskSuccess(task)
		}

	case <-ctx.Done():
		task.Error = ctx.Err()
		task.Status = TaskStatusFailed
		task.FinishedAt = &now
		logger().Errorf("任务执行超时：task=%s, timeout=%v", task.ID, task.Timeout)
		w.pool.notifyTaskFailed(task)
	}
}

// IsBusy 检查工作器是否忙碌
func (w *Worker) IsBusy() bool {
	return w.isBusy.Load()
}

// GetCurrentTask 获取当前执行的任务
func (w *Worker) GetCurrentTask() *Task {
	return w.currentTask.Load()
}
