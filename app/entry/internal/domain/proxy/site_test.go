package proxy

// ===================== 站点配置域 handler 测试（site_test.go） =====================
//
// 覆盖 site.go 8 个 handler：body→请求消息绑定（含嵌套 config/playlist 包装结构、
// bytes base64 解码）、路径参数合并、元数据透传、成功信封（snake_case data）与
// 错误映射，全部经 bufconn fake gRPC 走「HTTP → handler → callProto → 下游」全链路。
//
// 说明：helpers_test.go 的 newTestEnv 只注册了 ArticleService，且 grpc server 在
// Serve 开始后禁止再 RegisterService；本站点 fake 在本文档自建 server + env
// （拨号仍复用 dialBufconn，与生产同款 metadata.Client() 中间件），不改动
// helpers_test.go。断言辅助 doRequest/decodeEnvelope/dataMap/testEnvelope 复用
// handle_test.go（同包）。
//
// allow: SIZE_OK — 代码行超 250 上限（约 1/3 为 8 方法 fake gRPC 服务本体）：
// fake 无法迁入已冻结的 helpers_test.go，grpc server 注册又必须在 Serve 前完成
// （不可复用 newTestEnv 的 server），wave 交付约束亦限定本站点仅新增
// site.go + site_test.go 两文件，故测试脚手架与用例合于本文件。

import (
	"context"
	"net/http"
	"strings"
	"sync"
	"testing"

	blogv1 "github.com/CycleZero/ley/api/blog/v1"
	pkgmeta "github.com/CycleZero/ley/pkg/meta"
	"github.com/gin-gonic/gin"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
	"google.golang.org/protobuf/proto"
)

// ===================== fake Site 服务 =====================

// siteCall 一次 SiteService 调用的 server 端快照：收到的元数据 + 请求消息
// （record 时 proto.Clone，避免与调用方共享内存）。
type siteCall struct {
	md  metadata.MD
	req proto.Message
}

// fakeSiteServer 假 blog Site 服务：8 个方法固定返回成功回复并记录请求快照；
// err 非 nil 时所有方法直接返回该错误（错误映射测试注入点）。
type fakeSiteServer struct {
	blogv1.UnimplementedSiteServiceServer

	err error

	mu    sync.Mutex
	calls []siteCall
}

// record 追加一次调用快照；err 注入时返回错误且不记录（模拟下游失败）。
func (f *fakeSiteServer) record(ctx context.Context, in proto.Message) error {
	if f.err != nil {
		return f.err
	}
	f.mu.Lock()
	f.calls = append(f.calls, siteCall{md: recordIncomingMD(ctx), req: proto.Clone(in)})
	f.mu.Unlock()
	return nil
}

// lastSiteCall 返回最近一次调用快照（无调用时第二个返回值为 false）。
func (f *fakeSiteServer) lastSiteCall() (siteCall, bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.calls) == 0 {
		return siteCall{}, false
	}
	return f.calls[len(f.calls)-1], true
}

// GetSiteConfig 固定返回一份站点配置。
func (f *fakeSiteServer) GetSiteConfig(ctx context.Context, in *blogv1.GetSiteConfigRequest) (*blogv1.GetSiteConfigReply, error) {
	if err := f.record(ctx, in); err != nil {
		return nil, err
	}
	return &blogv1.GetSiteConfigReply{Config: &blogv1.SiteConfig{
		SiteTitle:    "Ley 博客",
		SiteSubtitle: "记录与分享",
		EnableLikes:  true,
		IcpNumber:    "粤ICP备00000000号",
	}}, nil
}

// UpdateSiteConfig 回显收到的配置（证明嵌套 config 绑定正确）。
func (f *fakeSiteServer) UpdateSiteConfig(ctx context.Context, in *blogv1.UpdateSiteConfigRequest) (*blogv1.UpdateSiteConfigReply, error) {
	if err := f.record(ctx, in); err != nil {
		return nil, err
	}
	return &blogv1.UpdateSiteConfigReply{Config: in.Config}, nil
}

