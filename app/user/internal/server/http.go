package server

import (
	"strings"
	"time"

	leyconf "ley/conf"
	userv1 "ley/api/user/v1"
	"ley/app/user/internal/service"
	pkgmiddleware "ley/pkg/middleware"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/transport/http"
)

// NewHTTPServer 创建 HTTP 服务器
//
// 参数:
//   bs     - 根 Bootstrap 配置，包含 Server.HTTP.Addr / .Timeout
//   svc    - 用户服务 HTTP/gRPC Handler 实现
//   logger - Kratos 日志接口
//
// HTTP 路由由 proto 文件中的 google.api.http 注解自动生成，
// 通过 RegisterUserServiceHTTPServer 注册到 HTTP mux 上。
func NewHTTPServer(bs *leyconf.Bootstrap, svc *service.UserService, logger log.Logger) *http.Server {
	var opts []http.ServerOption

	// ---- 网络配置 ----
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

	// ---- 组装通用中间件链 ----
	opts = append(opts, http.Middleware(
		pkgmiddleware.CommonServerMiddlewares(logger)...,
	))

	// ---- 创建并注册 ----
	srv := http.NewServer(opts...)
	userv1.RegisterUserServiceHTTPServer(srv, svc)

	return srv
}
