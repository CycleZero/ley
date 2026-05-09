package biz

import (
	"context"
	"fmt"
	"strings"
	"time"

	kerrors "github.com/go-kratos/kratos/v2/errors"
	"github.com/go-kratos/kratos/v2/log"
)

// =============================================================================
// Tag — 标签业务模型
//
// 标签与文章是多对多关系，通过 articles_tags 中间表关联。
// ArticleCount 由系统在文章发布/删除时自动维护，不可由 API 直接修改。
// =============================================================================

type Tag struct {
	ID           uint      // 主键
	Name         string    // 标签名称（唯一，1-64字符）
	Slug         string    // URL slug（自动从名称生成）
	ArticleCount int64     // 关联文章数（系统维护）
	CreatedAt    time.Time // 创建时间
}

// =============================================================================
// TagRepo — 标签数据访问接口（依赖倒置原则）
// =============================================================================

type TagRepo interface {
	Create(ctx context.Context, tag *Tag) error
	FindByName(ctx context.Context, name string) (*Tag, error)
	FindOrCreate(ctx context.Context, name, slug string) (*Tag, error)
	List(ctx context.Context) ([]*Tag, error)
	Delete(ctx context.Context, id uint) error
	IncrementArticleCount(ctx context.Context, id uint, delta int64) error
}

// =============================================================================
// Category — 分类业务模型
//
// 分类支持层级结构（通过 ParentID 指向父分类）。
// Children 在 data 层查询分类树时动态组装，不存储在数据库中。
// ArticleCount 由系统维护。
// =============================================================================

type Category struct {
	ID           uint        // 主键
	Name         string      // 分类名称（1-64字符）
	Slug         string      // URL slug
	Description  string      // 分类描述（最长500字符）
	ParentID     *uint       // 父分类 ID（nil=根分类）
	SortOrder    int         // 排序权重（越小越靠前）
	ArticleCount int64       // 关联文章数（系统维护）
	Children     []*Category // 子分类（data层查询树时填充）
	CreatedAt    time.Time   // 创建时间
}

// =============================================================================
// CategoryRepo — 分类数据访问接口（依赖倒置原则）
// =============================================================================

type CategoryRepo interface {
	Create(ctx context.Context, cat *Category) error
	Update(ctx context.Context, cat *Category) error
	Delete(ctx context.Context, id uint) error
	FindByID(ctx context.Context, id uint) (*Category, error)
	ListChildren(ctx context.Context, parentID uint) ([]*Category, error)
	ListTree(ctx context.Context) ([]*Category, error)
	IncrementArticleCount(ctx context.Context, id uint, delta int64) error
}

// =============================================================================
// 标签/分类错误定义
// =============================================================================

var (
	ErrTagNotFound          = kerrors.NotFound("TAG_NOT_FOUND", "标签不存在")
	ErrTagNameExists        = kerrors.Conflict("TAG_NAME_EXISTS", "标签名称已存在")
	ErrCategoryNotFound     = kerrors.NotFound("CATEGORY_NOT_FOUND", "分类不存在")
	ErrCategoryNameExists   = kerrors.Conflict("CATEGORY_NAME_EXISTS", "分类名称或Slug已存在")
	ErrCategoryHasChildren  = kerrors.Conflict("CATEGORY_HAS_CHILDREN", "请先删除子分类")
	ErrCategoryHasArticles  = kerrors.Conflict("CATEGORY_HAS_ARTICLES", "分类下存在文章，不允许删除")
	ErrCategorySlugEmpty    = kerrors.BadRequest("CATEGORY_SLUG_EMPTY", "分类Slug不能为空")
	ErrTagNameEmpty         = kerrors.BadRequest("TAG_NAME_EMPTY", "标签名称不能为空")
	ErrCategoryCircularRef  = kerrors.BadRequest("CATEGORY_CIRCULAR_REF", "分类之间不能构成循环引用")
)

// =============================================================================
// TagUseCase — 标签业务用例
//
// 标签管理相对简单：创建、全量列表、删除。
// 标签与文章的关联由 ArticleUseCase 管理（通过 AssociateTags / SyncTags）。
// 标签的 ArticleCount 由系统在文章发布/归档/删除时自动更新。
// =============================================================================

