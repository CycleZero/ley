package task

import (
	"errors"
	"runtime"
)

var (
	// ErrPoolShutdown 工作池已关闭
	ErrPoolShutdown = errors.New("worker pool is shutdown")

	// ErrTaskNotFound 任务未找到
	ErrTaskNotFound = errors.New("task not found")

	// ErrQueueFull 队列已满
	ErrQueueFull = errors.New("task queue is full")
)

// runtimeNumCPU 获取运行时 CPU 核心数
func runtimeNumCPU() int {
	numCPU := runtime.NumCPU()
	if numCPU < 2 {
		return 2 // 至少使用 2 个工作器
	}
	if numCPU > 16 {
		return 16 // 最多使用 16 个工作器
	}
	return numCPU
}
