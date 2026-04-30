package data

import (
	"context"
	"errors"
	"fmt"
	"time"

	"ley/app/post/internal/biz"
	"ley/app/post/internal/model"
	"ley/pkg/cache"
	"ley/pkg/util"

	"go.opentelemetry.io/otel/attribute"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// =============================================================================
// 标签缓存常量
// =============================================================================

const (
	cacheKeyTagAll = "tag:all"        // 全量标签缓存键
	cacheTTLTagAll = 60 * time.Minute // 全量标签缓存过期时间
)

// =============================================================================
// 标签相关错误定义
// =============================================================================

var (
	ErrTagNotFound   = errors.New("tag not found")       // 标签不存在
	ErrTagNameExists = errors.New("tag name already exists") // 标签名称已存在
)

// =============================================================================
// tagRepo — TagRepo 接口实现
// =============================================================================

// tagRepo TagRepo 接口实现
// 封装标签相关的数据库操作和缓存逻辑。
type tagRepo struct{ data *Data }

// 编译时接口实现检查
var _ biz.TagRepo = (*tagRepo)(nil)

// =============================================================================
// Create — 创建标签
// =============================================================================

// Create 创建标签
// 将标签写入数据库，若名称冲突则返回 ErrTagNameExists。
func (r *tagRepo) Create(ctx context.Context, tag *model.Tag) error {
	ctx, span := r.data.StartSpan(ctx, "TagRepo.Create")
	defer span.End()

	span.SetAttributes(attribute.String("tag.name", tag.Name), attribute.String("tag.slug", tag.Slug))
	r.data.Log(ctx).Debugw("开始创建标签", "name", tag.Name, "slug", tag.Slug)

	// 执行数据库插入
	err := r.data.db.WithContext(ctx).Create(tag).Error
	if err != nil {
		// 判断是否为唯一键冲突（名称重复）
		if util.IsUniqueViolation(err) {
			r.data.Log(ctx).Warnw("创建标签失败：名称重复", "name", tag.Name, "error", err)
			return ErrTagNameExists
		}
		r.data.Log(ctx).Errorw("创建标签数据库写入失败", "name", tag.Name, "error", err)
		return fmt.Errorf("create tag: %w", err)
	}

	// 创建标签后使缓存失效，保证下次查询拿到最新数据
	r.invalidateTagCache(ctx)

	r.data.Log(ctx).Infow("标签创建成功", "id", tag.ID, "name", tag.Name)
	return nil
}

// =============================================================================
// FindByName — 按名称查询标签
// =============================================================================

// FindByName 按名称查询标签
// 直接查询数据库，不使用缓存。
func (r *tagRepo) FindByName(ctx context.Context, name string) (*model.Tag, error) {
	ctx, span := r.data.StartSpan(ctx, "TagRepo.FindByName")
	defer span.End()

	span.SetAttributes(attribute.String("tag.name", name))
	r.data.Log(ctx).Debugw("按名称查询标签", "name", name)

	var tag model.Tag
	err := r.data.db.WithContext(ctx).Where("name = ?", name).First(&tag).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			r.data.Log(ctx).Debugw("按名称查询标签未找到", "name", name)
			return nil, ErrTagNotFound
		}
		r.data.Log(ctx).Errorw("按名称查询标签失败", "name", name, "error", err)
		return nil, fmt.Errorf("find tag by name: %w", err)
	}
	r.data.Log(ctx).Debugw("按名称查询标签成功", "name", name, "id", tag.ID)
	return &tag, nil
}

// =============================================================================
// FindOrCreate — 查找标签，不存在则创建（幂等）
// =============================================================================

// FindOrCreate 查找标签，不存在则创建（幂等）
// 使用 ON CONFLICT DO NOTHING 确保并发安全创建。
func (r *tagRepo) FindOrCreate(ctx context.Context, name, slug string) (*model.Tag, error) {
	ctx, span := r.data.StartSpan(ctx, "TagRepo.FindOrCreate")
	defer span.End()

	span.SetAttributes(attribute.String("tag.name", name), attribute.String("tag.slug", slug))
	r.data.Log(ctx).Debugw("开始查找或创建标签", "name", name, "slug", slug)

	// 第一步：按名称查找，已存在则直接返回
	tag, err := r.FindByName(ctx, name)
	if err == nil {
		r.data.Log(ctx).Debugw("标签已存在，直接返回", "name", name, "id", tag.ID)
		return tag, nil
	}
	// 非"未找到"的错误直接返回
	if !errors.Is(err, ErrTagNotFound) {
		return nil, err
	}
	r.data.Log(ctx).Debugw("标签不存在，尝试创建", "name", name)

	// 第二步：创建标签，ON CONFLICT DO NOTHING 应对并发创建
	tag = &model.Tag{Name: name, Slug: slug}
	createErr := r.data.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "name"}}, // 名称唯一冲突时忽略
		DoNothing: true,
	}).Create(tag).Error
	if createErr != nil {
		r.data.Log(ctx).Errorw("查找或创建标签失败", "name", name, "error", createErr)
		return nil, fmt.Errorf("find or create tag: %w", createErr)
	}

	// 第三步：创建后重新查询以获取完整的数据库生成字段（ID、时间戳等）
	tag, err = r.FindByName(ctx, name)
	if err != nil {
		r.data.Log(ctx).Errorw("创建标签后重新查询失败", "name", name, "error", err)
		return nil, err
	}

	// 创建标签后清除全量标签缓存
	r.invalidateTagCache(ctx)
	r.data.Log(ctx).Debugw("查找或创建标签完成", "name", name, "id", tag.ID)
	return tag, nil
}

