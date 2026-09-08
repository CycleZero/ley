package service

import (
	"context"
	"time"

	blogv1 "github.com/CycleZero/ley/api/blog/v1"
	commonv1 "github.com/CycleZero/ley/api/common/v1"
	"github.com/CycleZero/ley/app/blog/internal/biz"
	"github.com/CycleZero/ley/pkg/meta"
	"github.com/CycleZero/ley/pkg/security"

	"github.com/go-kratos/kratos/v2/log"
)

// ArticleService 文章服务传输层：proto ↔ biz DTO 转换，仅在入口打 Debug 日志。
type ArticleService struct {
	blogv1.UnimplementedArticleServiceServer
	uc  *biz.ArticleUseCase
	log *log.Helper
}

// NewArticleService 构造文章服务（依赖由 Wire 注入）。
func NewArticleService(uc *biz.ArticleUseCase, logger log.Logger) *ArticleService {
	return &ArticleService{uc: uc, log: log.NewHelper(logger)}
}

// CreateArticle 创建文章；status=published 时复用发布流程立即发布。
func (s *ArticleService) CreateArticle(ctx context.Context, req *blogv1.CreateArticleRequest) (*blogv1.CreateArticleReply, error) {
	s.log.WithContext(ctx).Debug("收到创建文章请求")
	a, err := s.uc.CreateArticle(ctx, req.Title, req.Content, req.Excerpt, req.CoverImage, toUintPtr(req.CategoryId), req.TagNames)
	if err != nil {
		return nil, err
	}
	// B-106: status=published 时创建后立即发布（复用 PublishArticle 的完整发布流程）
	if req.Status == "published" {
		published, err := s.uc.PublishArticle(ctx, a.ID)
		if err != nil {
			return nil, err
		}
		a = published
	}
	return &blogv1.CreateArticleReply{Article: toArticleInfo(a)}, nil
}

// UpdateArticle 更新文章（仅作者本人或管理员，归属校验在 biz）。
func (s *ArticleService) UpdateArticle(ctx context.Context, req *blogv1.UpdateArticleRequest) (*blogv1.UpdateArticleReply, error) {
	s.log.WithContext(ctx).Debug("收到更新文章请求")
	a, err := s.uc.UpdateArticle(ctx, uint(req.Id), strPtr(req.Title), strPtr(req.Content), strPtr(req.Excerpt), strPtr(req.CoverImage), toUintPtr(req.CategoryId), req.TagNames)
	if err != nil {
		return nil, err
	}
	return &blogv1.UpdateArticleReply{Article: toArticleInfo(a)}, nil
}

// DeleteArticle 删除文章（仅作者本人或管理员）。
func (s *ArticleService) DeleteArticle(ctx context.Context, req *blogv1.DeleteArticleRequest) (*blogv1.DeleteArticleReply, error) {
	s.log.WithContext(ctx).Debug("收到删除文章请求")
	return &blogv1.DeleteArticleReply{}, s.uc.DeleteArticle(ctx, uint(req.Id))
}

// GetArticle 获取文章详情（identifier 支持 ID 或 slug；可见性由 biz 判定）。
func (s *ArticleService) GetArticle(ctx context.Context, req *blogv1.GetArticleRequest) (*blogv1.GetArticleReply, error) {
	a, err := s.uc.GetArticle(ctx, req.Identifier)
	if err != nil {
		return nil, err
	}
	return &blogv1.GetArticleReply{Article: toArticleInfo(a)}, nil
}

// ListArticles 分页查询文章列表（按状态/分类/标签/作者过滤，支持排序）。
func (s *ArticleService) ListArticles(ctx context.Context, req *blogv1.ListArticlesRequest) (*blogv1.ListArticlesReply, error) {
	articles, total, err := s.uc.ListArticles(ctx, biz.ArticleListQuery{
		Status: req.Status, CategoryID: toUintPtr(req.CategoryId), Tags: req.Tags,
		AuthorID: toUintPtr(req.AuthorId), SortBy: req.SortBy, SortOrder: req.SortOrder,
		Page: int(req.Page), PageSize: int(req.PageSize),
	})
	if err != nil {
		return nil, err
	}
	infos := make([]*blogv1.ArticleInfo, 0, len(articles))
	for _, a := range articles {
		infos = append(infos, toArticleInfo(a))
	}
	return &blogv1.ListArticlesReply{Articles: infos, Total: total, Page: req.Page, PageSize: req.PageSize}, nil
}

// PublishArticle 发布文章（草稿→已发布，仅作者本人或管理员）。
func (s *ArticleService) PublishArticle(ctx context.Context, req *blogv1.PublishArticleRequest) (*blogv1.PublishArticleReply, error) {
	s.log.WithContext(ctx).Debug("收到发布文章请求")
	a, err := s.uc.PublishArticle(ctx, uint(req.Id))
	if err != nil {
		return nil, err
	}
	return &blogv1.PublishArticleReply{Article: toArticleInfo(a)}, nil
}

