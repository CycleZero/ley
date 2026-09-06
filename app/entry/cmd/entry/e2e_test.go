package main

// ===================== E2E 认证链路场景（e2e_test.go） =====================
//
// 覆盖权威路由表分类语义的端到端证据（HTTP → 中间件链 → handler → gRPC →
// fake server 全链路，断言点在两端：HTTP 信封 + server 端 x-md-global-* 元数据）：
//
//	PUBLIC（可选认证）：无 token 匿名放行 / 有效 token 注入身份 / 过期 token 匿名回退；
//	AUTH（强制认证）：无 token 401 信封 / 黑名单 token 401「令牌已吊销」（先于解析）/
//	              有效 token 放行且身份透传 / 过期 token 401「无效的认证令牌」。

import (
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/CycleZero/ley/pkg/meta"
)

// ===================== 1. PUBLIC：无 token 匿名访问 =====================

// TestE2E_Public_NoToken_AnonymousForwarded：Given 匿名请求 GET /api/v1/tags
// （PUBLIC 分类，无 Authorization 头）；When 走完整链路到 fake blog；
// Then 200 成功信封 + fake 收到调用且**不含**任何认证元数据（匿名语义），
// 但真实客户端 IP 仍透传；响应带全局链产物（RequestID + CORS 头）。
func TestE2E_Public_NoToken_AnonymousForwarded(t *testing.T) {
	// Given
	env := newE2EEnv(t, 0)
	// When
	w := e2eDo(t, env, http.MethodGet, "/api/v1/tags", "", "")
	// Then：HTTP 信封
	envOK := wantOK(t, w)
	wantCommonHeaders(t, w)
	raw := dataStr(t, envOK)
	for _, want := range []string{`"tags":`, `"id":"1"`, `"name":"Go"`, `"slug":"go"`, `"article_count":"3"`} {
		if !strings.Contains(raw, want) {
			t.Errorf("data 缺少 snake_case 片段 %s，data=%s", want, raw)
		}
	}
	// Then：fake 收到调用且无认证元数据；真实 IP 透传（AddMetaData 种子 → handler 注入）
	snap, ok := env.fakeBlog.lastTagList()
	if !ok {
		t.Fatal("fake blog 未收到 ListTags 调用")
	}
	for _, key := range []string{meta.AuthUserIDKey, meta.AuthUserNameKey, meta.AuthUserRoleKey} {
		mdAbsent(t, snap.md, key)
	}
	if got := mdVal(t, snap.md, meta.AuthRealClientIpKey); got != e2eClientIP {
		t.Errorf("x-md-global-auth-real-ip = %q, 期望 %q", got, e2eClientIP)
	}
}

// ===================== 2. PUBLIC：带有效 token 身份注入 =====================

// TestE2E_Public_ValidToken_IdentityPropagated：Given 携带 reader token 的请求
// GET /api/v1/articles（PUBLIC 分类）；When 走完整链路；Then 200 + fake 收到
// x-md-global-auth-user-id/user-name/role——「可选认证注入身份」的核心透传证据。
func TestE2E_Public_ValidToken_IdentityPropagated(t *testing.T) {
	// Given
	env := newE2EEnv(t, 0)
	token := issueToken(t, env, 42, "tester", "reader")
	// When
	w := e2eDo(t, env, http.MethodGet, "/api/v1/articles", token, "")
	// Then：HTTP 信封 + snake_case
	envOK := wantOK(t, w)
	wantCommonHeaders(t, w)
	raw := dataStr(t, envOK)
	for _, want := range []string{`"articles":`, `"id":"7"`, `"title":"第一篇文章"`, `"slug":"hello-entry"`, `"total":"1"`} {
		if !strings.Contains(raw, want) {
			t.Errorf("data 缺少 snake_case 片段 %s，data=%s", want, raw)
		}
	}
	// Then：fake 收到完整认证元数据（身份透传）
	snap, ok := env.fakeBlog.lastArticleList()
	if !ok {
		t.Fatal("fake blog 未收到 ListArticles 调用")
	}
	if got := mdVal(t, snap.md, meta.AuthUserIDKey); got != "42" {
		t.Errorf("x-md-global-auth-user-id = %q, 期望 42", got)
	}
	if got := mdVal(t, snap.md, meta.AuthUserNameKey); got != "tester" {
		t.Errorf("x-md-global-auth-user-name = %q, 期望 tester", got)
	}
	if got := mdVal(t, snap.md, meta.AuthUserRoleKey); got != "reader" {
		t.Errorf("x-md-global-auth-user-role = %q, 期望 reader", got)
	}
	if got := mdVal(t, snap.md, meta.AuthRealClientIpKey); got != e2eClientIP {
		t.Errorf("x-md-global-auth-real-ip = %q, 期望 %q", got, e2eClientIP)
	}
}

// ===================== 3. PUBLIC：过期 token 回退匿名（可选认证语义） =====================

// TestE2E_Public_ExpiredToken_AnonymousFallback：Given 携带过期 token 的请求
// GET /api/v1/tags；When 走完整链路；Then 仍 200（可选认证对无效令牌放行），
// 但 fake 收到的元数据**不含**认证信息——过期令牌不得注入身份，按匿名处理。
func TestE2E_Public_ExpiredToken_AnonymousFallback(t *testing.T) {
	// Given：负有效期 JWT → 签发出即过期；同签名密钥保证「过期」是唯一失败原因
	env := newE2EEnv(t, -time.Minute)
	expired := issueToken(t, env, 42, "tester", "reader")
	// When
	w := e2eDo(t, env, http.MethodGet, "/api/v1/tags", expired, "")
	// Then：匿名放行 200
	wantOK(t, w)
	// Then：fake 收到调用，但无认证元数据（身份不注入）
	snap, ok := env.fakeBlog.lastTagList()
	if !ok {
		t.Fatal("fake blog 未收到 ListTags 调用")
	}
	for _, key := range []string{meta.AuthUserIDKey, meta.AuthUserNameKey, meta.AuthUserRoleKey} {
		mdAbsent(t, snap.md, key)
	}
}

