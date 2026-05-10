package eventbus

import (
	"context"
	"encoding/json"
	"io"
	"sync/atomic"
	"testing"
	"time"

	"github.com/CycleZero/ley/pkg/mq"

	klog "github.com/go-kratos/kratos/v2/log"
)

func newTestEB() EventBus {
	return NewEventBus(mq.NewMemoryConnection(256), EventBusConfig{
		Source:            "ley-test",
		PublishTimeout:    3 * time.Second,
		PendingBufferSize: 64,
	}, klog.NewStdLogger(io.Discard))
}

func TestEventBus_PublishAsync(t *testing.T) {
	t.Run("不阻塞调用方", func(t *testing.T) {
		eb := newTestEB()
		defer eb.Close()

		err := eb.PublishAsync(context.Background(), "test.topic", map[string]string{"k": "v"})
		if err != nil {
			t.Fatalf("PublishAsync should not error, got %v", err)
		}
	})

	t.Run("消费者能收到事件", func(t *testing.T) {
		eb := newTestEB()
		defer eb.Close()

		ctx := context.Background()

		var received int32
		sub, err := eb.Subscribe(ctx, "verify.topic", func(ctx context.Context, env *Envelope) error {
			atomic.AddInt32(&received, 1)
			if env.Topic != "verify.topic" {
				t.Errorf("expected topic verify.topic, got %s", env.Topic)
			}
			return nil
		})
		if err != nil {
			t.Fatalf("subscribe: %v", err)
		}
		defer sub.Unsubscribe()

		_ = eb.PublishAsync(ctx, "verify.topic", map[string]string{"k": "v"})
		time.Sleep(300 * time.Millisecond)

		if atomic.LoadInt32(&received) < 1 {
			t.Errorf("expected at least 1 message received, got %d", received)
		}
	})

	t.Run("不同主题互不干扰", func(t *testing.T) {
		eb := newTestEB()
		defer eb.Close()

		ctx := context.Background()

		var aCount, bCount int32
		subA, _ := eb.Subscribe(ctx, "topic.a", func(ctx context.Context, env *Envelope) error {
			atomic.AddInt32(&aCount, 1)
			return nil
		})
		subB, _ := eb.Subscribe(ctx, "topic.b", func(ctx context.Context, env *Envelope) error {
			atomic.AddInt32(&bCount, 1)
			return nil
		})
		defer subA.Unsubscribe()
		defer subB.Unsubscribe()

		_ = eb.PublishAsync(ctx, "topic.a", "hello")
		_ = eb.PublishAsync(ctx, "topic.b", "world")
		time.Sleep(300 * time.Millisecond)

		if atomic.LoadInt32(&aCount) != 1 {
			t.Errorf("topic.a expected 1, got %d", aCount)
		}
		if atomic.LoadInt32(&bCount) != 1 {
			t.Errorf("topic.b expected 1, got %d", bCount)
		}
	})

	t.Run("关闭后发布返回错误", func(t *testing.T) {
		eb := newTestEB()
		eb.Close()

		err := eb.PublishAsync(context.Background(), "test", "data")
		if err != ErrEventBusClosed {
			t.Errorf("expected ErrEventBusClosed, got %v", err)
		}
	})
}

func TestEventBus_PublishSync(t *testing.T) {
	t.Run("同步发布成功", func(t *testing.T) {
		eb := newTestEB()
		defer eb.Close()

		ctx := context.Background()
		var received int32
		sub, _ := eb.Subscribe(ctx, "sync.topic", func(ctx context.Context, env *Envelope) error {
			atomic.AddInt32(&received, 1)
			return nil
		})
		defer sub.Unsubscribe()

		err := eb.PublishSync(ctx, "sync.topic", "payload", 5*time.Second)
		if err != nil {
			t.Fatalf("PublishSync should succeed: %v", err)
		}
		time.Sleep(100 * time.Millisecond)

		if atomic.LoadInt32(&received) != 1 {
			t.Errorf("expected 1, got %d", received)
		}
	})

	t.Run("关闭后返回错误", func(t *testing.T) {
		eb := newTestEB()
		eb.Close()

		err := eb.PublishSync(context.Background(), "test", "data", time.Second)
		if err != ErrEventBusClosed {
			t.Errorf("expected ErrEventBusClosed, got %v", err)
		}
	})
}

func TestEventBus_ProducerCache(t *testing.T) {
	eb := newTestEB()
	defer eb.Close()

	// 通过 PublishAsync 触发 producer 创建
	_ = eb.PublishAsync(context.Background(), "x", "a")
	_ = eb.PublishAsync(context.Background(), "x", "b")
	_ = eb.PublishAsync(context.Background(), "y", "c")
	time.Sleep(200 * time.Millisecond)

	// 获取内部实现验证
	impl := eb.(*eventBus)
	impl.mu.RLock()
	count := len(impl.producers)
	impl.mu.RUnlock()

	if count != 2 {
		t.Errorf("expected 2 producers, got %d", count)
	}
}

