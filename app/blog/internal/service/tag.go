package service

import (
	"context"

	blogv1 "github.com/CycleZero/ley/api/blog/v1"
	"github.com/CycleZero/ley/app/blog/internal/biz"

	"github.com/go-kratos/kratos/v2/log"
)

// TagService 标签服务传输层：proto ↔ biz DTO 转换，仅在入口打 Debug 日志。
type TagService struct {
	blogv1.UnimplementedTagServiceServer
	uc  *biz.TagUseCase
	log *log.Helper
}

// NewTagService 构造标签服务（依赖由 Wire 注入）。
func NewTagService(uc *biz.TagUseCase, logger log.Logger) *TagService {
	return &TagService{uc: uc, log: log.NewHelper(logger)}
}

// CreateTag 创建标签（需管理员权限，鉴权由 entry 与 biz 双重收口）。
func (s *TagService) CreateTag(ctx context.Context, req *blogv1.CreateTagRequest) (*blogv1.CreateTagReply, error) {
	s.log.WithContext(ctx).Debug("收到创建标签请求")
	t, err := s.uc.CreateTag(ctx, req.Name)
	if err != nil {
		return nil, err
	}
	return &blogv1.CreateTagReply{Tag: toTagInfo(t)}, nil
}

// ListTags 查询全部标签（公开接口）。
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

// DeleteTag 删除标签（需管理员权限；存在关联文章时由 biz 拒绝）。
func (s *TagService) DeleteTag(ctx context.Context, req *blogv1.DeleteTagRequest) (*blogv1.DeleteTagReply, error) {
	s.log.WithContext(ctx).Debug("收到删除标签请求")
	return &blogv1.DeleteTagReply{}, s.uc.DeleteTag(ctx, uint(req.Id))
}

// toTagInfo 将 biz 标签对象转为 proto TagInfo。
func toTagInfo(t *biz.Tag) *blogv1.TagInfo {
	if t == nil {
		return nil
	}
	return &blogv1.TagInfo{Id: uint64(t.ID), Name: t.Name, Slug: t.Slug, ArticleCount: t.ArticleCount}
}
