// cors.go —— 跨域（CORS）中间件。
//
// 两种形态（由 allowOrigins 决定）：
//   - 空列表（默认）：放行所有来源，响应 Access-Control-Allow-Origin: *。
//     "*" 与携带凭据互斥（浏览器规范），故该形态不返回 Allow-Credentials 头；
//   - 显式来源列表：仅放行匹配 Origin 的请求——响应回显请求 Origin 供浏览器比对，
//     并附带 Allow-Credentials: true（前端 Token 存 Cookie，跨域需带凭据）。
//
// 预检（OPTIONS）请求处理完毕后直接 204 中止，不再进入后续中间件/handler。
package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// 静态允许头（与前端契约对齐）。
const (
	corsAllowMethods = "GET, POST, PUT, PATCH, DELETE, HEAD, OPTIONS"
	corsAllowHeaders = "Origin, X-Requested-With, Content-Type, Accept, Authorization"
	corsMaxAge       = "86400" // 预检结果浏览器缓存时长（秒），避免高频预检
)

// CORS 构建跨域中间件。
//
//	allowOrigins：允许的来源列表（如 ["http://localhost:3000"]）；空列表表示放行所有来源。
func CORS(allowOrigins []string) gin.HandlerFunc {
	origins := allowOrigins
	wildcard := len(origins) == 0

	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")

		// 计算本次响应的允许来源：不匹配来源时不返回任何 CORS 头（浏览器自行拦截）
		allowOrigin := ""
		switch {
		case wildcard:
			allowOrigin = "*"
		case origin != "" && contains(origins, origin):
			allowOrigin = origin
		}

		if allowOrigin != "" {
			c.Header("Access-Control-Allow-Origin", allowOrigin)
			if !wildcard {
				// 仅指定来源形态支持凭据："*" + Credentials 组合浏览器非法
				c.Header("Access-Control-Allow-Credentials", "true")
			}
			c.Header("Access-Control-Allow-Methods", corsAllowMethods)
			c.Header("Access-Control-Allow-Headers", corsAllowHeaders)
			c.Header("Access-Control-Max-Age", corsMaxAge)
		}

		if c.Request.Method == http.MethodOptions {
			// 预检请求：头已就绪，直接 204 结束（无需进入业务 handler）
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}

// contains 判断字符串是否在切片内（小切片线性扫描，无需引入 slices）。
func contains(list []string, target string) bool {
	for _, v := range list {
		if v == target {
			return true
		}
	}
	return false
}
