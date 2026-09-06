package service

import (
	"context"
	"errors"
	"io"
	"testing"

	blogv1 "github.com/CycleZero/ley/api/blog/v1"
	"github.com/CycleZero/ley/app/blog/internal/biz"
	"github.com/CycleZero/ley/pkg/meta"
	"github.com/go-kratos/kratos/v2/log"
)

// =============================================================================
// TagService
// =============================================================================

func newTestTagService(t *testing.T) *TagService {
	t.Helper()
	uc := biz.NewTagUseCase(newMockTagRepo(), log.DefaultLogger)
	return NewTagService(uc, log.DefaultLogger)
}

func TestServiceCreateTag(t *testing.T) {
	svc := newTestTagService(t)
	resp, err := svc.CreateTag(context.Background(), &blogv1.CreateTagRequest{Name: "Go"})
	if err != nil {
		t.Fatalf("CreateTag 失败: %v", err)
	}
	if resp.Tag.Name != "Go" {
		t.Errorf("标签名不匹配: %s", resp.Tag.Name)
	}
	if resp.Tag.Id == 0 {
		t.Error("标签应有 ID")
	}
}

func TestServiceCreateTagEmpty(t *testing.T) {
	svc := newTestTagService(t)
	if _, err := svc.CreateTag(context.Background(), &blogv1.CreateTagRequest{Name: "  "}); err == nil {
		t.Fatal("空标签名应报错")
	}
}

func TestServiceListTags(t *testing.T) {
	svc := newTestTagService(t)
	svc.CreateTag(context.Background(), &blogv1.CreateTagRequest{Name: "Go"})
	svc.CreateTag(context.Background(), &blogv1.CreateTagRequest{Name: "React"})

	resp, err := svc.ListTags(context.Background(), &blogv1.ListTagsRequest{})
	if err != nil {
		t.Fatalf("ListTags 失败: %v", err)
	}
	if len(resp.Tags) != 2 {
		t.Errorf("应有 2 个标签: %d", len(resp.Tags))
	}
}

// =============================================================================
// CategoryService
// =============================================================================

func newTestCategoryService(t *testing.T) *CategoryService {
	t.Helper()
	uc := biz.NewCategoryUseCase(&mockCategoryRepo{}, log.DefaultLogger)
	return NewCategoryService(uc, log.DefaultLogger)
}

func TestServiceCreateCategory(t *testing.T) {
	svc := newTestCategoryService(t)
	resp, err := svc.CreateCategory(context.Background(), &blogv1.CreateCategoryRequest{Name: "技术"})
	if err != nil {
		t.Fatalf("Create 失败: %v", err)
	}
	if resp.Category.Id == 0 {
		t.Error("分类应有 ID")
	}
}

func TestServiceListCategories(t *testing.T) {
	svc := newTestCategoryService(t)
	resp, err := svc.ListCategories(context.Background(), &blogv1.ListCategoriesRequest{})
	if err != nil {
		t.Fatalf("ListCategories 失败: %v", err)
	}
	if len(resp.Categories) == 0 {
		t.Error("列表不应为空")
	}
	if resp.Categories[0].Name != "技术" {
		t.Errorf("分类名不匹配: %s", resp.Categories[0].Name)
	}
}

func TestServiceUpdateCategory(t *testing.T) {
	svc := newTestCategoryService(t)
	resp, err := svc.UpdateCategory(context.Background(), &blogv1.UpdateCategoryRequest{
		Id:   1,
		Name: "新分类名",
		Slug: "new-slug",
	})
	if err != nil {
		t.Fatalf("Update 失败: %v", err)
	}
	if resp.Category == nil {
		t.Error("应返回更新后的分类")
	}
}

// 业务约束：slug 必填
func TestServiceUpdateCategoryWithoutSlug(t *testing.T) {
	svc := newTestCategoryService(t)
	_, err := svc.UpdateCategory(context.Background(), &blogv1.UpdateCategoryRequest{
		Id:   1,
		Name: "新分类名",
	})
	if err == nil {
		t.Fatal("缺 slug 应报错")
	}
}

