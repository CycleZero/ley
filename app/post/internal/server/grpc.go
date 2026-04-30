package server

import (
	"strings"
	"time"

	leyconf "ley/conf"
	postv1 "ley/api/post/v1"
	"ley/app/post/internal/service"
	pkgmiddleware "ley/pkg/middleware"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/transport/grpc"
)

// NewGRPCServer 创建 gRPC 服务器并注册文章+标签+分类服务
func NewGRPCServer(bs *leyconf.Bootstrap, svc *service.PostService, logger log.Logger) *grpc.Server {
	var opts []grpc.ServerOption

	if bs.Server != nil && bs.Server.Grpc != nil {
		if addr := strings.TrimSpace(bs.Server.Grpc.Addr); addr != "" {
			opts = append(opts, grpc.Address(addr))
		}
		if bs.Server.Grpc.Timeout != nil {
			timeout := time.Duration(0)
			timeout = bs.Server.Grpc.Timeout.AsDuration()
			if timeout > 0 {
				opts = append(opts, grpc.Timeout(timeout))
			}
		}
	}

	opts = append(opts, grpc.Middleware(
		pkgmiddleware.CommonServerMiddlewares(logger)...,
	))

	srv := grpc.NewServer(opts...)
	postv1.RegisterPostServiceServer(srv, svc)

	return srv
}
