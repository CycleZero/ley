package biz

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode"

	"ley/pkg/eventbus"
	"ley/pkg/meta"

	kerrors "github.com/go-kratos/kratos/v2/errors"
	"github.com/go-kratos/kratos/v2/log"
)

// =============================================================================
// ArticleStatus — 文章状态枚举
// 草稿(0): 仅作者可见，不计入分类/标签统计
// 已发布(1): 公开可见，计入分类/标签的文章计数
// 已归档(2): 不再展示在默认列表，计数从分类/标签中移除
// =============================================================================

type ArticleStatus int8

const (
	ArticleStatusDraft     ArticleStatus = 0 // 草稿
	ArticleStatusPublished ArticleStatus = 1 // 已发布
	ArticleStatusArchived  ArticleStatus = 2 // 已归档
)

// =============================================================================
// Article — 文章业务模型
//
// 这是 biz 层的领域实体，不包含任何持久化标签（GORM tag 等由 data 层的 PO 持有）。
// 对外暴露的字段含义：
//   - ID: 数据库自增主键，用作 API 标识和内部关联键
//   - Slug: URL 友好的唯一标识符，由标题自动生成（拼音首字母 + 连字符）
//   - AuthorID/AuthorName/AuthorAvatar: 作者信息由 data 层在查询时批量填充
//   - CategoryID/CategoryName: 分类信息由 data 层在查询时关联填充
//   - Tags: 标签列表，data 层通过 articles_tags 中间表填充
//   - IsLiked: 当前请求用户是否已点赞（未认证时为 false），由 GetArticle 时动态计算
//   - PublishedAt: 首次发布时间，仅在发布时写入一次，后续更新不重设
// =============================================================================

type Article struct {
	ID             uint           // 主键
	Title          string         // 标题 (2-200字符)
	Slug           string         // URL slug (唯一)
	Content        string         // 正文 (Markdown)
	Excerpt        string         // 摘要 (自动截取前500字符)
	CoverImage     string         // 封面图URL
	Status         ArticleStatus  // 当前状态
	AuthorID       uint           // 作者主键
	AuthorName     string         // 作者显示名 (data层关联填充)
	AuthorAvatar   string         // 作者头像 (data层关联填充)
	CategoryID     *uint          // 分类主键 (可为nil)
	CategoryName   string         // 分类名 (data层关联填充)
	Tags           []*Tag         // 标签列表
	ViewCount      int64          // 浏览数
	LikeCount      int64          // 点赞数
	CommentCount   int64          // 评论数
	IsTop          bool           // 是否置顶
	IsLiked        bool           // 当前用户是否已点赞 (运行时计算)
	PublishedAt    *time.Time     // 首次发布时间
	CreatedAt      time.Time      // 创建时间
	UpdatedAt      time.Time      // 最后更新时间
}

// =============================================================================
// ArticleListQuery — 列表查询参数
//
// 封装列表查询所需的所有过滤、排序和分页参数。
// Status 为空时默认查询已发布文章。
// Tags 采用 AND 逻辑：文章必须同时拥有所有指定标签。
// =============================================================================

type ArticleListQuery struct {
	Status     string   // 文章状态过滤: draft / published / archived
	CategoryID *uint    // 分类ID过滤
	Tags       []string // 标签名称过滤 (AND 逻辑)
	AuthorID   *uint    // 作者ID过滤
	SortBy     string   // 排序字段: created_at / updated_at / published_at / view_count / is_top
	SortOrder  string   // 排序方向: asc / desc
	Page       int      // 页码 (从1开始)
	PageSize   int      // 每页数量
}

// =============================================================================
// ArticleRepo — 文章数据访问接口（依赖倒置原则）
//
// biz 层只依赖此接口，data 层提供具体实现（GORM + Redis 缓存）。
// 所有涉及事务一致性的操作（如发布时同时更新文章状态和分类计数）
// 由 data 层在一个事务中完成。
// =============================================================================

type ArticleRepo interface {
	// 基础 CRUD
	Create(ctx context.Context, article *Article) error
	Update(ctx context.Context, article *Article) error
	Delete(ctx context.Context, id uint) error
	FindByID(ctx context.Context, id uint) (*Article, error)
	FindBySlug(ctx context.Context, slug string) (*Article, error)
	List(ctx context.Context, query ArticleListQuery) ([]*Article, int64, error)

	// 标签关联
	AssociateTags(ctx context.Context, articleID uint, tagIDs []uint) error
	SyncTags(ctx context.Context, articleID uint, tagIDs []uint) error
	ListTagsByArticleID(ctx context.Context, articleID uint) ([]*Tag, error)

	// 点赞（幂等实现：ON CONFLICT DO NOTHING / DELETE WHERE exists）
	InsertLike(ctx context.Context, articleID, userID uint) error
	DeleteLike(ctx context.Context, articleID, userID uint) error
	IsLiked(ctx context.Context, articleID, userID uint) (bool, error)

	// 原子计数更新
	IncrementViewCount(ctx context.Context, id uint, delta int64) error
	IncrementCommentCount(ctx context.Context, id uint, delta int64) error

	// 分类/标签计数同步（由 data 层在事务中与主更新一起执行）
	UpdateCategoryCount(ctx context.Context, categoryID uint, delta int64) error
	UpdateTagsArticleCount(ctx context.Context, tagIDs []uint, delta int64) error
}

// =============================================================================
// ArticleUseCase — 文章业务用例
//
// 封装文章创建、编辑、发布、归档、删除等全部业务逻辑。
// 依赖:
//   - ArticleRepo: 文章持久化 + 计数更新
//   - TagRepo: 标签查找/创建（CreateArticle / UpdateArticle 时解析标签名）
//   - CategoryRepo: 分类计数更新（发布/归档/删除时调整 article_count）
//   - EventBus: 异步发布生命周期事件（article.published / article.liked / article.viewed）
//
// 设计原则:
//   - 所有业务规则（校验、状态机、权限）在 biz 层实现
//   - 所有持久化和缓存逻辑在 data 层实现
//   - 草稿不计入分类/标签的 article_count，发布时才 +1
// =============================================================================

type ArticleUseCase struct {
	repo    ArticleRepo       // 文章数据访问
	tagRepo TagRepo           // 标签数据访问
	catRepo CategoryRepo      // 分类数据访问
	eb      eventbus.EventBus // 事件总线
	log     *log.Helper       // 结构化日志
}

// NewArticleUseCase 构造函数（由 Wire 调用注入依赖）
func NewArticleUseCase(repo ArticleRepo, tagRepo TagRepo, catRepo CategoryRepo, eb eventbus.EventBus, logger log.Logger) *ArticleUseCase {
	return &ArticleUseCase{repo: repo, tagRepo: tagRepo, catRepo: catRepo, eb: eb, log: log.NewHelper(logger)}
}

// =============================================================================
// 业务错误定义（Kratos errors，每个错误自带 HTTP 状态码）
//
// service 层直接 return err 即可，无需额外映射。
//   - BadRequest (400): 输入校验失败
//   - Unauthorized (401): 未认证
//   - Forbidden (403): 权限不足
//   - NotFound (404): 资源不存在
//   - Conflict (409): 状态冲突
// =============================================================================

