// Package proxy 是 entry 的代理域：每 API 一个 gin handler，handler 内部经
// gRPC client 调用 auth/blog，响应在 handler 层显式包统一信封 {code,msg,data}。
//
// 本包不承载领域业务逻辑（AGENTS.md「Entry 入口服务特别说明」）：业务规则与数据
// 访问全部留在 auth/blog 微服务，这里只做「入口 + 转发 + 聚合」。
package proxy

import (
	authv1 "github.com/CycleZero/ley/api/auth/v1"
	blogv1 "github.com/CycleZero/ley/api/blog/v1"
	"github.com/CycleZero/ley/app/entry/infra"
)

// ServiceHub 下游 gRPC 客户端聚合：持有 auth/blog 全部生成 client，
// 供各代理 handler 转发调用（hub 只做客户端装配，不含任何业务逻辑）。
type ServiceHub struct {
	Auth     authv1.AuthServiceClient     // 认证服务（register/login/refresh/logout/profile）
	Article  blogv1.ArticleServiceClient  // 博客文章服务
	Tag      blogv1.TagServiceClient      // 标签服务
	Category blogv1.CategoryServiceClient // 分类服务（树形）
	File     blogv1.FileServiceClient     // 文件服务（MinIO 直传/预签名）
	Site     blogv1.SiteServiceClient     // 站点配置服务
}

// NewProxyHub 基于 infra 层拨号的 auth/blog 连接构建代理服务聚合。
//
// authConn/blogConn 由 infra.InfraProviderSet 注入（命名包装类型 *infra.AuthClientConn /
// *infra.BlogClientConn 消除 wire 对同源 *grpc.ClientConn 的绑定歧义，
// 见 infra/grpc.go 命名包装说明）。底层连接已挂载 kratos metadata.Client() 中间件，
// handler 内经 pkg/meta.NewClientCtx 注入的用户上下文（x-md-global-*）会随调用透传下游。
func NewProxyHub(authConn *infra.AuthClientConn, blogConn *infra.BlogClientConn) *ServiceHub {
	initProxyMetrics()
	return &ServiceHub{
		Auth:     authv1.NewAuthServiceClient(authConn.Conn()),
		Article:  blogv1.NewArticleServiceClient(blogConn.Conn()),
		Tag:      blogv1.NewTagServiceClient(blogConn.Conn()),
		Category: blogv1.NewCategoryServiceClient(blogConn.Conn()),
		File:     blogv1.NewFileServiceClient(blogConn.Conn()),
		Site:     blogv1.NewSiteServiceClient(blogConn.Conn()),
	}
}
