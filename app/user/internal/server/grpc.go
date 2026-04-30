package server

import (
	"strings"
	"time"

	leyconf "ley/conf"
	userv1 "ley/api/user/v1"
	"ley/app/user/internal/service"
	pkgmiddleware "ley/pkg/middleware"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/transport/grpc"
)

// NewGRPCServer 创建 gRPC 服务器
//
// 参数:
//   bs     - 根 Bootstrap 配置，包含 Server.GRPC.Addr / .Timeout
//   svc    - 用户服务 HTTP/gRPC Handler 实现
//   logger - Kratos 日志接口，由 main 函数通过 pkg/log.GetKratosLogger() 初始化后注入
//
// 中间件链（按执行顺序）：
//   Recovery(捕获panic) → Tracing(OpenTelemetry Span) → Logging(请求日志) → Metadata(元数据透传)
func NewGRPCServer(bs *leyconf.Bootstrap, svc *service.UserService, logger log.Logger) *grpc.Server {
	var opts []grpc.ServerOption

	// ---- 网络配置 ----
	if bs.Server != nil && bs.Server.Grpc != nil {
		if addr := strings.TrimSpace(bs.Server.Grpc.Addr); addr != "" {
			opts = append(opts, grpc.Address(addr))
		}
		if bs.Server.Grpc.Timeout != nil {
			timeout := time.Duration(0) // 显式类型标注
			timeout = bs.Server.Grpc.Timeout.AsDuration()
			if timeout > 0 {
				opts = append(opts, grpc.Timeout(timeout))
			}
		}
	}

	// ---- 组装通用中间件链 ----
	// pkg/middleware.CommonServerMiddlewares 返回:
	//   [recovery.Recovery, tracing.Server, logging.Server, metadata.Server]
	opts = append(opts, grpc.Middleware(
		pkgmiddleware.CommonServerMiddlewares(logger)...,
	))

	// ---- 创建并注册 ----
	srv := grpc.NewServer(opts...)
	userv1.RegisterUserServiceServer(srv, svc)

	return srv
}
