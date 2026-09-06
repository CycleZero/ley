package service

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	blogv1 "github.com/CycleZero/ley/api/blog/v1"
	"github.com/CycleZero/ley/app/blog/internal/biz"
	"github.com/CycleZero/ley/pkg/cache"
	"github.com/CycleZero/ley/pkg/meta"
	"github.com/CycleZero/ley/pkg/eventbus"
	mqPkg "github.com/CycleZero/ley/pkg/mq"
	"github.com/CycleZero/ley/pkg/testutil/datatest"
	"github.com/go-kratos/kratos/v2/log"
)

// =============================================================================
// Mock Repos（service 包按接口实现最小版本）
// =============================================================================

type mockArticleRepo struct {
	mu       sync.Mutex
	articles map[uint]*biz.Article
	nextID   uint
}

func newMockArticleRepo() *mockArticleRepo {
	return &mockArticleRepo{articles: make(map[uint]*biz.Article), nextID: 1}
}

func (m *mockArticleRepo) Create(ctx context.Context, a *biz.Article) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	a.ID = m.nextID
	m.nextID++
	a.CreatedAt, a.UpdatedAt = time.Now(), time.Now()
	cp := *a
	m.articles[a.ID] = &cp
	return nil
}
func (m *mockArticleRepo) Update(ctx context.Context, a *biz.Article) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	cp := *a
	cp.UpdatedAt = time.Now()
	m.articles[a.ID] = &cp
	return nil
}
func (m *mockArticleRepo) Delete(ctx context.Context, id uint) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.articles, id)
	return nil
}
func (m *mockArticleRepo) FindByID(ctx context.Context, id uint) (*biz.Article, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	a, ok := m.articles[id]
	if !ok {
		return nil, errors.New("not found")
	}
	cp := *a
	return &cp, nil
}
func (m *mockArticleRepo) FindBySlug(ctx context.Context, slug string) (*biz.Article, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, a := range m.articles {
		if a.Slug == slug {
			cp := *a
			return &cp, nil
		}
	}
	return nil, errors.New("not found")
}
func (m *mockArticleRepo) List(ctx context.Context, q biz.ArticleListQuery) ([]*biz.Article, int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]*biz.Article, 0, len(m.articles))
	for _, a := range m.articles {
		if q.Status != "" && articleStatusName(a.Status) != q.Status {
			continue
		}
		cp := *a
		out = append(out, &cp)
	}
	return out, int64(len(out)), nil
}
func (m *mockArticleRepo) Search(ctx context.Context, keyword string, page, pageSize int) ([]*biz.Article, int64, error) {
	return nil, 0, nil
}
func (m *mockArticleRepo) AssociateTags(ctx context.Context, articleID uint, tagIDs []uint) error { return nil }
func (m *mockArticleRepo) SyncTags(ctx context.Context, articleID uint, tagIDs []uint) error       { return nil }
func (m *mockArticleRepo) InsertLike(ctx context.Context, articleID, userID uint) error            { return nil }
func (m *mockArticleRepo) DeleteLike(ctx context.Context, articleID, userID uint) error            { return nil }
func (m *mockArticleRepo) IsLiked(ctx context.Context, articleID, userID uint) (bool, error)       { return false, nil }
func (m *mockArticleRepo) IncrementViewCount(ctx context.Context, id uint, delta int64) error      { return nil }
func (m *mockArticleRepo) UpdateTagsArticleCount(ctx context.Context, tagIDs []uint, delta int64) error {
	return nil
}
func (m *mockArticleRepo) FlushViewCounts(ctx context.Context, counts map[uint]int64) error { return nil }

type mockTagRepo struct {
	mu   sync.Mutex
	tags map[string]*biz.Tag
	next uint
}

