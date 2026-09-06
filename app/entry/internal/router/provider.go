package router

import (
	"github.com/CycleZero/ley/app/entry/conf"
	"github.com/CycleZero/ley/app/entry/internal/router/middleware"
	jwtpkg "github.com/CycleZero/ley/pkg/jwt"
	"github.com/gin-gonic/gin"
	"github.com/google/wire"
	"golang.org/x/time/rate"
)

// RouterProviderSet 路由层 Wire ProviderSet。
var RouterProviderSet = wire.NewSet(
	NewRegisterFunc,
	NewRegisterMiddleWire,
	middleware.MiddleWireProviderSet,
)

// IsMiddleWireRegisterFinished 标记自定义中间件是否已完成注册。
//
// RegisterRouter 依赖此标记保证中间件先于路由注册（与 gin-template 惯例一致）。
var IsMiddleWireRegisterFinished = false

// globalMiddleWires 全局中间件链：由 RegisteredMiddleWire.Register() 依据远程业务
// 配置组装，RegisterRouter 在路由注册前统一挂载（root.Use）。
//
// 链路顺序（对齐 app.go 约定 Recovery → RequestLogger → RequestID → CORS → [限流]）：
// RequestLogger（含 Recovery 之后的第一个全局中间件，任何请求都有日志兜底）→
// AddMetaData（RequestID + 匿名种子 meta）→ CORS → [RateLimit（配置启用时）]。
var globalMiddleWires []gin.HandlerFunc

// RegisteredMiddleWire 已注册的中间件集合。
//
// 内部字段由 wire 注入依赖后构建（NewRegisterMiddleWire）；Register() 负责：
//  1. 发布 JWT 认证工厂到 middleware.AuthMiddleWire，供路由组按需挂载；
//  2. 组装全局中间件链（globalMiddleWires），由 RegisterRouter 挂到引擎。
type RegisteredMiddleWire struct {
	// jwtAuth JWT 认证中间件工厂（依赖解析器+黑名单，构建于启动期）
	jwtAuth func(optional bool) gin.HandlerFunc
	// holder 远程业务配置持有者：Register() 时读取 CORS/限流配置
	holder *conf.RemoteConfigHolder
}

// Register 完成中间件注册（必须在路由注册前调用，app.go NewMainApp 内保证调用顺序）：
//
//  1. 发布 JWT 认证工厂到 middleware.AuthMiddleWire——后续 wave 的路由组挂载：
//     强制认证 group.Use(middleware.AuthMiddleWire(false), middleware.RequireRole("admin"))；
//     可选认证 group.Use(middleware.AuthMiddleWire(true))；
//  2. 依据远程业务配置组装全局中间件链存入 globalMiddleWires（RegisterRouter 挂载）。
//
// 中间件于注册时刻固化配置快照（CORS 来源、限流参数、JWT 密钥均构建于启动期）：
// 远程配置热更后需重启服务生效——中间件在路由树上不可热替换，属已知约定。
func (r *RegisteredMiddleWire) Register() {
	middleware.AuthMiddleWire = r.jwtAuth

	// 组装全局链；限流可选：远程配置 ratelimit.rps > 0 才启用（缺省关闭）
	cfg := r.holder.Get()
	chain := []gin.HandlerFunc{
		middleware.RequestLogger(),
		middleware.AddMetaData(),
		middleware.CORS(cfg.CORS.AllowOrigins),
	}
	if cfg.RateLimit.RPS > 0 {
		chain = append(chain, middleware.NewRateLimiter(rate.Limit(cfg.RateLimit.RPS), cfg.RateLimit.Burst))
	}
	globalMiddleWires = chain

	IsMiddleWireRegisterFinished = true
}

// NewRegisterMiddleWire 创建中间件注册器。
//
//	jwt：JWT 解析器（middleware.NewJWT 依远程配置构建）；
//	blacklist：JWT 黑名单检查器（infra.NewBlacklistCache；未配置 Redis 时内部降级禁用）；
//	holder：远程业务配置持有者（CORS/限流参数来源）。
func NewRegisterMiddleWire(jwt jwtpkg.JWT, blacklist jwtpkg.BlackListCache, holder *conf.RemoteConfigHolder) RegisteredMiddleWire {
	return RegisteredMiddleWire{
		jwtAuth: middleware.NewJWTAuth(jwt, blacklist),
		holder:  holder,
	}
}