type TagUseCase struct {
	repo TagRepo     // 标签数据访问
	log  *log.Helper // 结构化日志
}

func NewTagUseCase(repo TagRepo, logger log.Logger) *TagUseCase {
	return &TagUseCase{repo: repo, log: log.NewHelper(logger)}
}

// =============================================================================
// CreateTag — 创建标签
//
// 流程：
//  1. 校验名称非空（去除首尾空白后）且长度不超过 64 字符
//  2. 根据名称自动生成 slug（小写 + 空格转连字符）
//  3. 委托 data 层创建（data 层通过 DB 唯一约束防止重复）
//
// 不检查名称是否已存在（由 data 层的唯一约束兜底，冲突时返回 ErrTagNameExists）。
// =============================================================================

func (uc *TagUseCase) CreateTag(ctx context.Context, name string) (*Tag, error) {
	uc.log.WithContext(ctx).Debugf("[CreateTag] 开始 name=%q", name)

	// ===================================================================
	// 步骤1: 校验名称
	// 去除首尾空白后检查是否为空和长度
	// ===================================================================
	name = strings.TrimSpace(name)
	if name == "" {
		uc.log.WithContext(ctx).Warnf("[CreateTag] 名称为空")
		return nil, ErrTagNameEmpty
	}
	if len(name) > 64 {
		uc.log.WithContext(ctx).Warnf("[CreateTag] 名称过长 len=%d", len(name))
		return nil, kerrors.BadRequest("TAG_NAME_TOO_LONG", "标签名称不能超过64个字符")
	}
	uc.log.WithContext(ctx).Debugf("[CreateTag] 名称校验通过 name=%q", name)

	// ===================================================================
	// 步骤2: 生成 slug
	// 小写 + 空格转连字符 — 例如 "Go语言" → "go语言"
	// ===================================================================
	slug := strings.ToLower(strings.ReplaceAll(name, " ", "-"))
	if slug == "" {
		uc.log.WithContext(ctx).Warnf("[CreateTag] slug生成为空 name=%q", name)
		return nil, ErrTagNameEmpty
	}
	uc.log.WithContext(ctx).Debugf("[CreateTag] 生成slug name=%q → slug=%q", name, slug)

	// ===================================================================
	// 步骤3: 委托 data 层创建
	// data 层在 INSERT 时若名称冲突返回 ErrTagNameExists
	// ===================================================================
	tag := &Tag{Name: name, Slug: slug}
	uc.log.WithContext(ctx).Debugf("[CreateTag] 调用 repo.Create name=%q slug=%q", tag.Name, tag.Slug)
	if err := uc.repo.Create(ctx, tag); err != nil {
		if kerrors.IsConflict(err) || kerrors.Code(err) == 409 {
			uc.log.WithContext(ctx).Warnf("[CreateTag] 名称冲突 name=%q", name)
			return nil, ErrTagNameExists
		}
		uc.log.WithContext(ctx).Errorf("[CreateTag] 创建失败 name=%q err=%v", name, err)
		return nil, fmt.Errorf("create tag: %w", err)
	}
	uc.log.WithContext(ctx).Debugf("[CreateTag] repo.Create 返回 id=%d", tag.ID)

	uc.log.WithContext(ctx).Infof("[CreateTag] 创建成功 id=%d name=%q slug=%q", tag.ID, tag.Name, tag.Slug)
	return tag, nil
}

// =============================================================================
// ListTags — 全量标签列表
//
// 返回所有标签（按 article_count 降序, name 升序排列）。
// data 层使用 Cache-Aside 模式：先查缓存，未命中则查 DB 并回写。
// =============================================================================

func (uc *TagUseCase) ListTags(ctx context.Context) ([]*Tag, error) {
	uc.log.WithContext(ctx).Debugf("[ListTags] 开始")
	tags, err := uc.repo.List(ctx)
	if err != nil {
		uc.log.WithContext(ctx).Errorf("[ListTags] 查询失败 err=%v", err)
		return nil, fmt.Errorf("list tags: %w", err)
	}
	uc.log.WithContext(ctx).Debugf("[ListTags] 完成 count=%d", len(tags))
	return tags, nil
}