func newMockTagRepo() *mockTagRepo {
	return &mockTagRepo{tags: make(map[string]*biz.Tag), next: 1}
}
func (m *mockTagRepo) Create(ctx context.Context, tag *biz.Tag) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if tag.ID == 0 {
		tag.ID = m.next
		m.next++
	}
	m.tags[tag.Name] = tag
	return nil
}
func (m *mockTagRepo) FindByName(ctx context.Context, name string) (*biz.Tag, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if t, ok := m.tags[name]; ok {
		return t, nil
	}
	return nil, errors.New("not found")
}
func (m *mockTagRepo) FindOrCreate(ctx context.Context, name, slug string) (*biz.Tag, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if t, ok := m.tags[name]; ok {
		return t, nil
	}
	t := &biz.Tag{ID: m.next, Name: name, Slug: slug}
	m.next++
	m.tags[name] = t
	return t, nil
}
func (m *mockTagRepo) List(ctx context.Context) ([]*biz.Tag, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]*biz.Tag, 0, len(m.tags))
	for _, t := range m.tags {
		out = append(out, t)
	}
	return out, nil
}
func (m *mockTagRepo) Delete(ctx context.Context, id uint) error { return nil }

type mockCategoryRepo struct{}

func (m *mockCategoryRepo) Create(ctx context.Context, cat *biz.Category) error {
	cat.ID = 1
	return nil
}
func (m *mockCategoryRepo) Update(ctx context.Context, cat *biz.Category) error { return nil }
func (m *mockCategoryRepo) Delete(ctx context.Context, id uint) error           { return nil }
func (m *mockCategoryRepo) FindByID(ctx context.Context, id uint) (*biz.Category, error) {
	return &biz.Category{ID: id, Name: "技术", Slug: "tech"}, nil
}
func (m *mockCategoryRepo) ListChildren(ctx context.Context, parentID uint) ([]*biz.Category, error) {
	pid := parentID
	return []*biz.Category{{ID: 2, Name: "子类", Slug: "sub", ParentID: &pid}}, nil
}
func (m *mockCategoryRepo) ListTree(ctx context.Context) ([]*biz.Category, error) {
	return []*biz.Category{{ID: 1, Name: "技术", Slug: "tech", ArticleCount: 1}}, nil
}
func (m *mockCategoryRepo) IncrementArticleCount(ctx context.Context, id uint, delta int64) error {
	return nil
}

type mockEB struct{}

func (m *mockEB) PublishAsync(ctx context.Context, topic string, event interface{}) error { return nil }
func (m *mockEB) PublishSync(ctx context.Context, topic string, event interface{}, timeout time.Duration) error {
	return nil
}
func (m *mockEB) Subscribe(ctx context.Context, topic string, handler eventbus.EventHandler, _ ...mqPkg.ConsumerOption) (eventbus.Subscription, error) {
	return nil, nil
}
func (m *mockEB) Close() error { return nil }

// =============================================================================
// Setup
// =============================================================================

// ctxWithUser 构造带用户身份的上下文（模拟网关透传）
func ctxWithUser(userID uint) context.Context {
	return meta.NewClientCtx(context.Background(), &meta.RequestMetaData{
		Auth: meta.Auth{UserID: uint64(userID), UserName: "tester"},
	})
}

func newTestArticleService(t *testing.T) *ArticleService {
	t.Helper()
	uc := biz.NewArticleUseCase(
		newMockArticleRepo(),
		newMockTagRepo(),
		&mockCategoryRepo{},
		&mockEB{},
		asCache(),
		nil,
		log.DefaultLogger,
	)
	return NewArticleService(uc, log.DefaultLogger)
}

// asCache 断言 cache.Cache 接口（datatest.InMemoryCache 实现）
func asCache() cache.Cache {
	return datatest.NewInMemoryCache()
}

// =============================================================================
// toArticleInfo 转换
// =============================================================================

func TestToArticleInfo(t *testing.T) {
	now := time.Now()
	published := now
	a := &biz.Article{
		ID: 42, Title: "标题", Slug: "slug-42", Content: "正文", Excerpt: "摘要",
		CoverImage: "https://img", Status: biz.ArticleStatusPublished,
		AuthorID: 7, AuthorName: "tester", AuthorAvatar: "https://av",
		CategoryID: uintPtr(3), CategoryName: "技术",
		ViewCount: 100, LikeCount: 5, IsTop: true, IsLiked: true,
		PublishedAt: &published, CreatedAt: now, UpdatedAt: now,
		Tags: []*biz.Tag{{ID: 1, Name: "Go"}, {ID: 2, Name: "Kratos"}},
	}

	info := toArticleInfo(a)
	if info.Id != 42 || info.Title != "标题" || info.Slug != "slug-42" {
		t.Errorf("基础字段映射错误: %+v", info)
	}
	if info.Status != "published" {
		t.Errorf("状态映射错误: %s", info.Status)
	}
	if info.Author.Username != "tester" || info.Author.Id != 7 {
		t.Errorf("作者映射错误: %+v", info.Author)
	}
	if info.Category == nil || info.Category.Name != "技术" || info.CategoryId != 3 {
		t.Errorf("分类映射错误: %+v", info.Category)
	}
	if len(info.Tags) != 2 || info.Tags[0].Name != "Go" {
		t.Errorf("标签映射错误: %+v", info.Tags)
	}
	if !info.IsTop || !info.IsLiked || info.ViewCount != 100 {
		t.Errorf("布尔/计数映射错误: %+v", info)
	}
	if info.PublishedAt == "" {
		t.Error("PublishedAt 应非空")
	}
}

