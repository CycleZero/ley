package proxy

// ===================== 认证域 handler 测试（auth_test.go） =====================
//
// 覆盖 auth.go 六个 handler 的转发正确性：请求体 JSON（snake_case）→ proto 字段
// 正确到达下游、响应信封与 snake_case 契约、错误映射（注入 gRPC/kratos 错误 →
// HTTP 状态 + 中文消息）、GET 无请求体、认证元数据透传。
//
// fake 扩展说明：helpers_test.go 的 fakeAuthServer 只实现了 Login/GetProfile，
// 其余方法由嵌入的 UnimplementedAuthServiceServer 兜底（返回 Unimplemented）。
// 本文件为同一类型补充 Register/RefreshToken/Logout/UpdateProfile 实现（测试
// 构建下编译进方法集，grpc 注册照常生效），状态按实例经 authFakeStates 表索引
// ——helpers_test.go 的结构体字段不可扩展，故不复用其内字段、也不改动该文件。
// 既有 handle_test.go 测试不受影响：它们不调用这四个 RPC。

import (
	"context"
	"net/http"
	"strings"
	"sync"
	"testing"

	authv1 "github.com/CycleZero/ley/api/auth/v1"
	commonv1 "github.com/CycleZero/ley/api/common/v1"
	pkgmeta "github.com/CycleZero/ley/pkg/meta"
	"github.com/gin-gonic/gin"
	kerrors "github.com/go-kratos/kratos/v2/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// ===================== 扩展 fake 实现（按实例状态表） =====================

// authFakeState 认证扩展 fake 的按实例状态：四类请求快照 + 注入错误。
type authFakeState struct {
	mu      sync.Mutex
	regs    []authRegSnap     // Register 快照
	refs    []authRefreshSnap // RefreshToken 快照
	logouts []authLogoutSnap  // Logout 快照
	updates []authUpdateSnap  // UpdateProfile 快照
	inject  error             // 注入错误：非 nil 时扩展方法直接返回（错误映射测试用）
}

// authRegSnap 一次 Register 调用的 server 端快照。
type authRegSnap struct {
	md       metadata.MD // server 端收到的元数据
	username string      // 请求体绑定结果
	email    string
	password string
}

// authRefreshSnap 一次 RefreshToken 调用的 server 端快照。
type authRefreshSnap struct {
	md           metadata.MD
	refreshToken string // 请求体绑定的 refresh_token
}

// authLogoutSnap 一次 Logout 调用的 server 端快照。
type authLogoutSnap struct {
	md           metadata.MD
	refreshToken string // 请求体绑定的 refresh_token
}

// authUpdateSnap 一次 UpdateProfile 调用的 server 端快照。
type authUpdateSnap struct {
	md     metadata.MD
	avatar string // 请求体绑定结果
	bio    string
}

// authFakeStates 扩展方法状态索引表：每个测试 env 的 fake 实例独立登记，
// t.Cleanup 注销，互不串扰（也兼容未来并行测试）。
var (
	authFakeStatesMu sync.Mutex
	authFakeStates   = make(map[*fakeAuthServer]*authFakeState)
)

// bindAuthFakeState 为 fakeAuth 实例登记扩展状态并在测试结束注销。
func bindAuthFakeState(t *testing.T, f *fakeAuthServer) *authFakeState {
	t.Helper()
	st := &authFakeState{}
	authFakeStatesMu.Lock()
	authFakeStates[f] = st
	authFakeStatesMu.Unlock()
	t.Cleanup(func() {
		authFakeStatesMu.Lock()
		delete(authFakeStates, f)
		authFakeStatesMu.Unlock()
	})
	return st
}

// authFakeStateOf 按实例取扩展状态；未登记（非本文件测试的 env）时返回 nil，
// 此时扩展方法按未实现处理（与 UnimplementedAuthServiceServer 兜底行为一致）。
func authFakeStateOf(f *fakeAuthServer) *authFakeState {
	authFakeStatesMu.Lock()
	defer authFakeStatesMu.Unlock()
	return authFakeStates[f]
}

// Register 扩展 Register fake：记录快照并返回固定成功回复
// （user 回显注册的 username/email + 固定 token_pair）。
func (f *fakeAuthServer) Register(ctx context.Context, in *authv1.RegisterRequest) (*authv1.RegisterReply, error) {
	st := authFakeStateOf(f)
	if st == nil {
		return nil, status.Error(codes.Unimplemented, "未实现 Register")
	}
	if st.inject != nil {
		return nil, st.inject
	}
	st.mu.Lock()
	st.regs = append(st.regs, authRegSnap{
		md: recordIncomingMD(ctx), username: in.Username, email: in.Email, password: in.Password,
	})
	st.mu.Unlock()
	return &authv1.RegisterReply{
		User:      &commonv1.UserInfo{Id: 1, Username: in.Username, Email: in.Email, Role: "reader"},
		TokenPair: &commonv1.TokenPair{AccessToken: "reg-access-token", RefreshToken: "reg-refresh-token", ExpiresIn: 900},
	}, nil
}

// RefreshToken 扩展 RefreshToken fake：固定返回新 token_pair（access/refresh 均换新）。
func (f *fakeAuthServer) RefreshToken(ctx context.Context, in *authv1.RefreshTokenRequest) (*authv1.RefreshTokenReply, error) {
	st := authFakeStateOf(f)
	if st == nil {
		return nil, status.Error(codes.Unimplemented, "未实现 RefreshToken")
	}
	if st.inject != nil {
		return nil, st.inject
	}
	st.mu.Lock()
	st.refs = append(st.refs, authRefreshSnap{md: recordIncomingMD(ctx), refreshToken: in.RefreshToken})
	st.mu.Unlock()
	return &authv1.RefreshTokenReply{
		User:      &commonv1.UserInfo{Id: 42, Username: "tester", Email: "t@example.com", Role: "reader"},
		TokenPair: &commonv1.TokenPair{AccessToken: "refreshed-access-token", RefreshToken: "refreshed-refresh-token", ExpiresIn: 900},
	}, nil
}

// Logout 扩展 Logout fake：记录请求并返回空回复（登出类接口无业务数据）。
func (f *fakeAuthServer) Logout(ctx context.Context, in *authv1.LogoutRequest) (*authv1.LogoutReply, error) {
	st := authFakeStateOf(f)
	if st == nil {
		return nil, status.Error(codes.Unimplemented, "未实现 Logout")
	}
	if st.inject != nil {
		return nil, st.inject
	}
	st.mu.Lock()
	st.logouts = append(st.logouts, authLogoutSnap{md: recordIncomingMD(ctx), refreshToken: in.RefreshToken})
	st.mu.Unlock()
	return &authv1.LogoutReply{}, nil
}

// UpdateProfile 扩展 UpdateProfile fake：固定回显 avatar/bio 到 user 字段
// （便于断言响应侧 snake_case 契约）。
func (f *fakeAuthServer) UpdateProfile(ctx context.Context, in *authv1.UpdateProfileRequest) (*authv1.UpdateProfileReply, error) {
	st := authFakeStateOf(f)
	if st == nil {
		return nil, status.Error(codes.Unimplemented, "未实现 UpdateProfile")
	}
	if st.inject != nil {
		return nil, st.inject
	}
	st.mu.Lock()
	st.updates = append(st.updates, authUpdateSnap{md: recordIncomingMD(ctx), avatar: in.Avatar, bio: in.Bio})
	st.mu.Unlock()
	return &authv1.UpdateProfileReply{
		User: &commonv1.UserInfo{Id: 42, Username: "tester", Email: "t@example.com", Avatar: in.Avatar, Bio: in.Bio, Role: "admin"},
	}, nil
}

// lastRegister 返回最近一次 Register 快照（无调用时第二个返回值为 false）。
func (s *authFakeState) lastRegister() (authRegSnap, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.regs) == 0 {
		return authRegSnap{}, false
	}
	return s.regs[len(s.regs)-1], true
}

