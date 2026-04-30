package data

import (
	"context"
	"encoding/json"
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
// 文章缓存常量
// =============================================================================

const (
	cacheKeyPostDetail = "post:detail:%s"      // 文章详情缓存键模板，参数为 UUID
	cacheKeyPostList   = "post:list:%d:%d:%s:%d:%s" // 文章列表缓存键模板
	cacheKeyPostSlug   = "post:slug:%s"         // slug→UUID 映射缓存键模板
	cacheKeyPostView   = "post:view:%d"         // 文章浏览量缓存键模板

	cacheTTLPostDetail = 10 * time.Minute // 文章详情缓存过期时间
	cacheTTLPostList   = 3 * time.Minute  // 文章列表缓存过期时间
	cacheTTLPostStale  = 2 * time.Minute  // 空值标记缓存过期时间（防止缓存穿透）
)

// =============================================================================
// 文章相关错误定义
// =============================================================================

var (
	ErrPostNotFound   = errors.New("post not found")       // 文章不存在
	ErrPostSlugExists = errors.New("post slug already exists") // slug 已存在
)

// =============================================================================
// postRepo — PostRepo 接口实现
// =============================================================================

// postRepo PostRepo 接口实现
// 封装文章相关的数据库操作和缓存逻辑，实现 Cache-Aside 模式。
type postRepo struct{ data *Data }

// 编译时接口实现检查
var _ biz.PostRepo = (*postRepo)(nil)

// =============================================================================
// Create — 创建文章
// =============================================================================

// Create 创建文章
// 将文章写入 PostgreSQL，若 slug 冲突则返回 ErrPostSlugExists。
func (r *postRepo) Create(ctx context.Context, post *model.Post) error {
	ctx, span := r.data.StartSpan(ctx, "PostRepo.Create")
	defer span.End()

	// 设置 Span 属性，记录文章标题和 slug
	span.SetAttributes(
		attribute.String("post.title", post.Title),
		attribute.String("post.slug", post.Slug),
	)
	r.data.Log(ctx).Debugw("开始创建文章", "title", post.Title, "slug", post.Slug)

	// 执行数据库插入
	err := r.data.db.WithContext(ctx).Create(post).Error
	if err != nil {
		// 判断是否为唯一键冲突（slug 重复）
		if util.IsUniqueViolation(err) {
			r.data.Log(ctx).Warnw("创建文章失败：slug 重复", "slug", post.Slug, "error", err)
			span.SetAttributes(attribute.Bool("error.duplicate_slug", true))
			return ErrPostSlugExists
		}
		r.data.Log(ctx).Errorw("创建文章数据库写入失败", "title", post.Title, "error", err)
		return fmt.Errorf("create post: %w", err)
	}

	r.data.Log(ctx).Infow("文章创建成功", "id", post.ID, "uuid", post.UUID, "slug", post.Slug)
	return nil
}

// =============================================================================
// Update — 更新文章（使用 map 避免零值跳过）
// =============================================================================

// Update 更新文章（使用 map 避免零值跳过）
// GORM 的 Updates 在使用 struct 时会忽略零值字段，因此采用 map[string]interface{} 确保零值也能更新。
func (r *postRepo) Update(ctx context.Context, post *model.Post) error {
	ctx, span := r.data.StartSpan(ctx, "PostRepo.Update")
	defer span.End()

	span.SetAttributes(attribute.Int("post.id", int(post.ID)))
	r.data.Log(ctx).Debugw("开始更新文章", "id", post.ID, "title", post.Title)

	// 使用 map 更新，避免 GORM struct 零值跳过问题
	result := r.data.db.WithContext(ctx).
		Model(&model.Post{}).
		Where("id = ?", post.ID).
		Updates(map[string]interface{}{
			"title":       post.Title,
			"slug":        post.Slug,
			"content":     post.Content,
			"excerpt":     post.Excerpt,
			"cover_image": post.CoverImage,
			"status":      post.Status,
			"category_id": post.CategoryID,
			"is_top":      post.IsTop,
		})

	if result.Error != nil {
		r.data.Log(ctx).Errorw("更新文章数据库操作失败", "id", post.ID, "error", result.Error)
		return fmt.Errorf("update post: %w", result.Error)
	}
	// 检查是否有行被实际更新
	if result.RowsAffected == 0 {
		r.data.Log(ctx).Debugw("更新文章未找到记录", "id", post.ID)
		return ErrPostNotFound
	}

	// 清除文章相关缓存，保证后续读取能拿到最新数据
	r.invalidatePostCache(ctx, post.ID, post.UUID, post.Slug)
	r.data.Log(ctx).Infow("文章更新成功", "id", post.ID)
	return nil
}

// =============================================================================
// Delete — 软删除文章
// =============================================================================

// Delete 软删除文章
// 先在数据库中标记删除，再清除相关缓存。
func (r *postRepo) Delete(ctx context.Context, id uint) error {
	ctx, span := r.data.StartSpan(ctx, "PostRepo.Delete")
	defer span.End()

	span.SetAttributes(attribute.Int("post.id", int(id)))
	r.data.Log(ctx).Debugw("开始删除文章", "id", id)

	// 先查询文章以便获取 UUID 和 slug 用于缓存清除
	post, err := r.FindByID(ctx, id)
	if err != nil {
		r.data.Log(ctx).Debugw("删除前查找文章失败", "id", id, "error", err)
		return err
	}

	// 执行软删除（GORM 自动设置 deleted_at）
	result := r.data.db.WithContext(ctx).Where("id = ?", id).Delete(&model.Post{})
	if result.Error != nil {
		r.data.Log(ctx).Errorw("删除文章数据库操作失败", "id", id, "error", result.Error)
		return fmt.Errorf("delete post: %w", result.Error)
	}

	// 清除该文章的所有相关缓存键
	r.invalidatePostCache(ctx, id, post.UUID, post.Slug)
	r.data.Log(ctx).Infow("文章删除成功", "id", id)
	return nil
}

// =============================================================================
// FindByID — 按主键查询（不走缓存）
// =============================================================================

// FindByID 按主键查询（不走缓存）
// 直接查询数据库，预加载关联标签。
func (r *postRepo) FindByID(ctx context.Context, id uint) (*model.Post, error) {
	ctx, span := r.data.StartSpan(ctx, "PostRepo.FindByID")
	defer span.End()

	span.SetAttributes(attribute.Int("post.id", int(id)))
	r.data.Log(ctx).Debugw("按主键查询文章", "id", id)

	var post model.Post
	// 预加载 Tags 关联，按主键查询第一条记录
	err := r.data.db.WithContext(ctx).Preload("Tags").Where("id = ?", id).First(&post).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			r.data.Log(ctx).Debugw("按主键查询文章未找到", "id", id)
			return nil, ErrPostNotFound
		}
		r.data.Log(ctx).Errorw("按ID查询文章失败", "id", id, "error", err)
		return nil, fmt.Errorf("find post by id: %w", err)
	}
	r.data.Log(ctx).Debugw("按主键查询文章成功", "id", id, "uuid", post.UUID)
	return &post, nil
}