// ListBackgrounds 固定返回一张激活背景。
func (f *fakeSiteServer) ListBackgrounds(ctx context.Context, in *blogv1.ListBackgroundsRequest) (*blogv1.ListBackgroundsReply, error) {
	if err := f.record(ctx, in); err != nil {
		return nil, err
	}
	return &blogv1.ListBackgroundsReply{Backgrounds: []*blogv1.SiteBackground{
		{Id: 1, Filename: "aurora.jpg", Url: "https://cdn.example.com/aurora.jpg", IsActive: true, SortOrder: 1},
	}}, nil
}

// UploadBackground 固定返回上传结果（Id 2 回显 filename，证明请求绑定到达）。
func (f *fakeSiteServer) UploadBackground(ctx context.Context, in *blogv1.UploadBackgroundRequest) (*blogv1.UploadBackgroundReply, error) {
	if err := f.record(ctx, in); err != nil {
		return nil, err
	}
	return &blogv1.UploadBackgroundReply{Background: &blogv1.SiteBackground{Id: 2, Filename: in.Filename, Url: "https://cdn.example.com/" + in.Filename, IsActive: false}}, nil
}

// DeleteBackground 空回复（删除类接口无业务数据）。
func (f *fakeSiteServer) DeleteBackground(ctx context.Context, in *blogv1.DeleteBackgroundRequest) (*blogv1.DeleteBackgroundReply, error) {
	if err := f.record(ctx, in); err != nil {
		return nil, err
	}
	return &blogv1.DeleteBackgroundReply{}, nil
}

// SetActiveBackground 空回复（激活结果由后续查询体现）。
func (f *fakeSiteServer) SetActiveBackground(ctx context.Context, in *blogv1.SetActiveBackgroundRequest) (*blogv1.SetActiveBackgroundReply, error) {
	if err := f.record(ctx, in); err != nil {
		return nil, err
	}
	return &blogv1.SetActiveBackgroundReply{}, nil
}

// GetMusicPlaylist 固定返回一首歌。
func (f *fakeSiteServer) GetMusicPlaylist(ctx context.Context, in *blogv1.GetMusicPlaylistRequest) (*blogv1.GetMusicPlaylistReply, error) {
	if err := f.record(ctx, in); err != nil {
		return nil, err
	}
	return &blogv1.GetMusicPlaylistReply{Playlist: &blogv1.MusicPlaylist{Tracks: []*blogv1.MusicTrack{
		{Title: "起风了", Artist: "买辣椒也用券", Url: "https://music.example.com/qfl.mp3", CoverUrl: "https://cdn.example.com/qfl.jpg"},
	}}}, nil
}

// UpdateMusicPlaylist 回显收到的歌单（证明嵌套 playlist 绑定正确）。
func (f *fakeSiteServer) UpdateMusicPlaylist(ctx context.Context, in *blogv1.UpdateMusicPlaylistRequest) (*blogv1.UpdateMusicPlaylistReply, error) {
	if err := f.record(ctx, in); err != nil {
		return nil, err
	}
	return &blogv1.UpdateMusicPlaylistReply{Playlist: in.Playlist}, nil
}

// ===================== 环境装配 =====================

// siteTestEnv 站点域测试环境：bufconn fake Site server + ServiceHub（仅 Site
// client）+ 裸 gin engine。装配方式与 helpers_test.go newTestEnv 对齐。
type siteTestEnv struct {
	t      *testing.T
	fake   *fakeSiteServer
	hub    *ServiceHub
	engine *gin.Engine
}

