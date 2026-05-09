package data

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"ley/app/blog/internal/biz"
	"ley/pkg/util"

	"go.opentelemetry.io/otel/attribute"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// =============================================================================
// PO 模型 — article schema 下的持久化对象
// =============================================================================

// ArticlePO — article.articles 表
type ArticlePO struct {
	gorm.Model
	Title        string     `gorm:"column:title;type:varchar(200);not null"`
	Slug         string     `gorm:"column:slug;type:varchar(200);uniqueIndex:idx_articles_slug,where:deleted_at IS NULL;not null"`
	Content      string     `gorm:"column:content;type:text;not null"`
	Excerpt      string     `gorm:"column:excerpt;type:text;default:''"`
	CoverImage   string     `gorm:"column:cover_image;type:varchar(512);default:''"`
	Status       int8       `gorm:"column:status;type:smallint;default:0"`
	AuthorID     uint       `gorm:"column:author_id;type:bigint;not null;index:idx_articles_author,where:deleted_at IS NULL"`
	CategoryID   *uint      `gorm:"column:category_id;type:bigint;index:idx_articles_category,where:deleted_at IS NULL"`
	ViewCount    int64      `gorm:"column:view_count;type:bigint;default:0"`
	LikeCount    int64      `gorm:"column:like_count;type:bigint;default:0"`
	CommentCount int64      `gorm:"column:comment_count;type:bigint;default:0"`
	IsTop        bool       `gorm:"column:is_top;type:boolean;default:false"`
	PublishedAt  *time.Time `gorm:"column:published_at;type:timestamptz"`
	Tags         []TagPO    `gorm:"many2many:article.articles_tags;foreignKey:id;joinForeignKey:article_id;References:id;joinReferences:tag_id"`
}

func (ArticlePO) TableName() string { return "article.articles" }

// ArticleTagPO — article.articles_tags 中间表
type ArticleTagPO struct {
	gorm.Model
	ArticleID uint `gorm:"column:article_id;type:bigint;not null"`
	TagID     uint `gorm:"column:tag_id;type:bigint;not null"`
}

func (ArticleTagPO) TableName() string { return "article.articles_tags" }

// ArticleLikePO — article.articles_likes 点赞表
type ArticleLikePO struct {
	gorm.Model
	ArticleID uint `gorm:"column:article_id;type:bigint;not null"`
	UserID    uint `gorm:"column:user_id;type:bigint;not null"`
}

func (ArticleLikePO) TableName() string { return "article.articles_likes" }

// =============================================================================
// 缓存常量
// =============================================================================

const (
	keyArticle      = "article:detail:%d" // 文章详情缓存: article:detail:{id}
	keyArticleSlug  = "article:slug:%s"   // slug→id 映射缓存: article:slug:{slug}
	ttlArticle      = 10 * time.Minute
	ttlArticleStale = 2 * time.Minute
)

// =============================================================================
// articleRepo — biz.ArticleRepo 接口实现
// =============================================================================

type articleRepo struct{ data *Data }

var _ biz.ArticleRepo = (*articleRepo)(nil)

// =============================================================================
// Create
// =============================================================================

// Create 创建文章（INSERT）。slug 冲突时返回 ErrArticleNotFound（调用方按 409 处理）。
func (r *articleRepo) Create(ctx context.Context, a *biz.Article) error {
	ctx, span := r.data.startSpan(ctx, "ArticleRepo.Create")
	defer span.End()
	span.SetAttributes(attribute.String("article.title", a.Title), attribute.String("article.slug", a.Slug))
	r.data.log.WithContext(ctx).Debugf("[ArticleRepo.Create] title=%q slug=%q", a.Title, a.Slug)

	po := toArticlePO(a)
	if err := r.data.db.WithContext(ctx).Create(po).Error; err != nil {
		if util.IsUniqueViolation(err) {
			r.data.log.WithContext(ctx).Warnf("[ArticleRepo.Create] slug冲突 slug=%q", a.Slug)
			return biz.ErrArticleNotFound // slug 重复
		}
		r.data.log.WithContext(ctx).Errorf("[ArticleRepo.Create] 插入失败 title=%q err=%v", a.Title, err)
		return fmt.Errorf("create article: %w", err)
	}
	a.ID = po.ID
	a.CreatedAt = po.CreatedAt
	a.UpdatedAt = po.UpdatedAt
	r.data.log.WithContext(ctx).Infof("[ArticleRepo.Create] 成功 id=%d slug=%q", a.ID, a.Slug)
	return nil
}

