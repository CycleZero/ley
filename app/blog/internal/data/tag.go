package data

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/CycleZero/ley/app/blog/internal/biz"
	"github.com/CycleZero/ley/pkg/util"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// =============================================================================
// TagPO — article.tags 表
// =============================================================================

type TagPO struct {
	gorm.Model
	Name         string `gorm:"column:name;type:varchar(64);uniqueIndex:idx_tags_name,where:deleted_at IS NULL;not null"`
	Slug         string `gorm:"column:slug;type:varchar(64);uniqueIndex:idx_tags_slug,where:deleted_at IS NULL;not null"`
	ArticleCount int64  `gorm:"column:article_count;type:bigint;default:0"`
}

func (TagPO) TableName() string { return "article.tags" }

// CategoryPO — article.categories 表
type CategoryPO struct {
	gorm.Model
	Name         string        `gorm:"column:name;type:varchar(64);not null"`
	Slug         string        `gorm:"column:slug;type:varchar(64);uniqueIndex;not null"`
	Description  string        `gorm:"column:description;type:text;default:''"`
	ParentID     *uint         `gorm:"column:parent_id;type:bigint;index"`
	SortOrder    int           `gorm:"column:sort_order;type:int;default:0"`
	ArticleCount int64         `gorm:"column:article_count;type:bigint;default:0"`
}

func (CategoryPO) TableName() string { return "article.categories" }

// =============================================================================
// 缓存
// =============================================================================

const (
	cacheKeyTagAll      = "tag:all"
	cacheKeyCatTree     = "category:tree"
	ttlTagAll           = 60 * time.Minute
	ttlCatTree          = 60 * time.Minute
)

// =============================================================================
// tagRepo — biz.TagRepo 接口实现
// =============================================================================

type tagRepo struct{ data *Data }

var _ biz.TagRepo = (*tagRepo)(nil)

func (r *tagRepo) Create(ctx context.Context, tag *biz.Tag) error {
	po := &TagPO{Name: tag.Name, Slug: tag.Slug}
	if err := r.data.db.WithContext(ctx).Create(po).Error; err != nil {
		if util.IsUniqueViolation(err) {
			return biz.ErrTagNameExists
		}
		return fmt.Errorf("create tag: %w", err)
	}
	tag.ID = po.ID
	tag.CreatedAt = po.CreatedAt
	r.data.cache.Delete(ctx, cacheKeyTagAll)
	r.data.log.WithContext(ctx).Infof("[TagRepo.Create] 成功 id=%d name=%q", tag.ID, tag.Name)
	return nil
}

func (r *tagRepo) FindByName(ctx context.Context, name string) (*biz.Tag, error) {
	var po TagPO
	if err := r.data.db.WithContext(ctx).Where("name = ?", name).First(&po).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, biz.ErrTagNotFound
		}
		return nil, err
	}
	return tagPOToBiz(&po), nil
}

func (r *tagRepo) FindOrCreate(ctx context.Context, name, slug string) (*biz.Tag, error) {
	// 先查
	tag, err := r.FindByName(ctx, name)
	if err == nil {
		return tag, nil
	}
	if !errors.Is(err, biz.ErrTagNotFound) {
		return nil, err
	}
	// 创建（ON CONFLICT 防并发）
	po := &TagPO{Name: name, Slug: slug}
	if err := r.data.db.WithContext(ctx).Clauses(clause.OnConflict{DoNothing: true}).Create(po).Error; err != nil {
		return nil, fmt.Errorf("find or create tag: %w", err)
	}
	// 重新查询取完整数据（如果 ON CONFLICT 跳过，po.ID 为 0）
	return r.FindByName(ctx, name)
}

func (r *tagRepo) List(ctx context.Context) ([]*biz.Tag, error) {
	// Cache-Aside
	var tags []*biz.Tag
	if err := r.data.cache.GetObject(ctx, cacheKeyTagAll, &tags); err == nil {
		return tags, nil
	}
	var pos []TagPO
	if err := r.data.db.WithContext(ctx).Order("article_count DESC, name ASC").Find(&pos).Error; err != nil {
		return nil, err
	}
	tags = make([]*biz.Tag, 0, len(pos))
	for i := range pos {
		tags = append(tags, tagPOToBiz(&pos[i]))
	}
	_ = r.data.cache.Set(ctx, cacheKeyTagAll, tags, ttlTagAll)
	return tags, nil
}

func (r *tagRepo) Delete(ctx context.Context, id uint) error {
	result := r.data.db.WithContext(ctx).Where("id = ?", id).Delete(&TagPO{})
	if result.RowsAffected == 0 {
		return biz.ErrTagNotFound
	}
	r.data.cache.Delete(ctx, cacheKeyTagAll)
	r.data.log.WithContext(ctx).Infof("[TagRepo.Delete] 成功 id=%d", id)
	return result.Error
}

func (r *tagRepo) IncrementArticleCount(ctx context.Context, id uint, delta int64) error {
	return r.data.db.WithContext(ctx).Model(&TagPO{}).Where("id = ?", id).
		UpdateColumn("article_count", gorm.Expr("GREATEST(article_count + ?, 0)", delta)).Error
}

