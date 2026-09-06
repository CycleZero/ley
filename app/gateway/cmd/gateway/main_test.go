package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// B-303: 健康端点静态 200，其余路径透传给代理
func TestHealthHandler(t *testing.T) {
	var proxied int
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { proxied++ })
	h := healthHandler(inner)

	for _, path := range []string{"/healthz", "/readyz"} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Errorf("%s 应返回 200: got %d", path, rec.Code)
		}
		if rec.Body.String() == "" {
			t.Errorf("%s body 不应为空", path)
		}
	}
	if proxied != 0 {
		t.Errorf("健康端点不应命中代理: proxied=%d", proxied)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/articles", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if proxied != 1 {
		t.Error("非健康路径应透传给代理")
	}
}
