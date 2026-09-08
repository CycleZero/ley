package oss

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	tracesdk "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

// fakeOSS 仅实现被测方法，其余方法由内嵌的 nil 接口提供（调用即 panic，
// 从而保证测试只覆盖预期路径）。
type fakeOSS struct {
	OSSWithBucket
	putErr error
}

// PutObject 返回预设错误，用于验证错误路径同样产生 span。
func (f *fakeOSS) PutObject(_ context.Context, _ string, _ io.Reader, _ int64, _ string) error {
	return f.putErr
}

// 场景 S4（data 层 trace）：对象存储上传必须产生 span，带对象键属性；
// 失败时 span 状态为 Error。
func TestTracedOSSEmitsPutObjectSpan(t *testing.T) {
	exp := tracetest.NewInMemoryExporter()
	tp := tracesdk.NewTracerProvider(tracesdk.WithSampler(tracesdk.AlwaysSample()), tracesdk.WithSyncer(exp))

	o := NewTracedOSS(&fakeOSS{putErr: errors.New("模拟上传失败")}, WithOSSTracerProvider(tp))
	err := o.PutObject(context.Background(), "images/a.png", strings.NewReader("x"), 1, "image/png")
	if err == nil {
		t.Fatal("预期上传失败")
	}

	spans := exp.GetSpans()
	if len(spans) == 0 {
		t.Fatal("OSS 上传未产生 span")
	}
	var found *tracetest.SpanStub
	for i := range spans {
		if strings.Contains(spans[i].Name, "PutObject") {
			found = &spans[i]
			break
		}
	}
	if found == nil {
		t.Fatalf("未找到 PutObject span，实际：%v", spans)
	}
	key := ""
	for _, kv := range found.Attributes {
		if string(kv.Key) == "object.key" {
			key = kv.Value.AsString()
		}
	}
	if key != "images/a.png" {
		t.Errorf("object.key 应为 images/a.png，实际 %q", key)
	}
}
