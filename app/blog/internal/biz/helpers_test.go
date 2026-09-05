package biz

import (
	"context"
	"io"
	"strings"
	"sync"
	"time"

	"github.com/CycleZero/ley/pkg/cache"
	"github.com/CycleZero/ley/pkg/eventbus"
	"github.com/CycleZero/ley/pkg/meta"
	mqPkg "github.com/CycleZero/ley/pkg/mq"
	"github.com/go-kratos/kratos/v2/log"
)

var _ ArticleRepo = (*mockArticleRepo)(nil)
var _ TagRepo = (*mockTagRepo)(nil)
var _ CategoryRepo = (*mockCategoryRepo)(nil)
var _ SiteRepo = (*mockSiteRepo)(nil)
var _ eventbus.EventBus = (*mockEventBus)(nil)

// =============================================================================
// Mock Repos
// =============================================================================

type mockArticleRepo struct {
	mu           sync.Mutex
	articles     map[uint]*Article
	slugIndex    map[string]uint
	likes        map[uint]map[uint]bool
	tags         map[uint][]uint
	nextID       uint
	CreateFn     func(ctx context.Context, article *Article) error
	UpdateFn     func(ctx context.Context, article *Article) error
	DeleteFn     func(ctx context.Context, id uint) error
	FindByIDFn   func(ctx context.Context, id uint) (*Article, error)
	FindBySlugFn func(ctx context.Context, slug string) (*Article, error)
	ListFn       func(ctx context.Context, query ArticleListQuery) ([]*Article, int64, error)
	SearchFn     func(ctx context.Context, keyword string, page, pageSize int) ([]*Article, int64, error)
}

func newMockArticleRepo() *mockArticleRepo {
	return &mockArticleRepo{
		articles:  make(map[uint]*Article),
		slugIndex: make(map[string]uint),
		likes:     make(map[uint]map[uint]bool),
		tags:      make(map[uint][]uint),
		nextID:    1,
	}
}

func (m *mockArticleRepo) Create(ctx context.Context, a *Article) error {
	if m.CreateFn != nil { return m.CreateFn(ctx, a) }
	m.mu.Lock(); defer m.mu.Unlock()
	if _, ok := m.slugIndex[a.Slug]; ok { return ErrSlugAlreadyExists }
	a.ID = m.nextID; m.nextID++
	a.CreatedAt = time.Now(); a.UpdatedAt = a.CreatedAt
	cp := *a; m.articles[a.ID] = &cp
	m.slugIndex[a.Slug] = a.ID
	return nil
}
func (m *mockArticleRepo) Update(ctx context.Context, a *Article) error {
	if m.UpdateFn != nil { return m.UpdateFn(ctx, a) }
	m.mu.Lock(); defer m.mu.Unlock()
	if _, ok := m.articles[a.ID]; !ok { return ErrArticleNotFound }
	a.UpdatedAt = time.Now()
	cp := *a; m.articles[a.ID] = &cp
	return nil
}
func (m *mockArticleRepo) Delete(ctx context.Context, id uint) error {
	if m.DeleteFn != nil { return m.DeleteFn(ctx, id) }
	m.mu.Lock(); defer m.mu.Unlock()
	if _, ok := m.articles[id]; !ok { return ErrArticleNotFound }
	delete(m.articles, id)
	return nil
}
func (m *mockArticleRepo) FindByID(ctx context.Context, id uint) (*Article, error) {
	if m.FindByIDFn != nil { return m.FindByIDFn(ctx, id) }
	m.mu.Lock(); defer m.mu.Unlock()
	a, ok := m.articles[id]
	if !ok { return nil, ErrArticleNotFound }
	cp := *a; return &cp, nil
}
func (m *mockArticleRepo) FindBySlug(ctx context.Context, slug string) (*Article, error) {
	if m.FindBySlugFn != nil { return m.FindBySlugFn(ctx, slug) }
	m.mu.Lock(); defer m.mu.Unlock()
	id, ok := m.slugIndex[slug]
	if !ok { return nil, ErrArticleNotFound }
	a := m.articles[id]
	cp := *a; return &cp, nil
}
func (m *mockArticleRepo) List(ctx context.Context, query ArticleListQuery) ([]*Article, int64, error) {
	if m.ListFn != nil { return m.ListFn(ctx, query) }
	m.mu.Lock(); defer m.mu.Unlock()
	result := make([]*Article, 0)
	for _, a := range m.articles {
		if query.AuthorID != nil && a.AuthorID != *query.AuthorID { continue }
		cp := *a; result = append(result, &cp)
	}
	return result, int64(len(result)), nil
}
func (m *mockArticleRepo) Search(ctx context.Context, keyword string, page, pageSize int) ([]*Article, int64, error) {
	if m.SearchFn != nil { return m.SearchFn(ctx, keyword, page, pageSize) }
	m.mu.Lock(); defer m.mu.Unlock()
	result := make([]*Article, 0)
	for _, a := range m.articles {
		if a.Status != ArticleStatusPublished { continue }
		if !strings.Contains(a.Title, keyword) && !strings.Contains(a.Content, keyword) { continue }
		cp := *a; result = append(result, &cp)
	}
	return result, int64(len(result)), nil
}
func (m *mockArticleRepo) AssociateTags(ctx context.Context, articleID uint, tagIDs []uint) error {
	m.mu.Lock(); defer m.mu.Unlock()
	m.tags[articleID] = append(m.tags[articleID], tagIDs...)
	return nil
}
func (m *mockArticleRepo) SyncTags(ctx context.Context, articleID uint, tagIDs []uint) error {
	m.mu.Lock(); defer m.mu.Unlock()
	m.tags[articleID] = tagIDs
	return nil
}
func (m *mockArticleRepo) InsertLike(ctx context.Context, articleID, userID uint) error {
	m.mu.Lock(); defer m.mu.Unlock()
	if m.likes[articleID] == nil { m.likes[articleID] = make(map[uint]bool) }
	m.likes[articleID][userID] = true
	return nil
}
func (m *mockArticleRepo) DeleteLike(ctx context.Context, articleID, userID uint) error {
	m.mu.Lock(); defer m.mu.Unlock()
	if m.likes[articleID] != nil { delete(m.likes[articleID], userID) }
	return nil
}
func (m *mockArticleRepo) IsLiked(ctx context.Context, articleID, userID uint) (bool, error) {
	m.mu.Lock(); defer m.mu.Unlock()
	return m.likes[articleID] != nil && m.likes[articleID][userID], nil
}
func (m *mockArticleRepo) IncrementViewCount(ctx context.Context, id uint, delta int64) error {
	m.mu.Lock(); defer m.mu.Unlock()
	if a, ok := m.articles[id]; ok { a.ViewCount += delta }
	return nil
}
func (m *mockArticleRepo) UpdateTagsArticleCount(ctx context.Context, tagIDs []uint, delta int64) error { return nil }
func (m *mockArticleRepo) FlushViewCounts(ctx context.Context, counts map[uint]int64) error { return nil }