// =============================================================================
// FindByUUID — 按 UUID 查询（Cache-Aside）
// =============================================================================

// FindByUUID 按 UUID 查询（Cache-Aside 模式）
// 1. 先查缓存，命中则直接返回
// 2. 缓存未命中则查询数据库
// 3. 数据库查询结果回填缓存
// 4. 记录不存在时写入空值标记，防止缓存穿透
func (r *postRepo) FindByUUID(ctx context.Context, uuid string) (*model.Post, error) {
	ctx, span := r.data.StartSpan(ctx, "PostRepo.FindByUUID")
	defer span.End()

	span.SetAttributes(attribute.String("post.uuid", uuid))
	cacheKey := fmt.Sprintf(cacheKeyPostDetail, uuid)
	r.data.Log(ctx).Debugw("按UUID查询文章，尝试读取缓存", "uuid", uuid, "cache_key", cacheKey)

	// 第一步：尝试从缓存读取
	post, err := r.readFromCache(ctx, cacheKey)
	if err == nil && post != nil {
		r.data.Log(ctx).Debugw("文章缓存命中", "uuid", uuid)
		return post, nil
	}
	// 缓存读取异常但非致命错误，记录日志后回退到数据库
	if err != nil && !errors.Is(err, cache.ErrKeyNotFound) && !errors.Is(err, ErrPostNotFound) {
		r.data.Log(ctx).Warnw("文章缓存读取异常，回退到数据库查询", "uuid", uuid, "error", err)
	}
	// 空值标记命中，说明数据库中也不存在该文章
	if errors.Is(err, ErrPostNotFound) {
		r.data.Log(ctx).Debugw("文章空值缓存标记命中", "uuid", uuid)
		return nil, ErrPostNotFound
	}
	r.data.Log(ctx).Debugw("文章缓存未命中，查询数据库", "uuid", uuid)

	// 第二步：缓存未命中或可忽略，查询数据库
	var m model.Post
	dbErr := r.data.db.WithContext(ctx).Preload("Tags").Where("uuid = ?", uuid).First(&m).Error
	if dbErr != nil {
		if errors.Is(dbErr, gorm.ErrRecordNotFound) {
			// 数据库中也不存在，写入空值缓存标记以防止缓存穿透
			r.data.Log(ctx).Debugw("数据库未找到文章，写入空值缓存标记", "uuid", uuid)
			_ = r.data.cache.Set(ctx, cacheKey, []byte("null"), cacheTTLPostStale)
			return nil, ErrPostNotFound
		}
		r.data.Log(ctx).Errorw("按UUID查询文章数据库失败", "uuid", uuid, "error", dbErr)
		return nil, fmt.Errorf("find post by uuid: %w", dbErr)
	}

	// 第三步：将数据库查询结果回填缓存
	r.data.Log(ctx).Debugw("数据库查询文章成功，回填缓存", "uuid", uuid, "id", m.ID)
	if setErr := r.data.cache.Set(ctx, cacheKey, &m, cacheTTLPostDetail); setErr != nil {
		r.data.Log(ctx).Warnw("文章缓存写入失败", "uuid", uuid, "error", setErr)
	}

	span.SetAttributes(attribute.String("post.title", m.Title), attribute.String("post.slug", m.Slug))
	return &m, nil
}

