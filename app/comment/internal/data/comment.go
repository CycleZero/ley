package data

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"ley/app/comment/internal/biz"
	"ley/app/comment/internal/model"
	"ley/pkg/cache"
	"ley/pkg/util"

	"go.opentelemetry.io/otel/attribute"
	"gorm.io/gorm"
)

// =============================================================================
// 缓存键模板与 TTL 常量
// =============================================================================

const (
	// cacheKeyCommentList 评论列表缓存键模板，格式: comment:list:{postID}:{page}:{pageSize}
	cacheKeyCommentList  = "comment:list:%d:%d:%d"
	// cacheKeyCommentCount 评论计数缓存键模板，格式: comment:count:{postID}
	cacheKeyCommentCount = "comment:count:%d"
	// cacheTTLCommentList 评论列表缓存 TTL，2 分钟
	cacheTTLCommentList  = 2 * time.Minute
	// cacheTTLCommentCount 评论计数缓存 TTL，5 分钟
	cacheTTLCommentCount = 5 * time.Minute
	// cacheTTLCommentStale 空列表防穿透缓存 TTL，1 分钟
	cacheTTLCommentStale = 1 * time.Minute
)

var (
	// ErrCommentNotFound 评论未找到错误
	ErrCommentNotFound = errors.New("comment not found")
)

// commentListCache 评论列表缓存结构体
// 序列化为 JSON 后存储至缓存中
type commentListCache struct {
	Total    int64             `json:"total"`    // 该文章符合条件的评论总数
	Comments []*model.Comment  `json:"comments"` // 当前页的评论列表
}

// =============================================================================
// commentRepo — CommentRepo 接口实现（Cache-Aside 模式）
// =============================================================================

// commentRepo 是 biz.CommentRepo 的具体实现
// 负责评论的 CRUD 操作、缓存管理与查询优化
type commentRepo struct{ data *Data }

// 编译期接口实现检查：确保 *commentRepo 实现了 biz.CommentRepo
var _ biz.CommentRepo = (*commentRepo)(nil)

// =============================================================================
// 评论 CRUD 操作
// =============================================================================

// Create 创建评论
// 1) 向数据库插入新评论记录
// 2) 插入成功后使该文章的所有缓存失效
// 3) 记录创建成功日志
func (r *commentRepo) Create(ctx context.Context, comment *model.Comment) error {
	ctx, span := r.data.StartSpan(ctx, "CommentRepo.Create")
	defer span.End()

	// 在 Span 上记录关键业务属性：文章 ID 和作者 ID
	span.SetAttributes(attribute.Int("post_id", int(comment.PostID)), attribute.Int("author_id", int(comment.AuthorID)))

	r.data.Log(ctx).Debugw("开始创建评论",
		"post_id", comment.PostID, "author_id", comment.AuthorID, "parent_id", comment.ParentID,
	)

	// 执行数据库插入
	err := r.data.db.WithContext(ctx).Create(comment).Error
	if err != nil {
		r.data.Log(ctx).Errorw("创建评论失败",
			"post_id", comment.PostID, "author_id", comment.AuthorID, "错误", err,
		)
		return fmt.Errorf("create comment: %w", err)
	}

	// 插入成功后使该文章的所有相关缓存失效
	r.invalidateCommentCache(ctx, comment.PostID)

	r.data.Log(ctx).Infow("评论创建成功",
		"id", comment.ID, "uuid", comment.UUID, "post_id", comment.PostID,
	)
	return nil
}