// =============================================================================
// Update
// =============================================================================

// Update 使用 map 更新，避免 GORM 零值跳过。更新后清除缓存。
func (r *articleRepo) Update(ctx context.Context, a *biz.Article) error {
	ctx, span := r.data.startSpan(ctx, "ArticleRepo.Update")
	defer span.End()
	span.SetAttributes(attribute.Int("article.id", int(a.ID)))
	r.data.log.WithContext(ctx).Debugf("[ArticleRepo.Update] id=%d title=%q status=%d", a.ID, a.Title, a.Status)

	result := r.data.db.WithContext(ctx).Model(&ArticlePO{}).Where("id = ?", a.ID).Updates(map[string]interface{}{
		"title":        a.Title,
		"slug":         a.Slug,
		"content":      a.Content,
		"excerpt":      a.Excerpt,
		"cover_image":  a.CoverImage,
		"status":       int8(a.Status),
		"category_id":  a.CategoryID,
		"is_top":       a.IsTop,
		"published_at": a.PublishedAt,
	})
	if result.Error != nil {
		r.data.log.WithContext(ctx).Errorf("[ArticleRepo.Update] 失败 id=%d err=%v", a.ID, result.Error)
		return fmt.Errorf("update article: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return biz.ErrArticleNotFound
	}
	r.invalidateCache(ctx, a.ID, a.Slug)
	r.data.log.WithContext(ctx).Infof("[ArticleRepo.Update] 成功 id=%d", a.ID)
	return nil
}

// =============================================================================
// Delete — 软删除并清除缓存
// =============================================================================

func (r *articleRepo) Delete(ctx context.Context, id uint) error {
	ctx, span := r.data.startSpan(ctx, "ArticleRepo.Delete")
	defer span.End()
	span.SetAttributes(attribute.Int("article.id", int(id)))
	r.data.log.WithContext(ctx).Debugf("[ArticleRepo.Delete] id=%d", id)

	// 先查获取 slug，用于缓存清除
	var po ArticlePO
	if err := r.data.db.WithContext(ctx).Select("slug").Where("id = ?", id).First(&po).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return biz.ErrArticleNotFound
		}
		return fmt.Errorf("delete article: %w", err)
	}

	if err := r.data.db.WithContext(ctx).Where("id = ?", id).Delete(&ArticlePO{}).Error; err != nil {
		r.data.log.WithContext(ctx).Errorf("[ArticleRepo.Delete] 失败 id=%d err=%v", id, err)
		return fmt.Errorf("delete article: %w", err)
	}
	r.invalidateCache(ctx, id, po.Slug)
	r.data.log.WithContext(ctx).Infof("[ArticleRepo.Delete] 成功 id=%d", id)
	return nil
}

// =============================================================================
// FindByID — 按主键查询（不走缓存，用于内部调用）
// =============================================================================

func (r *articleRepo) FindByID(ctx context.Context, id uint) (*biz.Article, error) {
	ctx, span := r.data.startSpan(ctx, "ArticleRepo.FindByID")
	defer span.End()
	span.SetAttributes(attribute.Int("article.id", int(id)))

	var po ArticlePO
	err := r.data.db.WithContext(ctx).Preload("Tags").Where("id = ?", id).First(&po).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, biz.ErrArticleNotFound
		}
		return nil, fmt.Errorf("find article by id: %w", err)
	}
	return articlePOToBiz(&po), nil
}

// =============================================================================
// FindBySlug — 按 Slug 查询（Cache-Aside）
// =============================================================================

