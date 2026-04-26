package mq

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream" // 请替换为你的通用接口模块路径
)

// natsConnection 实现了 Connection 接口
type natsConnection struct {
	nc          *nats.Conn
	js          jetstream.JetStream
	streams     map[string]jetstream.Stream // 缓存已创建的 Stream 对象
	streamsLock sync.RWMutex
}

// NewNATSConnection 创建一个新的 NATS JetStream 连接
func NewNATSConnection(url string, opts ...nats.Option) (Connection, error) {
	nc, err := nats.Connect(url, opts...)
	if err != nil {
		return nil, fmt.Errorf("nats: failed to connect: %w", err)
	}

	// 使用新的 jetstream API 创建上下文
	js, err := jetstream.New(nc)
	if err != nil {
		nc.Close()
		return nil, fmt.Errorf("nats: failed to create jetstream context: %w", err)
	}

	return &natsConnection{
		nc:      nc,
		js:      js,
		streams: make(map[string]jetstream.Stream),
	}, nil
}

func (c *natsConnection) NewProducer(ctx context.Context, topic string) (Producer, error) {
	return &natsProducer{
		conn:  c,
		topic: topic,
	}, nil
}

func (c *natsConnection) NewConsumer(ctx context.Context, topic string, options ...ConsumerOption) (Consumer, error) {
	cfg := &ConsumerConfig{
		// 默认拉取模式，显式确认，预取10条消息
		Group:         "default",
		AutoAck:       false,
		PrefetchCount: 10,
		PollTimeout:   100 * time.Millisecond,
	}
	for _, opt := range options {
		opt(cfg)
	}

	return &natsConsumer{
		conn:       c,
		topic:      topic,
		cfg:        cfg,
		ctx:        ctx,
		cancelFunc: nil, // 在 Consume 中设置
	}, nil
}

func (c *natsConnection) getOrCreateStream(ctx context.Context, topic string) (jetstream.Stream, error) {
	// 快速检查缓存
	c.streamsLock.RLock()
	stream, ok := c.streams[topic]
	c.streamsLock.RUnlock()
	if ok {
		return stream, nil
	}

	c.streamsLock.Lock()
	defer c.streamsLock.Unlock()
	// 再次检查，防止在获取锁期间被其他 goroutine 创建
	if stream, ok := c.streams[topic]; ok {
		return stream, nil
	}

	// 创建或获取 Stream，名字根据主题派生
	streamName := fmt.Sprintf("STREAM_%s", topic)
	stream, err := c.js.CreateOrUpdateStream(ctx, jetstream.StreamConfig{
		Name:     streamName,
		Subjects: []string{topic}, // 使该流订阅此主题
	})
	if err != nil {
		return nil, fmt.Errorf("nats: failed to create stream for topic %s: %w", topic, err)
	}
	c.streams[topic] = stream
	return stream, nil
}

func (c *natsConnection) Close() error {
	c.streamsLock.Lock()
	c.streams = nil // 清空缓存
	c.streamsLock.Unlock()
	c.nc.Close()
	return nil
}

func (c *natsConnection) Ping(ctx context.Context) error {
	if !c.nc.IsConnected() {
		return ErrConnectionClosed
	}
	return nil
}

// natsProducer 实现了 Producer 接口
type natsProducer struct {
	conn  *natsConnection
	topic string
}

func (p *natsProducer) Send(ctx context.Context, msg *Message) error {
	if msg.Timestamp.IsZero() {
		msg.Timestamp = time.Now()
	}

	// 将通用消息转换为 JetStream 消息
	jsMsg := &nats.Msg{
		Subject: p.topic,
		Data:    msg.Payload,
		Header:  nats.Header{},
	}
	for k, v := range msg.Attributes {
		jsMsg.Header.Add(k, v)
	}

	// 使用 jetstream.PublishMsg，它会等待服务器确认
	_, err := p.conn.js.PublishMsg(ctx, jsMsg)
	return err
}

func (p *natsProducer) SendBatch(ctx context.Context, msgs []*Message) error {
	for _, msg := range msgs {
		if err := p.Send(ctx, msg); err != nil {
			return err
		}
	}
	return nil
}

func (p *natsProducer) Close() error {
	// 无资源需要释放
	return nil
}

// natsConsumer 实现了 Consumer 接口
type natsConsumer struct {
	conn       *natsConnection
	topic      string
	cfg        *ConsumerConfig
	ctx        context.Context
	cancelFunc context.CancelFunc
	cons       jetstream.Consumer
	cc         jetstream.ConsumeContext // 用于 Consume 方法的上下文
}

func (c *natsConsumer) Consume(ctx context.Context, handler func(context.Context, *Message) error) error {
	stream, err := c.conn.getOrCreateStream(ctx, c.topic)
	if err != nil {
		return err
	}

	// 创建或更新一个持久的 Pull Consumer
	consumerName := fmt.Sprintf("CONSUMER_%s_%s", c.topic, c.cfg.Group)
	cons, err := stream.CreateOrUpdateConsumer(ctx, jetstream.ConsumerConfig{
		Durable:       consumerName, // 持久的消费者，重启后可以继续
		AckPolicy:     jetstream.AckExplicitPolicy,
		FilterSubject: c.topic,
		MaxWaiting:    c.cfg.PrefetchCount, // 控制预取数量
	})
	if err != nil {
		return fmt.Errorf("nats: failed to create consumer: %w", err)
	}
	c.cons = cons

	// 创建一个可取消的上下文，用于消费循环
	consumeCtx, cancel := context.WithCancel(ctx)
	c.cancelFunc = cancel
	defer cancel()

	// 使用 jetstream.Consumer.Consume 方法进行拉取消费
	cc, err := cons.Consume(func(msg jetstream.Msg) {
		// 将 jetstream.Msg 转换为通用 Message
		mqMsg := &Message{
			Payload:    msg.Data(),
			Attributes: make(map[string]string),
			Timestamp:  time.Now(),
			AckFunc: func() error {
				// 使用 Double-Ack 确保服务器收到确认
				return msg.Ack()
			},
			NackFunc: func(requeue bool) error {
				if requeue {
					return msg.Nak()
				}
				return msg.Term()
			},
		}
		for key, values := range msg.Headers() {
			if len(values) > 0 {
				mqMsg.Attributes[key] = values[0]
			}
		}
		// 调用用户的处理函数
		if err := handler(ctx, mqMsg); err != nil {
			// 如果处理失败且未自动确认，则进行否定确认
			if !c.cfg.AutoAck {
				// 根据配置决定是 Nak (重试) 还是 Term (终止)
				if c.cfg.MaxRetries > 0 {
					// 这里可以通过 header 记录重试次数，实现更复杂的逻辑
					_ = msg.Nak()
				} else {
					_ = msg.Term()
				}
			}
			return
		}
		// 如果处理成功且未自动确认，则进行确认
		if !c.cfg.AutoAck {
			_ = msg.Ack()
		}
	})
	if err != nil {
		return fmt.Errorf("nats: failed to start consuming: %w", err)
	}
	c.cc = cc

	// 等待上下文结束 (无论是外部取消还是内部错误)
	<-consumeCtx.Done()
	return consumeCtx.Err()
}

func (c *natsConsumer) Close() error {
	if c.cancelFunc != nil {
		c.cancelFunc()
	}
	if c.cc != nil {
		c.cc.Stop()
	}
	// 不需要删除 Consumer，因为它是持久的，以便下次可以继续使用
	return nil
}
