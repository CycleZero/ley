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
)

// =============================================================================
// 分类缓存常量
// =============================================================================

const (
	cacheKeyCategoryTree = "category:tree"     // 分类树缓存键
	cacheTTLCategoryTree = 60 * time.Minute     // 分类树缓存过期时间
)

// =============================================================================
// 分类相关错误定义
// =============================================================================

var (
	ErrCategoryNotFound = errors.New("category not found")          // 分类不存在
	ErrCategoryExists   = errors.New("category name or slug already exists") // 分类名称或 slug 已存在
)

// =============================================================================
// categoryRepo — CategoryRepo 接口实现
// =============================================================================

// categoryRepo CategoryRepo 接口实现
// 封装分类相关的数据库操作和缓存逻辑。
type categoryRepo struct{ data *Data }

// 编译时接口实现检查
var _ biz.CategoryRepo = (*categoryRepo)(nil)

// =============================================================================
// Create — 创建分类
// =============================================================================

// Create 创建分类
// 将分类写入数据库，若名称或 slug 冲突则返回 ErrCategoryExists。
func (r *categoryRepo) Create(ctx context.Context, cat *model.Category) error {
	ctx, span := r.data.StartSpan(ctx, "CategoryRepo.Create")
	defer span.End()

	span.SetAttributes(attribute.String("category.name", cat.Name), attribute.String("category.slug", cat.Slug))
	r.data.Log(ctx).Debugw("开始创建分类", "name", cat.Name, "slug", cat.Slug, "parent_id", cat.ParentID)

	// 执行数据库插入
	err := r.data.db.WithContext(ctx).Create(cat).Error
	if err != nil {
		// 判断是否为唯一键冲突
		if util.IsUniqueViolation(err) {
			r.data.Log(ctx).Warnw("创建分类失败：名称或slug重复", "name", cat.Name, "slug", cat.Slug, "error", err)
			return ErrCategoryExists
		}
		r.data.Log(ctx).Errorw("创建分类数据库写入失败", "name", cat.Name, "error", err)
		return fmt.Errorf("create category: %w", err)
	}

	// 创建分类后清除分类树缓存
	r.invalidateCategoryCache(ctx)

	r.data.Log(ctx).Infow("分类创建成功", "id", cat.ID, "name", cat.Name, "slug", cat.Slug)
	return nil
}

// =============================================================================
// Update — 更新分类（使用 map）
// =============================================================================

// Update 更新分类（使用 map）
// 使用 map[string]interface{} 避免 GORM struct 零值跳过问题。
func (r *categoryRepo) Update(ctx context.Context, cat *model.Category) error {
	ctx, span := r.data.StartSpan(ctx, "CategoryRepo.Update")
	defer span.End()

	span.SetAttributes(attribute.Int("category.id", int(cat.ID)))
	r.data.Log(ctx).Debugw("开始更新分类", "id", cat.ID, "name", cat.Name)

	// 使用 map 更新，避免零值字段被 GORM 忽略
	result := r.data.db.WithContext(ctx).
		Model(&model.Category{}).
		Where("id = ?", cat.ID).
		Updates(map[string]interface{}{
			"name":        cat.Name,
			"slug":        cat.Slug,
			"description": cat.Description,
			"parent_id":   cat.ParentID,
			"sort_order":  cat.SortOrder,
		})

	if result.Error != nil {
		r.data.Log(ctx).Errorw("更新分类数据库操作失败", "id", cat.ID, "error", result.Error)
		return fmt.Errorf("update category: %w", result.Error)
	}
	// 检查是否有行被实际更新
	if result.RowsAffected == 0 {
		r.data.Log(ctx).Debugw("更新分类未找到记录", "id", cat.ID)
		return ErrCategoryNotFound
	}

	// 更新分类后清除分类树缓存
	r.invalidateCategoryCache(ctx)
	r.data.Log(ctx).Infow("分类更新成功", "id", cat.ID)
	return nil
}

// =============================================================================
// Delete — 软删除分类
// =============================================================================