func (r *articleRepo) FindBySlug(ctx context.Context, slug string) (*biz.Article, error) {
	ctx, span := r.data.startSpan(ctx, "ArticleRepo.FindBySlug")
	defer span.End()
	span.SetAttributes(attribute.String("article.slug", slug))

	slugKey := fmt.Sprintf(keyArticleSlug, slug)
	r.data.log.WithContext(ctx).Debugf("[ArticleRepo.FindBySlug] slug=%q", slug)

	// 尝试从缓存获取 slug→id 映射
	if idBytes, err := r.data.cache.Get(ctx, slugKey); err == nil {
		idStr := string(idBytes)
		if idStr == nullSentinel {
			return nil, biz.ErrArticleNotFound
		}
		var id int
		fmt.Sscanf(idStr, "%d", &id)
		a, err := r.FindByID(ctx, uint(id))
		if err == nil {
			return a, nil
		}
		// 缓存映射失效（文章可能被删），继续查 DB
	}

	// DB 查询
	var po ArticlePO
	dbErr := r.data.db.WithContext(ctx).Preload("Tags").Where("slug = ?", slug).First(&po).Error
	if dbErr != nil {
		if errors.Is(dbErr, gorm.ErrRecordNotFound) {
			r.data.cache.Delete(ctx, slugKey)
			_ = r.data.cache.Set(ctx, slugKey, []byte(nullSentinel), ttlArticleStale)
			return nil, biz.ErrArticleNotFound
		}
		return nil, fmt.Errorf("find article by slug: %w", dbErr)
	}

	// 回写 slug→id 映射缓存
	_ = r.data.cache.Set(ctx, slugKey, []byte(strconv.Itoa(int(po.ID))), ttlArticle)
	r.data.log.WithContext(ctx).Debugf("[ArticleRepo.FindBySlug] 命中 slug=%q id=%d", slug, po.ID)
	return articlePOToBiz(&po), nil
}

// =============================================================================
// List — 分页列表（带过滤和排序）
// =============================================================================

func (r *articleRepo) List(ctx context.Context, query biz.ArticleListQuery) ([]*biz.Article, int64, error) {
	ctx, span := r.data.startSpan(ctx, "ArticleRepo.List")
	defer span.End()
	span.SetAttributes(attribute.Int("page", query.Page), attribute.Int("page_size", query.PageSize))

	if query.Page < 1 {
		query.Page = 1
	}
	if query.PageSize < 1 || query.PageSize > 50 {
		query.PageSize = 20
	}
	offset := (query.Page - 1) * query.PageSize

	r.data.log.WithContext(ctx).Debugf("[ArticleRepo.List] status=%q page=%d size=%d", query.Status, query.Page, query.PageSize)

	// 构建基础查询并预加载 Tags
	db := r.data.db.WithContext(ctx).Preload("Tags")

	// 状态过滤
	if query.Status != "" {
		db = db.Where("status = ?", statusToInt(query.Status))
	}
	if query.CategoryID != nil {
		db = db.Where("category_id = ?", *query.CategoryID)
	}
	if query.AuthorID != nil {
		db = db.Where("author_id = ?", *query.AuthorID)
	}

	// 标签 AND 过滤（HAVING COUNT 确保同时拥有所有标签）
	if len(query.Tags) > 0 {
		db = db.Joins("JOIN \"article\".articles_tags at2 ON at2.article_id = \"article\".articles.id").
			Joins("JOIN \"article\".tags t ON t.id = at2.tag_id").
			Where("t.name IN ?", query.Tags).
			Group("\"article\".articles.id").
			Having("COUNT(DISTINCT t.id) = ?", len(query.Tags))
	}

	// 统计总数
	var total int64
	if err := db.Model(&ArticlePO{}).Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count articles: %w", err)
	}
	if total == 0 {
		return nil, 0, nil
	}

	// 排序
	order := buildOrderClause(query.SortBy, query.SortOrder)

	// 分页查询
	var pos []ArticlePO
	if err := db.Order(order).Offset(offset).Limit(query.PageSize).Find(&pos).Error; err != nil {
		return nil, 0, fmt.Errorf("list articles: %w", err)
	}

	articles := make([]*biz.Article, 0, len(pos))
	for i := range pos {
		articles = append(articles, articlePOToBiz(&pos[i]))
	}
	return articles, total, nil
}

