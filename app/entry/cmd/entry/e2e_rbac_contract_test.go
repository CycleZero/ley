package main

// ===================== E2E RBAC / 限流 / 信封契约场景 =====================
//
// 覆盖剩余端到端证据：
//
//	ADMIN 分类：reader 越权 403 信封且下游不被调用 / admin 放行且 body→gRPC→响应回显；
//	限流：默认配置（rps=0 不挂限流）连续请求不误伤 429（装配受限说明见用例注释）；
//	信封契约：登录成功 {code:0,msg:ok}+token_pair snake_case、404/405 失败信封、
//	OPTIONS 预检 204 + CORS 头（HTTP → 中间件链 → handler → gRPC → fake 全链路）。

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/CycleZero/ley/pkg/meta"
)

// ===================== 8. ADMIN：reader 越权 → 403 =====================

// TestE2E_Admin_ReaderToken_403Forbidden：Given reader token；
// When PUT /api/v1/site/config（ADMIN 分类：强制认证 + RequireRole("admin")）；
// Then 403 信封「无权访问」且 fake blog 未被调用——越权请求被 RBAC 前置拦截，
// 不产生任何下游副作用。
func TestE2E_Admin_ReaderToken_403Forbidden(t *testing.T) {
	// Given
	env := newE2EEnv(t, 0)
	reader := issueToken(t, env, 100, "reader01", "reader")
	// When：reader 尝试更新站点配置
	w := e2eDo(t, env, http.MethodPut, "/api/v1/site/config", reader,
		`{"config":{"site_title":"越权标题"}}`)
	// Then
	wantFail(t, w, http.StatusForbidden, "无权访问")
	wantCommonHeaders(t, w)
	if n := env.fakeBlog.siteUpdateCount(); n != 0 {
		t.Errorf("403 拦截后不应调用下游 UpdateSiteConfig，实际 %d 次", n)
	}
}

// ===================== 9. ADMIN：admin 放行 =====================

// TestE2E_Admin_AdminToken_UpdateSucceeds：Given admin token；
// When PUT /api/v1/site/config（body 携带 site_title/footer_text）；
// Then 200 成功信封 + fake 收到请求字段与 admin 元数据 + 响应回显 config
// （snake_case）——「body 绑定 → 转发 → 响应序列化」闭环证据。
func TestE2E_Admin_AdminToken_UpdateSucceeds(t *testing.T) {
	// Given
	env := newE2EEnv(t, 0)
	admin := issueToken(t, env, 1, "root", "admin")
	// When
	w := e2eDo(t, env, http.MethodPut, "/api/v1/site/config", admin,
		`{"config":{"site_title":"新标题","footer_text":"© 2026 Ley"}}`)
	// Then：信封 + 响应回显（fake 原样返回请求 config）
	envOK := wantOK(t, w)
	wantCommonHeaders(t, w)
	raw := dataStr(t, envOK)
	for _, want := range []string{`"config":`, `"site_title":"新标题"`, `"footer_text":"© 2026 Ley"`} {
		if !strings.Contains(raw, want) {
			t.Errorf("data 缺少 snake_case 片段 %s，data=%s", want, raw)
		}
	}
	// Then：fake 收到请求字段 + admin 元数据
	snap, ok := env.fakeBlog.lastSiteUpdate()
	if !ok {
		t.Fatal("fake blog 未收到 UpdateSiteConfig 调用")
	}
	if snap.title != "新标题" {
		t.Errorf("fake 收到 site_title = %q, 期望 新标题", snap.title)
	}
	if got := mdVal(t, snap.md, meta.AuthUserIDKey); got != "1" {
		t.Errorf("x-md-global-auth-user-id = %q, 期望 1", got)
	}
	if got := mdVal(t, snap.md, meta.AuthUserRoleKey); got != "admin" {
		t.Errorf("x-md-global-auth-user-role = %q, 期望 admin", got)
	}
}

// ===================== 10. 限流：默认配置不误伤（装配受限，标注放宽） =====================