// ArchiveArticle 归档文章（已发布→已归档，仅作者本人或管理员）。
func (s *ArticleService) ArchiveArticle(ctx context.Context, req *blogv1.ArchiveArticleRequest) (*blogv1.ArchiveArticleReply, error) {
	s.log.WithContext(ctx).Debug("收到归档文章请求")
	a, err := s.uc.ArchiveArticle(ctx, uint(req.Id))
	if err != nil {
		return nil, err
	}
	return &blogv1.ArchiveArticleReply{Article: toArticleInfo(a)}, nil
}

// SearchArticles 关键字搜索文章（标题/正文，分页）。
func (s *ArticleService) SearchArticles(ctx context.Context, req *blogv1.SearchArticlesRequest) (*blogv1.SearchArticlesReply, error) {
	articles, total, err := s.uc.SearchArticles(ctx, req.Keyword, int(req.Page), int(req.PageSize))
	if err != nil {
		return nil, err
	}
	infos := make([]*blogv1.ArticleInfo, 0, len(articles))
	for _, a := range articles {
		infos = append(infos, toArticleInfo(a))
	}
	return &blogv1.SearchArticlesReply{Articles: infos, Total: total}, nil
}

// LikeArticle 点赞文章（需登录，重复点赞由 biz 幂等处理）。
func (s *ArticleService) LikeArticle(ctx context.Context, req *blogv1.LikeArticleRequest) (*blogv1.LikeArticleReply, error) {
	s.log.WithContext(ctx).Debug("收到点赞请求")
	return &blogv1.LikeArticleReply{}, s.uc.LikeArticle(ctx, uint(req.Id))
}

// UnlikeArticle 取消点赞文章（需登录）。
func (s *ArticleService) UnlikeArticle(ctx context.Context, req *blogv1.UnlikeArticleRequest) (*blogv1.UnlikeArticleReply, error) {
	s.log.WithContext(ctx).Debug("收到取消点赞请求")
	return &blogv1.UnlikeArticleReply{}, s.uc.UnlikeArticle(ctx, uint(req.Id))
}

// ViewArticle 记录文章浏览（按真实客户端 IP 去重，返回本次是否计数）。
func (s *ArticleService) ViewArticle(ctx context.Context, req *blogv1.ViewArticleRequest) (*blogv1.ViewArticleReply, error) {
	// B-206: 优先取网关透传的 x-md-global-auth-real-ip；缺失时回退 transport 层
	// X-Forwarded-For 等头（网关已追加 XFF），避免去重键坍缩为全站共享。
	clientIP := meta.GetRequestMetaData(ctx).RealClientIp
	if clientIP == "" {
		clientIP = security.GetRealIp(ctx)
	}
	counted, err := s.uc.ViewArticle(ctx, uint(req.Id), clientIP)
	if err != nil {
		return nil, err
	}
	return &blogv1.ViewArticleReply{Counted: counted}, nil
}

// toArticleInfo 将 biz 文章对象转为 proto ArticleInfo（含作者/分类/标签，时间为 UTC RFC3339）。
func toArticleInfo(a *biz.Article) *blogv1.ArticleInfo {
	if a == nil {
		return nil
	}
	info := &blogv1.ArticleInfo{
		Id: uint64(a.ID), Title: a.Title, Slug: a.Slug, Content: a.Content, Excerpt: a.Excerpt, CoverImage: a.CoverImage,
		Status: articleStatusStr(a.Status), AuthorId: uint64(a.AuthorID),
		Author:     &commonv1.AuthorInfo{Id: uint64(a.AuthorID), Username: a.AuthorName, Avatar: a.AuthorAvatar},
		CategoryId: derefUint(a.CategoryID), ViewCount: a.ViewCount, LikeCount: a.LikeCount,
		IsTop: a.IsTop, IsLiked: a.IsLiked, CreatedAt: a.CreatedAt.UTC().Format(time.RFC3339), UpdatedAt: a.UpdatedAt.UTC().Format(time.RFC3339),
	}
	if a.CategoryID != nil {
		info.Category = &blogv1.CategoryInfo{Id: uint64(*a.CategoryID), Name: a.CategoryName}
	}
	if a.PublishedAt != nil {
		info.PublishedAt = a.PublishedAt.UTC().Format(time.RFC3339)
	}
	if len(a.Tags) > 0 {
		info.Tags = make([]*blogv1.TagInfo, 0, len(a.Tags))
		for _, t := range a.Tags {
			info.Tags = append(info.Tags, toTagInfo(t))
		}
	}
	return info
}

// articleStatusStr 将 biz 文章状态转为对外字符串（draft/published/archived）。
func articleStatusStr(s biz.ArticleStatus) string {
	switch s {
	case biz.ArticleStatusPublished:
		return "published"
	case biz.ArticleStatusArchived:
		return "archived"
	default:
		return "draft"
	}
}
