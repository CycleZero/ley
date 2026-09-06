// rbac.go —— 角色鉴权中间件。
//
// 职责：按角色白名单拦截请求。**必须在 JWT 认证中间件之后执行**——
// 本中间件只读 gin 上下文里的 roleKey，该键由 auth.go 在令牌校验通过后注入；
// 若直接挂载于匿名请求（未走认证），将因缺少角色信息而一律 403。
package middleware

import (
	"net/http"

	"github.com/CycleZero/ley/app/entry/internal/common"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// RequireRole 构建角色鉴权中间件。
//
//	allowed：允许访问的角色白名单（如 RequireRole("admin") 或 RequireRole("author", "admin")）。
//
// 判定流程：
//  1. 读取 JWT 认证中间件注入的角色（c.Get(roleKey)）；
//  2. 缺失或类型错误（非 string）→ 403「无权访问：缺少角色信息」
//     （正常接线不会出现，多为认证中间件未先行执行或挂载顺序错误）；
//  3. 角色不在白名单 → 403「无权访问」；
//  4. 命中白名单 → 放行。
//
// 挂载示例（路由组内中间件按声明顺序执行：认证在前、鉴权在后）：
//
//	admin := root.Group("/api/admin", middleware.AuthMiddleWire(false), middleware.RequireRole("admin"))
func RequireRole(allowed ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		roleVal, exists := c.Get(roleKey)
		if !exists {
			// 缺少角色：通常是认证中间件未执行（例如匿名路由误挂鉴权）
			mwLogger().Warn("鉴权拒绝：缺少角色信息",
				zap.String("method", c.Request.Method), zap.String("path", c.Request.URL.Path))
			common.Fail(c, http.StatusForbidden, http.StatusForbidden, "无权访问：缺少角色信息")
			c.Abort()
			return
		}
		role, ok := roleVal.(string)
		if !ok {
			// 类型错误：认证中间件注入异常或角色被其他中间件污染
			mwLogger().Warn("鉴权拒绝：角色信息类型错误",
				zap.String("method", c.Request.Method), zap.String("path", c.Request.URL.Path))
			common.Fail(c, http.StatusForbidden, http.StatusForbidden, "无权访问：缺少角色信息")
			c.Abort()
			return
		}

		// 白名单匹配
		for _, allowedRole := range allowed {
			if role == allowedRole {
				c.Next()
				return
			}
		}

		// 角色合法但不在白名单：正常业务拒绝，记 Warn（4xx 属客户端问题，不刷 Error）
		mwLogger().Warn("鉴权拒绝：当前角色无访问权限",
			zap.String("method", c.Request.Method), zap.String("path", c.Request.URL.Path),
			zap.String("role", role), zap.Strings("allowed", allowed))
		common.Fail(c, http.StatusForbidden, http.StatusForbidden, "无权访问")
		c.Abort()
	}
}
