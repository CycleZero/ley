package proxy

// ===================== 文章域 handler 测试（article_test.go） =====================
//
// 覆盖 article.go 的 11 个 handler：
//  1. body 绑定（Create/Update：snake_case 输入 → proto 请求字段）；
//  2. 路径参数合并（Update 的 id+body 合并、ID 操作表：Publish/Archive/Like/
//     Unlike/View/Delete 的 :id → req.Id）；
//  3. GET query 绑定（Get 的 identifier 字符串、List 全字段 + tags 两种形态、
//     Search，非法数值 400 短路）；
//  4. 错误映射（kratos NotFound → 404 信封，且不触发 fake 记录）；
//  5. 信封契约（成功 data 为 protojson UseProtoNames snake_case、空回复 {}）。
//
// fake 说明：helpers_test.go 的 fakeBlogServer 仅实现了 DeleteArticle，且其状态
// 字段声明在该文件中、无法在不编辑它（任务约束只读）的前提下扩展记录能力——
// 故本文件自带完整实现 ArticleService 全部 11 方法的 fakeArticleServer 与独立
// bufconn 环境（newArticleTestEnv），复用 helpers_test.go 的无状态辅助
// （dialBufconn/withAuthMeta/doRequest/decodeEnvelope/mdVal）。

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"testing"

	blogv1 "github.com/CycleZero/ley/api/blog/v1"
	pkgmeta "github.com/CycleZero/ley/pkg/meta"
	"github.com/gin-gonic/gin"
	kerrors "github.com/go-kratos/kratos/v2/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/test/bufconn"
	"google.golang.org/protobuf/proto"
)

// ===================== fakeArticleServer：文章方法完整 fake =====================

// fakeArticleServer 完整实现 ArticleService 的 11 个方法：固定回复 + 按调用顺序
// 记录快照（请求消息指针 + 透传元数据），供各测试断言路径/query/body 绑定结果。
type fakeArticleServer struct {
	blogv1.UnimplementedArticleServiceServer

	getErr error // 非 nil 时 GetArticle 直接返回该错误（错误映射测试注入点）

	mu    sync.Mutex
	calls []articleCall // 全部方法调用快照（按调用顺序）
}

// articleCall 一次文章方法调用的 server 端快照。
type articleCall struct {
	method string        // 方法名（与 lastCall 入参对应）
	md     metadata.MD   // server 端收到的元数据（断言 x-md-global-* 透传）
	req    proto.Message // 收到的请求消息（断言时按 method 断言到具体类型）
}

// record 记录一次调用快照（错误注入路径不记录——调用未到达 server 业务逻辑）。
func (f *fakeArticleServer) record(method string, ctx context.Context, req proto.Message) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls = append(f.calls, articleCall{method: method, md: recordIncomingMD(ctx), req: req})
}

// lastCall 返回 method 最近一次调用快照（无调用时第二个返回值为 false）。
func (f *fakeArticleServer) lastCall(method string) (articleCall, bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for i := len(f.calls) - 1; i >= 0; i-- {
		if c := f.calls[i]; c.method == method {
			return c, true
		}
	}
	return articleCall{}, false
}

// fixedArticle 构造固定回复用文章样例：ID/标题回显请求，其余为可断言
// snake_case 序列化的稳定字段（cover_image/category_id/created_at 等）。
func fixedArticle(id uint64, title string) *blogv1.ArticleInfo {
	return &blogv1.ArticleInfo{
		Id: id, Title: title, Slug: "sample-slug", Status: "draft",
		CoverImage: "/covers/sample.png", CategoryId: 2, CreatedAt: "2026-09-06T10:00:00Z",
	}
}

func (f *fakeArticleServer) CreateArticle(ctx context.Context, in *blogv1.CreateArticleRequest) (*blogv1.CreateArticleReply, error) {
	f.record("CreateArticle", ctx, in)
	return &blogv1.CreateArticleReply{Article: fixedArticle(101, in.Title)}, nil
}

