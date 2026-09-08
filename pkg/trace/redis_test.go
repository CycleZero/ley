package trace

import (
	"context"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	tracesdk "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

// 场景 S4（data 层 trace）：go-redis 命令调用必须产生 span，
// 即使连接失败（错误路径）也要记录，并带 db.system=redis 属性。
func TestRedisHookEmitsCommandSpan(t *testing.T) {
	exp := tracetest.NewInMemoryExporter()
	tp := tracesdk.NewTracerProvider(tracesdk.WithSampler(tracesdk.AlwaysSample()), tracesdk.WithSyncer(exp))

	// 指向必然拒绝连接的地址，触发错误路径；禁用重试保证快速失败。
	client := redis.NewClient(&redis.Options{
		Addr:        "127.0.0.1:1",
		DialTimeout: 20 * time.Millisecond,
		MaxRetries:  -1,
	})
	client.AddHook(NewRedisHook(WithRedisTracerProvider(tp)))

	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()
	_ = client.Get(ctx, "trace:key").Err()

	spans := exp.GetSpans()
	if len(spans) == 0 {
		t.Fatal("Redis 命令调用未产生任何 span")
	}
	got := findSpan(spans, "GET")
	if got == nil {
		t.Fatalf("未产生 GET 命令 span，实际 span：%v", spanNames(spans))
	}
	if attrs := attrMap(got.Attributes); attrs["db.system"] != "redis" {
		t.Errorf("db.system 应为 redis，实际 %q", attrs["db.system"])
	}
}
