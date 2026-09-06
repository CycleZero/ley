package proxy

// ===================== 标签/分类域测试基建（tag_category_test.go） =====================
//
// helpers_test.go 的 newTestEnv 只在 bufconn server 上注册 auth/article 两个服务，
// 且并行 wave 约定禁止改动共享基建（helpers_test.go 只读）——故本域测试自带
// fake TagService/CategoryService 与独立 bufconn 环境：拨号方式复用 dialBufconn
// （与生产同款 kratos metadata.Client() 中间件，元数据透传断言真实有效），
// handler 一律走生产实现 tag.go/category.go，测试只注册路由。
//
// 覆盖域：标签 3 个接口 + 分类 4 个接口（用例见 tag_test.go / category_test.go）。

import (
	"context"
	"sync"
	"testing"

	blogv1 "github.com/CycleZero/ley/api/blog/v1"
	"github.com/gin-gonic/gin"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/test/bufconn"
)

// ===================== fake TagService =====================

// fakeTagServer 假标签服务：实现 TagService 全部 3 个方法
// （固定回复 + 按调用顺序记录快照）。
type fakeTagServer struct {
	blogv1.UnimplementedTagServiceServer

	// err 非 nil 时各方法直接返回该错误（错误映射测试注入点）
	err error

	mu      sync.Mutex
	creates []tagCreateSnap // CreateTag 调用快照
	lists   []tagListSnap   // ListTags 调用快照
	deletes []tagDeleteSnap // DeleteTag 调用快照
}

type tagCreateSnap struct {
	md   metadata.MD // server 端收到的元数据
	name string      // 请求体绑定结果
}

type tagListSnap struct {
	md metadata.MD
}

type tagDeleteSnap struct {
	md metadata.MD
	id uint64 // 路径参数合并后的请求 Id
}

// CreateTag 记录请求并固定返回 id=11 的标签（计数非零，保证 JSON 不丢字段）。
func (f *fakeTagServer) CreateTag(ctx context.Context, in *blogv1.CreateTagRequest) (*blogv1.CreateTagReply, error) {
	if f.err != nil {
		return nil, f.err
	}
	f.mu.Lock()
	f.creates = append(f.creates, tagCreateSnap{md: recordIncomingMD(ctx), name: in.Name})
	f.mu.Unlock()
	return &blogv1.CreateTagReply{Tag: &blogv1.TagInfo{Id: 11, Name: in.Name, Slug: "go-lang", ArticleCount: 5}}, nil
}

// ListTags 固定返回两个标签。
func (f *fakeTagServer) ListTags(ctx context.Context, _ *blogv1.ListTagsRequest) (*blogv1.ListTagsReply, error) {
	if f.err != nil {
		return nil, f.err
	}
	f.mu.Lock()
	f.lists = append(f.lists, tagListSnap{md: recordIncomingMD(ctx)})
	f.mu.Unlock()
	return &blogv1.ListTagsReply{Tags: []*blogv1.TagInfo{
		{Id: 1, Name: "go", Slug: "go-lang", ArticleCount: 3},
		{Id: 2, Name: "gin", Slug: "gin-web", ArticleCount: 1},
	}}, nil
}

// DeleteTag 记录请求并返回空回复（删除类接口无业务数据）。
func (f *fakeTagServer) DeleteTag(ctx context.Context, in *blogv1.DeleteTagRequest) (*blogv1.DeleteTagReply, error) {
	if f.err != nil {
		return nil, f.err
	}
	f.mu.Lock()
	f.deletes = append(f.deletes, tagDeleteSnap{md: recordIncomingMD(ctx), id: in.Id})
	f.mu.Unlock()
	return &blogv1.DeleteTagReply{}, nil
}

// lastTagCreate 返回最近一次 CreateTag 快照（无调用时第二个返回值为 false）。
func (f *fakeTagServer) lastTagCreate() (tagCreateSnap, bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.creates) == 0 {
		return tagCreateSnap{}, false
	}
	return f.creates[len(f.creates)-1], true
}

// lastTagDelete 返回最近一次 DeleteTag 快照（无调用时第二个返回值为 false）。
func (f *fakeTagServer) lastTagDelete() (tagDeleteSnap, bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.deletes) == 0 {
		return tagDeleteSnap{}, false
	}
	return f.deletes[len(f.deletes)-1], true
}