func TestEventBus_Close(t *testing.T) {
	t.Run("重复关闭幂等", func(t *testing.T) {
		eb := newTestEB()
		if err := eb.Close(); err != nil {
			t.Fatalf("first close: %v", err)
		}
		if err := eb.Close(); err != nil {
			t.Fatalf("second close should be idempotent: %v", err)
		}
	})

	t.Run("关闭后 producer 缓存清空", func(t *testing.T) {
		eb := newTestEB()
		_ = eb.PublishAsync(context.Background(), "a", "data")
		_ = eb.PublishAsync(context.Background(), "b", "data")
		time.Sleep(200 * time.Millisecond)
		eb.Close()

		impl := eb.(*eventBus)
		impl.mu.RLock()
		count := len(impl.producers)
		impl.mu.RUnlock()
		if count != 0 {
			t.Errorf("producers should be empty after close, got %d", count)
		}
	})
}

func TestEventBus_Envelope(t *testing.T) {
	t.Run("事件被正确包装为Envelope", func(t *testing.T) {
		eb := NewEventBus(mq.NewMemoryConnection(64), EventBusConfig{
			Source: "test-service",
		}, klog.NewStdLogger(io.Discard))
		defer eb.Close()

		type TestEvent struct {
			Message string `json:"message"`
		}

		ctx := context.Background()
		var received Envelope
		sub, _ := eb.Subscribe(ctx, "envelope.test", func(ctx context.Context, env *Envelope) error {
			received = *env
			return nil
		})
		defer sub.Unsubscribe()

		err := eb.PublishSync(ctx, "envelope.test", TestEvent{Message: "hello"}, 5*time.Second)
		if err != nil {
			t.Fatalf("PublishSync: %v", err)
		}
		time.Sleep(100 * time.Millisecond)

		if received.EventID == "" {
			t.Error("EventID should not be empty")
		}
		if received.Source != "test-service" {
			t.Errorf("expected source test-service, got %s", received.Source)
		}
		if received.Topic != "envelope.test" {
			t.Errorf("expected topic envelope.test, got %s", received.Topic)
		}
		if received.EventType != "TestEvent" {
			t.Errorf("expected EventType TestEvent, got %s", received.EventType)
		}

		var decoded TestEvent
		if err := json.Unmarshal(received.Data, &decoded); err != nil {
			t.Fatalf("unmarshal Data: %v", err)
		}
		if decoded.Message != "hello" {
			t.Errorf("expected hello, got %s", decoded.Message)
		}
	})

	t.Run("Envelope.UnmarshalData 正确解码", func(t *testing.T) {
		eb := newTestEB()
		defer eb.Close()

		type MyEvent struct {
			Value int `json:"value"`
		}

		ctx := context.Background()
		var decoded MyEvent
		sub, _ := eb.Subscribe(ctx, "unmarshal.test", func(ctx context.Context, env *Envelope) error {
			return env.UnmarshalData(&decoded)
		})
		defer sub.Unsubscribe()

		_ = eb.PublishSync(ctx, "unmarshal.test", MyEvent{Value: 42}, 5*time.Second)
		time.Sleep(100 * time.Millisecond)

		if decoded.Value != 42 {
			t.Errorf("expected 42, got %d", decoded.Value)
		}
	})
}

func TestEventBus_Concurrent(t *testing.T) {
	t.Run("并发发布不panic", func(t *testing.T) {
		eb := newTestEB()
		defer eb.Close()

		done := make(chan struct{})
		go func() {
			for i := 0; i < 100; i++ {
				go func(n int) {
					_ = eb.PublishAsync(context.Background(), "concurrent.topic", n)
				}(i)
			}
			done <- struct{}{}
		}()

		select {
		case <-done:
			time.Sleep(300 * time.Millisecond)
		case <-time.After(5 * time.Second):
			t.Fatal("concurrent publish timeout")
		}
	})
}

func TestEventBus_Unsubscribe(t *testing.T) {
	t.Run("取消订阅后不再收到事件", func(t *testing.T) {
		eb := newTestEB()
		defer eb.Close()

		ctx := context.Background()
		var received int32
		sub, _ := eb.Subscribe(ctx, "cancel.test", func(ctx context.Context, env *Envelope) error {
			atomic.AddInt32(&received, 1)
			return nil
		})

		_ = eb.PublishSync(ctx, "cancel.test", "msg", 5*time.Second)
		time.Sleep(100 * time.Millisecond)

		sub.Unsubscribe()
		time.Sleep(50 * time.Millisecond)

		_ = eb.PublishSync(ctx, "cancel.test", "msg2", 5*time.Second)
		time.Sleep(100 * time.Millisecond)

		if atomic.LoadInt32(&received) != 1 {
			t.Errorf("expected exactly 1 message after unsubscribe, got %d", received)
		}
	})
}
