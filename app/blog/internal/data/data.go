package data

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/CycleZero/ley/app/blog/internal/biz"
	"github.com/CycleZero/ley/app/blog/internal/conf"
	"github.com/CycleZero/ley/pkg/cache"
	"github.com/CycleZero/ley/pkg/infra"
	"github.com/CycleZero/ley/pkg/oss"

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
// 同时启动浏览量定时 flush 后台任务（每 5 分钟将 Redis 缓存增量持久化到数据库）。
func NewData(db *gorm.DB, c cache.Cache, logger log.Logger) (*Data, func()) {
	d := &Data{
		db:     db,
		cache:  c,
		log:    log.NewHelper(logger),
		tracer: otel.Tracer("blog-service.data"),
	}

	// 启动浏览量定时 flush 任务
	stopFlusher := d.startViewCountFlusher()

	return d, func() {
		stopFlusher()
	}
}

// startSpan 创建追踪 Span，附带 PostgreSQL 语义属性。
func (d *Data) startSpan(ctx context.Context, name string) (context.Context, trace.Span) {
	ctx, span := d.tracer.Start(ctx, name, trace.WithSpanKind(trace.SpanKindClient))
	span.SetAttributes(
		attribute.String("db.system", infra.DBSystemName(d.db)),
		attribute.String("db.service", "blog-service"),
	)
	return ctx, span
}

// =============================================================================
// 浏览量定时 flush（后台 goroutine）
// =============================================================================

const viewCountFlushInterval = 5 * time.Minute

// startViewCountFlusher 启动定时 flush 任务，返回 stop 函数。
func (d *Data) startViewCountFlusher() func() {
	ticker := time.NewTicker(viewCountFlushInterval)
	done := make(chan struct{})

	go func() {
		for {
			select {
			case <-ticker.C:
				d.flushViewCounts(context.Background())
			case <-done:
				ticker.Stop()
				return
			}
		}
	}()

	return func() {
		close(done)
		// 优雅关闭时执行最后一次 flush
		d.flushViewCounts(context.Background())
	}
}

// flushViewCounts 将 Redis 中 article:view:* 的增量批量持久化到数据库。
// 流程：Scan 扫描 key → MGet 读取增量 → 事务 UPDATE → 删除已 flush 的 Redis key。
func (d *Data) flushViewCounts(ctx context.Context) {
	keys, err := d.cache.ScanAll(ctx, "ley:article:view:*")
	if err != nil {
		d.log.Errorf("[FlushViewCounts] ScanAll 失败: %v", err)
		return
	}
	if len(keys) == 0 {
		return
	}

	d.log.Infof("[FlushViewCounts] 开始 flush, keys=%d", len(keys))

	// MGet 读取所有增量值
	kv, err := d.cache.MGet(ctx, keys)
	if err != nil {
		d.log.Errorf("[FlushViewCounts] MGet 失败: %v", err)
		return
	}

	// 解析 counts map
	counts := make(map[uint]int64)
	for key, raw := range kv {
		// key 格式: ley:article:view:{id}
		parts := strings.Split(key, ":")
		if len(parts) < 4 {
			continue
		}
		id, err := strconv.ParseUint(parts[3], 10, 64)
		if err != nil {
			continue
		}
		delta, err := strconv.ParseInt(string(raw), 10, 64)
		if err != nil || delta <= 0 {
			continue
		}
		counts[uint(id)] = delta
	}

	if len(counts) == 0 {
		return
	}

	// 事务批量写入数据库
	if err := d.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for id, delta := range counts {
			if err := tx.Model(&ArticlePO{}).Where("id = ?", id).
				UpdateColumn("view_count", gorm.Expr("view_count + ?", delta)).Error; err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		d.log.Errorf("[FlushViewCounts] 数据库批量更新失败: %v", err)
		return
	}

	// 删除已 flush 的 Redis key（逐个删除，忽略错误）
	for _, key := range keys {
		_ = d.cache.Delete(ctx, key)
	}

	var totalDelta int64
	for _, d := range counts {
		totalDelta += d
	}
	d.log.Infof("[FlushViewCounts] 成功 flush, articles=%d total_delta=%d", len(counts), totalDelta)
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

// ProvideOSS 按配置选择对象存储后端：
//   - provider=aliyun 时使用阿里云 OSS（Region 必填）
//   - 默认（provider 为空或 minio）使用 MinIO/S3 兼容服务
func ProvideOSS(sc *conf.Config) (oss.OSS, func()) {
	// 阿里云 OSS：endpoint 字段忽略，Region 必填
	if sc.Minio.Provider == "aliyun" {
		o, err := oss.NewAliyunOSS(oss.AliyunConfig{
			Region:          sc.Minio.Region,
			AccessKeyID:     sc.Minio.AccessKeyId,
			AccessKeySecret: sc.Minio.AccessKeySecret,
			BucketName:      sc.Minio.BucketName,
		})
		if err != nil {
			panic(fmt.Errorf("create aliyun oss client: %w", err))
		}
		return o, func() {}
	}

	// MinIO/S3 兼容：MinioOSS 包装 minio.Core（内嵌 Client 全部能力 +
	// 分页列举 / multipart / 范围读等低层 API）
	core, err := minio.NewCore(sc.Minio.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(sc.Minio.AccessKeyId, sc.Minio.AccessKeySecret, ""),
		Secure: false,
	})
	if err != nil {
		panic(fmt.Errorf("create minio client: %w", err))
	}
	o := oss.NewMinioOSS(core, sc.Minio.BucketName)
	return o, func() {}
}

func ProvideRegistrar(etcdClient *clientv3.Client) registry.Registrar {
	return etcdreg.New(etcdClient)
}

// =============================================================================
// Wire ProviderSet
// =============================================================================

var ProviderSet = wire.NewSet(
	wire.Bind(new(biz.LikesGate), new(biz.SiteRepo)),
	NewData,
	NewArticleRepo,
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
func NewArticleRepo(d *Data) biz.ArticleRepo {
	d.db.AutoMigrate(&ArticlePO{}, &ArticleTagPO{}, &ArticleLikePO{})
	return &articleRepo{data: d}
}

// NewTagRepo 创建 TagRepo 实现。
func NewTagRepo(d *Data) biz.TagRepo {
	d.db.AutoMigrate(&TagPO{})
	return &tagRepo{data: d}
}

// NewCategoryRepo 创建 CategoryRepo 实现。
func NewCategoryRepo(d *Data) biz.CategoryRepo {
	d.db.AutoMigrate(&CategoryPO{})
	return &categoryRepo{data: d}
}

// NewFileRepo 创建 FileRepo 实现（依赖 MinIO）。
func NewFileRepo(d *Data, oss oss.OSS) biz.FileRepo {
	d.db.AutoMigrate(&FilePO{})
	return &fileRepo{data: d, oss: oss}
}

// NewSiteRepo 创建 SiteRepo 实现（依赖 MinIO）。
func NewSiteRepo(d *Data, oss oss.OSS) biz.SiteRepo {
	d.db.AutoMigrate(&SiteSettingPO{}, &SiteBackgroundPO{})
	return &siteRepo{data: d, oss: oss}
}

// nullSentinel 空值标记 — 写入缓存表示数据库确认该记录不存在。
const nullSentinel = "null"
