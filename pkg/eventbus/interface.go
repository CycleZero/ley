// Package eventbus 提供事件总线抽象
//
// 本包定义 EventBus 接口、事件信封 Envelope 以及各领域事件结构体。
// 核心设计原则：
//   - 调用方只依赖 EventBus 接口，不依赖具体实现（NATS/InMemory）
//   - 所有事件通过 Envelope 包装，携带 event_id、occurred_at、version 等元数据
//   - PublishAsync 用于非关键事件（fire-and-forget），PublishSync 用于关键事件
//   - 消费者通过 Subscribe 注册 EventHandler，接收完整的 Envelope
package eventbus

import (
	"context"
	"encoding/json"
	"time"

	"ley/pkg/mq"
)

// =============================================================================
// EventBus — 事件总线接口（依赖倒置）
// =============================================================================

// EventBus 事件总线接口
// 定义事件发布和订阅的标准契约，与底层消息中间件（NATS/Kafka/内存）解耦。
type EventBus interface {
	// PublishAsync 异步发布事件（fire-and-forget）
	// 事件在独立 goroutine 中发送，不阻塞主流程。
	// 调用方收到 nil 仅表示事件已提交到内部队列，不代表 broker 已确认。
	// 适用于非关键事件（通知、统计），发布失败仅记录日志。
	PublishAsync(ctx context.Context, topic string, event interface{}) error

	// PublishSync 同步发布事件，等待 broker 确认（或超时）
	// 调用方收到 nil 表示 broker 已确认收到。超时或失败返回 error。
	// 适用于关键事件，调用方可根据 error 决定重试或回滚。
	PublishSync(ctx context.Context, topic string, event interface{}, timeout time.Duration) error

	// Subscribe 订阅指定主题的事件
	// handler 在独立 goroutine 中处理每条消息，返回 nil 表示处理成功（触发 Ack），
	// 返回 error 表示处理失败（触发 Nack，根据配置决定重试或进入 DLQ）。
	// 返回的 Subscription 可用于取消订阅。
	Subscribe(ctx context.Context, topic string, handler EventHandler, opts ...mq.ConsumerOption) (Subscription, error)

	// Close 优雅关闭事件总线，等待 pending 事件发送完成
	Close() error
}

// =============================================================================
// EventHandler — 事件处理器
// =============================================================================

// EventHandler 事件处理器函数类型
// 接收上下文和事件信封，处理完成后返回 nil 表示 Ack，返回 error 表示 Nack。
type EventHandler func(ctx context.Context, envelope *Envelope) error

// =============================================================================
// Subscription — 订阅句柄
// =============================================================================

// Subscription 表示一个活跃的事件订阅
type Subscription interface {
	// Unsubscribe 取消订阅，停止接收事件
	Unsubscribe() error
}

// =============================================================================
// Envelope — 事件信封
// =============================================================================

// Envelope 事件信封，为每个事件添加元数据
// 消费者收到 Envelope 后，根据 Topic 和 EventType 决定如何处理 Data。
type Envelope struct {
	EventID    string          `json:"event_id"`    // 事件唯一 ID（UUID v7，用于去重和排查）
	EventType  string          `json:"event_type"`  // 事件类型名（Go 结构体简称，如 "ArticleCreated"）
	Topic      string          `json:"topic"`       // 事件主题（领域分类，如 "article.created"）
	Source     string          `json:"source"`      // 事件来源（服务名+实例 ID）
	Version    string          `json:"version"`     // 事件结构版本（向前兼容用）
	OccurredAt time.Time       `json:"occurred_at"` // 事件发生时间
	Data       json.RawMessage `json:"data"`        // 事件数据（JSON 序列化的事件结构体）
}

// UnmarshalData 将信封中的数据反序列化为指定类型
func (e *Envelope) UnmarshalData(v interface{}) error {
	return json.Unmarshal(e.Data, v)
}

// =============================================================================
// EventBusConfig — 事件总线配置
// =============================================================================

// EventBusConfig 事件总线配置
type EventBusConfig struct {
	// Source 事件来源标识（如 "blog-service"），写入 Envelope.Source
	Source string
	// PublishTimeout 异步发布的超时时间（默认 5s）
	PublishTimeout time.Duration
	// PendingBufferSize 异步发布内部缓冲大小（默认 1024）
	PendingBufferSize int
}

// DefaultEventBusConfig 返回默认配置
func DefaultEventBusConfig() EventBusConfig {
	return EventBusConfig{
		Source:            "ley-unknown",
		PublishTimeout:    5 * time.Second,
		PendingBufferSize: 1024,
	}
}