// =============================================================================
// FindBySlug — 按 Slug 查询
// =============================================================================

// FindBySlug 按 Slug 查询
// 先查 slug→UUID 映射缓存，再调用 FindByUUID 获取完整文章数据。
func (r *postRepo) FindBySlug(ctx context.Context, slug string) (*model.Post, error) {
	ctx, span := r.data.StartSpan(ctx, "PostRepo.FindBySlug")
	defer span.End()

	span.SetAttributes(attribute.String("post.slug", slug))
	slugCacheKey := fmt.Sprintf(cacheKeyPostSlug, slug)
	r.data.Log(ctx).Debugw("按Slug查询文章，尝试读取Slug缓存", "slug", slug, "cache_key", slugCacheKey)

	// 第一步：查询 slug→UUID 映射缓存
	uuidBytes, err := r.data.cache.Get(ctx, slugCacheKey)
	if err == nil {
		uuid := string(uuidBytes)
		// 空值标记，表示该 slug 对应的文章不存在
		if uuid == "" || uuid == "null" {
			r.data.Log(ctx).Debugw("Slug空值缓存标记命中", "slug", slug)
			return nil, ErrPostNotFound
		}
		// 缓存命中，跳转到 FindByUUID 获取完整文章
		r.data.Log(ctx).Debugw("Slug缓存命中，跳转到UUID查询", "slug", slug, "uuid", uuid)
		span.SetAttributes(attribute.String("resolved.uuid", uuid))
		return r.FindByUUID(ctx, uuid)
	}
	if !errors.Is(err, cache.ErrKeyNotFound) {
		r.data.Log(ctx).Warnw("Slug缓存读取异常，回退到数据库查询", "slug", slug, "error", err)
	}
	r.data.Log(ctx).Debugw("Slug缓存未命中，查询数据库", "slug", slug)

	// 第二步：缓存未命中，查询数据库
	var m model.Post
	dbErr := r.data.db.WithContext(ctx).Preload("Tags").Where("slug = ?", slug).First(&m).Error
	if dbErr != nil {
		if errors.Is(dbErr, gorm.ErrRecordNotFound) {
			r.data.Log(ctx).Debugw("数据库未找到Slug对应的文章", "slug", slug)
			return nil, ErrPostNotFound
		}
		r.data.Log(ctx).Errorw("按Slug查询文章数据库失败", "slug", slug, "error", dbErr)
		return nil, fmt.Errorf("find post by slug: %w", dbErr)
	}

	// 第三步：回填缓存（slug→UUID 映射 + 文章详情）
	r.data.Log(ctx).Debugw("数据库查询Slug成功，回填缓存", "slug", slug, "uuid", m.UUID, "id", m.ID)
	if setErr := r.data.cache.Set(ctx, slugCacheKey, m.UUID, cacheTTLPostDetail); setErr != nil {
		r.data.Log(ctx).Warnw("Slug缓存写入失败", "slug", slug, "error", setErr)
	}
	if setErr := r.data.cache.Set(ctx, fmt.Sprintf(cacheKeyPostDetail, m.UUID), &m, cacheTTLPostDetail); setErr != nil {
		r.data.Log(ctx).Warnw("文章详情缓存写入失败", "uuid", m.UUID, "error", setErr)
	}

	return &m, nil
}

