package log

import (
	"context"
	"fmt"
	"math"
	"strings"
	"sync"
	"time"

	"github.com/CycleZero/ley/pkg/otelx"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploghttp"
	otellog "go.opentelemetry.io/otel/log"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap/zapcore"
)

// otlpLogCore 把 zap 日志桥接到 OTLP 日志导出器。
// 除常规字段外，还把 trace_id/span_id 映射为日志记录的链路上下文，
// 使日志与链路在 SigNoz 等后端中可互相跳转。
type otlpLogCore struct {
	logger   otellog.Logger
	minLevel zapcore.Level
	fields   []zapcore.Field
}

var (
	otlpLogMu       sync.Mutex
	otlpLogProvider *sdklog.LoggerProvider
)

// newOTLPLogCore 初始化 OTLP 日志导出并返回可挂到 zap Tee 的 Core。
// endpoint 为空返回 (nil, nil)，由调用方决定是否降级为仅本地日志。
func newOTLPLogCore(serviceName, endpoint string, minLevel zapcore.Level) (zapcore.Core, error) {
	if strings.TrimSpace(endpoint) == "" {
		return nil, nil
	}
	host, path, insecure := otelx.ParseEndpoint(endpoint, true, "/v1/logs")
	opts := []otlploghttp.Option{otlploghttp.WithEndpoint(host)}
	if path != "" {
		opts = append(opts, otlploghttp.WithURLPath(path))
	}
	if insecure {
		opts = append(opts, otlploghttp.WithInsecure())
	}
	exp, err := otlploghttp.New(context.Background(), opts...)
	if err != nil {
		return nil, fmt.Errorf("创建 OTLP 日志导出器失败：%w", err)
	}
	res, err := resource.New(context.Background(),
		resource.WithFromEnv(),
		resource.WithTelemetrySDK(),
		resource.WithAttributes(attribute.String("service.name", serviceName)),
	)
	if err != nil {
		res = resource.NewSchemaless(attribute.String("service.name", serviceName))
	}
	lp := sdklog.NewLoggerProvider(
		sdklog.WithResource(res),
		sdklog.WithProcessor(sdklog.NewBatchProcessor(exp)),
	)
	otlpLogMu.Lock()
	otlpLogProvider = lp
	otlpLogMu.Unlock()
	return &otlpLogCore{logger: lp.Logger(serviceName), minLevel: minLevel}, nil
}

// ShutdownOTLP 刷出并关闭 OTLP 日志导出器；未启用时为空操作。
func ShutdownOTLP(ctx context.Context) error {
	otlpLogMu.Lock()
	lp := otlpLogProvider
	otlpLogMu.Unlock()
	if lp == nil {
		return nil
	}
	return lp.Shutdown(ctx)
}

// Enabled 仅放行不低于阈值的级别。
func (c *otlpLogCore) Enabled(lvl zapcore.Level) bool { return lvl >= c.minLevel }

// With 派生子 Core 并合并字段，保证带字段的日志同样上报。
func (c *otlpLogCore) With(fields []zapcore.Field) zapcore.Core {
	merged := make([]zapcore.Field, 0, len(c.fields)+len(fields))
	merged = append(merged, c.fields...)
	merged = append(merged, fields...)
	return &otlpLogCore{logger: c.logger, minLevel: c.minLevel, fields: merged}
}

// Check 按级别决定是否把当前 Core 加入写入链。
func (c *otlpLogCore) Check(ent zapcore.Entry, ce *zapcore.CheckedEntry) *zapcore.CheckedEntry {
	if c.Enabled(ent.Level) {
		return ce.AddCore(ent, c)
	}
	return ce
}

// Write 把一条 zap 日志转换为 OTLP 日志记录并提交导出。
func (c *otlpLogCore) Write(ent zapcore.Entry, fields []zapcore.Field) error {
	all := make([]zapcore.Field, 0, len(c.fields)+len(fields))
	all = append(all, c.fields...)
	all = append(all, fields...)

	var rec otellog.Record
	rec.SetTimestamp(ent.Time)
	rec.SetObservedTimestamp(time.Now())
	rec.SetSeverity(toOtelSeverity(ent.Level))
	rec.SetSeverityText(ent.Level.CapitalString())
	rec.SetBody(otellog.StringValue(ent.Message))
	rec.AddAttributes(attributesFromFields(all)...)
	c.logger.Emit(contextWithTrace(all), rec)
	return nil
}