// 业务约束：有子分类的分类不能删除（mock ListChildren 返回 1 个子类）
func TestServiceDeleteCategoryWithChildren(t *testing.T) {
	svc := newTestCategoryService(t)
	_, err := svc.DeleteCategory(context.Background(), &blogv1.DeleteCategoryRequest{Id: 1})
	if err == nil {
		t.Fatal("有子分类应拒绝删除")
	}
}

// =============================================================================
// SiteService
// =============================================================================

// mockSiteRepo 实现 biz.SiteRepo
type mockSiteRepo struct {
	cfg *biz.SiteSetting
}

func (m *mockSiteRepo) IsLikesEnabled(ctx context.Context) (bool, error) { return true, nil }
func (m *mockSiteRepo) GetConfig(ctx context.Context) (*biz.SiteSetting, error) {
	if m.cfg == nil {
		return nil, errors.New("not found")
	}
	cp := *m.cfg
	return &cp, nil
}
func (m *mockSiteRepo) SaveConfig(ctx context.Context, config *biz.SiteSetting) error {
	cp := *config
	m.cfg = &cp
	return nil
}
func (m *mockSiteRepo) CreateBackground(ctx context.Context, bg *biz.SiteBackground, file io.Reader) error {
	return nil
}
func (m *mockSiteRepo) DeleteBackground(ctx context.Context, id uint) error { return nil }
func (m *mockSiteRepo) ListBackgrounds(ctx context.Context) ([]*biz.SiteBackground, error) {
	return []*biz.SiteBackground{{ID: 1, URL: "https://bg/1.png", IsActive: true}}, nil
}
func (m *mockSiteRepo) SetActiveBackground(ctx context.Context, id uint) error { return nil }

func newTestSiteService(t *testing.T) (*SiteService, *mockSiteRepo) {
	t.Helper()
	repo := &mockSiteRepo{cfg: &biz.SiteSetting{SiteTitle: "Ley"}}
	uc := biz.NewSiteUseCase(repo, log.DefaultLogger)
	return NewSiteService(uc, log.DefaultLogger), repo
}

func TestServiceGetSiteConfig(t *testing.T) {
	svc, repo := newTestSiteService(t)
	repo.cfg = &biz.SiteSetting{SiteTitle: "Ley", SiteSubtitle: "记录与分享"}

	resp, err := svc.GetSiteConfig(context.Background(), &blogv1.GetSiteConfigRequest{})
	if err != nil {
		t.Fatalf("GetSiteConfig 失败: %v", err)
	}
	if resp.Config.SiteTitle != "Ley" {
		t.Errorf("配置不匹配: %+v", resp.Config)
	}
}

func TestServiceUpdateSiteConfig(t *testing.T) {
	svc, _ := newTestSiteService(t)
	adminCtx := meta.NewClientCtx(context.Background(), &meta.RequestMetaData{
		Auth: meta.Auth{UserID: 1, UserName: "admin", Role: "admin"},
	})
	resp, err := svc.UpdateSiteConfig(adminCtx, &blogv1.UpdateSiteConfigRequest{
		Config: &blogv1.SiteConfig{
			SiteTitle: "我的博客",
			IcpNumber: "京ICP备00000000号",
		},
	})
	if err != nil {
		t.Fatalf("UpdateSiteConfig 失败: %v", err)
	}
	if resp.Config.SiteTitle != "我的博客" {
		t.Errorf("配置未更新: %+v", resp.Config)
	}
}

func TestServiceListBackgrounds(t *testing.T) {
	svc, _ := newTestSiteService(t)
	resp, err := svc.ListBackgrounds(context.Background(), &blogv1.ListBackgroundsRequest{})
	if err != nil {
		t.Fatalf("ListBackgrounds 失败: %v", err)
	}
	if len(resp.Backgrounds) != 1 || !resp.Backgrounds[0].IsActive {
		t.Errorf("背景列表不匹配: %+v", resp.Backgrounds)
	}
}
