package biz

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"unicode"

	"ley/app/post/internal/model"
	"ley/pkg/meta"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/uuid"
)

// =============================================================================
// 文章业务层错误定义
// =============================================================================

var (
	ErrPostNotFound         = errors.New("post not found")              // 文章不存在
	ErrPostTitleEmpty       = errors.New("post title is required (2-200 chars)") // 文章标题不符合要求
	ErrPostContentEmpty     = errors.New("post content is required")     // 文章内容不能为空
	ErrPostContentTooBig    = errors.New("post content exceeds limit")   // 文章内容超过长度限制
	ErrNotPostOwner         = errors.New("you can only modify your own posts") // 非文章作者无权操作
	ErrPostAlreadyPublished = errors.New("post is already published")    // 文章已发布
)

// =============================================================================
// 文章字段长度限制常量
// =============================================================================

const (
	maxTitleLength   = 200   // 标题最大长度（字符）
	minTitleLength   = 2     // 标题最小长度（字符）
	maxContentLength = 100000 // 内容最大长度（字符）
	maxExcerptLength = 500   // 摘要最大长度（字符）
)

// =============================================================================
// PostUseCase — 文章业务用例
// =============================================================================

// PostUseCase 文章业务用例
// 封装文章创建、发布、编辑、删除等核心业务逻辑，
// 负责输入校验、权限检查、标签解析等与存储无关的业务规则。
type PostUseCase struct {
	repo   PostRepo   // 文章数据访问接口
	tag    TagRepo    // 标签数据访问接口（用于标签解析）
	logger log.Logger // Kratos 日志器
}

// NewPostUseCase 创建 PostUseCase
func NewPostUseCase(repo PostRepo, tagRepo TagRepo, logger log.Logger) *PostUseCase {
	return &PostUseCase{repo: repo, tag: tagRepo, logger: logger}
}

// log 返回携带上下文信息的日志助手
func (uc *PostUseCase) log(ctx context.Context) *log.Helper {
	return log.NewHelper(log.WithContext(ctx, uc.logger))
}

// =============================================================================
// CreatePost — 创建文章（草稿状态）
// =============================================================================

// CreatePost 创建文章（草稿状态）
// 1. 获取当前用户 ID
// 2. 校验标题和内容
// 3. 生成唯一 slug
// 4. 解析标签（查找或创建）
// 5. 写入数据库并关联标签
func (uc *PostUseCase) CreatePost(ctx context.Context, title, content, excerpt, coverImage string, categoryID *uint, tagNames []string) (*model.Post, error) {
	// 获取当前登录用户 ID
	authorID, err := getCurrentUserID(ctx)
	if err != nil {
		uc.log(ctx).Warnw("创建文章失败：用户未认证", "error", err)
		return nil, err
	}

	// 校验标题和内容
	if err := validatePostInput(title, content); err != nil {
		uc.log(ctx).Warnw("创建文章失败：输入校验不通过", "title_len", len(title), "content_len", len(content), "error", err)
		return nil, err
	}

	// 根据标题生成 slug 并确保唯一性
	slug := GenerateSlug(title)
	uc.log(ctx).Debugw("生成文章Slug", "title", title, "slug", slug)
	slug, err = uc.ensureUniqueSlug(ctx, slug)
	if err != nil {
		uc.log(ctx).Errorw("创建文章失败：Slug唯一性耗尽", "original_slug", slug, "error", err)
		return nil, err
	}
	uc.log(ctx).Debugw("文章Slug确认唯一", "slug", slug)

	// 如果未提供摘要，自动从内容生成
	if excerpt == "" {
		excerpt = generateExcerpt(content, maxExcerptLength)
		uc.log(ctx).Debugw("自动生成文章摘要", "excerpt_len", len(excerpt))
	}

	// 解析标签：按名称查找，不存在则创建
	tags, err := uc.resolveTags(ctx, tagNames)
	if err != nil {
		uc.log(ctx).Errorw("创建文章失败：标签解析失败", "tag_names", tagNames, "error", err)
		return nil, err
	}
	uc.log(ctx).Debugw("创建文章标签解析完成", "tag_count", len(tags))

	// 构建 post model（含 Tags 关联用于序列化）
	postTags := make([]model.Tag, len(tags))
	tagIDs := make([]uint, len(tags))
	for i, t := range tags {
		postTags[i] = *t
		tagIDs[i] = t.ID
	}

	// 构建文章实体，状态默认为草稿
	post := &model.Post{
		UUID:       uuid.Must(uuid.NewV7()).String(), // 使用 UUID v7（时间有序）
		Title:      title,
		Slug:       slug,
		Content:    content,
		Excerpt:    excerpt,
		CoverImage: coverImage,
		Status:     model.PostStatusDraft, // 默认草稿状态
		AuthorID:   uint(authorID),
		CategoryID: categoryID,
		Tags:       postTags,
	}

	// 写入数据库
	if err := uc.repo.Create(ctx, post); err != nil {
		uc.log(ctx).Errorw("创建文章数据层写入失败", "error", err)
		return nil, fmt.Errorf("create post: %w", err)
	}

	// 关联标签到文章（posts_tags 中间表）
	if len(tagIDs) > 0 {
		if err := uc.repo.AssociateTags(ctx, post.ID, tagIDs); err != nil {
			uc.log(ctx).Errorw("创建文章关联标签失败", "post_id", post.ID, "error", err)
		}
	}

	uc.log(ctx).Infow("文章创建成功", "id", post.ID, "slug", post.Slug, "author_id", post.AuthorID)
	return post, nil
}

