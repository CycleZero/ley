// routes_test.go —— 权威路由表契约测试。
//
// 目标：routes.go 的 RouteRules 是单点真相——handler 方法新增/删除、路径改动、
// 分类调整都会让本文件变红。测试策略：
//
//  1. 注册表核对：RegisterRouter 走完整注册路径后，引擎 Routes() 中每条规则
//     恰有一条、总数精确（39 业务 + healthz/readyz），杜绝重复与漏注册；
//  2. 分类鉴权核对：按分类发真实请求做权限矩阵断言——中间件层拦截返回
//     401/403 精确信封；放行请求会抵达 handler（本测试 hub 为 nil，
//     实际调用会 panic，由 CustomRecovery 归一为 500/绑定 400）——因此
//     「非 401/403」即证明请求穿透鉴权链抵达业务层，分类装配正确；
//  3. 健康检查与 404 兜底信封存在。
package router

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/CycleZero/ley/app/entry/conf"
	"github.com/CycleZero/ley/app/entry/internal/common"
	"github.com/CycleZero/ley/app/entry/internal/router/middleware"
	jwtpkg "github.com/CycleZero/ley/pkg/jwt"

	"github.com/gin-gonic/gin"
)

// newRouteTestEngine 走真实组装路径构建引擎：
// gin.New → Recovery（panic 归一 500 信封）→ RegisteredMiddleWire.Register()
// （发布 AuthMiddleWire + 全局链）→ RegisterRouter(e, nil)。
//
// hub 传 nil：业务 handler 只被注册不被调用（构造仅存结构体指针，不解引用），
// 测试不依赖下游 gRPC。放行类请求到达 handler 后 panic → Recovery 500，
// 恰好作为「穿透鉴权链」的可观测信号（见文件头策略 2）。
func newRouteTestEngine(t *testing.T) *gin.Engine {
	t.Helper()
	// 清理包级状态，避免污染同包其他用例（对齐 provider_test.go 约定）
	t.Cleanup(func() {
		IsMiddleWireRegisterFinished = false
		globalMiddleWires = nil
		middleware.AuthMiddleWire = nil
	})

	holder, stop := conf.NewRemoteConfigHolder(nil) // etcd 为空 → 降级为默认配置 holder
	t.Cleanup(stop)

	jwt := jwtpkg.NewJWT(&jwtpkg.Config{
		SigningKey:  testSigningKey,
		Issuer:      "router-routes-test",
		ExpiredTime: 15 * time.Minute,
	})
	reg := NewRegisterMiddleWire(jwt, jwtpkg.NewBlackList(nil), holder)
	reg.Register()

	e := gin.New()
	e.Use(gin.CustomRecovery(func(c *gin.Context, err any) {
		// 静默恢复：hub 为 nil 时 handler 必然 panic，属预期信号，不刷日志
		c.AbortWithStatusJSON(http.StatusInternalServerError, common.Response{
			Code: http.StatusInternalServerError,
			Msg:  "服务器内部错误",
			Data: nil,
		})
	}))
	RegisterRouter(e, nil)
	return e
}

// routeRules 测试期望集：与注册共用同一构建函数（表即真相，两侧同源）。
func routeRules() []RouteRule {
	return newRouteRules(nil)
}

// performReq 发起任意方法请求；headers 携带 Bearer token 等。
func performReq(e *gin.Engine, method, path string, headers map[string]string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, nil)
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	return rec
}

// authHeader 构造 Authorization 头值。
func authHeader(token string) map[string]string {
	return map[string]string{"Authorization": "Bearer " + token}
}

// mintToken 用测试签名密钥签发指定角色的 access token（与引擎中间件同密钥）。
func mintToken(t *testing.T, role string) string {
	t.Helper()
	j := jwtpkg.NewJWT(&jwtpkg.Config{
		SigningKey:  testSigningKey,
		Issuer:      "router-routes-test",
		ExpiredTime: 15 * time.Minute,
	})
	token, err := j.GenerateToken(jwtpkg.Payload{UserId: 1, UserName: "tester", Role: role})
	if err != nil {
		t.Fatalf("签发测试令牌失败: %v", err)
	}
	return token
}

