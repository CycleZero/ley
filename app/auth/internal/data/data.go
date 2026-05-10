package data

import (
	"context"

	"github.com/CycleZero/ley/app/auth/internal/biz"
	"github.com/CycleZero/ley/app/auth/internal/conf"
	"github.com/CycleZero/ley/pkg/cache"
	"github.com/CycleZero/ley/pkg/infra"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/wire"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"gorm.io/gorm"
)

// Data 数据层聚合结构体 — 封装所有持久化和中间件资源。
type Data struct {
	db     *gorm.DB
	cache  cache.Cache
	logger *log.Helper
	tracer trace.Tracer
}

// NewData 创建 Data 实例，初始化 OpenTelemetry Tracer。
func NewData(db *gorm.DB, c cache.Cache, tracerName string, logger log.Logger) (*Data, func()) {
	d := &Data{
		db:     db,
		cache:  c,
		logger: log.NewHelper(logger),
		tracer: otel.Tracer(tracerName),
	}
	cleanup := func() {}
	return d, cleanup
}

// StartSpan 创建当前操作的 OpenTelemetry Span。
func (d *Data) StartSpan(ctx context.Context, spanName string) (context.Context, trace.Span) {
	ctx, span := d.tracer.Start(ctx, spanName, trace.WithSpanKind(trace.SpanKindClient))
	span.SetAttributes(
		attribute.String("db.system", "postgresql"),
		attribute.String("db.service", "auth-service"),
	)
	return ctx, span
}

// =============================================================================
// Provider 函数 — 从配置创建基础设施依赖
// =============================================================================

func ProvideDB(confData *conf.Data) *gorm.DB {
	return infra.NewDB(infra.NewDbParams{
		Driver: confData.Database.Driver,
		Host:   confData.Database.Host,
		Port:   int(confData.Database.Port),
		User:   confData.Database.Username,
		Pass:   confData.Database.Password,
		DBName: confData.Database.Database,
	})
}

func ProvideCache(confData *conf.Data) cache.Cache {
	return cache.NewRedisCache(confData.Redis.Host, int(confData.Redis.Port), confData.Redis.Password, int(confData.Redis.Db))
}

func ProvideTracerName() string {
	return "auth-service"
}

// =============================================================================
// Wire ProviderSet
// =============================================================================

var ProviderSet = wire.NewSet(NewData, NewUserRepo, ProvideDB, ProvideCache, ProvideTracerName)

// NewUserRepo 创建 UserRepo 实现（返回 biz.UserRepo 接口）。
func NewUserRepo(d *Data) biz.UserRepo {
	return &userRepo{data: d}
}