// =============================================================================
// DeleteTag — 删除标签
//
// 流程：
//  1. 委托 data 层软删除
//  2. data 层检查 RowsAffected = 0 → 标签不存在
//
// 注意：删除标签不会级联删除 articles_tags 中的关联记录。
// 标签被删除后，已关联该标签的文章不再显示此标签。
// =============================================================================

func (uc *TagUseCase) DeleteTag(ctx context.Context, id uint) error {
	uc.log.WithContext(ctx).Debugf("[DeleteTag] 开始 id=%d", id)
	if err := uc.repo.Delete(ctx, id); err != nil {
		uc.log.WithContext(ctx).Errorf("[DeleteTag] 删除失败 id=%d err=%v", id, err)
		return fmt.Errorf("delete tag: %w", err)
	}
	uc.log.WithContext(ctx).Infof("[DeleteTag] 删除成功 id=%d", id)
	return nil
}

// =============================================================================
// CategoryUseCase — 分类业务用例
//
// 分类管理支持层级结构，核心规则：
//   - 不允许删除存在子分类的父分类（需先删除子分类）
//   - 不允许删除存在文章的分类（需先转移或删除文章）
//   - SortOrder 控制同级分类的显示顺序
// =============================================================================

type CategoryUseCase struct {
	repo CategoryRepo  // 分类数据访问
	log  *log.Helper   // 结构化日志
}

func NewCategoryUseCase(repo CategoryRepo, logger log.Logger) *CategoryUseCase {
	return &CategoryUseCase{repo: repo, log: log.NewHelper(logger)}
}

// =============================================================================
// CreateCategory — 创建分类
//
// 流程：
//  1. 校验名称和 slug 非空
//  2. 若未提供 slug，从名称自动生成
//  3. 若有 ParentID，验证父分类存在
//  4. 委托 data 层创建
// =============================================================================

func (uc *CategoryUseCase) CreateCategory(ctx context.Context, name, slug, description string, parentID *uint, sortOrder int) (*Category, error) {
	uc.log.WithContext(ctx).Debugf("[CreateCategory] 开始 name=%q slug=%q parent_id=%v sort_order=%d",
		name, slug, parentID, sortOrder)

	// ===================================================================
	// 步骤1: 校验名称和 slug
	// ===================================================================
	name = strings.TrimSpace(name)
	if name == "" {
		uc.log.WithContext(ctx).Warnf("[CreateCategory] 名称为空")
		return nil, kerrors.BadRequest("CATEGORY_NAME_EMPTY", "分类名称不能为空")
	}
	if slug == "" {
		uc.log.WithContext(ctx).Debugf("[CreateCategory] 未提供slug, 自动生成")
		slug = generateSlug(name)
	}
	uc.log.WithContext(ctx).Debugf("[CreateCategory] 校验通过 name=%q slug=%q", name, slug)

	// ===================================================================
	// 步骤2: 验证父分类存在性（如果指定了 parent_id）
	// 父分类不存在 → 返回 404
	// ===================================================================
	if parentID != nil && *parentID > 0 {
		uc.log.WithContext(ctx).Debugf("[CreateCategory] 验证父分类 parent_id=%d", *parentID)
		if _, err := uc.repo.FindByID(ctx, *parentID); err != nil {
			uc.log.WithContext(ctx).Warnf("[CreateCategory] 父分类不存在 parent_id=%d", *parentID)
			return nil, ErrCategoryNotFound
		}
		uc.log.WithContext(ctx).Debugf("[CreateCategory] 父分类存在")
	} else {
		// parentID = nil 或 0 → 作为根分类
		parentID = nil
	}

	// ===================================================================
	// 步骤3: 委托 data 层创建
	// ===================================================================
	cat := &Category{
		Name:        name,
		Slug:        slug,
		Description: description,
		ParentID:    parentID,
		SortOrder:   sortOrder,
	}
	uc.log.WithContext(ctx).Debugf("[CreateCategory] 调用 repo.Create")
	if err := uc.repo.Create(ctx, cat); err != nil {
		if kerrors.IsConflict(err) || kerrors.Code(err) == 409 {
			uc.log.WithContext(ctx).Warnf("[CreateCategory] 名称/Slug冲突 name=%q slug=%q", name, slug)
			return nil, ErrCategoryNameExists
		}
		uc.log.WithContext(ctx).Errorf("[CreateCategory] 创建失败 name=%q err=%v", name, err)
		return nil, fmt.Errorf("create category: %w", err)
	}
	uc.log.WithContext(ctx).Debugf("[CreateCategory] repo.Create 返回 id=%d", cat.ID)

	uc.log.WithContext(ctx).Infof("[CreateCategory] 创建成功 id=%d name=%q slug=%q", cat.ID, cat.Name, cat.Slug)
	return cat, nil
}