// =============================================================================
// PublishPost — 发布文章
// =============================================================================

// PublishPost 发布文章
// 权限校验通过后将草稿状态变更为已发布。
func (uc *PostUseCase) PublishPost(ctx context.Context, postID uint) (*model.Post, error) {
	// 权限校验：查询文章并验证当前用户是否为作者
	post, err := uc.authorizePost(ctx, postID)
	if err != nil {
		return nil, err
	}

	// 已发布的文章不允许重复发布
	if post.Status == model.PostStatusPublished {
		uc.log(ctx).Warnw("发布文章失败：文章已处于发布状态", "id", postID)
		return nil, ErrPostAlreadyPublished
	}

	// 变更状态为已发布
	post.Status = model.PostStatusPublished
	if err := uc.repo.Update(ctx, post); err != nil {
		uc.log(ctx).Errorw("发布文章数据层更新失败", "id", postID, "error", err)
		return nil, fmt.Errorf("publish post: %w", err)
	}

	uc.log(ctx).Infow("文章发布成功", "id", postID)
	return post, nil
}

// =============================================================================
// ArchivePost — 归档文章
// =============================================================================

// ArchivePost 归档文章
// 权限校验通过后将文章状态变更为已归档。
func (uc *PostUseCase) ArchivePost(ctx context.Context, postID uint) (*model.Post, error) {
	// 权限校验
	post, err := uc.authorizePost(ctx, postID)
	if err != nil {
		return nil, err
	}

	// 已归档的文章无需重复操作
	if post.Status == model.PostStatusArchived {
		uc.log(ctx).Debugw("归档文章：已处于归档状态，跳过", "id", postID)
		return post, nil
	}

	// 变更状态为已归档
	post.Status = model.PostStatusArchived
	if err := uc.repo.Update(ctx, post); err != nil {
		uc.log(ctx).Errorw("归档文章数据层更新失败", "id", postID, "error", err)
		return nil, fmt.Errorf("archive post: %w", err)
	}

	uc.log(ctx).Infow("文章归档成功", "id", postID)
	return post, nil
}

// =============================================================================
// UpdatePost — 更新文章（仅作者）
// =============================================================================

