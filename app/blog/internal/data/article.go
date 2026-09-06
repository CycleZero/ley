package data

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/CycleZero/ley/app/blog/internal/biz"
	"github.com/CycleZero/ley/pkg/util"

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
	Excerpt      string     `gorm:"column:excerpt;type:text"`
	CoverImage   string     `gorm:"column:cover_image;type:varchar(512);default:''"`
	Status       int8       `gorm:"column:status;type:smallint;default:0"`
	AuthorID     uint       `gorm:"column:author_id;type:bigint;not null;index:idx_articles_author,where:deleted_at IS NULL"`
	CategoryID   *uint      `gorm:"column:category_id;type:bigint;index:idx_articles_category,where:deleted_at IS NULL"`
	ViewCount    int64      `gorm:"column:view_count;type:bigint;default:0"`
	LikeCount    int64      `gorm:"column:like_count;type:bigint;default:0"`
	IsTop        bool       `gorm:"column:is_top;type:boolean;default:false"`
	PublishedAt  *time.Time `gorm:"column:published_at"`
	Tags         []TagPO    `gorm:"many2many:articles_tags;foreignKey:id;joinForeignKey:article_id;References:id;joinReferences:tag_id"`
}

func (ArticlePO) TableName() string { return "articles" }

// ArticleTagPO — article.articles_tags 中间表
type ArticleTagPO struct {
	gorm.Model
	ArticleID uint `gorm:"column:article_id;type:bigint;not null"`
	TagID     uint `gorm:"column:tag_id;type:bigint;not null"`
}

func (ArticleTagPO) TableName() string { return "articles_tags" }

// ArticleLikePO — article.articles_likes 点赞表
type ArticleLikePO struct {
	gorm.Model
	ArticleID uint `gorm:"column:article_id;type:bigint;not null;uniqueIndex:idx_article_user"`
	UserID    uint `gorm:"column:user_id;type:bigint;not null;uniqueIndex:idx_article_user"`
}

func (ArticleLikePO) TableName() string { return "articles_likes" }

// =============================================================================
// 缓存常量
// =============================================================================

const (
	keyArticleSlug  = "article:slug:%s" // slug→id 映射缓存: article:slug:{slug}
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
			return biz.ErrSlugAlreadyExists
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

	// B-205: 记录变更前的 slug（UPDATE 后无法再取旧值），slug 变更时用于清除旧映射缓存
	var oldSlug string
	_ = r.data.db.WithContext(ctx).Model(&ArticlePO{}).Select("slug").Where("id = ?", a.ID).Scan(&oldSlug).Error

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
	// B-205: slug 变更时旧 slug→id 映射仍指向本文（缓存 10min），需主动清除
	if oldSlug != "" && oldSlug != a.Slug {
		r.data.cache.Delete(ctx, fmt.Sprintf(keyArticleSlug, oldSlug))
	}
	r.invalidateCache(ctx, a.Slug)
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
	r.invalidateCache(ctx, po.Slug)
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
	a := articlePOToBiz(&po)
	r.enrichAuthorAndCategory(ctx, []*biz.Article{a})
	return a, nil
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
	a := articlePOToBiz(&po)
	r.enrichAuthorAndCategory(ctx, []*biz.Article{a})
	return a, nil
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

	// 标签 AND 过滤：文章须同时拥有全部给定标签。
	// 用相关子查询计数，兼容 MySQL 与 PostgreSQL（原 Group+Having 写法含 PG
	// schema 限定符，在 MySQL 上语法错误，且与 GORM Count 语义冲突）。
	if len(query.Tags) > 0 {
		tagSub := "(SELECT COUNT(DISTINCT t.id) FROM articles_tags at2 " +
			"JOIN tags t ON t.id = at2.tag_id " +
			"WHERE at2.article_id = articles.id AND t.name IN ?)"
		db = db.Where(tagSub+" = ?", query.Tags, len(query.Tags))
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
	r.enrichAuthorAndCategory(ctx, articles)
	return articles, total, nil
}

// =============================================================================
// Search — 关键词搜索（仅已发布文章）
//
// 对标题、摘要、正文做 LIKE 模糊匹配，按发布时间倒序分页返回。
// =============================================================================

func (r *articleRepo) Search(ctx context.Context, keyword string, page, pageSize int) ([]*biz.Article, int64, error) {
	ctx, span := r.data.startSpan(ctx, "ArticleRepo.Search")
	defer span.End()
	span.SetAttributes(attribute.String("keyword", keyword))

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 50 {
		pageSize = 10
	}
	offset := (page - 1) * pageSize

	r.data.log.WithContext(ctx).Debugf("[ArticleRepo.Search] keyword=%q page=%d size=%d", keyword, page, pageSize)

	// LIKE 转义：防止 % _ 通配符干扰
	escaped := strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(keyword)
	pattern := "%" + escaped + "%"

	db := r.data.db.WithContext(ctx).Preload("Tags").
		Where("status = ?", statusToInt("published")).
		// ESCAPE '\\'（SQL 文本）在 MySQL 与 PostgreSQL 中都是合法的单反斜杠转义写法
		Where("title LIKE ? ESCAPE '\\\\' OR excerpt LIKE ? ESCAPE '\\\\' OR content LIKE ? ESCAPE '\\\\'", pattern, pattern, pattern)

	// 统计总数
	var total int64
	if err := db.Model(&ArticlePO{}).Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count search articles: %w", err)
	}
	if total == 0 {
		return nil, 0, nil
	}

	// 分页查询（发布时间倒序）
	var pos []ArticlePO
	if err := db.Order("published_at DESC, id DESC").Offset(offset).Limit(pageSize).Find(&pos).Error; err != nil {
		return nil, 0, fmt.Errorf("search articles: %w", err)
	}

	articles := make([]*biz.Article, 0, len(pos))
	for i := range pos {
		articles = append(articles, articlePOToBiz(&pos[i]))
	}
	r.enrichAuthorAndCategory(ctx, articles)
	return articles, total, nil
}

