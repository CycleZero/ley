package biz

import (
	"context"
	"errors"
	"fmt"

	"ley/app/post/internal/model"

	"github.com/go-kratos/kratos/v2/log"
)

// =============================================================================
// 标签/分类业务层错误定义
// =============================================================================

var (
	ErrTagNotFound   = errors.New("tag not found")       // 标签不存在
	ErrTagNameExists = errors.New("tag name already exists") // 标签名称已存在
)

// =============================================================================
// TagUseCase — 标签业务用例
// =============================================================================

// TagUseCase 标签业务用例
// 封装标签的创建、查询、删除等核心业务逻辑。
type TagUseCase struct {
	repo   TagRepo    // 标签数据访问接口
	logger log.Logger // Kratos 日志器
}

// NewTagUseCase 创建 TagUseCase
func NewTagUseCase(repo TagRepo, logger log.Logger) *TagUseCase {
	return &TagUseCase{repo: repo, logger: logger}
}

// log 返回携带上下文信息的日志助手
func (uc *TagUseCase) log(ctx context.Context) *log.Helper {
	return log.NewHelper(log.WithContext(ctx, uc.logger))
}

// =============================================================================
// CreateTag — 创建标签
// =============================================================================

// CreateTag 创建标签
// 根据名称创建标签，自动生成 slug。
func (uc *TagUseCase) CreateTag(ctx context.Context, name string) (*model.Tag, error) {
	// 根据名称生成 slug
	slug := tagNameToSlug(name)
	tag := &model.Tag{Name: name, Slug: slug}

	uc.log(ctx).Debugw("开始创建标签", "name", name, "slug", slug)

	// 委托数据层创建
	if err := uc.repo.Create(ctx, tag); err != nil {
		// 名称冲突为预期错误，直接传递
		if errors.Is(err, ErrTagNameExists) {
			uc.log(ctx).Warnw("创建标签失败：名称已存在", "name", name)
			return nil, err
		}
		uc.log(ctx).Errorw("创建标签数据层操作失败", "name", name, "error", err)
		return nil, fmt.Errorf("create tag: %w", err)
	}

	uc.log(ctx).Infow("标签创建成功", "id", tag.ID, "name", name)
	return tag, nil
}

// =============================================================================
// ListTags — 全量标签
// =============================================================================

// ListTags 全量标签
// 委托给数据层 List 方法。
func (uc *TagUseCase) ListTags(ctx context.Context) ([]*model.Tag, error) {
	uc.log(ctx).Debugw("查询全量标签列表")
	return uc.repo.List(ctx)
}

// =============================================================================
// DeleteTag — 删除标签
// =============================================================================

// DeleteTag 删除标签
// 委托给数据层 Delete 方法。
func (uc *TagUseCase) DeleteTag(ctx context.Context, id uint) error {
	uc.log(ctx).Debugw("开始删除标签", "id", id)

	if err := uc.repo.Delete(ctx, id); err != nil {
		uc.log(ctx).Errorw("删除标签数据层操作失败", "id", id, "error", err)
		return fmt.Errorf("delete tag: %w", err)
	}

	uc.log(ctx).Infow("标签删除成功", "id", id)
	return nil
}

// =============================================================================
// CategoryUseCase — 分类业务用例
// =============================================================================

// CategoryUseCase 分类业务用例
// 封装分类的创建、更新、删除、查询等核心业务逻辑。
type CategoryUseCase struct {
	repo   CategoryRepo // 分类数据访问接口
	logger log.Logger   // Kratos 日志器
}

// NewCategoryUseCase 创建 CategoryUseCase
func NewCategoryUseCase(repo CategoryRepo, logger log.Logger) *CategoryUseCase {
	return &CategoryUseCase{repo: repo, logger: logger}
}

// log 返回携带上下文信息的日志助手
func (uc *CategoryUseCase) log(ctx context.Context) *log.Helper {
	return log.NewHelper(log.WithContext(ctx, uc.logger))
}

// =============================================================================
// CreateCategory — 创建分类
// =============================================================================

// CreateCategory 创建分类
// 支持设置父分类 ID 和排序权重。
func (uc *CategoryUseCase) CreateCategory(ctx context.Context, name, slug, description string, parentID *uint, sortOrder int) (*model.Category, error) {
	// 构建分类实体
	cat := &model.Category{
		Name:        name,
		Slug:        slug,
		Description: description,
		ParentID:    parentID,
		SortOrder:   sortOrder,
	}

	uc.log(ctx).Debugw("开始创建分类", "name", name, "slug", slug, "parent_id", parentID, "sort_order", sortOrder)

	// 委托数据层创建
	if err := uc.repo.Create(ctx, cat); err != nil {
		uc.log(ctx).Errorw("创建分类数据层操作失败", "name", name, "error", err)
		return nil, fmt.Errorf("create category: %w", err)
	}

	uc.log(ctx).Infow("分类创建成功", "id", cat.ID, "name", name)
	return cat, nil
}

// =============================================================================
// UpdateCategory — 更新分类
// =============================================================================

// UpdateCategory 更新分类
// 先查询再更新，确保分类存在性。
func (uc *CategoryUseCase) UpdateCategory(ctx context.Context, id uint, name, slug, description string, parentID *uint, sortOrder int) (*model.Category, error) {
	uc.log(ctx).Debugw("开始更新分类", "id", id, "name", name)

	// 先查询分类确认存在
	cat, err := uc.repo.FindByID(ctx, id)
	if err != nil {
		uc.log(ctx).Debugw("更新分类未找到", "id", id, "error", err)
		return nil, err
	}

	// 更新字段
	cat.Name = name
	cat.Slug = slug
	cat.Description = description
	cat.ParentID = parentID
	cat.SortOrder = sortOrder

	// 委托数据层更新
	if err := uc.repo.Update(ctx, cat); err != nil {
		uc.log(ctx).Errorw("更新分类数据层操作失败", "id", id, "error", err)
		return nil, fmt.Errorf("update category: %w", err)
	}

	uc.log(ctx).Infow("分类更新成功", "id", id)
	return cat, nil
}

// =============================================================================
// DeleteCategory — 删除分类（含子分类检查）
// =============================================================================

// DeleteCategory 删除分类（含子分类检查）
// 如果分类下还有子分类，不允许删除。
func (uc *CategoryUseCase) DeleteCategory(ctx context.Context, id uint) error {
	uc.log(ctx).Debugw("开始删除分类", "id", id)

	// 检查是否存在子分类
	children, _ := uc.repo.ListChildren(ctx, id)
	if len(children) > 0 {
		uc.log(ctx).Warnw("删除分类失败：存在子分类，请先删除子分类", "id", id, "child_count", len(children))
		return errors.New("cannot delete category with sub-categories; remove children first")
	}

	// 委托数据层删除
	if err := uc.repo.Delete(ctx, id); err != nil {
		uc.log(ctx).Errorw("删除分类数据层操作失败", "id", id, "error", err)
		return err
	}

	uc.log(ctx).Infow("分类删除成功", "id", id)
	return nil
}

// =============================================================================
// ListCategories — 查询分类树
// =============================================================================

// ListCategories 查询分类树
// 委托给数据层 ListTree 方法，返回完整分类树结构。
func (uc *CategoryUseCase) ListCategories(ctx context.Context) ([]*model.Category, error) {
	uc.log(ctx).Debugw("查询分类树")
	return uc.repo.ListTree(ctx)
}
