package main

// ===================== E2E fake gRPC 服务（e2e_fakes_test.go） =====================
//
// 端到端测试专用的 bufconn fake server：与 domain/proxy 包 helpers_test.go 的
// fake 同构但独立成文——本文件位于 package main（app/entry 根），无法复用 proxy
// 测试包内未导出的 fake（跨包需复制模式，见 helpers_test.go 既有约定）。
//
// 覆盖 E2E 场景所需的代表性 RPC（不必全 39 个 handler——注册表正确性已由
// internal/router/routes_test.go 契约测试冻结）：
//
//	auth：Login（匿名登录，信封契约）、GetProfile（AUTH /users/me 身份透传）；
//	blog：TagService.ListTags（PUBLIC 匿名语义）、ArticleService.ListArticles
//	      （PUBLIC 带 token 身份透传）、SiteService.UpdateSiteConfig（ADMIN 越权/放行）。
//
// 每个 fake 记录 server 端收到的 incoming metadata（x-md-global-* 透传断言）与
// 关键请求字段；其余方法由嵌入的 Unimplemented* 兜底（返回 Unimplemented，
// 不会在 E2E 中触发）。

import (
	"context"
	"sync"
	"testing"

	authv1 "github.com/CycleZero/ley/api/auth/v1"
	blogv1 "github.com/CycleZero/ley/api/blog/v1"
	commonv1 "github.com/CycleZero/ley/api/common/v1"
	"google.golang.org/grpc/metadata"
)

// ===================== 通用快照 =====================

// mdSnap 一次 gRPC 调用的 server 端元数据快照（断言 x-md-global-* 透传）。
type mdSnap struct {
	md metadata.MD
}

// recordIncomingMD 提取 server 端收到的 incoming metadata 副本。
func recordIncomingMD(ctx context.Context) metadata.MD {
	md, _ := metadata.FromIncomingContext(ctx)
	return md.Copy()
}

// ===================== fake auth 服务 =====================

// e2eFakeAuth 假 auth 服务：实现 Login/GetProfile（固定响应 + 记录元数据），
// 其余方法由 UnimplementedAuthServiceServer 兜底。
type e2eFakeAuth struct {
	authv1.UnimplementedAuthServiceServer

	mu       sync.Mutex
	logins   []mdSnap // Login 调用快照（按序）
	profiles []mdSnap // GetProfile 调用快照（按序）
}

// Login 固定返回成功回复：user alice + 固定 token_pair（snake_case 契约断言用）。
func (f *e2eFakeAuth) Login(ctx context.Context, _ *authv1.LoginRequest) (*authv1.LoginReply, error) {
	f.mu.Lock()
	f.logins = append(f.logins, mdSnap{md: recordIncomingMD(ctx)})
	f.mu.Unlock()
	return &authv1.LoginReply{
		User: &commonv1.UserInfo{Id: 1, Username: "alice", Email: "alice@example.com", Role: "reader"},
		TokenPair: &commonv1.TokenPair{
			AccessToken: "e2e-access-token-1", RefreshToken: "e2e-refresh-token-1", ExpiresIn: 900,
		},
	}, nil
}

// GetProfile 固定返回当前用户（身份不随请求体传递：entry 经 x-md-global-* 透传，
// server 端快照即透传证据）。
func (f *e2eFakeAuth) GetProfile(ctx context.Context, _ *authv1.GetProfileRequest) (*authv1.GetProfileReply, error) {
	f.mu.Lock()
	f.profiles = append(f.profiles, mdSnap{md: recordIncomingMD(ctx)})
	f.mu.Unlock()
	return &authv1.GetProfileReply{
		User: &commonv1.UserInfo{Id: 42, Username: "tester", Email: "tester@example.com", Role: "reader"},
	}, nil
}

// profileCount 返回 GetProfile 调用次数（0 即下游未被调用——中间件前置拦截断言）。
func (f *e2eFakeAuth) profileCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.profiles)
}

// lastLogin 返回最近一次 Login 快照（无调用时第二个返回值为 false）。
func (f *e2eFakeAuth) lastLogin() (mdSnap, bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.logins) == 0 {
		return mdSnap{}, false
	}
	return f.logins[len(f.logins)-1], true
}

// lastProfile 返回最近一次 GetProfile 快照（无调用时第二个返回值为 false）。
func (f *e2eFakeAuth) lastProfile() (mdSnap, bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.profiles) == 0 {
		return mdSnap{}, false
	}
	return f.profiles[len(f.profiles)-1], true
}

// ===================== fake blog 服务 =====================

// siteUpdateSnap 一次 UpdateSiteConfig 调用的 server 端快照（元数据 + 请求标题）。
type siteUpdateSnap struct {
	md    metadata.MD
	title string // 请求体绑定的 config.site_title
}