// =============================================================================
// List — 分页查询文章列表
// =============================================================================

// List 分页查询文章列表
// 支持按状态、分类、作者、标签过滤，以及排序、分页。
func (r *postRepo) List(ctx context.Context, query biz.PostListQuery) ([]*model.Post, int64, error) {
	ctx, span := r.data.StartSpan(ctx, "PostRepo.List")
	defer span.End()

	span.SetAttributes(
		attribute.Int("page", query.Page),
		attribute.Int("page_size", query.PageSize),
		attribute.String("status", query.Status),
	)
	r.data.Log(ctx).Debugw("开始分页查询文章列表",
		"status", query.Status, "category_id", query.CategoryID,
		"tags", query.Tags, "author_id", query.AuthorID,
		"page", query.Page, "page_size", query.PageSize,
		"sort_by", query.SortBy, "sort_order", query.SortOrder,
	)

	// 页码校验：最小为第1页
	if query.Page < 1 {
		query.Page = 1
	}
	// 每页数量校验：范围 [1, MaxPageSize]，默认20
	if query.PageSize < 1 || query.PageSize > util.MaxPageSize {
		query.PageSize = 20
	}

	// 构建基础查询，预加载标签
	db := r.data.db.WithContext(ctx).Preload("Tags")

	// 按文章状态过滤
	if query.Status != "" {
		db = db.Where("status = ?", postStatusToInt(query.Status))
		r.data.Log(ctx).Debugw("列表过滤条件：状态", "status", query.Status)
	}
	// 按分类过滤
	if query.CategoryID != nil {
		db = db.Where("category_id = ?", *query.CategoryID)
		r.data.Log(ctx).Debugw("列表过滤条件：分类ID", "category_id", *query.CategoryID)
	}
	// 按作者过滤
	if query.AuthorID != nil {
		db = db.Where("author_id = ?", *query.AuthorID)
		r.data.Log(ctx).Debugw("列表过滤条件：作者ID", "author_id", *query.AuthorID)
	}
	// 按标签过滤（AND 逻辑，需要同时拥有所有指定标签）
	if len(query.Tags) > 0 {
		db = db.Joins("JOIN \"post\".posts_tags pt ON pt.post_id = \"post\".posts.id").
			Joins("JOIN \"post\".tags t ON t.id = pt.tag_id").
			Where("t.name IN ?", query.Tags).
			Group("\"post\".posts.id").
			Having("COUNT(DISTINCT t.id) = ?", len(query.Tags))
		r.data.Log(ctx).Debugw("列表过滤条件：标签", "tags", query.Tags, "expected_count", len(query.Tags))
	}

	// 计数查询：统计符合条件的文章总数
	var total int64
	if err := db.Model(&model.Post{}).Count(&total).Error; err != nil {
		r.data.Log(ctx).Errorw("统计文章总数失败", "error", err)
		return nil, 0, fmt.Errorf("count posts: %w", err)
	}
	r.data.Log(ctx).Debugw("文章列表总数统计完成", "total", total)

	// 无结果则直接返回
	if total == 0 {
		r.data.Log(ctx).Debugw("文章列表无结果", "total", 0)
		span.SetAttributes(attribute.Int64("total", 0))
		return nil, 0, nil
	}

	// 构建排序子句并执行分页查询
	orderClause := buildPostOrderClause(query.SortBy, query.SortOrder)
	offset := (query.Page - 1) * query.PageSize
	var posts []*model.Post
	if err := db.Order(orderClause).Offset(offset).Limit(query.PageSize).Find(&posts).Error; err != nil {
		r.data.Log(ctx).Errorw("分页查询文章列表失败", "error", err)
		return nil, 0, fmt.Errorf("list posts: %w", err)
	}

	span.SetAttributes(attribute.Int64("total", total))
	r.data.Log(ctx).Debugw("文章列表查询结果",
		"返回条数", len(posts), "总数", total, "当前页", query.Page,
	)
	return posts, total, nil
}

