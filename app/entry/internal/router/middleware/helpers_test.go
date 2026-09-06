package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	jwtpkg "github.com/CycleZero/ley/pkg/jwt"
	"github.com/CycleZero/ley/pkg/log"
	"github.com/CycleZero/ley/pkg/testutil/datatest"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// TestMain 初始化全局日志器为丢弃日志（对齐 conf/infra 测试的既有约定：
// pkg/log.GetLogger 在未初始化时直接 panic）并切到 gin 测试模式。
func TestMain(m *testing.M) {
	log.SetGlobalLogger(&log.Logger{Logger: zap.NewNop()})
	gin.SetMode(gin.TestMode)
	os.Exit(m.Run())
}

// testSigningKey 测试签名密钥（>=32 字节，对齐生产的 256 位要求）。
const testSigningKey = "entry-middleware-test-secret-0123456789abcdef"

// newTestJWT 构造测试 JWT 解析器（含签发能力）。
//
//	exp：access token 有效期；传负数（如 -time.Minute）可签发已过期 token。
func newTestJWT(exp time.Duration) jwtpkg.JWT {
	return jwtpkg.NewJWT(&jwtpkg.Config{
		SigningKey:         testSigningKey,
		Issuer:             "entry-middleware-test",
		ExpiredTime:        exp,
		RefreshExpiredTime: exp * 7,
	})
}

// issueToken 用测试 JWT 签发指定身份的 access token。
func issueToken(t *testing.T, j jwtpkg.JWT, userID uint64, userName, role string) string {
	t.Helper()
	token, err := j.GenerateToken(jwtpkg.Payload{UserId: userID, UserName: userName, Role: role})
	if err != nil {
		t.Fatalf("签发测试令牌失败: %v", err)
	}
	return token
}

// enabledBlacklist 构造启用的黑名单检查器（datatest.InMemoryCache 作 cache.Cache
// 测试实现：黑名单的 Add 经其 Set 落内存，IsTokenBlackListed 经 Exists 命中）。
func enabledBlacklist(t *testing.T) jwtpkg.BlackListCache {
	t.Helper()
	bl := jwtpkg.NewBlackList(datatest.NewInMemoryCache())
	if !bl.IsEnabled() {
		t.Fatalf("测试前置失败：黑名单应处于启用状态")
	}
	return bl
}

// newEngine 构建挂载指定中间件的测试引擎，并注册 GET /ping 探测路由。
// 探测 handler 返回标准信封：{code:0, data:{echo:"pong"}}。
func newEngine(middlewares ...gin.HandlerFunc) *gin.Engine {
	e := gin.New()
	e.Use(middlewares...)
	e.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "ok", "data": gin.H{"echo": "pong"}})
	})
	return e
}

// perform 向引擎发起请求（可携带请求头）。
func perform(t *testing.T, e *gin.Engine, method, path string, headers map[string]string) *httptest.ResponseRecorder {
	t.Helper()
	return performReq(t, e, method, path, headers, "")
}

// performFrom 向引擎发起请求并指定来源地址（模拟不同客户端 IP，限流测试用）。
func performFrom(t *testing.T, e *gin.Engine, method, path, remoteAddr string) *httptest.ResponseRecorder {
	t.Helper()
	return performReq(t, e, method, path, nil, remoteAddr)
}

// performReq perform/performFrom 的共同实现。
func performReq(t *testing.T, e *gin.Engine, method, path string, headers map[string]string, remoteAddr string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, nil)
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	if remoteAddr != "" {
		req.RemoteAddr = remoteAddr
	}
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	return rec
}

// envelope 测试用信封结构（对齐 common.Response：{code,msg,data}）。
type envelope struct {
	Code int            `json:"code"`
	Msg  string         `json:"msg"`
	Data map[string]any `json:"data"`
}

// decodeEnvelope 解析信封 JSON（data 为 null 时得到 nil map，字段访问需判空）。
func decodeEnvelope(t *testing.T, rec *httptest.ResponseRecorder) envelope {
	t.Helper()
	var env envelope
	if err := json.Unmarshal(rec.Body.Bytes(), &env); err != nil {
		t.Fatalf("解析响应信封失败 (body=%s): %v", rec.Body.String(), err)
	}
	return env
}

// wantStatus 断言 HTTP 状态码。
func wantStatus(t *testing.T, rec *httptest.ResponseRecorder, status int) {
	t.Helper()
	if rec.Code != status {
		t.Fatalf("状态码不符：期望 %d，实际 %d（body=%s）", status, rec.Code, rec.Body.String())
	}
}

// wantFail 断言失败信封：HTTP 状态与业务码镜像一致（T3 约定 code 镜像 httpStatus），
// 且消息精确匹配。
func wantFail(t *testing.T, rec *httptest.ResponseRecorder, status int, msg string) {
	t.Helper()
	wantStatus(t, rec, status)
	env := decodeEnvelope(t, rec)
	if env.Code != status {
		t.Fatalf("业务码不符：期望 %d（镜像 HTTP 状态），实际 %d", status, env.Code)
	}
	if env.Msg != msg {
		t.Fatalf("错误消息不符：期望 %q，实际 %q", msg, env.Msg)
	}
	if env.Data != nil {
		t.Fatalf("失败信封 data 应为 null，实际 %v", env.Data)
	}
}