var (
	ErrArticleNotFound         = kerrors.NotFound("ARTICLE_NOT_FOUND", "文章不存在")
	ErrArticleTitleInvalid     = kerrors.BadRequest("TITLE_INVALID", "文章标题须为 2-200 个字符")
	ErrArticleContentEmpty     = kerrors.BadRequest("CONTENT_EMPTY", "文章内容不能为空")
	ErrArticleContentTooBig    = kerrors.BadRequest("CONTENT_TOO_BIG", "文章内容不能超过 100000 个字符")
	ErrNotArticleOwner         = kerrors.Forbidden("NOT_ARTICLE_OWNER", "只能操作自己的文章")
	ErrArticleAlreadyPublished = kerrors.Conflict("ALREADY_PUBLISHED", "文章已发布")
	ErrUserNotAuthenticated    = kerrors.Unauthorized("NOT_AUTHENTICATED", "用户未认证")
)

// =============================================================================
// 字段长度常量
// =============================================================================

const (
	MinTitleLength   = 2      // 标题最小字符数
	MaxTitleLength   = 200    // 标题最大字符数
	MaxContentLength = 100000 // 内容最大字符数
	MaxExcerptLength = 500    // 摘要最大字符数（自动截断时使用）
)

// =============================================================================
// CreateArticle — 创建文章（草稿状态）
//
// 流程步骤（共7步）：
//  1. 从 context 提取当前登录用户 ID（由 Gateway JWT 中间件注入）
//  2. 校验标题长度 (2-200) 和内容非空且不超过 100000 字符
//  3. 根据标题生成 URL slug：
//     a. 英文字母 → 小写保留，数字保留
//     b. 中文字符 → 查表转拼音首字母
//     c. 其他字符 → 替换为连字符 '-'
//     d. 去除重复连字符和首尾连字符
//     e. 通过 FindBySlug 检查唯一性，冲突时追加 "-2", "-3"...
//  4. 如果未提供摘要，从正文字符数截取前 500 字符并追加 "..."
//  5. 解析标签名称：遍历 tagNames，对每个名称执行 FindOrCreate
//  6. 构造 Article 实体（状态 = Draft）并写入数据库
//  7. 关联标签到 articles_tags 中间表
//
// 注意：
//   - 草稿不更新分类/article.article_count 和标签的 article_count
//   - 这些计数仅在 PublishArticle 时 +1
// =============================================================================

func (uc *ArticleUseCase) CreateArticle(ctx context.Context, title, content, excerpt, coverImage string, categoryID *uint, tagNames []string) (*Article, error) {
	// ===================================================================
	// 步骤1: 从 context 中提取当前登录用户 ID
	// 该值由 Gateway 的 JWT 验证中间件在请求到达前注入到 context.Value("user_id")
	// ===================================================================
	authorID, err := getCurrentUserID(ctx)
	if err != nil {
		uc.log.WithContext(ctx).Debugf("[CreateArticle] 步骤1失败: 无法从context提取user_id: %v", err)
		return nil, ErrUserNotAuthenticated
	}
	uc.log.WithContext(ctx).Debugf("[CreateArticle] 步骤1完成: author_id=%d", authorID)

	// ===================================================================
	// 步骤2: 输入校验
	// 校验标题长度 (2-200字符) 和内容 (非空且不超过100000字符)
	// 校验不通过立即返回 BadRequest 错误
	// ===================================================================
	uc.log.WithContext(ctx).Debugf("[CreateArticle] 步骤2: 开始输入校验 title_len=%d content_len=%d tag_count=%d",
		len(title), len(content), len(tagNames))
	if err := validateArticleInput(title, content); err != nil {
		uc.log.WithContext(ctx).Warnf("[CreateArticle] 步骤2校验失败: title=%q content_len=%d err=%v", title, len(content), err)
		return nil, err
	}
	uc.log.WithContext(ctx).Debugf("[CreateArticle] 步骤2完成: 输入校验通过")

	// ===================================================================
	// 步骤3: 生成唯一 Slug
	// 从标题生成拼音 slug，然后通过 ensureUniqueSlug 循环重试确保唯一
	// ===================================================================
	uc.log.WithContext(ctx).Debugf("[CreateArticle] 步骤3: 开始生成slug, 原始标题=%q", title)

	slug := generateSlug(title)
	uc.log.WithContext(ctx).Debugf("[CreateArticle] 步骤3a: 拼音转换完成 title=%q → raw_slug=%q", title, slug)

	// ensureUniqueSlug 检查 slug 是否已被占用，最多重试100次追加数字后缀 "-2", "-3"...
	slug, err = ensureUniqueSlug(ctx, slug, uc.repo)
	if err != nil {
		uc.log.WithContext(ctx).Errorf("[CreateArticle] 步骤3失败: slug唯一性检查耗尽 title=%q err=%v", title, err)
		return nil, fmt.Errorf("slug: %w", err)
	}
	uc.log.WithContext(ctx).Debugf("[CreateArticle] 步骤3完成: 最终 slug=%q", slug)

	// ===================================================================
	// 步骤4: 自动生成摘要（如果调用方未提供）
	// 从正文内容截取前 500 个字符（按 rune 计数，正确处理中文等多字节字符），追加 "..."
	// ===================================================================
	if excerpt == "" {
		uc.log.WithContext(ctx).Debugf("[CreateArticle] 步骤4: 未提供摘要，自动从正文生成 content_len=%d", len(content))
		excerpt = truncateContent(content, MaxExcerptLength)
		uc.log.WithContext(ctx).Debugf("[CreateArticle] 步骤4完成: 自动摘要 excerpt_len=%d", len(excerpt))
	} else {
		uc.log.WithContext(ctx).Debugf("[CreateArticle] 步骤4: 使用调用方提供的摘要 excerpt_len=%d", len(excerpt))
	}

	// ===================================================================
	// 步骤5: 解析标签名称
	// 对每个 tagName 执行 FindOrCreate：
	//   - 去除首尾空白
	//   - 跳过空字符串
	//   - 按名称查找 → 存在则直接返回
	//   - 不存在则创建（ON CONFLICT DO NOTHING，防并发重复）
	// ===================================================================
	uc.log.WithContext(ctx).Debugf("[CreateArticle] 步骤5: 开始解析标签 names=%v", tagNames)
	tags, err := resolveTags(ctx, tagNames, uc.tagRepo)
	if err != nil {
		uc.log.WithContext(ctx).Errorf("[CreateArticle] 步骤5失败: 标签解析出错 names=%v err=%v", tagNames, err)
		return nil, err
	}
	uc.log.WithContext(ctx).Debugf("[CreateArticle] 步骤5完成: 解析到 %d 个标签", len(tags))

	// ===================================================================
	// 步骤6: 构造 Article 实体并写入数据库
	// 状态设为 Draft（ArticleStatusDraft = 0）
	// AuthorID 设为当前登录用户
	// 创建失败时，如果是唯一冲突（slug重复），直接返回给调用方
	// 其他错误包装后返回
	// ===================================================================
	uc.log.WithContext(ctx).Debugf("[CreateArticle] 步骤6: 构造 Article 实体 author_id=%d category_id=%v tag_count=%d",
		authorID, categoryID, len(tags))

	article := &Article{
		Title:      title,
		Slug:       slug,
		Content:    content,
		Excerpt:    excerpt,
		CoverImage: coverImage,
		Status:     ArticleStatusDraft,
		AuthorID:   uint(authorID),
		CategoryID: normalizeCategoryID(categoryID),
		Tags:       tags,
	}

	uc.log.WithContext(ctx).Debugf("[CreateArticle] 步骤6a: 调用 repo.Create article_slug=%q article_title=%q",
		article.Slug, article.Title)
	if err := uc.repo.Create(ctx, article); err != nil {
		// 判断是否为唯一键冲突（slug重复）。Kratos Conflict 错误码为 409
		if kerrors.IsConflict(err) || kerrors.Code(err) == 409 {
			uc.log.WithContext(ctx).Warnf("[CreateArticle] 步骤6失败: 唯一键冲突 slug=%q err=%v", slug, err)
			return nil, err
		}
		uc.log.WithContext(ctx).Errorf("[CreateArticle] 步骤6失败: 数据库写入错误 title=%q err=%v", title, err)
		return nil, fmt.Errorf("create article: %w", err)
	}
	uc.log.WithContext(ctx).Debugf("[CreateArticle] 步骤6b: repo.Create 返回, 获得新ID=%d", article.ID)

	// ===================================================================
	// 步骤7: 关联标签到 articles_tags 中间表
	// 仅在存在至少一个标签时执行
	// 关联失败不阻塞主流程（标签可以后续手动添加）
	// ===================================================================
	if len(tags) > 0 {
		tagIDList := tagIDs(tags)
		uc.log.WithContext(ctx).Debugf("[CreateArticle] 步骤7: 关联标签 article_id=%d tag_ids=%v", article.ID, tagIDList)
		if err := uc.repo.AssociateTags(ctx, article.ID, tagIDList); err != nil {
			// 标签关联失败记录 Warn（标签可后续添加），不阻塞文章创建成功
			uc.log.WithContext(ctx).Warnf("[CreateArticle] 步骤7警告: 标签关联失败 article_id=%d err=%v", article.ID, err)
		} else {
			uc.log.WithContext(ctx).Debugf("[CreateArticle] 步骤7完成: 标签关联成功 article_id=%d tag_count=%d", article.ID, len(tags))
		}
	} else {
		uc.log.WithContext(ctx).Debugf("[CreateArticle] 步骤7跳过: 无标签需要关联")
	}

	uc.log.WithContext(ctx).Infof("[CreateArticle] 创建成功 id=%d slug=%q author_id=%d status=draft", article.ID, article.Slug, article.AuthorID)
	return article, nil
}

