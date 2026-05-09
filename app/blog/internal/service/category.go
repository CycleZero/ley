package service

import (
	"context"

	blogv1 "ley/api/blog/v1"
	"ley/app/blog/internal/biz"

	"github.com/go-kratos/kratos/v2/log"
)

type CategoryService struct {
	blogv1.UnimplementedCategoryServiceServer
	uc  *biz.CategoryUseCase
	log *log.Helper
}

func NewCategoryService(uc *biz.CategoryUseCase, logger log.Logger) *CategoryService {
	return &CategoryService{uc: uc, log: log.NewHelper(logger)}
}

func (s *CategoryService) CreateCategory(ctx context.Context, req *blogv1.CreateCategoryRequest) (*blogv1.CreateCategoryReply, error) {
	c, err := s.uc.CreateCategory(ctx, req.Name, req.Slug, req.Description, toUintPtr(req.ParentId), int(req.SortOrder))
	if err != nil {
		return nil, err
	}
	return &blogv1.CreateCategoryReply{Category: toCategoryInfo(c)}, nil
}

func (s *CategoryService) ListCategories(ctx context.Context, _ *blogv1.ListCategoriesRequest) (*blogv1.ListCategoriesReply, error) {
	cats, err := s.uc.ListCategories(ctx)
	if err != nil {
		return nil, err
	}
	infos := make([]*blogv1.CategoryInfo, 0, len(cats))
	for _, c := range cats {
		infos = append(infos, toCategoryInfo(c))
	}
	return &blogv1.ListCategoriesReply{Categories: infos}, nil
}

func (s *CategoryService) UpdateCategory(ctx context.Context, req *blogv1.UpdateCategoryRequest) (*blogv1.UpdateCategoryReply, error) {
	c, err := s.uc.UpdateCategory(ctx, uint(req.Id), req.Name, req.Slug, req.Description, toUintPtr(req.ParentId), int(req.SortOrder))
	if err != nil {
		return nil, err
	}
	return &blogv1.UpdateCategoryReply{Category: toCategoryInfo(c)}, nil
}

func (s *CategoryService) DeleteCategory(ctx context.Context, req *blogv1.DeleteCategoryRequest) (*blogv1.DeleteCategoryReply, error) {
	return &blogv1.DeleteCategoryReply{}, s.uc.DeleteCategory(ctx, uint(req.Id))
}

func toCategoryInfo(c *biz.Category) *blogv1.CategoryInfo {
	if c == nil {
		return nil
	}
	info := &blogv1.CategoryInfo{
		Id: uint64(c.ID), Name: c.Name, Slug: c.Slug, Description: c.Description,
		ParentId: derefUint(c.ParentID), SortOrder: int32(c.SortOrder), ArticleCount: c.ArticleCount,
	}
	for _, ch := range c.Children {
		info.Children = append(info.Children, toCategoryInfo(ch))
	}
	return info
}
