package server

import (
	"strings"

	blogv1 "ley/api/blog/v1"
	"ley/app/blog/internal/service"
	leyconf "ley/conf"
	pkgmw "ley/pkg/middleware"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/transport/grpc"
)

func NewGRPCServer(bs *leyconf.Bootstrap, articleSvc *service.ArticleService, tagSvc *service.TagService, catSvc *service.CategoryService, fileSvc *service.FileService, logger log.Logger) *grpc.Server {
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

	blogv1.RegisterArticleServiceServer(srv, articleSvc)
	blogv1.RegisterTagServiceServer(srv, tagSvc)
	blogv1.RegisterCategoryServiceServer(srv, catSvc)
	blogv1.RegisterFileServiceServer(srv, fileSvc)

	return srv
}