func (f *fakeArticleServer) UpdateArticle(ctx context.Context, in *blogv1.UpdateArticleRequest) (*blogv1.UpdateArticleReply, error) {
	f.record("UpdateArticle", ctx, in)
	return &blogv1.UpdateArticleReply{Article: fixedArticle(in.Id, in.Title)}, nil
}

func (f *fakeArticleServer) DeleteArticle(ctx context.Context, in *blogv1.DeleteArticleRequest) (*blogv1.DeleteArticleReply, error) {
	f.record("DeleteArticle", ctx, in)
	return &blogv1.DeleteArticleReply{}, nil
}

func (f *fakeArticleServer) GetArticle(ctx context.Context, in *blogv1.GetArticleRequest) (*blogv1.GetArticleReply, error) {
	if f.getErr != nil {
		return nil, f.getErr
	}
	f.record("GetArticle", ctx, in)
	a := fixedArticle(1, "示例文章")
	a.Slug = in.Identifier // 回显 identifier，证明字符串路径参数原样透传
	return &blogv1.GetArticleReply{Article: a}, nil
}

func (f *fakeArticleServer) ListArticles(ctx context.Context, in *blogv1.ListArticlesRequest) (*blogv1.ListArticlesReply, error) {
	f.record("ListArticles", ctx, in)
	return &blogv1.ListArticlesReply{
		Articles: []*blogv1.ArticleInfo{{Id: 1, Title: "甲"}, {Id: 2, Title: "乙"}},
		Total:    2,
		Page:     in.Page, PageSize: in.PageSize, // 回显分页，证明 query 绑定生效
	}, nil
}

func (f *fakeArticleServer) SearchArticles(ctx context.Context, in *blogv1.SearchArticlesRequest) (*blogv1.SearchArticlesReply, error) {
	f.record("SearchArticles", ctx, in)
	return &blogv1.SearchArticlesReply{
		Articles: []*blogv1.ArticleInfo{{Id: 9, Title: "命中"}},
		Total:    1,
	}, nil
}

func (f *fakeArticleServer) PublishArticle(ctx context.Context, in *blogv1.PublishArticleRequest) (*blogv1.PublishArticleReply, error) {
	f.record("PublishArticle", ctx, in)
	return &blogv1.PublishArticleReply{Article: &blogv1.ArticleInfo{Id: in.Id, Status: "published"}}, nil
}

func (f *fakeArticleServer) ArchiveArticle(ctx context.Context, in *blogv1.ArchiveArticleRequest) (*blogv1.ArchiveArticleReply, error) {
	f.record("ArchiveArticle", ctx, in)
	return &blogv1.ArchiveArticleReply{Article: &blogv1.ArticleInfo{Id: in.Id, Status: "archived"}}, nil
}

func (f *fakeArticleServer) LikeArticle(ctx context.Context, in *blogv1.LikeArticleRequest) (*blogv1.LikeArticleReply, error) {
	f.record("LikeArticle", ctx, in)
	return &blogv1.LikeArticleReply{}, nil
}

func (f *fakeArticleServer) UnlikeArticle(ctx context.Context, in *blogv1.UnlikeArticleRequest) (*blogv1.UnlikeArticleReply, error) {
	f.record("UnlikeArticle", ctx, in)
	return &blogv1.UnlikeArticleReply{}, nil
}

func (f *fakeArticleServer) ViewArticle(ctx context.Context, in *blogv1.ViewArticleRequest) (*blogv1.ViewArticleReply, error) {
	f.record("ViewArticle", ctx, in)
	return &blogv1.ViewArticleReply{Counted: true}, nil
}

// ===================== 文章域测试环境 =====================

