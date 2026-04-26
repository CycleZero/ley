package task

import (
	"fmt"
	"time"
)

// Example_basicUsage 基本使用示例
func Example_basicUsage() {
	// 创建任务队列
	queue := NewTaskQueue(DefaultConfig())
	queue.Start()
	defer queue.Stop()

	// 提交简单任务
	taskID, err := queue.SubmitFunc("simple_task", func() error {
		fmt.Println("执行简单任务")
		return nil
	})

	if err != nil {
		fmt.Printf("提交任务失败：%v\n", err)
		return
	}

	fmt.Printf("任务已提交：ID=%s\n", taskID)

	// 等待任务执行
	time.Sleep(100 * time.Millisecond)
}

// Example_withOptions 带配置选项的任务
func Example_withOptions() {
	queue := NewTaskQueue(DefaultConfig())
	queue.Start()
	defer queue.Stop()

	taskID, err := queue.SubmitFunc(
		"configured_task",
		func() error {
			fmt.Println("执行配置任务")
			return nil
		},
		WithPriority(80),           // 高优先级
		WithMaxRetries(5),          // 最多重试 5 次
		WithTimeout(5*time.Second), // 5 秒超时
		WithMetadata("user_id", 123),
	)

	if err != nil {
		fmt.Printf("提交任务失败：%v\n", err)
		return
	}

	fmt.Printf("任务已提交：ID=%s\n", taskID)
	time.Sleep(100 * time.Millisecond)
}

// Example_batchSubmit 批量提交示例
func Example_batchSubmit() {
	queue := NewTaskQueue(TaskQueueConfig{
		WorkerCount:  4,
		MaxQueueSize: 1000,
	})
	queue.Start()
	defer queue.Stop()

	// 批量创建任务
	tasks := make([]*Task, 0, 10)
	for i := 0; i < 10; i++ {
		task := NewTask(
			fmt.Sprintf("batch_task_%d", i),
			func() error {
				fmt.Println("执行批量任务")
				return nil
			},
			WithPriority(50),
		)
		tasks = append(tasks, task)
	}

	// 批量提交
	err := queue.pool.SubmitBatch(tasks)
	if err != nil {
		fmt.Printf("批量提交失败：%v\n", err)
		return
	}

	fmt.Printf("已批量提交 %d 个任务\n", len(tasks))
	time.Sleep(500 * time.Millisecond)
}

// Example_monitor 监控示例
func Example_monitor() {
	queue := NewTaskQueue(DefaultConfig())
	queue.Start()
	defer queue.Stop()

	// 提交一些任务
	for i := 0; i < 5; i++ {
		queue.SubmitFunc(
			fmt.Sprintf("monitor_task_%d", i),
			func() error {
				time.Sleep(10 * time.Millisecond)
				return nil
			},
		)
	}

	time.Sleep(500 * time.Millisecond)

	// 获取统计信息
	stats := queue.GetStats()
	fmt.Printf("总任务数：%d\n", stats.TotalTasks)
	fmt.Printf("成功任务数：%d\n", stats.SuccessTasks)
	fmt.Printf("失败任务数：%d\n", stats.FailedTasks)
	fmt.Printf("活跃工作器：%d/%d\n", stats.ActiveWorkers, stats.WorkerCount)
	fmt.Printf("平均执行时间：%d ms\n", stats.AvgExecuteTime/1e6)

	// 导出指标
	exporter := NewMetricsExporter(queue)
	metricsJSON, _ := exporter.ExportMetrics()
	fmt.Printf("指标 JSON: %s\n", string(metricsJSON))

	// 健康检查
	health := exporter.HealthCheck()
	fmt.Printf("健康状态：%s\n", health.Status)
}

// Example_retry 重试机制示例
func Example_retry() {
	queue := NewTaskQueue(DefaultConfig())
	queue.Start()
	defer queue.Stop()

	var attempt int
	maxAttempts := 3

	queue.SubmitFunc(
		"retry_example",
		func() error {
			attempt++
			fmt.Printf("执行尝试 %d/%d\n", attempt, maxAttempts)

			if attempt < maxAttempts {
				return fmt.Errorf("模拟失败 (尝试 %d)", attempt)
			}

			fmt.Println("任务执行成功")
			return nil
		},
		WithMaxRetries(3),
	)

	time.Sleep(500 * time.Millisecond)
}

// Example_timeout 超时控制示例
func Example_timeout() {
	queue := NewTaskQueue(DefaultConfig())
	queue.Start()
	defer queue.Stop()

	queue.SubmitFunc(
		"timeout_example",
		func() error {
			fmt.Println("开始执行长时间任务")
			time.Sleep(2 * time.Second) // 模拟长时间任务
			fmt.Println("任务完成")
			return nil
		},
		WithTimeout(500*time.Millisecond), // 设置 500ms 超时
	)

	time.Sleep(3 * time.Second)

	// 任务会因超时而失败
}