// =============================================================================
// categoryRepo — biz.CategoryRepo 接口实现
// =============================================================================

type categoryRepo struct{ data *Data }

var _ biz.CategoryRepo = (*categoryRepo)(nil)

func (r *categoryRepo) Create(ctx context.Context, cat *biz.Category) error {
	po := &CategoryPO{Name: cat.Name, Slug: cat.Slug, Description: cat.Description, ParentID: cat.ParentID, SortOrder: cat.SortOrder}
	if err := r.data.db.WithContext(ctx).Create(po).Error; err != nil {
		if util.IsUniqueViolation(err) {
			return biz.ErrCategoryNameExists
		}
		return fmt.Errorf("create category: %w", err)
	}
	cat.ID = po.ID
	cat.CreatedAt = po.CreatedAt
	r.data.cache.Delete(ctx, cacheKeyCatTree)
	r.data.log.WithContext(ctx).Infof("[CategoryRepo.Create] 成功 id=%d name=%q", cat.ID, cat.Name)
	return nil
}

func (r *categoryRepo) Update(ctx context.Context, cat *biz.Category) error {
	result := r.data.db.WithContext(ctx).Model(&CategoryPO{}).Where("id = ?", cat.ID).Updates(map[string]interface{}{
		"name": cat.Name, "slug": cat.Slug, "description": cat.Description,
		"parent_id": cat.ParentID, "sort_order": cat.SortOrder,
	})
	if result.RowsAffected == 0 {
		return biz.ErrCategoryNotFound
	}
	r.data.cache.Delete(ctx, cacheKeyCatTree)
	r.data.log.WithContext(ctx).Infof("[CategoryRepo.Update] 成功 id=%d", cat.ID)
	return result.Error
}

func (r *categoryRepo) Delete(ctx context.Context, id uint) error {
	result := r.data.db.WithContext(ctx).Where("id = ?", id).Delete(&CategoryPO{})
	if result.RowsAffected == 0 {
		return biz.ErrCategoryNotFound
	}
	r.data.cache.Delete(ctx, cacheKeyCatTree)
	r.data.log.WithContext(ctx).Infof("[CategoryRepo.Delete] 成功 id=%d", id)
	return result.Error
}

func (r *categoryRepo) FindByID(ctx context.Context, id uint) (*biz.Category, error) {
	var po CategoryPO
	if err := r.data.db.WithContext(ctx).Where("id = ?", id).First(&po).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, biz.ErrCategoryNotFound
		}
		return nil, err
	}
	return catPOToBiz(&po), nil
}

func (r *categoryRepo) ListChildren(ctx context.Context, parentID uint) ([]*biz.Category, error) {
	var pos []CategoryPO
	if err := r.data.db.WithContext(ctx).Where("parent_id = ?", parentID).Order("sort_order ASC").Find(&pos).Error; err != nil {
		return nil, err
	}
	result := make([]*biz.Category, 0, len(pos))
	for i := range pos {
		result = append(result, catPOToBiz(&pos[i]))
	}
	return result, nil
}

func (r *categoryRepo) ListTree(ctx context.Context) ([]*biz.Category, error) {
	// Cache-Aside
	var tree []*biz.Category
	if err := r.data.cache.GetObject(ctx, cacheKeyCatTree, &tree); err == nil {
		return tree, nil
	}
	var all []CategoryPO
	if err := r.data.db.WithContext(ctx).Order("sort_order ASC, id ASC").Find(&all).Error; err != nil {
		return nil, err
	}
	// 内存构建树
	nodeMap := make(map[uint]*biz.Category, len(all))
	for i := range all {
		nodeMap[all[i].ID] = catPOToBiz(&all[i])
	}
	var roots []*biz.Category
	for _, po := range all {
		node := nodeMap[po.ID]
		if po.ParentID != nil && *po.ParentID != 0 {
			if parent, ok := nodeMap[*po.ParentID]; ok {
				parent.Children = append(parent.Children, node)
				continue
			}
		}
		roots = append(roots, node)
	}
	_ = r.data.cache.Set(ctx, cacheKeyCatTree, roots, ttlCatTree)
	return roots, nil
}

func (r *categoryRepo) IncrementArticleCount(ctx context.Context, id uint, delta int64) error {
	return r.data.db.WithContext(ctx).Model(&CategoryPO{}).Where("id = ?", id).
		UpdateColumn("article_count", gorm.Expr("GREATEST(article_count + ?, 0)", delta)).Error
}

// =============================================================================
// PO ↔ biz 转换
// =============================================================================

func tagPOToBiz(po *TagPO) *biz.Tag {
	return &biz.Tag{ID: po.ID, Name: po.Name, Slug: po.Slug, ArticleCount: po.ArticleCount, CreatedAt: po.CreatedAt}
}

func catPOToBiz(po *CategoryPO) *biz.Category {
	return &biz.Category{
		ID: po.ID, Name: po.Name, Slug: po.Slug, Description: po.Description,
		ParentID: po.ParentID, SortOrder: po.SortOrder, ArticleCount: po.ArticleCount, CreatedAt: po.CreatedAt,
	}
}
