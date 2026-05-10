package server

import (
	"encoding/json"
	"net/http"
	"strings"

	blogv1 "github.com/CycleZero/ley/api/blog/v1"
	"github.com/CycleZero/ley/app/blog/internal/service"
	leyconf "github.com/CycleZero/ley/conf"
	pkgmw "github.com/CycleZero/ley/pkg/middleware"

	"github.com/go-kratos/kratos/v2/log"
	khttp "github.com/go-kratos/kratos/v2/transport/http"
)

func NewHTTPServer(bs *leyconf.Bootstrap, articleSvc *service.ArticleService, tagSvc *service.TagService, catSvc *service.CategoryService, fileSvc *service.FileService, logger log.Logger) *khttp.Server {
	var opts []khttp.ServerOption
	if bs.Server != nil && bs.Server.Http != nil {
		if addr := strings.TrimSpace(bs.Server.Http.Addr); addr != "" {
			opts = append(opts, khttp.Address(addr))
		}
		if bs.Server.Http.Timeout != nil {
			if d := bs.Server.Http.Timeout.AsDuration(); d > 0 {
				opts = append(opts, khttp.Timeout(d))
			}
		}
	}
	opts = append(opts, khttp.Middleware(pkgmw.CommonServerMiddlewares(logger)...))
	srv := khttp.NewServer(opts...)

	blogv1.RegisterArticleServiceHTTPServer(srv, articleSvc)
	blogv1.RegisterTagServiceHTTPServer(srv, tagSvc)
	blogv1.RegisterCategoryServiceHTTPServer(srv, catSvc)
	blogv1.RegisterFileServiceHTTPServer(srv, fileSvc)

	srv.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	})
	srv.HandleFunc("/readyz", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "ready"})
	})

	return srv
}