// e2eFakeBlog 假 blog 服务：一个结构体实现 Tag/Article/Site 三个 ServiceServer
// （注册时分别 Register*ServiceServer），E2E 覆盖的代表性方法各自记录快照。
type e2eFakeBlog struct {
	blogv1.UnimplementedArticleServiceServer
	blogv1.UnimplementedTagServiceServer
	blogv1.UnimplementedSiteServiceServer

	mu           sync.Mutex
	tagLists     []mdSnap         // ListTags 快照
	articleLists []mdSnap         // ListArticles 快照
	siteUpdates  []siteUpdateSnap // UpdateSiteConfig 快照（按序）
}

// ListTags 返回固定标签列表（公开接口；供匿名语义断言）。
func (f *e2eFakeBlog) ListTags(ctx context.Context, _ *blogv1.ListTagsRequest) (*blogv1.ListTagsReply, error) {
	f.mu.Lock()
	f.tagLists = append(f.tagLists, mdSnap{md: recordIncomingMD(ctx)})
	f.mu.Unlock()
	return &blogv1.ListTagsReply{
		Tags: []*blogv1.TagInfo{{Id: 1, Name: "Go", Slug: "go", ArticleCount: 3}},
	}, nil
}

// ListArticles 返回固定文章列表（供 PUBLIC 带 token 的身份透传断言）。
func (f *e2eFakeBlog) ListArticles(ctx context.Context, _ *blogv1.ListArticlesRequest) (*blogv1.ListArticlesReply, error) {
	f.mu.Lock()
	f.articleLists = append(f.articleLists, mdSnap{md: recordIncomingMD(ctx)})
	f.mu.Unlock()
	return &blogv1.ListArticlesReply{
		Articles: []*blogv1.ArticleInfo{
			{Id: 7, Title: "第一篇文章", Slug: "hello-entry", Status: "published", AuthorId: 1},
		},
		Total: 1, Page: 1, PageSize: 10,
	}, nil
}

// UpdateSiteConfig 回显请求 config（响应侧 snake_case 契约断言 + 请求标题记录）。
func (f *e2eFakeBlog) UpdateSiteConfig(ctx context.Context, in *blogv1.UpdateSiteConfigRequest) (*blogv1.UpdateSiteConfigReply, error) {
	f.mu.Lock()
	f.siteUpdates = append(f.siteUpdates, siteUpdateSnap{
		md: recordIncomingMD(ctx), title: in.Config.GetSiteTitle(),
	})
	f.mu.Unlock()
	// 回显同一 config 指针：请求体内容原样出现在响应 data，证明 body 绑定→转发→响应闭环
	return &blogv1.UpdateSiteConfigReply{Config: in.Config}, nil
}

// siteUpdateCount 返回 UpdateSiteConfig 调用次数（0 即越权请求未达下游）。
func (f *e2eFakeBlog) siteUpdateCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.siteUpdates)
}

// lastArticleList 返回最近一次 ListArticles 快照（无调用时第二个返回值为 false）。
func (f *e2eFakeBlog) lastArticleList() (mdSnap, bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.articleLists) == 0 {
		return mdSnap{}, false
	}
	return f.articleLists[len(f.articleLists)-1], true
}

// lastTagList 返回最近一次 ListTags 快照（无调用时第二个返回值为 false）。
func (f *e2eFakeBlog) lastTagList() (mdSnap, bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.tagLists) == 0 {
		return mdSnap{}, false
	}
	return f.tagLists[len(f.tagLists)-1], true
}

// lastSiteUpdate 返回最近一次 UpdateSiteConfig 快照（无调用时第二个返回值为 false）。
func (f *e2eFakeBlog) lastSiteUpdate() (siteUpdateSnap, bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.siteUpdates) == 0 {
		return siteUpdateSnap{}, false
	}
	return f.siteUpdates[len(f.siteUpdates)-1], true
}

// ===================== 元数据断言辅助 =====================

// mdVal 断言 key 在 server 端收到的元数据中存在并返回其值。
func mdVal(t *testing.T, md metadata.MD, key string) string {
	t.Helper()
	vals := md.Get(key)
	if len(vals) == 0 {
		t.Fatalf("server 端元数据缺少 %q（实际键: %v）", key, mdKeys(md))
	}
	return vals[0]
}

// mdAbsent 断言 key 未出现在 server 端元数据中（匿名请求场景）。
func mdAbsent(t *testing.T, md metadata.MD, key string) {
	t.Helper()
	if vals := md.Get(key); len(vals) > 0 {
		t.Fatalf("server 端元数据不应包含 %q，实际值 %v", key, vals)
	}
}

// mdKeys 返回元数据全部键，便于失败信息展示。
func mdKeys(md metadata.MD) []string {
	keys := make([]string, 0, len(md))
	for k := range md {
		keys = append(keys, k)
	}
	return keys
}
