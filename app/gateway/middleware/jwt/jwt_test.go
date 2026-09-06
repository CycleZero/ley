package jwt

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	jwtv1 "github.com/CycleZero/ley/api/gateway/middleware/jwt/v1"
	jwtpkg "github.com/CycleZero/ley/pkg/jwt"
	"github.com/CycleZero/ley/pkg/meta"
	"github.com/CycleZero/ley/pkg/testutil/datatest"
)

// B-101: 网关在 ParseAccessToken 成功后须校验 access token 是否已被吊销（登出/轮换黑名单）
func TestIsRevoked(t *testing.T) {
	c := datatest.NewInMemoryCache()
	bl := jwtpkg.NewBlackList(c)
	if err := bl.Add("revoked-token"); err != nil {
		t.Fatalf("预置黑名单失败: %v", err)
	}

	if !isRevoked(bl, "revoked-token") {
		t.Error("已拉黑 token 应判定为吊销")
	}
	if isRevoked(bl, "live-token") {
		t.Error("未拉黑 token 不应判定为吊销")
	}
	if isRevoked(nil, "any-token") {
		t.Error("未配置黑名单（nil）时不应判定吊销")
	}
}

// stubRT 是 http.RoundTripper 的测试替身
type stubRT func(req *http.Request) (*http.Response, error)

func (f stubRT) RoundTrip(req *http.Request) (*http.Response, error) { return f(req) }

// B-103: 有效 token 命中后，网关须注入 x-md-global-auth-user-role 头供下游做 admin 判定
func TestBuildHandlerInjectsRoleHeader(t *testing.T) {
	const signingKey = "test-signing-key-0123456789-256bit-random"
	options := &jwtv1.JWT{
		SigningKey: signingKey,
		Issuer:     "test-issuer",
		Enabled:    true,
	}
	j := jwtpkg.NewJWT(&jwtpkg.Config{SigningKey: signingKey, ExpiredTime: time.Hour, Issuer: "test-issuer"})

	holder := &jwtHolder{}
	holder.set(&jwtInstance{jwt: j, config: options})

	handler := buildHandler(options, holder)
	next := stubRT(func(req *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusOK}, nil
	})
	rt := handler(next)

	pair, err := j.GenerateTokenPair(jwtpkg.Payload{UserId: 9, UserName: "boss", Role: "admin"})
	if err != nil {
		t.Fatalf("GenerateTokenPair 失败: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/users/me", nil)
	req.Header.Set("Authorization", "Bearer "+pair.AccessToken)

	resp, err := rt.RoundTrip(req)
	if err != nil {
		t.Fatalf("RoundTrip 失败: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("应放行: status=%d", resp.StatusCode)
	}
	if got := req.Header.Get(meta.AuthUserRoleKey); got != "admin" {
		t.Errorf("应注入角色头 %s=admin, got %q", meta.AuthUserRoleKey, got)
	}
	if got := req.Header.Get(meta.AuthUserIDKey); got != "9" {
		t.Errorf("用户ID头异常: %q", got)
	}
}

// B-304: 合成 401 响应须为统一信封格式（与 wrapresp 错误形态一致）
func TestUnauthorizedResponseEnvelope(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/users/me", nil)
	resp := newUnauthorizedResponse(req, "token has been revoked")
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("状态码异常: %d", resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("读 body 失败: %v", err)
	}
	expect := `{"code":401,"msg":"token has been revoked","data":null}`
	if string(body) != expect {
		t.Errorf("信封格式不符:\n got %s\nwant %s", body, expect)
	}
}

// B-307: 网关透传真实客户端 IP（XFF 首跳）
func TestBuildHandlerInjectsRealIP(t *testing.T) {
	const signingKey = "test-signing-key-0123456789-256bit-random"
	options := &jwtv1.JWT{SigningKey: signingKey, Issuer: "test-issuer", Enabled: true}
	j := jwtpkg.NewJWT(&jwtpkg.Config{SigningKey: signingKey, ExpiredTime: time.Hour, Issuer: "test-issuer"})

	holder := &jwtHolder{}
	holder.set(&jwtInstance{jwt: j, config: options})
	handler := buildHandler(options, holder)
	next := stubRT(func(req *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusOK}, nil
	})
	rt := handler(next)

	pair, _ := j.GenerateTokenPair(jwtpkg.Payload{UserId: 1, UserName: "x"})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/articles/1", nil)
	req.Header.Set("Authorization", "Bearer "+pair.AccessToken)
	req.Header.Set("X-Forwarded-For", "203.0.113.9, 10.0.0.1")
	if _, err := rt.RoundTrip(req); err != nil {
		t.Fatalf("RoundTrip 失败: %v", err)
	}
	if got := req.Header.Get(meta.AuthRealClientIpKey); got != "203.0.113.9" {
		t.Errorf("应透传 XFF 首跳 IP: got %q", got)
	}
}

// B-307: XFF 缺失时回退 RemoteAddr
func TestClientIPFromRequest(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	req.RemoteAddr = "198.51.100.7:45678"
	if got := clientIPFromRequest(req); got != "198.51.100.7" {
		t.Errorf("RemoteAddr 回退失败: got %q", got)
	}
}
