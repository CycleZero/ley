// Package eventbus 事件总线封装
//
// 基于 pkg/mq 抽象层提供发布/订阅能力，支持 NATS JetStream 与内存实现。
// 调用方依赖 EventBus 接口，不依赖具体实现。
package eventbus

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/CycleZero/ley/pkg/mq"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/uuid"
)

// =============================================================================
// eventBus — EventBus 接口实现
// =============================================================================

type eventBus struct {
	conn      mq.Connection
	logger    log.Logger
	config    EventBusConfig
	producers map[string]mq.Producer
	mu        sync.RWMutex
	closed    atomic.Bool
	stopCh    chan struct{}
	pending   chan *pendingEvent
	workerWg  sync.WaitGroup
}

type pendingEvent struct {
	ctx       context.Context
	topic     string
	payload   []byte
	eventType string
	eventID   string
}

// NewEventBus 创建事件总线实例
func NewEventBus(conn mq.Connection, config EventBusConfig, logger log.Logger) EventBus {
	if config.PublishTimeout <= 0 {
		config.PublishTimeout = 5 * time.Second
	}
	if config.PendingBufferSize <= 0 {
		config.PendingBufferSize = 1024
	}
	if config.Source == "" {
		config.Source = "ley-unknown"
	}

	eb := &eventBus{
		conn:      conn,
		logger:    logger,
		config:    config,
		producers: make(map[string]mq.Producer),
		stopCh:    make(chan struct{}),
		pending:   make(chan *pendingEvent, config.PendingBufferSize),
	}

	eb.workerWg.Add(1)
	go eb.worker()

	return eb
}

// =============================================================================
// PublishAsync — 异步发布（fire-and-forget）
// =============================================================================

func (eb *eventBus) PublishAsync(ctx context.Context, topic string, eventData interface{}) error {
	if eb.closed.Load() {
		return ErrEventBusClosed
	}

	payload, err := json.Marshal(eventData)
	if err != nil {
		eb.logHelper(ctx).Warnw("事件序列化失败", "topic", topic, "error", err)
		return err
	}

	// context.WithoutCancel: 保留 trace/metadata，但解绑 cancel（异步执行不受原请求取消影响）
	asyncCtx := context.WithoutCancel(ctx)

	ev := &pendingEvent{
		ctx:       asyncCtx,
		topic:     topic,
		payload:   payload,
		eventType: typeName(eventData),
		eventID:   uuid.Must(uuid.NewV7()).String(),
	}

	select {
	case eb.pending <- ev:
		return nil
	default:
		eb.logHelper(ctx).Warnw("事件缓冲区已满", "topic", topic, "event_type", ev.eventType)
		return ErrEventBusFull
	}
}

// =============================================================================
// PublishSync — 同步发布（等待 broker 确认）
// =============================================================================

func (eb *eventBus) PublishSync(ctx context.Context, topic string, eventData interface{}, timeout time.Duration) error {
	if eb.closed.Load() {
		return ErrEventBusClosed
	}

	pubCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	raw, err := json.Marshal(eventData)
	if err != nil {
		return err
	}

	return eb.publishRaw(pubCtx, topic, raw, typeName(eventData))
}

// =============================================================================
// Subscribe — 订阅事件
// =============================================================================

func (eb *eventBus) Subscribe(ctx context.Context, topic string, handler EventHandler, opts ...mq.ConsumerOption) (Subscription, error) {
	consumer, err := eb.conn.NewConsumer(ctx, topic, opts...)
	if err != nil {
		return nil, err
	}

	subCtx, cancel := context.WithCancel(ctx)

	go func() {
		_ = consumer.Consume(subCtx, func(ctx context.Context, msg *mq.Message) error {
			var envelope Envelope
			if err := json.Unmarshal(msg.Payload, &envelope); err != nil {
				eb.logHelper(ctx).Warnw("事件信封反序列化失败", "topic", topic, "error", err)
				return err
			}
			return handler(ctx, &envelope)
		})
	}()

	return &subscription{cancel: cancel, consumer: consumer}, nil
}

// =============================================================================
// Close — 优雅关闭
// =============================================================================