// articleTestEnv 文章域独立测试环境：bufconn fake（完整 11 方法）+ 与生产同款
// 拨号（dialBufconn 复用：kratos metadata.Client() 中间件保证透传断言真实）。
type articleTestEnv struct {
	fake   *fakeArticleServer
	hub    *ServiceHub
	engine *gin.Engine
}

// newArticleTestEnv 装配文章域环境（本文件独享，避免为扩展 fake 触碰只读的
// helpers_test.go；auth/tag 等域未用到，hub 仅装配 Article client）。
func newArticleTestEnv(t *testing.T) *articleTestEnv {
	t.Helper()
	gin.SetMode(gin.TestMode)

	lis := bufconn.Listen(1 << 20)
	fake := &fakeArticleServer{}
	srv := grpc.NewServer()
	blogv1.RegisterArticleServiceServer(srv, fake)
	go func() { _ = srv.Serve(lis) }()
	t.Cleanup(func() { srv.Stop() })

	conn := dialBufconn(t, lis, "article")
	return &articleTestEnv{
		fake:   fake,
		hub:    &ServiceHub{Article: blogv1.NewArticleServiceClient(conn)},
		engine: gin.New(),
	}
}

// ===================== 1. body 绑定 + 成功信封（Create） =====================

// TestArticleHandler_CreateArticle：Given 认证用户（uid=7）携带完整 snake_case
// body；When 经 CreateArticle handler 转发；Then 200 成功信封、data 为
// snake_case 原始 JSON（无 camelCase 键）、fake 收到全部绑定字段且元数据透传。
func TestArticleHandler_CreateArticle(t *testing.T) {
	// Given
	env := newArticleTestEnv(t)
	h := NewArticleHandler(env.hub)
	env.engine.POST("/api/v1/articles", withAuthMeta(7, "author1", "author"), h.CreateArticle)
	// When
	w := doRequest(t, env.engine, http.MethodPost, "/api/v1/articles",
		`{"title":"你好世界","content":"# 正文","excerpt":"摘要","cover_image":"/covers/a.png","category_id":2,"tag_names":["Go","Gin"],"status":"draft"}`)
	// Then：信封 + snake_case data（含嵌套 ArticleInfo 字段）
	if w.Code != http.StatusOK {
		t.Fatalf("HTTP 状态 = %d, 期望 %d，body=%s", w.Code, http.StatusOK, w.Body.String())
	}
	resp := decodeEnvelope(t, w)
	if resp.Code != 0 || resp.Msg != "ok" {
		t.Errorf("信封 = {code:%d msg:%q}, 期望 {code:0 msg:ok}", resp.Code, resp.Msg)
	}
	raw := string(resp.Data)
	// 注意：protojson 将 64 位整数编码为 JSON 字符串（proto3 JSON 规范），
	// 故 uint64 id/category_id 断言带引号形态
	for _, want := range []string{`"id":"101"`, `"title":"你好世界"`, `"cover_image":"/covers/sample.png"`,
		`"category_id":"2"`, `"created_at":"2026-09-06T10:00:00Z"`} {
		if !strings.Contains(raw, want) {
			t.Errorf("data 缺少片段 %s，data=%s", want, raw)
		}
	}
	for _, forbid := range []string{`"coverImage"`, `"createdAt"`, `"categoryId"`} {
		if strings.Contains(raw, forbid) {
			t.Errorf("data 出现 camelCase 键 %s（期望 UseProtoNames），data=%s", forbid, raw)
		}
	}
	// Then：fake 收到绑定后的请求字段
	snap, ok := env.fake.lastCall("CreateArticle")
	if !ok {
		t.Fatal("fake 未收到 CreateArticle 调用")
	}
	req := snap.req.(*blogv1.CreateArticleRequest)
	if req.Title != "你好世界" || req.Content != "# 正文" || req.Excerpt != "摘要" ||
		req.CoverImage != "/covers/a.png" || req.Status != "draft" || req.CategoryId != 2 {
		t.Errorf("fake 收到字段 = %+v, 与 body 不一致", req)
	}
	if len(req.TagNames) != 2 || req.TagNames[0] != "Go" || req.TagNames[1] != "Gin" {
		t.Errorf("fake 收到 tag_names = %v, 期望 [Go Gin]", req.TagNames)
	}
	// Then：认证元数据透传（uid=7 到达 server 端）
	if got := mdVal(t, snap.md, pkgmeta.AuthUserIDKey); got != "7" {
		t.Errorf("x-md-global-auth-user-id = %q, 期望 %q", got, "7")
	}
}

