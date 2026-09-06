package router

import (
	"net/http"
	"strings"

	"github.com/CycleZero/ley/app/entry/internal/common"
	"github.com/CycleZero/ley/app/entry/internal/domain/proxy"
	"github.com/CycleZero/ley/app/entry/internal/router/middleware"
	"github.com/gin-gonic/gin"
)

// RegisterFunc 路由注册函数类型。
//
// serviceHub 后续 wave 承载各代理 handler 后，业务路由在此统一挂载。
type RegisterFunc func(root *gin.Engine, serviceHub *proxy.ServiceHub)

// NewRegisterFunc 创建路由注册函数。
func NewRegisterFunc() RegisterFunc {
	return RegisterRouter
}

// RegisterRouter 注册全部路由：业务路由（39 条权威表）+ 健康检查 + 404/405 信封兜底。
//
// 必须先于本函数调用 RegisteredMiddleWire.Register()（见 app.go），
// 否则 panic——保证后续 wave 中间件在路由注册前完成挂载。
func RegisterRouter(root *gin.Engine, serviceHub *proxy.ServiceHub) {
	if !IsMiddleWireRegisterFinished {
		panic("中间件注册未完成：请在路由注册前调用 RegisteredMiddleWire.Register()")
	}

	// 挂载全局中间件链（Register() 已按远程业务配置组装，见 provider.go globalMiddleWires）。
	// gin 中 Use 追加的中间件先于本函数注册的路由/404 兜底执行（Recovery 由 app.go 最先挂载），
	// 请求链路顺序：Recovery → RequestLogger → AddMetaData(RequestID) → CORS → [RateLimit] → handler。
	for _, mw := range globalMiddleWires {
		root.Use(mw)
	}

	// ── 业务路由：按权威路由表（routes.go newRouteRules）分类注册 ──
	// 每个分类对应一个独立分组，组内挂载该分类的鉴权链（认证 + 可选 RBAC）。
	// 分组懒创建：仅当表中出现该类路由时才建组；组间注册顺序交错无碍
	// （gin 按 HTTP 方法分树，组中间件在路由注册时快照进该路由的 handler 链）。
	classGroups := make(map[RouteClass]*gin.RouterGroup)
	for _, rule := range newRouteRules(serviceHub) {
		g := classGroups[rule.Class]
		if g == nil {
			g = root.Group(apiV1BasePath, classMiddleWires(rule.Class)...)
			classGroups[rule.Class] = g
		}
		// 表内 Path 为完整路径（含 /api/v1 前缀），注册到分组需剥离公共前缀
		g.Handle(rule.Method, strings.TrimPrefix(rule.Path, apiV1BasePath), rule.Handler)
	}

	// 健康检查：供探活/负载均衡使用
	root.GET("/healthz", func(c *gin.Context) {
		common.OK(c, nil)
	})
	root.GET("/readyz", func(c *gin.Context) {
		// 本 wave 无依赖可检查；后续 wave 在此探测 etcd/下游 gRPC 就绪状态
		common.OK(c, nil)
	})

	// 未匹配路由 → 404 信封
	root.NoRoute(func(c *gin.Context) {
		common.JSON(c, http.StatusNotFound, http.StatusNotFound, "接口不存在", nil)
	})
	// 方法不允许 → 405 信封（需 Engine.HandleMethodNotAllowed = true 才会命中）
	root.NoMethod(func(c *gin.Context) {
		common.JSON(c, http.StatusMethodNotAllowed, http.StatusMethodNotAllowed, "请求方法不允许", nil)
	})
}

// classMiddleWires 返回路由分类对应的鉴权中间件链（声明顺序即执行顺序）。
//
// 分类 → 链映射与 routes.go 文件头注释一致，是分类语义的唯一实现点：
//
//	PUBLIC        → 可选认证（无 token 放行，有 token 注入身份）
//	AUTH          → 强制认证（缺失/无效 token 一律 401）
//	AUTHOR_OR_ADMIN / ADMIN → 强制认证 + 角色白名单（403 拦截越权）
func classMiddleWires(class RouteClass) []gin.HandlerFunc {
	switch class {
	case RouteClassPublic:
		return []gin.HandlerFunc{middleware.AuthMiddleWire(true)}
	case RouteClassAuth:
		return []gin.HandlerFunc{middleware.AuthMiddleWire(false)}
	case RouteClassAuthorOrAdmin:
		return []gin.HandlerFunc{middleware.AuthMiddleWire(false), middleware.RequireRole("author", "admin")}
	case RouteClassAdmin:
		return []gin.HandlerFunc{middleware.AuthMiddleWire(false), middleware.RequireRole("admin")}
	default:
		panic("未知路由分类：" + class.String())
	}
}
