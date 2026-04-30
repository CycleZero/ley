package server

import (
	"strings"
	"time"

	leyconf "ley/conf"
	commentv1 "ley/api/comment/v1"
	"ley/app/comment/internal/service"
	pkgmiddleware "ley/pkg/middleware"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/transport/http"
)

// NewHTTPServer 创建 HTTP 服务器并注册评论服务 REST 路由
func NewHTTPServer(bs *leyconf.Bootstrap, svc *service.CommentService, logger log.Logger) *http.Server {
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
	commentv1.RegisterCommentServiceHTTPServer(srv, svc)

	return srv
}
