// request_logger.go —— HTTP 请求日志中间件。
//
// 分级纪律（AGENTS.md「日志分级纪律」+「热路径禁用 Error」）：
//   - 2xx/3xx → Info（正常业务路径）；
//   - 4xx    → Warn（客户端问题/业务拒绝，可恢复，不构成服务故障）；
//   - 5xx    → Error（仅真故障）。
//
// 每请求的「开始」行走 Debug（细节默认关闭），避免正常流量刷屏。
//
// 本文件同时承载 mwLogger()：本包所有中间件共用的日志器获取函数
// （对齐 conf/infra 的降级模式——全局日志器未初始化时丢弃日志，防止单测 panic）。
package middleware

import (
	"time"

	"github.com/CycleZero/ley/pkg/log"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// mwLogger 返回全局日志器；尚未初始化（单测/启动早期）时降级为丢弃日志，
// 避免 nil *zap.Logger 方法调用 panic。生产路径 main.go 在 Wire 之前已初始化。
func mwLogger() *log.Logger {
	l := log.GetLogger()
	if l != nil && l.Logger != nil {
		return l
	}
	return &log.Logger{Logger: zap.NewNop()}
}

// RequestLogger 请求日志中间件（全局链最前，保证任何请求都有日志兜底）。
//
// 记录字段：method/path/status/耗时（ms）/客户端 IP/RequestID（AddMetaData 注入）。
func RequestLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		path := c.Request.URL.Path
		if rawQuery := c.Request.URL.RawQuery; rawQuery != "" {
			path += "?" + rawQuery
		}
		// 开始行走 Debug：默认关闭的细节日志，避免正常流量刷屏
		ctxLogger(c).Debug("HTTP 请求开始",
			zap.String("method", c.Request.Method),
			zap.String("path", path),
			zap.String("ip", c.ClientIP()),
		)

		// 执行后续中间件与业务 handler
		c.Next()

		// 请求完成：按状态码分级落日志
		status := c.Writer.Status()
		fields := []zap.Field{
			zap.String("method", c.Request.Method),
			zap.String("path", path),
			zap.Int("status", status),
			zap.Int64("cost_ms", time.Since(start).Milliseconds()),
			zap.String("ip", c.ClientIP()),
		}
		// RequestID 由 AddMetaData 种入（全局链在本中间件之后），此时必然可读
		if requestID := c.GetString(requestIDKey); requestID != "" {
			fields = append(fields, zap.String("request_id", requestID))
		}

		switch {
		case status >= 500:
			// 仅真故障刷 Error（热路径纪律）
			ctxLogger(c).Error("HTTP 请求处理失败", fields...)
		case status >= 400:
			// 4xx：客户端问题或业务拒绝，Warn 即可
			ctxLogger(c).Warn("HTTP 请求被拒绝", fields...)
		default:
			ctxLogger(c).Info("HTTP 请求完成", fields...)
		}
	}
}

// ctxLogger 返回绑定当前请求上下文的日志器（自动注入 trace_id/span_id/user_id/role）。
// 全局日志器未初始化时回退为 Nop，避免单测 panic。
func ctxLogger(c *gin.Context) *log.Logger {
	return mwLogger().WithContext(c.Request.Context())
}
