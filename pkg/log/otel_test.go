package log

import (
	"context"
	"errors"
	"testing"
	"time"

	otellog "go.opentelemetry.io/otel/log"
	"go.opentelemetry.io/otel/log/embedded"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// fakeLogger 捕获 Core 提交的日志记录与上下文，用于断言字段映射结果。
type fakeLogger struct {
	embedded.Logger
	record otellog.Record
	ctx    context.Context
}

func (f *fakeLogger) Emit(ctx context.Context, record otellog.Record) {
	f.ctx = ctx
	f.record = record
}

func (f *fakeLogger) Enabled(context.Context, otellog.EnabledParameters) bool { return true }

const (
	testTraceID = "0af7651916cd43dd8448eb211c80319c"
	testSpanID  = "b7ad6b7169203331"
)

func TestToOtelSeverity(t *testing.T) {
	tests := []struct {
		in   zapcore.Level
		want otellog.Severity
	}{
		{zapcore.DebugLevel, otellog.SeverityDebug},
		{zapcore.InfoLevel, otellog.SeverityInfo},
		{zapcore.WarnLevel, otellog.SeverityWarn},
		{zapcore.ErrorLevel, otellog.SeverityError},
		{zapcore.FatalLevel, otellog.SeverityFatal},
	}
	for _, tt := range tests {
		if got := toOtelSeverity(tt.in); got != tt.want {
			t.Errorf("toOtelSeverity(%v) = %v，期望 %v", tt.in, got, tt.want)
		}
	}
}

func TestTraceContextFromFields(t *testing.T) {
	t.Run("标准 trace_id/span_id", func(t *testing.T) {
		tid, sid, ok := traceContextFromFields([]zapcore.Field{
			zap.String("trace_id", testTraceID),
			zap.String("span_id", testSpanID),
		})
		if !ok || tid.String() != testTraceID || sid.String() != testSpanID {
			t.Errorf("解析结果 = (%s, %s, %v)，期望 (%s, %s, true)", tid, sid, ok, testTraceID, testSpanID)
		}
	})
	t.Run("兼容 Kratos trace.id 写法", func(t *testing.T) {
		tid, _, ok := traceContextFromFields([]zapcore.Field{zap.String("trace.id", testTraceID)})
		if !ok || tid.String() != testTraceID {
			t.Errorf("trace.id 解析失败：(%s, %v)", tid, ok)
		}
	})
	t.Run("无效 trace_id 返回 false", func(t *testing.T) {
		if _, _, ok := traceContextFromFields([]zapcore.Field{zap.String("trace_id", "not-hex")}); ok {
			t.Error("无效 trace_id 不应判定为有效链路")
		}
	})
}

func TestAttributesFromFields(t *testing.T) {
	attrs := attributesFromFields([]zapcore.Field{
		zap.String("method", "GET"),
		zap.Int64("status", 200),
		zap.Bool("ok", true),
		zap.Duration("cost", 1500*time.Millisecond),
		zap.Error(errors.New("boom")),
		zap.String("trace_id", testTraceID),
		zap.String("", "空键忽略"),
	})
	got := map[string]otellog.Value{}
	for _, kv := range attrs {
		got[kv.Key] = kv.Value
	}
	if _, ok := got["trace_id"]; ok {
		t.Error("trace_id 应由链路上下文承载，不应作为属性重复写入")
	}
	if _, ok := got[""]; ok {
		t.Error("空键字段不应写入属性")
	}
	if v := got["method"]; v.AsString() != "GET" {
		t.Errorf("method = %q，期望 GET", v.AsString())
	}
	if v := got["status"]; v.AsInt64() != 200 {
		t.Errorf("status = %d，期望 200", v.AsInt64())
	}
	if v := got["ok"]; !v.AsBool() {
		t.Error("ok 应为 true")
	}
	if v := got["cost"]; v.AsString() != "1.5s" {
		t.Errorf("cost = %q，期望 1.5s", v.AsString())
	}
	if v := got["error"]; v.AsString() != "boom" {
		t.Errorf("error = %q，期望 boom", v.AsString())
	}
}

func TestOTLPLogCoreWrite(t *testing.T) {
	fl := &fakeLogger{}
	core := &otlpLogCore{logger: fl, minLevel: zapcore.InfoLevel}

	ent := zapcore.Entry{Level: zapcore.WarnLevel, Time: time.Unix(1700000000, 0), Message: "登录失败"}
	if err := core.Write(ent, []zapcore.Field{
		zap.String("trace_id", testTraceID),
		zap.String("span_id", testSpanID),
		zap.String("user", "poyuan"),
	}); err != nil {
		t.Fatalf("Write 失败：%v", err)
	}
	if got := fl.record.Body().AsString(); got != "登录失败" {
		t.Errorf("body = %q，期望 登录失败", got)
	}
	if got := fl.record.Severity(); got != otellog.SeverityWarn {
		t.Errorf("severity = %v，期望 WARN", got)
	}
	sc := trace.SpanContextFromContext(fl.ctx)
	if !sc.IsValid() || sc.TraceID().String() != testTraceID || sc.SpanID().String() != testSpanID {
		t.Errorf("链路上下文未正确注入：%v", sc)
	}
	foundUser := false
	fl.record.WalkAttributes(func(kv otellog.KeyValue) bool {
		if kv.Key == "trace_id" {
			t.Error("trace_id 不应作为属性写入")
		}
		if kv.Key == "user" && kv.Value.AsString() == "poyuan" {
			foundUser = true
		}
		return true
	})
	if !foundUser {
		t.Error("缺少 user 属性")
	}
}

func TestOTLPLogCoreWithAndEnabled(t *testing.T) {
	fl := &fakeLogger{}
	base := &otlpLogCore{logger: fl, minLevel: zapcore.InfoLevel}
	if base.Enabled(zapcore.DebugLevel) {
		t.Error("Info 级别下不应放行 Debug")
	}
	if !base.Enabled(zapcore.ErrorLevel) {
		t.Error("Info 级别下应放行 Error")
	}
	child := base.With([]zapcore.Field{zap.String("service", "ley-auth")})
	cc, ok := child.(*otlpLogCore)
	if !ok || len(cc.fields) != 1 {
		t.Fatalf("With 未正确合并字段：%#v", child)
	}
}