// =============================================================================
// Search — 全文搜索
// =============================================================================

// Search 全文搜索
// 使用 PostgreSQL 全文搜索（tsvector/tsquery）对文章进行全文检索。
func (r *postRepo) Search(ctx context.Context, keyword string, page, pageSize int) ([]*model.Post, int64, error) {
	ctx, span := r.data.StartSpan(ctx, "PostRepo.Search")
	defer span.End()

	span.SetAttributes(
		attribute.String("keyword", keyword),
		attribute.Int("page", page),
		attribute.Int("page_size", pageSize),
	)
	r.data.Log(ctx).Debugw("开始全文搜索文章", "keyword", keyword, "page", page, "page_size", pageSize)

	// 页码和每页数量校验
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > util.MaxPageSize {
		pageSize = 20
	}

	// 构建全文搜索 SQL 片段：
	// ts_rank 计算文本相关性排名
	// plainto_tsquery 将关键词转换为全文搜索查询
	rankSQL := "ts_rank(search_vector, plainto_tsquery('simple', ?))"
	condSQL := "search_vector @@ plainto_tsquery('simple', ?) AND status = 1 AND deleted_at IS NULL"

	db := r.data.db.WithContext(ctx).Preload("Tags")

	// 先统计搜索结果总数
	var total int64
	if err := db.Model(&model.Post{}).Where(condSQL, keyword).Count(&total).Error; err != nil {
		r.data.Log(ctx).Errorw("搜索文章总数统计失败", "keyword", keyword, "error", err)
		return nil, 0, fmt.Errorf("search count: %w", err)
	}
	r.data.Log(ctx).Debugw("搜索文章总数统计完成", "keyword", keyword, "total", total)

	// 无结果则直接返回
	if total == 0 {
		r.data.Log(ctx).Debugw("全文搜索无结果", "keyword", keyword)
		return nil, 0, nil
	}

	// 执行分页搜索查询，按相关性排名降序
	var posts []*model.Post
	offset := (page - 1) * pageSize
	if err := db.Select("*, "+rankSQL+" AS rank", keyword).
		Where(condSQL, keyword).
		Order("rank DESC").Offset(offset).Limit(pageSize).
		Find(&posts).Error; err != nil {
		r.data.Log(ctx).Errorw("全文搜索文章失败", "keyword", keyword, "error", err)
		return nil, 0, fmt.Errorf("search posts: %w", err)
	}

	r.data.Log(ctx).Infow("文章全文搜索完成",
		"keyword", keyword, "total", total, "returned", len(posts),
	)
	return posts, total, nil
}

// =============================================================================
// IncrementViewCount — 原子增加浏览计数
// =============================================================================