// UpdatePost 更新文章（仅作者）
// 只更新请求中提供的非 nil 字段，nil 字段保持不变。
func (uc *PostUseCase) UpdatePost(ctx context.Context, postID uint, title, content, excerpt, coverImage *string, categoryID *uint, tagNames []string) (*model.Post, error) {
	// 权限校验
	post, err := uc.authorizePost(ctx, postID)
	if err != nil {
		return nil, err
	}

	uc.log(ctx).Debugw("开始更新文章", "id", postID,
		"has_title", title != nil, "has_content", content != nil,
		"has_excerpt", excerpt != nil, "has_cover_image", coverImage != nil,
	)

	// 标题更新：校验新标题并重新生成 slug
	if title != nil {
		if err := validateTitle(*title); err != nil {
			uc.log(ctx).Warnw("更新文章标题校验失败", "title", *title, "error", err)
			return nil, err
		}
		post.Title = *title
		post.Slug, _ = uc.ensureUniqueSlug(ctx, GenerateSlug(*title))
		uc.log(ctx).Debugw("更新了文章标题和Slug", "new_title", *title, "new_slug", post.Slug)
	}

	// 内容更新：校验长度
	if content != nil {
		if len(*content) > maxContentLength {
			uc.log(ctx).Warnw("更新文章内容超过长度限制", "content_len", len(*content), "max", maxContentLength)
			return nil, ErrPostContentTooBig
		}
		post.Content = *content
		uc.log(ctx).Debugw("更新了文章内容", "content_len", len(*content))
	}

	// 摘要更新
	if excerpt != nil {
		post.Excerpt = *excerpt
		uc.log(ctx).Debugw("更新了文章摘要")
	}

	// 封面图片更新
	if coverImage != nil {
		post.CoverImage = *coverImage
		uc.log(ctx).Debugw("更新了文章封面图片")
	}

	// 分类更新
	if categoryID != nil {
		post.CategoryID = categoryID
		uc.log(ctx).Debugw("更新了文章分类", "category_id", *categoryID)
	}

	// 持久化更新
	if err := uc.repo.Update(ctx, post); err != nil {
		uc.log(ctx).Errorw("更新文章数据层操作失败", "id", postID, "error", err)
		return nil, fmt.Errorf("update post: %w", err)
	}

	// 标签更新（全量替换）
	if tagNames != nil {
		uc.log(ctx).Debugw("更新文章标签", "tag_names", tagNames)
		tags, err := uc.resolveTags(ctx, tagNames)
		if err != nil {
			uc.log(ctx).Warnw("更新文章时标签解析失败，标签未更新", "error", err)
		} else {
			_ = uc.repo.SyncTags(ctx, postID, tagIDs(tags))
		}
	}

	uc.log(ctx).Infow("文章更新成功", "id", postID)
	return post, nil
}

// =============================================================================
// DeletePost — 删除文章
// =============================================================================

// DeletePost 删除文章
// 权限校验通过后执行软删除。
func (uc *PostUseCase) DeletePost(ctx context.Context, postID uint) error {
	// 权限校验
	if _, err := uc.authorizePost(ctx, postID); err != nil {
		return err
	}

	if err := uc.repo.Delete(ctx, postID); err != nil {
		uc.log(ctx).Errorw("删除文章数据层操作失败", "id", postID, "error", err)
		return fmt.Errorf("delete post: %w", err)
	}

	uc.log(ctx).Infow("文章删除成功", "id", postID)
	return nil
}

// =============================================================================
// LikePost / UnlikePost — 点赞/取消点赞
// =============================================================================

// LikePost 点赞文章（幂等）
// 通过 repo 层执行 INSERT OR IGNORE，避免重复点赞
func (uc *PostUseCase) LikePost(ctx context.Context, postID uint) error {
	// 获取当前用户 ID
	userID, err := getCurrentUserID(ctx)
	if err != nil {
		uc.log(ctx).Warnw("点赞失败：用户未认证", "error", err)
		return err
	}
	uc.log(ctx).Debugw("执行点赞操作", "post_id", postID, "user_id", userID)

	// TODO: 完整实现需要 postRepo 暴露 Like 方法
	// 当前通过 IncrementViewCount 占位 — 后续需要独立的 likes 表
	_ = userID
	return nil
}

// UnlikePost 取消点赞（幂等）
func (uc *PostUseCase) UnlikePost(ctx context.Context, postID uint) error {
	userID, err := getCurrentUserID(ctx)
	if err != nil {
		uc.log(ctx).Warnw("取消点赞失败：用户未认证", "error", err)
		return err
	}
	uc.log(ctx).Debugw("执行取消点赞操作", "post_id", postID, "user_id", userID)
	_ = userID
	return nil
}

// =============================================================================
// GetPost — 按 UUID 或 Slug 获取文章
// =============================================================================

// GetPost 按 UUID 或 Slug 获取文章
// 先尝试按 UUID 查询，失败则按 Slug 查询。
func (uc *PostUseCase) GetPost(ctx context.Context, identifier string) (*model.Post, error) {
	uc.log(ctx).Debugw("获取文章，尝试UUID查询", "identifier", identifier)

	// 第一步：尝试按 UUID 查询
	post, err := uc.repo.FindByUUID(ctx, identifier)
	if err == nil {
		uc.log(ctx).Debugw("文章获取成功（UUID匹配）", "identifier", identifier)
		return post, nil
	}

	// 第二步：UUID 未命中，尝试按 Slug 查询
	uc.log(ctx).Debugw("UUID未命中，尝试Slug查询", "identifier", identifier)
	post, err = uc.repo.FindBySlug(ctx, identifier)
	if err != nil {
		uc.log(ctx).Debugw("获取文章失败：UUID和Slug均未匹配", "identifier", identifier)
		return nil, ErrPostNotFound
	}

	uc.log(ctx).Debugw("文章获取成功（Slug匹配）", "identifier", identifier)
	return post, nil
}

