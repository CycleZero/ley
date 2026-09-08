// observability.go —— entry 请求级链路与 HTTP 指标中间件。
//
// 职责：
//  1. 从入站请求头提取 W3C traceparent（跨服务链路续接），为每个请求创建 server span；
//  2. 把带 span 的 context 写回 *http.Request，供后续中间件/handler 用 pkg/log.Ctx 取 trace_id；
//  3. 记录 HTTP 指标：请求数、耗时、当前并发数（标签基数有界：method/route/status）。
//
// 位置：全局链最前（Recovery 之后、RequestLogger 之前），保证访问日志与业务日志都带 trace_id。
package middleware

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

// HTTP 指标仪器：在中间件构建期（进程启动、metrics.New 之后）创建一次，
// 避免用 sync.Once 延迟绑定到尚未初始化的空 MeterProvider。
var (
	httpRequests metric.Int64Counter
	httpDuration metric.Float64Histogram
	httpActive   metric.Int64UpDownCounter
)

// Observability 返回 entry 的链路 + HTTP 指标中间件。
//
// 参数 serviceName 作为 tracer/meter 名称，便于按服务区分指标来源。
func Observability(serviceName string) gin.HandlerFunc {
	tracer := otel.GetTracerProvider().Tracer(serviceName)
	meter := otel.GetMeterProvider().Meter(serviceName)
	httpRequests, _ = meter.Int64Counter("http.server.requests", metric.WithUnit("{request}"))
	httpDuration, _ = meter.Float64Histogram("http.server.request.duration", metric.WithUnit("s"))
	httpActive, _ = meter.Int64UpDownCounter("http.server.active_requests", metric.WithUnit("{request}"))

	return func(c *gin.Context) {
		// 1. 续接上游链路：从 traceparent 提取父 span 上下文
		ctx := otel.GetTextMapPropagator().Extract(c.Request.Context(), propagation.HeaderCarrier(c.Request.Header))
		ctx, span := tracer.Start(ctx, c.Request.Method+" "+c.Request.URL.Path,
			trace.WithSpanKind(trace.SpanKindServer),
			trace.WithAttributes(
				attribute.String("http.request.method", c.Request.Method),
				attribute.String("url.path", c.Request.URL.Path),
			),
		)
		// 2. 写回请求上下文，供后续中间件/handler 的 log.Ctx 读取 trace_id
		c.Request = c.Request.WithContext(ctx)

		if httpActive != nil {
			httpActive.Add(ctx, 1)
			defer httpActive.Add(ctx, -1)
		}
		start := time.Now()

		c.Next()

		// 3. 请求结束：路由模板（c.FullPath）作为有界标签，避免原始路径造成高基数
		status := c.Writer.Status()
		route := c.FullPath()
		if route == "" {
			route = "unmatched"
		}
		attrs := metric.WithAttributes(
			attribute.String("http.request.method", c.Request.Method),
			attribute.String("http.route", route),
			attribute.Int("http.response.status_code", status),
		)
		if httpRequests != nil {
			httpRequests.Add(ctx, 1, attrs)
		}
		if httpDuration != nil {
			httpDuration.Record(ctx, time.Since(start).Seconds(), attrs)
		}

		span.SetAttributes(
			attribute.String("http.route", route),
			attribute.Int("http.response.status_code", status),
		)
		if status >= 500 {
			span.SetStatus(codes.Error, "HTTP "+strconv.Itoa(status))
		}
		span.End()
	}
}