// lastRefresh 返回最近一次 RefreshToken 快照。
func (s *authFakeState) lastRefresh() (authRefreshSnap, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.refs) == 0 {
		return authRefreshSnap{}, false
	}
	return s.refs[len(s.refs)-1], true
}

// lastLogout 返回最近一次 Logout 快照。
func (s *authFakeState) lastLogout() (authLogoutSnap, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.logouts) == 0 {
		return authLogoutSnap{}, false
	}
	return s.logouts[len(s.logouts)-1], true
}

// lastUpdate 返回最近一次 UpdateProfile 快照。
func (s *authFakeState) lastUpdate() (authUpdateSnap, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.updates) == 0 {
		return authUpdateSnap{}, false
	}
	return s.updates[len(s.updates)-1], true
}

// ===================== 路由装配 =====================

// registerAuthRoutes 以生产形态注册全部 6 条认证路由：NewAuthHandler(env.hub)
// 装配 handler；需登录的 users/me 路由挂 withAuthMeta 模拟 T4 认证中间件种入
// 元数据。路径与 proto http 注解一致（最终由 router 域 wave 11 统一注册，
// 此处仅测试本文件 handler 的转发正确性）。
func registerAuthRoutes(env *testEnv) {
	h := NewAuthHandler(env.hub)
	env.engine.POST("/api/v1/auth/register", h.Register)
	env.engine.POST("/api/v1/auth/login", h.Login)
	env.engine.POST("/api/v1/auth/refresh", h.RefreshToken)
	env.engine.POST("/api/v1/auth/logout", h.Logout)
	env.engine.GET("/api/v1/users/me", withAuthMeta(42, "tester", "admin"), h.GetProfile)
	env.engine.PUT("/api/v1/users/me", withAuthMeta(42, "tester", "admin"), h.UpdateProfile)
}