// Delete 软删除分类
// 执行软删除并清除相关缓存。
func (r *categoryRepo) Delete(ctx context.Context, id uint) error {
	ctx, span := r.data.StartSpan(ctx, "CategoryRepo.Delete")
	defer span.End()

	span.SetAttributes(attribute.Int("category.id", int(id)))
	r.data.Log(ctx).Debugw("开始删除分类", "id", id)

	// 执行软删除
	result := r.data.db.WithContext(ctx).Where("id = ?", id).Delete(&model.Category{})
	if result.Error != nil {
		r.data.Log(ctx).Errorw("删除分类数据库操作失败", "id", id, "error", result.Error)
		return fmt.Errorf("delete category: %w", result.Error)
	}
	// 检查是否有行被实际删除
	if result.RowsAffected == 0 {
		r.data.Log(ctx).Debugw("删除分类未找到记录", "id", id)
		return ErrCategoryNotFound
	}

	// 删除分类后清除分类树缓存
	r.invalidateCategoryCache(ctx)
	r.data.Log(ctx).Infow("分类删除成功", "id", id)
	return nil
}

// =============================================================================
// FindByID — 按主键查询
// =============================================================================

// FindByID 按主键查询
// 直接查询数据库，不使用缓存。
func (r *categoryRepo) FindByID(ctx context.Context, id uint) (*model.Category, error) {
	r.data.Log(ctx).Debugw("按主键查询分类", "id", id)

	var cat model.Category
	err := r.data.db.WithContext(ctx).Where("id = ?", id).First(&cat).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			r.data.Log(ctx).Debugw("按主键查询分类未找到", "id", id)
			return nil, ErrCategoryNotFound
		}
		r.data.Log(ctx).Errorw("按主键查询分类失败", "id", id, "error", err)
		return nil, fmt.Errorf("find category by id: %w", err)
	}
	return &cat, nil
}

// =============================================================================
// FindBySlug — 按 slug 查询
// =============================================================================

// FindBySlug 按 slug 查询
// 直接查询数据库，不使用缓存。
func (r *categoryRepo) FindBySlug(ctx context.Context, slug string) (*model.Category, error) {
	r.data.Log(ctx).Debugw("按Slug查询分类", "slug", slug)

	var cat model.Category
	err := r.data.db.WithContext(ctx).Where("slug = ?", slug).First(&cat).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			r.data.Log(ctx).Debugw("按Slug查询分类未找到", "slug", slug)
			return nil, ErrCategoryNotFound
		}
		r.data.Log(ctx).Errorw("按Slug查询分类失败", "slug", slug, "error", err)
		return nil, fmt.Errorf("find category by slug: %w", err)
	}
	return &cat, nil
}

// =============================================================================
// List — 查询顶级分类列表
// =============================================================================

// List 查询顶级分类列表
// 委托给 ListTree 实现，返回分类树的根节点列表。
func (r *categoryRepo) List(ctx context.Context) ([]*model.Category, error) {
	tree, err := r.ListTree(ctx)
	if err != nil {
		return nil, err
	}
	return tree, nil
}

// =============================================================================
// ListTree — 查询完整分类树（Cache-Aside）
// =============================================================================

// ListTree 查询完整分类树（Cache-Aside 模式）
// 1. 先查缓存，命中则直接返回
// 2. 缓存未命中则查询全部分类并通过 buildCategoryTree 构建树结构
// 3. 数据库查询结果回填缓存
func (r *categoryRepo) ListTree(ctx context.Context) ([]*model.Category, error) {
	ctx, span := r.data.StartSpan(ctx, "CategoryRepo.ListTree")
	defer span.End()

	r.data.Log(ctx).Debugw("查询分类树，尝试读取缓存", "cache_key", cacheKeyCategoryTree)

	// 第一步：尝试从缓存读取
	var cached []*model.Category
	err := r.data.cache.GetObject(ctx, cacheKeyCategoryTree, &cached)
	if err == nil {
		r.data.Log(ctx).Debugw("分类树缓存命中", "root_count", len(cached))
		span.SetAttributes(attribute.Int("category.count", len(cached)))
		return cached, nil
	}
	if !errors.Is(err, cache.ErrKeyNotFound) {
		r.data.Log(ctx).Warnw("分类树缓存读取异常，回退到数据库查询", "error", err)
	}
	r.data.Log(ctx).Debugw("分类树缓存未命中，查询数据库")

	// 第二步：缓存未命中，查询数据库获取全量分类
	var all []*model.Category
	if err := r.data.db.WithContext(ctx).Order("sort_order ASC, id ASC").Find(&all).Error; err != nil {
		r.data.Log(ctx).Errorw("查询全量分类列表失败", "error", err)
		return nil, fmt.Errorf("list categories: %w", err)
	}
	r.data.Log(ctx).Debugw("数据库查询全量分类成功，开始构建分类树", "total_count", len(all))

	// 第三步：将扁平列表构建为树结构
	tree := buildCategoryTree(all)
	r.data.Log(ctx).Debugw("分类树构建完成", "root_count", len(tree))

	// 第四步：将构建好的分类树回填缓存
	if setErr := r.data.cache.Set(ctx, cacheKeyCategoryTree, tree, cacheTTLCategoryTree); setErr != nil {
		r.data.Log(ctx).Warnw("分类树缓存写入失败", "error", setErr)
	}

	span.SetAttributes(attribute.Int("category.root_count", len(tree)))
	return tree, nil
}

