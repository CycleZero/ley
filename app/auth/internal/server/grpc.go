package server

import (
	"strings"

	authv1 "github.com/CycleZero/ley/api/auth/v1"
	"github.com/CycleZero/ley/app/auth/internal/service"
	leyconf "github.com/CycleZero/ley/conf"
	pkgmw "github.com/CycleZero/ley/pkg/middleware"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/transport/grpc"
)

func NewGRPCServer(bs *leyconf.Bootstrap, svc *service.AuthService, logger log.Logger) *grpc.Server {
	var opts []grpc.ServerOption
	if bs.Server != nil && bs.Server.Grpc != nil {
		if addr := strings.TrimSpace(bs.Server.Grpc.Addr); addr != "" {
			opts = append(opts, grpc.Address(addr))
		}
		if bs.Server.Grpc.Timeout != nil {
			if d := bs.Server.Grpc.Timeout.AsDuration(); d > 0 {
				opts = append(opts, grpc.Timeout(d))
			}
		}
	}
	opts = append(opts, grpc.Middleware(pkgmw.CommonServerMiddlewares(logger)...))
	srv := grpc.NewServer(opts...)
	authv1.RegisterAuthServiceServer(srv, svc)
	return srv
}