// ===================== 2. 路径参数合并 =====================

// TestArticleHandler_UpdateArticle_PathIDAndBody：Given PUT /articles/:id=42 +
// body；When UpdateArticle handler 先合并路径 id 再 callProto 绑定 body；
// Then fake 收到 Id=42 且 body 字段一并生效（id 不被 body 覆盖）。
func TestArticleHandler_UpdateArticle_PathIDAndBody(t *testing.T) {
	// Given
	env := newArticleTestEnv(t)
	h := NewArticleHandler(env.hub)
	env.engine.PUT("/api/v1/articles/:id", h.UpdateArticle)
	// When
	w := doRequest(t, env.engine, http.MethodPut, "/api/v1/articles/42",
		`{"title":"新标题","content":"新正文","category_id":3,"tag_names":["T1"]}`)
	// Then
	if w.Code != http.StatusOK {
		t.Fatalf("HTTP 状态 = %d, 期望 %d，body=%s", w.Code, http.StatusOK, w.Body.String())
	}
	resp := decodeEnvelope(t, w)
	if raw := string(resp.Data); !strings.Contains(raw, `"id":"42"`) {
		t.Errorf("回复 article 应为 id=42（路径参数合并失败），data=%s", raw)
	}
	snap, ok := env.fake.lastCall("UpdateArticle")
	if !ok {
		t.Fatal("fake 未收到 UpdateArticle 调用")
	}
	req := snap.req.(*blogv1.UpdateArticleRequest)
	if req.Id != 42 {
		t.Errorf("fake 收到 Id = %d, 期望 42（路径参数未合并）", req.Id)
	}
	if req.Title != "新标题" || req.Content != "新正文" || req.CategoryId != 3 {
		t.Errorf("fake 收到字段 = %+v, body 字段丢失", req)
	}
	if len(req.TagNames) != 1 || req.TagNames[0] != "T1" {
		t.Errorf("fake 收到 tag_names = %v, 期望 [T1]", req.TagNames)
	}
}

// ===================== 3. GET：identifier / query 绑定 =====================

// TestArticleHandler_GetArticle_IdentifierString：Given GET /articles/{identifier}
// 携带 Slug（字符串）；When BindPathString 原样透传；Then fake 收到该字符串，
// 且 reply 回显 slug（snake_case data）。
func TestArticleHandler_GetArticle_IdentifierString(t *testing.T) {
	// Given
	env := newArticleTestEnv(t)
	h := NewArticleHandler(env.hub)
	env.engine.GET("/api/v1/articles/:identifier", h.GetArticle)
	// When
	w := doRequest(t, env.engine, http.MethodGet, "/api/v1/articles/my-first-post", "")
	// Then
	if w.Code != http.StatusOK {
		t.Fatalf("HTTP 状态 = %d, 期望 %d，body=%s", w.Code, http.StatusOK, w.Body.String())
	}
	resp := decodeEnvelope(t, w)
	if raw := string(resp.Data); !strings.Contains(raw, `"slug":"my-first-post"`) {
		t.Errorf("identifier 未原样透传（reply 应回显 slug），data=%s", raw)
	}
	snap, ok := env.fake.lastCall("GetArticle")
	if !ok {
		t.Fatal("fake 未收到 GetArticle 调用")
	}
	if got := snap.req.(*blogv1.GetArticleRequest).Identifier; got != "my-first-post" {
		t.Errorf("fake 收到 identifier = %q, 期望 my-first-post", got)
	}
}

