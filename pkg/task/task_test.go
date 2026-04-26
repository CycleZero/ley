package task

import (
	"errors"
	"fmt"
	"sync/atomic"
	"testing"
	"time"
)

func TestMain(m *testing.M) {
	// 初始化测试日志器
	initTestLogger()
	m.Run()
}

func TestTaskQueue_Basic(t *testing.T) {
	// 创建任务队列
	queue := NewTaskQueue(DefaultConfig())
	queue.Start()
	defer queue.Stop()

	// 提交简单任务
	var executed int32
	taskID, err := queue.SubmitFunc("test_task", func() error {
		atomic.AddInt32(&executed, 1)
		return nil
	})

	if err != nil {
		t.Fatalf("提交任务失败：%v", err)
	}

	// 等待任务执行
	time.Sleep(100 * time.Millisecond)

	// 验证任务执行
	if atomic.LoadInt32(&executed) != 1 {
		t.Errorf("期望执行 1 次，实际执行 %d 次", atomic.LoadInt32(&executed))
	}

	// 验证任务状态
	task, ok := queue.GetTask(taskID)
	if !ok {
		t.Fatal("未找到任务")
	}

	if task.Status != TaskStatusSuccess {
		t.Errorf("期望任务状态为 success，实际为 %s", task.Status)
	}
}

func TestTaskQueue_Priority(t *testing.T) {
	queue := NewTaskQueue(TaskQueueConfig{
		WorkerCount:  1, // 单工作器以确保顺序
		MaxQueueSize: 100,
	})
	queue.Start()
	defer queue.Stop()

	var executionOrder []int

	// 提交不同优先级的任务
	queue.SubmitFunc("low_priority", func() error {
		executionOrder = append(executionOrder, 1)
		return nil
	}, WithPriority(10))

	queue.SubmitFunc("high_priority", func() error {
		executionOrder = append(executionOrder, 3)
		return nil
	}, WithPriority(90))

	queue.SubmitFunc("medium_priority", func() error {
		executionOrder = append(executionOrder, 2)
		return nil
	}, WithPriority(50))

	// 等待所有任务执行
	time.Sleep(500 * time.Millisecond)

	// 验证执行顺序（高优先级先执行）
	if len(executionOrder) != 3 {
		t.Fatalf("期望执行 3 个任务，实际执行 %d 个", len(executionOrder))
	}

	// 高优先级应该先执行
	if executionOrder[0] != 3 {
		t.Errorf("期望高优先级先执行，实际执行顺序：%v", executionOrder)
	}
}

func TestTaskQueue_Retry(t *testing.T) {
	queue := NewTaskQueue(DefaultConfig())
	queue.Start()
	defer queue.Stop()

	var attempt int32
	maxAttempts := int32(3)

	taskID, err := queue.SubmitFunc("retry_task", func() error {
		count := atomic.AddInt32(&attempt, 1)
		if count < maxAttempts {
			return errors.New("模拟失败")
		}
		return nil
	}, WithMaxRetries(3))

	if err != nil {
		t.Fatalf("提交任务失败：%v", err)
	}

	// 等待任务执行和重试
	time.Sleep(500 * time.Millisecond)

	// 验证重试次数
	if atomic.LoadInt32(&attempt) != maxAttempts {
		t.Errorf("期望执行 %d 次，实际执行 %d 次", maxAttempts, atomic.LoadInt32(&attempt))
	}

	// 验证最终状态
	task, ok := queue.GetTask(taskID)
	if !ok {
		t.Fatal("未找到任务")
	}

	if task.Status != TaskStatusSuccess {
		t.Errorf("期望最终状态为 success，实际为 %s", task.Status)
	}
}

func TestTaskQueue_Timeout(t *testing.T) {
	queue := NewTaskQueue(DefaultConfig())
	queue.Start()
	defer queue.Stop()

	taskID, err := queue.SubmitFunc("timeout_task", func() error {
		time.Sleep(2 * time.Second) // 超过超时时间
		return nil
	}, WithTimeout(100*time.Millisecond))

	if err != nil {
		t.Fatalf("提交任务失败：%v", err)
	}

	// 等待任务超时
	time.Sleep(300 * time.Millisecond)

	// 验证任务状态
	task, ok := queue.GetTask(taskID)
	if !ok {
		t.Fatal("未找到任务")
	}

	if task.Status != TaskStatusFailed {
		t.Errorf("期望任务状态为 failed，实际为 %s", task.Status)
	}
}