// =============================================================================
// UpdateArticle — 更新文章（部分字段更新）
//
// 调用方通过指针传递需要更新的字段：
//   - *string 非 nil → 更新此字段
//   - *string 为 nil → 保持原值不变
//
// 流程步骤：
//  1. 权限校验：验证当前用户是否为文章作者
//  2. 标题更新：校验新标题 → 重新生成 slug
//  3. 内容更新：校验长度不超过 100000
//  4. 摘要/封面/分类更新：直接赋值
//  5. 持久化到数据库
//  6. 若 tagNames 非 nil，全量替换标签关联（先删后增）
// =============================================================================

func (uc *ArticleUseCase) UpdateArticle(ctx context.Context, id uint, title, content, excerpt, coverImage *string, categoryID *uint, tagNames []string) (*Article, error) {
	uc.log.WithContext(ctx).Debugf("[UpdateArticle] 开始 article_id=%d has_title=%v has_content=%v has_excerpt=%v has_cover=%v has_category=%v has_tags=%v",
		id, title != nil, content != nil, excerpt != nil, coverImage != nil, categoryID != nil, tagNames != nil)

	// ===================================================================
	// 步骤1: 权限校验
	// 查询文章 → 校验当前用户是否为作者 → 非作者返回 403
	// ===================================================================
	article, err := uc.authorizeArticle(ctx, id)
	if err != nil {
		uc.log.WithContext(ctx).Warnf("[UpdateArticle] 步骤1权限校验失败 article_id=%d err=%v", id, err)
		return nil, err
	}
	uc.log.WithContext(ctx).Debugf("[UpdateArticle] 步骤1权限校验通过 author_id=%d requester_id=%d", article.AuthorID, article.AuthorID)

	// ===================================================================
	// 步骤2: 标题更新
	// 标题变更时：校验新标题长度 → 重新生成 slug（确保唯一）
	// slug 生成失败时返回错误（不静默忽略）
	// ===================================================================
	if title != nil {
		uc.log.WithContext(ctx).Debugf("[UpdateArticle] 步骤2: 标题变更 old=%q new=%q", article.Title, *title)
		if err := validateTitle(*title); err != nil {
			uc.log.WithContext(ctx).Warnf("[UpdateArticle] 步骤2失败: 标题校验不通过 title=%q len=%d err=%v", *title, len(*title), err)
			return nil, err
		}
		article.Title = *title
		// 根据新标题生成 slug 并确保唯一
		newSlug, err := ensureUniqueSlug(ctx, generateSlug(*title), uc.repo)
		if err != nil {
			uc.log.WithContext(ctx).Errorf("[UpdateArticle] 步骤2失败: slug生成失败 title=%q err=%v", *title, err)
			return nil, fmt.Errorf("slug: %w", err)
		}
		uc.log.WithContext(ctx).Debugf("[UpdateArticle] 步骤2完成: slug变更 old=%q new=%q", article.Slug, newSlug)
		article.Slug = newSlug
	} else {
		uc.log.WithContext(ctx).Debugf("[UpdateArticle] 步骤2跳过: 标题未变更")
	}

	// ===================================================================
	// 步骤3: 内容更新
	// 校验新内容长度不超过 MaxContentLength (100000)
	// ===================================================================
	if content != nil {
		contentLen := len(*content)
		uc.log.WithContext(ctx).Debugf("[UpdateArticle] 步骤3: 内容变更 old_len=%d new_len=%d", len(article.Content), contentLen)
		if contentLen > MaxContentLength {
			uc.log.WithContext(ctx).Warnf("[UpdateArticle] 步骤3失败: 内容超长 len=%d max=%d", contentLen, MaxContentLength)
			return nil, ErrArticleContentTooBig
		}
		article.Content = *content
		uc.log.WithContext(ctx).Debugf("[UpdateArticle] 步骤3完成: 内容已更新")
	} else {
		uc.log.WithContext(ctx).Debugf("[UpdateArticle] 步骤3跳过: 内容未变更")
	}

	// ===================================================================
	// 步骤4: 摘要、封面图、分类更新
	// 这些字段无需额外校验，直接赋值
	// ===================================================================
	if excerpt != nil {
		uc.log.WithContext(ctx).Debugf("[UpdateArticle] 步骤4a: 摘要变更 old_len=%d new_len=%d", len(article.Excerpt), len(*excerpt))
		article.Excerpt = *excerpt
	}
	if coverImage != nil {
		uc.log.WithContext(ctx).Debugf("[UpdateArticle] 步骤4b: 封面图变更 old=%q new=%q", article.CoverImage, *coverImage)
		article.CoverImage = *coverImage
	}
	if categoryID != nil {
		// 按 proto 约定，uint64 category_id = 0 表示"移除分类"
		// 此时文章不再有关联分类（CategoryID 设为 nil）
		isClearing := *categoryID == 0
		uc.log.WithContext(ctx).Debugf("[UpdateArticle] 步骤4c: 分类变更 old=%v new=%v clearing=%v",
			article.CategoryID, *categoryID, isClearing)

		// 已发布文章变更分类（包括移除），需调整新旧计数
		if article.Status == ArticleStatusPublished && article.CategoryID != nil {
			uc.log.WithContext(ctx).Debugf("[UpdateArticle] 步骤4c-计数: 旧分类-1 category_id=%d",
				*article.CategoryID)
			_ = uc.catRepo.IncrementArticleCount(ctx, *article.CategoryID, -1)
		}

		// 只有设置到有效分类时才给新分类 +1（移除分类时不 +1）
		if !isClearing && article.Status == ArticleStatusPublished {
			uc.log.WithContext(ctx).Debugf("[UpdateArticle] 步骤4c-计数: 新分类+1 category_id=%d",
				*categoryID)
			_ = uc.catRepo.IncrementArticleCount(ctx, *categoryID, 1)
		}

		// 更新 article 的 CategoryID
		if isClearing {
			article.CategoryID = nil
		} else {
			article.CategoryID = categoryID
		}
		uc.log.WithContext(ctx).Debugf("[UpdateArticle] 步骤4c完成: CategoryID=%v", article.CategoryID)
	}

	// ===================================================================
	// 步骤5: 持久化到数据库
	// data 层使用 map[string]interface{} 更新，避免 GORM struct 零值跳过问题
	// ===================================================================
	uc.log.WithContext(ctx).Debugf("[UpdateArticle] 步骤5: 调用 repo.Update article_id=%d", id)
	if err := uc.repo.Update(ctx, article); err != nil {
		uc.log.WithContext(ctx).Errorf("[UpdateArticle] 步骤5失败: 数据库更新失败 article_id=%d err=%v", id, err)
		return nil, fmt.Errorf("update article: %w", err)
	}
	uc.log.WithContext(ctx).Debugf("[UpdateArticle] 步骤5完成: repo.Update 成功")

	// ===================================================================
	// 步骤6: 标签全量替换
	// tagNames 为 nil → 不做任何操作（保持原有标签不变）
	// tagNames 为非 nil 空切片 → 删除所有标签
	// tagNames 为非 nil 有值 → 全量替换为新标签列表
	// ===================================================================
	if tagNames != nil {
		uc.log.WithContext(ctx).Debugf("[UpdateArticle] 步骤6: 标签变更 names=%v", tagNames)
		// 解析标签名称列表（FindOrCreate），得到完整的 Tag 对象（含 ID）
		tags, err := resolveTags(ctx, tagNames, uc.tagRepo)
		if err != nil {
			uc.log.WithContext(ctx).Errorf("[UpdateArticle] 步骤6失败: 标签解析失败 names=%v err=%v", tagNames, err)
			return nil, err
		}
		// SyncTags 在事务中先删除旧关联再插入新关联
		if err := uc.repo.SyncTags(ctx, id, tagIDs(tags)); err != nil {
			uc.log.WithContext(ctx).Errorf("[UpdateArticle] 步骤6失败: 标签同步失败 article_id=%d err=%v", id, err)
			return nil, fmt.Errorf("sync tags: %w", err)
		}
		uc.log.WithContext(ctx).Debugf("[UpdateArticle] 步骤6完成: 标签已同步 new_count=%d", len(tags))
	} else {
		uc.log.WithContext(ctx).Debugf("[UpdateArticle] 步骤6跳过: tagNames为nil, 标签不变")
	}

	uc.log.WithContext(ctx).Infof("[UpdateArticle] 更新成功 id=%d title=%q", article.ID, article.Title)

	// 步骤7: 异步发布更新事件
	_ = uc.eb.PublishAsync(ctx, TopicArticleUpdated, &ArticleUpdatedEvent{
		ArticleID: uint64(article.ID),
		Title:     article.Title,
		AuthorID:  uint64(article.AuthorID),
	})

	return article, nil
}