// ===================== 测试 =====================

// TestAuth_Register_ForwardsBodyAndEnvelope：Given 匿名注册请求体（snake_case）；
// When 经 Register handler 转发；Then 200 成功信封、data 为 UseProtoNames 的
// snake_case JSON（token_pair）、fake 收到 username/email/password、
// 匿名请求不携带认证元数据但真实 IP 仍透传。
func TestAuth_Register_ForwardsBodyAndEnvelope(t *testing.T) {
	// Given
	env := newTestEnv(t)
	st := bindAuthFakeState(t, env.fakeAuth)
	registerAuthRoutes(env)
	// When
	w := doRequest(t, env.engine, http.MethodPost, "/api/v1/auth/register",
		`{"username":"alice","email":"alice@example.com","password":"Secret123"}`)
	// Then：信封 + snake_case
	if w.Code != http.StatusOK {
		t.Fatalf("HTTP 状态 = %d, 期望 %d，body=%s", w.Code, http.StatusOK, w.Body.String())
	}
	resp := decodeEnvelope(t, w)
	if resp.Code != 0 || resp.Msg != "ok" {
		t.Errorf("信封 = {code:%d msg:%q}, 期望 {code:0 msg:ok}", resp.Code, resp.Msg)
	}
	raw := string(resp.Data)
	for _, want := range []string{`"token_pair":`, `"access_token":"reg-access-token"`, `"username":"alice"`, `"email":"alice@example.com"`} {
		if !strings.Contains(raw, want) {
			t.Errorf("data 缺少 snake_case 片段 %s，data=%s", want, raw)
		}
	}
	// Then：fake 收到请求体绑定结果
	snap, ok := st.lastRegister()
	if !ok {
		t.Fatal("fake auth 未收到 Register 调用")
	}
	if snap.username != "alice" || snap.email != "alice@example.com" || snap.password != "Secret123" {
		t.Errorf("fake 收到 username=%q email=%q password=%q, 期望 alice/alice@example.com/Secret123",
			snap.username, snap.email, snap.password)
	}
	// Then：匿名请求——无认证键，真实 IP 透传
	for _, key := range []string{pkgmeta.AuthUserIDKey, pkgmeta.AuthUserNameKey, pkgmeta.AuthUserRoleKey} {
		mdAbsent(t, snap.md, key)
	}
	if got := mdVal(t, snap.md, pkgmeta.AuthRealClientIpKey); got != "203.0.113.9" {
		t.Errorf("x-md-global-auth-real-ip = %q, 期望 %q", got, "203.0.113.9")
	}
}