// tagListCount 返回 ListTags 被调用次数（列表接口无请求字段，仅需计数断言）。
func (f *fakeTagServer) tagListCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.lists)
}

// ===================== fake CategoryService =====================

// fakeCategoryServer 假分类服务：实现 CategoryService 全部 4 个方法
// （固定回复 + 按调用顺序记录快照）。
type fakeCategoryServer struct {
	blogv1.UnimplementedCategoryServiceServer

	// err 非 nil 时各方法直接返回该错误（错误映射测试注入点）
	err error

	mu      sync.Mutex
	creates []catCreateSnap // CreateCategory 调用快照
	lists   []catListSnap   // ListCategories 调用快照
	updates []catUpdateSnap // UpdateCategory 调用快照
	deletes []catDeleteSnap // DeleteCategory 调用快照
}

type catCreateSnap struct {
	md          metadata.MD
	name        string // 请求体绑定结果
	slug        string
	description string
	parentID    uint64
	sortOrder   int32
}

type catListSnap struct {
	md metadata.MD
}

type catUpdateSnap struct {
	md          metadata.MD
	id          uint64 // 路径参数合并结果
	name        string // 请求体绑定结果
	slug        string
	description string
	parentID    uint64
	sortOrder   int32
}

type catDeleteSnap struct {
	md metadata.MD
	id uint64
}

// CreateCategory 记录请求并回显分类（id=9 固定；计数非零保证 JSON 不丢字段）。
func (f *fakeCategoryServer) CreateCategory(ctx context.Context, in *blogv1.CreateCategoryRequest) (*blogv1.CreateCategoryReply, error) {
	if f.err != nil {
		return nil, f.err
	}
	f.mu.Lock()
	f.creates = append(f.creates, catCreateSnap{
		md: recordIncomingMD(ctx), name: in.Name, slug: in.Slug,
		description: in.Description, parentID: in.ParentId, sortOrder: in.SortOrder,
	})
	f.mu.Unlock()
	return &blogv1.CreateCategoryReply{Category: &blogv1.CategoryInfo{
		Id: 9, Name: in.Name, Slug: in.Slug, Description: in.Description,
		ParentId: in.ParentId, SortOrder: in.SortOrder, ArticleCount: 2,
	}}, nil
}

// ListCategories 固定返回一棵含 children 子树的分类树（覆盖嵌套序列化契约）。
func (f *fakeCategoryServer) ListCategories(ctx context.Context, _ *blogv1.ListCategoriesRequest) (*blogv1.ListCategoriesReply, error) {
	if f.err != nil {
		return nil, f.err
	}
	f.mu.Lock()
	f.lists = append(f.lists, catListSnap{md: recordIncomingMD(ctx)})
	f.mu.Unlock()
	return &blogv1.ListCategoriesReply{Categories: []*blogv1.CategoryInfo{
		{
			Id: 1, Name: "后端", Slug: "backend", Description: "后端技术",
			SortOrder: 1, ArticleCount: 4,
			Children: []*blogv1.CategoryInfo{{Id: 2, Name: "Go", Slug: "go", SortOrder: 1, ArticleCount: 2}},
		},
	}}, nil
}

// UpdateCategory 记录请求并回显分类（id 取路径参数合并结果）。
func (f *fakeCategoryServer) UpdateCategory(ctx context.Context, in *blogv1.UpdateCategoryRequest) (*blogv1.UpdateCategoryReply, error) {
	if f.err != nil {
		return nil, f.err
	}
	f.mu.Lock()
	f.updates = append(f.updates, catUpdateSnap{
		md: recordIncomingMD(ctx), id: in.Id, name: in.Name, slug: in.Slug,
		description: in.Description, parentID: in.ParentId, sortOrder: in.SortOrder,
	})
	f.mu.Unlock()
	return &blogv1.UpdateCategoryReply{Category: &blogv1.CategoryInfo{
		Id: in.Id, Name: in.Name, Slug: in.Slug, Description: in.Description,
		ParentId: in.ParentId, SortOrder: in.SortOrder,
	}}, nil
}