// =============================================================================
// DeleteArticle — 软删除文章
//
// 流程步骤：
//  1. 权限校验：只有文章作者可以删除
//  2. 执行软删除（GORM 自动设置 deleted_at）
//  3. 如果文章当前为已发布状态：
//     a. 分类 article_count -1（使用 GREATEST 防止变为负数）
//     b. 标签 article_count -1（同上）
//
// 注意：计数调整失败不阻塞删除（记录 Warn 日志即可）
// =============================================================================

func (uc *ArticleUseCase) DeleteArticle(ctx context.Context, id uint) error {
	uc.log.WithContext(ctx).Debugf("[DeleteArticle] 开始 article_id=%d", id)

	// 步骤1: 权限与存在性校验
	article, err := uc.authorizeArticle(ctx, id)
	if err != nil {
		uc.log.WithContext(ctx).Warnf("[DeleteArticle] 权限校验失败 article_id=%d err=%v", id, err)
		return err
	}
	uc.log.WithContext(ctx).Debugf("[DeleteArticle] 步骤1权限通过 article_id=%d author_id=%d status=%d", id, article.AuthorID, article.Status)

	// 步骤2: 执行软删除
	uc.log.WithContext(ctx).Debugf("[DeleteArticle] 步骤2: 执行软删除 article_id=%d", id)
	if err := uc.repo.Delete(ctx, id); err != nil {
		uc.log.WithContext(ctx).Errorf("[DeleteArticle] 删除失败 article_id=%d err=%v", id, err)
		return fmt.Errorf("delete article: %w", err)
	}
	uc.log.WithContext(ctx).Debugf("[DeleteArticle] 步骤2完成: 软删除成功")

	// 步骤3: 如果文章已发布，调整分类和标签的计数
	if article.Status == ArticleStatusPublished {
		uc.log.WithContext(ctx).Debugf("[DeleteArticle] 步骤3: 文章原为发布状态, 调整关联计数 category_id=%v tag_count=%d",
			article.CategoryID, len(article.Tags))

		if article.CategoryID != nil {
			uc.log.WithContext(ctx).Debugf("[DeleteArticle] 步骤3a: 分类计数-1 category_id=%d", *article.CategoryID)
			if err := uc.catRepo.IncrementArticleCount(ctx, *article.CategoryID, -1); err != nil {
				uc.log.WithContext(ctx).Warnf("[DeleteArticle] 分类计数调整失败 category_id=%d err=%v",
					*article.CategoryID, err)
			}
		}
		if len(article.Tags) > 0 {
			uc.log.WithContext(ctx).Debugf("[DeleteArticle] 步骤3b: 标签计数-1 tag_ids=%v", tagIDs(article.Tags))
			if err := uc.repo.UpdateTagsArticleCount(ctx, tagIDs(article.Tags), -1); err != nil {
				uc.log.WithContext(ctx).Warnf("[DeleteArticle] 标签计数调整失败 tag_count=%d err=%v",
					len(article.Tags), err)
			}
		}
	} else {
		uc.log.WithContext(ctx).Debugf("[DeleteArticle] 步骤3跳过: 文章非发布状态 status=%d", article.Status)
	}

	uc.log.WithContext(ctx).Infof("[DeleteArticle] 删除成功 id=%d", id)

	// 步骤4: 异步发布删除事件
	_ = uc.eb.PublishAsync(ctx, TopicArticleDeleted, &ArticleDeletedEvent{
		ArticleID: uint64(id),
		AuthorID:  uint64(article.AuthorID),
	})

	return nil
}

// =============================================================================
// PublishArticle — 发布文章
//
// 流程步骤：
//  1. 权限校验：只有文章作者可以发布
//  2. 状态校验：已发布的文章不可重复发布（返回 409 Conflict）
//  3. 变更状态为 Published，同时设置 published_at = NOW()
//  4. 持久化状态变更
//  5. 分类 article_count +1（仅在有分类时）
//  6. 标签 article_count +1（逐一原子更新）
//  7. 异步发布 article.published 事件（Notification 服务消费 → 邮件推送）
//
// 计数更新失败不阻塞发布（记录 Warn 日志），因为计数可通过定时任务修复。
// =============================================================================