// TestRouteTableRegistered 契约 1：权威表 39 条规则全部注册且无重复。
//
// 引擎 Routes() 恰含 39 业务路由 + 2 健康检查；逐条核对方法与完整路径。
// 重复注册/表中漏注册/多余注册都会在此变红。
func TestRouteTableRegistered(t *testing.T) {
	e := newRouteTestEngine(t)

	rules := routeRules()
	if len(rules) != 39 {
		t.Fatalf("权威路由表应含 39 条规则，实际 %d（新增/删除需同步修订本表与文档）", len(rules))
	}

	// 建立注册表索引：method+path → 出现次数
	registered := make(map[string]int)
	for _, r := range e.Routes() {
		registered[r.Method+" "+r.Path]++
	}

	// 每条规则恰注册一次
	for _, rule := range rules {
		key := rule.Method + " " + rule.Path
		if n := registered[key]; n != 1 {
			t.Fatalf("规则 %s 应恰注册 1 次，实际 %d 次", key, n)
		}
	}

	// 无表外业务路由：总数精确（39 + /healthz + /readyz）
	if total := len(e.Routes()); total != len(rules)+2 {
		t.Fatalf("引擎路由总数应为 %d（39 业务 + 2 健康检查），实际 %d", len(rules)+2, total)
	}
}

// TestRouteClassAuthMatrix 契约 2：分类鉴权装配正确（权限矩阵）。
//
// 判定信号（hub=nil，见 newRouteTestEngine 注释）：
//   - 401/403 精确信封 → 被鉴权链拦截（分类装配生效，且拦截先于 handler 执行）；
//   - 其余状态码（400 绑定失败 / 500 handler panic）→ 请求穿透鉴权链抵达业务层，
//     证明该分类未多挂、未漏挂拦截中间件。
//
// 令牌：anonymous 无 token；author/user 角色令牌用与中间件同密钥签发。
// 覆盖策略：PUBLIC 全量匿名 + 抽样坏令牌；AUTH 全量匿名；AUTHOR_OR_ADMIN 全量
// 匿名/author/user 三角色；ADMIN 全量匿名/author/admin。
func TestRouteClassAuthMatrix(t *testing.T) {
	e := newRouteTestEngine(t)
	authorToken := mintToken(t, "author")
	userToken := mintToken(t, "user")
	adminToken := mintToken(t, "admin")

	rules := routeRules()
	for _, rule := range rules {
		key := rule.Method + " " + rule.Path
		switch rule.Class {
		case RouteClassPublic:
			// 匿名可访问：不得被 401/403 拦截
			if rec := performReq(e, rule.Method, rule.Path, nil); rec.Code == http.StatusUnauthorized || rec.Code == http.StatusForbidden {
				t.Fatalf("PUBLIC %s 匿名访问应放行，实际被拦截 %d", key, rec.Code)
			}
			// 坏令牌按匿名放行（可选认证语义）
			if rec := performReq(e, rule.Method, rule.Path, authHeader("garbage-token")); rec.Code == http.StatusUnauthorized || rec.Code == http.StatusForbidden {
				t.Fatalf("PUBLIC %s 坏令牌应按匿名放行，实际被拦截 %d", key, rec.Code)
			}
		case RouteClassAuth:
			// 无令牌 → 强制 401
			if rec := performReq(e, rule.Method, rule.Path, nil); rec.Code != http.StatusUnauthorized {
				t.Fatalf("AUTH %s 无令牌应 401，实际 %d", key, rec.Code)
			}
			// 任意有效令牌 → 放行至 handler（不校验角色）
			if rec := performReq(e, rule.Method, rule.Path, authHeader(userToken)); rec.Code == http.StatusUnauthorized || rec.Code == http.StatusForbidden {
				t.Fatalf("AUTH %s 有效令牌应放行，实际被拦截 %d", key, rec.Code)
			}
		case RouteClassAuthorOrAdmin:
			if rec := performReq(e, rule.Method, rule.Path, nil); rec.Code != http.StatusUnauthorized {
				t.Fatalf("AUTHOR_OR_ADMIN %s 无令牌应 401，实际 %d", key, rec.Code)
			}
			if rec := performReq(e, rule.Method, rule.Path, authHeader(userToken)); rec.Code != http.StatusForbidden {
				t.Fatalf("AUTHOR_OR_ADMIN %s user 角色应 403，实际 %d", key, rec.Code)
			}
			if rec := performReq(e, rule.Method, rule.Path, authHeader(authorToken)); rec.Code == http.StatusUnauthorized || rec.Code == http.StatusForbidden {
				t.Fatalf("AUTHOR_OR_ADMIN %s author 角色应放行，实际被拦截 %d", key, rec.Code)
			}
		case RouteClassAdmin:
			if rec := performReq(e, rule.Method, rule.Path, nil); rec.Code != http.StatusUnauthorized {
				t.Fatalf("ADMIN %s 无令牌应 401，实际 %d", key, rec.Code)
			}
			if rec := performReq(e, rule.Method, rule.Path, authHeader(authorToken)); rec.Code != http.StatusForbidden {
				t.Fatalf("ADMIN %s author 角色应 403，实际 %d", key, rec.Code)
			}
			if rec := performReq(e, rule.Method, rule.Path, authHeader(adminToken)); rec.Code == http.StatusUnauthorized || rec.Code == http.StatusForbidden {
				t.Fatalf("ADMIN %s admin 角色应放行，实际被拦截 %d", key, rec.Code)
			}
		default:
			t.Fatalf("路由 %s 分类 %d 未在矩阵覆盖范围内（分类常量被改动？）", key, rule.Class)
		}
	}
}