func (eb *eventBus) Close() error {
	if eb.closed.Swap(true) {
		return nil
	}

	// 通知 worker 停止，然后等待退出
	close(eb.stopCh)
	close(eb.pending)
	eb.workerWg.Wait()

	eb.mu.Lock()
	defer eb.mu.Unlock()

	var errs []error
	for topic, producer := range eb.producers {
		if err := producer.Close(); err != nil {
			errs = append(errs, err)
			eb.logHelper(context.Background()).Warnw("关闭生产者失败", "topic", topic, "error", err)
		}
	}
	eb.producers = nil

	if err := eb.conn.Close(); err != nil {
		errs = append(errs, err)
	}

	if len(errs) > 0 {
		return fmt.Errorf("eventbus close: %v", errs)
	}
	return nil
}

// =============================================================================
// 内部方法
// =============================================================================

func (eb *eventBus) worker() {
	defer eb.workerWg.Done()
	for ev := range eb.pending {
		// 检查是否收到停止信号，若已停止则丢弃剩余事件直接退出
		select {
		case <-eb.stopCh:
			return
		default:
		}

		pubCtx, cancel := context.WithTimeout(ev.ctx, eb.config.PublishTimeout)
		err := eb.publishRaw(pubCtx, ev.topic, ev.payload, ev.eventType)
		cancel()
		if err != nil {
			eb.logHelper(ev.ctx).Warnw("异步发布事件失败",
				"topic", ev.topic, "event_type", ev.eventType, "event_id", ev.eventID, "error", err)
		}
	}
}

// publishRaw 核心发布逻辑：用已序列化的 payload 构造 Envelope 并发送
func (eb *eventBus) publishRaw(ctx context.Context, topic string, rawPayload []byte, et string) error {
	envelope := Envelope{
		EventID:    uuid.Must(uuid.NewV7()).String(),
		EventType:  et,
		Topic:      topic,
		Source:     eb.config.Source,
		Version:    "1.0",
		OccurredAt: time.Now(),
		Data:       rawPayload,
	}

	payload, err := json.Marshal(envelope)
	if err != nil {
		return err
	}

	producer, err := eb.getOrCreateProducer(ctx, topic)
	if err != nil {
		return err
	}

	msg := &mq.Message{
		Payload:   payload,
		Timestamp: time.Now(),
		Attributes: map[string]string{
			"topic":        topic,
			"event_type":   et,
			"event_id":     envelope.EventID,
			"event_source": eb.config.Source,
		},
	}

	return producer.Send(ctx, msg)
}

func (eb *eventBus) getOrCreateProducer(ctx context.Context, topic string) (mq.Producer, error) {
	eb.mu.RLock()
	if p, ok := eb.producers[topic]; ok {
		eb.mu.RUnlock()
		return p, nil
	}
	eb.mu.RUnlock()

	eb.mu.Lock()
	defer eb.mu.Unlock()

	if p, ok := eb.producers[topic]; ok {
		return p, nil
	}

	producer, err := eb.conn.NewProducer(ctx, topic)
	if err != nil {
		return nil, err
	}
	eb.producers[topic] = producer
	return producer, nil
}

func (eb *eventBus) logHelper(ctx context.Context) *log.Helper {
	return log.NewHelper(log.WithContext(ctx, eb.logger))
}

// =============================================================================
// subscription — Subscription 接口实现
// =============================================================================

type subscription struct {
	cancel   context.CancelFunc
	consumer mq.Consumer
}

func (s *subscription) Unsubscribe() error {
	s.cancel()
	return s.consumer.Close()
}

// =============================================================================
// 错误定义
// =============================================================================

var (
	ErrEventBusClosed = &EventBusError{"event bus is closed"}
	ErrEventBusFull   = &EventBusError{"event bus buffer is full"}
)

type EventBusError struct {
	msg string
}

func NewEventBusError(msg string) *EventBusError { return &EventBusError{msg: msg} }
func (e *EventBusError) Error() string           { return e.msg }

// =============================================================================
// 工具函数
// =============================================================================

// typeName 从 fmt.Sprintf("%T", v) 提取简短类型名
// "*pkg.eventbus.ArticleCreatedEvent" → "ArticleCreatedEvent"
// "eventbus.ArticleCreatedEvent"     → "ArticleCreatedEvent"
func typeName(v interface{}) string {
	s := fmt.Sprintf("%T", v)
	if idx := strings.LastIndexByte(s, '.'); idx >= 0 {
		return s[idx+1:]
	}
	return s
}