func (uc *ArticleUseCase) PublishArticle(ctx context.Context, id uint) (*Article, error) {
	uc.log.WithContext(ctx).Debugf("[PublishArticle] 开始 article_id=%d", id)

	// 步骤1: 权限与存在性校验
	article, err := uc.authorizeArticle(ctx, id)
	if err != nil {
		uc.log.WithContext(ctx).Warnf("[PublishArticle] 权限校验失败 article_id=%d err=%v", id, err)
		return nil, err
	}
	uc.log.WithContext(ctx).Debugf("[PublishArticle] 步骤1权限通过 article_id=%d author_id=%d current_status=%d", id, article.AuthorID, article.Status)

	// 步骤2: 状态校验 — 已发布不可重复发布
	if article.Status == ArticleStatusPublished {
		uc.log.WithContext(ctx).Warnf("[PublishArticle] 步骤2失败: 文章已发布 article_id=%d", id)
		return nil, ErrArticleAlreadyPublished
	}

	// 步骤3: 变更状态 + 记录首次发布时间
	previousStatus := article.Status
	now := time.Now()
	article.Status = ArticleStatusPublished
	article.PublishedAt = &now
	uc.log.WithContext(ctx).Debugf("[PublishArticle] 步骤3: 状态变更 %d→%d published_at=%s",
		previousStatus, article.Status, now.Format(time.RFC3339))

	// 步骤4: 持久化
	uc.log.WithContext(ctx).Debugf("[PublishArticle] 步骤4: 调用 repo.Update article_id=%d", id)
	if err := uc.repo.Update(ctx, article); err != nil {
		uc.log.WithContext(ctx).Errorf("[PublishArticle] 步骤4失败: 数据库更新失败 article_id=%d err=%v", id, err)
		return nil, fmt.Errorf("publish article: %w", err)
	}
	uc.log.WithContext(ctx).Debugf("[PublishArticle] 步骤4完成: 状态已持久化")

	// 步骤5: 分类计数 +1
	if article.CategoryID != nil {
		uc.log.WithContext(ctx).Debugf("[PublishArticle] 步骤5: 分类计数+1 category_id=%d", *article.CategoryID)
		if err := uc.catRepo.IncrementArticleCount(ctx, *article.CategoryID, 1); err != nil {
			uc.log.WithContext(ctx).Warnf("[PublishArticle] 步骤5警告: 分类计数更新失败 category_id=%d err=%v",
				*article.CategoryID, err)
		}
	} else {
		uc.log.WithContext(ctx).Debugf("[PublishArticle] 步骤5跳过: 无分类")
	}

	// 步骤6: 标签计数 +1
	if len(article.Tags) > 0 {
		tagIDList := tagIDs(article.Tags)
		uc.log.WithContext(ctx).Debugf("[PublishArticle] 步骤6: 标签计数+1 tag_ids=%v count=%d", tagIDList, len(tagIDList))
		if err := uc.repo.UpdateTagsArticleCount(ctx, tagIDList, 1); err != nil {
			uc.log.WithContext(ctx).Warnf("[PublishArticle] 步骤6警告: 标签计数更新失败 tag_count=%d err=%v",
				len(tagIDList), err)
		}
	} else {
		uc.log.WithContext(ctx).Debugf("[PublishArticle] 步骤6跳过: 无标签")
	}

	// 步骤7: 异步发布 article.published 事件
	// 事件消费者: Notification 服务 → 邮件推送给订阅者
	uc.log.WithContext(ctx).Debugf("[PublishArticle] 步骤7: 发布事件 topic=%s article_id=%d", TopicArticlePublished, article.ID)
	_ = uc.eb.PublishAsync(ctx, TopicArticlePublished, &ArticlePublishedEvent{
		ArticleID: uint64(article.ID),
		Title:     article.Title,
		Slug:      article.Slug,
		AuthorID:  uint64(article.AuthorID),
	})

	uc.log.WithContext(ctx).Infof("[PublishArticle] 发布成功 id=%d title=%q", id, article.Title)
	return article, nil
}

// =============================================================================
// ArchiveArticle — 归档文章
//
// 流程步骤：
//  1. 权限校验
//  2. 如果已经处于归档状态，直接返回（幂等）
//  3. 记录当前是否为已发布（用于后续计数调整）
//  4. 状态变更为 Archived
//  5. 持久化
//  6. 如果之前是已发布状态，分类和标签计数 -1
//
// 归档后文章不再展示在默认列表，但仍可通过 ID/Slug 访问。
// =============================================================================

func (uc *ArticleUseCase) ArchiveArticle(ctx context.Context, id uint) (*Article, error) {
	uc.log.WithContext(ctx).Debugf("[ArchiveArticle] 开始 article_id=%d", id)

	// 步骤1: 权限校验
	article, err := uc.authorizeArticle(ctx, id)
	if err != nil {
		uc.log.WithContext(ctx).Warnf("[ArchiveArticle] 权限校验失败 article_id=%d err=%v", id, err)
		return nil, err
	}
	uc.log.WithContext(ctx).Debugf("[ArchiveArticle] 步骤1权限通过 article_id=%d status=%d", id, article.Status)

	// 步骤2: 幂等检查 — 已归档则直接返回
	if article.Status == ArticleStatusArchived {
		uc.log.WithContext(ctx).Debugf("[ArchiveArticle] 步骤2: 已是归档状态，幂等返回 article_id=%d", id)
		return article, nil
	}

	// 步骤3: 记录归档前状态（用于判断是否需要调整计数）
	wasPublished := article.Status == ArticleStatusPublished
	uc.log.WithContext(ctx).Debugf("[ArchiveArticle] 步骤3: 状态变更 %d→%d was_published=%v", article.Status, ArticleStatusArchived, wasPublished)

	// 步骤4+5: 变更状态并持久化
	article.Status = ArticleStatusArchived
	if err := uc.repo.Update(ctx, article); err != nil {
		uc.log.WithContext(ctx).Errorf("[ArchiveArticle] 持久化失败 article_id=%d err=%v", id, err)
		return nil, fmt.Errorf("archive article: %w", err)
	}

	// 步骤6: 如果之前是已发布，调整计数
	if wasPublished {
		uc.log.WithContext(ctx).Debugf("[ArchiveArticle] 步骤6: 文章原为发布状态, 调整关联计数 category_id=%v tag_count=%d",
			article.CategoryID, len(article.Tags))
		if article.CategoryID != nil {
			if err := uc.catRepo.IncrementArticleCount(ctx, *article.CategoryID, -1); err != nil {
				uc.log.WithContext(ctx).Warnf("[ArchiveArticle] 分类计数调整失败 category_id=%d err=%v",
					*article.CategoryID, err)
			}
		}
		if len(article.Tags) > 0 {
			if err := uc.repo.UpdateTagsArticleCount(ctx, tagIDs(article.Tags), -1); err != nil {
				uc.log.WithContext(ctx).Warnf("[ArchiveArticle] 标签计数调整失败 tag_count=%d err=%v",
					len(article.Tags), err)
			}
		}
	} else {
		uc.log.WithContext(ctx).Debugf("[ArchiveArticle] 步骤6跳过: 非发布状态, 无需调整计数")
	}

	uc.log.WithContext(ctx).Infof("[ArchiveArticle] 归档成功 id=%d", id)
	return article, nil
}

// =============================================================================
// GetArticle — 获取文章（按 ID 数字或 Slug 字符串）
//
// identifier 可能是纯数字（数据库 ID）或 slug。
// 先尝试按 ID 查找，失败则按 Slug 查找。
// 查询成功后填充当前用户的点赞状态。
// =============================================================================

