package common

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	pkgmeta "github.com/CycleZero/ley/pkg/meta"
)

// newTestCtx 构造带指定直连地址（RemoteAddr）的 gin 测试上下文。
func newTestCtx(remoteAddr string) (*gin.Context, *httptest.ResponseRecorder) {
	gin.SetMode(gin.TestMode) // 关闭 gin 调试输出，保证测试输出干净
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/example", nil)
	c.Request.RemoteAddr = remoteAddr
	return c, w
}

// TestBuildRequestMeta_RealClientIPPriority 真实客户端 IP 解析优先级：
// X-Forwarded-For 首段 → X-Real-IP → ClientIP(RemoteAddr) 兜底。
func TestBuildRequestMeta_RealClientIPPriority(t *testing.T) {
	cases := []struct {
		name       string
		xff        string // X-Forwarded-For 请求头
		xRealIP    string // X-Real-IP 请求头
		remoteAddr string // 直连地址
		want       string // 期望解析结果
	}{
		{
			name:       "XFF 首段优先于 X-Real-IP 与直连地址",
			xff:        "1.2.3.4, 10.0.0.2",
			xRealIP:    "9.9.9.9",
			remoteAddr: "198.51.100.1:8888",
			want:       "1.2.3.4",
		},
		{
			name:       "XFF 首段去除首尾空白",
			xff:        " 203.0.113.7 , 203.0.113.8 ",
			xRealIP:    "9.9.9.9",
			remoteAddr: "198.51.100.1:8888",
			want:       "203.0.113.7",
		},
		{
			name:       "无 XFF 时回退 X-Real-IP",
			xRealIP:    "9.9.9.9",
			remoteAddr: "198.51.100.1:8888",
			want:       "9.9.9.9",
		},
		{
			name:       "XFF 为纯空白时回退 X-Real-IP",
			xff:        "   ",
			xRealIP:    "9.9.9.9",
			remoteAddr: "198.51.100.1:8888",
			want:       "9.9.9.9",
		},
		{
			name:       "XFF 与 X-Real-IP 均缺失时回退 ClientIP",
			remoteAddr: "203.0.113.9:1234",
			want:       "203.0.113.9",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// Given: 种入各 IP 来源
			c, _ := newTestCtx(tc.remoteAddr)
			if tc.xff != "" {
				c.Request.Header.Set("X-Forwarded-For", tc.xff)
			}
			if tc.xRealIP != "" {
				c.Request.Header.Set("X-Real-IP", tc.xRealIP)
			}
			// When: 组装请求元数据
			m := BuildRequestMeta(c)
			// Then: 真实 IP 命中期望来源
			if m.RealClientIp != tc.want {
				t.Errorf("RealClientIp = %q, 期望 %q", m.RealClientIp, tc.want)
			}
		})
	}
}

// TestRequestMeta_SeedAndRead 元数据种入与读取：SetRequestMeta 后
// GetRequestMeta 原样取回（同一指针），未种入时返回 nil。
func TestRequestMeta_SeedAndRead(t *testing.T) {
	// Given: 认证中间件种入元数据
	c, _ := newTestCtx("203.0.113.9:1234")
	seeded := &RequestMetaData{Auth: Auth{UserID: 42, UserName: "alice", Role: "admin"}}
	SetRequestMeta(c, seeded)
	// When: 从上下文读取
	got := GetRequestMeta(c)
	// Then: 取回同一实例，字段一致
	if got != seeded {
		t.Fatalf("GetRequestMeta 未返回种入的同一实例")
	}
	if got.Auth.UserID != 42 || got.Auth.UserName != "alice" || got.Auth.Role != "admin" {
		t.Errorf("auth = %+v, 期望 {UserID:42 UserName:alice Role:admin}", got.Auth)
	}

	// Given: 匿名请求（中间件未种入）
	anonCtx, _ := newTestCtx("203.0.113.9:1234")
	// When / Then: 读取返回 nil
	if m := GetRequestMeta(anonCtx); m != nil {
		t.Errorf("匿名请求 GetRequestMeta = %+v, 期望 nil", m)
	}
}

// TestBuildRequestMeta_AuthConversion 认证信息转换：种入的内部元数据
// 在 BuildRequestMeta 输出中完整转为 pkg/meta 形态；匿名请求输出零值 Auth。
func TestBuildRequestMeta_AuthConversion(t *testing.T) {
	// Given: 已认证请求（中间件种入 Auth + XFF 请求头）
	c, _ := newTestCtx("203.0.113.9:1234")
	c.Request.Header.Set("X-Forwarded-For", "1.2.3.4, 10.0.0.2")
	SetRequestMeta(c, &RequestMetaData{Auth: Auth{UserID: 42, UserName: "alice", Role: "admin"}})
	// When: 组装面向下游的元数据
	m := BuildRequestMeta(c)
	// Then: 字段逐一转换到 pkg/meta.RequestMetaData
	want := &pkgmeta.RequestMetaData{
		Auth:         pkgmeta.Auth{UserID: 42, UserName: "alice", Role: "admin"},
		RealClientIp: "1.2.3.4",
	}
	if *m != *want {
		t.Errorf("BuildRequestMeta = %+v, 期望 %+v", *m, *want)
	}

	// Given: 匿名请求（无 Auth、无 XFF，仅直连地址）
	anonCtx, _ := newTestCtx("203.0.113.9:1234")
	// When: 组装元数据
	anon := BuildRequestMeta(anonCtx)
	// Then: Auth 为零值，真实 IP 兜底仍生效
	if anon.Auth.UserID != 0 || anon.Auth.UserName != "" || anon.Auth.Role != "" {
		t.Errorf("匿名请求 auth = %+v, 期望零值", anon.Auth)
	}
	if anon.RealClientIp != "203.0.113.9" {
		t.Errorf("匿名请求 RealClientIp = %q, 期望兜底 203.0.113.9", anon.RealClientIp)
	}
}