// =============================================================================
// UpdateCategory — 更新分类
//
// 流程：
//  1. 查询分类确认存在
//  2. 覆盖更新全部字段（name, slug, description, parent_id, sort_order）
//  3. 委托 data 层持久化
// =============================================================================

func (uc *CategoryUseCase) UpdateCategory(ctx context.Context, id uint, name, slug, description string, parentID *uint, sortOrder int) (*Category, error) {
	uc.log.WithContext(ctx).Debugf("[UpdateCategory] 开始 id=%d name=%q slug=%q", id, name, slug)

	// 步骤1: 查询分类确认存在
	cat, err := uc.repo.FindByID(ctx, id)
	if err != nil {
		uc.log.WithContext(ctx).Debugf("[UpdateCategory] 分类不存在 id=%d", id)
		return nil, ErrCategoryNotFound
	}
	uc.log.WithContext(ctx).Debugf("[UpdateCategory] 分类找到 id=%d old_name=%q", id, cat.Name)

	// ===================================================================
	// 步骤2: 校验新值
	// ===================================================================
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, kerrors.BadRequest("CATEGORY_NAME_EMPTY", "分类名称不能为空")
	}
	if slug == "" {
		return nil, ErrCategorySlugEmpty
	}

	// ===================================================================
	// 步骤3: 覆盖更新字段
	// ===================================================================
	// 步骤2: 覆盖更新字段
	// 不允许把自己设为子分类（parent_id 不能指向自己）
	// ===================================================================
	if parentID != nil && *parentID > 0 {
		if *parentID == id {
			uc.log.WithContext(ctx).Warnf("[UpdateCategory] 不能将自己设为父分类 id=%d", id)
			return nil, kerrors.BadRequest("CATEGORY_SELF_PARENT", "不能将分类设为自己的子分类")
		}
		// 检查循环引用：从新父分类向上溯源，不能出现当前分类
		if err := uc.checkCircularRef(ctx, id, *parentID); err != nil {
			uc.log.WithContext(ctx).Warnf("[UpdateCategory] 循环引用 id=%d new_parent=%d", id, *parentID)
			return nil, err
		}
		cat.ParentID = parentID
	} else {
		cat.ParentID = nil
	}
	cat.Name = name
	cat.Slug = slug
	cat.Description = description
	cat.SortOrder = sortOrder
	uc.log.WithContext(ctx).Debugf("[UpdateCategory] 字段更新完成 parent_id=%v sort_order=%d", cat.ParentID, cat.SortOrder)

	// 步骤4: 委托 data 层持久化
	uc.log.WithContext(ctx).Debugf("[UpdateCategory] 调用 repo.Update")
	if err := uc.repo.Update(ctx, cat); err != nil {
		uc.log.WithContext(ctx).Errorf("[UpdateCategory] 更新失败 id=%d err=%v", id, err)
		return nil, fmt.Errorf("update category: %w", err)
	}

	uc.log.WithContext(ctx).Infof("[UpdateCategory] 更新成功 id=%d name=%q", id, cat.Name)
	return cat, nil
}

// =============================================================================
// DeleteCategory — 删除分类
//
// 流程（2道防线）：
//  1. 检查是否存在子分类 → 有则返回 ErrCategoryHasChildren
//  2. 检查是否存在关联文章 → 有则返回 ErrCategoryHasArticles
//  3. 委托 data 层软删除
// =============================================================================

