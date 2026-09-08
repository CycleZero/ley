package proxy

import (
	"context"
	"time"

	"github.com/CycleZero/ley/pkg/log"
	"github.com/CycleZero/ley/pkg/metrics"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.uber.org/zap"
)

// 代理转发业务指标：在 NewProxyHub 构建期创建（此时 metrics.New 已完成）。
var (
	proxyRequests metric.Int64Counter
	proxyDuration metric.Float64Histogram
)

// initProxyMetrics 创建代理域指标仪器。
func initProxyMetrics() {
	proxyRequests = metrics.Counter("entry_proxy_requests_total", "entry 代理转发请求数")
	proxyDuration = metrics.Histogram("entry_proxy_duration_seconds", "entry 代理转发耗时", "s")
}

// recordProxyCall 记录一次代理转发的日志与指标。
//
// 分级纪律：5xx 记 Error（真故障），其余失败记 Warn（业务拒绝/客户端问题），
// 成功记 Debug（避免正常流量刷屏）；指标标签 operation/result 基数有界。
func recordProxyCall(ctx context.Context, operation string, httpStatus int, elapsed time.Duration, err error) {
	result := "success"
	if err != nil {
		result = "error"
	}
	if proxyRequests != nil {
		proxyRequests.Add(ctx, 1, metric.WithAttributes(
			attribute.String("operation", operation),
			attribute.String("result", result),
		))
	}
	if proxyDuration != nil {
		proxyDuration.Record(ctx, elapsed.Seconds(), metric.WithAttributes(
			attribute.String("operation", operation),
		))
	}

	logger := log.Ctx(ctx)
	switch {
	case err == nil:
		logger.Debug("代理转发完成", zap.String("operation", operation), zap.Int64("cost_ms", elapsed.Milliseconds()))
	case httpStatus >= 500:
		logger.Error("代理转发失败", zap.String("operation", operation), zap.Int64("cost_ms", elapsed.Milliseconds()))
	default:
		logger.Warn("代理转发被拒绝", zap.String("operation", operation), zap.Int("http_status", httpStatus), zap.Int64("cost_ms", elapsed.Milliseconds()))
	}
}