// Sync 无需额外动作：刷出由 SDK 批量处理器负责。
func (c *otlpLogCore) Sync() error { return nil }

// contextWithTrace 从日志字段还原链路上下文；SDK 据此填充日志记录的 trace_id/span_id。
func contextWithTrace(fields []zapcore.Field) context.Context {
	traceID, spanID, ok := traceContextFromFields(fields)
	if !ok {
		return context.Background()
	}
	sc := trace.NewSpanContext(trace.SpanContextConfig{
		TraceID:    traceID,
		SpanID:     spanID,
		TraceFlags: trace.FlagsSampled,
		Remote:     true,
	})
	return trace.ContextWithSpanContext(context.Background(), sc)
}

// toOtelSeverity 映射 zap 级别到 OTel 严重度。
func toOtelSeverity(lvl zapcore.Level) otellog.Severity {
	switch lvl {
	case zapcore.DebugLevel:
		return otellog.SeverityDebug
	case zapcore.InfoLevel:
		return otellog.SeverityInfo
	case zapcore.WarnLevel:
		return otellog.SeverityWarn
	case zapcore.ErrorLevel:
		return otellog.SeverityError
	default:
		return otellog.SeverityFatal
	}
}

// attributesFromFields 把 zap 字段转换为 OTel 属性；链路字段由日志记录的链路上下文承载，不重复写入。
func attributesFromFields(fields []zapcore.Field) []otellog.KeyValue {
	attrs := make([]otellog.KeyValue, 0, len(fields))
	for _, f := range fields {
		if f.Key == "" || isTraceField(f.Key) {
			continue
		}
		attrs = append(attrs, attributeFromField(f))
	}
	return attrs
}

// attributeFromField 按 zap 字段类型转换 OTel 属性，未知类型退化为字符串。
func attributeFromField(f zapcore.Field) otellog.KeyValue {
	switch f.Type {
	case zapcore.StringType:
		return otellog.String(f.Key, f.String)
	case zapcore.BoolType:
		return otellog.Bool(f.Key, f.Integer == 1)
	case zapcore.Int64Type, zapcore.Int32Type, zapcore.Int16Type, zapcore.Int8Type,
		zapcore.Uint64Type, zapcore.Uint32Type, zapcore.Uint16Type, zapcore.Uint8Type, zapcore.UintptrType:
		return otellog.Int64(f.Key, f.Integer)
	case zapcore.Float64Type:
		return otellog.Float64(f.Key, math.Float64frombits(uint64(f.Integer)))
	case zapcore.Float32Type:
		return otellog.Float64(f.Key, float64(math.Float32frombits(uint32(f.Integer))))
	case zapcore.DurationType:
		return otellog.String(f.Key, time.Duration(f.Integer).String())
	case zapcore.TimeType:
		return otellog.String(f.Key, time.Unix(0, f.Integer).Format(time.RFC3339Nano))
	case zapcore.StringerType:
		if s, ok := f.Interface.(fmt.Stringer); ok {
			return otellog.String(f.Key, s.String())
		}
	case zapcore.ErrorType:
		if err, ok := f.Interface.(error); ok {
			return otellog.String(f.Key, err.Error())
		}
	}
	return otellog.String(f.Key, fmt.Sprint(f.Interface))
}

// isTraceField 判断字段是否为链路关联字段（兼容 log.Ctx 的 trace_id 与 Kratos 的 trace.id 两种写法）。
func isTraceField(key string) bool {
	switch key {
	case FieldTraceID, FieldSpanID, "trace.id", "span.id":
		return true
	default:
		return false
	}
}

// traceContextFromFields 从日志字段解析链路上下文；trace_id 无效时 ok=false。
func traceContextFromFields(fields []zapcore.Field) (trace.TraceID, trace.SpanID, bool) {
	var traceID trace.TraceID
	var spanID trace.SpanID
	for _, f := range fields {
		if f.Type != zapcore.StringType || f.String == "" {
			continue
		}
		switch f.Key {
		case FieldTraceID, "trace.id":
			if id, err := trace.TraceIDFromHex(f.String); err == nil {
				traceID = id
			}
		case FieldSpanID, "span.id":
			if id, err := trace.SpanIDFromHex(f.String); err == nil {
				spanID = id
			}
		}
	}
	return traceID, spanID, traceID.IsValid()
}