func TestTaskQueue_BatchSubmit(t *testing.T) {
	queue := NewTaskQueue(TaskQueueConfig{
		WorkerCount:  4,
		MaxQueueSize: 100,
	})
	queue.Start()
	defer queue.Stop()

	var completed int32

	// 批量创建任务
	tasks := make([]*Task, 0, 10)
	for i := 0; i < 10; i++ {
		task := NewTask(fmt.Sprintf("batch_task_%d", i), func() error {
			atomic.AddInt32(&completed, 1)
			time.Sleep(10 * time.Millisecond)
			return nil
		})
		tasks = append(tasks, task)
	}

	err := queue.pool.SubmitBatch(tasks)
	if err != nil {
		t.Fatalf("批量提交失败：%v", err)
	}

	// 等待所有任务完成
	time.Sleep(500 * time.Millisecond)

	// 验证完成数量
	if atomic.LoadInt32(&completed) != 10 {
		t.Errorf("期望完成 10 个任务，实际完成 %d 个", atomic.LoadInt32(&completed))
	}
}

func TestTaskQueue_Metrics(t *testing.T) {
	queue := NewTaskQueue(DefaultConfig())
	queue.Start()
	defer queue.Stop()

	// 提交一些任务
	for i := 0; i < 5; i++ {
		queue.SubmitFunc(fmt.Sprintf("metrics_task_%d", i), func() error {
			time.Sleep(10 * time.Millisecond)
			return nil
		})
	}

	// 等待任务完成
	time.Sleep(500 * time.Millisecond)

	// 获取统计信息
	stats := queue.GetStats()

	if stats.TotalTasks != 5 {
		t.Errorf("期望总任务数为 5，实际为 %d", stats.TotalTasks)
	}

	if stats.SuccessTasks != 5 {
		t.Errorf("期望成功任务数为 5，实际为 %d", stats.SuccessTasks)
	}

	// 测试指标导出
	exporter := NewMetricsExporter(queue)
	metricsJSON, err := exporter.ExportMetrics()
	if err != nil {
		t.Fatalf("导出指标失败：%v", err)
	}

	if len(metricsJSON) == 0 {
		t.Error("指标 JSON 为空")
	}

	t.Logf("指标：%s", string(metricsJSON))
}

func TestTaskQueue_HealthCheck(t *testing.T) {
	queue := NewTaskQueue(DefaultConfig())
	queue.Start()
	defer queue.Stop()

	// 提交一些慢速任务
	for i := 0; i < 3; i++ {
		queue.SubmitFunc(fmt.Sprintf("health_task_%d", i), func() error {
			time.Sleep(100 * time.Millisecond)
			return nil
		})
	}

	// 立即检查健康状态（任务应该正在执行）
	time.Sleep(10 * time.Millisecond)
	exporter := NewMetricsExporter(queue)
	health := exporter.HealthCheck()

	t.Logf("健康状态：%+v", health)

	// 验证健康检查返回了数据
	if health.Timestamp == "" {
		t.Error("期望有时间戳")
	}
	if health.Checks == nil {
		t.Error("期望有检查项")
	}
}

func TestTaskQueue_ConcurrentSubmit(t *testing.T) {
	queue := NewTaskQueue(TaskQueueConfig{
		WorkerCount:  8,
		MaxQueueSize: 1000,
	})
	queue.Start()
	defer queue.Stop()

	var completed int32
	taskCount := 100

	// 并发提交任务
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func() {
			for j := 0; j < taskCount/10; j++ {
				queue.SubmitFunc("concurrent_task", func() error {
					atomic.AddInt32(&completed, 1)
					time.Sleep(time.Millisecond)
					return nil
				})
			}
			done <- true
		}()
	}

	// 等待所有提交完成
	for i := 0; i < 10; i++ {
		<-done
	}

	// 等待所有任务执行
	time.Sleep(2 * time.Second)

	// 验证完成数量
	if atomic.LoadInt32(&completed) != int32(taskCount) {
		t.Errorf("期望完成 %d 个任务，实际完成 %d 个", taskCount, atomic.LoadInt32(&completed))
	}

	stats := queue.GetStats()
	t.Logf("统计信息：总任务=%d, 成功=%d, 失败=%d",
		stats.TotalTasks, stats.SuccessTasks, stats.FailedTasks)
}

// 基准测试
func BenchmarkTaskQueue_Submit(b *testing.B) {
	queue := NewTaskQueue(TaskQueueConfig{
		WorkerCount:  8,
		MaxQueueSize: 10000,
	})
	queue.Start()
	defer queue.Stop()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		queue.SubmitFunc("bench_task", func() error {
			return nil
		})
	}
}

func BenchmarkTaskQueue_SubmitWithExecution(b *testing.B) {
	queue := NewTaskQueue(TaskQueueConfig{
		WorkerCount:  8,
		MaxQueueSize: 10000,
	})
	queue.Start()
	defer queue.Stop()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		queue.SubmitFunc("bench_task", func() error {
			time.Sleep(time.Microsecond)
			return nil
		})
	}
}