// newSiteTestEnv 装配站点域测试环境（bufconn 监听 + grpc server 注册 fakeSite
// + dialBufconn 拨号 + hub；拨号中间件链与生产一致，见 helpers_test.go 说明）。
func newSiteTestEnv(t *testing.T) *siteTestEnv {
	t.Helper()
	gin.SetMode(gin.TestMode)

	lis := bufconn.Listen(1 << 20)
	fake := &fakeSiteServer{}
	srv := grpc.NewServer()
	blogv1.RegisterSiteServiceServer(srv, fake)
	go func() { _ = srv.Serve(lis) }()
	t.Cleanup(func() { srv.Stop() })

	blogConn := dialBufconn(t, lis, "blog")
	return &siteTestEnv{
		t:      t,
		fake:   fake,
		hub:    &ServiceHub{Site: blogv1.NewSiteServiceClient(blogConn)},
		engine: gin.New(),
	}
}

// registerSiteRoutes 挂载 8 个 handler 路由（形态对齐未来 router wave 装配）。
// 公开接口（config/背景列表/歌单 GET）不挂认证；管理接口挂 withAuthMeta 模拟
// T4 JWT 中间件，供元数据透传断言使用。
func registerSiteRoutes(env *siteTestEnv) {
	h := NewSiteHandler(env.hub)
	env.engine.GET("/api/v1/site/config", h.GetSiteConfig)
	env.engine.PUT("/api/v1/site/config", withAuthMeta(1, "owner", "admin"), h.UpdateSiteConfig)
	env.engine.GET("/api/v1/site/backgrounds", h.ListBackgrounds)
	env.engine.POST("/api/v1/site/backgrounds", withAuthMeta(1, "owner", "admin"), h.UploadBackground)
	env.engine.DELETE("/api/v1/site/backgrounds/:id", withAuthMeta(1, "owner", "admin"), h.DeleteBackground)
	env.engine.PUT("/api/v1/site/backgrounds/:id/active", withAuthMeta(1, "owner", "admin"), h.SetActiveBackground)
	env.engine.GET("/api/v1/site/music/playlist", h.GetMusicPlaylist)
	env.engine.PUT("/api/v1/site/music/playlist", withAuthMeta(1, "owner", "admin"), h.UpdateMusicPlaylist)
}

// assertNoDownstream 断言 fake 未收到任何调用（body/路径绑定失败短路场景）。
func (env *siteTestEnv) assertNoDownstream() {
	env.t.Helper()
	if _, ok := env.fake.lastSiteCall(); ok {
		env.t.Error("失败请求不应触发下游 SiteService 调用")
	}
}

// assertAuthMeta 断言管理接口透传的认证元数据（uid=1/role=admin，与
// registerSiteRoutes 的 withAuthMeta 种入值一致）。
func (env *siteTestEnv) assertAuthMeta(snap siteCall) {
	env.t.Helper()
	if got := mdVal(env.t, snap.md, pkgmeta.AuthUserIDKey); got != "1" {
		env.t.Errorf("x-md-global-auth-user-id = %q, 期望 %q", got, "1")
	}
	if got := mdVal(env.t, snap.md, pkgmeta.AuthUserRoleKey); got != "admin" {
		env.t.Errorf("x-md-global-auth-user-role = %q, 期望 %q", got, "admin")
	}
}

// ===================== 1. GET 类公开接口（空请求体 + 成功信封） =====================

