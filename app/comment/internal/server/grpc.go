package server

import (
	"strings"
	"time"

	leyconf "ley/conf"
	commentv1 "ley/api/comment/v1"
	"ley/app/comment/internal/service"
	pkgmiddleware "ley/pkg/middleware"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/transport/grpc"
)

// NewGRPCServer 创建 gRPC 服务器并注册评论服务
func NewGRPCServer(bs *leyconf.Bootstrap, svc *service.CommentService, logger log.Logger) *grpc.Server {
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
	commentv1.RegisterCommentServiceServer(srv, svc)

	return srv
}
