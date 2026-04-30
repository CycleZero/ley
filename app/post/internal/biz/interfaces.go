package biz

import (
	"context"

	"ley/app/post/internal/model"
)

// =============================================================================
// PostListQuery — 文章列表查询参数
// =============================================================================

// PostListQuery 文章列表查询参数
// 封装列表查询所需的所有过滤、排序和分页参数。
type PostListQuery struct {
	Status     string   // 文章状态过滤：draft/published/archived
	CategoryID *uint    // 分类ID过滤（可为空）
	Tags       []string // 标签名称过滤（AND 逻辑，需同时拥有所有指定标签）
	AuthorID   *uint    // 作者ID过滤（可为空）
	SortBy     string   // 排序字段：created_at/updated_at/published_at/view_count/is_top
	SortOrder  string   // 排序方向：asc/desc
	Page       int      // 页码（从1开始）
	PageSize   int      // 每页数量
}

// =============================================================================
// PostRepo — 文章数据访问接口
// =============================================================================

// PostRepo 文章数据访问接口
// 定义文章相关的全部数据访问操作，由 data 层实现。
type PostRepo interface {
	// Create 创建文章
	Create(ctx context.Context, post *model.Post) error
	// Update 更新文章
	Update(ctx context.Context, post *model.Post) error
	// Delete 软删除文章
	Delete(ctx context.Context, id uint) error
	// FindByID 按主键查询文章
	FindByID(ctx context.Context, id uint) (*model.Post, error)
	// FindByUUID 按UUID查询文章（走缓存）
	FindByUUID(ctx context.Context, uuid string) (*model.Post, error)
	// FindBySlug 按Slug查询文章（走缓存）
	FindBySlug(ctx context.Context, slug string) (*model.Post, error)
	// List 分页查询文章列表
	List(ctx context.Context, query PostListQuery) ([]*model.Post, int64, error)
	// Search 全文搜索文章
	Search(ctx context.Context, keyword string, page, pageSize int) ([]*model.Post, int64, error)
	// IncrementViewCount 原子增加文章浏览计数
	IncrementViewCount(ctx context.Context, id uint, delta int64) error
	// AssociateTags 添加标签关联（幂等，重复关联不会报错）
	AssociateTags(ctx context.Context, postID uint, tagIDs []uint) error
	// SyncTags 全量替换标签关联（在事务中先删后增）
	SyncTags(ctx context.Context, postID uint, tagIDs []uint) error
	// ListTagsByPostID 获取文章所有标签
	ListTagsByPostID(ctx context.Context, postID uint) ([]*model.Tag, error)
}

// =============================================================================
// TagRepo — 标签数据访问接口
// =============================================================================

// TagRepo 标签数据访问接口
// 定义标签相关的全部数据访问操作，由 data 层实现。
type TagRepo interface {
	// Create 创建标签
	Create(ctx context.Context, tag *model.Tag) error
	// FindByName 按名称查询标签
	FindByName(ctx context.Context, name string) (*model.Tag, error)
	// FindOrCreate 查找标签，不存在则创建（幂等）
	FindOrCreate(ctx context.Context, name, slug string) (*model.Tag, error)
	// List 查询全量标签列表
	List(ctx context.Context) ([]*model.Tag, error)
	// Delete 软删除标签
	Delete(ctx context.Context, id uint) error
	// IncrementPostCount 原子增减标签下文章计数
	IncrementPostCount(ctx context.Context, id uint, delta int) error
}

// =============================================================================
// CategoryRepo — 分类数据访问接口
// =============================================================================

// CategoryRepo 分类数据访问接口
// 定义分类相关的全部数据访问操作，由 data 层实现。
type CategoryRepo interface {
	// Create 创建分类
	Create(ctx context.Context, cat *model.Category) error
	// Update 更新分类
	Update(ctx context.Context, cat *model.Category) error
	// Delete 软删除分类
	Delete(ctx context.Context, id uint) error
	// FindByID 按主键查询分类
	FindByID(ctx context.Context, id uint) (*model.Category, error)
	// FindBySlug 按slug查询分类
	FindBySlug(ctx context.Context, slug string) (*model.Category, error)
	// List 查询顶级分类列表
	List(ctx context.Context) ([]*model.Category, error)
	// ListChildren 查询指定父分类的直接子分类
	ListChildren(ctx context.Context, parentID uint) ([]*model.Category, error)
	// ListTree 查询完整分类树
	ListTree(ctx context.Context) ([]*model.Category, error)
	// IncrementPostCount 原子增减分类下文章计数
	IncrementPostCount(ctx context.Context, id uint, delta int) error
}
