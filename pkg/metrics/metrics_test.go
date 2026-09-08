package metrics

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// scrape 触发一次 /metrics 抓取并返回响应体，模拟 Prometheus 采集表面。
func scrape(t *testing.T, p *Provider) string {
	t.Helper()
	rec := httptest.NewRecorder()
	p.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/metrics", nil))
	body, err := io.ReadAll(rec.Result().Body)
	if err != nil {
		t.Fatalf("读取 /metrics 响应失败：%v", err)
	}
	return string(body)
}

// 场景 S1（happy）：业务计数器经 Prometheus /metrics 端点暴露且数值可被采集。
func TestProviderExposesBusinessCounter(t *testing.T) {
	p, err := New("test-service")
	if err != nil {
		t.Fatalf("创建 metrics Provider 失败：%v", err)
	}
	defer func() { _ = p.Shutdown(context.Background()) }()

	counter, err := p.Int64Counter("demo_requests", "演示请求总数")
	if err != nil {
		t.Fatalf("创建计数器失败：%v", err)
	}
	counter.Add(context.Background(), 3)

	body := scrape(t, p)
	if !strings.Contains(body, "demo_requests_total") {
		t.Fatalf("/metrics 未暴露计数器 demo_requests_total，实际：%s", body)
	}
	if !hasMetricValue(body, "demo_requests_total", "3") {
		t.Fatalf("计数器数值应为 3，实际：%s", body)
	}
}

// hasMetricValue 在 Prometheus 文本格式中查找指定指标名且值为 value 的样本行，
// 忽略标签集合差异（避免断言耦合 exporter 的标签渲染细节）。
func hasMetricValue(body, name, value string) bool {
	for _, line := range strings.Split(body, "\n") {
		if strings.HasPrefix(line, name) && strings.HasSuffix(line, " "+value) {
			return true
		}
	}
	return false
}

// 场景 S1（Kratos 中间件）：Kratos 服务端 metrics 中间件必须把每次调用计入计数器，
// 保证 auth/blog 的 transport 指标开箱即得。
func TestProviderKratosServerMiddlewareRecords(t *testing.T) {
	p, err := New("test-service")
	if err != nil {
		t.Fatalf("创建 metrics Provider 失败：%v", err)
	}
	defer func() { _ = p.Shutdown(context.Background()) }()

	mw, err := p.KratosServerMiddleware()
	if err != nil {
		t.Fatalf("构建 Kratos 服务端 metrics 中间件失败：%v", err)
	}
	handler := mw(func(ctx context.Context, req any) (any, error) { return "ok", nil })
	if _, err := handler(context.Background(), nil); err != nil {
		t.Fatalf("中间件调用失败：%v", err)
	}

	body := scrape(t, p)
	if !strings.Contains(body, "server_requests_code_total") {
		t.Fatalf("Kratos 服务端指标未暴露，实际：%s", body)
	}
}

// 场景 S3（regression）：Shutdown 后 Provider 可安全重复关闭，不 panic。
func TestProviderShutdownIdempotent(t *testing.T) {
	p, err := New("test-service")
	if err != nil {
		t.Fatalf("创建 metrics Provider 失败：%v", err)
	}
	if err := p.Shutdown(context.Background()); err != nil {
		t.Fatalf("首次 Shutdown 失败：%v", err)
	}
	if err := p.Shutdown(context.Background()); err != nil {
		t.Fatalf("重复 Shutdown 应无错误，实际：%v", err)
	}
}