// IncrementViewCount 原子增加浏览计数
// 使用 SQL 表达式 view_count + ? 确保并发安全。
func (r *postRepo) IncrementViewCount(ctx context.Context, id uint, delta int64) error {
	ctx, span := r.data.StartSpan(ctx, "PostRepo.IncrementViewCount")
	defer span.End()

	span.SetAttributes(attribute.Int("post.id", int(id)), attribute.Int64("delta", delta))
	r.data.Log(ctx).Debugw("原子增加文章浏览计数", "id", id, "delta", delta)

	// 使用 gorm.Expr 执行原子更新，避免读写竞争
	result := r.data.db.WithContext(ctx).Model(&model.Post{}).
		Where("id = ?", id).UpdateColumn("view_count", gorm.Expr("view_count + ?", delta))

	if result.Error != nil {
		r.data.Log(ctx).Errorw("增加文章浏览计数失败", "id", id, "delta", delta, "error", result.Error)
		return fmt.Errorf("increment view count: %w", result.Error)
	}

	r.data.Log(ctx).Debugw("文章浏览计数增加成功", "id", id, "delta", delta)
	return nil
}

// =============================================================================
// AssociateTags — 添加标签关联（幂等）
// =============================================================================

// AssociateTags 添加标签关联（幂等）
// 使用 ON CONFLICT DO NOTHING 确保重复关联不会报错。
func (r *postRepo) AssociateTags(ctx context.Context, postID uint, tagIDs []uint) error {
	ctx, span := r.data.StartSpan(ctx, "PostRepo.AssociateTags")
	defer span.End()

	span.SetAttributes(attribute.Int("post.id", int(postID)), attribute.Int("tag_count", len(tagIDs)))
	r.data.Log(ctx).Debugw("开始关联文章标签", "post_id", postID, "tag_count", len(tagIDs), "tag_ids", tagIDs)

	// 无标签则直接返回
	if len(tagIDs) == 0 {
		r.data.Log(ctx).Debugw("标签列表为空，跳过关联", "post_id", postID)
		return nil
	}

	// 构建 post_tags 中间表记录
	postTags := make([]*model.PostTag, 0, len(tagIDs))
	for _, tagID := range tagIDs {
		postTags = append(postTags, &model.PostTag{PostID: postID, TagID: tagID})
	}

	// ON CONFLICT DO NOTHING：主键冲突时静默忽略，保证幂等性
	err := r.data.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "post_id"}, {Name: "tag_id"}},
		DoNothing: true,
	}).Create(&postTags).Error
	if err != nil {
		r.data.Log(ctx).Errorw("关联文章标签失败", "post_id", postID, "tag_ids", tagIDs, "error", err)
		return fmt.Errorf("associate tags: %w", err)
	}
	r.data.Log(ctx).Debugw("关联文章标签成功", "post_id", postID, "tag_count", len(tagIDs))
	return nil
}

// =============================================================================
// SyncTags — 全量替换标签关联
// =============================================================================

// SyncTags 全量替换标签关联
// 在事务中执行：先删除旧关联，再插入新关联。
func (r *postRepo) SyncTags(ctx context.Context, postID uint, tagIDs []uint) error {
	ctx, span := r.data.StartSpan(ctx, "PostRepo.SyncTags")
	defer span.End()

	span.SetAttributes(attribute.Int("post.id", int(postID)), attribute.Int("tag_count", len(tagIDs)))
	r.data.Log(ctx).Debugw("开始全量替换文章标签", "post_id", postID, "new_tag_ids", tagIDs)

	// 在事务中执行，确保删除和插入的原子性
	return r.data.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 第一步：删除所有旧的标签关联
		if err := tx.Where("post_id = ?", postID).Delete(&model.PostTag{}).Error; err != nil {
			r.data.Log(ctx).Errorw("删除旧标签关联失败", "post_id", postID, "error", err)
			return fmt.Errorf("sync tags delete old: %w", err)
		}
		r.data.Log(ctx).Debugw("已删除旧标签关联", "post_id", postID)

		// 第二步：如果没有新标签，事务结束
		if len(tagIDs) == 0 {
			r.data.Log(ctx).Debugw("新标签列表为空，仅删除旧关联", "post_id", postID)
			return nil
		}

		// 第三步：插入新的标签关联
		postTags := make([]*model.PostTag, 0, len(tagIDs))
		for _, tagID := range tagIDs {
			postTags = append(postTags, &model.PostTag{PostID: postID, TagID: tagID})
		}
		if err := tx.Create(&postTags).Error; err != nil {
			r.data.Log(ctx).Errorw("插入新标签关联失败", "post_id", postID, "error", err)
			return fmt.Errorf("sync tags insert new: %w", err)
		}
		r.data.Log(ctx).Debugw("全量替换文章标签成功", "post_id", postID, "tag_count", len(tagIDs))
		return nil
	})
}