// DeleteCategory 记录请求并返回空回复。
func (f *fakeCategoryServer) DeleteCategory(ctx context.Context, in *blogv1.DeleteCategoryRequest) (*blogv1.DeleteCategoryReply, error) {
	if f.err != nil {
		return nil, f.err
	}
	f.mu.Lock()
	f.deletes = append(f.deletes, catDeleteSnap{md: recordIncomingMD(ctx), id: in.Id})
	f.mu.Unlock()
	return &blogv1.DeleteCategoryReply{}, nil
}

// lastCatCreate / lastCatUpdate / lastCatDelete 返回最近一次对应调用快照。
func (f *fakeCategoryServer) lastCatCreate() (catCreateSnap, bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.creates) == 0 {
		return catCreateSnap{}, false
	}
	return f.creates[len(f.creates)-1], true
}

func (f *fakeCategoryServer) lastCatUpdate() (catUpdateSnap, bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.updates) == 0 {
		return catUpdateSnap{}, false
	}
	return f.updates[len(f.updates)-1], true
}

func (f *fakeCategoryServer) lastCatDelete() (catDeleteSnap, bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.deletes) == 0 {
		return catDeleteSnap{}, false
	}
	return f.deletes[len(f.deletes)-1], true
}

// catListCount 返回 ListCategories 被调用次数。
func (f *fakeCategoryServer) catListCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.lists)
}

// ===================== 本域测试环境装配 =====================

// tagCategoryEnv 标签/分类域单测环境：独立 bufconn server 上注册 fakeTag/fakeCat，
// 复用 dialBufconn 拨号（与生产同款中间件），由连接装配仅含 Tag/Category 的 ServiceHub。
type tagCategoryEnv struct {
	t       *testing.T
	fakeTag *fakeTagServer
	fakeCat *fakeCategoryServer
	hub     *ServiceHub
	engine  *gin.Engine
}

// newTagCategoryEnv 装配一套标签/分类域测试环境（清理由 t.Cleanup 完成）。
func newTagCategoryEnv(t *testing.T) *tagCategoryEnv {
	t.Helper()
	gin.SetMode(gin.TestMode)

	lis := bufconn.Listen(1 << 20)
	fakeTag := &fakeTagServer{}
	fakeCat := &fakeCategoryServer{}
	srv := grpc.NewServer()
	blogv1.RegisterTagServiceServer(srv, fakeTag)
	blogv1.RegisterCategoryServiceServer(srv, fakeCat)
	go func() { _ = srv.Serve(lis) }()
	t.Cleanup(func() { srv.Stop() })

	blogConn := dialBufconn(t, lis, "tag-category")
	hub := &ServiceHub{
		Tag:      blogv1.NewTagServiceClient(blogConn),
		Category: blogv1.NewCategoryServiceClient(blogConn),
	}

	return &tagCategoryEnv{
		t: t, fakeTag: fakeTag, fakeCat: fakeCat,
		hub:    hub,
		engine: gin.New(),
	}
}

// registerTagRoutes 挂载标签 3 个接口路由（挂 withAuthMeta 模拟后台鉴权中间件，
// 顺带让 DeleteTag 测试可断言元数据透传；handler 为生产实现 NewTagHandler）。
func registerTagRoutes(env *tagCategoryEnv) {
	tagH := NewTagHandler(env.hub)
	env.engine.POST("/api/v1/tags", withAuthMeta(1, "root", "admin"), tagH.CreateTag)
	env.engine.GET("/api/v1/tags", tagH.ListTags)
	env.engine.DELETE("/api/v1/tags/:id", withAuthMeta(1, "root", "admin"), tagH.DeleteTag)
}

// registerCategoryRoutes 挂载分类 4 个接口路由（同上，鉴权元数据仅后台写操作）。
func registerCategoryRoutes(env *tagCategoryEnv) {
	catH := NewCategoryHandler(env.hub)
	env.engine.POST("/api/v1/categories", withAuthMeta(1, "root", "admin"), catH.CreateCategory)
	env.engine.GET("/api/v1/categories", catH.ListCategories)
	env.engine.PUT("/api/v1/categories/:id", withAuthMeta(1, "root", "admin"), catH.UpdateCategory)
	env.engine.DELETE("/api/v1/categories/:id", withAuthMeta(1, "root", "admin"), catH.DeleteCategory)
}