// TestArticleHandler_ListArticles_QueryBind：Given 完整过滤/分页 query（tags 用
// 重复键形态 ?tags=go&tags=gin）；When ListArticles 经 query 显式逐字段绑定；
// Then fake 收到全部 query 字段（含 repeated tags 展开），回复回显分页。
func TestArticleHandler_ListArticles_QueryBind(t *testing.T) {
	// Given
	env := newArticleTestEnv(t)
	h := NewArticleHandler(env.hub)
	env.engine.GET("/api/v1/articles", h.ListArticles)
	// When
	w := doRequest(t, env.engine, http.MethodGet,
		"/api/v1/articles?status=published&category_id=2&tags=go&tags=gin&author_id=7&sort_by=created_at&sort_order=desc&page=1&page_size=10", "")
	// Then
	if w.Code != http.StatusOK {
		t.Fatalf("HTTP 状态 = %d, 期望 %d，body=%s", w.Code, http.StatusOK, w.Body.String())
	}
	resp := decodeEnvelope(t, w)
	raw := string(resp.Data)
	// total 为 int64 → protojson 字符串形态；page/page_size 为 int32 → 数字形态
	for _, want := range []string{`"total":"2"`, `"page":1`, `"page_size":10`} {
		if !strings.Contains(raw, want) {
			t.Errorf("data 缺少分页回显 %s，data=%s", want, raw)
		}
	}
	snap, ok := env.fake.lastCall("ListArticles")
	if !ok {
		t.Fatal("fake 未收到 ListArticles 调用")
	}
	req := snap.req.(*blogv1.ListArticlesRequest)
	if req.Status != "published" || req.CategoryId != 2 || req.AuthorId != 7 ||
		req.SortBy != "created_at" || req.SortOrder != "desc" || req.Page != 1 || req.PageSize != 10 {
		t.Errorf("fake 收到 query 绑定 = %+v, 与 URL query 不一致", req)
	}
	if len(req.Tags) != 2 || req.Tags[0] != "go" || req.Tags[1] != "gin" {
		t.Errorf("fake 收到 tags = %v, 期望 [go gin]（重复键形态展开失败）", req.Tags)
	}
}

// TestArticleHandler_ListArticles_TagsCSV：Given tags 用逗号分隔形态
// （kratos HTTP client binding.EncodeURL 默认编码）；When 绑定；
// Then 同样展开为 repeated tags。
func TestArticleHandler_ListArticles_TagsCSV(t *testing.T) {
	// Given
	env := newArticleTestEnv(t)
	h := NewArticleHandler(env.hub)
	env.engine.GET("/api/v1/articles", h.ListArticles)
	// When
	w := doRequest(t, env.engine, http.MethodGet, "/api/v1/articles?tags=go,gin", "")
	// Then
	if w.Code != http.StatusOK {
		t.Fatalf("HTTP 状态 = %d, 期望 %d，body=%s", w.Code, http.StatusOK, w.Body.String())
	}
	snap, ok := env.fake.lastCall("ListArticles")
	if !ok {
		t.Fatal("fake 未收到 ListArticles 调用")
	}
	if got := snap.req.(*blogv1.ListArticlesRequest).Tags; len(got) != 2 || got[0] != "go" || got[1] != "gin" {
		t.Errorf("fake 收到 tags = %v, 期望 [go gin]（逗号分隔形态展开失败）", got)
	}
}