// TestAuth_Register_WeakPasswordError：Given fake 注入 kratos BadRequest
// （语义与 auth 服务 user.go 的 ErrPasswordWeak 一致，400↔InvalidArgument
// 往返映射）；When 转发；Then 400 信封 + 中文业务消息透出。
//
// 说明：注册的重复检查错误（ErrUserDuplicate，Conflict 409）在 kratos wire 上
// 编码为 codes.Aborted，已由 common.MapGRPCError 统一映射回 409（见
// grpcerr.go），此处只覆盖 400 一类；409 往返由 handle_test.go 与
// 其他域 handler 的错误用例覆盖。
func TestAuth_Register_WeakPasswordError(t *testing.T) {
	// Given
	env := newTestEnv(t)
	st := bindAuthFakeState(t, env.fakeAuth)
	st.inject = kerrors.BadRequest("PASSWORD_WEAK", "密码必须包含大写字母、小写字母和数字")
	registerAuthRoutes(env)
	// When
	w := doRequest(t, env.engine, http.MethodPost, "/api/v1/auth/register",
		`{"username":"alice","email":"alice@example.com","password":"weak"}`)
	// Then
	if w.Code != http.StatusBadRequest {
		t.Fatalf("HTTP 状态 = %d, 期望 %d，body=%s", w.Code, http.StatusBadRequest, w.Body.String())
	}
	resp := decodeEnvelope(t, w)
	if resp.Code != http.StatusBadRequest || resp.Msg != "密码必须包含大写字母、小写字母和数字" {
		t.Errorf("信封 = {code:%d msg:%q}, 期望 {code:400 msg:密码必须包含大写字母、小写字母和数字}", resp.Code, resp.Msg)
	}
}

// TestAuth_Login_ForwardsBody：Given 登录请求体；When 经 Login handler 转发
// （走 helpers_test.go 的 Login fake）；Then fake 收到 account/password、
// 响应含 snake_case token_pair——证明 handler 装配指向 hub.Auth.Login。
func TestAuth_Login_ForwardsBody(t *testing.T) {
	// Given
	env := newTestEnv(t)
	registerAuthRoutes(env)
	// When
	w := doRequest(t, env.engine, http.MethodPost, "/api/v1/auth/login", `{"account":"alice","password":"secret123"}`)
	// Then
	if w.Code != http.StatusOK {
		t.Fatalf("HTTP 状态 = %d, 期望 %d，body=%s", w.Code, http.StatusOK, w.Body.String())
	}
	snap, ok := env.fakeAuth.lastLogin()
	if !ok {
		t.Fatal("fake auth 未收到 Login 调用")
	}
	if snap.account != "alice" || snap.password != "secret123" {
		t.Errorf("fake 收到 account=%q password=%q, 期望 alice/secret123", snap.account, snap.password)
	}
	if raw := string(decodeEnvelope(t, w).Data); !strings.Contains(raw, `"token_pair":`) {
		t.Errorf("data 缺少 snake_case 片段 token_pair，data=%s", raw)
	}
}

// TestAuth_RefreshToken_Success：Given 携带 refresh_token 的请求体；
// When 经 RefreshToken handler 转发；Then fake 收到 refresh_token、
// 响应为新 token_pair（snake_case）。
func TestAuth_RefreshToken_Success(t *testing.T) {
	// Given
	env := newTestEnv(t)
	st := bindAuthFakeState(t, env.fakeAuth)
	registerAuthRoutes(env)
	// When
	w := doRequest(t, env.engine, http.MethodPost, "/api/v1/auth/refresh", `{"refresh_token":"rt-old-1"}`)
	// Then
	if w.Code != http.StatusOK {
		t.Fatalf("HTTP 状态 = %d, 期望 %d，body=%s", w.Code, http.StatusOK, w.Body.String())
	}
	snap, ok := st.lastRefresh()
	if !ok {
		t.Fatal("fake auth 未收到 RefreshToken 调用")
	}
	if snap.refreshToken != "rt-old-1" {
		t.Errorf("fake 收到 refresh_token=%q, 期望 rt-old-1", snap.refreshToken)
	}
	raw := string(decodeEnvelope(t, w).Data)
	for _, want := range []string{`"access_token":"refreshed-access-token"`, `"refresh_token":"refreshed-refresh-token"`} {
		if !strings.Contains(raw, want) {
			t.Errorf("data 缺少 snake_case 片段 %s，data=%s", want, raw)
		}
	}
}

