package data

import (
	"context"

	"ley/app/user/internal/biz"
	"ley/pkg/cache"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/wire"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"gorm.io/gorm"
)

// Data 用户服务数据层聚合结构
// 聚合 DB（GORM）、缓存（Cache）、日志（log.Helper）、链路追踪（OTel）
// 各 Repo 通过 Data 获取底层能力，面向接口而非具体实现
type Data struct {
	db     *gorm.DB     // PostgreSQL GORM 数据库连接（从 pkg/infra 注入）
	cache  cache.Cache  // Redis 缓存客户端（实现 cache.Cache 接口）
	logger log.Logger   // Kratos log.Logger 接口（由 main 初始化注入，支持 OTel 上下文传播）
	tracer trace.Tracer // OpenTelemetry Tracer（用于创建 Span 进行链路追踪）
}

// NewData 创建 Data 实例
// db: GORM 数据库连接（pkg/infra.NewDB）
// c: Cache 接口实现（pkg/cache.NewRedisCache）
// logger: Kratos log.Logger 接口（main 函数中通过 pkg/log.GetKratosLogger() 传入）
// 初始化 Tracer 名称为 "user-service.data"，用于 OTel 链路追踪区分服务层级
func NewData(db *gorm.DB, c cache.Cache, logger log.Logger) *Data {
	return &Data{
		db:     db,
		cache:  c,
		logger: logger,
		tracer: otel.Tracer("user-service.data"),
	}
}

// Log 返回携带 SpanContext 的 Kratos log.Helper
// 自动从 ctx 中提取 OpenTelemetry trace_id/span_id 注入日志
// 调用方通过返回的 Helper 打印日志时，自动携带链路追踪信息
func (d *Data) Log(ctx context.Context) *log.Helper {
	return log.NewHelper(log.WithContext(ctx, d.logger))
}

// StartSpan 创建 OTel 子 Span，统一注入 db.system 属性
// spanName 推荐格式: "UserRepo.MethodName"
// 返回带新 Span 的 ctx，调用方必须 defer span.End() 确保 Span 正确关闭
// 默认设置为 SpanKindClient 并注入 PostgreSQL 服务属性
func (d *Data) StartSpan(ctx context.Context, spanName string) (context.Context, trace.Span) {
	return d.tracer.Start(ctx, spanName,
		trace.WithSpanKind(trace.SpanKindClient),      // 标记为客户端调用 Span
		trace.WithAttributes(
			attribute.String("db.system", "postgresql"), // OTel 语义约定：数据库系统类型
			attribute.String("db.service", "user-service"), // 数据库所属服务名称
		),
	)
}

// ProviderSet Wire 依赖注入集合
// 包含 Data 聚合结构及其 Repo 构造函数，供 Wire 自动组装依赖链
var ProviderSet = wire.NewSet(NewData, NewUserRepo)

// NewUserRepo 创建 UserRepo 接口实现
// 将 Data 实例注入 userRepo，返回 biz.UserRepo 接口
// 调用方（biz 层）只依赖接口，不依赖具体实现
func NewUserRepo(data *Data) biz.UserRepo {
	return &userRepo{data: data}
}
