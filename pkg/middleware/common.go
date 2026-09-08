package middleware

import (
	"context"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/middleware"
	"github.com/go-kratos/kratos/v2/middleware/logging"
	"github.com/go-kratos/kratos/v2/middleware/metadata"
	"github.com/go-kratos/kratos/v2/middleware/recovery"
	"github.com/go-kratos/kratos/v2/middleware/tracing"

	"github.com/CycleZero/ley/pkg/metrics"
)

// CommonServerMiddlewares 返回 Kratos 服务端通用中间件链（auth/blog 共用）：
// recovery → tracing → metrics → logging → metadata。
// tracing 先于 metrics，保证指标携带链路上下文；logging 使用 ctx 感知的日志器，
// 自动带 trace_id/span_id（见 pkg/log.GetKratosLogger）。
func CommonServerMiddlewares(logger log.Logger) []middleware.Middleware {
	return []middleware.Middleware{
		recovery.Recovery(recovery.WithHandler(func(ctx context.Context, req, err any) error {
			return nil
		})),
		tracing.Server(),
		metrics.ServerMiddleware(),
		logging.Server(logger),
		metadata.Server(),
	}
}