// TestAuth_RefreshToken_Unauthorized：Given fake 注入 kratos Unauthorized
// （语义与 auth 服务 auth.go 的 TOKEN_INVALID 一致）；When 转发；
// Then 401 信封 + 中文消息透出（无效 refresh_token 语义）。
func TestAuth_RefreshToken_Unauthorized(t *testing.T) {
	// Given
	env := newTestEnv(t)
	st := bindAuthFakeState(t, env.fakeAuth)
	st.inject = kerrors.Unauthorized("TOKEN_INVALID", "令牌无效或已过期")
	registerAuthRoutes(env)
	// When
	w := doRequest(t, env.engine, http.MethodPost, "/api/v1/auth/refresh", `{"refresh_token":"rt-stale"}`)
	// Then
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("HTTP 状态 = %d, 期望 %d，body=%s", w.Code, http.StatusUnauthorized, w.Body.String())
	}
	resp := decodeEnvelope(t, w)
	if resp.Code != http.StatusUnauthorized || resp.Msg != "令牌无效或已过期" {
		t.Errorf("信封 = {code:%d msg:%q}, 期望 {code:401 msg:令牌无效或已过期}", resp.Code, resp.Msg)
	}
}

// TestAuth_Logout_Success：Given 携带 refresh_token 的请求体；When 转发；
// Then fake 收到 refresh_token、空回复 data 为 {}。
func TestAuth_Logout_Success(t *testing.T) {
	// Given
	env := newTestEnv(t)
	st := bindAuthFakeState(t, env.fakeAuth)
	registerAuthRoutes(env)
	// When
	w := doRequest(t, env.engine, http.MethodPost, "/api/v1/auth/logout", `{"refresh_token":"rt-logout-9"}`)
	// Then
	if w.Code != http.StatusOK {
		t.Fatalf("HTTP 状态 = %d, 期望 %d，body=%s", w.Code, http.StatusOK, w.Body.String())
	}
	snap, ok := st.lastLogout()
	if !ok {
		t.Fatal("fake auth 未收到 Logout 调用")
	}
	if snap.refreshToken != "rt-logout-9" {
		t.Errorf("fake 收到 refresh_token=%q, 期望 rt-logout-9", snap.refreshToken)
	}
	if raw := string(decodeEnvelope(t, w).Data); raw != "{}" {
		t.Errorf("data = %s, 期望 {}（空回复序列化结果）", raw)
	}
}

// TestAuth_GetProfile_NoBodyAndMeta：Given 认证请求（uid=42）GET 无请求体；
// When 经 GetProfile handler 转发；Then 200、data 为 user 信息（snake_case）、
// 认证元数据与真实 IP 到达 server 端——GET 无 body 不应影响绑定与转发。
func TestAuth_GetProfile_NoBodyAndMeta(t *testing.T) {
	// Given
	env := newTestEnv(t)
	registerAuthRoutes(env)
	// When：无请求体（GET 语义）
	w := doRequest(t, env.engine, http.MethodGet, "/api/v1/users/me", "")
	// Then
	if w.Code != http.StatusOK {
		t.Fatalf("HTTP 状态 = %d, 期望 %d，body=%s", w.Code, http.StatusOK, w.Body.String())
	}
	raw := string(decodeEnvelope(t, w).Data)
	// protojson 将 uint64/uint64 一律序列化为 JSON 字符串（防精度丢失），id 断言用引号包裹
	for _, want := range []string{`"id":"42"`, `"username":"tester"`, `"email":"t@example.com"`, `"role":"admin"`} {
		if !strings.Contains(raw, want) {
			t.Errorf("data 缺少片段 %s，data=%s", want, raw)
		}
	}
	snap, ok := env.fakeAuth.lastProfile()
	if !ok {
		t.Fatal("fake auth 未收到 GetProfile 调用")
	}
	if got := mdVal(t, snap.md, pkgmeta.AuthUserIDKey); got != "42" {
		t.Errorf("x-md-global-auth-user-id = %q, 期望 %q", got, "42")
	}
	if got := mdVal(t, snap.md, pkgmeta.AuthRealClientIpKey); got != "203.0.113.9" {
		t.Errorf("x-md-global-auth-real-ip = %q, 期望 %q", got, "203.0.113.9")
	}
}

