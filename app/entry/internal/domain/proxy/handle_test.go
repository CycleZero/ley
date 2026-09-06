package proxy

// ===================== 代理转发核心测试（handle_test.go） =====================
//
// 覆盖 callProto 骨架的四条主链路 + 边界：
//  1. 请求体绑定（protojson snake_case 输入 → proto 请求，未知字段容忍）；
//  2. 路径参数合并（BindPath* 显式辅助，含非法值 400 信封）；
//  3. 元数据透传（KEY：x-md-global-auth-user-id/name/role 到达 server 端，
//     证明 infra 拨号的 metadata.Client() 接线正确；匿名请求不携带认证键）；
//  4. 信封契约（成功 data 为 protojson UseProtoNames 的 snake_case 原始 JSON；
//     失败 code 镜像 HTTP 状态、msg 为中文）。
//
// fake gRPC server 见 helpers_test.go：-short 零外部依赖。

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	authv1 "github.com/CycleZero/ley/api/auth/v1"
	blogv1 "github.com/CycleZero/ley/api/blog/v1"
	pkgmeta "github.com/CycleZero/ley/pkg/meta"
	"github.com/gin-gonic/gin"
	kerrors "github.com/go-kratos/kratos/v2/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
)

// testEnvelope 本地信封结构：Data 以 RawMessage 保留原始 JSON，
// 便于断言 data 内部的 snake_case 契约（避免 map 解码丢失键序/类型）。
type testEnvelope struct {
	Code int             `json:"code"`
	Msg  string          `json:"msg"`
	Data json.RawMessage `json:"data"`
}

// registerLogin 注册登录路由（handler 形态与后续 wave 的真实 handler 一致）。
func registerLogin(env *testEnv) {
	env.engine.POST("/api/auth/login", func(c *gin.Context) {
		req := &authv1.LoginRequest{}
		callProto(c, req, func(ctx context.Context, m proto.Message) (proto.Message, error) {
			return env.hub.Auth.Login(ctx, m.(*authv1.LoginRequest))
		})
	})
}

// registerProfile 注册个人资料路由（认证中间件种入 meta，模拟 T4 行为）。
func registerProfile(env *testEnv) {
	env.engine.GET("/api/auth/profile", withAuthMeta(42, "tester", "admin"), func(c *gin.Context) {
		req := &authv1.GetProfileRequest{}
		callProto(c, req, func(ctx context.Context, m proto.Message) (proto.Message, error) {
			return env.hub.Auth.GetProfile(ctx, m.(*authv1.GetProfileRequest))
		})
	})
}

// registerDeleteArticle 注册删除文章路由：演示路径参数合并辅助的标准用法。
func registerDeleteArticle(env *testEnv) {
	env.engine.DELETE("/api/blog/articles/:id", withAuthMeta(7, "author1", "author"), func(c *gin.Context) {
		req := &blogv1.DeleteArticleRequest{}
		// 路径参数合并（显式逐字段）：路由 :id → proto Id 字段
		if !BindPathUint64(c, "id", &req.Id) {
			return
		}
		callProto(c, req, func(ctx context.Context, m proto.Message) (proto.Message, error) {
			return env.hub.Article.DeleteArticle(ctx, m.(*blogv1.DeleteArticleRequest))
		})
	})
}

// doRequest 以固定 X-Forwarded-For 执行一次 HTTP 请求（真实 IP 断言依赖它）。
func doRequest(t *testing.T, engine *gin.Engine, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	var rdr io.Reader
	if body != "" {
		rdr = strings.NewReader(body)
	}
	req := httptest.NewRequest(method, path, rdr)
	req.Header.Set("X-Forwarded-For", "203.0.113.9")
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)
	return w
}

// decodeEnvelope 解析响应为信封结构。
func decodeEnvelope(t *testing.T, w *httptest.ResponseRecorder) testEnvelope {
	t.Helper()
	var env testEnvelope
	if err := json.Unmarshal(w.Body.Bytes(), &env); err != nil {
		t.Fatalf("解析响应体失败: %v，body=%s", err, w.Body.String())
	}
	return env
}

// dataMap 将信封 data 解为 map（data 为 null 时返回空 map）。
func dataMap(t *testing.T, env testEnvelope) map[string]any {
	t.Helper()
	if len(env.Data) == 0 || string(env.Data) == "null" {
		return nil
	}
	m := map[string]any{}
	if err := json.Unmarshal(env.Data, &m); err != nil {
		t.Fatalf("解析 data 失败: %v，data=%s", err, env.Data)
	}
	return m
}

