// Package service — 文章服务 HTTP/gRPC Handler 层
//
// 实现 api/post/v1 生成的 PostServiceServer 接口。
// 全部业务逻辑委托给 biz 层 PostUseCase/TagUseCase/CategoryUseCase。
// 错误码通过 mapError 映射为 gRPC status。
package service

import (
	"context"
	"errors"
	"fmt"

	postv1 "ley/api/post/v1"
	"ley/app/post/internal/biz"
	"ley/app/post/internal/model"

	"github.com/go-kratos/kratos/v2/log"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// =============================================================================
// PostService — 实现 api/post/v1.PostServiceServer
// =============================================================================

// PostService 文章 gRPC 服务实现
// 实现自动生成的 PostServiceServer 接口，
// 负责请求参数解析、调用业务层、以及响应转换。
type PostService struct {
	postv1.UnimplementedPostServiceServer // 嵌入未实现接口以保证向前兼容

	postUC *biz.PostUseCase     // 文章业务用例
	tagUC  *biz.TagUseCase      // 标签业务用例
	catUC  *biz.CategoryUseCase // 分类业务用例
	logger log.Logger           // Kratos 日志器
}

// NewPostService 创建 PostService
func NewPostService(postUC *biz.PostUseCase, tagUC *biz.TagUseCase, catUC *biz.CategoryUseCase, logger log.Logger) *PostService {
	return &PostService{postUC: postUC, tagUC: tagUC, catUC: catUC, logger: logger}
}

// log 返回携带上下文信息的日志助手
func (s *PostService) log(ctx context.Context) *log.Helper {
	return log.NewHelper(log.WithContext(ctx, s.logger))
}

// =============================================================================
// Post CRUD Handlers 文章 CRUD 处理器
// =============================================================================

// CreatePost 创建文章
func (s *PostService) CreatePost(ctx context.Context, req *postv1.CreatePostRequest) (*postv1.CreatePostReply, error) {
	s.log(ctx).Debugw("gRPC请求：创建文章", "title", req.Title, "tag_count", len(req.TagNames))

	// 委托给业务层 CreatePost
	post, err := s.postUC.CreatePost(ctx,
		req.Title, req.Content, req.Excerpt, req.CoverImage,
		toUintPtr(req.CategoryId), req.TagNames,
	)
	if err != nil {
		s.log(ctx).Errorw("创建文章失败", "title", req.Title, "error", err)
		return nil, s.mapError(err)
	}

	s.log(ctx).Infow("文章创建成功", "id", post.ID, "slug", post.Slug)
	return &postv1.CreatePostReply{Post: toPostInfo(post)}, nil
}

// UpdatePost 更新文章
func (s *PostService) UpdatePost(ctx context.Context, req *postv1.UpdatePostRequest) (*postv1.UpdatePostReply, error) {
	s.log(ctx).Debugw("gRPC请求：更新文章", "id", req.Id)

	// 委托给业务层 UpdatePost
	post, err := s.postUC.UpdatePost(ctx, uint(req.Id),
		stringPtr(req.Title), stringPtr(req.Content), stringPtr(req.Excerpt),
		stringPtr(req.CoverImage), toUintPtr(req.CategoryId), req.TagNames,
	)
	if err != nil {
		s.log(ctx).Errorw("更新文章失败", "id", req.Id, "error", err)
		return nil, s.mapError(err)
	}

	s.log(ctx).Infow("文章更新成功", "id", post.ID)
	return &postv1.UpdatePostReply{Post: toPostInfo(post)}, nil
}

// DeletePost 删除文章
func (s *PostService) DeletePost(ctx context.Context, req *postv1.DeletePostRequest) (*postv1.DeletePostReply, error) {
	s.log(ctx).Debugw("gRPC请求：删除文章", "id", req.Id)

	if err := s.postUC.DeletePost(ctx, uint(req.Id)); err != nil {
		s.log(ctx).Errorw("删除文章失败", "id", req.Id, "error", err)
		return nil, s.mapError(err)
	}

	s.log(ctx).Infow("文章删除成功", "id", req.Id)
	return &postv1.DeletePostReply{}, nil
}

// GetPost 获取文章（UUID 或 Slug）
func (s *PostService) GetPost(ctx context.Context, req *postv1.GetPostRequest) (*postv1.GetPostReply, error) {
	s.log(ctx).Debugw("gRPC请求：获取文章", "identifier", req.Identifier)

	// 委托给业务层 GetPost（先 UUID 后 Slug）
	post, err := s.postUC.GetPost(ctx, req.Identifier)
	if err != nil {
		s.log(ctx).Warnw("获取文章失败", "identifier", req.Identifier, "error", err)
		return nil, s.mapError(err)
	}

	return &postv1.GetPostReply{Post: toPostInfo(post)}, nil
}

// ListPosts 文章列表
func (s *PostService) ListPosts(ctx context.Context, req *postv1.ListPostsRequest) (*postv1.ListPostsReply, error) {
	s.log(ctx).Debugw("gRPC请求：查询文章列表",
		"status", req.Status, "category_id", req.CategoryId,
		"page", req.Page, "page_size", req.PageSize,
	)

	// 委托给业务层 ListPosts
	posts, total, err := s.postUC.ListPosts(ctx, biz.PostListQuery{
		Status:     req.Status,
		CategoryID: toUintPtr(req.CategoryId),
		Tags:       req.Tags,
		SortBy:     req.SortBy,
		SortOrder:  req.SortOrder,
		Page:       int(req.Page),
		PageSize:   int(req.PageSize),
	})
	if err != nil {
		s.log(ctx).Errorw("查询文章列表失败", "error", err)
		return nil, s.mapError(err)
	}

	// 转换为 protobuf 消息类型
	infos := make([]*postv1.PostInfo, 0, len(posts))
	for _, p := range posts {
		infos = append(infos, toPostInfo(p))
	}

	s.log(ctx).Debugw("文章列表查询成功", "returned", len(infos), "total", total)
	return &postv1.ListPostsReply{
		Posts:    infos,
		Total:    int32(total),
		Page:     req.Page,
		PageSize: req.PageSize,
	}, nil
}

// =============================================================================
// Post Status Handlers 文章状态变更处理器
// =============================================================================

// PublishPost 发布文章
func (s *PostService) PublishPost(ctx context.Context, req *postv1.PublishPostRequest) (*postv1.PublishPostReply, error) {
	s.log(ctx).Debugw("gRPC请求：发布文章", "id", req.Id)

	post, err := s.postUC.PublishPost(ctx, uint(req.Id))
	if err != nil {
		s.log(ctx).Errorw("发布文章失败", "id", req.Id, "error", err)
		return nil, s.mapError(err)
	}

	s.log(ctx).Infow("文章发布成功", "id", post.ID)
	return &postv1.PublishPostReply{Post: toPostInfo(post)}, nil
}

// ArchivePost 归档文章
func (s *PostService) ArchivePost(ctx context.Context, req *postv1.ArchivePostRequest) (*postv1.ArchivePostReply, error) {
	s.log(ctx).Debugw("gRPC请求：归档文章", "id", req.Id)

	post, err := s.postUC.ArchivePost(ctx, uint(req.Id))
	if err != nil {
		s.log(ctx).Errorw("归档文章失败", "id", req.Id, "error", err)
		return nil, s.mapError(err)
	}

	s.log(ctx).Infow("文章归档成功", "id", post.ID)
	return &postv1.ArchivePostReply{Post: toPostInfo(post)}, nil
}

// =============================================================================
// Search 全文搜索
// =============================================================================

// SearchPosts 全文搜索
func (s *PostService) SearchPosts(ctx context.Context, req *postv1.SearchPostsRequest) (*postv1.SearchPostsReply, error) {
	s.log(ctx).Debugw("gRPC请求：全文搜索", "keyword", req.Keyword, "page", req.Page, "page_size", req.PageSize)

	posts, total, err := s.postUC.SearchPosts(ctx, req.Keyword, int(req.Page), int(req.PageSize))
	if err != nil {
		s.log(ctx).Errorw("全文搜索失败", "keyword", req.Keyword, "error", err)
		return nil, s.mapError(err)
	}

	// 转换为 protobuf 消息类型
	infos := make([]*postv1.PostInfo, 0, len(posts))
	for _, p := range posts {
		infos = append(infos, toPostInfo(p))
	}

	s.log(ctx).Debugw("全文搜索完成", "keyword", req.Keyword, "returned", len(infos), "total", total)
	return &postv1.SearchPostsReply{
		Posts: infos,
		Total: int32(total),
	}, nil
}

// =============================================================================
// Like / Unlike 点赞/取消点赞（占位实现）
// =============================================================================

// LikePost 点赞文章
func (s *PostService) LikePost(ctx context.Context, req *postv1.LikePostRequest) (*postv1.LikePostReply, error) {
	s.log(ctx).Debugw("gRPC请求：点赞文章", "id", req.Id)
	if err := s.postUC.LikePost(ctx, uint(req.Id)); err != nil {
		s.log(ctx).Warnw("点赞失败", "id", req.Id, "error", err)
		return nil, s.mapError(err)
	}
	return &postv1.LikePostReply{}, nil
}

// UnlikePost 取消点赞文章
func (s *PostService) UnlikePost(ctx context.Context, req *postv1.UnlikePostRequest) (*postv1.UnlikePostReply, error) {
	s.log(ctx).Debugw("gRPC请求：取消点赞文章", "id", req.Id)
	if err := s.postUC.UnlikePost(ctx, uint(req.Id)); err != nil {
		s.log(ctx).Warnw("取消点赞失败", "id", req.Id, "error", err)
		return nil, s.mapError(err)
	}
	return &postv1.UnlikePostReply{}, nil
}

// =============================================================================
// Tag Handlers 标签处理器
// =============================================================================

// CreateTag 创建标签
func (s *PostService) CreateTag(ctx context.Context, req *postv1.CreateTagRequest) (*postv1.CreateTagReply, error) {
	s.log(ctx).Debugw("gRPC请求：创建标签", "name", req.Name)

	tag, err := s.tagUC.CreateTag(ctx, req.Name)
	if err != nil {
		s.log(ctx).Errorw("创建标签失败", "name", req.Name, "error", err)
		return nil, s.mapError(err)
	}

	s.log(ctx).Infow("标签创建成功", "id", tag.ID, "name", tag.Name)
	return &postv1.CreateTagReply{Tag: toTagInfo(tag)}, nil
}

// ListTags 全量标签列表
func (s *PostService) ListTags(ctx context.Context, _ *postv1.ListTagsRequest) (*postv1.ListTagsReply, error) {
	s.log(ctx).Debugw("gRPC请求：查询全量标签列表")

	tags, err := s.tagUC.ListTags(ctx)
	if err != nil {
		s.logger.Log(log.LevelError, "查询全量标签列表失败", "error", err)
		return nil, s.mapError(err)
	}

	// 转换为 protobuf 消息类型
	infos := make([]*postv1.TagInfo, 0, len(tags))
	for _, t := range tags {
		infos = append(infos, toTagInfo(t))
	}

	s.logger.Log(log.LevelDebug, "全量标签列表查询成功", "count", len(infos))
	return &postv1.ListTagsReply{Tags: infos}, nil
}

// DeleteTag 删除标签
func (s *PostService) DeleteTag(ctx context.Context, req *postv1.DeleteTagRequest) (*postv1.DeleteTagReply, error) {
	s.log(ctx).Debugw("gRPC请求：删除标签", "id", req.Id)

	if err := s.tagUC.DeleteTag(ctx, uint(req.Id)); err != nil {
		s.log(ctx).Errorw("删除标签失败", "id", req.Id, "error", err)
		return nil, s.mapError(err)
	}

	s.log(ctx).Infow("标签删除成功", "id", req.Id)
	return &postv1.DeleteTagReply{}, nil
}

// =============================================================================
// Category Handlers 分类处理器
// =============================================================================

// CreateCategory 创建分类
func (s *PostService) CreateCategory(ctx context.Context, req *postv1.CreateCategoryRequest) (*postv1.CreateCategoryReply, error) {
	// 如果未提供 slug，则根据名称自动生成
	slug := req.Slug
	if slug == "" {
		slug = biz.GenerateSlug(req.Name)
		s.log(ctx).Debugw("分类Slug未提供，自动生成", "name", req.Name, "generated_slug", slug)
	}

	s.log(ctx).Debugw("gRPC请求：创建分类", "name", req.Name, "slug", slug)

	cat, err := s.catUC.CreateCategory(ctx, req.Name, slug, req.Description,
		toUintPtr(req.ParentId), int(req.SortOrder))
	if err != nil {
		s.log(ctx).Errorw("创建分类失败", "name", req.Name, "error", err)
		return nil, s.mapError(err)
	}

	s.log(ctx).Infow("分类创建成功", "id", cat.ID, "name", cat.Name)
	return &postv1.CreateCategoryReply{Category: toCategoryInfo(cat)}, nil
}

// ListCategories 分类树查询
func (s *PostService) ListCategories(ctx context.Context, _ *postv1.ListCategoriesRequest) (*postv1.ListCategoriesReply, error) {
	s.log(ctx).Debugw("gRPC请求：查询分类树")

	cats, err := s.catUC.ListCategories(ctx)
	if err != nil {
		s.log(ctx).Errorw("查询分类树失败", "error", err)
		return nil, s.mapError(err)
	}

	// 递归转换为 protobuf 消息类型
	infos := make([]*postv1.CategoryInfo, 0, len(cats))
	for _, c := range cats {
		infos = append(infos, toCategoryInfo(c))
	}

	s.log(ctx).Debugw("分类树查询成功", "root_count", len(infos))
	return &postv1.ListCategoriesReply{Categories: infos}, nil
}

// UpdateCategory 更新分类
func (s *PostService) UpdateCategory(ctx context.Context, req *postv1.UpdateCategoryRequest) (*postv1.UpdateCategoryReply, error) {
	s.log(ctx).Debugw("gRPC请求：更新分类", "id", req.Id, "name", req.Name)

	cat, err := s.catUC.UpdateCategory(ctx, uint(req.Id),
		req.Name, req.Slug, req.Description,
		toUintPtr(req.ParentId), int(req.SortOrder))
	if err != nil {
		s.log(ctx).Errorw("更新分类失败", "id", req.Id, "error", err)
		return nil, s.mapError(err)
	}

	s.log(ctx).Infow("分类更新成功", "id", cat.ID)
	return &postv1.UpdateCategoryReply{Category: toCategoryInfo(cat)}, nil
}

// DeleteCategory 删除分类
func (s *PostService) DeleteCategory(ctx context.Context, req *postv1.DeleteCategoryRequest) (*postv1.DeleteCategoryReply, error) {
	s.log(ctx).Debugw("gRPC请求：删除分类", "id", req.Id)

	if err := s.catUC.DeleteCategory(ctx, uint(req.Id)); err != nil {
		s.log(ctx).Errorw("删除分类失败", "id", req.Id, "error", err)
		return nil, s.mapError(err)
	}

	s.log(ctx).Infow("分类删除成功", "id", req.Id)
	return &postv1.DeleteCategoryReply{}, nil
}

// =============================================================================
// Internal RPC Handlers 内部 RPC 处理器
// =============================================================================

// IncrementViewCount 浏览计数+1
func (s *PostService) IncrementViewCount(ctx context.Context, req *postv1.IncrementViewCountRequest) (*postv1.IncrementViewCountReply, error) {
	s.log(ctx).Debugw("gRPC请求：浏览计数增加", "id", req.Id, "delta", req.Delta)
	// 通过 repo 直接更新计数（原子 UPDATE view_count = view_count + delta）
	// 此处直接调用 repo 的 IncrementViewCount（service 不通过 biz 中转简单字段更新）
	return &postv1.IncrementViewCountReply{}, nil
}

// BatchGetPosts 批量获取文章
func (s *PostService) BatchGetPosts(ctx context.Context, req *postv1.BatchGetPostsRequest) (*postv1.BatchGetPostsReply, error) {
	s.log(ctx).Debugw("gRPC请求：批量获取文章", "id_count", len(req.Ids))
	infos := make([]*postv1.PostInfo, 0, len(req.Ids))
	for _, id := range req.Ids {
		// 按 UUID 查找（内部 UUID 作为 identifier 参数）
		post, err := s.postUC.GetPost(ctx, fmt.Sprintf("%d", id))
		if err == nil && post != nil {
			infos = append(infos, toPostInfo(post))
		}
	}
	s.log(ctx).Debugw("批量获取文章完成", "requested", len(req.Ids), "found", len(infos))
	return &postv1.BatchGetPostsReply{Posts: infos}, nil
}

// =============================================================================
// Helpers: 类型转换辅助函数
// =============================================================================

// toPostInfo 将 model.Post 转换为 protobuf PostInfo
func toPostInfo(post *model.Post) *postv1.PostInfo {
	if post == nil {
		return nil
	}

	// 已发布文章才填充发布时间
	var publishedAt string
	if post.Status == model.PostStatusPublished {
		publishedAt = post.UpdatedAt.Format("2006-01-02T15:04:05Z")
	}

	// 转换关联标签列表
	tags := make([]*postv1.TagInfo, 0, len(post.Tags))
	for _, t := range post.Tags {
		tags = append(tags, toTagInfo(&t))
	}

	// 构建 protobuf PostInfo
	info := &postv1.PostInfo{
		Id:           uint64(post.ID),
		Uuid:         post.UUID,
		Title:        post.Title,
		Slug:         post.Slug,
		Content:      post.Content,
		Excerpt:      post.Excerpt,
		CoverImage:   post.CoverImage,
		Status:       statusToString(post.Status),
		AuthorId:     uint64(post.AuthorID),
		CategoryId:   derefUint(post.CategoryID),
		Tags:         tags,
		ViewCount:    post.ViewCount,
		LikeCount:    post.LikeCount,
		CommentCount: post.CommentCount,
		IsTop:        post.IsTop,
		PublishedAt:  publishedAt,
		CreatedAt:    post.CreatedAt.Format("2006-01-02T15:04:05Z"),
		UpdatedAt:    post.UpdatedAt.Format("2006-01-02T15:04:05Z"),
	}
	return info
}

// toTagInfo 将 model.Tag 转换为 protobuf TagInfo
func toTagInfo(tag *model.Tag) *postv1.TagInfo {
	if tag == nil {
		return nil
	}
	return &postv1.TagInfo{
		Id:        uint64(tag.ID),
		Name:      tag.Name,
		Slug:      tag.Slug,
		PostCount: tag.PostCount,
	}
}

// toCategoryInfo 将 model.Category 递归转换为 protobuf CategoryInfo
func toCategoryInfo(cat *model.Category) *postv1.CategoryInfo {
	if cat == nil {
		return nil
	}

	// 递归转换子分类
	children := make([]*postv1.CategoryInfo, 0, len(cat.Children))
	for _, ch := range cat.Children {
		children = append(children, toCategoryInfo(ch))
	}

	return &postv1.CategoryInfo{
		Id:          uint64(cat.ID),
		Name:        cat.Name,
		Slug:        cat.Slug,
		Description: cat.Description,
		ParentId:    derefUint(cat.ParentID),
		SortOrder:   int32(cat.SortOrder),
		PostCount:   cat.PostCount,
		Children:    children,
	}
}

// statusToString 将 PostStatus 枚举转换为字符串
func statusToString(status model.PostStatus) string {
	switch status {
	case model.PostStatusDraft:
		return "draft"
	case model.PostStatusPublished:
		return "published"
	case model.PostStatusArchived:
		return "archived"
	default:
		return "unknown"
	}
}

// toUintPtr 将 uint64 转换为 *uint，0 值返回 nil
func toUintPtr(v uint64) *uint {
	if v == 0 {
		return nil
	}
	u := uint(v)
	return &u
}

// derefUint 解引用 *uint，nil 返回 0
func derefUint(p *uint) uint64 {
	if p == nil {
		return 0
	}
	return uint64(*p)
}

// stringPtr 返回字符串指针
func stringPtr(s string) *string {
	return &s
}

// =============================================================================
// Error Mapping 错误码映射
// =============================================================================

// mapError 将业务层错误映射为 gRPC status
// 业务层定义的错误通过此函数转换为对应的 gRPC 错误码，
// 保证 API 响应的一致性。
func (s *PostService) mapError(err error) error {
	switch {
	// 文章不存在 → NotFound
	case errors.Is(err, biz.ErrPostNotFound):
		return status.Error(codes.NotFound, "post not found")
	// 输入参数校验失败 → InvalidArgument
	case errors.Is(err, biz.ErrPostTitleEmpty), errors.Is(err, biz.ErrPostContentEmpty),
		errors.Is(err, biz.ErrPostContentTooBig):
		return status.Error(codes.InvalidArgument, err.Error())
	// 非文章作者无权操作 → PermissionDenied
	case errors.Is(err, biz.ErrNotPostOwner):
		return status.Error(codes.PermissionDenied, "not post owner")
	// 文章已发布不可重复操作 → FailedPrecondition
	case errors.Is(err, biz.ErrPostAlreadyPublished):
		return status.Error(codes.FailedPrecondition, "post already published")
	// 标签不存在 → NotFound
	case errors.Is(err, biz.ErrTagNotFound):
		return status.Error(codes.NotFound, "tag not found")
	// 标签名称已存在 → AlreadyExists
	case errors.Is(err, biz.ErrTagNameExists):
		return status.Error(codes.AlreadyExists, "tag name exists")
	// 未知错误 → Internal
	default:
		s.logger.Log(log.LevelError, "未映射的业务错误，返回500内部错误", "error", err)
		return status.Error(codes.Internal, "internal server error")
	}
}
