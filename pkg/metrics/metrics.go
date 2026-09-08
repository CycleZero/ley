package metrics

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/CycleZero/ley/pkg/otelx"
	"github.com/go-kratos/kratos/v2/middleware"
	kmetrics "github.com/go-kratos/kratos/v2/middleware/metrics"
	prom "github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	otelprom "go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/metric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.40.0"
)

// Provider 统一的可观测指标提供者：
//   - 基于 OTel Metrics SDK，指标出口为 Prometheus（OTel 兼容，可平滑切换 OTLP）；
//   - 同时提供业务自定义指标与 Kratos transport 指标中间件。
//
// 生命周期：进程启动时 New 一次，HTTP 层用 Handler 暴露 /metrics，
// 退出时调用 Shutdown 刷出未聚合数据。
type Provider struct {
	serviceName string
	registry    *prom.Registry
	provider    *sdkmetric.MeterProvider
	meter       metric.Meter

	shutdownOnce sync.Once
	shutdownErr  error
}

// options 指标 Provider 的可选配置。
type options struct {
	otlpEndpoint string
}

// Option 指标 Provider 配置项。
type Option func(*options)

// WithOTLPEndpoint 启用 OTLP/HTTP 指标上报（与 /metrics 抓取并存）。
// endpoint 支持 host:port 或 http(s)://host:port；为空时仅暴露 Prometheus。
func WithOTLPEndpoint(endpoint string) Option {
	return func(o *options) { o.otlpEndpoint = endpoint }
}

// New 创建指标 Provider：注册 Prometheus exporter、构建 MeterProvider 并设为全局，
// 使 OTel 原生插桩（Kratos metrics 中间件、otelgrpc 等）自动生效。
func New(serviceName string, opts ...Option) (*Provider, error) {
	o := options{}
	for _, opt := range opts {
		opt(&o)
	}

	registry := prom.NewRegistry()
	exporter, err := otelprom.New(otelprom.WithRegisterer(registry))
	if err != nil {
		return nil, fmt.Errorf("创建 Prometheus exporter 失败：%w", err)
	}

	res, err := resource.New(context.Background(),
		resource.WithFromEnv(),
		resource.WithTelemetrySDK(),
		resource.WithAttributes(semconv.ServiceNameKey.String(serviceName)),
	)
	if err != nil {
		// 资源解析失败（如 OTEL_RESOURCE_ATTRIBUTES 非法）时降级为最小资源，不阻断启动。
		res = resource.NewSchemaless(semconv.ServiceNameKey.String(serviceName))
	}

	// Prometheus 抓取与 OTLP 推送并存：本地 /metrics 便于排障，
	// 远程 OTLP 让生产环境无需额外采集器即可把指标送进可观测后端。
	readerOpts := []sdkmetric.Option{sdkmetric.WithReader(exporter)}
	if strings.TrimSpace(o.otlpEndpoint) != "" {
		otlpReader, err := newOTLPReader(o.otlpEndpoint)
		if err != nil {
			return nil, err
		}
		readerOpts = append(readerOpts, sdkmetric.WithReader(otlpReader))
	}

	mp := sdkmetric.NewMeterProvider(append([]sdkmetric.Option{sdkmetric.WithResource(res)}, readerOpts...)...)
	otel.SetMeterProvider(mp)

	p := &Provider{
		serviceName: serviceName,
		registry:    registry,
		provider:    mp,
		meter:       mp.Meter(serviceName),
	}
	SetDefault(p)
	return p, nil
}

// Handler 返回 Prometheus 抓取处理器（挂载到 /metrics）。
func (p *Provider) Handler() http.Handler {
	return promhttp.HandlerFor(p.registry, promhttp.HandlerOpts{})
}

// Meter 返回底层 Meter，供需要直接创建自定义指标的调用方使用。
func (p *Provider) Meter() metric.Meter {
	return p.meter
}

// Shutdown 刷出并关闭 MeterProvider；可安全重复调用。
func (p *Provider) Shutdown(ctx context.Context) error {
	p.shutdownOnce.Do(func() {
		p.shutdownErr = p.provider.Shutdown(ctx)
	})
	return p.shutdownErr
}

// Int64Counter 创建单调递增计数器（如 xxx_total）。
func (p *Provider) Int64Counter(name, description string) (metric.Int64Counter, error) {
	c, err := p.meter.Int64Counter(name, metric.WithDescription(description))
	if err != nil {
		return nil, fmt.Errorf("创建计数器 %s 失败：%w", name, err)
	}
	return c, nil
}