// =============================================================================
// 计数更新（原子操作）
// =============================================================================

func (r *articleRepo) IncrementViewCount(ctx context.Context, id uint, delta int64) error {
	return r.data.db.WithContext(ctx).Model(&ArticlePO{}).Where("id = ?", id).
		UpdateColumn("view_count", gorm.Expr("view_count + ?", delta)).Error
}

func (r *articleRepo) IncrementCommentCount(ctx context.Context, id uint, delta int64) error {
	return r.data.db.WithContext(ctx).Model(&ArticlePO{}).Where("id = ?", id).
		UpdateColumn("comment_count", gorm.Expr("GREATEST(comment_count + ?, 0)", delta)).Error
}

func (r *articleRepo) UpdateCategoryCount(ctx context.Context, categoryID uint, delta int64) error {
	return r.data.db.WithContext(ctx).Model(&CategoryPO{}).Where("id = ?", categoryID).
		UpdateColumn("article_count", gorm.Expr("GREATEST(article_count + ?, 0)", delta)).Error
}

func (r *articleRepo) UpdateTagsArticleCount(ctx context.Context, tagIDs []uint, delta int64) error {
	return r.data.db.WithContext(ctx).Model(&TagPO{}).Where("id IN ?", tagIDs).
		UpdateColumn("article_count", gorm.Expr("GREATEST(article_count + ?, 0)", delta)).Error
}

// =============================================================================
// 标签关联
// =============================================================================

// AssociateTags 添加标签关联（ON CONFLICT DO NOTHING 保证幂等）
func (r *articleRepo) AssociateTags(ctx context.Context, articleID uint, tagIDs []uint) error {
	if len(tagIDs) == 0 {
		return nil
	}
	tags := make([]*ArticleTagPO, 0, len(tagIDs))
	for _, tid := range tagIDs {
		tags = append(tags, &ArticleTagPO{ArticleID: articleID, TagID: tid})
	}
	return r.data.db.WithContext(ctx).Clauses(clause.OnConflict{DoNothing: true}).Create(&tags).Error
}

// SyncTags 全量替换标签（事务：先删后插）
func (r *articleRepo) SyncTags(ctx context.Context, articleID uint, tagIDs []uint) error {
	return r.data.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("article_id = ?", articleID).Delete(&ArticleTagPO{}).Error; err != nil {
			return err
		}
		if len(tagIDs) == 0 {
			return nil
		}
		tags := make([]*ArticleTagPO, 0, len(tagIDs))
		for _, tid := range tagIDs {
			tags = append(tags, &ArticleTagPO{ArticleID: articleID, TagID: tid})
		}
		return tx.Create(&tags).Error
	})
}

// ListTagsByArticleID 查询文章关联的标签
func (r *articleRepo) ListTagsByArticleID(ctx context.Context, articleID uint) ([]*biz.Tag, error) {
	var tags []TagPO
	err := r.data.db.WithContext(ctx).
		Joins("JOIN \"article\".articles_tags at2 ON at2.tag_id = \"article\".tags.id").
		Where("at2.article_id = ?", articleID).Find(&tags).Error
	if err != nil {
		return nil, err
	}
	result := make([]*biz.Tag, 0, len(tags))
	for i := range tags {
		result = append(result, tagPOToBiz(&tags[i]))
	}
	return result, nil
}

// =============================================================================
// 点赞（事务保证计数一致性）
// =============================================================================

