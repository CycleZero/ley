package log

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/CycleZero/ley/pkg/meta"
	tracesdk "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// newCaptureLogger 构造写入内存缓冲的日志器，便于断言结构化字段与级别过滤。
// 参数 level 为最低输出级别；返回日志器与底层缓冲。
func newCaptureLogger(level zapcore.Level) (*Logger, *bytes.Buffer) {
	buf := &bytes.Buffer{}
	enc := zapcore.NewJSONEncoder(zapcore.EncoderConfig{
		MessageKey:  "M",
		LevelKey:    "L",
		TimeKey:     "T",
		EncodeLevel: zapcore.LowercaseLevelEncoder,
	})
	core := zapcore.NewCore(enc, zapcore.AddSync(buf), level)
	return &Logger{zap.New(core)}, buf
}

// useCaptureLogger 将捕获日志器临时设为全局并返回还原函数，保证用例间互不污染。
func useCaptureLogger(l *Logger) func() {
	prev := globalLogger
	SetGlobalLogger(l)
	return func() { SetGlobalLogger(prev) }
}

// 场景 S1（happy）：请求上下文携带 trace span 与用户元数据时，
// Ctx 派生的日志器必须自动注入 trace_id/span_id/user_id/role，实现日志与链路关联。
func TestCtxInjectsTraceAndUserFields(t *testing.T) {
	l, buf := newCaptureLogger(zapcore.DebugLevel)
	defer useCaptureLogger(l)()

	tp := tracesdk.NewTracerProvider(tracesdk.WithSampler(tracesdk.AlwaysSample()))
	ctx, span := tp.Tracer("test").Start(context.Background(), "op")
	defer span.End()
	sc := trace.SpanContextFromContext(ctx)
	ctx = meta.NewClientCtx(ctx, &meta.RequestMetaData{
		Auth: meta.Auth{UserID: 42, UserName: "alice", Role: "admin"},
	})

	Ctx(ctx).Info("查询文章")

	out := buf.String()
	for _, want := range []string{sc.TraceID().String(), sc.SpanID().String(), `"user_id":42`, `"role":"admin"`} {
		if !strings.Contains(out, want) {
			t.Fatalf("日志缺少字段 %q，实际输出：%s", want, out)
		}
	}
}

// 场景 S2（edge）：上下文无 span、无用户元数据时，Ctx 不得注入空的关联字段，
// 避免每条日志出现 trace_id="" 的噪声。
func TestCtxOmitsEmptyCorrelationFields(t *testing.T) {
	l, buf := newCaptureLogger(zapcore.DebugLevel)
	defer useCaptureLogger(l)()

	Ctx(context.Background()).Info("无链路上下文")

	out := buf.String()
	for _, unwanted := range []string{"trace_id", "span_id", "user_id"} {
		if strings.Contains(out, unwanted) {
			t.Fatalf("无上下文时不应出现字段 %q，实际输出：%s", unwanted, out)
		}
	}
}

// 场景 S1（Kratos 适配）：Kratos log.Helper 经 WithContext 携带 span 时，
// 中文日志必须带上 trace_id/span_id，且消息体保持中文原文。
func TestKratosLoggerInjectsTraceID(t *testing.T) {
	l, buf := newCaptureLogger(zapcore.DebugLevel)
	defer useCaptureLogger(l)()

	tp := tracesdk.NewTracerProvider(tracesdk.WithSampler(tracesdk.AlwaysSample()))
	ctx, span := tp.Tracer("test").Start(context.Background(), "op")
	defer span.End()
	sc := trace.SpanContextFromContext(ctx)

	GetKratosLogHelper().WithContext(ctx).Info("文章创建成功")

	out := buf.String()
	if !strings.Contains(out, sc.TraceID().String()) {
		t.Fatalf("Kratos 日志缺少 trace_id，实际输出：%s", out)
	}
	if !strings.Contains(out, "文章创建成功") {
		t.Fatalf("Kratos 日志缺少中文消息，实际输出：%s", out)
	}
}

// 场景 S2（edge）：Kratos 日志在无 span 上下文时不得出现空 trace_id 字段。
func TestKratosLoggerOmitsEmptyTraceID(t *testing.T) {
	l, buf := newCaptureLogger(zapcore.DebugLevel)
	defer useCaptureLogger(l)()

	GetKratosLogHelper().WithContext(context.Background()).Info("无链路日志")

	if out := buf.String(); strings.Contains(out, "trace_id") {
		t.Fatalf("无 span 时不应出现 trace_id 字段，实际输出：%s", out)
	}
}

// 场景 S3（regression）：四级日志在 Warn 阈值下仅输出 Warn/Error，
// 保证分级纪律（Debug/Info 被抑制）不因本次改造回退。
func TestLoggerLevelFiltering(t *testing.T) {
	l, buf := newCaptureLogger(zapcore.WarnLevel)
	defer useCaptureLogger(l)()

	l.Debug("调试")
	l.Info("信息")
	l.Warn("警告")
	l.Error("错误")

	out := buf.String()
	if strings.Contains(out, "调试") || strings.Contains(out, "信息") {
		t.Fatalf("Warn 阈值下不应输出 Debug/Info，实际输出：%s", out)
	}
	if !strings.Contains(out, "警告") || !strings.Contains(out, "错误") {
		t.Fatalf("Warn 阈值下应输出 Warn/Error，实际输出：%s", out)
	}
}