// ===================== 4. AUTH：无 token → 401 信封 =====================

// TestE2E_Auth_NoToken_401Envelope：Given 匿名请求 GET /api/v1/users/me
// （AUTH 分类，强制认证）；When 走完整链路；Then 401 信封 {code:401,msg:未提供
// 认证令牌,data:null} 且 fake auth 未被调用（前置拦截，未到达下游）。
//
// 信封契约即前端 401-refresh 触发契约：HTTP 状态与业务码同为 401。
func TestE2E_Auth_NoToken_401Envelope(t *testing.T) {
	// Given
	env := newE2EEnv(t, 0)
	// When
	w := e2eDo(t, env, http.MethodGet, "/api/v1/users/me", "", "")
	// Then
	wantFail(t, w, http.StatusUnauthorized, "未提供认证令牌")
	wantCommonHeaders(t, w)
	if n := env.fakeAuth.profileCount(); n != 0 {
		t.Errorf("401 前置拦截后不应调用下游 GetProfile，实际 %d 次", n)
	}
}

// ===================== 5. AUTH：黑名单 token → 401「令牌已吊销」 =====================

// TestE2E_Auth_BlacklistedToken_401Revoked：Given 一枚**合法有效**但已加入黑名单
// 的 token；When 请求 /users/me；Then 401「令牌已吊销」且下游未被调用——
// 证明黑名单检查先于令牌解析（若先解析后查黑名单，合法 token 会放行到下游 200）。
func TestE2E_Auth_BlacklistedToken_401Revoked(t *testing.T) {
	// Given：签发合法 token → 加入启用的内存黑名单
	env := newE2EEnv(t, 0)
	token := issueToken(t, env, 7, "bob", "reader")
	if err := env.blacklist.Add(token); err != nil {
		t.Fatalf("预置黑名单失败: %v", err)
	}
	if !env.blacklist.IsTokenBlackListed(token) {
		t.Fatal("测试前置失败：令牌应已入黑名单")
	}
	// When
	w := e2eDo(t, env, http.MethodGet, "/api/v1/users/me", token, "")
	// Then
	wantFail(t, w, http.StatusUnauthorized, "令牌已吊销")
	wantCommonHeaders(t, w)
	if n := env.fakeAuth.profileCount(); n != 0 {
		t.Errorf("黑名单拦截后不应调用下游 GetProfile，实际 %d 次", n)
	}
}

// ===================== 6. AUTH：有效 token → 放行 + 身份透传 =====================

// TestE2E_Auth_ValidToken_ProfileAndIdentity：Given reader token；
// When GET /api/v1/users/me；Then 200 成功信封 + data 为 fake GetProfile 的
// snake_case JSON（id 为字符串形态）+ server 端收到完整认证元数据。
func TestE2E_Auth_ValidToken_ProfileAndIdentity(t *testing.T) {
	// Given
	env := newE2EEnv(t, 0)
	token := issueToken(t, env, 42, "tester", "reader")
	// When
	w := e2eDo(t, env, http.MethodGet, "/api/v1/users/me", token, "")
	// Then：信封 + protojson uint64 序列化为 JSON 字符串（防精度丢失）
	envOK := wantOK(t, w)
	wantCommonHeaders(t, w)
	raw := dataStr(t, envOK)
	for _, want := range []string{`"user":`, `"id":"42"`, `"username":"tester"`, `"email":"tester@example.com"`, `"role":"reader"`} {
		if !strings.Contains(raw, want) {
			t.Errorf("data 缺少 snake_case 片段 %s，data=%s", want, raw)
		}
	}
	// Then：fake 收到认证元数据（身份不随 body 传递，全靠 x-md-global-* 透传）
	snap, ok := env.fakeAuth.lastProfile()
	if !ok {
		t.Fatal("fake auth 未收到 GetProfile 调用")
	}
	if got := mdVal(t, snap.md, meta.AuthUserIDKey); got != "42" {
		t.Errorf("x-md-global-auth-user-id = %q, 期望 42", got)
	}
	if got := mdVal(t, snap.md, meta.AuthUserRoleKey); got != "reader" {
		t.Errorf("x-md-global-auth-user-role = %q, 期望 reader", got)
	}
	if got := mdVal(t, snap.md, meta.AuthRealClientIpKey); got != e2eClientIP {
		t.Errorf("x-md-global-auth-real-ip = %q, 期望 %q", got, e2eClientIP)
	}
}

// ===================== 7. AUTH：过期 token → 401 信封 =====================

// TestE2E_Auth_ExpiredToken_401Invalid：Given 过期 token（负有效期签发）；
// When GET /api/v1/users/me（强制认证）；Then 401「无效的认证令牌」且下游
// 未被调用——强制认证路径对过期令牌无回退语义。
func TestE2E_Auth_ExpiredToken_401Invalid(t *testing.T) {
	// Given
	env := newE2EEnv(t, -time.Minute)
	expired := issueToken(t, env, 42, "tester", "reader")
	// When
	w := e2eDo(t, env, http.MethodGet, "/api/v1/users/me", expired, "")
	// Then
	wantFail(t, w, http.StatusUnauthorized, "无效的认证令牌")
	if n := env.fakeAuth.profileCount(); n != 0 {
		t.Errorf("401 拦截后不应调用下游 GetProfile，实际 %d 次", n)
	}
}