// TestE2E_RateLimit_DefaultUnlimited：Given 默认远程配置（RemoteConfigHolder
// 在无 etcd 的单测下降级为零值：rps=0）；When 连续 5 次请求同一 PUBLIC 路由；
// Then 全部 200——回归护栏：默认配置不得意外挂载限流（rps=0 时 Register()
// 不把 RateLimiter 组装进全局链，见 router/provider.go Register）。
//
// ⚠️ 放宽标注（场景 10 的 429 语义未在本套件做端到端断言）：
// RemoteConfigHolder 的配置只能经 etcd watch 注入（NewRemoteConfigHolder(nil)
// 恒为零值配置，原子指针字段不导出、无测试注入点），因此「rps=1 限流生效」的
// 端到端装配在 app 根测试不可达。429 语义已由 middleware 层单测覆盖：
// ratelimit_test.go TestRateLimiter（同一中间件、同一 429 信封实现），
// 本套件以「默认不挂载、不误伤」作回归互补。
func TestE2E_RateLimit_DefaultUnlimited(t *testing.T) {
	// Given
	env := newE2EEnv(t, 0)
	// When：连续 5 次请求（同 IP）
	for i := 0; i < 5; i++ {
		w := e2eDo(t, env, http.MethodGet, "/api/v1/tags", "", "")
		if w.Code != http.StatusOK {
			t.Fatalf("第 %d 次请求状态 = %d, 期望 200（默认配置不得误限流），body=%s",
				i+1, w.Code, w.Body.String())
		}
	}
}

// ===================== 11. 信封契约 =====================

// TestE2E_EnvelopeContract_LoginAndErrorShapes：契约收口用例——
//
//	成功：POST /api/v1/auth/login → {code:0,msg:ok} + data 内 token_pair/user
//	     全部 snake_case；匿名请求（PUBLIC 分类）不携带认证元数据但真实 IP 透传；
//	兜底：未匹配路由 → 404 {code:404,msg:接口不存在}；
//	方法不允许 → 405 {code:405,msg:请求方法不允许}（Engine.HandleMethodNotAllowed）；
//	预检：OPTIONS → 204 + CORS 头（浏览器跨域契约）。
func TestE2E_EnvelopeContract_LoginAndErrorShapes(t *testing.T) {
	// Given
	env := newE2EEnv(t, 0)

	// When/Then 1：登录成功信封 + snake_case data
	w := e2eDo(t, env, http.MethodPost, "/api/v1/auth/login", "",
		`{"account":"alice","password":"secret123"}`)
	envOK := wantOK(t, w)
	wantCommonHeaders(t, w)
	raw := dataStr(t, envOK)
	for _, want := range []string{
		`"token_pair":`, `"access_token":"e2e-access-token-1"`,
		`"refresh_token":"e2e-refresh-token-1"`, `"expires_in":"900"`,
		`"username":"alice"`,
	} {
		if !strings.Contains(raw, want) {
			t.Errorf("data 缺少 snake_case 片段 %s，data=%s", want, raw)
		}
	}
	// 登录 fake 收到调用：匿名（无认证元数据）+ 真实 IP 透传
	snap, ok := env.fakeAuth.lastLogin()
	if !ok {
		t.Fatal("fake auth 未收到 Login 调用")
	}
	for _, key := range []string{meta.AuthUserIDKey, meta.AuthUserNameKey, meta.AuthUserRoleKey} {
		mdAbsent(t, snap.md, key)
	}
	if got := mdVal(t, snap.md, meta.AuthRealClientIpKey); got != e2eClientIP {
		t.Errorf("x-md-global-auth-real-ip = %q, 期望 %q", got, e2eClientIP)
	}

	// When/Then 2：404 兜底信封（NoRoute 同样走全局中间件链）
	w = e2eDo(t, env, http.MethodGet, "/api/v1/not-exist", "", "")
	wantFail(t, w, http.StatusNotFound, "接口不存在")
	wantCommonHeaders(t, w)

	// When/Then 3：405 方法不允许信封（路径存在但方法不匹配）
	w = e2eDo(t, env, http.MethodPatch, "/api/v1/tags", "", "")
	wantFail(t, w, http.StatusMethodNotAllowed, "请求方法不允许")

	// When/Then 4：CORS 预检 → 204，不进入业务 handler
	req := newPreflight("/api/v1/tags")
	rec := performRaw(t, env, req)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("预检状态 = %d, 期望 204", rec.Code)
	}
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "*" {
		t.Fatalf("预检应携带 Access-Control-Allow-Origin: *，实际 %q", got)
	}
}

// newPreflight 构造 OPTIONS 预检请求（携带 Origin 与预检头）。
func newPreflight(path string) *http.Request {
	req := httptest.NewRequest(http.MethodOptions, path, nil)
	req.Header.Set("Origin", "http://localhost:3000")
	req.Header.Set("Access-Control-Request-Method", "GET")
	req.Header.Set("X-Forwarded-For", e2eClientIP)
	return req
}

// performRaw 以独立响应记录器执行预构造的请求（预检等非常规形态用）。
func performRaw(t *testing.T, env *e2eEnv, req *http.Request) *httptest.ResponseRecorder {
	t.Helper()
	w := httptest.NewRecorder()
	env.app.Engine.ServeHTTP(w, req)
	return w
}