// TestAuthBlockedEnvelope 契约 2 补充：拦截响应为标准失败信封（code 镜像 HTTP 状态）。
func TestAuthBlockedEnvelope(t *testing.T) {
	e := newRouteTestEngine(t)

	// AUTH 无令牌 → 401 信封
	rec := performReq(e, http.MethodGet, apiV1BasePath+"/users/me", nil)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("users/me 无令牌应 401，实际 %d", rec.Code)
	}
	assertEnvelope(t, rec, http.StatusUnauthorized, "未提供认证令牌")

	// ADMIN 作者越权 → 403 信封
	authorToken := mintToken(t, "author")
	rec = performReq(e, http.MethodPut, apiV1BasePath+"/site/config", authHeader(authorToken))
	if rec.Code != http.StatusForbidden {
		t.Fatalf("site/config author 越权应 403，实际 %d", rec.Code)
	}
	assertEnvelope(t, rec, http.StatusForbidden, "无权访问")
}

// TestHealthzReadyzAndNotFound 契约 3：健康检查 200；NoRoute 返回 404 信封。
func TestHealthzReadyzAndNotFound(t *testing.T) {
	e := newRouteTestEngine(t)

	for _, path := range []string{"/healthz", "/readyz"} {
		if rec := performReq(e, http.MethodGet, path, nil); rec.Code != http.StatusOK {
			t.Fatalf("%s 应返回 200，实际 %d", path, rec.Code)
		}
	}

	rec := performReq(e, http.MethodGet, "/api/v1/not-exist", nil)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("未匹配路由应 404，实际 %d", rec.Code)
	}
	assertEnvelope(t, rec, http.StatusNotFound, "接口不存在")
}

// assertEnvelope 断言信封：HTTP 状态 == code（镜像约定）且消息精确匹配。
func assertEnvelope(t *testing.T, rec *httptest.ResponseRecorder, code int, msg string) {
	t.Helper()
	var env struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &env); err != nil {
		t.Fatalf("解析响应信封失败 (body=%s): %v", rec.Body.String(), err)
	}
	if env.Code != code {
		t.Fatalf("业务码应为 %d（镜像 HTTP 状态），实际 %d", code, env.Code)
	}
	if !strings.Contains(env.Msg, msg) {
		t.Fatalf("错误消息应包含 %q，实际 %q", msg, env.Msg)
	}
}

// TestRouteClassNames 契约 4：分类可读名与鉴权链语义一一对应（防 String() 漂移）。
func TestRouteClassNames(t *testing.T) {
	want := map[RouteClass]string{
		RouteClassPublic:        "PUBLIC",
		RouteClassAuth:          "AUTH",
		RouteClassAuthorOrAdmin: "AUTHOR_OR_ADMIN",
		RouteClassAdmin:         "ADMIN",
	}
	for class, name := range want {
		if got := class.String(); got != name {
			t.Fatalf("分类 %d 可读名应为 %q，实际 %q", class, name, got)
		}
	}
}