// ===================== 1. 请求体绑定 + 成功信封（snake_case） =====================

// TestCallProto_Login_SuccessEnvelope：Given 合法登录 body；
// When 经 callProto 转发到 fake auth；Then 断言 200 成功信封、
// data 为 protojson UseProtoNames 的 snake_case 原始 JSON（无 camelCase 键）、
// fake server 收到绑定后的请求字段。
func TestCallProto_Login_SuccessEnvelope(t *testing.T) {
	// Given
	env := newTestEnv(t)
	registerLogin(env)
	// When
	w := doRequest(t, env.engine, http.MethodPost, "/api/auth/login", `{"account":"alice","password":"secret123"}`)
	// Then：信封契约
	if w.Code != http.StatusOK {
		t.Fatalf("HTTP 状态 = %d, 期望 %d，body=%s", w.Code, http.StatusOK, w.Body.String())
	}
	resp := decodeEnvelope(t, w)
	if resp.Code != 0 || resp.Msg != "ok" {
		t.Errorf("信封 = {code:%d msg:%q}, 期望 {code:0 msg:ok}", resp.Code, resp.Msg)
	}
	raw := string(resp.Data)
	// snake_case 契约：proto 字段名（token_pair/access_token），而非 json_name camelCase
	for _, want := range []string{`"token_pair":`, `"access_token":"access-token-1"`, `"username":"alice"`} {
		if !strings.Contains(raw, want) {
			t.Errorf("data 缺少 snake_case 片段 %s，data=%s", want, raw)
		}
	}
	for _, forbid := range []string{`"tokenPair"`, `"accessToken"`, `"userName"`} {
		if strings.Contains(raw, forbid) {
			t.Errorf("data 出现 camelCase 键 %s（期望 UseProtoNames），data=%s", forbid, raw)
		}
	}
	// Then：fake server 收到请求体绑定结果
	snap, ok := env.fakeAuth.lastLogin()
	if !ok {
		t.Fatal("fake auth 未收到 Login 调用")
	}
	if snap.account != "alice" || snap.password != "secret123" {
		t.Errorf("fake 收到 account=%q password=%q, 期望 alice/secret123", snap.account, snap.password)
	}
}

// TestCallProto_Login_DiscardUnknownField：Given body 携带下游 proto 未知字段；
// When 转发；Then 请求仍成功（DiscardUnknown 容忍，与宽松网关契约一致）。
func TestCallProto_Login_DiscardUnknownField(t *testing.T) {
	// Given
	env := newTestEnv(t)
	registerLogin(env)
	// When
	w := doRequest(t, env.engine, http.MethodPost, "/api/auth/login", `{"account":"alice","password":"x","unknown_field":true}`)
	// Then
	if w.Code != http.StatusOK {
		t.Fatalf("HTTP 状态 = %d, 期望 200（未知字段应被忽略），body=%s", w.Code, w.Body.String())
	}
}

// TestCallProto_Login_BadJSON：Given 非法 JSON body；When 转发；
// Then 400 信封 + 参数解析失败，且不触发下游调用。
func TestCallProto_Login_BadJSON(t *testing.T) {
	// Given
	env := newTestEnv(t)
	registerLogin(env)
	// When
	w := doRequest(t, env.engine, http.MethodPost, "/api/auth/login", `{"account":`)
	// Then
	if w.Code != http.StatusBadRequest {
		t.Fatalf("HTTP 状态 = %d, 期望 %d，body=%s", w.Code, http.StatusBadRequest, w.Body.String())
	}
	resp := decodeEnvelope(t, w)
	if resp.Code != http.StatusBadRequest || resp.Msg != "参数解析失败" {
		t.Errorf("信封 = {code:%d msg:%q}, 期望 {code:400 msg:参数解析失败}", resp.Code, resp.Msg)
	}
	if _, ok := env.fakeAuth.lastLogin(); ok {
		t.Error("非法 JSON 不应触发下游 Login 调用")
	}
}

// ===================== 2. 元数据透传（核心断言） =====================

