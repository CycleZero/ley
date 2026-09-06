package middleware

import (
	"net/http"
	"testing"
	"time"

	"golang.org/x/time/rate"
)

// TestRateLimiter 令牌桶限流：burst 内放行、超出 429 信封、不同 IP 独立桶。
//
// 确定性策略：速率取 rate.Every(time.Hour)（≈0.0003 req/s）——测试时间窗口内
// refill 可忽略，判定完全由 burst 决定，不依赖 wall clock。
func TestRateLimiter(t *testing.T) {
	slow := rate.Every(time.Hour)
	const ipA = "192.0.2.1:12345"
	const ipB = "10.1.2.3:54321"

	e := newEngine(NewRateLimiter(slow, 3))

	// ── burst 容量内 3 连发全部放行（同一 IP）──
	for i := 0; i < 3; i++ {
		rec := performFrom(t, e, http.MethodGet, "/ping", ipA)
		if rec.Code != http.StatusOK {
			t.Fatalf("第 %d 次请求应在 burst 内放行，实际 %d", i+1, rec.Code)
		}
	}

	// ── 第 4 次超出 burst → 429 信封 ──
	rec := performFrom(t, e, http.MethodGet, "/ping", ipA)
	wantFail(t, rec, http.StatusTooManyRequests, "请求过于频繁，请稍后再试")

	// ── 不同 IP 独立桶：B 桶仍是满的，burst 3 连发全部放行 ──
	for i := 0; i < 3; i++ {
		rec := performFrom(t, e, http.MethodGet, "/ping", ipB)
		if rec.Code != http.StatusOK {
			t.Fatalf("IP B 第 %d 次请求应使用独立桶放行，实际 %d", i+1, rec.Code)
		}
	}
	// A 桶被限流不影响 B（上一环已证）；B 桶超发同样 429
	rec = performFrom(t, e, http.MethodGet, "/ping", ipB)
	wantFail(t, rec, http.StatusTooManyRequests, "请求过于频繁，请稍后再试")
}

// TestRateLimiterDisabledWhenLimitZero 速率 <=0 视为未启用：直接放行不拦截
// （远程业务配置缺省 rps=0 即此形态，兜底保证构造安全）。
func TestRateLimiterDisabledWhenLimitZero(t *testing.T) {
	e := newEngine(NewRateLimiter(0, 3))
	// 连续多次请求全部放行
	for i := 0; i < 5; i++ {
		rec := performFrom(t, e, http.MethodGet, "/ping", "192.0.2.1:12345")
		if rec.Code != http.StatusOK {
			t.Fatalf("限流禁用时应全部放行，第 %d 次实际 %d", i+1, rec.Code)
		}
	}
}
