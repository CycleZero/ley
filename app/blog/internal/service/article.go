package service

import (
	"context"

	blogv1 "ley/api/blog/v1"
	commonv1 "ley/api/common/v1"
	"ley/app/blog/internal/biz"

	"github.com/go-kratos/kratos/v2/log"
)

type ArticleService struct {
	blogv1.UnimplementedArticleServiceServer
	uc  *biz.ArticleUseCase
	log *log.Helper
}

func NewArticleService(uc *biz.ArticleUseCase, logger log.Logger) *ArticleService {
	return &ArticleService{uc: uc, log: log.NewHelper(logger)}
}

func (s *ArticleService) CreateArticle(ctx context.Context, req *blogv1.CreateArticleRequest) (*blogv1.CreateArticleReply, error) {
	a, err := s.uc.CreateArticle(ctx, req.Title, req.Content, req.Excerpt, req.CoverImage, toUintPtr(req.CategoryId), req.TagNames)
	if err != nil {
		return nil, err
	}
	return &blogv1.CreateArticleReply{Article: toArticleInfo(a)}, nil
}

func (s *ArticleService) UpdateArticle(ctx context.Context, req *blogv1.UpdateArticleRequest) (*blogv1.UpdateArticleReply, error) {
	a, err := s.uc.UpdateArticle(ctx, uint(req.Id), strPtr(req.Title), strPtr(req.Content), strPtr(req.Excerpt), strPtr(req.CoverImage), toUintPtr(req.CategoryId), req.TagNames)
	if err != nil {
		return nil, err
	}
	return &blogv1.UpdateArticleReply{Article: toArticleInfo(a)}, nil
}

func (s *ArticleService) DeleteArticle(ctx context.Context, req *blogv1.DeleteArticleRequest) (*blogv1.DeleteArticleReply, error) {
	return &blogv1.DeleteArticleReply{}, s.uc.DeleteArticle(ctx, uint(req.Id))
}

func (s *ArticleService) GetArticle(ctx context.Context, req *blogv1.GetArticleRequest) (*blogv1.GetArticleReply, error) {
	a, err := s.uc.GetArticle(ctx, req.Identifier)
	if err != nil {
		return nil, err
	}
	return &blogv1.GetArticleReply{Article: toArticleInfo(a)}, nil
}

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

func (s *ArticleService) PublishArticle(ctx context.Context, req *blogv1.PublishArticleRequest) (*blogv1.PublishArticleReply, error) {
	a, err := s.uc.PublishArticle(ctx, uint(req.Id))
	if err != nil {
		return nil, err
	}
	return &blogv1.PublishArticleReply{Article: toArticleInfo(a)}, nil
}

func (s *ArticleService) ArchiveArticle(ctx context.Context, req *blogv1.ArchiveArticleRequest) (*blogv1.ArchiveArticleReply, error) {
	a, err := s.uc.ArchiveArticle(ctx, uint(req.Id))
	if err != nil {
		return nil, err
	}
	return &blogv1.ArchiveArticleReply{Article: toArticleInfo(a)}, nil
}

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

func (s *ArticleService) LikeArticle(ctx context.Context, req *blogv1.LikeArticleRequest) (*blogv1.LikeArticleReply, error) {
	return &blogv1.LikeArticleReply{}, s.uc.LikeArticle(ctx, uint(req.Id))
}

func (s *ArticleService) UnlikeArticle(ctx context.Context, req *blogv1.UnlikeArticleRequest) (*blogv1.UnlikeArticleReply, error) {
	return &blogv1.UnlikeArticleReply{}, s.uc.UnlikeArticle(ctx, uint(req.Id))
}

func toArticleInfo(a *biz.Article) *blogv1.ArticleInfo {
	if a == nil {
		return nil
	}
	info := &blogv1.ArticleInfo{
		Id: uint64(a.ID), Title: a.Title, Slug: a.Slug, Content: a.Content, Excerpt: a.Excerpt, CoverImage: a.CoverImage,
		Status: articleStatusStr(a.Status), AuthorId: uint64(a.AuthorID),
		Author: &commonv1.AuthorInfo{Id: uint64(a.AuthorID), Username: a.AuthorName, Avatar: a.AuthorAvatar},
		CategoryId: derefUint(a.CategoryID), ViewCount: a.ViewCount, LikeCount: a.LikeCount, CommentCount: a.CommentCount,
		IsTop: a.IsTop, IsLiked: a.IsLiked, CreatedAt: a.CreatedAt.Format("2006-01-02T15:04:05Z"), UpdatedAt: a.UpdatedAt.Format("2006-01-02T15:04:05Z"),
	}
	if a.CategoryID != nil {
		info.Category = &blogv1.CategoryInfo{Id: uint64(*a.CategoryID), Name: a.CategoryName}
	}
	if a.PublishedAt != nil {
		info.PublishedAt = a.PublishedAt.Format("2006-01-02T15:04:05Z")
	}
	if len(a.Tags) > 0 {
		info.Tags = make([]*blogv1.TagInfo, 0, len(a.Tags))
		for _, t := range a.Tags {
			info.Tags = append(info.Tags, toTagInfo(t))
		}
	}
	return info
}

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