// =============================================================================
// ListTagsByPostID — 获取文章所有标签
// =============================================================================

// ListTagsByPostID 获取文章所有标签
// 通过 posts_tags 中间表 JOIN 查询文章关联的所有标签。
func (r *postRepo) ListTagsByPostID(ctx context.Context, postID uint) ([]*model.Tag, error) {
	var tags []*model.Tag
	err := r.data.db.WithContext(ctx).
		Joins("JOIN \"post\".posts_tags pt ON pt.tag_id = \"post\".tags.id").
		Where("pt.post_id = ?", postID).Find(&tags).Error
	if err != nil {
		return nil, fmt.Errorf("list tags by post id: %w", err)
	}
	return tags, nil
}

// =============================================================================
// invalidatePostCache — 清除文章相关缓存
// =============================================================================

// invalidatePostCache 清除文章相关缓存
// 当文章数据发生变更时，清除文章详情缓存、浏览量缓存和 slug 映射缓存。
func (r *postRepo) invalidatePostCache(ctx context.Context, id uint, uuid string, slug string) {
	// 构建需要清除的缓存键列表
	cacheKeys := []string{
		fmt.Sprintf(cacheKeyPostDetail, uuid), // 文章详情缓存
		fmt.Sprintf(cacheKeyPostView, id),      // 浏览量缓存
	}
	if slug != "" {
		cacheKeys = append(cacheKeys, fmt.Sprintf(cacheKeyPostSlug, slug)) // slug 映射缓存
	}
	r.data.Log(ctx).Debugw("开始清除文章相关缓存", "id", id, "uuid", uuid, "slug", slug)

	// 逐个删除缓存键
	for _, key := range cacheKeys {
		if err := r.data.cache.Delete(ctx, key); err != nil {
			r.data.Log(ctx).Warnw("删除文章缓存失败", "key", key, "error", err)
		}
	}
}

// =============================================================================
// readFromCache — 从缓存读取文章，处理空值标记
// =============================================================================

// readFromCache 从缓存读取文章，处理空值标记
// 1. 缓存未命中返回 ErrKeyNotFound
// 2. 缓存值为 "null" 表示空值标记（缓存穿透保护）
// 3. 缓存值正常则反序列化为 model.Post
func (r *postRepo) readFromCache(ctx context.Context, cacheKey string) (*model.Post, error) {
	data, err := r.data.cache.Get(ctx, cacheKey)
	if err != nil {
		return nil, err
	}
	// 空值标记：表示数据库中没有该记录，防止缓存穿透
	if string(data) == "null" {
		return nil, ErrPostNotFound
	}
	// 正常反序列化
	var post model.Post
	if err := json.Unmarshal(data, &post); err != nil {
		return nil, fmt.Errorf("cache deserialize post: %w", err)
	}
	return &post, nil
}

// =============================================================================
// Helper 函数
// =============================================================================

// postStatusToInt 将文章状态字符串转换为数据库中的 int8 值
// draft → 0, published → 1, archived → 2
func postStatusToInt(status string) int8 {
	switch status {
	case "published":
		return 1
	case "archived":
		return 2
	default:
		return 0 // draft 或未知状态
	}
}

// buildPostOrderClause 构建排序 SQL 子句
// 根据 sortBy 和 sortOrder 参数生成 ORDER BY 子句，
// is_top 排序时优先置顶文章。
func buildPostOrderClause(sortBy, sortOrder string) string {
	var column string
	switch sortBy {
	case "created_at":
		column = "created_at"
	case "updated_at":
		column = "updated_at"
	case "published_at":
		column = "published_at"
	case "view_count":
		column = "view_count"
	case "is_top":
		column = "is_top DESC, published_at" // 置顶文章优先，其次按发布时间
	default:
		column = "created_at" // 默认按创建时间排序
	}
	// 追加排序方向
	if sortOrder == "asc" {
		column += " ASC"
	} else {
		column += " DESC"
	}
	return column
}