// ---- tag repo mock ----

type mockTagRepo struct {
	mu     sync.Mutex
	tags   map[string]*Tag
	nextID uint
}

func newMockTagRepo() *mockTagRepo {
	return &mockTagRepo{tags: make(map[string]*Tag), nextID: 1}
}
func (m *mockTagRepo) Create(ctx context.Context, tag *Tag) error {
	m.mu.Lock(); defer m.mu.Unlock()
	if _, ok := m.tags[tag.Name]; ok { return ErrTagNameExists }
	tag.ID = m.nextID; m.nextID++
	tag.CreatedAt = time.Now()
	cp := *tag; m.tags[tag.Name] = &cp
	return nil
}
func (m *mockTagRepo) FindByName(ctx context.Context, name string) (*Tag, error) {
	m.mu.Lock(); defer m.mu.Unlock()
	t, ok := m.tags[name]
	if !ok { return nil, ErrTagNotFound }
	cp := *t; return &cp, nil
}
func (m *mockTagRepo) FindOrCreate(ctx context.Context, name, slug string) (*Tag, error) {
	m.mu.Lock(); defer m.mu.Unlock()
	if t, ok := m.tags[name]; ok { cp := *t; return &cp, nil }
	tag := &Tag{Name: name, Slug: slug, ID: m.nextID, CreatedAt: time.Now()}
	m.nextID++
	m.tags[name] = tag
	cp := *tag; return &cp, nil
}
func (m *mockTagRepo) List(ctx context.Context) ([]*Tag, error) {
	m.mu.Lock(); defer m.mu.Unlock()
	result := make([]*Tag, 0, len(m.tags))
	for _, t := range m.tags { cp := *t; result = append(result, &cp) }
	return result, nil
}
func (m *mockTagRepo) Delete(ctx context.Context, id uint) error {
	m.mu.Lock(); defer m.mu.Unlock()
	for k, v := range m.tags { if v.ID == id { delete(m.tags, k); return nil } }
	return ErrTagNotFound
}

// ---- category repo mock ----

type mockCategoryRepo struct {
	mu     sync.Mutex
	cats   map[uint]*Category
	nextID uint
}