// InsertLike 点赞（事务：INSERT OR NOTHING → UPDATE like_count IF inserted）
func (r *articleRepo) InsertLike(ctx context.Context, articleID, userID uint) error {
	return r.data.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		result := tx.Exec(
			`INSERT INTO "article".articles_likes (article_id, user_id, created_at, updated_at) VALUES (?, ?, NOW(), NOW()) ON CONFLICT (article_id, user_id) WHERE deleted_at IS NULL DO NOTHING`,
			articleID, userID)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected > 0 {
			return tx.Model(&ArticlePO{}).Where("id = ?", articleID).
				UpdateColumn("like_count", gorm.Expr("like_count + 1")).Error
		}
		return nil
	})
}

// DeleteLike 取消点赞（事务：DELETE → UPDATE like_count IF deleted）
func (r *articleRepo) DeleteLike(ctx context.Context, articleID, userID uint) error {
	return r.data.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		result := tx.Where("article_id = ? AND user_id = ? AND deleted_at IS NULL", articleID, userID).
			Delete(&ArticleLikePO{})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected > 0 {
			return tx.Model(&ArticlePO{}).Where("id = ?", articleID).
				UpdateColumn("like_count", gorm.Expr("GREATEST(like_count - 1, 0)")).Error
		}
		return nil
	})
}

// IsLiked 查询用户是否已点赞
func (r *articleRepo) IsLiked(ctx context.Context, articleID, userID uint) (bool, error) {
	var count int64
	err := r.data.db.WithContext(ctx).Model(&ArticleLikePO{}).
		Where("article_id = ? AND user_id = ? AND deleted_at IS NULL", articleID, userID).Count(&count).Error
	return count > 0, err
}

// =============================================================================
// 缓存辅助
// =============================================================================

func (r *articleRepo) invalidateCache(ctx context.Context, id uint, slug string) {
	keys := []string{fmt.Sprintf(keyArticle, id)}
	if slug != "" {
		keys = append(keys, fmt.Sprintf(keyArticleSlug, slug))
	}
	for _, k := range keys {
		_ = r.data.cache.Delete(ctx, k)
	}
}

// =============================================================================
// 实体转换
// =============================================================================

func articlePOToBiz(po *ArticlePO) *biz.Article {
	a := &biz.Article{
		ID: po.ID, Title: po.Title, Slug: po.Slug, Content: po.Content,
		Excerpt: po.Excerpt, CoverImage: po.CoverImage, Status: biz.ArticleStatus(po.Status),
		AuthorID: po.AuthorID, CategoryID: po.CategoryID, ViewCount: po.ViewCount,
		LikeCount: po.LikeCount, CommentCount: po.CommentCount, IsTop: po.IsTop,
		PublishedAt: po.PublishedAt, CreatedAt: po.CreatedAt, UpdatedAt: po.UpdatedAt,
	}
	if len(po.Tags) > 0 {
		a.Tags = make([]*biz.Tag, 0, len(po.Tags))
		for i := range po.Tags {
			a.Tags = append(a.Tags, tagPOToBiz(&po.Tags[i]))
		}
	}
	return a
}

func toArticlePO(a *biz.Article) *ArticlePO {
	return &ArticlePO{
		Title: a.Title, Slug: a.Slug, Content: a.Content, Excerpt: a.Excerpt,
		CoverImage: a.CoverImage, Status: int8(a.Status), AuthorID: a.AuthorID,
		CategoryID: a.CategoryID, IsTop: a.IsTop, PublishedAt: a.PublishedAt,
	}
}

// =============================================================================
// 排序
// =============================================================================

func buildOrderClause(sortBy, sortOrder string) string {
	col := "created_at"
	switch sortBy {
	case "updated_at":
		col = "updated_at"
	case "published_at":
		col = "published_at"
	case "view_count":
		col = "view_count"
	case "is_top":
		col = "is_top DESC, published_at"
	}
	dir := "DESC"
	if sortOrder == "asc" {
		dir = "ASC"
	}
	if sortBy == "is_top" {
		return "is_top DESC, published_at " + dir
	}
	return col + " " + dir
}

func statusToInt(s string) int8 {
	switch s {
	case "published":
		return 1
	case "archived":
		return 2
	default:
		return 0
	}
}