// Update 更新评论内容
// 仅更新 content 和 updated_at 字段，防止误改其他字段
// 若无行被影响则返回 ErrCommentNotFound
func (r *commentRepo) Update(ctx context.Context, comment *model.Comment) error {
	ctx, span := r.data.StartSpan(ctx, "CommentRepo.Update")
	defer span.End()

	span.SetAttributes(attribute.Int("comment.id", int(comment.ID)))

	r.data.Log(ctx).Debugw("开始更新评论", "id", comment.ID, "post_id", comment.PostID)

	// 使用 Updates(map) 仅更新 content 和 updated_at，避免零值覆盖问题
	result := r.data.db.WithContext(ctx).Model(comment).Where("id = ?", comment.ID).
		Updates(map[string]interface{}{"content": comment.Content, "updated_at": time.Now()})

	if result.Error != nil {
		r.data.Log(ctx).Errorw("更新评论失败", "id", comment.ID, "错误", result.Error)
		return fmt.Errorf("update comment: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		r.data.Log(ctx).Debugw("未找到要更新的评论", "id", comment.ID)
		return ErrCommentNotFound
	}

	// 更新后使缓存失效
	r.invalidateCommentCache(ctx, comment.PostID)
	r.data.Log(ctx).Infow("评论更新成功", "id", comment.ID)
	return nil
}

// Delete 软删除评论
// 先查询评论以获取 PostID（用于失效缓存），再执行 GORM 软删除
func (r *commentRepo) Delete(ctx context.Context, id uint) error {
	ctx, span := r.data.StartSpan(ctx, "CommentRepo.Delete")
	defer span.End()

	span.SetAttributes(attribute.Int("comment.id", int(id)))

	r.data.Log(ctx).Debugw("开始删除评论", "id", id)

	// 先查询评论，获取 PostID 用于后续缓存失效
	comment, err := r.FindByID(ctx, id)
	if err != nil {
		r.data.Log(ctx).Debugw("删除前查询评论失败", "id", id, "错误", err)
		return err
	}

	// GORM 软删除：设置 deleted_at 时间戳
	result := r.data.db.WithContext(ctx).Where("id = ?", id).Delete(&model.Comment{})
	if result.Error != nil {
		r.data.Log(ctx).Errorw("删除评论失败", "id", id, "错误", result.Error)
		return fmt.Errorf("delete comment: %w", result.Error)
	}

	// 删除后使缓存失效
	r.invalidateCommentCache(ctx, comment.PostID)
	r.data.Log(ctx).Infow("评论删除成功", "id", id)
	return nil
}

// =============================================================================
// 评论查询操作
// =============================================================================

// FindByID 按主键查询评论
// 若未找到记录返回 ErrCommentNotFound，其他数据库错误向上层包装返回
func (r *commentRepo) FindByID(ctx context.Context, id uint) (*model.Comment, error) {
	r.data.Log(ctx).Debugw("按ID查询评论", "id", id)

	var comment model.Comment
	err := r.data.db.WithContext(ctx).Where("id = ?", id).First(&comment).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			r.data.Log(ctx).Debugw("按ID查询评论未找到", "id", id)
			return nil, ErrCommentNotFound
		}
		r.data.Log(ctx).Errorw("按ID查询评论失败", "id", id, "错误", err)
		return nil, fmt.Errorf("find comment by id: %w", err)
	}
	r.data.Log(ctx).Debugw("按ID查询评论成功", "id", comment.ID, "post_id", comment.PostID)
	return &comment, nil
}

// FindByUUID 按 UUID 查询评论
// UUID 是业务唯一标识，用于对外暴露的 API 查询
func (r *commentRepo) FindByUUID(ctx context.Context, uuid string) (*model.Comment, error) {
	r.data.Log(ctx).Debugw("按UUID查询评论", "uuid", uuid)

	var comment model.Comment
	err := r.data.db.WithContext(ctx).Where("uuid = ?", uuid).First(&comment).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			r.data.Log(ctx).Debugw("按UUID查询评论未找到", "uuid", uuid)
			return nil, ErrCommentNotFound
		}
		r.data.Log(ctx).Errorw("按UUID查询评论失败", "uuid", uuid, "错误", err)
		return nil, fmt.Errorf("find comment by uuid: %w", err)
	}
	r.data.Log(ctx).Debugw("按UUID查询评论成功", "id", comment.ID, "uuid", uuid)
	return &comment, nil
}