func newMockCategoryRepo() *mockCategoryRepo {
	return &mockCategoryRepo{cats: make(map[uint]*Category), nextID: 1}
}
func (m *mockCategoryRepo) Create(ctx context.Context, cat *Category) error {
	m.mu.Lock(); defer m.mu.Unlock()
	cat.ID = m.nextID; m.nextID++
	cat.CreatedAt = time.Now()
	cp := *cat; m.cats[cat.ID] = &cp
	return nil
}
func (m *mockCategoryRepo) Update(ctx context.Context, cat *Category) error {
	m.mu.Lock(); defer m.mu.Unlock()
	if _, ok := m.cats[cat.ID]; !ok { return ErrCategoryNotFound }
	cp := *cat; m.cats[cat.ID] = &cp
	return nil
}
func (m *mockCategoryRepo) Delete(ctx context.Context, id uint) error {
	m.mu.Lock(); defer m.mu.Unlock()
	if _, ok := m.cats[id]; !ok { return ErrCategoryNotFound }
	delete(m.cats, id)
	return nil
}
func (m *mockCategoryRepo) FindByID(ctx context.Context, id uint) (*Category, error) {
	m.mu.Lock(); defer m.mu.Unlock()
	c, ok := m.cats[id]
	if !ok { return nil, ErrCategoryNotFound }
	cp := *c; return &cp, nil
}
func (m *mockCategoryRepo) ListChildren(ctx context.Context, parentID uint) ([]*Category, error) {
	m.mu.Lock(); defer m.mu.Unlock()
	var result []*Category
	for _, c := range m.cats {
		if c.ParentID != nil && *c.ParentID == parentID {
			cp := *c; result = append(result, &cp)
		}
	}
	return result, nil
}
func (m *mockCategoryRepo) ListTree(ctx context.Context) ([]*Category, error) {
	m.mu.Lock(); defer m.mu.Unlock()
	var roots []*Category
	for _, c := range m.cats {
		if c.ParentID == nil {
			cp := *c; roots = append(roots, &cp)
		}
	}
	return roots, nil
}
func (m *mockCategoryRepo) IncrementArticleCount(ctx context.Context, id uint, delta int64) error {
	m.mu.Lock(); defer m.mu.Unlock()
	if c, ok := m.cats[id]; ok { c.ArticleCount += delta }
	return nil
}

// ---- site repo mock ----

type mockSiteRepo struct {
	config       *SiteSetting
	bgs          []*SiteBackground
	bgNextID     uint
	GetConfigFn  func(ctx context.Context) (*SiteSetting, error)
	SaveConfigFn func(ctx context.Context, cfg *SiteSetting) error
}

func newMockSiteRepo() *mockSiteRepo {
	return &mockSiteRepo{config: &SiteSetting{}, bgNextID: 1}
}
func (m *mockSiteRepo) GetConfig(ctx context.Context) (*SiteSetting, error) {
	if m.GetConfigFn != nil { return m.GetConfigFn(ctx) }
	if m.config == nil { return &SiteSetting{}, nil }
	cp := *m.config; return &cp, nil
}
func (m *mockSiteRepo) SaveConfig(ctx context.Context, cfg *SiteSetting) error {
	if m.SaveConfigFn != nil { return m.SaveConfigFn(ctx, cfg) }
	cp := *cfg; m.config = &cp
	return nil
}
func (m *mockSiteRepo) CreateBackground(ctx context.Context, bg *SiteBackground, file io.Reader) error {
	bg.ID = m.bgNextID; m.bgNextID++
	bg.CreatedAt = time.Now()
	cp := *bg; m.bgs = append(m.bgs, &cp)
	return nil
}
func (m *mockSiteRepo) DeleteBackground(ctx context.Context, id uint) error {
	for i, b := range m.bgs {
		if b.ID == id { m.bgs = append(m.bgs[:i], m.bgs[i+1:]...); return nil }
	}
	return ErrBackgroundNotFound
}
func (m *mockSiteRepo) ListBackgrounds(ctx context.Context) ([]*SiteBackground, error) {
	result := make([]*SiteBackground, len(m.bgs))
	for i, b := range m.bgs { cp := *b; result[i] = &cp }
	return result, nil
}
func (m *mockSiteRepo) SetActiveBackground(ctx context.Context, id uint) error {
	for _, b := range m.bgs { b.IsActive = (b.ID == id) }
	return nil
}

// =============================================================================
// Mock Event Bus
// =============================================================================

type mockEventBus struct {
	mu      sync.Mutex
	events  []mockEvent
}

type mockEvent struct {
	Topic string
	Data  interface{}
}

func newMockEventBus() *mockEventBus { return &mockEventBus{} }