// =============================================================================
// List — 查询全量标签列表（Cache-Aside）
// =============================================================================

// List 查询全量标签列表（Cache-Aside 模式）
// 1. 先查缓存，命中则直接返回
// 2. 缓存未命中则查询数据库
// 3. 数据库查询结果回填缓存
func (r *tagRepo) List(ctx context.Context) ([]*model.Tag, error) {
	ctx, span := r.data.StartSpan(ctx, "TagRepo.List")
	defer span.End()

	r.data.Log(ctx).Debugw("查询全量标签列表，尝试读取缓存", "cache_key", cacheKeyTagAll)

	// 第一步：尝试从缓存读取
	var tags []*model.Tag
	err := r.data.cache.GetObject(ctx, cacheKeyTagAll, &tags)
	if err == nil {
		r.data.Log(ctx).Debugw("全量标签列表缓存命中", "count", len(tags))
		return tags, nil
	}
	if !errors.Is(err, cache.ErrKeyNotFound) {
		r.data.Log(ctx).Warnw("全量标签列表缓存读取异常，回退到数据库查询", "error", err)
	}
	r.data.Log(ctx).Debugw("全量标签列表缓存未命中，查询数据库")

	// 第二步：缓存未命中，查询数据库（按文章数量降序、名称升序）
	if err := r.data.db.WithContext(ctx).Order("post_count DESC, name ASC").Find(&tags).Error; err != nil {
		r.data.Log(ctx).Errorw("查询全量标签列表数据库失败", "error", err)
		return nil, fmt.Errorf("list tags: %w", err)
	}

	// 第三步：将数据库查询结果回填缓存
	r.data.Log(ctx).Debugw("数据库查询全量标签成功，回填缓存", "count", len(tags))
	if setErr := r.data.cache.Set(ctx, cacheKeyTagAll, tags, cacheTTLTagAll); setErr != nil {
		r.data.Log(ctx).Warnw("全量标签缓存写入失败", "error", setErr)
	}

	span.SetAttributes(attribute.Int("tag.count", len(tags)))
	return tags, nil
}

// =============================================================================
// Delete — 软删除标签
// =============================================================================

// Delete 软删除标签
// 执行软删除并清除相关缓存。
func (r *tagRepo) Delete(ctx context.Context, id uint) error {
	ctx, span := r.data.StartSpan(ctx, "TagRepo.Delete")
	defer span.End()

	span.SetAttributes(attribute.Int("tag.id", int(id)))
	r.data.Log(ctx).Debugw("开始删除标签", "id", id)

	// 执行软删除
	result := r.data.db.WithContext(ctx).Where("id = ?", id).Delete(&model.Tag{})
	if result.Error != nil {
		r.data.Log(ctx).Errorw("删除标签数据库操作失败", "id", id, "error", result.Error)
		return fmt.Errorf("delete tag: %w", result.Error)
	}
	// 检查是否有行被实际删除
	if result.RowsAffected == 0 {
		r.data.Log(ctx).Debugw("删除标签未找到记录", "id", id)
		return ErrTagNotFound
	}

	// 删除标签后使缓存失效
	r.invalidateTagCache(ctx)
	r.data.Log(ctx).Infow("标签删除成功", "id", id)
	return nil
}

// =============================================================================
// IncrementPostCount — 原子增减标签文章计数
// =============================================================================

// IncrementPostCount 原子增减标签文章计数
// 使用 SQL 表达式 post_count + ? 确保并发安全。
func (r *tagRepo) IncrementPostCount(ctx context.Context, id uint, delta int) error {
	ctx, span := r.data.StartSpan(ctx, "TagRepo.IncrementPostCount")
	defer span.End()

	span.SetAttributes(attribute.Int("tag.id", int(id)), attribute.Int("delta", delta))
	r.data.Log(ctx).Debugw("原子增减标签文章计数", "id", id, "delta", delta)

	// 使用 gorm.Expr 执行原子更新，避免读写竞争
	result := r.data.db.WithContext(ctx).Model(&model.Tag{}).
		Where("id = ?", id).UpdateColumn("post_count", gorm.Expr("post_count + ?", delta))

	if result.Error != nil {
		r.data.Log(ctx).Errorw("增减标签文章计数失败", "id", id, "delta", delta, "error", result.Error)
		return fmt.Errorf("increment tag post count: %w", result.Error)
	}

	r.data.Log(ctx).Debugw("标签文章计数更新成功", "id", id, "delta", delta)
	return nil
}

// =============================================================================
// invalidateTagCache — 清除标签缓存
// =============================================================================

// invalidateTagCache 清除标签缓存
// 标签发生变更时清除全量标签列表缓存。
func (r *tagRepo) invalidateTagCache(ctx context.Context) {
	r.data.Log(ctx).Debugw("清除全量标签缓存", "cache_key", cacheKeyTagAll)
	if err := r.data.cache.Delete(ctx, cacheKeyTagAll); err != nil {
		r.data.Log(ctx).Warnw("删除全量标签缓存失败", "error", err)
	}
}
