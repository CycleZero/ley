package trace

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
)

// RedisOption 配置 Redis 插桩。
type RedisOption func(*redisOptions)

type redisOptions struct {
	tp trace.TracerProvider
}

// WithRedisTracerProvider 显式指定 TracerProvider（不指定则用全局）。
func WithRedisTracerProvider(tp trace.TracerProvider) RedisOption {
	return func(o *redisOptions) { o.tp = tp }
}

// RedisHook 是 go-redis 的 OTel Hook：
// 为每条命令/管道产生一个 client span（db.system=redis），并记录耗时指标。
// 通过 client.AddHook(NewRedisHook()) 挂载后，pkg/cache 与 pkg/infra 的 Redis
// 调用自动获得链路与指标，无需逐个方法改造。
type RedisHook struct {
	opts redisOptions
}

// NewRedisHook 创建 Redis 插桩 Hook。
func NewRedisHook(opts ...RedisOption) *RedisHook {
	o := redisOptions{}
	for _, opt := range opts {
		opt(&o)
	}
	return &RedisHook{opts: o}
}

// DialHook 连接建立不产生 span（高频、无业务语义）。
func (h *RedisHook) DialHook(next redis.DialHook) redis.DialHook {
	return next
}

// ProcessHook 为单条命令创建 span。
func (h *RedisHook) ProcessHook(next redis.ProcessHook) redis.ProcessHook {
	return func(ctx context.Context, cmd redis.Cmder) error {
		ctx, done := h.begin(ctx, cmd)
		err := next(ctx, cmd)
		done(err)
		return err
	}
}

// ProcessPipelineHook 为管道批量命令创建 span（记录命令数量）。
func (h *RedisHook) ProcessPipelineHook(next redis.ProcessPipelineHook) redis.ProcessPipelineHook {
	return func(ctx context.Context, cmds []redis.Cmder) error {
		ctx, done := h.begin(ctx, cmds...)
		err := next(ctx, cmds)
		done(err)
		return err
	}
}

// begin 启动 span 并返回结束回调。
func (h *RedisHook) begin(ctx context.Context, cmds ...redis.Cmder) (context.Context, func(error)) {
	if ctx == nil {
		ctx = context.Background()
	}
	name := "redis"
	if len(cmds) > 0 {
		name += " " + strings.ToUpper(cmds[0].FullName())
	}
	attrs := []attribute.KeyValue{
		attribute.String("db.system", "redis"),
		attribute.Int("db.operation.batch_size", len(cmds)),
	}
	if len(cmds) > 0 {
		attrs = append(attrs, attribute.String("db.operation.name", strings.ToUpper(cmds[0].Name())))
	}
	ctx, span := h.tracer().Start(ctx, name,
		trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(attrs...),
	)
	start := time.Now()
	return ctx, func(err error) {
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
		}
		span.End()
		recordRedisOperation(ctx, len(cmds), err, time.Since(start))
	}
}

// tracer 返回 Hook 使用的 tracer（显式注入优先）。
func (h *RedisHook) tracer() trace.Tracer {
	if h.opts.tp != nil {
		return h.opts.tp.Tracer(tracerName)
	}
	return otel.GetTracerProvider().Tracer(tracerName)
}

// redisMetricsOnce 保证指标仪器仅创建一次。
var (
	redisMetricsOnce sync.Once
	redisDuration    metric.Float64Histogram
)

// recordRedisOperation 记录 Redis 操作耗时直方图（metrics 未初始化时为空操作）。
func recordRedisOperation(ctx context.Context, batchSize int, err error, elapsed time.Duration) {
	redisMetricsOnce.Do(func() {
		redisDuration, _ = otel.GetMeterProvider().Meter("ley/redis").Float64Histogram(
			"redis.client.operation.duration",
			metric.WithUnit("s"),
			metric.WithDescription("Redis 操作耗时"),
		)
	})
	if redisDuration == nil {
		return
	}
	redisDuration.Record(ctx, elapsed.Seconds(),
		metric.WithAttributes(
			attribute.Int("db.operation.batch_size", batchSize),
			attribute.String("result", resultLabel(err)),
		),
	)
}