// TestSite_GetPublic_SuccessEnvelope：Given 三个公开 GET 接口（站点配置/背景
// 列表/歌单）；When 转发；Then 200 成功信封、data 为 UseProtoNames 的
// snake_case 原始 JSON（含各自嵌套结构），fake 均收到调用。
func TestSite_GetPublic_SuccessEnvelope(t *testing.T) {
	cases := []struct {
		name, path string
		want       []string // data 必须包含的 snake_case 片段
		forbid     []string // data 不得出现的 camelCase 键
	}{
		{"GetSiteConfig", "/api/v1/site/config",
			[]string{`"config":`, `"site_title":"Ley 博客"`, `"enable_likes":true`, `"icp_number":"粤ICP备00000000号"`},
			[]string{`"siteTitle"`, `"enableLikes"`}},
		{"ListBackgrounds", "/api/v1/site/backgrounds",
			[]string{`"backgrounds":`, `"filename":"aurora.jpg"`, `"is_active":true`, `"sort_order":1`},
			nil},
		{"GetMusicPlaylist", "/api/v1/site/music/playlist",
			[]string{`"playlist":`, `"tracks":`, `"title":"起风了"`, `"cover_url":"https://cdn.example.com/qfl.jpg"`},
			nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// Given
			env := newSiteTestEnv(t)
			registerSiteRoutes(env)
			// When
			w := doRequest(t, env.engine, http.MethodGet, tc.path, "")
			// Then：信封契约 + snake_case data
			if w.Code != http.StatusOK {
				t.Fatalf("HTTP 状态 = %d, 期望 %d，body=%s", w.Code, http.StatusOK, w.Body.String())
			}
			resp := decodeEnvelope(t, w)
			if resp.Code != 0 || resp.Msg != "ok" {
				t.Errorf("信封 = {code:%d msg:%q}, 期望 {code:0 msg:ok}", resp.Code, resp.Msg)
			}
			raw := string(resp.Data)
			for _, want := range tc.want {
				if !strings.Contains(raw, want) {
					t.Errorf("data 缺少片段 %s，data=%s", want, raw)
				}
			}
			for _, forbid := range tc.forbid {
				if strings.Contains(raw, forbid) {
					t.Errorf("data 出现 camelCase 键 %s（期望 UseProtoNames），data=%s", forbid, raw)
				}
			}
			if _, ok := env.fake.lastSiteCall(); !ok {
				t.Fatal("fake 未收到调用")
			}
		})
	}
}

// ===================== 2. 嵌套 body 绑定（config/playlist 包装 + base64 bytes） =====================

// TestSite_UpdateSiteConfig_NestedConfigBody：Given 管理 PUT 携带
// {"config":{...}} 嵌套 body；When 转发；Then fake 收到解壳后的 config 字段
// （证明 protojson 对 body:"*" 全消息的自动剥壳），回复回显配置并带认证元数据。
func TestSite_UpdateSiteConfig_NestedConfigBody(t *testing.T) {
	// Given
	env := newSiteTestEnv(t)
	registerSiteRoutes(env)
	// When
	w := doRequest(t, env.engine, http.MethodPut, "/api/v1/site/config",
		`{"config":{"site_title":"Ley","seo_description":"技术博客","enable_likes":false},"extra_unknown":1}`)
	// Then
	if w.Code != http.StatusOK {
		t.Fatalf("HTTP 状态 = %d, 期望 %d，body=%s", w.Code, http.StatusOK, w.Body.String())
	}
	snap, ok := env.fake.lastSiteCall()
	if !ok {
		t.Fatal("fake 未收到 UpdateSiteConfig 调用")
	}
	got, want := snap.req, &blogv1.UpdateSiteConfigRequest{Config: &blogv1.SiteConfig{
		SiteTitle: "Ley", SeoDescription: "技术博客", EnableLikes: false,
	}}
	if !proto.Equal(got, want) {
		t.Errorf("fake 收到请求 = %v, 期望 %v（嵌套 config 解壳失败或零值字段丢失）", got, want)
	}
	env.assertAuthMeta(snap)
}

