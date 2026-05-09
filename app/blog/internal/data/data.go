package data

import (
	"context"
	"fmt"

	"ley/app/blog/internal/biz"
	"ley/app/blog/internal/conf"
	"ley/pkg/cache"
	"ley/pkg/infra"
	"ley/pkg/oss"

	etcdreg "github.com/go-kratos/kratos/contrib/registry/etcd/v2"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/registry"
	"github.com/google/wire"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"go.etcd.io/etcd/client/v3"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"gorm.io/gorm"
)

// Data 数据层聚合 — 持有 DB、Cache、Logger、Tracer。
type Data struct {
	db     *gorm.DB
	cache  cache.Cache
	log    *log.Helper
	tracer trace.Tracer
}

// NewData 创建 Data 实例，注册 OTel Tracer 为 "blog-service.data"。
func NewData(db *gorm.DB, c cache.Cache, logger log.Logger) (*Data, func()) {
	d := &Data{
		db:     db,
		cache:  c,
		log:    log.NewHelper(logger),
		tracer: otel.Tracer("blog-service.data"),
	}
	return d, func() {}
}

// startSpan 创建追踪 Span，附带 PostgreSQL 语义属性。
func (d *Data) startSpan(ctx context.Context, name string) (context.Context, trace.Span) {
	ctx, span := d.tracer.Start(ctx, name, trace.WithSpanKind(trace.SpanKindClient))
	span.SetAttributes(
		attribute.String("db.system", "postgresql"),
		attribute.String("db.service", "blog-service"),
	)
	return ctx, span
}

// =============================================================================
// Provider 函数
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

func ProvideOSS(sc *conf.Config) (oss.OSS, func()) {
	client, err := minio.New(sc.Minio.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(sc.Minio.AccessKeyId, sc.Minio.AccessKeySecret, ""),
		Secure: false,
	})
	if err != nil {
		panic(fmt.Errorf("create minio client: %w", err))
	}
	o := oss.NewMinioOSS(client, sc.Minio.BucketName)
	return o, func() {}
}

func ProvideRegistrar(etcdClient *clientv3.Client) registry.Registrar {
	return etcdreg.New(etcdClient)
}

// =============================================================================
// Wire ProviderSet
// =============================================================================

var ProviderSet = wire.NewSet(
	NewData,
	NewArticleRepo,
	NewCommentRepo,
	NewTagRepo,
	NewCategoryRepo,
	NewFileRepo,
	NewSiteRepo,
	ProvideDB,
	ProvideCache,
	ProvideOSS,
	ProvideRegistrar,
)

// NewArticleRepo 创建 ArticleRepo 实现。
func NewArticleRepo(d *Data) biz.ArticleRepo { return &articleRepo{data: d} }

// NewCommentRepo 创建 CommentRepo 实现。
func NewCommentRepo(d *Data) biz.CommentRepo { return &commentRepo{data: d} }

// NewTagRepo 创建 TagRepo 实现。
func NewTagRepo(d *Data) biz.TagRepo { return &tagRepo{data: d} }

// NewCategoryRepo 创建 CategoryRepo 实现。
func NewCategoryRepo(d *Data) biz.CategoryRepo { return &categoryRepo{data: d} }

// NewFileRepo 创建 FileRepo 实现（依赖 MinIO）。
func NewFileRepo(d *Data, oss oss.OSS) biz.FileRepo { return &fileRepo{data: d, oss: oss} }

// NewSiteRepo 创建 SiteRepo 实现（依赖 MinIO）。
func NewSiteRepo(d *Data, oss oss.OSS) biz.SiteRepo { return &siteRepo{data: d, oss: oss} }

// nullSentinel 空值标记 — 写入缓存表示数据库确认该记录不存在。
const nullSentinel = "null"
