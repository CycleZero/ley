package router

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/CycleZero/ley/app/entry/conf"
	"github.com/CycleZero/ley/app/entry/internal/router/middleware"
	jwtpkg "github.com/CycleZero/ley/pkg/jwt"
	"github.com/CycleZero/ley/pkg/log"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// TestMain 初始化全局日志器为丢弃日志（对齐 conf/infra 测试约定）并切 gin 测试模式。
func TestMain(m *testing.M) {
	log.SetGlobalLogger(&log.Logger{Logger: zap.NewNop()})
	gin.SetMode(gin.TestMode)
	os.Exit(m.Run())
}

const testSigningKey = "router-provider-test-secret-0123456789abcdef"

// TestRegisterMountsGlobalMiddlewares 验证 RegisteredMiddleWire.Register() 的真实注册：
//  1. 发布 JWT 认证工厂（middleware.AuthMiddleWire）；
//  2. 按默认远程配置（rps=0 无限流）组装 3 个全局中间件；
//  3. 对齐 app.go 的组装顺序（Recovery → Register → RegisterRouter）走完整注册路径后，
//     全局链真实生效：健康检查带 RequestID 头与 CORS 头，404 兜底也走全局链。
func TestRegisterMountsGlobalMiddlewares(t *testing.T) {
	// 清理包级状态，避免污染同包其他用例
	defer func() {
		IsMiddleWireRegisterFinished = false
		globalMiddleWires = nil
		middleware.AuthMiddleWire = nil
	}()

	holder, stop := conf.NewRemoteConfigHolder(nil) // etcd 为空 → 降级为默认配置 holder
	defer stop()

	jwt := jwtpkg.NewJWT(&jwtpkg.Config{
		SigningKey:  testSigningKey,
		Issuer:      "router-provider-test",
		ExpiredTime: 15 * time.Minute,
	})
	reg := NewRegisterMiddleWire(jwt, jwtpkg.NewBlackList(nil), holder)
	reg.Register()

	if !IsMiddleWireRegisterFinished {
		t.Fatalf("Register() 应置位 IsMiddleWireRegisterFinished")
	}
	if middleware.AuthMiddleWire == nil {
		t.Fatalf("Register() 应发布 middleware.AuthMiddleWire（供路由组挂载）")
	}
	if len(globalMiddleWires) != 3 {
		t.Fatalf("默认配置（无限流）应组装 3 个全局中间件，实际 %d", len(globalMiddleWires))
	}

	// 对齐 app.go 组装顺序：Recovery → Register() → RegisterRouter(e, hub)
	e := gin.New()
	e.Use(gin.CustomRecovery(func(c *gin.Context, err any) {
		c.AbortWithStatusJSON(http.StatusInternalServerError,
			gin.H{"code": http.StatusInternalServerError, "msg": "服务器内部错误", "data": nil})
	}))
	// 本 wave 路由不含代理 handler，serviceHub 未被解引用，传 nil 即可
	RegisterRouter(e, nil)

	// 健康检查：全局链生效（RequestID + 默认 CORS *）
	rec := perform(e, "/healthz")
	if rec.Code != http.StatusOK {
		t.Fatalf("healthz 应返回 200，实际 %d", rec.Code)
	}
	if rec.Header().Get("X-Request-ID") == "" {
		t.Fatalf("全局链应注入 X-Request-ID 响应头")
	}
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "*" {
		t.Fatalf("默认 CORS 应放行所有来源，实际 %q", got)
	}

	// 404 兜底同样经过全局链：信封 + RequestID 头
	rec = perform(e, "/not-exist")
	if rec.Code != http.StatusNotFound {
		t.Fatalf("未匹配路由应返回 404，实际 %d", rec.Code)
	}
	if rec.Header().Get("X-Request-ID") == "" {
		t.Fatalf("404 兜底也应带 X-Request-ID 头")
	}
}

// perform 向引擎发起 GET 请求。
func perform(e *gin.Engine, path string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, path, nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	return rec
}