// TestCallProto_MetaPropagation：Given 认证中间件种入 uid=42/name=tester/role=admin、
// 请求携带 X-Forwarded-For；When GET 个人资料经 callProto 转发；
// Then fake server 端收到的 incoming metadata 含全部 x-md-global-auth-* 键——
// 证明 infra 拨号的 kratos metadata.Client() 把 pkg/meta.NewClientCtx 注入的
// 上下文正确序列化到 gRPC wire（entry 元数据透传的核心证据）。
func TestCallProto_MetaPropagation(t *testing.T) {
	// Given
	env := newTestEnv(t)
	registerProfile(env)
	// When
	w := doRequest(t, env.engine, http.MethodGet, "/api/auth/profile", "")
	// Then
	if w.Code != http.StatusOK {
		t.Fatalf("HTTP 状态 = %d, 期望 %d，body=%s", w.Code, http.StatusOK, w.Body.String())
	}
	snap, ok := env.fakeAuth.lastProfile()
	if !ok {
		t.Fatal("fake auth 未收到 GetProfile 调用")
	}
	if got := mdVal(t, snap.md, pkgmeta.AuthUserIDKey); got != "42" {
		t.Errorf("x-md-global-auth-user-id = %q, 期望 %q", got, "42")
	}
	if got := mdVal(t, snap.md, pkgmeta.AuthUserNameKey); got != "tester" {
		t.Errorf("x-md-global-auth-user-name = %q, 期望 %q", got, "tester")
	}
	if got := mdVal(t, snap.md, pkgmeta.AuthUserRoleKey); got != "admin" {
		t.Errorf("x-md-global-auth-user-role = %q, 期望 %q", got, "admin")
	}
	if got := mdVal(t, snap.md, pkgmeta.AuthRealClientIpKey); got != "203.0.113.9" {
		t.Errorf("x-md-global-auth-real-ip = %q, 期望 %q（X-Forwarded-For 首段）", got, "203.0.113.9")
	}
}

// TestCallProto_AnonymousNoAuthKeys：Given 未认证请求（登录接口，无 meta 种入）；
// When 转发；Then server 端不出现任何 x-md-global-auth-user-* 键，
// 但真实 IP 仍透传（BuildRequestMeta 对匿名请求回填 IP）。
func TestCallProto_AnonymousNoAuthKeys(t *testing.T) {
	// Given
	env := newTestEnv(t)
	registerLogin(env)
	// When
	w := doRequest(t, env.engine, http.MethodPost, "/api/auth/login", `{"account":"alice","password":"x"}`)
	// Then
	if w.Code != http.StatusOK {
		t.Fatalf("HTTP 状态 = %d, 期望 %d，body=%s", w.Code, http.StatusOK, w.Body.String())
	}
	snap, ok := env.fakeAuth.lastLogin()
	if !ok {
		t.Fatal("fake auth 未收到 Login 调用")
	}
	for _, key := range []string{pkgmeta.AuthUserIDKey, pkgmeta.AuthUserNameKey, pkgmeta.AuthUserRoleKey} {
		mdAbsent(t, snap.md, key)
	}
	if got := mdVal(t, snap.md, pkgmeta.AuthRealClientIpKey); got != "203.0.113.9" {
		t.Errorf("x-md-global-auth-real-ip = %q, 期望 %q", got, "203.0.113.9")
	}
}

// ===================== 3. 路径参数合并 =====================

// TestCallProto_DeleteArticle_PathParamAndMeta：Given 认证请求 DELETE :id=99；
// When handler 经 BindPathUint64 合并路径参数后 callProto 转发；
// Then fake server 收到 Id=99 且认证元数据（uid=7）同样到达。
func TestCallProto_DeleteArticle_PathParamAndMeta(t *testing.T) {
	// Given
	env := newTestEnv(t)
	registerDeleteArticle(env)
	// When
	w := doRequest(t, env.engine, http.MethodDelete, "/api/blog/articles/99", "")
	// Then
	if w.Code != http.StatusOK {
		t.Fatalf("HTTP 状态 = %d, 期望 %d，body=%s", w.Code, http.StatusOK, w.Body.String())
	}
	resp := decodeEnvelope(t, w)
	if resp.Code != 0 {
		t.Errorf("业务码 = %d, 期望 0（删除类空回复 data 应为 {}）", resp.Code)
	}
	if raw := string(resp.Data); raw != "{}" {
		t.Errorf("data = %s, 期望 {}（空回复序列化结果）", raw)
	}
	snap, ok := env.fakeBlog.lastDelete()
	if !ok {
		t.Fatal("fake blog 未收到 DeleteArticle 调用")
	}
	if snap.id != 99 {
		t.Errorf("fake 收到 Id = %d, 期望 99（路径参数合并失败）", snap.id)
	}
	if got := mdVal(t, snap.md, pkgmeta.AuthUserIDKey); got != "7" {
		t.Errorf("x-md-global-auth-user-id = %q, 期望 %q", got, "7")
	}
	if got := mdVal(t, snap.md, pkgmeta.AuthUserRoleKey); got != "author" {
		t.Errorf("x-md-global-auth-user-role = %q, 期望 %q", got, "author")
	}
}