// =============================================================================
// 评论列表查询（Cache-Aside 模式）
// =============================================================================

// ListByPost 分页查询文章已通过审核的评论（Cache-Aside 模式）
// 1) 对 page/pageSize 做边界保护
// 2) 尝试从缓存读取评论列表
// 3) 缓存未命中则回源数据库查询，并回写缓存
// 返回: 评论列表, 总评论数, 错误
func (r *commentRepo) ListByPost(ctx context.Context, postID uint, page, pageSize int) ([]*model.Comment, int64, error) {
	ctx, span := r.data.StartSpan(ctx, "CommentRepo.ListByPost")
	defer span.End()

	span.SetAttributes(
		attribute.Int("post_id", int(postID)),
		attribute.Int("page", page), attribute.Int("page_size", pageSize),
	)

	r.data.Log(ctx).Debugw("开始分页查询评论列表",
		"post_id", postID, "page", page, "page_size", pageSize,
	)

	// 边界保护：page 最小为 1
	if page < 1 {
		page = 1
		r.data.Log(ctx).Debugw("页码参数修正为最小值", "post_id", postID, "corrected_page", page)
	}
	// 边界保护：pageSize 在 1 到 MaxPageSize 之间，默认 20
	if pageSize < 1 || pageSize > util.MaxPageSize {
		originalPageSize := pageSize
		pageSize = 20
		r.data.Log(ctx).Debugw("页大小参数超出范围，使用默认值",
			"post_id", postID, "original_page_size", originalPageSize, "corrected_page_size", pageSize,
		)
	}

	// 构建缓存键
	cacheKey := fmt.Sprintf(cacheKeyCommentList, postID, page, pageSize)
	r.data.Log(ctx).Debugw("查询评论列表缓存", "cache_key", cacheKey, "post_id", postID)

	// 尝试从缓存读取
	comments, total, err := r.readListCache(ctx, cacheKey)
	if err == nil {
		// 缓存命中：直接返回
		r.data.Log(ctx).Debugw("评论列表缓存命中",
			"post_id", postID, "page", page, "total", total, "returned", len(comments),
		)
		span.SetAttributes(attribute.Int64("cache_hit", 1))
		return comments, total, nil
	}
	if !errors.Is(err, cache.ErrKeyNotFound) {
		// 缓存读取异常（非 "键不存在"），打印警告并回源数据库
		r.data.Log(ctx).Warnw("评论列表缓存读取异常，回源数据库查询",
			"cache_key", cacheKey, "错误", err,
		)
	} else {
		r.data.Log(ctx).Debugw("评论列表缓存未命中，回源数据库查询",
			"cache_key", cacheKey, "post_id", postID,
		)
	}
	span.SetAttributes(attribute.Int64("cache_hit", 0))

	// 回源数据库：查询已通过审核的评论
	db := r.data.db.WithContext(ctx).
		Where("post_id = ? AND status = ?", postID, model.CommentStatusApproved)

	// 先统计总数
	r.data.Log(ctx).Debugw("数据库统计评论总数", "post_id", postID)
	if countErr := db.Model(&model.Comment{}).Count(&total).Error; countErr != nil {
		r.data.Log(ctx).Errorw("统计文章评论数失败", "post_id", postID, "错误", countErr)
		return nil, 0, fmt.Errorf("count comments: %w", countErr)
	}
	r.data.Log(ctx).Debugw("数据库评论总数统计完成", "post_id", postID, "total", total)

	// 再分页查询
	offset := (page - 1) * pageSize
	r.data.Log(ctx).Debugw("数据库分页查询评论", "post_id", postID, "offset", offset, "limit", pageSize)
	if findErr := db.Order("created_at ASC").Offset(offset).Limit(pageSize).Find(&comments).Error; findErr != nil {
		r.data.Log(ctx).Errorw("分页查询评论列表失败", "post_id", postID, "错误", findErr)
		return nil, 0, fmt.Errorf("list comments: %w", findErr)
	}
	r.data.Log(ctx).Debugw("数据库分页查询评论完成",
		"post_id", postID, "returned", len(comments), "total", total,
	)

	// 回写缓存
	r.writeListCache(ctx, cacheKey, comments, total)

	span.SetAttributes(attribute.Int64("total", total), attribute.Int("returned", len(comments)))
	return comments, total, nil
}

