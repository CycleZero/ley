// Package data 文章服务数据访问层
// 提供数据库和缓存的统一访问入口，
// 封装 GORM、Redis 缓存以及 OpenTelemetry 链路追踪。
package data

import (
	"context"

	"ley/app/post/internal/biz"
	"ley/pkg/cache"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/wire"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"gorm.io/gorm"
)

// Data 文章服务数据层聚合结构
// 将数据库连接、缓存实例、日志器和 Tracer 聚合在一起，
// 供各个 Repo 实现（postRepo、tagRepo、categoryRepo）使用。
type Data struct {
	db     *gorm.DB     // PostgreSQL 数据库连接
	cache  cache.Cache  // Redis 缓存实例
	logger log.Logger   // Kratos 日志器
	tracer trace.Tracer // OpenTelemetry Tracer，用于分布式链路追踪
}

// NewData 创建 Data 实例
// 初始化 Data 并注册 OpenTelemetry Tracer，名称为 "post-service.data"。
func NewData(db *gorm.DB, c cache.Cache, logger log.Logger) *Data {
	return &Data{
		db:     db,
		cache:  c,
		logger: logger,
		tracer: otel.Tracer("post-service.data"),
	}
}

// Log 返回携带 SpanContext 的 Kratos log.Helper
// 将上下文中的 TraceID/SpanID 自动注入到日志输出中。
func (d *Data) Log(ctx context.Context) *log.Helper {
	return log.NewHelper(log.WithContext(ctx, d.logger))
}

// StartSpan 创建 OTel 子 Span
// 为每次数据库或缓存操作创建 Client 类型的 Span，
// 并附加 PostgreSQL 相关的属性标签。
func (d *Data) StartSpan(ctx context.Context, spanName string) (context.Context, trace.Span) {
	return d.tracer.Start(ctx, spanName,
		trace.WithSpanKind(trace.SpanKindClient), // Client 类型 Span，表示本服务作为调用方
		trace.WithAttributes(
			attribute.String("db.system", "postgresql"), // 数据库类型标识
			attribute.String("db.service", "post-service"), // 服务名称标识
		),
	)
}

// ProviderSet Wire 依赖注入集合
// 包含 Data 及 PostRepo/TagRepo/CategoryRepo 构造函数
var ProviderSet = wire.NewSet(NewData, NewPostRepo, NewTagRepo, NewCategoryRepo)

// NewPostRepo 创建 PostRepo 接口实现
// 返回 postRepo 实例，实现 biz.PostRepo 接口，用于文章数据访问。
func NewPostRepo(data *Data) biz.PostRepo { return &postRepo{data: data} }

// NewTagRepo 创建 TagRepo 接口实现
// 返回 tagRepo 实例，实现 biz.TagRepo 接口，用于标签数据访问。
func NewTagRepo(data *Data) biz.TagRepo { return &tagRepo{data: data} }

// NewCategoryRepo 创建 CategoryRepo 接口实现
// 返回 categoryRepo 实例，实现 biz.CategoryRepo 接口，用于分类数据访问。
func NewCategoryRepo(data *Data) biz.CategoryRepo { return &categoryRepo{data: data} }
