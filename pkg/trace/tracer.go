package trace

import (
	"context"
	"os"
	"strconv"
	"sync"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	tracesdk "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

// tracerName 默认 tracer 名称（用于 transport / 业务插桩）。
const tracerName = "ley/otel"

// Config 链路追踪初始化配置。Endpoint 为空时回退到 OTel 标准环境变量
// （OTEL_EXPORTER_OTLP_TRACES_ENDPOINT / OTEL_EXPORTER_OTLP_ENDPOINT），
// 保证与任意 OTel 兼容后端（Jaeger/Tempo/OTLP Collector）对接。
type Config struct {
	Endpoint       string  // OTLP HTTP 端点（host:port，不含 scheme）
	ServiceName    string  // 服务名（写入 resource.service.name）
	ServiceVersion string  // 服务版本（写入 resource.service.version）
	Environment    string  // 部署环境（写入 resource.deployment.environment）
	SampleRatio    float64 // 采样率 0~1；<=0 时读取 OTEL_TRACES_SAMPLER_ARG，仍无效则 1.0
	Insecure       bool    // 是否使用明文 HTTP 上报（默认 true，与内网 Collector 一致）
}

// Provider 持有 TracerProvider，负责优雅关闭以刷出未上报的 span。
type Provider struct {
	tp *tracesdk.TracerProvider

	shutdownOnce sync.Once
	shutdownErr  error
}

// Shutdown 关闭 TracerProvider 并刷出缓冲中的 span；可安全重复调用。
func (p *Provider) Shutdown(ctx context.Context) error {
	if p == nil || p.tp == nil {
		return nil
	}
	p.shutdownOnce.Do(func() { p.shutdownErr = p.tp.Shutdown(ctx) })
	return p.shutdownErr
}

// TracerProvider 返回底层 TracerProvider（供需要显式注入的插桩使用）。
func (p *Provider) TracerProvider() trace.TracerProvider {
	if p == nil || p.tp == nil {
		return otel.GetTracerProvider()
	}
	return p.tp
}

// globalProvider 记录最近一次 Init 的 Provider，供 Shutdown 兜底刷出。
var globalProvider *Provider

// Init 初始化全局链路追踪：OTLP/HTTP 上报 + W3C 上下文传播 + 资源属性。
// 幂等性由调用方保证（进程启动时调用一次）。
func Init(cfg Config) (*Provider, error) {
	exporterOpts := make([]otlptracehttp.Option, 0, 2)
	if cfg.Endpoint != "" {
		exporterOpts = append(exporterOpts, otlptracehttp.WithEndpoint(cfg.Endpoint))
		if cfg.Insecure {
			exporterOpts = append(exporterOpts, otlptracehttp.WithInsecure())
		}
	}
	exporter, err := otlptracehttp.New(context.Background(), exporterOpts...)
	if err != nil {
		return nil, err
	}

	res, err := resource.New(context.Background(),
		resource.WithFromEnv(),
		resource.WithTelemetrySDK(),
		resource.WithAttributes(resourceAttributes(cfg)...),
	)
	if err != nil {
		res = resource.NewSchemaless(resourceAttributes(cfg)...)
	}

	tp := tracesdk.NewTracerProvider(
		tracesdk.WithSampler(tracesdk.ParentBased(tracesdk.TraceIDRatioBased(sampleRatio(cfg.SampleRatio)))),
		tracesdk.WithBatcher(exporter),
		tracesdk.WithResource(res),
	)
	otel.SetTracerProvider(tp)
	// W3C TraceContext + Baggage：跨 HTTP/gRPC 服务自动透传 traceparent。
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	p := &Provider{tp: tp}
	globalProvider = p
	return p, nil
}

// resourceAttributes 组装资源属性（仅在字段非空时写入，避免空值污染）。
func resourceAttributes(cfg Config) []attribute.KeyValue {
	attrs := make([]attribute.KeyValue, 0, 3)
	if cfg.ServiceName != "" {
		attrs = append(attrs, attribute.String("service.name", cfg.ServiceName))
	}
	if cfg.ServiceVersion != "" {
		attrs = append(attrs, attribute.String("service.version", cfg.ServiceVersion))
	}
	if cfg.Environment != "" {
		attrs = append(attrs, attribute.String("deployment.environment", cfg.Environment))
	}
	return attrs
}

// sampleRatio 解析采样率：显式配置优先，其次 OTEL_TRACES_SAMPLER_ARG，最后 1.0。
func sampleRatio(configured float64) float64 {
	if configured > 0 && configured <= 1 {
		return configured
	}
	if raw := os.Getenv("OTEL_TRACES_SAMPLER_ARG"); raw != "" {
		if v, err := strconv.ParseFloat(raw, 64); err == nil && v > 0 && v <= 1 {
			return v
		}
	}
	return 1.0
}

// InitTracer 兼容旧签名：以明文 OTLP 初始化指定服务名的全局追踪。
// 新代码请使用 Init（可配置采样率、环境、版本，并支持优雅关闭）。
func InitTracer(endpoint, serviceName string) error {
	_, err := Init(Config{Endpoint: endpoint, ServiceName: serviceName, Insecure: true})
	return err
}

// Shutdown 关闭全局 Provider 并刷出缓冲 span；未初始化时为空操作。
func Shutdown(ctx context.Context) error {
	if globalProvider == nil {
		return nil
	}
	return globalProvider.Shutdown(ctx)
}