// TestCallProto_DeleteArticle_BadPathParam：Given :id 非法（非整数）；
// When BindPathUint64 解析失败；Then 400 信封并短路——不触发下游调用。
func TestCallProto_DeleteArticle_BadPathParam(t *testing.T) {
	// Given
	env := newTestEnv(t)
	registerDeleteArticle(env)
	// When
	w := doRequest(t, env.engine, http.MethodDelete, "/api/blog/articles/abc", "")
	// Then
	if w.Code != http.StatusBadRequest {
		t.Fatalf("HTTP 状态 = %d, 期望 %d，body=%s", w.Code, http.StatusBadRequest, w.Body.String())
	}
	resp := decodeEnvelope(t, w)
	if resp.Code != http.StatusBadRequest || !strings.Contains(resp.Msg, "必须为无符号整数") {
		t.Errorf("信封 = {code:%d msg:%q}, 期望 {code:400 msg:含'必须为无符号整数'}", resp.Code, resp.Msg)
	}
	if _, ok := env.fakeBlog.lastDelete(); ok {
		t.Error("非法路径参数不应触发下游 DeleteArticle 调用")
	}
}

// ===================== 4. 错误映射 =====================

// TestCallProto_GRPCStatusError：Given fake 返回 gRPC NotFound（原始 status 错误）；
// When 转发失败；Then 404 信封、msg 取 gRPC 状态消息（中文透出）。
func TestCallProto_GRPCStatusError(t *testing.T) {
	// Given
	env := newTestEnv(t)
	env.fakeAuth.loginErr = status.Error(codes.NotFound, "账号或密码错误")
	registerLogin(env)
	// When
	w := doRequest(t, env.engine, http.MethodPost, "/api/auth/login", `{"account":"alice","password":"wrong"}`)
	// Then
	if w.Code != http.StatusNotFound {
		t.Fatalf("HTTP 状态 = %d, 期望 %d，body=%s", w.Code, http.StatusNotFound, w.Body.String())
	}
	resp := decodeEnvelope(t, w)
	if resp.Code != http.StatusNotFound || resp.Msg != "账号或密码错误" {
		t.Errorf("信封 = {code:%d msg:%q}, 期望 {code:404 msg:账号或密码错误}", resp.Code, resp.Msg)
	}
	if resp.Data != nil && string(resp.Data) != "null" {
		t.Errorf("失败信封 data = %s, 期望 null", resp.Data)
	}
}

// TestCallProto_KratosError：Given fake 返回 kratos 业务错误（带 ErrorInfo reason）；
// When 转发失败；Then 404 信封、msg 取 kratos 业务消息（模拟真实微服务错误形态）。
func TestCallProto_KratosError(t *testing.T) {
	// Given
	env := newTestEnv(t)
	env.fakeAuth.loginErr = kerrors.NotFound("USER_NOT_FOUND", "账号或密码错误")
	registerLogin(env)
	// When
	w := doRequest(t, env.engine, http.MethodPost, "/api/auth/login", `{"account":"alice","password":"wrong"}`)
	// Then
	if w.Code != http.StatusNotFound {
		t.Fatalf("HTTP 状态 = %d, 期望 %d，body=%s", w.Code, http.StatusNotFound, w.Body.String())
	}
	resp := decodeEnvelope(t, w)
	if resp.Code != http.StatusNotFound || resp.Msg != "账号或密码错误" {
		t.Errorf("信封 = {code:%d msg:%q}, 期望 {code:404 msg:账号或密码错误}", resp.Code, resp.Msg)
	}
}