// TestArticleHandler_ListArticles_BadQueryParam：Given category_id 非整数；
// When bindQueryUint64 解析失败；Then 400 信封并短路——不触发下游调用。
func TestArticleHandler_ListArticles_BadQueryParam(t *testing.T) {
	// Given
	env := newArticleTestEnv(t)
	h := NewArticleHandler(env.hub)
	env.engine.GET("/api/v1/articles", h.ListArticles)
	// When
	w := doRequest(t, env.engine, http.MethodGet, "/api/v1/articles?category_id=abc&page=1", "")
	// Then
	if w.Code != http.StatusBadRequest {
		t.Fatalf("HTTP 状态 = %d, 期望 %d，body=%s", w.Code, http.StatusBadRequest, w.Body.String())
	}
	resp := decodeEnvelope(t, w)
	if resp.Code != http.StatusBadRequest || !strings.Contains(resp.Msg, "必须为无符号整数") {
		t.Errorf("信封 = {code:%d msg:%q}, 期望 {code:400 msg:含'必须为无符号整数'}", resp.Code, resp.Msg)
	}
	if _, ok := env.fake.lastCall("ListArticles"); ok {
		t.Error("非法 query 不应触发下游 ListArticles 调用")
	}
}

// TestArticleHandler_SearchArticles_QueryBind：Given keyword + 分页 query；
// When SearchArticles 绑定；Then fake 收到 keyword/page/page_size。
func TestArticleHandler_SearchArticles_QueryBind(t *testing.T) {
	// Given
	env := newArticleTestEnv(t)
	h := NewArticleHandler(env.hub)
	env.engine.GET("/api/v1/articles/search", h.SearchArticles)
	// When
	w := doRequest(t, env.engine, http.MethodGet, "/api/v1/articles/search?keyword=hello&page=2&page_size=5", "")
	// Then
	if w.Code != http.StatusOK {
		t.Fatalf("HTTP 状态 = %d, 期望 %d，body=%s", w.Code, http.StatusOK, w.Body.String())
	}
	resp := decodeEnvelope(t, w)
	if raw := string(resp.Data); !strings.Contains(raw, `"total":"1"`) || !strings.Contains(raw, `"title":"命中"`) {
		t.Errorf("搜索回复异常，data=%s", raw)
	}
	snap, ok := env.fake.lastCall("SearchArticles")
	if !ok {
		t.Fatal("fake 未收到 SearchArticles 调用")
	}
	req := snap.req.(*blogv1.SearchArticlesRequest)
	if req.Keyword != "hello" || req.Page != 2 || req.PageSize != 5 {
		t.Errorf("fake 收到 query 绑定 = %+v, 与 URL query 不一致", req)
	}
}

// ===================== 4. 错误映射 =====================

// TestArticleHandler_GetArticle_NotFoundError：Given fake 返回 kratos NotFound
// （真实微服务错误形态）；When 转发失败；Then 404 信封 + 业务消息透出，
// 且错误路径不记录调用（证明短路发生在 gRPC 返回后）。
func TestArticleHandler_GetArticle_NotFoundError(t *testing.T) {
	// Given
	env := newArticleTestEnv(t)
	env.fake.getErr = kerrors.NotFound("ARTICLE_NOT_FOUND", "文章不存在")
	h := NewArticleHandler(env.hub)
	env.engine.GET("/api/v1/articles/:identifier", h.GetArticle)
	// When
	w := doRequest(t, env.engine, http.MethodGet, "/api/v1/articles/nope", "")
	// Then
	if w.Code != http.StatusNotFound {
		t.Fatalf("HTTP 状态 = %d, 期望 %d，body=%s", w.Code, http.StatusNotFound, w.Body.String())
	}
	resp := decodeEnvelope(t, w)
	if resp.Code != http.StatusNotFound || resp.Msg != "文章不存在" {
		t.Errorf("信封 = {code:%d msg:%q}, 期望 {code:404 msg:文章不存在}", resp.Code, resp.Msg)
	}
	if resp.Data != nil && string(resp.Data) != "null" {
		t.Errorf("失败信封 data = %s, 期望 null", resp.Data)
	}
	if _, ok := env.fake.lastCall("GetArticle"); ok {
		t.Error("错误路径不应记录 GetArticle 调用")
	}
}