// TestAuth_UpdateProfile_ForwardsAndMeta：Given 认证请求 PUT 携带 avatar/bio；
// When 经 UpdateProfile handler 转发；Then fake 收到 avatar/bio、
// 响应回显（snake_case avatar/bio）、认证元数据到达 server 端。
func TestAuth_UpdateProfile_ForwardsAndMeta(t *testing.T) {
	// Given
	env := newTestEnv(t)
	st := bindAuthFakeState(t, env.fakeAuth)
	registerAuthRoutes(env)
	// When
	w := doRequest(t, env.engine, http.MethodPut, "/api/v1/users/me",
		`{"avatar":"https://cdn.example.com/a.png","bio":"你好，世界"}`)
	// Then
	if w.Code != http.StatusOK {
		t.Fatalf("HTTP 状态 = %d, 期望 %d，body=%s", w.Code, http.StatusOK, w.Body.String())
	}
	snap, ok := st.lastUpdate()
	if !ok {
		t.Fatal("fake auth 未收到 UpdateProfile 调用")
	}
	if snap.avatar != "https://cdn.example.com/a.png" || snap.bio != "你好，世界" {
		t.Errorf("fake 收到 avatar=%q bio=%q, 期望 https://cdn.example.com/a.png/你好，世界", snap.avatar, snap.bio)
	}
	raw := string(decodeEnvelope(t, w).Data)
	for _, want := range []string{`"avatar":"https://cdn.example.com/a.png"`, `"bio":"你好，世界"`, `"username":"tester"`} {
		if !strings.Contains(raw, want) {
			t.Errorf("data 缺少 snake_case 片段 %s，data=%s", want, raw)
		}
	}
	if got := mdVal(t, snap.md, pkgmeta.AuthUserIDKey); got != "42" {
		t.Errorf("x-md-global-auth-user-id = %q, 期望 %q", got, "42")
	}
	if got := mdVal(t, snap.md, pkgmeta.AuthUserRoleKey); got != "admin" {
		t.Errorf("x-md-global-auth-user-role = %q, 期望 %q", got, "admin")
	}
}

// TestAuth_UpdateProfile_BadJSON：Given 非法 JSON body；When 转发；
// Then 400 信封 + 参数解析失败，且不触发下游调用。
func TestAuth_UpdateProfile_BadJSON(t *testing.T) {
	// Given
	env := newTestEnv(t)
	st := bindAuthFakeState(t, env.fakeAuth)
	registerAuthRoutes(env)
	// When
	w := doRequest(t, env.engine, http.MethodPut, "/api/v1/users/me", `{"avatar":`)
	// Then
	if w.Code != http.StatusBadRequest {
		t.Fatalf("HTTP 状态 = %d, 期望 %d，body=%s", w.Code, http.StatusBadRequest, w.Body.String())
	}
	resp := decodeEnvelope(t, w)
	if resp.Code != http.StatusBadRequest || resp.Msg != "参数解析失败" {
		t.Errorf("信封 = {code:%d msg:%q}, 期望 {code:400 msg:参数解析失败}", resp.Code, resp.Msg)
	}
	if _, ok := st.lastUpdate(); ok {
		t.Error("非法 JSON 不应触发下游 UpdateProfile 调用")
	}
}

// 编译期断言：AuthHandler 方法集即 6 个 gin handler（防签名漂移）。
var _ = []gin.HandlerFunc{
	(&AuthHandler{}).Register, (&AuthHandler{}).Login, (&AuthHandler{}).RefreshToken,
	(&AuthHandler{}).Logout, (&AuthHandler{}).GetProfile, (&AuthHandler{}).UpdateProfile,
}
