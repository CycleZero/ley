# Task Queue - 企业级本地任务队列

高性能、高效率、高可靠性的本地任务队列系统，支持多工作器并发执行。

## 特性

- **高性能**: 基于优先级队列的任务调度，支持批量提交
- **多工作器**: 自动根据 CPU 核心数配置工作器数量，支持自定义
- **任务重试**: 内置重试机制，可配置最大重试次数
- **超时控制**: 支持任务超时自动终止
- **优先级调度**: 支持 0-100 优先级，数值越高优先级越高
- **监控指标**: 实时统计任务执行情况，支持健康检查
- **线程安全**: 完全线程安全，支持高并发提交

## 快速开始

### 基本使用

```go
package main

import (
    "leyline-doc-backend/pkg/task"
    "time"
)

func main() {
    // 创建任务队列
    queue := task.NewTaskQueue(task.DefaultConfig())
    queue.Start()
    defer queue.Stop()
    
    // 提交任务
    taskID, err := queue.SubmitFunc("my_task", func() error {
        // 任务逻辑
        return nil
    })
    
    if err != nil {
        // 处理错误
    }
    
    // 等待任务执行
    time.Sleep(100 * time.Millisecond)
}
```

### 配置选项

```go
queue := task.NewTaskQueue(task.TaskQueueConfig{
    WorkerCount:  8,      // 工作器数量 (0=自动)
    MaxQueueSize: 10000,  // 最大队列大小
})

// 提交带配置的任务
taskID, err := queue.SubmitFunc(
    "configured_task",
    taskFunc,
    task.WithPriority(80),            // 高优先级 (0-100)
    task.WithMaxRetries(5),           // 最多重试 5 次
    task.WithTimeout(5*time.Second),  // 5 秒超时
    task.WithMetadata("key", value),  // 元数据
)
```

### 批量提交

```go
tasks := make([]*task.Task, 0, 10)
for i := 0; i < 10; i++ {
    t := task.NewTask(
        fmt.Sprintf("batch_task_%d", i),
        taskFunc,
        task.WithPriority(50),
    )
    tasks = append(tasks, t)
}

err := queue.pool.SubmitBatch(tasks)
```

### 监控和指标

```go
// 获取统计信息
stats := queue.GetStats()
fmt.Printf("总任务数：%d\n", stats.TotalTasks)
fmt.Printf("成功任务数：%d\n", stats.SuccessTasks)
fmt.Printf("活跃工作器：%d/%d\n", stats.ActiveWorkers, stats.WorkerCount)

// 导出指标 JSON
exporter := task.NewMetricsExporter(queue)
metricsJSON, _ := exporter.ExportMetrics()

// 健康检查
health := exporter.HealthCheck()
if health.Status != "healthy" {
    // 处理不健康状态
}
```

## API 参考

### TaskQueue

| 方法 | 说明 |
|------|------|
| `NewTaskQueue(config)` | 创建任务队列 |
| `Start()` | 启动任务队列 |
| `Stop()` | 停止任务队列 |
| `Submit(task)` | 提交任务 |
| `SubmitFunc(name, fn, opts...)` | 提交任务函数 |
| `GetTask(taskID)` | 获取任务 |
| `CancelTask(taskID)` | 取消任务 |
| `GetStats()` | 获取统计信息 |
| `GetQueueSize()` | 获取队列大小 |

### Task 选项

| 函数 | 说明 |
|------|------|
| `WithPriority(n)` | 设置优先级 (0-100) |
| `WithMaxRetries(n)` | 设置最大重试次数 |
| `WithTimeout(d)` | 设置超时时间 |
| `WithMetadata(k, v)` | 设置元数据 |

### 任务状态

| 状态 | 说明 |
|------|------|
| `TaskStatusPending` | 等待执行 |
| `TaskStatusRunning` | 执行中 |
| `TaskStatusSuccess` | 执行成功 |
| `TaskStatusFailed` | 执行失败 |
| `TaskStatusCancelled` | 已取消 |

## 统计指标

| 指标 | 说明 |
|------|------|
| `TotalTasks` | 总任务数 |
| `SuccessTasks` | 成功任务数 |
| `FailedTasks` | 失败任务数 |
| `RetryTasks` | 重试任务数 |
| `ActiveWorkers` | 活跃工作器数 |
| `QueuedTasks` | 排队任务数 |
| `AvgExecuteTime` | 平均执行时间 (纳秒) |

## 最佳实践

1. **合理设置工作器数量**: 默认根据 CPU 核心数自动配置，IO 密集型任务可适当增加
2. **设置合适的超时时间**: 避免任务长时间占用工作器
3. **配置重试机制**: 对于可能失败的任务，配置适当的重试次数
4. **使用优先级**: 重要任务设置高优先级
5. **监控队列状态**: 定期检查队列大小和失败率
6. **优雅关闭**: 使用 `defer queue.Stop()` 确保资源释放

## 测试

```bash
cd pkg/task
go test -v -race
```

## 注意事项

- 任务函数应该是幂等的，因为可能会重试执行
- 避免在任务函数中执行阻塞操作，如需阻塞请使用超时控制
- 任务队列停止时会等待所有执行中的任务完成
- 取消操作只对等待执行的任务有效，执行中的任务无法取消