// ===================== 5. ID 操作类 handler（表驱动） =====================

// TestArticleHandler_IDActionHandlers：Given 6 个纯路径 ID 操作
// （Publish/Archive/Like/Unlike/View/Delete）；When 各自 handler 合并 :id 转发；
// Then fake 收到对应调用且 Id 正确，回复信封 data 符合方法语义
// （publish/archive 带状态、view 带 counted、like/unlike/delete 为空 {}）。
func TestArticleHandler_IDActionHandlers(t *testing.T) {
	// 6 个 ID 操作：suffix 为 /api/v1/articles 后的完整路径（含 id）
	cases := []struct {
		name    string
		method  string
		suffix  string // 动作路径段（"" 为 DELETE 纯 id 路由）
		rpc     string
		id      uint64
		wantRaw string // 期望的信封 data 片段（Contains 断言）
	}{
		{"publish", http.MethodPost, "/publish", "PublishArticle", 11, `"status":"published"`},
		{"archive", http.MethodPost, "/archive", "ArchiveArticle", 12, `"status":"archived"`},
		{"like", http.MethodPost, "/like", "LikeArticle", 13, `{}`},
		{"unlike", http.MethodDelete, "/like", "UnlikeArticle", 14, `{}`},
		{"view", http.MethodPost, "/view", "ViewArticle", 15, `"counted":true`},
		{"delete", http.MethodDelete, "", "DeleteArticle", 16, `{}`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// Given：每个子测试独立环境，仅注册自身路由。
			// 路由必须用 :id 通配段（gin 经 c.Param 取路径参数），请求 URL 携带具体 id
			env := newArticleTestEnv(t)
			h := NewArticleHandler(env.hub)
			handler := map[string]func(c *gin.Context){
				"PublishArticle": h.PublishArticle,
				"ArchiveArticle": h.ArchiveArticle,
				"LikeArticle":    h.LikeArticle,
				"UnlikeArticle":  h.UnlikeArticle,
				"ViewArticle":    h.ViewArticle,
				"DeleteArticle":  h.DeleteArticle,
			}[tc.rpc]
			env.engine.Handle(tc.method, "/api/v1/articles/:id"+tc.suffix, handler)
			url := "/api/v1/articles/" + strconv.FormatUint(tc.id, 10) + tc.suffix
			// When
			w := doRequest(t, env.engine, tc.method, url, "")
			// Then
			if w.Code != http.StatusOK {
				t.Fatalf("HTTP 状态 = %d, 期望 %d，body=%s", w.Code, http.StatusOK, w.Body.String())
			}
			resp := decodeEnvelope(t, w)
			if raw := string(resp.Data); !strings.Contains(raw, tc.wantRaw) {
				t.Errorf("data = %s, 期望含 %s", raw, tc.wantRaw)
			}
			snap, ok := env.fake.lastCall(tc.rpc)
			if !ok {
				t.Fatalf("fake 未收到 %s 调用", tc.rpc)
			}
			if got := idOf(t, snap); got != tc.id {
				t.Errorf("fake 收到 Id = %d, 期望 %d（路径参数合并失败）", got, tc.id)
			}
		})
	}
}

// idOf 从快照请求中取出 Id（ID 操作类请求均携带 uint64 Id，类型由调用方保证）。
func idOf(t *testing.T, snap articleCall) uint64 {
	t.Helper()
	switch req := snap.req.(type) {
	case *blogv1.PublishArticleRequest:
		return req.Id
	case *blogv1.ArchiveArticleRequest:
		return req.Id
	case *blogv1.LikeArticleRequest:
		return req.Id
	case *blogv1.UnlikeArticleRequest:
		return req.Id
	case *blogv1.ViewArticleRequest:
		return req.Id
	case *blogv1.DeleteArticleRequest:
		return req.Id
	}
	t.Fatalf("快照请求类型 %T 无 Id 字段", snap.req)
	return 0
}