// Float64Histogram 创建耗时/大小直方图（unit 例：s、By）。
func (p *Provider) Float64Histogram(name, description, unit string) (metric.Float64Histogram, error) {
	h, err := p.meter.Float64Histogram(name,
		metric.WithDescription(description),
		metric.WithUnit(unit),
	)
	if err != nil {
		return nil, fmt.Errorf("创建直方图 %s 失败：%w", name, err)
	}
	return h, nil
}

// Int64UpDownCounter 创建可增可减计数器（如当前并发请求数）。
func (p *Provider) Int64UpDownCounter(name, description string) (metric.Int64UpDownCounter, error) {
	c, err := p.meter.Int64UpDownCounter(name, metric.WithDescription(description))
	if err != nil {
		return nil, fmt.Errorf("创建可增减计数器 %s 失败：%w", name, err)
	}
	return c, nil
}

// Counter 基于全局 MeterProvider 创建业务计数器。
// 必须在 metrics.New 之后调用（业务用例构造函数在 Wire 阶段执行，满足该时序）。
func Counter(name, description string) metric.Int64Counter {
	c, _ := otel.GetMeterProvider().Meter("ley/business").Int64Counter(name, metric.WithDescription(description))
	return c
}

// Histogram 基于全局 MeterProvider 创建业务直方图（如耗时/大小）。
func Histogram(name, description, unit string) metric.Float64Histogram {
	h, _ := otel.GetMeterProvider().Meter("ley/business").Float64Histogram(
		name, metric.WithDescription(description), metric.WithUnit(unit))
	return h
}

// KratosServerMiddleware 构建 Kratos 服务端 transport 指标中间件
// （请求数 server_requests_code_total + 耗时 server_requests_seconds_bucket）。
func (p *Provider) KratosServerMiddleware() (middleware.Middleware, error) {
	requests, err := kmetrics.DefaultRequestsCounter(p.meter, kmetrics.DefaultServerRequestsCounterName)
	if err != nil {
		return nil, fmt.Errorf("创建服务端请求计数器失败：%w", err)
	}
	seconds, err := kmetrics.DefaultSecondsHistogram(p.meter, kmetrics.DefaultServerSecondsHistogramName)
	if err != nil {
		return nil, fmt.Errorf("创建服务端耗时直方图失败：%w", err)
	}
	return kmetrics.Server(kmetrics.WithRequests(requests), kmetrics.WithSeconds(seconds)), nil
}

// KratosClientMiddleware 构建 Kratos 客户端 transport 指标中间件
// （服务间调用请求数 + 耗时）。
func (p *Provider) KratosClientMiddleware() (middleware.Middleware, error) {
	requests, err := kmetrics.DefaultRequestsCounter(p.meter, kmetrics.DefaultClientRequestsCounterName)
	if err != nil {
		return nil, fmt.Errorf("创建客户端请求计数器失败：%w", err)
	}
	seconds, err := kmetrics.DefaultSecondsHistogram(p.meter, kmetrics.DefaultClientSecondsHistogramName)
	if err != nil {
		return nil, fmt.Errorf("创建客户端耗时直方图失败：%w", err)
	}
	return kmetrics.Client(kmetrics.WithRequests(requests), kmetrics.WithSeconds(seconds)), nil
}

// newOTLPReader 创建 OTLP/HTTP 指标导出器（15s 周期上报，进程退出时由 Provider.Shutdown 刷出）。
func newOTLPReader(raw string) (sdkmetric.Reader, error) {
	endpoint, path, insecure := otelx.ParseEndpoint(raw, true, "/v1/metrics")
	opts := []otlpmetrichttp.Option{otlpmetrichttp.WithEndpoint(endpoint)}
	if path != "" {
		opts = append(opts, otlpmetrichttp.WithURLPath(path))
	}
	if insecure {
		opts = append(opts, otlpmetrichttp.WithInsecure())
	}
	exp, err := otlpmetrichttp.New(context.Background(), opts...)
	if err != nil {
		return nil, fmt.Errorf("创建 OTLP 指标导出器失败：%w", err)
	}
	return sdkmetric.NewPeriodicReader(exp, sdkmetric.WithInterval(15*time.Second)), nil
}