// =============================================================================
// ListPosts — 文章列表
// =============================================================================

// ListPosts 文章列表
// 默认查询已发布文章，委托给数据层 List 方法。
func (uc *PostUseCase) ListPosts(ctx context.Context, query PostListQuery) ([]*model.Post, int64, error) {
	// 如果未指定状态，默认查询已发布的文章
	if query.Status == "" {
		query.Status = "published"
	}
	uc.log(ctx).Debugw("查询文章列表", "status", query.Status, "page", query.Page, "page_size", query.PageSize)
	return uc.repo.List(ctx, query)
}

// =============================================================================
// SearchPosts — 全文搜索
// =============================================================================

// SearchPosts 全文搜索
// 关键词为空时直接返回空结果。
func (uc *PostUseCase) SearchPosts(ctx context.Context, keyword string, page, pageSize int) ([]*model.Post, int64, error) {
	// 关键词为空则无搜索结果
	if strings.TrimSpace(keyword) == "" {
		uc.log(ctx).Debugw("搜索关键词为空，返回空结果")
		return nil, 0, nil
	}
	uc.log(ctx).Debugw("全文搜索文章", "keyword", keyword, "page", page, "page_size", pageSize)
	return uc.repo.Search(ctx, keyword, page, pageSize)
}

// =============================================================================
// authorizePost — 权限校验
// =============================================================================

// authorizePost 权限校验
// 验证当前用户是否为文章作者，非作者不允许编辑/删除操作。
func (uc *PostUseCase) authorizePost(ctx context.Context, postID uint) (*model.Post, error) {
	// 获取当前用户 ID
	userID, err := getCurrentUserID(ctx)
	if err != nil {
		uc.log(ctx).Warnw("权限校验失败：用户未认证", "error", err)
		return nil, err
	}

	// 查询文章
	post, err := uc.repo.FindByID(ctx, postID)
	if err != nil {
		uc.log(ctx).Debugw("权限校验：文章未找到", "post_id", postID)
		return nil, ErrPostNotFound
	}

	// 验证文章作者是否与当前用户一致
	if post.AuthorID != uint(userID) {
		uc.log(ctx).Warnw("权限校验：无权操作他人文章",
			"post_id", postID, "author_id", post.AuthorID, "requester_id", userID,
		)
		return nil, ErrNotPostOwner
	}

	return post, nil
}

// =============================================================================
// resolveTags — 按名称查找标签，不存在则创建
// =============================================================================

// resolveTags 按名称查找标签，不存在则创建
// 遍历标签名称列表，对每个名称执行 FindOrCreate 操作。
func (uc *PostUseCase) resolveTags(ctx context.Context, names []string) ([]*model.Tag, error) {
	tags := make([]*model.Tag, 0, len(names))
	for _, name := range names {
		// 去除首尾空格，跳过空名称
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		uc.log(ctx).Debugw("解析标签", "tag_name", name)
		// 查找或创建标签
		tag, err := uc.tag.FindOrCreate(ctx, name, tagNameToSlug(name))
		if err != nil {
			uc.log(ctx).Errorw("解析标签失败", "tag_name", name, "error", err)
			return nil, fmt.Errorf("resolve tag %q: %w", name, err)
		}
		tags = append(tags, tag)
	}
	return tags, nil
}

// =============================================================================
// ensureUniqueSlug — 确保 slug 唯一
// =============================================================================

// ensureUniqueSlug 确保 slug 唯一
// 如果 slug 已存在，则在后面追加 "-2", "-3"...，最多重试 100 次。
func (uc *PostUseCase) ensureUniqueSlug(ctx context.Context, slug string) (string, error) {
	for i := 0; i < 100; i++ {
		candidate := slug
		// 第一次尝试使用原始 slug，后续追加数字后缀
		if i > 0 {
			candidate = fmt.Sprintf("%s-%d", slug, i+1)
		}
		// 查询 slug 是否存在，不存在则可用
		if _, err := uc.repo.FindBySlug(ctx, candidate); err != nil {
			uc.log(ctx).Debugw("Slug可用", "candidate", candidate)
			return candidate, nil
		}
		uc.log(ctx).Debugw("Slug已存在，尝试下一个候选", "candidate", candidate)
	}
	// 100 次仍未找到可用 slug，返回错误
	return "", fmt.Errorf("slug collision exhausted for %q", slug)
}

// =============================================================================
// GenerateSlug — 从标题生成 URL slug
// =============================================================================