// TestSite_UploadBackground_Base64Bytes：Given 管理 POST 携带
// {"filename":...,"content":"<base64>"}；When 转发；Then fake 收到的 content 为
// base64 解码后的原始字节（证明 protojson bytes 自动解码）。
func TestSite_UploadBackground_Base64Bytes(t *testing.T) {
	// Given
	env := newSiteTestEnv(t)
	registerSiteRoutes(env)
	// When：base64("abc")=YWJj
	w := doRequest(t, env.engine, http.MethodPost, "/api/v1/site/backgrounds",
		`{"filename":"bg.png","content":"YWJj"}`)
	// Then
	if w.Code != http.StatusOK {
		t.Fatalf("HTTP 状态 = %d, 期望 %d，body=%s", w.Code, http.StatusOK, w.Body.String())
	}
	snap, ok := env.fake.lastSiteCall()
	if !ok {
		t.Fatal("fake 未收到 UploadBackground 调用")
	}
	got, ok := snap.req.(*blogv1.UploadBackgroundRequest)
	if !ok {
		t.Fatalf("fake 收到请求类型 = %T, 期望 *UploadBackgroundRequest", snap.req)
	}
	if got.Filename != "bg.png" || string(got.Content) != "abc" {
		t.Errorf("fake 收到 filename=%q content=%q, 期望 bg.png/abc（base64 解码失败）", got.Filename, got.Content)
	}
	if raw := string(decodeEnvelope(t, w).Data); !strings.Contains(raw, `"filename":"bg.png"`) {
		t.Errorf("回复 data 缺少回显 filename，data=%s", raw)
	}
}

// TestSite_UpdateMusicPlaylist_NestedPlaylistBody：Given 管理 PUT 携带
// {"playlist":{"tracks":[...]}} 嵌套 body；When 转发；Then fake 收到解壳后的
// 完整歌单（tracks 逐字段绑定）。
func TestSite_UpdateMusicPlaylist_NestedPlaylistBody(t *testing.T) {
	// Given
	env := newSiteTestEnv(t)
	registerSiteRoutes(env)
	// When
	w := doRequest(t, env.engine, http.MethodPut, "/api/v1/site/music/playlist",
		`{"playlist":{"tracks":[{"title":"Lemon","artist":"米津玄師","url":"https://m.example.com/lemon.mp3","cover_url":"https://c.example.com/lemon.jpg"}]}}`)
	// Then
	if w.Code != http.StatusOK {
		t.Fatalf("HTTP 状态 = %d, 期望 %d，body=%s", w.Code, http.StatusOK, w.Body.String())
	}
	snap, ok := env.fake.lastSiteCall()
	if !ok {
		t.Fatal("fake 未收到 UpdateMusicPlaylist 调用")
	}
	want := &blogv1.UpdateMusicPlaylistRequest{Playlist: &blogv1.MusicPlaylist{Tracks: []*blogv1.MusicTrack{
		{Title: "Lemon", Artist: "米津玄師", Url: "https://m.example.com/lemon.mp3", CoverUrl: "https://c.example.com/lemon.jpg"},
	}}}
	if !proto.Equal(snap.req, want) {
		t.Errorf("fake 收到请求 = %v, 期望 %v（嵌套 playlist 解壳失败）", snap.req, want)
	}
}

// TestSite_UpdateSiteConfig_BadJSON：Given 非法 JSON body；When PUT 转发；
// Then 400 信封「参数解析失败」且不触发下游调用。
func TestSite_UpdateSiteConfig_BadJSON(t *testing.T) {
	// Given
	env := newSiteTestEnv(t)
	registerSiteRoutes(env)
	// When
	w := doRequest(t, env.engine, http.MethodPut, "/api/v1/site/config", `{"config":`)
	// Then
	if w.Code != http.StatusBadRequest {
		t.Fatalf("HTTP 状态 = %d, 期望 %d，body=%s", w.Code, http.StatusBadRequest, w.Body.String())
	}
	resp := decodeEnvelope(t, w)
	if resp.Code != http.StatusBadRequest || resp.Msg != "参数解析失败" {
		t.Errorf("信封 = {code:%d msg:%q}, 期望 {code:400 msg:参数解析失败}", resp.Code, resp.Msg)
	}
	env.assertNoDownstream()
}

// ===================== 3. 路径参数合并（:id 删除/设激活） =====================

