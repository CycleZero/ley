package server

import (
	"strings"
	"time"

	leyconf "ley/conf"
	postv1 "ley/api/post/v1"
	"ley/app/post/internal/service"
	pkgmiddleware "ley/pkg/middleware"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/transport/http"
)

// NewHTTPServer 创建 HTTP 服务器并注册文章服务 REST 路由
func NewHTTPServer(bs *leyconf.Bootstrap, svc *service.PostService, logger log.Logger) *http.Server {
	var opts []http.ServerOption

	if bs.Server != nil && bs.Server.Http != nil {
		if addr := strings.TrimSpace(bs.Server.Http.Addr); addr != "" {
			opts = append(opts, http.Address(addr))
		}
		if bs.Server.Http.Timeout != nil {
			timeout := time.Duration(0)
			timeout = bs.Server.Http.Timeout.AsDuration()
			if timeout > 0 {
				opts = append(opts, http.Timeout(timeout))
			}
		}
	}

	opts = append(opts, http.Middleware(
		pkgmiddleware.CommonServerMiddlewares(logger)...,
	))

	srv := http.NewServer(opts...)
	postv1.RegisterPostServiceHTTPServer(srv, svc)

	return srv
}