func (uc *ArticleUseCase) GetArticle(ctx context.Context, identifier string) (*Article, error) {
	uc.log.WithContext(ctx).Debugf("[GetArticle] 开始 identifier=%q", identifier)

	// ===================================================================
	// 尝试 parseUint: 若 identifier 是纯数字 → 按 ID 查询
	// parseUint 返回 0 表示 identifier 不是纯数字 → 跳过 ID 查询
	// ===================================================================
	if id := parseUint(identifier); id > 0 {
		uc.log.WithContext(ctx).Debugf("[GetArticle] 尝试按ID查询 id=%d", id)
		article, err := uc.repo.FindByID(ctx, id)
		if err == nil {
			uc.log.WithContext(ctx).Debugf("[GetArticle] ID查询命中 id=%d title=%q status=%d", id, article.Title, article.Status)
			// 填充当前用户是否已点赞（未认证用户跳过）
			uc.fillIsLiked(ctx, article)
			return article, nil
		}
		uc.log.WithContext(ctx).Debugf("[GetArticle] ID查询未命中 id=%d, 尝试Slug查询", id)
	} else {
		uc.log.WithContext(ctx).Debugf("[GetArticle] identifier非纯数字, 直接尝试Slug查询")
	}

	// ===================================================================
	// ID 查询失败 → 按 Slug 查询
	// ===================================================================
	uc.log.WithContext(ctx).Debugf("[GetArticle] 尝试按Slug查询 slug=%q", identifier)
	article, err := uc.repo.FindBySlug(ctx, identifier)
	if err != nil {
		uc.log.WithContext(ctx).Debugf("[GetArticle] Slug也未匹配 identifier=%q", identifier)
		return nil, ErrArticleNotFound
	}

	uc.log.WithContext(ctx).Debugf("[GetArticle] Slug查询命中 slug=%q id=%d title=%q", identifier, article.ID, article.Title)
	uc.fillIsLiked(ctx, article)
	return article, nil
}

// =============================================================================
// ListArticles — 文章列表（分页 + 过滤 + 排序）
//
// 默认查询已发布文章。所有过滤、排序、分页参数通过 ArticleListQuery 传递。
// 委托给 data 层 List 方法执行具体的 SQL 查询。
// =============================================================================

func (uc *ArticleUseCase) ListArticles(ctx context.Context, query ArticleListQuery) ([]*Article, int64, error) {
	uc.log.WithContext(ctx).Debugf("[ListArticles] 开始 status=%q page=%d page_size=%d sort_by=%s sort_order=%s category_id=%v author_id=%v tags=%v",
		query.Status, query.Page, query.PageSize, query.SortBy, query.SortOrder, query.CategoryID, query.AuthorID, query.Tags)

	// 如果未指定状态，默认只返回已发布文章
	if query.Status == "" {
		uc.log.WithContext(ctx).Debugf("[ListArticles] 未指定status, 默认查询已发布文章")
		query.Status = "published"
	}

	articles, total, err := uc.repo.List(ctx, query)
	if err != nil {
		uc.log.WithContext(ctx).Errorf("[ListArticles] 查询失败 err=%v", err)
		return nil, 0, err
	}

	uc.log.WithContext(ctx).Debugf("[ListArticles] 查询完成 total=%d returned=%d", total, len(articles))
	return articles, total, nil
}

// =============================================================================
// SearchArticles — 全文搜索（预留接口，后续接入搜索引擎）
//
// 当前返回空结果，不作为功能入口。
// 后续接入 Elasticsearch / Meilisearch 后在此处实现对搜索结果的处理。
// =============================================================================

func (uc *ArticleUseCase) SearchArticles(ctx context.Context, keyword string, page, pageSize int) ([]*Article, int64, error) {
	uc.log.WithContext(ctx).Debugf("[SearchArticles] 预留接口 keyword=%q page=%d page_size=%d", keyword, page, pageSize)
	return nil, 0, nil
}

// =============================================================================
// LikeArticle — 点赞（幂等）
//
// 流程：
//  1. 从 context 提取当前用户 ID
//  2. 调用 data 层 InsertLike
//     data 层在事务中执行:
//       INSERT ... ON CONFLICT DO NOTHING (幂等)
//       若 RowsAffected > 0 → UPDATE articles SET like_count = like_count + 1
//  3. 异步发布 article.liked 事件
// =============================================================================

func (uc *ArticleUseCase) LikeArticle(ctx context.Context, articleID uint) error {
	uc.log.WithContext(ctx).Debugf("[LikeArticle] 开始 article_id=%d", articleID)

	// 步骤1: 提取用户ID
	userID, err := getCurrentUserID(ctx)
	if err != nil {
		uc.log.WithContext(ctx).Warnf("[LikeArticle] 未认证 article_id=%d", articleID)
		return ErrUserNotAuthenticated
	}
	uc.log.WithContext(ctx).Debugf("[LikeArticle] 用户ID提取成功 user_id=%d", userID)

	// 步骤2: 执行点赞（data 层保证幂等）

	uc.log.WithContext(ctx).Debugf("[LikeArticle] 调用 repo.InsertLike article_id=%d user_id=%d", articleID, userID)
	if err := uc.repo.InsertLike(ctx, articleID, uint(userID)); err != nil {
		uc.log.WithContext(ctx).Errorf("[LikeArticle] 点赞失败 article_id=%d user_id=%d err=%v", articleID, userID, err)
		return fmt.Errorf("like article: %w", err)
	}
	uc.log.WithContext(ctx).Debugf("[LikeArticle] repo.InsertLike 完成")

	// 步骤3: 异步发布事件
	_ = uc.eb.PublishAsync(ctx, TopicArticleLiked, &ArticleLikedEvent{
		ArticleID: uint64(articleID),
		UserID:    userID,
	})

	uc.log.WithContext(ctx).Infof("[LikeArticle] 点赞成功 article_id=%d user_id=%d", articleID, userID)
	return nil
}

// =============================================================================
// UnlikeArticle — 取消点赞（幂等）
//
// 流程：
//  1. 从 context 提取当前用户 ID
//  2. 调用 data 层 DeleteLike
//     data 层在事务中执行:
//       DELETE FROM articles_likes WHERE ... AND deleted_at IS NULL
//       若 RowsAffected > 0 → UPDATE articles SET like_count = GREATEST(like_count - 1, 0)
// =============================================================================

func (uc *ArticleUseCase) UnlikeArticle(ctx context.Context, articleID uint) error {
	uc.log.WithContext(ctx).Debugf("[UnlikeArticle] 开始 article_id=%d", articleID)

	// 步骤1: 提取用户ID
	userID, err := getCurrentUserID(ctx)
	if err != nil {
		uc.log.WithContext(ctx).Warnf("[UnlikeArticle] 未认证 article_id=%d", articleID)
		return ErrUserNotAuthenticated
	}
	uc.log.WithContext(ctx).Debugf("[UnlikeArticle] 用户ID提取成功 user_id=%d", userID)

	// 步骤2: 执行取消点赞
	uc.log.WithContext(ctx).Debugf("[UnlikeArticle] 调用 repo.DeleteLike article_id=%d user_id=%d", articleID, userID)
	if err := uc.repo.DeleteLike(ctx, articleID, uint(userID)); err != nil {
		uc.log.WithContext(ctx).Errorf("[UnlikeArticle] 取消点赞失败 article_id=%d user_id=%d err=%v", articleID, userID, err)
		return fmt.Errorf("unlike article: %w", err)
	}

	uc.log.WithContext(ctx).Infof("[UnlikeArticle] 取消点赞成功 article_id=%d user_id=%d", articleID, userID)
	return nil
}

// =============================================================================
// IncrementView — 浏览计数原子 +1
//
// 通过 data 层执行原子 SQL: UPDATE articles SET view_count = view_count + 1 WHERE id = ?
// 同时异步发布 article.viewed 事件（Analytics 服务消费 → 统计仪表盘更新）。
// =============================================================================