// =============================================================================
// 计数更新（原子操作）
// =============================================================================

func (r *articleRepo) IncrementViewCount(ctx context.Context, id uint, delta int64) error {
	return r.data.db.WithContext(ctx).Model(&ArticlePO{}).Where("id = ?", id).
		UpdateColumn("view_count", gorm.Expr("view_count + ?", delta)).Error
}

func (r *articleRepo) UpdateTagsArticleCount(ctx context.Context, tagIDs []uint, delta int64) error {
	err := r.data.db.WithContext(ctx).Model(&TagPO{}).Where("id IN ?", tagIDs).
		UpdateColumn("article_count", gorm.Expr("article_count + ?", delta)).Error
	// B-204: 计数变化须失效 tag:all 缓存（tag 列表携带 article_count）
	r.data.cache.Delete(ctx, cacheKeyTagAll)
	return err
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

// =============================================================================
// 点赞（事务保证计数一致性）
// =============================================================================

// InsertLike 点赞（事务：INSERT OR IGNORE → UPDATE like_count IF inserted）
func (r *articleRepo) InsertLike(ctx context.Context, articleID, userID uint) error {
	return r.data.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		like := &ArticleLikePO{ArticleID: articleID, UserID: userID}
		result := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(like)
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
			// B-207: 计数下限保护（防取消点赞把计数打到负数）
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

func (r *articleRepo) invalidateCache(ctx context.Context, slug string) {
	if slug == "" {
		return
	}
	_ = r.data.cache.Delete(ctx, fmt.Sprintf(keyArticleSlug, slug))
}

// =============================================================================
// 实体转换
// =============================================================================

// enrichAuthorAndCategory 批量填充文章的作者显示名/头像与分类名。
//
// auth 与 blog 当前同库部署（同一 MySQL ley 实例），作者（users 表，auth 域）与
// 分类（categories 表）可直查填充；跨域查询失败时仅告警，回退为空的现状行为。
// 若未来分库，此处需改为经 auth RPC 或事件同步作者快照。
func (r *articleRepo) enrichAuthorAndCategory(ctx context.Context, articles []*biz.Article) {
	if len(articles) == 0 {
		return
	}

	authorIDs := make([]uint, 0, len(articles))
	seenAuthor := make(map[uint]struct{}, len(articles))
	for _, a := range articles {
		if a == nil {
			continue
		}
		if _, ok := seenAuthor[a.AuthorID]; !ok {
			seenAuthor[a.AuthorID] = struct{}{}
			authorIDs = append(authorIDs, a.AuthorID)
		}
	}
	if len(authorIDs) > 0 {
		var users []struct {
			ID       uint
			Username string
			Avatar   string
		}
		if err := r.data.db.WithContext(ctx).Table("users").
			Select("id, username, avatar").
			Where("id IN ?", authorIDs).
			Scan(&users).Error; err != nil {
			r.data.log.WithContext(ctx).Warnf("[ArticleRepo.enrich] 查询作者信息失败 err=%v", err)
		} else {
			userMap := make(map[uint]struct {
				Username string
				Avatar   string
			}, len(users))
			for _, u := range users {
				userMap[u.ID] = struct {
					Username string
					Avatar   string
				}{Username: u.Username, Avatar: u.Avatar}
			}
			for _, a := range articles {
				if a == nil {
					continue
				}
				if u, ok := userMap[a.AuthorID]; ok {
					a.AuthorName = u.Username
					a.AuthorAvatar = u.Avatar
				}
			}
		}
	}

	catIDs := make([]uint, 0, len(articles))
	seenCat := make(map[uint]struct{}, len(articles))
	for _, a := range articles {
		if a == nil || a.CategoryID == nil {
			continue
		}
		if _, ok := seenCat[*a.CategoryID]; !ok {
			seenCat[*a.CategoryID] = struct{}{}
			catIDs = append(catIDs, *a.CategoryID)
		}
	}
	if len(catIDs) > 0 {
		var cats []struct {
			ID   uint
			Name string
		}
		if err := r.data.db.WithContext(ctx).Table("categories").
			Select("id, name").
			Where("id IN ?", catIDs).
			Scan(&cats).Error; err != nil {
			r.data.log.WithContext(ctx).Warnf("[ArticleRepo.enrich] 查询分类信息失败 err=%v", err)
		} else {
			catMap := make(map[uint]string, len(cats))
			for _, c := range cats {
				catMap[c.ID] = c.Name
			}
			for _, a := range articles {
				if a != nil && a.CategoryID != nil {
					a.CategoryName = catMap[*a.CategoryID]
				}
			}
		}
	}
}

func articlePOToBiz(po *ArticlePO) *biz.Article {
	a := &biz.Article{
		ID: po.ID, Title: po.Title, Slug: po.Slug, Content: po.Content,
		Excerpt: po.Excerpt, CoverImage: po.CoverImage, Status: biz.ArticleStatus(po.Status),
		AuthorID: po.AuthorID, CategoryID: po.CategoryID, ViewCount: po.ViewCount,
		LikeCount: po.LikeCount, IsTop: po.IsTop,
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