func (m *mockEventBus) PublishAsync(ctx context.Context, topic string, event interface{}) error {
	m.mu.Lock(); defer m.mu.Unlock()
	m.events = append(m.events, mockEvent{Topic: topic, Data: event})
	return nil
}
func (m *mockEventBus) PublishSync(ctx context.Context, topic string, event interface{}, timeout time.Duration) error {
	return m.PublishAsync(ctx, topic, event)
}
func (m *mockEventBus) Subscribe(ctx context.Context, topic string, handler eventbus.EventHandler, opts ...mqPkg.ConsumerOption) (eventbus.Subscription, error) {
	return nil, nil
}
func (m *mockEventBus) Close() error { return nil }
func (m *mockEventBus) Events() []mockEvent { m.mu.Lock(); defer m.mu.Unlock(); return m.events }
func (m *mockEventBus) Reset() { m.mu.Lock(); defer m.mu.Unlock(); m.events = nil }

// =============================================================================
// Mock Cache
// =============================================================================

type mockCache struct{}

func newMockCache() cache.Cache { return &mockCache{} }

func (m *mockCache) Get(ctx context.Context, key string) ([]byte, error)           { return nil, cache.ErrKeyNotFound }
func (m *mockCache) GetObject(ctx context.Context, key string, value any) error     { return cache.ErrKeyNotFound }
func (m *mockCache) MGet(ctx context.Context, keys []string) (map[string][]byte, error) { return make(map[string][]byte), nil }
func (m *mockCache) Set(ctx context.Context, key string, value any, expiration time.Duration) error { return nil }
func (m *mockCache) SetNX(ctx context.Context, key string, value any, expiration time.Duration) (bool, error) { return true, nil }
func (m *mockCache) Delete(ctx context.Context, key string) error                    { return nil }
func (m *mockCache) Exists(ctx context.Context, key string) (bool, error)            { return false, nil }
func (m *mockCache) TTL(ctx context.Context, key string) (time.Duration, error)      { return -2, nil }
func (m *mockCache) Expire(ctx context.Context, key string, expiration time.Duration) error { return nil }
func (m *mockCache) Incr(ctx context.Context, key string) (int64, error)            { return 1, nil }
func (m *mockCache) Decr(ctx context.Context, key string) (int64, error)            { return -1, nil }
func (m *mockCache) GetOrSet(ctx context.Context, key string, loader func() (any, error), expiration time.Duration) ([]byte, error) { return nil, cache.ErrKeyNotFound }
func (m *mockCache) GetDel(ctx context.Context, key string) ([]byte, error)         { return nil, cache.ErrKeyNotFound }
func (m *mockCache) GetObjectDel(ctx context.Context, key string, value any) error { return cache.ErrKeyNotFound }
func (m *mockCache) ScanAll(ctx context.Context, pattern string) ([]string, error)   { return nil, nil }
func (m *mockCache) Flush(ctx context.Context) error                                { return nil }
func (m *mockCache) Close() error                                                   { return nil }

// =============================================================================
// Context Helpers
// =============================================================================
// Context Helpers
// =============================================================================

func ctxWithUser(userID uint64) context.Context {
	return meta.NewClientCtx(context.Background(), &meta.RequestMetaData{
		Auth: meta.Auth{UserID: userID, UserName: "testuser"},
	})
}

func ctxWithRole(userID uint64, role string) context.Context {
	return meta.NewClientCtx(context.Background(), &meta.RequestMetaData{
		Auth: meta.Auth{UserID: userID, UserName: "testuser", Role: role},
	})
}

func testLogger() log.Logger { return log.DefaultLogger }

func setupArticleUseCase() (*ArticleUseCase, *mockArticleRepo, *mockTagRepo, *mockCategoryRepo, *mockEventBus) {
	ar := newMockArticleRepo()
	tr := newMockTagRepo()
	cr := newMockCategoryRepo()
	eb := newMockEventBus()
	c := newMockCache()
	uc := NewArticleUseCase(ar, tr, cr, eb, c, testLogger())
	return uc, ar, tr, cr, eb
}

func setupTagUseCase() (*TagUseCase, *mockTagRepo) {
	tr := newMockTagRepo()
	uc := NewTagUseCase(tr, testLogger())
	return uc, tr
}

func setupCategoryUseCase() (*CategoryUseCase, *mockCategoryRepo) {
	cr := newMockCategoryRepo()
	uc := NewCategoryUseCase(cr, testLogger())
	return uc, cr
}

func setupSiteUseCase() (*SiteUseCase, *mockSiteRepo) {
	sr := newMockSiteRepo()
	uc := NewSiteUseCase(sr, testLogger())
	return uc, sr
}