func (uc *ArticleUseCase) IncrementView(ctx context.Context, articleID uint) error {
	uc.log.WithContext(ctx).Debugf("[IncrementView] article_id=%d", articleID)

	// 原子递增浏览计数（data 层使用 gorm.Expr("view_count + ?") 避免并发竞争）
	if err := uc.repo.IncrementViewCount(ctx, articleID, 1); err != nil {
		uc.log.WithContext(ctx).Errorf("[IncrementView] 计数更新失败 article_id=%d err=%v", articleID, err)
		return fmt.Errorf("increment view: %w", err)
	}
	uc.log.WithContext(ctx).Debugf("[IncrementView] 计数更新成功")

	// 异步发布浏览事件
	_ = uc.eb.PublishAsync(ctx, TopicArticleViewed, &ArticleViewedEvent{
		ArticleID: uint64(articleID),
	})

	uc.log.WithContext(ctx).Debugf("[IncrementView] 完成 article_id=%d", articleID)
	return nil
}

// =============================================================================
// authorizeArticle — 权限校验（内部辅助方法）
//
// 同时完成"文章存在性检查"和"作者身份验证"两个操作。
// 先查文章（不存在 → 404），再比作者 ID（不匹配 → 403）。
// 返回完整的 Article 对象供调用方继续操作（避免重复查询）。
// =============================================================================

func (uc *ArticleUseCase) authorizeArticle(ctx context.Context, id uint) (*Article, error) {
	uc.log.WithContext(ctx).Debugf("[authorizeArticle] article_id=%d", id)

	// 提取当前用户 ID
	userID, err := getCurrentUserID(ctx)
	if err != nil {
		uc.log.WithContext(ctx).Debugf("[authorizeArticle] 未认证 article_id=%d", id)
		return nil, ErrUserNotAuthenticated
	}
	uc.log.WithContext(ctx).Debugf("[authorizeArticle] 当前用户ID=%d", userID)

	// 查询文章
	article, err := uc.repo.FindByID(ctx, id)
	if err != nil {
		uc.log.WithContext(ctx).Debugf("[authorizeArticle] 文章不存在 article_id=%d", id)
		return nil, ErrArticleNotFound
	}
	uc.log.WithContext(ctx).Debugf("[authorizeArticle] 文章找到 author_id=%d status=%d", article.AuthorID, article.Status)

	// 验证作者身份
	if article.AuthorID != uint(userID) {
		uc.log.WithContext(ctx).Warnf("[authorizeArticle] 权限拒绝 article_id=%d author=%d requester=%d",
			id, article.AuthorID, userID)
		return nil, ErrNotArticleOwner
	}

	uc.log.WithContext(ctx).Debugf("[authorizeArticle] 权限验证通过 article_id=%d", id)
	return article, nil
}

// =============================================================================
// fillIsLiked — 填充当前用户对文章的点赞状态
//
// 仅在用户已认证时执行（未认证用户跳过，IsLiked 保持默认 false）。
// 查询失败不影响主流程（静默返回，IsLiked 保持 false）。
// =============================================================================

func (uc *ArticleUseCase) fillIsLiked(ctx context.Context, article *Article) {
	if article == nil {
		return
	}
	uc.log.WithContext(ctx).Debugf("[fillIsLiked] article_id=%d", article.ID)

	// 检查用户是否已认证
	userID, err := getCurrentUserID(ctx)
	if err != nil {
		uc.log.WithContext(ctx).Debugf("[fillIsLiked] 用户未认证, 跳过 article_id=%d", article.ID)
		return // 未认证用户，IsLiked 保持 false
	}

	// 查询点赞记录
	liked, err := uc.repo.IsLiked(ctx, article.ID, uint(userID))
	if err != nil {
		uc.log.WithContext(ctx).Warnf("[fillIsLiked] 查询点赞状态失败 article_id=%d user_id=%d err=%v", article.ID, userID, err)
		return // 查询失败不阻塞，IsLiked 保持 false
	}

	article.IsLiked = liked
	uc.log.WithContext(ctx).Debugf("[fillIsLiked] 完成 article_id=%d is_liked=%v", article.ID, liked)
}

// =============================================================================
// validateArticleInput — 校验文章标题和内容
// =============================================================================

func validateArticleInput(title, content string) error {
	if err := validateTitle(title); err != nil {
		return err
	}
	if len(content) == 0 {
		return ErrArticleContentEmpty
	}
	if len(content) > MaxContentLength {
		return ErrArticleContentTooBig
	}
	return nil
}

// validateTitle — 校验标题长度 (2-200字符)
func validateTitle(title string) error {
	if len(title) < MinTitleLength || len(title) > MaxTitleLength {
		return ErrArticleTitleInvalid
	}
	return nil
}

// =============================================================================
// generateSlug — 从标题生成 URL slug
//
// 转换规则：
//   1. 小写字母和数字 → 保留
//   2. 大写字母 → 转小写
//   3. 中文字符 (U+4E00 ~ U+9FFF) → 查表转拼音首字母
//   4. 其他字符 → 替换为连字符 '-'
//   5. 清理：合并重复连字符，去除首尾连字符
//   6. 空 slug → 返回 "article"
//   7. 超长 → 截断为 200 字符
// =============================================================================

func generateSlug(title string) string {
	var b strings.Builder
	b.Grow(len(title)) // 预分配内存，减少扩容

	// 逐字符转换
	for _, r := range title {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			// 小写字母和数字直接保留
			b.WriteRune(r)
		case r >= 'A' && r <= 'Z':
			// 大写字母转小写
			b.WriteRune(unicode.ToLower(r))
		case r >= 0x4e00 && r <= 0x9fff:
			// 中文字符：查表获取拼音首字母（如"博"→"b"）
			if p := firstPinyin(r); p != "" {
				b.WriteString(p)
			}
		default:
			// 其他所有字符（空格、标点、emoji等）→ 连字符
			b.WriteByte('-')
		}
	}

	// 清理处理后的字符串
	s := b.String()

	// 合并连续连字符: "foo---bar" → "foo-bar"
	for strings.Contains(s, "--") {
		s = strings.ReplaceAll(s, "--", "-")
	}

	// 去除首尾连字符: "-hello-world-" → "hello-world"
	s = strings.Trim(s, "-")

	// 空 slug 后备（纯中文标题无拼音匹配时）
	if s == "" {
		return "article"
	}

	// 截断超长 slug（数据库 VARCHAR(200)）
	if len(s) > 200 {
		s = s[:200]
	}

	return s
}

// =============================================================================
// ensureUniqueSlug — 确保 slug 在数据库中唯一
//
// 策略：从原始 slug 开始，逐次追加 "-2", "-3"...，最多重试 100 次。
// 通过 FindBySlug 检查占用：若查询返回错误（记录不存在）→ slug 可用。
// 100 次后仍未找到 → 返回错误。
//
// 注意：这是一个"检查-再操作"模式，在高并发下存在竞态窗口。
// 最终的兜底是数据库唯一约束（articles.slug UNIQUE WHERE deleted_at IS NULL）。
// =============================================================================

func ensureUniqueSlug(ctx context.Context, slug string, repo ArticleRepo) (string, error) {
	for i := 0; i < 100; i++ {
		candidate := slug
		if i > 0 {
			candidate = fmt.Sprintf("%s-%d", slug, i+1) // slug-2, slug-3, ...
		}
		// FindBySlug 返回 error 表示 slug 不存在 → 可用
		if _, err := repo.FindBySlug(ctx, candidate); err != nil {
			return candidate, nil
		}
	}
	return "", errors.New("slug 重试次数耗尽（100次）")
}