// =============================================================================
// ListChildren — 查询指定父分类的直接子分类
// =============================================================================

// ListChildren 查询指定父分类的直接子分类
// 按排序权重升序排列。
func (r *categoryRepo) ListChildren(ctx context.Context, parentID uint) ([]*model.Category, error) {
	ctx, span := r.data.StartSpan(ctx, "CategoryRepo.ListChildren")
	defer span.End()

	span.SetAttributes(attribute.Int("parent_id", int(parentID)))
	r.data.Log(ctx).Debugw("查询子分类列表", "parent_id", parentID)

	var children []*model.Category
	err := r.data.db.WithContext(ctx).Where("parent_id = ?", parentID).Order("sort_order ASC").Find(&children).Error
	if err != nil {
		r.data.Log(ctx).Errorw("查询子分类列表失败", "parent_id", parentID, "error", err)
		return nil, fmt.Errorf("list children categories: %w", err)
	}
	r.data.Log(ctx).Debugw("子分类列表查询完成", "parent_id", parentID, "child_count", len(children))
	return children, nil
}

// =============================================================================
// IncrementPostCount — 原子增减分类文章计数
// =============================================================================

// IncrementPostCount 原子增减分类文章计数
// 使用 SQL 表达式 post_count + ? 确保并发安全。
func (r *categoryRepo) IncrementPostCount(ctx context.Context, id uint, delta int) error {
	_, span := r.data.StartSpan(ctx, "CategoryRepo.IncrementPostCount")
	defer span.End()

	span.SetAttributes(attribute.Int("category.id", int(id)), attribute.Int("delta", delta))
	r.data.Log(ctx).Debugw("原子增减分类文章计数", "id", id, "delta", delta)

	// 使用 gorm.Expr 执行原子更新
	result := r.data.db.WithContext(ctx).Model(&model.Category{}).
		Where("id = ?", id).UpdateColumn("post_count", gorm.Expr("post_count + ?", delta))

	if result.Error != nil {
		r.data.Log(ctx).Errorw("增减分类文章计数失败", "id", id, "delta", delta, "error", result.Error)
		return fmt.Errorf("increment category post count: %w", result.Error)
	}

	r.data.Log(ctx).Debugw("分类文章计数更新成功", "id", id, "delta", delta)
	return nil
}

// =============================================================================
// invalidateCategoryCache — 清除分类缓存
// =============================================================================

// invalidateCategoryCache 清除分类缓存
// 分类发生变更时清除分类树缓存。
func (r *categoryRepo) invalidateCategoryCache(ctx context.Context) {
	r.data.Log(ctx).Debugw("清除分类树缓存", "cache_key", cacheKeyCategoryTree)
	if err := r.data.cache.Delete(ctx, cacheKeyCategoryTree); err != nil {
		r.data.Log(ctx).Warnw("删除分类树缓存失败", "error", err)
	}
}

// =============================================================================
// buildCategoryTree — 将扁平分类列表构建为树结构
// =============================================================================

// buildCategoryTree 将扁平分类列表构建为树结构
// 1. 遍历所有分类建立 ID 索引
// 2. 将有 ParentID 的分类挂载到对应父分类的 Children 中
// 3. 没有父分类或父分类不存在的作为根节点返回
func buildCategoryTree(categories []*model.Category) []*model.Category {
	if len(categories) == 0 {
		return nil
	}

	// 建立 ID → Category 的映射表，便于 O(1) 查找父节点
	idMap := make(map[uint]*model.Category, len(categories))
	for _, c := range categories {
		idMap[c.ID] = c
	}

	// 遍历所有分类，构建父子关系
	var roots []*model.Category
	for _, c := range categories {
		// 如果 ParentID 非空且父分类存在于映射表中，则挂载为子节点
		if c.ParentID != nil && *c.ParentID != 0 {
			if parent, ok := idMap[*c.ParentID]; ok {
				parent.Children = append(parent.Children, c)
				continue
			}
		}
		// 否则作为根节点
		roots = append(roots, c)
	}
	return roots
}
