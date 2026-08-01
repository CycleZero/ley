package jwt

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/CycleZero/ley/pkg/meta"
	"github.com/CycleZero/ley/pkg/testutil/datatest"
	"github.com/go-kratos/kratos/v2/transport"
)

// =============================================================================
// 测试辅助：fake transporter（模拟 Kratos transport 上下文）
// =============================================================================

type fakeHeader struct {
	h map[string]string
}

func (f *fakeHeader) Get(key string) string { return f.h[key] }
func (f *fakeHeader) Set(key, value string) { f.h[key] = value }
func (f *fakeHeader) Add(key, value string) { f.h[key] = value }
func (f *fakeHeader) Values(key string) []string {
	if v, ok := f.h[key]; ok {
		return []string{v}
	}
	return nil
}
func (f *fakeHeader) Keys() []string {
	keys := make([]string, 0, len(f.h))
	for k := range f.h {
		keys = append(keys, k)
	}
	return keys
}

type fakeTransporter struct {
	kind      transport.Kind
	endpoint  string
	operation string
	reqH      transport.Header
	replyH    transport.Header
}

func (t *fakeTransporter) Kind() transport.Kind        { return t.kind }
func (t *fakeTransporter) Endpoint() string            { return t.endpoint }
func (t *fakeTransporter) Operation() string           { return t.operation }
func (t *fakeTransporter) RequestHeader() transport.Header { return t.reqH }
func (t *fakeTransporter) ReplyHeader() transport.Header   { return t.replyH }

func newTestJWT(ttl time.Duration) JWT {
	return NewJWT(&Config{
		SigningKey:  "test-signing-key-0123456789-256bit-random",
		ExpiredTime: ttl,
		Issuer:      "test-issuer",
	})
}

// =============================================================================
// 生成与解析
// =============================================================================

func TestGenerateTokenAndParse(t *testing.T) {
	j := newTestJWT(time.Hour)
	token, err := j.GenerateToken(Payload{UserId: 42, UserName: "tester"})
	if err != nil {
		t.Fatalf("GenerateToken 失败: %v", err)
	}

	claims, err := j.ParseToken(token)
	if err != nil {
		t.Fatalf("ParseToken 失败: %v", err)
	}
	if claims.UserId != 42 {
		t.Errorf("UserId 不匹配: got %d want 42", claims.UserId)
	}
	if claims.UserName != "tester" {
		t.Errorf("UserName 不匹配: got %q want tester", claims.UserName)
	}
	if claims.TokenType != TokenTypeAccess {
		t.Errorf("TokenType 不匹配: got %q want access", claims.TokenType)
	}
	if claims.Issuer != "test-issuer" {
		t.Errorf("Issuer 不匹配: got %q", claims.Issuer)
	}
	if !claims.ExpiresAt.After(time.Now()) {
		t.Error("ExpiresAt 应在未来")
	}
}

func TestGenerateTokenPair(t *testing.T) {
	j := newTestJWT(time.Minute)
	pair, err := j.GenerateTokenPair(Payload{UserId: 7, UserName: "reader"})
	if err != nil {
		t.Fatalf("GenerateTokenPair 失败: %v", err)
	}
	if pair.AccessToken == "" || pair.RefreshToken == "" {
		t.Fatal("token 对不应为空")
	}
	if pair.AccessToken == pair.RefreshToken {
		t.Error("access 与 refresh 不应相同")
	}

	// access token 用 ParseAccessToken 成功
	if _, err := j.ParseAccessToken(pair.AccessToken); err != nil {
		t.Errorf("ParseAccessToken(access) 失败: %v", err)
	}
	// refresh token 用 ParseRefreshToken 成功
	if _, err := j.ParseRefreshToken(pair.RefreshToken); err != nil {
		t.Errorf("ParseRefreshToken(refresh) 失败: %v", err)
	}
	// 类型互换必须失败（防用 refresh 冒充 access）
	if _, err := j.ParseAccessToken(pair.RefreshToken); err == nil {
		t.Error("ParseAccessToken(refresh) 应失败")
	}
	if _, err := j.ParseRefreshToken(pair.AccessToken); err == nil {
		t.Error("ParseRefreshToken(access) 应失败")
	}
}

// 回归测试：jti 唯一性
// NumericDate 序列化到秒精度，若不加 jti，同一秒内签发的 token 完全相同，
// 会导致 Token 轮换时新 token 被旧 token 的黑名单连带拉黑。
func TestGenerateTokenPairUniqueWithinSameSecond(t *testing.T) {
	j := newTestJWT(time.Minute)
	p1, err := j.GenerateTokenPair(Payload{UserId: 1, UserName: "x"})
	if err != nil {
		t.Fatalf("第一次生成失败: %v", err)
	}
	p2, err := j.GenerateTokenPair(Payload{UserId: 1, UserName: "x"})
	if err != nil {
		t.Fatalf("第二次生成失败: %v", err)
	}
	if p1.RefreshToken == p2.RefreshToken {
		t.Error("同一秒内两次签发的 refresh token 必须不同（jti 缺失）")
	}
	if p1.AccessToken == p2.AccessToken {
		t.Error("同一秒内两次签发的 access token 必须不同（jti 缺失）")
	}
}

