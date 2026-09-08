package service

import (
	"context"

	blogv1 "github.com/CycleZero/ley/api/blog/v1"
	"github.com/CycleZero/ley/app/blog/internal/biz"

	"github.com/go-kratos/kratos/v2/log"
)

// CategoryService 分类服务传输层：树形分类的 proto ↔ biz DTO 转换。
type CategoryService struct {
	blogv1.UnimplementedCategoryServiceServer
	uc  *biz.CategoryUseCase
	log *log.Helper
}

// NewCategoryService 构造分类服务（依赖由 Wire 注入）。
func NewCategoryService(uc *biz.CategoryUseCase, logger log.Logger) *CategoryService {
	return &CategoryService{uc: uc, log: log.NewHelper(logger)}
}

// CreateCategory 创建分类（需管理员权限）。
func (s *CategoryService) CreateCategory(ctx context.Context, req *blogv1.CreateCategoryRequest) (*blogv1.CreateCategoryReply, error) {
	s.log.WithContext(ctx).Debug("收到创建分类请求")
	c, err := s.uc.CreateCategory(ctx, req.Name, req.Slug, req.Description, toUintPtr(req.ParentId), int(req.SortOrder))
	if err != nil {
		return nil, err
	}
	return &blogv1.CreateCategoryReply{Category: toCategoryInfo(c)}, nil
}

// ListCategories 查询分类树（公开接口）。
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

// UpdateCategory 更新分类（需管理员权限；循环引用由 biz 校验拒绝）。
func (s *CategoryService) UpdateCategory(ctx context.Context, req *blogv1.UpdateCategoryRequest) (*blogv1.UpdateCategoryReply, error) {
	s.log.WithContext(ctx).Debug("收到更新分类请求")
	c, err := s.uc.UpdateCategory(ctx, uint(req.Id), req.Name, req.Slug, req.Description, toUintPtr(req.ParentId), int(req.SortOrder))
	if err != nil {
		return nil, err
	}
	return &blogv1.UpdateCategoryReply{Category: toCategoryInfo(c)}, nil
}

// DeleteCategory 删除分类（需管理员权限；含子分类或文章时由 biz 拒绝）。
func (s *CategoryService) DeleteCategory(ctx context.Context, req *blogv1.DeleteCategoryRequest) (*blogv1.DeleteCategoryReply, error) {
	s.log.WithContext(ctx).Debug("收到删除分类请求")
	return &blogv1.DeleteCategoryReply{}, s.uc.DeleteCategory(ctx, uint(req.Id))
}

// toCategoryInfo 将 biz 分类对象递归转为 proto CategoryInfo（含子分类树）。
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