func (uc *CategoryUseCase) DeleteCategory(ctx context.Context, id uint) error {
	uc.log.WithContext(ctx).Debugf("[DeleteCategory] 开始 id=%d", id)

	// ===================================================================
	// 防线1: 检查是否存在子分类
	// ListChildren 查询 parent_id = id 的直接子分类
	// ===================================================================
	uc.log.WithContext(ctx).Debugf("[DeleteCategory] 检查子分类")
	children, err := uc.repo.ListChildren(ctx, id)
	if err != nil {
		uc.log.WithContext(ctx).Errorf("[DeleteCategory] 查询子分类失败 id=%d err=%v", id, err)
		return fmt.Errorf("check children: %w", err)
	}
	if len(children) > 0 {
		uc.log.WithContext(ctx).Warnf("[DeleteCategory] 存在子分类 id=%d child_count=%d", id, len(children))
		return ErrCategoryHasChildren
	}
	uc.log.WithContext(ctx).Debugf("[DeleteCategory] 无子分类")

	// ===================================================================
	// 防线2: 检查是否存在关联文章
	// 通过数据层查询分类下的文章数（article_count 可被系统维护，但以 DB 查询为准）
	// ===================================================================
	uc.log.WithContext(ctx).Debugf("[DeleteCategory] 检查关联文章")
	cat, err := uc.repo.FindByID(ctx, id)
	if err != nil {
		uc.log.WithContext(ctx).Debugf("[DeleteCategory] 分类不存在 id=%d", id)
		return ErrCategoryNotFound
	}
	if cat.ArticleCount > 0 {
		uc.log.WithContext(ctx).Warnf("[DeleteCategory] 分类下有文章 id=%d article_count=%d", id, cat.ArticleCount)
		return ErrCategoryHasArticles
	}
	uc.log.WithContext(ctx).Debugf("[DeleteCategory] 无关联文章 article_count=%d", cat.ArticleCount)

	// ===================================================================
	// 步骤3: 委托 data 层软删除
	// ===================================================================
	uc.log.WithContext(ctx).Debugf("[DeleteCategory] 调用 repo.Delete id=%d", id)
	if err := uc.repo.Delete(ctx, id); err != nil {
		uc.log.WithContext(ctx).Errorf("[DeleteCategory] 删除失败 id=%d err=%v", id, err)
		return fmt.Errorf("delete category: %w", err)
	}

	uc.log.WithContext(ctx).Infof("[DeleteCategory] 删除成功 id=%d", id)
	return nil
}

// =============================================================================
// ListCategories — 查询完整分类树
//
// data 层：
//   1. 查询全量分类
//   2. 按 ParentID 组装为树结构（Children 字段填充）
//   3. Cache-Aside 缓存（TTL 60min）
// =============================================================================

func (uc *CategoryUseCase) ListCategories(ctx context.Context) ([]*Category, error) {
	uc.log.WithContext(ctx).Debugf("[ListCategories] 开始")
	tree, err := uc.repo.ListTree(ctx)
	if err != nil {
		uc.log.WithContext(ctx).Errorf("[ListCategories] 查询失败 err=%v", err)
		return nil, fmt.Errorf("list categories: %w", err)
	}
	uc.log.WithContext(ctx).Debugf("[ListCategories] 完成 root_count=%d", len(tree))
	return tree, nil
}

// =============================================================================
// checkCircularRef — 检查分类循环引用
//
// 从 newParentID 出发沿 parent 链向上溯源，若途中遇到 selfID 则构成循环。
// 例如已有 A → B → C，若将 A 的 parent 设为 C，则沿 C → B → A 查到自己 → 拒绝。
// 最大溯源深度 100 层，超出也视为异常拒绝。
// =============================================================================

func (uc *CategoryUseCase) checkCircularRef(ctx context.Context, selfID, parentID uint) error {
	currentID := parentID
	for i := 0; i < 100; i++ {
		if currentID == selfID {
			return ErrCategoryCircularRef
		}
		parent, err := uc.repo.FindByID(ctx, currentID)
		if err != nil {
			return nil // 父链中断（分类不存在），安全
		}
		if parent.ParentID == nil {
			return nil // 到达根分类，安全
		}
		currentID = *parent.ParentID
	}
	return kerrors.InternalServer("CIRCULAR_CHECK_DEPTH", "分类层级超过最大限制(100层)")
}