func TestRefreshTokenTTLIsSevenTimesAccess(t *testing.T) {
	j := newTestJWT(time.Minute)
	pair, err := j.GenerateTokenPair(Payload{UserId: 1})
	if err != nil {
		t.Fatalf("GenerateTokenPair 失败: %v", err)
	}
	accessClaims, _ := j.ParseAccessToken(pair.AccessToken)
	refreshClaims, _ := j.ParseRefreshToken(pair.RefreshToken)

	accessTTL := accessClaims.ExpiresAt.Time.Sub(accessClaims.IssuedAt.Time)
	refreshTTL := refreshClaims.ExpiresAt.Time.Sub(refreshClaims.IssuedAt.Time)
	if refreshTTL != accessTTL*7 {
		t.Errorf("refresh TTL 应为 access 的 7 倍: access=%v refresh=%v", accessTTL, refreshTTL)
	}
}

func TestParseExpiredToken(t *testing.T) {
	j := newTestJWT(-time.Minute) // 过期时间在过去
	token, err := j.GenerateToken(Payload{UserId: 1})
	if err != nil {
		t.Fatalf("GenerateToken 失败: %v", err)
	}
	if _, err := j.ParseToken(token); err == nil {
		t.Error("过期 token 应解析失败")
	}
}

func TestParseWrongSigningKey(t *testing.T) {
	j1 := newTestJWT(time.Hour)
	j2 := NewJWT(&Config{SigningKey: "another-key", ExpiredTime: time.Hour})
	token, _ := j1.GenerateToken(Payload{UserId: 1})
	if _, err := j2.ParseToken(token); err == nil {
		t.Error("错误密钥签发的 token 应解析失败")
	}
}

func TestParseGarbageToken(t *testing.T) {
	j := newTestJWT(time.Hour)
	if _, err := j.ParseToken("not-a-jwt"); err == nil {
		t.Error("垃圾字符串应解析失败")
	}
	if _, err := j.ParseToken(""); err == nil {
		t.Error("空 token 应解析失败")
	}
}

func TestExtractToken(t *testing.T) {
	cases := []struct {
		header string
		want   string
	}{
		{"Bearer abc123", "abc123"},
		{"bearer abc123", "bearer abc123"}, // 大小写敏感，原样返回
		{"abc123", "abc123"},               // 无前缀原样返回
		{"", ""},
	}
	for _, c := range cases {
		if got := extractToken(c.header); got != c.want {
			t.Errorf("extractToken(%q) = %q, want %q", c.header, got, c.want)
		}
	}
}

// =============================================================================
// 黑名单
// =============================================================================

func TestBlackList(t *testing.T) {
	c := datatest.NewInMemoryCache()
	bl := NewBlackList(c)

	if !bl.IsEnabled() {
		t.Error("有缓存时黑名单应启用")
	}
	if bl.IsTokenBlackListed("fresh-token") {
		t.Error("未加入黑名单的 token 不应命中")
	}
	if err := bl.Add("token-1"); err != nil {
		t.Fatalf("Add 失败: %v", err)
	}
	if !bl.IsTokenBlackListed("token-1") {
		t.Error("已加入黑名单的 token 应命中")
	}
}

func TestBlackListDisabledWithNilCache(t *testing.T) {
	bl := NewBlackList(nil)
	if bl.IsEnabled() {
		t.Error("nil 缓存时黑名单应禁用")
	}
	if bl.IsTokenBlackListed("any") {
		t.Error("禁用时不应命中")
	}
	if err := bl.Add("any"); err == nil {
		t.Error("nil 缓存时 Add 应报错")
	}
}

// =============================================================================
// Server 中间件：解析 Bearer token 并注入用户上下文
// =============================================================================

func TestServerMiddlewareInjectsUserMeta(t *testing.T) {
	j := newTestJWT(time.Hour)
	pair, err := j.GenerateTokenPair(Payload{UserId: 42, UserName: "tester"})
	if err != nil {
		t.Fatalf("GenerateTokenPair 失败: %v", err)
	}

	header := &fakeHeader{h: map[string]string{"Authorization": "Bearer " + pair.AccessToken}}
	tr := &fakeTransporter{kind: transport.KindHTTP, reqH: header, replyH: &fakeHeader{h: map[string]string{}}}
	ctx := transport.NewServerContext(context.Background(), tr)

	var gotMeta *meta.RequestMetaData
	handler := j.Server()(func(ctx context.Context, req interface{}) (interface{}, error) {
		gotMeta = meta.GetRequestMetaData(ctx)
		return "ok", nil
	})

	resp, err := handler(ctx, nil)
	if err != nil {
		t.Fatalf("handler 失败: %v", err)
	}
	if resp != "ok" {
		t.Errorf("resp = %v", resp)
	}
	if gotMeta == nil || gotMeta.Auth.UserID != 42 {
		t.Errorf("未注入用户上下文: %+v", gotMeta)
	}
	if gotMeta.Auth.UserName != "tester" {
		t.Errorf("UserName 不匹配: %q", gotMeta.Auth.UserName)
	}
}

