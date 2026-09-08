package metrics

import (
	"net/http"
	"sync"

	"github.com/go-kratos/kratos/v2/middleware"
	kmetrics "github.com/go-kratos/kratos/v2/middleware/metrics"
	"go.opentelemetry.io/otel"
)

// 进程级默认 Provider：New 会自动登记，供各服务暴露 /metrics 端点。
var (
	defaultMu       sync.RWMutex
	defaultProvider *Provider
)

// SetDefault 登记默认 Provider（New 内部已调用；如需替换可显式调用）。
func SetDefault(p *Provider) {
	defaultMu.Lock()
	defaultProvider = p
	defaultMu.Unlock()
}

// DefaultHandler 返回默认 Provider 的 Prometheus 抓取处理器；未初始化时返回 nil。
func DefaultHandler() http.Handler {
	defaultMu.RLock()
	p := defaultProvider
	defaultMu.RUnlock()
	if p == nil {
		return nil
	}
	return p.Handler()
}

var (
	serverMWOnce sync.Once
	serverMW     middleware.Middleware
	clientMWOnce sync.Once
	clientMW     middleware.Middleware
)

// ServerMiddleware 返回基于全局 MeterProvider 的 Kratos 服务端指标中间件。
//
// 注意：仪器在首次调用时绑定当时的全局 MeterProvider，请确保 metrics.New
// 早于服务构建（main 启动顺序：日志 → 追踪 → 指标 → Wire）。
func ServerMiddleware() middleware.Middleware {
	serverMWOnce.Do(func() {
		meter := otel.GetMeterProvider().Meter("ley/transport")
		requests, _ := kmetrics.DefaultRequestsCounter(meter, kmetrics.DefaultServerRequestsCounterName)
		seconds, _ := kmetrics.DefaultSecondsHistogram(meter, kmetrics.DefaultServerSecondsHistogramName)
		serverMW = kmetrics.Server(kmetrics.WithRequests(requests), kmetrics.WithSeconds(seconds))
	})
	return serverMW
}

// ClientMiddleware 返回基于全局 MeterProvider 的 Kratos 客户端指标中间件
// （entry 调用 auth/blog 的服务间调用指标）。
func ClientMiddleware() middleware.Middleware {
	clientMWOnce.Do(func() {
		meter := otel.GetMeterProvider().Meter("ley/transport")
		requests, _ := kmetrics.DefaultRequestsCounter(meter, kmetrics.DefaultClientRequestsCounterName)
		seconds, _ := kmetrics.DefaultSecondsHistogram(meter, kmetrics.DefaultClientSecondsHistogramName)
		clientMW = kmetrics.Client(kmetrics.WithRequests(requests), kmetrics.WithSeconds(seconds))
	})
	return clientMW
}