// =============================================================================
// resolveTags — 解析标签名称列表
//
// 对每个名称执行：
//   1. 去除首尾空白字符
//   2. 跳过空字符串
//   3. 调用 TagRepo.FindOrCreate（名称→slug 转换后查找/创建）
//
// FindOrCreate 实现细节（在 data 层）：
//   1. FindByName → 存在则直接返回
//   2. 不存在 → INSERT ... ON CONFLICT (name) DO NOTHING（防并发重复）
//   3. 再次 FindByName 获取完整的带 ID 记录
// =============================================================================

func resolveTags(ctx context.Context, names []string, tagRepo TagRepo) ([]*Tag, error) {
	// 预分配切片容量，避免多次扩容
	tags := make([]*Tag, 0, len(names))

	for _, name := range names {
		// 去除首尾空白
		name = strings.TrimSpace(name)
		if name == "" {
			continue // 跳过空名称
		}

		// 查找或创建标签
		tag, err := tagRepo.FindOrCreate(ctx, name, tagNameToSlug(name))
		if err != nil {
			return nil, fmt.Errorf("resolve tag %q: %w", name, err)
		}
		tags = append(tags, tag)
	}

	return tags, nil
}

// tagIDs — 从 Tag 切片提取 ID 切片
func tagIDs(tags []*Tag) []uint {
	ids := make([]uint, len(tags))
	for i, t := range tags {
		ids[i] = t.ID
	}
	return ids
}

// tagNameToSlug — 标签名称转 slug（小写 + 空格转连字符）
// 例如 "Go语言" → "go语言"
func tagNameToSlug(name string) string {
	return strings.ToLower(strings.ReplaceAll(strings.TrimSpace(name), " ", "-"))
}

// =============================================================================
// truncateContent — 从正文截取摘要
//
// 按 rune（Unicode 码点）计数而非按 byte，正确处理中文等多字节字符。
// 超出 maxLen 时追加 "..." 省略号。
// =============================================================================

func truncateContent(content string, maxLen int) string {
	content = strings.TrimSpace(content)

	runes := []rune(content)
	if len(runes) <= maxLen {
		return content // 内容较短，全文作为摘要
	}

	// 截取前 maxLen 个字符，追加省略号
	return string(runes[:maxLen]) + "..."
}

// =============================================================================
// 拼音首字母映射表（简体中文常用字）
//
// pinyinBlk 是汉字字符序列，pinyinTbl 是对应的拼音首字母序列。
// 两个字符串通过下标一一对应：pinyinBlk[i] 的拼音首字母 = pinyinTbl[i]。
// firstPinyin 使用 strings.IndexRune 进行 O(n) 线性查找。
// =============================================================================

var pinyinBlk = "的一是不了在人有我他这个们中来上大为和国地到以说时要就出会可也你对生能而子那得于着下自之年过发后作里用道行所然家种事成分现经动工学如地方从部同定比关高本性看又法意力员实长等"
var pinyinTbl = "ddsblzrywztgmmzlsdwhgddyssyjcckhkddssnneznydzxxgzggffhlzyydgxrsjzcbxggfyycsdsndxmb"

// firstPinyin 查找中文字符的拼音首字母
// 若 r 不在汉字表中 → 返回空字符串
// 若索引越界（数据不一致）→ 返回空字符串
func firstPinyin(r rune) string {
	idx := strings.IndexRune(pinyinBlk, r)
	if idx < 0 || idx >= len(pinyinTbl) {
		return "" // 未在映射表中，跳过
	}
	return string(pinyinTbl[idx])
}

// =============================================================================
// Context 工具函数
// =============================================================================

// normalizeCategoryID 规范化分类 ID
// proto 中 uint64 category_id = 0 表示"无分类"，转为 nil
func normalizeCategoryID(categoryID *uint) *uint {
	if categoryID == nil || *categoryID == 0 {
		return nil
	}
	return categoryID
}

// getCurrentUserID 从 pkg/meta 获取当前登录用户 ID。
//
// 用户认证信息由 Gateway JWT 中间件注入到 Kratos metadata（x-md-global- 全局透传）。
// GetRequestMetaData 优先从服务端上下文解析，失败则回退到客户端上下文。
func getCurrentUserID(ctx context.Context) (uint64, error) {
	reqMeta := meta.GetRequestMetaData(ctx)
	if reqMeta != nil && reqMeta.Auth.UserID > 0 {
		return reqMeta.Auth.UserID, nil
	}
	return 0, ErrUserNotAuthenticated
}

// parseUint 尝试将字符串解析为无符号整数
//
// 用于 GetArticle 判断 identifier 是数字 ID 还是 slug 字符串。
// 对超过 64 位范围的数字返回 0（按 slug 查询），避免截断误判。
func parseUint(s string) uint {
	if s == "" {
		return 0
	}
	// 逐字符检查是否为纯数字，同时检测是否超过 uint64 安全范围
	// uint64 最长 20 位（math.MaxUint64 = 18446744073709551615）
	if len(s) > 20 {
		return 0
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return 0
		}
	}
	var v uint
	fmt.Sscanf(s, "%d", &v)
	if v == 0 && s != "0" {
		return 0 // Sscanf 失败（溢出）
	}
	return v
}

// =============================================================================
// 事件定义
// =============================================================================

// 事件主题常量 — 由 ArticleUseCase 发布，其他服务订阅消费
const (
	TopicArticlePublished = "article.published" // 文章发布事件
	TopicArticleUpdated   = "article.updated"   // 文章更新事件
	TopicArticleDeleted   = "article.deleted"   // 文章删除事件
	TopicArticleViewed    = "article.viewed"    // 文章浏览事件
	TopicArticleLiked     = "article.liked"     // 文章点赞事件
	TopicCommentCreated   = "comment.created"   // 评论创建事件
	TopicCommentApproved  = "comment.approved"  // 评论审核通过事件
)

// ArticleUpdatedEvent 文章更新事件
type ArticleUpdatedEvent struct {
	ArticleID uint64 `json:"article_id"`
	Title     string `json:"title"`
	AuthorID  uint64 `json:"author_id"`
}

// ArticleDeletedEvent 文章删除事件
type ArticleDeletedEvent struct {
	ArticleID uint64 `json:"article_id"`
	AuthorID  uint64 `json:"author_id"`
}

// ArticlePublishedEvent 文章发布事件
// 消费者: Notification 服务 → 发送新文章邮件给订阅者
type ArticlePublishedEvent struct {
	ArticleID uint64 `json:"article_id"`
	Title     string `json:"title"`
	Slug      string `json:"slug"`
	AuthorID  uint64 `json:"author_id"`
}

// ArticleViewedEvent 文章浏览事件
// 消费者: Analytics 服务 → PV/UV 统计
type ArticleViewedEvent struct {
	ArticleID uint64 `json:"article_id"`
}

// ArticleLikedEvent 文章点赞事件
// 消费者: 预留
type ArticleLikedEvent struct {
	ArticleID uint64 `json:"article_id"`
	UserID    uint64 `json:"user_id"`
}

// CommentCreatedEvent 评论创建事件
// 消费者: Notification 服务 → 通知文章作者"有新评论"、通知被回复者"有回复"
type CommentCreatedEvent struct {
	CommentID uint64  `json:"comment_id"`
	ArticleID uint64  `json:"article_id"`
	AuthorID  uint64  `json:"author_id"`
	ParentID  *uint64 `json:"parent_id,omitempty"` // 非 nil 表示回复
}

// CommentApprovedEvent 评论审核通过事件
// 消费者: Notification 服务 → 通知评论作者"评论已通过"
type CommentApprovedEvent struct {
	CommentID uint64 `json:"comment_id"`
	ArticleID uint64 `json:"article_id"`
	AuthorID  uint64 `json:"author_id"`
}