func TestServerMiddlewareIgnoresInvalidToken(t *testing.T) {
	j := newTestJWT(time.Hour)

	// 无效 token：不阻断请求，但不注入用户
	header := &fakeHeader{h: map[string]string{"Authorization": "Bearer invalid-token"}}
	tr := &fakeTransporter{kind: transport.KindHTTP, reqH: header, replyH: &fakeHeader{h: map[string]string{}}}
	ctx := transport.NewServerContext(context.Background(), tr)

	var gotMeta *meta.RequestMetaData
	handler := j.Server()(func(ctx context.Context, req interface{}) (interface{}, error) {
		gotMeta = meta.GetRequestMetaData(ctx)
		return "ok", nil
	})
	if _, err := handler(ctx, nil); err != nil {
		t.Fatalf("无效 token 不应阻断请求: %v", err)
	}
	if gotMeta.Auth.UserID != 0 {
		t.Errorf("无效 token 不应注入用户: %+v", gotMeta)
	}
}

func TestServerMiddlewareWithoutAuthHeader(t *testing.T) {
	j := newTestJWT(time.Hour)
	header := &fakeHeader{h: map[string]string{}}
	tr := &fakeTransporter{kind: transport.KindHTTP, reqH: header, replyH: &fakeHeader{h: map[string]string{}}}
	ctx := transport.NewServerContext(context.Background(), tr)

	handler := j.Server()(func(ctx context.Context, req interface{}) (interface{}, error) {
		return "ok", nil
	})
	if _, err := handler(ctx, nil); err != nil {
		t.Fatalf("无 Authorization 头不应阻断: %v", err)
	}
}

// =============================================================================
// Client 中间件：将用户上下文透传到下游 metadata
// =============================================================================

func TestClientMiddlewarePropagatesMeta(t *testing.T) {
	j := newTestJWT(time.Hour)

	// 构造带用户 meta 的客户端上下文
	reqMeta := &meta.RequestMetaData{Auth: meta.Auth{UserID: 88, UserName: "alice"}}
	ctx := meta.NewClientCtx(context.Background(), reqMeta)

	var gotUserID uint64
	handler := j.Client()(func(ctx context.Context, req interface{}) (interface{}, error) {
		md := meta.GetClientMeta(ctx)
		gotUserID = md.Auth.UserID
		return nil, nil
	})
	if _, err := handler(ctx, nil); err != nil {
		t.Fatalf("handler 失败: %v", err)
	}
	if gotUserID != 88 {
		t.Errorf("下游未收到用户 ID: got %d want 88", gotUserID)
	}
}

func TestClientMiddlewareWithoutMeta(t *testing.T) {
	j := newTestJWT(time.Hour)
	handler := j.Client()(func(ctx context.Context, req interface{}) (interface{}, error) {
		return nil, nil
	})
	if _, err := handler(context.Background(), nil); err != nil {
		t.Fatalf("无 meta 时不应报错: %v", err)
	}
}

// =============================================================================
// 黑名单 token 全程联动：refresh 后旧 token 失效
// =============================================================================

func TestRefreshTokenRotationWithBlacklist(t *testing.T) {
	j := newTestJWT(time.Hour)
	pair, err := j.GenerateTokenPair(Payload{UserId: 1, UserName: "tester"})
	if err != nil {
		t.Fatalf("GenerateTokenPair 失败: %v", err)
	}

	bl := NewBlackList(datatest.NewInMemoryCache())

	// 旧 refresh 未被拉黑
	if bl.IsTokenBlackListed(pair.RefreshToken) {
		t.Fatal("旧 refresh 初始不应被拉黑")
	}
	// 模拟 RefreshToken 流程：旧 refresh 入黑名单
	if err := bl.Add(pair.RefreshToken); err != nil {
		t.Fatalf("Add 失败: %v", err)
	}
	if !bl.IsTokenBlackListed(pair.RefreshToken) {
		t.Error("旧 refresh 应已被拉黑（防重放）")
	}
	// 旧 access 也入黑名单（登出场景）
	if err := bl.Add(pair.AccessToken); err != nil {
		t.Fatalf("Add 失败: %v", err)
	}
	if !bl.IsTokenBlackListed(pair.AccessToken) {
		t.Error("旧 access 应已被拉黑")
	}
}

// 防御：黑名单 key 前缀必须固定，防止 key 冲突
func TestBlacklistKeyPrefixStable(t *testing.T) {
	if blacklistKeyPrefix != "jwt:blacklist:" {
		t.Errorf("黑名单前缀被意外修改: %q", blacklistKeyPrefix)
	}
	if !strings.HasPrefix(blacklistKeyPrefix, "jwt:") {
		t.Error("黑名单前缀应以 jwt: 开头")
	}
}

var _ = errors.Is // 保留 errors 引用（供后续扩展）