func TestToArticleInfoNil(t *testing.T) {
	if toArticleInfo(nil) != nil {
		t.Error("nil 文章应返回 nil")
	}
}

func TestToArticleInfoDraftWithoutCategory(t *testing.T) {
	a := &biz.Article{ID: 1, Title: "草稿", Status: biz.ArticleStatusDraft, CreatedAt: time.Now(), UpdatedAt: time.Now()}
	info := toArticleInfo(a)
	if info.Status != "draft" {
		t.Errorf("状态映射错误: %s", info.Status)
	}
	if info.Category != nil {
		t.Error("无分类应为 nil")
	}
	if info.PublishedAt != "" {
		t.Error("草稿无发布时间")
	}
}

func TestArticleStatusStr(t *testing.T) {
	cases := map[biz.ArticleStatus]string{
		biz.ArticleStatusPublished: "published",
		biz.ArticleStatusArchived:  "archived",
		biz.ArticleStatusDraft:      "draft",
		biz.ArticleStatus(99):       "draft", // 未知状态回退 draft
	}
	for status, want := range cases {
		if got := articleStatusStr(status); got != want {
			t.Errorf("articleStatusStr(%d) = %q, want %q", status, got, want)
		}
	}
}

// =============================================================================
// utils 指针转换
// =============================================================================

func TestPtrHelpers(t *testing.T) {
	if strPtr("") != nil {
		t.Error("空字符串应返回 nil")
	}
	if strPtr("x") == nil || *strPtr("x") != "x" {
		t.Error("strPtr 转换错误")
	}
	if toUintPtr(0) != nil {
		t.Error("0 应返回 nil")
	}
	if toUintPtr(5) == nil || *toUintPtr(5) != 5 {
		t.Error("toUintPtr 转换错误")
	}
	if derefUint(nil) != 0 {
		t.Error("nil 解引用应为 0")
	}
	n := uint(9)
	if derefUint(&n) != 9 {
		t.Error("derefUint 转换错误")
	}
	if derefBool(nil) != false {
		t.Error("nil bool 解引用应为 false")
	}
	b := true
	if derefBool(&b) != true {
		t.Error("derefBool 转换错误")
	}
	if boolPtr(false) == nil || *boolPtr(false) != false {
		t.Error("boolPtr 转换错误")
	}
}

// =============================================================================
// Handler 链路
// =============================================================================

func TestServiceCreateArticle(t *testing.T) {
	svc := newTestArticleService(t)
	resp, err := svc.CreateArticle(ctxWithUser(1), &blogv1.CreateArticleRequest{
		Title:    "我的文章",
		Content:  "正文内容",
		TagNames: []string{"Go"},
	})
	if err != nil {
		t.Fatalf("CreateArticle 失败: %v", err)
	}
	if resp.Article.Id == 0 {
		t.Error("返回的文章应有 ID")
	}
	if resp.Article.Title != "我的文章" {
		t.Errorf("标题不匹配: %s", resp.Article.Title)
	}
}

func TestServiceGetArticleBySlug(t *testing.T) {
	svc := newTestArticleService(t)
	created, err := svc.CreateArticle(ctxWithUser(1), &blogv1.CreateArticleRequest{Title: "文章A", Content: "c"})
	if err != nil {
		t.Fatalf("CreateArticle 失败: %v", err)
	}

	resp, err := svc.GetArticle(ctxWithUser(1), &blogv1.GetArticleRequest{Identifier: created.Article.Slug})
	if err != nil {
		t.Fatalf("GetArticle 失败: %v", err)
	}
	if resp.Article.Id != created.Article.Id {
		t.Errorf("文章不匹配: %d != %d", resp.Article.Id, created.Article.Id)
	}
}

