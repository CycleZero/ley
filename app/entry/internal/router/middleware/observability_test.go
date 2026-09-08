package middleware

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/CycleZero/ley/pkg/log"
	"github.com/CycleZero/ley/pkg/metrics"
	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	tracesdk "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// 场景 S1（happy，真实 HTTP 表面）：一次真实 HTTP 请求经过 entry 观测中间件后，
// 必须同时产出：traceparent 续接的 server span、带 trace_id 的访问日志、
// 可被 Prometheus 抓取的 HTTP 指标，且响应体不受插桩影响。
func TestObservabilityEmitsSpanLogAndMetrics(t *testing.T) {
	// 1. 指标 Provider（必须先于 Observability 构建，仪器绑定到该 provider）
	p, err := metrics.New("entry-observability-test")
	if err != nil {
		t.Fatalf("初始化 metrics 失败：%v", err)
	}
	defer func() { _ = p.Shutdown(context.Background()) }()

	// 2. 追踪：内存导出器 + W3C 传播
	exp := tracetest.NewInMemoryExporter()
	tp := tracesdk.NewTracerProvider(tracesdk.WithSampler(tracesdk.AlwaysSample()), tracesdk.WithSyncer(exp))
	prevTP := otel.GetTracerProvider()
	prevProp := otel.GetTextMapPropagator()
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.TraceContext{})
	defer func() {
		otel.SetTracerProvider(prevTP)
		otel.SetTextMapPropagator(prevProp)
	}()

	// 3. 日志捕获
	buf := &bytes.Buffer{}
	enc := zapcore.NewJSONEncoder(zapcore.EncoderConfig{MessageKey: "M", LevelKey: "L", TimeKey: "T", EncodeLevel: zapcore.LowercaseLevelEncoder})
	prevLogger := log.SwapGlobalLogger(&log.Logger{Logger: zap.New(zapcore.NewCore(enc, zapcore.AddSync(buf), zapcore.DebugLevel))})
	defer log.SetGlobalLogger(prevLogger)

	// 4. 真实 Gin 引擎 + 观测中间件 + 访问日志中间件
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(Observability("entry-observability-test"), RequestLogger())
	r.GET("/demo", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"ok": true}) })
	srv := httptest.NewServer(r)
	defer srv.Close()

	// 5. 携带上游 traceparent 的真实 HTTP 请求
	const parentTraceID = "4bf92f3577b34da6a3ce929d0e0e4736"
	req, err := http.NewRequest(http.MethodGet, srv.URL+"/demo", nil)
	if err != nil {
		t.Fatalf("构造请求失败：%v", err)
	}
	req.Header.Set("traceparent", "00-"+parentTraceID+"-00f067aa0ba902b7-01")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("发起请求失败：%v", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	// 断言 1：响应体不受插桩影响
	if resp.StatusCode != http.StatusOK || !strings.Contains(string(body), `"ok":true`) {
		t.Fatalf("响应异常：status=%d body=%s", resp.StatusCode, body)
	}

	// 断言 2：日志带 trace_id（续接上游链路）
	if logs := buf.String(); !strings.Contains(logs, parentTraceID) {
		t.Fatalf("访问日志缺少 trace_id=%s，实际：%s", parentTraceID, logs)
	}

	// 断言 3：产生了 /demo 的 server span，且 trace_id 与上游一致
	spans := exp.GetSpans()
	if len(spans) == 0 {
		t.Fatal("未产生任何 span")
	}
	if got := spans[0].SpanContext.TraceID().String(); got != parentTraceID {
		t.Fatalf("span trace_id 应为上游 %s，实际 %s", parentTraceID, got)
	}
	if !strings.Contains(spans[0].Name, "/demo") {
		t.Fatalf("span 名称应包含 /demo，实际 %s", spans[0].Name)
	}

	// 断言 4：/metrics 暴露 HTTP 指标
	rec := httptest.NewRecorder()
	p.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/metrics", nil))
	if metricsBody := rec.Body.String(); !strings.Contains(metricsBody, "http_server_requests") {
		t.Fatalf("/metrics 未暴露 http_server_requests，实际：%s", metricsBody)
	}
}
