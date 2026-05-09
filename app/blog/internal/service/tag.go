package service

import (
	"context"

	blogv1 "ley/api/blog/v1"
	"ley/app/blog/internal/biz"

	"github.com/go-kratos/kratos/v2/log"
)

type TagService struct {
	blogv1.UnimplementedTagServiceServer
	uc  *biz.TagUseCase
	log *log.Helper
}

func NewTagService(uc *biz.TagUseCase, logger log.Logger) *TagService {
	return &TagService{uc: uc, log: log.NewHelper(logger)}
}

func (s *TagService) CreateTag(ctx context.Context, req *blogv1.CreateTagRequest) (*blogv1.CreateTagReply, error) {
	t, err := s.uc.CreateTag(ctx, req.Name)
	if err != nil {
		return nil, err
	}
	return &blogv1.CreateTagReply{Tag: toTagInfo(t)}, nil
}

func (s *TagService) ListTags(ctx context.Context, _ *blogv1.ListTagsRequest) (*blogv1.ListTagsReply, error) {
	tags, err := s.uc.ListTags(ctx)
	if err != nil {
		return nil, err
	}
	infos := make([]*blogv1.TagInfo, 0, len(tags))
	for _, t := range tags {
		infos = append(infos, toTagInfo(t))
	}
	return &blogv1.ListTagsReply{Tags: infos}, nil
}

func (s *TagService) DeleteTag(ctx context.Context, req *blogv1.DeleteTagRequest) (*blogv1.DeleteTagReply, error) {
	return &blogv1.DeleteTagReply{}, s.uc.DeleteTag(ctx, uint(req.Id))
}

func toTagInfo(t *biz.Tag) *blogv1.TagInfo {
	if t == nil {
		return nil
	}
	return &blogv1.TagInfo{Id: uint64(t.ID), Name: t.Name, Slug: t.Slug, ArticleCount: t.ArticleCount}
}
