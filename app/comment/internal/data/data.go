package data

import (
	"context"

	"ley/app/comment/internal/biz"
	"ley/pkg/cache"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/wire"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"gorm.io/gorm"
)

// Data 评论服务数据层聚合结构
// 持有数据库连接、缓存客户端、日志器与分布式追踪实例
type Data struct {
	db     *gorm.DB      // GORM 数据库连接，用于操作评论相关的数据库表
	cache  cache.Cache   // 缓存客户端，提供 Get/Set/Delete 等缓存操作能力
	logger log.Logger    // Kratos 日志器，用于输出结构化日志
	tracer trace.Tracer  // OpenTelemetry Tracer，用于创建分布式追踪 Span
}

// NewData 创建 Data 实例
// db: GORM 数据库连接
// c: 缓存客户端
// logger: Kratos 日志器
// 返回初始化完成的 Data 指针，内部自动创建名为 "comment-service.data" 的 Tracer
func NewData(db *gorm.DB, c cache.Cache, logger log.Logger) *Data {
	d := &Data{
		db:     db,
		cache:  c,
		logger: logger,
		tracer: otel.Tracer("comment-service.data"), // 初始化 OpenTelemetry Tracer，服务名称为 comment-service.data
	}
	log.NewHelper(logger).Debug("Data 层实例创建成功")
	return d
}

// Log 返回携带 SpanContext 的 Kratos log.Helper
// 通过 log.WithContext 将 ctx 中的 Span 信息注入日志上下文，
// 使得输出的日志自动携带 trace_id 等链路追踪信息
func (d *Data) Log(ctx context.Context) *log.Helper {
	return log.NewHelper(log.WithContext(ctx, d.logger))
}

// StartSpan 创建 OTel 子 Span
// spanName: Span 名称，通常为操作名称如 "CommentRepo.Create"
// 返回嵌入了 Span 的新 ctx 和 Span 实例，调用方负责 defer span.End()
// 默认 SpanKind 为 Client，并附加 PostgreSQL 数据库相关属性
func (d *Data) StartSpan(ctx context.Context, spanName string) (context.Context, trace.Span) {
	return d.tracer.Start(ctx, spanName,
		trace.WithSpanKind(trace.SpanKindClient),        // 标记为 Client 类型 Span
		trace.WithAttributes(
			attribute.String("db.system", "postgresql"),      // 数据库类型：PostgreSQL
			attribute.String("db.service", "comment-service"), // 所属服务：评论服务
		),
	)
}

// ProviderSet Wire 依赖注入集合
// 包含 Data 及 CommentRepo 构造函数
var ProviderSet = wire.NewSet(NewData, NewCommentRepo)

// NewCommentRepo 创建 CommentRepo 接口实现
// 通过依赖注入将 Data 实例传入 commentRepo，返回 biz.CommentRepo 接口
// 这是 data 层到 biz 层的适配器，实现了依赖倒置原则
func NewCommentRepo(data *Data) biz.CommentRepo { return &commentRepo{data: data} }
