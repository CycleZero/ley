package middleware

import (
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
)

// TestCORSDefaultWildcard 默认形态（allowOrigins 为空）：放行所有来源 "*"。
func TestCORSDefaultWildcard(t *testing.T) {
	e := newEngine(CORS(nil))

	// ── 预检请求：204 中止（不进入业务 handler）+ 全套 CORS 头 ──
	rec := perform(t, e, http.MethodOptions, "/ping", map[string]string{
		"Origin":                        "http://example.com",
		"Access-Control-Request-Method": "GET",
	})
	wantStatus(t, rec, http.StatusNoContent)
	if rec.Body.Len() != 0 {
		t.Fatalf("预检 204 不应有响应体，实际 %q", rec.Body.String())
	}
	h := rec.Header()
	if got := h.Get("Access-Control-Allow-Origin"); got != "*" {
		t.Fatalf("默认应放行所有来源，Allow-Origin 期望 *，实际 %q", got)
	}
	if got := h.Get("Access-Control-Allow-Methods"); got != corsAllowMethods {
		t.Fatalf("Allow-Methods 不符：期望 %q，实际 %q", corsAllowMethods, got)
	}
	if got := h.Get("Access-Control-Allow-Headers"); got != corsAllowHeaders {
		t.Fatalf("Allow-Headers 不符：期望 %q，实际 %q", corsAllowHeaders, got)
	}
	if got := h.Get("Access-Control-Max-Age"); got != corsMaxAge {
		t.Fatalf("Max-Age 不符：期望 %q，实际 %q", corsMaxAge, got)
	}
	// "*" 与凭据互斥（浏览器规范）：wildcard 形态不得携带 Allow-Credentials
	if got := h.Get("Access-Control-Allow-Credentials"); got != "" {
		t.Fatalf("wildcard 形态不应返回 Allow-Credentials，实际 %q", got)
	}

	// ── 正常请求：追加 CORS 头并放行 handler ──
	rec = perform(t, e, http.MethodGet, "/ping", map[string]string{"Origin": "http://example.com"})
	wantStatus(t, rec, http.StatusOK)
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "*" {
		t.Fatalf("正常请求应追加 Allow-Origin: *，实际 %q", got)
	}
}

// TestCORSConfiguredOrigins 显式来源列表形态：匹配来源回显 + 凭据，不匹配不返回 CORS 头。
func TestCORSConfiguredOrigins(t *testing.T) {
	origins := []string{"http://a.example", "http://b.example"}
	e := newEngine(CORS(origins))

	// ── 匹配来源的正常请求：回显 Origin + 允许凭据 ──
	rec := perform(t, e, http.MethodGet, "/ping", map[string]string{"Origin": "http://b.example"})
	wantStatus(t, rec, http.StatusOK)
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "http://b.example" {
		t.Fatalf("匹配来源应回显 Origin，期望 %q，实际 %q", "http://b.example", got)
	}
	if got := rec.Header().Get("Access-Control-Allow-Credentials"); got != "true" {
		t.Fatalf("指定来源形态应允许凭据，实际 %q", got)
	}

	// ── 匹配来源的预检：204 + 回显 ──
	rec = perform(t, e, http.MethodOptions, "/ping", map[string]string{"Origin": "http://a.example"})
	wantStatus(t, rec, http.StatusNoContent)
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "http://a.example" {
		t.Fatalf("预检应回显匹配 Origin，期望 %q，实际 %q", "http://a.example", got)
	}

	// ── 未匹配来源：正常请求放行但无 CORS 头（浏览器自行拦截）──
	rec = perform(t, e, http.MethodGet, "/ping", map[string]string{"Origin": "http://evil.example"})
	wantStatus(t, rec, http.StatusOK)
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Fatalf("未匹配来源不应返回 Allow-Origin，实际 %q", got)
	}

	// ── 未匹配来源的预检：204 无 CORS 头 ──
	rec = perform(t, e, http.MethodOptions, "/ping", map[string]string{"Origin": "http://evil.example"})
	wantStatus(t, rec, http.StatusNoContent)
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Fatalf("未匹配来源预检不应返回 Allow-Origin，实际 %q", got)
	}

	// ── 同源请求（无 Origin 头）：不需要 CORS，无 CORS 头 ──
	rec = perform(t, e, http.MethodGet, "/ping", nil)
	wantStatus(t, rec, http.StatusOK)
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Fatalf("同源请求不应返回 Allow-Origin，实际 %q", got)
	}
}

// TestCORSPreflightSkipsHandler 预检必须中止链路：业务 handler 不得执行。
func TestCORSPreflightSkipsHandler(t *testing.T) {
	var hit bool
	e := gin.New()
	e.Use(CORS([]string{"http://a.example"}))
	e.GET("/side-effect", func(c *gin.Context) {
		hit = true // 若预检走到这里即失败
		c.Status(http.StatusOK)
	})

	rec := perform(t, e, http.MethodOptions, "/side-effect", map[string]string{"Origin": "http://a.example"})
	wantStatus(t, rec, http.StatusNoContent)
	if hit {
		t.Fatalf("预检请求不应进入业务 handler")
	}
}
