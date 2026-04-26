package infra

import "ley/pkg/task"

var globalTaskQueue *task.TaskQueue

func TaskQueue() *task.TaskQueue {
	if globalTaskQueue == nil {
		initTaskQueue()
	}
	return globalTaskQueue
}

func initTaskQueue() {
	globalTaskQueue = task.NewTaskQueue(task.DefaultConfig())
	globalTaskQueue.Start()
}

func StopTaskQueue() {
	if globalTaskQueue == nil {
		return
	}
	globalTaskQueue.Stop()
}
