package mq

import (
	"context"
	"errors"
	"io"
	"time"
)

// 基础错误定义
var (
	ErrNotSupported     = errors.New("mq: operation not supported")
	ErrConnectionClosed = errors.New("mq: connection closed")
	ErrNoMessage        = errors.New("mq: no message available")
)

// Message 表示一条消息，包含通用元数据和负载
type Message struct {
	// ID 消息唯一标识，可能由中间件生成
	ID string
	// Key 路由键或分区键
	Key string
	// Payload 消息体
	Payload []byte
	// Attributes 扩展属性，如 headers
	Attributes map[string]string
	// Timestamp 消息产生时间
	Timestamp time.Time
	// AckFunc 确认函数，调用表示消息处理成功
	AckFunc func() error
	// NackFunc 否定确认，可要求重试
	NackFunc func(requeue bool) error
}

// Producer 定义消息生产者接口
type Producer interface {
	// Send 发送单条消息，同步等待确认
	Send(ctx context.Context, msg *Message) error
	// SendBatch 批量发送，提高吞吐
	SendBatch(ctx context.Context, msgs []*Message) error
	// Close 释放生产者资源
	io.Closer
}

// Consumer 定义消息消费者接口
type Consumer interface {
	// Consume 开始消费消息，通过回调函数处理每条消息
	// handler 返回 nil 表示处理成功，将自动 Ack；返回 error 表示失败，将 Nack
	// 该方法应阻塞直到上下文取消或发生致命错误
	Consume(ctx context.Context, handler func(ctx context.Context, msg *Message) error) error
	// Close 停止消费并释放资源
	io.Closer
}

// Connection 代表与消息中间件的连接，可创建生产者和消费者
type Connection interface {
	// NewProducer 创建生产者，topic 为默认主题/队列名
	NewProducer(ctx context.Context, topic string) (Producer, error)
	// NewConsumer 创建消费者，可指定消费组等配置
	NewConsumer(ctx context.Context, topic string, options ...ConsumerOption) (Consumer, error)
	// Close 关闭连接
	io.Closer
	// Ping 检查连接是否存活
	Ping(ctx context.Context) error
}

// ConsumerOption 用于定制消费者行为
type ConsumerOption func(*ConsumerConfig)

type ConsumerConfig struct {
	Group         string        // 消费组名称
	AutoAck       bool          // 是否自动确认
	PrefetchCount int           // 预取数量
	MaxRetries    int           // 最大重试次数
	PollTimeout   time.Duration // 轮询超时
}

func WithConsumerGroup(group string) ConsumerOption {
	return func(c *ConsumerConfig) {
		c.Group = group
	}
}

func WithAutoAck(enabled bool) ConsumerOption {
	return func(c *ConsumerConfig) {
		c.AutoAck = enabled
	}
}

func WithPrefetchCount(n int) ConsumerOption {
	return func(c *ConsumerConfig) {
		c.PrefetchCount = n
	}
}

func WithPollTimeout(d time.Duration) ConsumerOption {
	return func(c *ConsumerConfig) {
		c.PollTimeout = d
	}
}

// ========== 内存实现示例 ==========

// 简单的内存队列实现，可用于测试或单进程场景
type memoryConnection struct {
	queues map[string]chan *Message
	closed bool
}

func NewMemoryConnection() Connection {
	return &memoryConnection{
		queues: make(map[string]chan *Message),
	}
}

func (c *memoryConnection) NewProducer(ctx context.Context, topic string) (Producer, error) {
	if c.closed {
		return nil, ErrConnectionClosed
	}
	c.ensureQueue(topic)
	return &memoryProducer{
		conn:  c,
		topic: topic,
	}, nil
}

func (c *memoryConnection) NewConsumer(ctx context.Context, topic string, opts ...ConsumerOption) (Consumer, error) {
	if c.closed {
		return nil, ErrConnectionClosed
	}
	cfg := &ConsumerConfig{
		PollTimeout: 100 * time.Millisecond,
	}
	for _, opt := range opts {
		opt(cfg)
	}
	c.ensureQueue(topic)
	return &memoryConsumer{
		conn:        c,
		topic:       topic,
		autoAck:     cfg.AutoAck,
		pollTimeout: cfg.PollTimeout,
	}, nil
}

func (c *memoryConnection) ensureQueue(topic string) {
	if _, ok := c.queues[topic]; !ok {
		c.queues[topic] = make(chan *Message, 1000)
	}
}

func (c *memoryConnection) Close() error {
	if c.closed {
		return nil
	}
	c.closed = true
	for _, ch := range c.queues {
		close(ch)
	}
	return nil
}

func (c *memoryConnection) Ping(ctx context.Context) error {
	if c.closed {
		return ErrConnectionClosed
	}
	return nil
}

type memoryProducer struct {
	conn  *memoryConnection
	topic string
}

func (p *memoryProducer) Send(ctx context.Context, msg *Message) error {
	if p.conn.closed {
		return ErrConnectionClosed
	}
	if msg.Timestamp.IsZero() {
		msg.Timestamp = time.Now()
	}
	// 设置默认 Ack/Nack 函数
	if msg.AckFunc == nil {
		msg.AckFunc = func() error { return nil }
	}
	if msg.NackFunc == nil {
		msg.NackFunc = func(bool) error { return nil }
	}

	select {
	case p.conn.queues[p.topic] <- msg:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (p *memoryProducer) SendBatch(ctx context.Context, msgs []*Message) error {
	for _, msg := range msgs {
		if err := p.Send(ctx, msg); err != nil {
			return err
		}
	}
	return nil
}

func (p *memoryProducer) Close() error {
	// 内存生产者无独立资源
	return nil
}

type memoryConsumer struct {
	conn        *memoryConnection
	topic       string
	autoAck     bool
	pollTimeout time.Duration
}

func (c *memoryConsumer) Consume(ctx context.Context, handler func(context.Context, *Message) error) error {
	for {
		select {
		case msg, ok := <-c.conn.queues[c.topic]:
			if !ok {
				return ErrConnectionClosed
			}
			if err := handler(ctx, msg); err != nil {
				// 处理失败，根据配置决定是否 Nack
				if !c.autoAck && msg.NackFunc != nil {
					_ = msg.NackFunc(true)
				}
				// 继续处理下一条（或可返回错误终止消费）
				continue
			}
			// 成功
			if !c.autoAck && msg.AckFunc != nil {
				_ = msg.AckFunc()
			}
		case <-ctx.Done():
			return ctx.Err()
		default:
			// 非阻塞检查，防止忙等
			time.Sleep(c.pollTimeout)
		}
	}
}

func (c *memoryConsumer) Close() error {
	// 内存消费者无独立资源
	return nil
}