// ListChildren 查询直接子评论
// 查询 parent_id 等于指定值且审核通过的评论，按创建时间升序排列
func (r *commentRepo) ListChildren(ctx context.Context, parentID uint) ([]*model.Comment, error) {
	r.data.Log(ctx).Debugw("查询子评论", "parent_id", parentID)

	var children []*model.Comment
	err := r.data.db.WithContext(ctx).
		Where("parent_id = ? AND status = ?", parentID, model.CommentStatusApproved).
		Order("created_at ASC").Find(&children).Error
	if err != nil {
		r.data.Log(ctx).Errorw("查询子评论失败", "parent_id", parentID, "错误", err)
		return nil, fmt.Errorf("list children comments: %w", err)
	}
	r.data.Log(ctx).Debugw("查询子评论完成", "parent_id", parentID, "count", len(children))
	return children, nil
}

// =============================================================================
// 审核状态管理
// =============================================================================

// UpdateStatus 更新评论审核状态
// 1) 先查询评论确认存在并获取 PostID
// 2) 更新 status 字段
// 3) 使该文章所有缓存失效
func (r *commentRepo) UpdateStatus(ctx context.Context, id uint, status model.CommentStatus) error {
	ctx, span := r.data.StartSpan(ctx, "CommentRepo.UpdateStatus")
	defer span.End()

	span.SetAttributes(attribute.Int("comment.id", int(id)), attribute.Int("status", int(status)))

	r.data.Log(ctx).Debugw("开始更新评论审核状态", "id", id, "status", status)

	// 先查询评论以获取 PostID
	comment, err := r.FindByID(ctx, id)
	if err != nil {
		r.data.Log(ctx).Debugw("更新状态前查询评论失败", "id", id, "错误", err)
		return err
	}

	// 更新审核状态
	result := r.data.db.WithContext(ctx).Model(&model.Comment{}).Where("id = ?", id).Update("status", status)
	if result.Error != nil {
		r.data.Log(ctx).Errorw("更新评论审核状态失败", "id", id, "status", status, "错误", result.Error)
		return fmt.Errorf("update comment status: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		r.data.Log(ctx).Debugw("更新审核状态时未找到评论", "id", id)
		return ErrCommentNotFound
	}

	// 状态变更后使缓存失效
	r.invalidateCommentCache(ctx, comment.PostID)
	r.data.Log(ctx).Infow("评论审核状态更新成功", "id", id, "status", status)
	return nil
}

// =============================================================================
// 统计与批量操作
// =============================================================================

// CountByPost 统计文章已通过审核的评论数（Cache-Aside 模式）
// 1) 先尝试从缓存读取评论计数
// 2) 缓存未命中则查询数据库，并将结果写入缓存
func (r *commentRepo) CountByPost(ctx context.Context, postID uint) (int64, error) {
	ctx, span := r.data.StartSpan(ctx, "CommentRepo.CountByPost")
	defer span.End()

	span.SetAttributes(attribute.Int("post_id", int(postID)))

	r.data.Log(ctx).Debugw("开始统计文章评论数", "post_id", postID)

	// 构建计数缓存键
	cacheKey := fmt.Sprintf(cacheKeyCommentCount, postID)
	r.data.Log(ctx).Debugw("查询评论计数缓存", "cache_key", cacheKey)

	// 尝试从缓存读取
	data, err := r.data.cache.Get(ctx, cacheKey)
	if err == nil {
		var count int64
		if unmarshalErr := json.Unmarshal(data, &count); unmarshalErr == nil {
			// 缓存命中且反序列化成功
			r.data.Log(ctx).Debugw("评论计数缓存命中", "post_id", postID, "count", count)
			return count, nil
		}
		// 缓存数据损坏，需重新加载
		r.data.Log(ctx).Warnw("评论计数缓存数据损坏，重新加载", "post_id", postID)
	}
	if !errors.Is(err, cache.ErrKeyNotFound) && err != nil {
		// 缓存读取异常（非键不存在），回源数据库
		r.data.Log(ctx).Warnw("评论计数缓存读取异常，回源数据库查询", "post_id", postID, "错误", err)
	} else {
		r.data.Log(ctx).Debugw("评论计数缓存未命中，回源数据库查询", "post_id", postID)
	}

	// 回源数据库查询
	r.data.Log(ctx).Debugw("数据库统计评论数", "post_id", postID)
	var count int64
	if err := r.data.db.WithContext(ctx).Model(&model.Comment{}).
		Where("post_id = ? AND status = ?", postID, model.CommentStatusApproved).
		Count(&count).Error; err != nil {
		r.data.Log(ctx).Errorw("统计文章评论数失败", "post_id", postID, "错误", err)
		return 0, fmt.Errorf("count comments by post: %w", err)
	}
	r.data.Log(ctx).Debugw("数据库评论计数完成", "post_id", postID, "count", count)

	// 回写缓存
	if setErr := r.data.cache.Set(ctx, cacheKey, count, cacheTTLCommentCount); setErr != nil {
		r.data.Log(ctx).Warnw("写入评论计数缓存失败", "错误", setErr)
	}

	span.SetAttributes(attribute.Int64("count", count))
	return count, nil
}

// BatchDeleteByPost 按文章 ID 批量软删除评论
// 通常用于文章被删除时的级联操作
func (r *commentRepo) BatchDeleteByPost(ctx context.Context, postID uint) error {
	ctx, span := r.data.StartSpan(ctx, "CommentRepo.BatchDeleteByPost")
	defer span.End()

	span.SetAttributes(attribute.Int("post_id", int(postID)))

	r.data.Log(ctx).Debugw("开始批量删除文章评论", "post_id", postID)

	// GORM 软删除所有属于该文章的评论
	result := r.data.db.WithContext(ctx).Where("post_id = ?", postID).Delete(&model.Comment{})
	if result.Error != nil {
		r.data.Log(ctx).Errorw("批量删除文章评论失败", "post_id", postID, "错误", result.Error)
		return fmt.Errorf("batch delete comments by post: %w", result.Error)
	}

	// 删除后使所有相关缓存失效
	r.invalidateCommentCache(ctx, postID)

	r.data.Log(ctx).Infow("文章评论批量删除成功",
		"post_id", postID, "rows_affected", result.RowsAffected,
	)
	return nil
}

// =============================================================================
// 缓存管理辅助方法
// =============================================================================

// invalidateCommentCache 使文章相关的所有缓存失效
// 同时清除评论计数缓存和评论列表缓存
func (r *commentRepo) invalidateCommentCache(ctx context.Context, postID uint) {
	r.data.Log(ctx).Debugw("开始失效评论缓存", "post_id", postID)

	// 清除评论计数缓存
	countKey := fmt.Sprintf(cacheKeyCommentCount, postID)
	if err := r.data.cache.Delete(ctx, countKey); err != nil {
		r.data.Log(ctx).Warnw("删除评论计数缓存失败", "key", countKey, "错误", err)
	} else {
		r.data.Log(ctx).Debugw("评论计数缓存已失效", "key", countKey)
	}

	// 清除评论列表缓存
	r.invalidateListCache(ctx, postID)
	r.data.Log(ctx).Debugw("文章评论缓存全部失效完成", "post_id", postID)
}

// invalidateListCache 使文章评论列表缓存失效
// 遍历常见的分页大小（10/20/50）和前 5 页组合，逐一删除对应缓存键
func (r *commentRepo) invalidateListCache(ctx context.Context, postID uint) {
	commonPageSizes := []int{10, 20, 50} // 常见的分页大小
	maxPage := 5                         // 默认清除前 5 页的缓存

	r.data.Log(ctx).Debugw("开始批量失效评论列表缓存",
		"post_id", postID, "page_sizes", commonPageSizes, "max_page", maxPage,
	)

	for _, size := range commonPageSizes {
		for page := 1; page <= maxPage; page++ {
			key := fmt.Sprintf(cacheKeyCommentList, postID, page, size)
			if err := r.data.cache.Delete(ctx, key); err != nil {
				r.data.Log(ctx).Warnw("删除评论列表缓存失败", "key", key, "错误", err)
			} else {
				r.data.Log(ctx).Debugw("评论列表缓存已失效", "key", key)
			}
		}
	}

	r.data.Log(ctx).Debugw("评论列表缓存批量失效完成", "post_id", postID)
}

// readListCache 从缓存读取评论列表
// 返回值: 评论列表, 总数, 错误
// 特殊情况: 缓存的 "null" 字符串表示已知空列表（防穿透标记），返回 cache.ErrKeyNotFound
func (r *commentRepo) readListCache(ctx context.Context, cacheKey string) ([]*model.Comment, int64, error) {
	data, err := r.data.cache.Get(ctx, cacheKey)
	if err != nil {
		// 缓存获取失败（键不存在或连接错误等），直接返回上游错误
		return nil, 0, err
	}

	// "null" 字符串是防穿透标记，表示已知该查询条件下无数据
	if string(data) == "null" {
		r.data.Log(ctx).Debugw("评论列表缓存命中空值标记（防穿透）", "cache_key", cacheKey)
		return nil, 0, cache.ErrKeyNotFound
	}

	// 反序列化缓存数据
	var cached commentListCache
	if unmarshalErr := json.Unmarshal(data, &cached); unmarshalErr != nil {
		r.data.Log(ctx).Warnw("评论列表缓存反序列化失败", "cache_key", cacheKey, "错误", unmarshalErr)
		return nil, 0, fmt.Errorf("comment list cache deserialize: %w", unmarshalErr)
	}

	r.data.Log(ctx).Debugw("评论列表缓存反序列化成功",
		"cache_key", cacheKey, "total", cached.Total, "count", len(cached.Comments),
	)
	return cached.Comments, cached.Total, nil
}

// writeListCache 将评论列表写入缓存
// 空列表（comments 长度为 0 且 total 为 0）使用较短 TTL 的 "null" 标记，防止缓存穿透
// 非空列表序列化为 JSON 后以正常 TTL 写入
func (r *commentRepo) writeListCache(ctx context.Context, cacheKey string, comments []*model.Comment, total int64) {
	// 空结果：写入 "null" 防穿透标记，TTL 较短以减少内存占用
	if len(comments) == 0 && total == 0 {
		r.data.Log(ctx).Debugw("写入防穿透空值缓存", "cache_key", cacheKey, "ttl", cacheTTLCommentStale)
		if err := r.data.cache.Set(ctx, cacheKey, []byte("null"), cacheTTLCommentStale); err != nil {
			r.data.Log(ctx).Warnw("写入空列表防穿透缓存失败", "错误", err)
		}
		return
	}

	// 序列化并写入缓存
	cached := commentListCache{Total: total, Comments: comments}
	r.data.Log(ctx).Debugw("写入评论列表缓存",
		"cache_key", cacheKey, "total", total, "count", len(comments), "ttl", cacheTTLCommentList,
	)
	if err := r.data.cache.Set(ctx, cacheKey, &cached, cacheTTLCommentList); err != nil {
		r.data.Log(ctx).Warnw("写入评论列表缓存失败", "错误", err)
	}
}