// TestSite_PathID_DeleteAndSetActive：Given 合法路径 id；When 删除背景
// （DELETE :id）/ 设为激活（PUT :id/active）；Then fake 收到合并后的 Id、
// 200 信封且空回复 data 为 {}。
func TestSite_PathID_DeleteAndSetActive(t *testing.T) {
	cases := []struct {
		name, method, path string
		want               proto.Message
	}{
		{"DeleteBackground", http.MethodDelete, "/api/v1/site/backgrounds/42", &blogv1.DeleteBackgroundRequest{Id: 42}},
		{"SetActiveBackground", http.MethodPut, "/api/v1/site/backgrounds/7/active", &blogv1.SetActiveBackgroundRequest{Id: 7}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// Given
			env := newSiteTestEnv(t)
			registerSiteRoutes(env)
			// When
			w := doRequest(t, env.engine, tc.method, tc.path, "")
			// Then
			if w.Code != http.StatusOK {
				t.Fatalf("HTTP 状态 = %d, 期望 %d，body=%s", w.Code, http.StatusOK, w.Body.String())
			}
			if raw := string(decodeEnvelope(t, w).Data); raw != "{}" {
				t.Errorf("data = %s, 期望 {}（空回复序列化结果）", raw)
			}
			snap, ok := env.fake.lastSiteCall()
			if !ok {
				t.Fatal("fake 未收到调用")
			}
			if !proto.Equal(snap.req, tc.want) {
				t.Errorf("fake 收到请求 = %v, 期望 %v（路径参数合并失败）", snap.req, tc.want)
			}
			env.assertAuthMeta(snap)
		})
	}
}

// TestSite_PathID_Invalid：Given 路径 id 非法（非整数）；When 删除/设激活；
// Then 400 信封并短路——不触发下游调用。
func TestSite_PathID_Invalid(t *testing.T) {
	cases := []struct {
		name, method, path string
	}{
		{"DeleteBackground", http.MethodDelete, "/api/v1/site/backgrounds/abc"},
		{"SetActiveBackground", http.MethodPut, "/api/v1/site/backgrounds/abc/active"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// Given
			env := newSiteTestEnv(t)
			registerSiteRoutes(env)
			// When
			w := doRequest(t, env.engine, tc.method, tc.path, "")
			// Then
			if w.Code != http.StatusBadRequest {
				t.Fatalf("HTTP 状态 = %d, 期望 %d，body=%s", w.Code, http.StatusBadRequest, w.Body.String())
			}
			resp := decodeEnvelope(t, w)
			if resp.Code != http.StatusBadRequest || !strings.Contains(resp.Msg, "必须为无符号整数") {
				t.Errorf("信封 = {code:%d msg:%q}, 期望 {code:400 msg:含'必须为无符号整数'}", resp.Code, resp.Msg)
			}
			env.assertNoDownstream()
		})
	}
}

// ===================== 4. 错误映射 =====================

// TestSite_GRPCErrorMapping：Given fake 返回 gRPC NotFound（中文消息）；
// When GET 站点配置转发失败；Then 404 信封、msg 透出中文状态消息、data 为 null。
func TestSite_GRPCErrorMapping(t *testing.T) {
	// Given
	env := newSiteTestEnv(t)
	env.fake.err = status.Error(codes.NotFound, "站点配置不存在")
	registerSiteRoutes(env)
	// When
	w := doRequest(t, env.engine, http.MethodGet, "/api/v1/site/config", "")
	// Then
	if w.Code != http.StatusNotFound {
		t.Fatalf("HTTP 状态 = %d, 期望 %d，body=%s", w.Code, http.StatusNotFound, w.Body.String())
	}
	resp := decodeEnvelope(t, w)
	if resp.Code != http.StatusNotFound || resp.Msg != "站点配置不存在" {
		t.Errorf("信封 = {code:%d msg:%q}, 期望 {code:404 msg:站点配置不存在}", resp.Code, resp.Msg)
	}
	if len(resp.Data) != 0 && string(resp.Data) != "null" {
		t.Errorf("失败信封 data = %s, 期望 null", resp.Data)
	}
}