func TestServiceListArticles(t *testing.T) {
	svc := newTestArticleService(t)
	ctx := ctxWithUser(1)
	a1, err := svc.CreateArticle(ctx, &blogv1.CreateArticleRequest{Title: "文章1", Content: "c"})
	if err != nil {
		t.Fatalf("CreateArticle 失败: %v", err)
	}
	a2, err := svc.CreateArticle(ctx, &blogv1.CreateArticleRequest{Title: "文章2", Content: "c"})
	if err != nil {
		t.Fatalf("CreateArticle 失败: %v", err)
	}
	// 发布后才能被 ListArticles（默认只查已发布）命中
	if _, err := svc.PublishArticle(ctx, &blogv1.PublishArticleRequest{Id: a1.Article.Id}); err != nil {
		t.Fatalf("PublishArticle 失败: %v", err)
	}
	if _, err := svc.PublishArticle(ctx, &blogv1.PublishArticleRequest{Id: a2.Article.Id}); err != nil {
		t.Fatalf("PublishArticle 失败: %v", err)
	}

	resp, err := svc.ListArticles(ctx, &blogv1.ListArticlesRequest{Page: 1, PageSize: 10})
	if err != nil {
		t.Fatalf("ListArticles 失败: %v", err)
	}
	if resp.Total != 2 {
		t.Errorf("Total 应为 2: %d", resp.Total)
	}
	// 发布后状态应正确
	for _, a := range resp.Articles {
		if a.Status != "published" {
			t.Errorf("文章状态应为 published: %s", a.Status)
		}
	}
}

func TestServiceDeleteArticle(t *testing.T) {
	svc := newTestArticleService(t)
	created, err := svc.CreateArticle(ctxWithUser(1), &blogv1.CreateArticleRequest{Title: "待删", Content: "c"})
	if err != nil {
		t.Fatalf("CreateArticle 失败: %v", err)
	}
	if _, err := svc.DeleteArticle(ctxWithUser(1), &blogv1.DeleteArticleRequest{Id: uint64(created.Article.Id)}); err != nil {
		t.Fatalf("DeleteArticle 失败: %v", err)
	}
}

// 错误透传：删除不存在的文章应返回 404（Kratos errors）
func TestServiceGetArticleNotFound(t *testing.T) {
	svc := newTestArticleService(t)
	_, err := svc.GetArticle(ctxWithUser(1), &blogv1.GetArticleRequest{Identifier: "no-such-slug"})
	if err == nil {
		t.Fatal("不存在的文章应报错")
	}
}

func uintPtr(v uint) *uint { return &v }

// articleStatusName 测试辅助：ArticleStatus(int8) → 名称
func articleStatusName(s biz.ArticleStatus) string {
	switch s {
	case biz.ArticleStatusPublished:
		return "published"
	case biz.ArticleStatusArchived:
		return "archived"
	default:
		return "draft"
	}
}

// B-106: CreateArticle 支持 status=published，创建后立即发布并可被公开列表命中
func TestServiceCreateArticlePublishesImmediately(t *testing.T) {
	svc := newTestArticleService(t)

	resp, err := svc.CreateArticle(ctxWithUser(1), &blogv1.CreateArticleRequest{
		Title:   "创建即发布",
		Content: "正文",
		Status:  "published",
	})
	if err != nil {
		t.Fatalf("CreateArticle 失败: %v", err)
	}
	if resp.Article.Status != "published" {
		t.Errorf("status=published 创建应返回 published: got %q", resp.Article.Status)
	}

	list, err := svc.ListArticles(context.Background(), &blogv1.ListArticlesRequest{})
	if err != nil {
		t.Fatalf("ListArticles 失败: %v", err)
	}
	found := false
	for _, a := range list.Articles {
		if a.Id == resp.Article.Id {
			found = true
			break
		}
	}
	if !found {
		t.Error("创建即发布的文章应出现在公开列表")
	}
}