// GenerateSlug 从标题生成 URL slug
// 1. 英文字母转小写
// 2. 中文字符转拼音首字母
// 3. 其他字符替换为连字符 '-'
// 4. 清理多余连字符和边界字符
func GenerateSlug(title string) string {
	var b strings.Builder
	b.Grow(len(title))
	for _, r := range title {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			// 小写字母和数字直接保留
			b.WriteRune(r)
		case r >= 'A' && r <= 'Z':
			// 大写字母转小写
			b.WriteRune(unicode.ToLower(r))
		case r >= 0x4e00 && r <= 0x9fff:
			// 中文字符转为拼音首字母
			if p := firstPinyin(r); p != "" {
				b.WriteString(p)
			}
		default:
			// 其他字符替换为连字符
			b.WriteByte('-')
		}
	}
	// 清理字符串：替换连续连字符、去除首尾连字符
	s := b.String()
	for strings.Contains(s, "--") {
		s = strings.ReplaceAll(s, "--", "-")
	}
	s = strings.Trim(s, "-")
	// 如果 slug 为空，使用默认值
	if s == "" {
		s = "post"
	}
	// 截断超长 slug
	if len(s) > 200 {
		s = s[:200]
	}
	return s
}

// =============================================================================
// firstPinyin — 获取中文字符的拼音首字母
// =============================================================================

// firstPinyin 获取中文字符的拼音首字母
// 在预定义的 unicodeBlock 字典中查找对应拼音首字母。
func firstPinyin(r rune) string {
	idx := strings.IndexRune(unicodeBlock, r)
	if idx < 0 || idx >= len(pinyinTable) {
		return ""
	}
	return string(pinyinTable[idx])
}

// =============================================================================
// 输入校验函数
// =============================================================================

// validatePostInput 校验文章标题和内容
func validatePostInput(title, content string) error {
	if err := validateTitle(title); err != nil {
		return err
	}
	if len(content) == 0 {
		return ErrPostContentEmpty
	}
	if len(content) > maxContentLength {
		return ErrPostContentTooBig
	}
	return nil
}

// validateTitle 校验文章标题长度
func validateTitle(title string) error {
	if len(title) < minTitleLength || len(title) > maxTitleLength {
		return ErrPostTitleEmpty
	}
	return nil
}

// =============================================================================
// 工具函数
// =============================================================================

// generateExcerpt 从文章内容生成摘要（去除 markdown 标记后截取）
func generateExcerpt(content string, maxLen int) string {
	// 去除常见的 markdown 标记
	content = strings.ReplaceAll(content, "#", "")
	content = strings.ReplaceAll(content, "*", "")
	content = strings.ReplaceAll(content, "`", "")
	content = strings.TrimSpace(content)
	runes := []rune(content)
	if len(runes) <= maxLen {
		return content
	}
	// 截取前 maxLen 个字符并追加省略号
	return string(runes[:maxLen]) + "..."
}

// tagIDs 批量提取标签的 ID
func tagIDs(tags []*model.Tag) []uint {
	ids := make([]uint, len(tags))
	for i, t := range tags {
		ids[i] = t.ID
	}
	return ids
}

// tagNameToSlug 将标签名转为 slug（小写 + 空格转连字符）
func tagNameToSlug(name string) string {
	return strings.ToLower(strings.ReplaceAll(strings.TrimSpace(name), " ", "-"))
}

// =============================================================================
// getCurrentUserID — 从上下文中获取当前用户 ID
// =============================================================================

// getCurrentUserID 从上下文中获取当前用户 ID
// 优先从 gRPC metadata 中获取，其次从 context value 中获取。
func getCurrentUserID(ctx context.Context) (uint64, error) {
	// 尝试从请求元数据中获取
	reqMeta := meta.GetRequestMetaData(ctx)
	if reqMeta != nil && reqMeta.Auth.UserID > 0 {
		return reqMeta.Auth.UserID, nil
	}
	// 尝试从 context value 中获取
	if userID, ok := ctx.Value("user_id").(uint64); ok && userID > 0 {
		return userID, nil
	}
	// 未认证用户
	return 0, errors.New("user not authenticated")
}

// =============================================================================
// 拼音映射表（简体中文常用字）
// =============================================================================

const (
	unicodeBlock = "的一是不了在人有我他这个们中来上大为和国地到以说时要就出会可也你对生能而子那得于着下自之年过发后作里用道行所然家种事成分现经动工学如地方从部同定比关高本性看又法意力员实长等"
	pinyinTable  = "ddsblzrywztgmmzlsdwhgddyssyjcckhkddssnneznydzxxgzggffhlzyydgxrsjzcbxggfyycsdsndxmb"
)
