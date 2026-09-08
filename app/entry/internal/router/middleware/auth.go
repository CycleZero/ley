// auth.go —— JWT 认证中间件（核心）。
//
// 职责：从 Authorization 头提取 Bearer token → 黑名单优先检查（吊销令牌不得放行）
// → 解析校验 → 将用户身份注入 gin 上下文（c.Set + common.SetRequestMeta），
// 供后续 RBAC 中间件与代理 handler 读取。
package middleware

import (
	"net/http"
	"strings"

	"github.com/CycleZero/ley/app/entry/internal/common"
	jwtpkg "github.com/CycleZero/ley/pkg/jwt"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// bearerPrefix Authorization 头的 Bearer 令牌前缀（scheme 大小写不敏感，RFC 6750 §2.1）。
const bearerPrefix = "Bearer "

// gin.Context 键约定（跨中间件/handler 传递用户身份，本包内统一读写）：
// rbac.go 依赖 roleKey 做角色鉴权，代理 handler 可读 userIDKey 等。
const (
	userIDKey   = "user_id"
	userNameKey = "user_name"
	roleKey     = "role"
)

// AuthMiddleWire 全局发布的 JWT 认证中间件工厂。
//
// 依赖（JWT 解析器、黑名单）由 router 层经 wire 注入构建，在
// RegisteredMiddleWire.Register() 时赋值本变量；赋值前为 nil。
// 路由组按需挂载，示例：
//
//	强制认证：root.Group("/api/admin", middleware.AuthMiddleWire(false), middleware.RequireRole("admin"))
//	可选认证：root.Group("/api", middleware.AuthMiddleWire(true))   // 匿名可访问，登录后识别身份
var AuthMiddleWire func(optional bool) gin.HandlerFunc

// NewJWTAuth 构建 JWT 认证中间件工厂。
//
//	jwt：令牌解析器（pkg/jwt.JWT，wire 依远程业务配置注入）；
//	blacklist：吊销黑名单检查器（未配置 Redis 时 IsEnabled=false，检查自动跳过）。
//
// 返回的工厂按路由组需求选择认证强度：
//   - optional=true  ：可选认证——未携带/格式错误/解析失败时按匿名请求放行
//     （解析失败记 Warn，匿名请求属于正常业务形态，不构成故障）；
//   - optional=false ：强制认证——任一环节失败直接 401 信封并中断。
//
// 注意：吊销令牌命中黑名单时**无论 optional 与否一律 401**——token 曾有效却被
// 显式吊销（登出/轮换），属于明确的安全信号，不应按匿名请求放行。
func NewJWTAuth(jwt jwtpkg.JWT, blacklist jwtpkg.BlackListCache) func(optional bool) gin.HandlerFunc {
	// 防御：配置缺失时以空密钥构造（解析必然失败 → 401），避免 nil 接口调用 panic。
	// 正常接线下 wire 恒注入非 nil 实现（见 middleware/provider.go NewJWT）。
	if jwt == nil {
		jwt = jwtpkg.NewJWT(&jwtpkg.Config{})
	}

	return func(optional bool) gin.HandlerFunc {
		return func(c *gin.Context) {
			// ── 1. 提取 Bearer token ──
			token, errKind := extractBearerToken(c.GetHeader("Authorization"))
			switch errKind {
			case bearerNone:
				// 未携带任何认证信息
				if optional {
					c.Next()
					return
				}
				common.Fail(c, http.StatusUnauthorized, http.StatusUnauthorized, "未提供认证令牌")
				c.Abort()
				return
			case bearerMalformed:
				// 携带了内容但缺少 Bearer 前缀（如裸 token / Basic 认证头）：格式非法
				if optional {
					mwLogger().Warn("认证头缺少 Bearer 前缀，按匿名请求放行",
						zap.String("method", c.Request.Method), zap.String("path", c.Request.URL.Path))
					c.Next()
					return
				}
				common.Fail(c, http.StatusUnauthorized, http.StatusUnauthorized, "无效的认证令牌")
				c.Abort()
				return
			}

			// ── 2. 黑名单优先检查（必须先于解析：防止已吊销 token 通过解析放行）──
			// blacklist 未启用（IsEnabled=false，未配置 Redis）时跳过，零额外开销
			if blacklist != nil && blacklist.IsEnabled() && blacklist.IsTokenBlackListed(token) {
				common.Fail(c, http.StatusUnauthorized, http.StatusUnauthorized, "令牌已吊销")
				c.Abort()
				return
			}

			// ── 3. 解析并校验（类型/签名/过期；RefreshToken 也会被拒）──
			claims, err := jwt.ParseAccessToken(token)
			if err != nil {
				if optional {
					mwLogger().Warn("解析认证令牌失败，按匿名请求放行",
						zap.String("method", c.Request.Method), zap.String("path", c.Request.URL.Path), zap.Error(err))
					c.Next()
					return
				}
				common.Fail(c, http.StatusUnauthorized, http.StatusUnauthorized, "无效的认证令牌")
				c.Abort()
				return
			}

			// ── 4. 注入用户身份 ──
			// Claims 实际字段：pkg/jwt.Claims 内嵌 Payload{UserId, UserName, Role}
			// （JSON 标签 user_id/user_name/role），嵌入字段直接提升访问
			c.Set(userIDKey, claims.UserId)
			c.Set(userNameKey, claims.UserName)
			c.Set(roleKey, claims.Role)
			// 种子内部请求元数据：RBAC 走 c.Get(roleKey)，代理 handler 走
			// common.GetRequestMeta → BuildRequestMeta 注入下游 gRPC（x-md-global-*）
			common.SetRequestMeta(c, &common.RequestMetaData{
				Auth: common.Auth{
					UserID:   claims.UserId,
					UserName: claims.UserName,
					Role:     claims.Role,
				},
				AccessToken: token,
			})
			c.Next()
		}
	}
}

// bearerErrKind 描述 Authorization 头的缺失形态（区分「未携带」与「格式非法」）。
type bearerErrKind int

const (
	bearerOK        bearerErrKind = iota // 提取成功
	bearerNone                           // 头缺失/为空：按「未提供」处理
	bearerMalformed                      // 有内容但无 Bearer 前缀：按「无效」处理
)

// extractBearerToken 校验并提取 Bearer token。
//
//	空头      → (_, bearerNone)，调用方回「未提供认证令牌」；
//	无前缀    → (_, bearerMalformed)，调用方回「无效的认证令牌」；
//	提取成功  → (token, bearerOK)，token 两侧空白已剔除。
func extractBearerToken(header string) (string, bearerErrKind) {
	header = strings.TrimSpace(header)
	if header == "" {
		return "", bearerNone
	}
	// scheme 大小写不敏感（RFC 6750 §2.1）
	if len(header) < len(bearerPrefix) || !strings.EqualFold(header[:len(bearerPrefix)], bearerPrefix) {
		return "", bearerMalformed
	}
	token := strings.TrimSpace(header[len(bearerPrefix):])
	if token == "" {
		// 形如 "Bearer "：前缀后无内容，等同未提供
		return "", bearerNone
	}
	return token, bearerOK
}
