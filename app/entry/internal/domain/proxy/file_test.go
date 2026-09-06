package proxy

// ===================== 文件域 handler 测试（file_test.go） =====================
//
// 覆盖 FileHandler 7 个 handler 的完整代理链路（HTTP → handler → gRPC → fake）：
//  1. UploadFile：body（含 base64 的 content）→ protojson 自动解码为 []byte——
//     核心断言解码正确 + 认证元数据透传；
//  2. GetFile/DeleteFile：:id 路径参数合并（BindPathUint64）+ 非法值 400 短路；
//  3. ListFiles/GetPresignedPutURL：GET query 绑定（page/page_size、filename/mime_type）
//     + 非法 query 400 短路；
//  4. CreatePresignedUpload/CompletePresignedUpload：body 绑定 + 成功信封；
//  5. 错误映射：kratos 业务错误/原始 gRPC 状态错误 → 失败信封；
//  6. 匿名请求（无 meta 中间件）不透传认证键、真实 IP 仍到达。
//
// helpers_test.go 的 bufconn fake 只注册了 auth/ArticleService，而本文件
// 需要独立的 FileService 注册——故在 file_test.go 内自建 bufconn 环境
// （fakeFileServer 注册在独立 grpc server 上），拨号仍复用 helpers_test.go
// 的 dialBufconn（与生产同款 kratos metadata.Client() 中间件，保证元数据
// 透传断言有效）。helpers_test.go 保持只读不被改动。

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	blogv1 "github.com/CycleZero/ley/api/blog/v1"
	pkgmeta "github.com/CycleZero/ley/pkg/meta"
	"github.com/gin-gonic/gin"
	kerrors "github.com/go-kratos/kratos/v2/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
	"google.golang.org/protobuf/proto"
)

// ===================== fake FileService（独立注册） =====================

// fakeFileServer 假 blog FileService：实现全部 7 个 RPC（固定回复 + 记录
// 收到的元数据与请求消息），errs 按 RPC 方法名注入错误（错误映射测试注入点）。
type fakeFileServer struct {
	blogv1.UnimplementedFileServiceServer

	mu    sync.Mutex
	errs  map[string]error // 方法名 → 注入错误（nil 不注入）
	calls []fileCallSnap   // 按调用顺序记录全部 RPC 快照
}

// fileCallSnap 一次 File RPC 的 server 端快照。
type fileCallSnap struct {
	md     metadata.MD   // server 端收到的元数据（断言 x-md-global-* 透传）
	method string        // RPC 方法名（如 "UploadFile"）
	req    proto.Message // 原始请求消息（断言时按具体 RPC 类型断言）
}

// setErr 为指定方法注入错误（err 为 nil 时清除注入）。
func (f *fakeFileServer) setErr(method string, err error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if err == nil {
		delete(f.errs, method)
		return
	}
	if f.errs == nil {
		f.errs = map[string]error{}
	}
	f.errs[method] = err
}

// errOf 返回指定方法注入的错误（无注入时 nil）。
func (f *fakeFileServer) errOf(method string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.errs[method]
}

// record 追加一次调用快照（并发安全）。
func (f *fakeFileServer) record(method string, md metadata.MD, req proto.Message) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls = append(f.calls, fileCallSnap{md: md, method: method, req: req})
}

// lastCall 返回最近一次调用快照（无调用时 ok=false）。
func (f *fakeFileServer) lastCall() (fileCallSnap, bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.calls) == 0 {
		return fileCallSnap{}, false
	}
	return f.calls[len(f.calls)-1], true
}

// callCount 返回累计调用次数（断言失败路径未触发下游）。
func (f *fakeFileServer) callCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.calls)
}

// fixedFileInfo 固定文件信息回复：各 File RPC 成功回复复用，便于统一断言。
func fixedFileInfo() *blogv1.FileInfo {
	return &blogv1.FileInfo{
		Id:        77,
		Filename:  "cover.png",
		MimeType:  "image/png",
		Size:      12345,
		Url:       "https://cdn.example.com/files/cover.png",
		CreatedAt: "2026-09-06T10:00:00Z",
	}
}

// 各 RPC 统一入口语义：注入错误优先返回，否则记录快照 + 固定回复。
func (f *fakeFileServer) UploadFile(ctx context.Context, in *blogv1.UploadFileRequest) (*blogv1.UploadFileReply, error) {
	if err := f.errOf("UploadFile"); err != nil {
		return nil, err
	}
	f.record("UploadFile", recordIncomingMD(ctx), in)
	return &blogv1.UploadFileReply{File: fixedFileInfo()}, nil
}

func (f *fakeFileServer) GetFile(ctx context.Context, in *blogv1.GetFileRequest) (*blogv1.GetFileReply, error) {
	if err := f.errOf("GetFile"); err != nil {
		return nil, err
	}
	f.record("GetFile", recordIncomingMD(ctx), in)
	return &blogv1.GetFileReply{File: fixedFileInfo()}, nil
}

func (f *fakeFileServer) DeleteFile(ctx context.Context, in *blogv1.DeleteFileRequest) (*blogv1.DeleteFileReply, error) {
	if err := f.errOf("DeleteFile"); err != nil {
		return nil, err
	}
	f.record("DeleteFile", recordIncomingMD(ctx), in)
	return &blogv1.DeleteFileReply{}, nil
}

func (f *fakeFileServer) ListFiles(ctx context.Context, in *blogv1.ListFilesRequest) (*blogv1.ListFilesReply, error) {
	if err := f.errOf("ListFiles"); err != nil {
		return nil, err
	}
	f.record("ListFiles", recordIncomingMD(ctx), in)
	return &blogv1.ListFilesReply{Files: []*blogv1.FileInfo{fixedFileInfo()}, Total: 1}, nil
}

func (f *fakeFileServer) GetPresignedPutURL(ctx context.Context, in *blogv1.GetPresignedPutURLRequest) (*blogv1.GetPresignedPutURLReply, error) {
	if err := f.errOf("GetPresignedPutURL"); err != nil {
		return nil, err
	}
	f.record("GetPresignedPutURL", recordIncomingMD(ctx), in)
	return &blogv1.GetPresignedPutURLReply{
		Url:       "https://minio.example.com/presign/cover.png",
		ObjectKey: "uploads/cover.png",
	}, nil
}

func (f *fakeFileServer) CreatePresignedUpload(ctx context.Context, in *blogv1.CreatePresignedUploadRequest) (*blogv1.CreatePresignedUploadReply, error) {
	if err := f.errOf("CreatePresignedUpload"); err != nil {
		return nil, err
	}
	f.record("CreatePresignedUpload", recordIncomingMD(ctx), in)
	return &blogv1.CreatePresignedUploadReply{
		PresignedUrl: "https://minio.example.com/presign/abc123.png",
		ObjectKey:    "uploads/abc123.png",
		ExpiresIn:    3600,
	}, nil
}

func (f *fakeFileServer) CompletePresignedUpload(ctx context.Context, in *blogv1.CompletePresignedUploadRequest) (*blogv1.CompletePresignedUploadReply, error) {
	if err := f.errOf("CompletePresignedUpload"); err != nil {
		return nil, err
	}
	f.record("CompletePresignedUpload", recordIncomingMD(ctx), in)
	return &blogv1.CompletePresignedUploadReply{File: fixedFileInfo()}, nil
}

// ===================== 环境装配（file 专用 bufconn） =====================

// fileTestEnv 文件域测试环境：独立 bufconn grpc server（仅注册 FileService）+
// 与生产同款拨号的 client 连接 + 由该连接装配的 ServiceHub + 裸 gin engine。
type fileTestEnv struct {
	t      *testing.T
	fake   *fakeFileServer // 假 FileService（断言注入点）
	hub    *ServiceHub     // 仅 File 字段装配（文件域 handler 只依赖 hub.File）
	engine *gin.Engine     // 裸 gin engine（gin.New()，无全局中间件）
}

// newFileTestEnv 装配文件域测试环境：bufconn 监听 + 注册 fakeFileServer +
// dialBufconn 拨号（复用 helpers_test.go，中间件链与生产 dialServiceConn 一致）。
func newFileTestEnv(t *testing.T) *fileTestEnv {
	t.Helper()
	gin.SetMode(gin.TestMode)

	lis := bufconn.Listen(1 << 20)
	fake := &fakeFileServer{}
	srv := grpc.NewServer()
	blogv1.RegisterFileServiceServer(srv, fake)
	// Serve 阻塞运行；t.Cleanup 中 srv.Stop() 使其正常返回
	go func() { _ = srv.Serve(lis) }()
	t.Cleanup(func() { srv.Stop() })

	conn := dialBufconn(t, lis, "blog-file")
	return &fileTestEnv{
		t:      t,
		fake:   fake,
		hub:    &ServiceHub{File: blogv1.NewFileServiceClient(conn)},
		engine: gin.New(),
	}
}

// registerFileHandler 将 FileHandler 的 7 条路由挂到 env.engine（路由与
// 权威路由表一致；gin v1.12 支持 :id 通配与静态段共存，/presigned-upload
// 等静态路由优先于 :id 匹配）。mw 为可选的认证等中间件（按序先执行）。
func registerFileHandler(env *fileTestEnv, mw ...gin.HandlerFunc) {
	t := env.t
	t.Helper()
	fh := NewFileHandler(env.hub)
	// 以 mw 开头挂载每个 handler（复制切片，避免 append 污染调用方底层数组）
	with := func(h gin.HandlerFunc) []gin.HandlerFunc {
		return append(append([]gin.HandlerFunc{}, mw...), h)
	}
	env.engine.POST("/api/v1/files/upload", with(fh.UploadFile)...)
	env.engine.GET("/api/v1/files/:id", with(fh.GetFile)...)
	env.engine.DELETE("/api/v1/files/:id", with(fh.DeleteFile)...)
	env.engine.GET("/api/v1/files", with(fh.ListFiles)...)
	env.engine.GET("/api/v1/files/presigned-upload", with(fh.GetPresignedPutURL)...)
	env.engine.POST("/api/v1/files/presigned-uploads", with(fh.CreatePresignedUpload)...)
	env.engine.POST("/api/v1/files/presigned-uploads/complete", with(fh.CompletePresignedUpload)...)
}

// mustFileCall 返回 fake 最近一次调用快照（无调用直接失败）。
func mustFileCall(t *testing.T, env *fileTestEnv) fileCallSnap {
	t.Helper()
	snap, ok := env.fake.lastCall()
	if !ok {
		t.Fatal("fake FileService 未收到任何调用")
	}
	return snap
}

// wantOK 断言 200 成功信封 {code:0 msg:ok} 并返回信封。
func wantOK(t *testing.T, w *httptest.ResponseRecorder) testEnvelope {
	t.Helper()
	if w.Code != http.StatusOK {
		t.Fatalf("HTTP 状态 = %d, 期望 %d，body=%s", w.Code, http.StatusOK, w.Body.String())
	}
	env := decodeEnvelope(t, w)
	if env.Code != 0 || env.Msg != "ok" {
		t.Fatalf("信封 = {code:%d msg:%q}, 期望 {code:0 msg:ok}，body=%s", env.Code, env.Msg, w.Body.String())
	}
	return env
}

// wantFail 断言失败信封：HTTP 状态 + 业务码镜像 httpStatus，msg 含 msgPart。
func wantFail(t *testing.T, w *httptest.ResponseRecorder, httpStatus int, msgPart string) {
	t.Helper()
	if w.Code != httpStatus {
		t.Fatalf("HTTP 状态 = %d, 期望 %d，body=%s", w.Code, httpStatus, w.Body.String())
	}
	env := decodeEnvelope(t, w)
	if env.Code != httpStatus || !strings.Contains(env.Msg, msgPart) {
		t.Fatalf("信封 = {code:%d msg:%q}, 期望 {code:%d msg:含%q}，body=%s",
			env.Code, env.Msg, httpStatus, msgPart, w.Body.String())
	}
}

// ===================== 1. UploadFile：base64 解码 + 元数据透传 =====================

// TestFile_Upload_Base64DecodeAndMeta：Given 认证请求 + body 中 content 为
// base64 文件字节；When POST 上传；Then 200 成功信封（data 为 snake_case
// FileInfo JSON）、fake 收到解码后的原始字节（base64 → []byte 由 protojson
// 绑定自动完成）+ filename/mime_type + 认证元数据与真实 IP 到达。
func TestFile_Upload_Base64DecodeAndMeta(t *testing.T) {
	// Given
	env := newFileTestEnv(t)
	registerFileHandler(env, withAuthMeta(7, "author1", "author"))
	content := []byte("hello world")
	body := fmt.Sprintf(`{"filename":"demo.txt","mime_type":"text/plain","content":%q}`,
		base64.StdEncoding.EncodeToString(content))
	// When
	w := doRequest(t, env.engine, http.MethodPost, "/api/v1/files/upload", body)
	// Then：成功信封 + snake_case 契约
	resp := wantOK(t, w)
	raw := string(resp.Data)
	for _, want := range []string{`"file":{`, `"id":"77"`, `"filename":"cover.png"`, `"mime_type":"image/png"`, `"created_at":"2026-09-06T10:00:00Z"`} {
		if !strings.Contains(raw, want) {
			t.Errorf("data 缺少片段 %s，data=%s", want, raw)
		}
	}
	if strings.Contains(raw, `"mimeType"`) || strings.Contains(raw, `"createdAt"`) {
		t.Errorf("data 出现 camelCase 键（期望 UseProtoNames），data=%s", raw)
	}
	// Then：fake 收到 base64 解码后的字节 + 请求字段
	snap := mustFileCall(t, env)
	in, ok := snap.req.(*blogv1.UploadFileRequest)
	if !ok {
		t.Fatalf("收到请求类型 %T, 期望 *blogv1.UploadFileRequest", snap.req)
	}
	if string(in.Content) != "hello world" {
		t.Errorf("content 解码结果 = %q, 期望 %q（base64 未正确解码）", in.Content, "hello world")
	}
	if in.Filename != "demo.txt" || in.MimeType != "text/plain" {
		t.Errorf("fake 收到 filename=%q mime_type=%q, 期望 demo.txt/text/plain", in.Filename, in.MimeType)
	}
	// Then：认证元数据 + 真实 IP 透传
	if got := mdVal(t, snap.md, pkgmeta.AuthUserIDKey); got != "7" {
		t.Errorf("x-md-global-auth-user-id = %q, 期望 %q", got, "7")
	}
	if got := mdVal(t, snap.md, pkgmeta.AuthUserNameKey); got != "author1" {
		t.Errorf("x-md-global-auth-user-name = %q, 期望 %q", got, "author1")
	}
	if got := mdVal(t, snap.md, pkgmeta.AuthRealClientIpKey); got != "203.0.113.9" {
		t.Errorf("x-md-global-auth-real-ip = %q, 期望 %q", got, "203.0.113.9")
	}
}

// TestFile_Upload_BadJSON：Given 非法 JSON body；When 转发；
// Then 400「参数解析失败」且不触发下游调用。
func TestFile_Upload_BadJSON(t *testing.T) {
	// Given
	env := newFileTestEnv(t)
	registerFileHandler(env, withAuthMeta(7, "author1", "author"))
	// When
	w := doRequest(t, env.engine, http.MethodPost, "/api/v1/files/upload", `{"filename":`)
	// Then
	wantFail(t, w, http.StatusBadRequest, "参数解析失败")
	if env.fake.callCount() != 0 {
		t.Errorf("非法 JSON 不应触发下游调用，实际调用 %d 次", env.fake.callCount())
	}
}

// TestFile_Upload_InvalidBase64：Given body 中 content 为非法 base64；
// When 转发；Then 400「参数解析失败」（protojson 解码失败即绑定失败）
// 且不触发下游调用。
func TestFile_Upload_InvalidBase64(t *testing.T) {
	// Given
	env := newFileTestEnv(t)
	registerFileHandler(env, withAuthMeta(7, "author1", "author"))
	// When
	w := doRequest(t, env.engine, http.MethodPost, "/api/v1/files/upload",
		`{"filename":"a.bin","mime_type":"application/octet-stream","content":"!!!not-base64!!!"}`)
	// Then
	wantFail(t, w, http.StatusBadRequest, "参数解析失败")
	if env.fake.callCount() != 0 {
		t.Errorf("非法 base64 不应触发下游调用，实际调用 %d 次", env.fake.callCount())
	}
}

// ===================== 2. GetFile/DeleteFile：路径参数 =====================

// TestFile_GetFile_PathParamAndSuccess：Given 认证请求 GET :id=99；
// When 转发；Then fake 收到 Id=99 且成功信封 data 携带固定 FileInfo。
func TestFile_GetFile_PathParamAndSuccess(t *testing.T) {
	// Given
	env := newFileTestEnv(t)
	registerFileHandler(env, withAuthMeta(7, "author1", "author"))
	// When
	w := doRequest(t, env.engine, http.MethodGet, "/api/v1/files/99", "")
	// Then
	resp := wantOK(t, w)
	if raw := string(resp.Data); !strings.Contains(raw, `"id":"77"`) {
		t.Errorf("data 缺少固定文件 id，data=%s", raw)
	}
	snap := mustFileCall(t, env)
	in, ok := snap.req.(*blogv1.GetFileRequest)
	if !ok {
		t.Fatalf("收到请求类型 %T, 期望 *blogv1.GetFileRequest", snap.req)
	}
	if in.Id != 99 {
		t.Errorf("fake 收到 Id = %d, 期望 99（路径参数合并失败）", in.Id)
	}
	if got := mdVal(t, snap.md, pkgmeta.AuthUserIDKey); got != "7" {
		t.Errorf("x-md-global-auth-user-id = %q, 期望 %q", got, "7")
	}
}

// TestFile_DeleteFile_PathParamEmptyReply：Given 认证请求 DELETE :id=123；
// When 转发；Then fake 收到 Id=123，成功信封 data 为 {}（空回复序列化结果）。
func TestFile_DeleteFile_PathParamEmptyReply(t *testing.T) {
	// Given
	env := newFileTestEnv(t)
	registerFileHandler(env, withAuthMeta(7, "author1", "author"))
	// When
	w := doRequest(t, env.engine, http.MethodDelete, "/api/v1/files/123", "")
	// Then
	resp := wantOK(t, w)
	if raw := string(resp.Data); raw != "{}" {
		t.Errorf("data = %s, 期望 {}（删除类空回复序列化结果）", raw)
	}
	snap := mustFileCall(t, env)
	in, ok := snap.req.(*blogv1.DeleteFileRequest)
	if !ok {
		t.Fatalf("收到请求类型 %T, 期望 *blogv1.DeleteFileRequest", snap.req)
	}
	if in.Id != 123 {
		t.Errorf("fake 收到 Id = %d, 期望 123（路径参数合并失败）", in.Id)
	}
}

// TestFile_DeleteFile_BadPathParam：Given :id 非法（非整数）；
// When BindPathUint64 解析失败；Then 400 信封并短路——不触发下游调用。
func TestFile_DeleteFile_BadPathParam(t *testing.T) {
	// Given
	env := newFileTestEnv(t)
	registerFileHandler(env, withAuthMeta(7, "author1", "author"))
	// When
	w := doRequest(t, env.engine, http.MethodDelete, "/api/v1/files/abc", "")
	// Then
	wantFail(t, w, http.StatusBadRequest, "必须为无符号整数")
	if env.fake.callCount() != 0 {
		t.Errorf("非法路径参数不应触发下游调用，实际调用 %d 次", env.fake.callCount())
	}
}

// ===================== 3. ListFiles / GetPresignedPutURL：query 绑定 =====================

// TestFile_ListFiles_QueryParams：Given GET query page/page_size；
// When 转发；Then fake 收到解析后的分页字段，成功信封 data 携带 files/total。
func TestFile_ListFiles_QueryParams(t *testing.T) {
	// Given
	env := newFileTestEnv(t)
	registerFileHandler(env, withAuthMeta(7, "author1", "author"))
	// When
	w := doRequest(t, env.engine, http.MethodGet, "/api/v1/files?page=2&page_size=20", "")
	// Then
	resp := wantOK(t, w)
	raw := string(resp.Data)
	for _, want := range []string{`"files":[`, `"total":"1"`} {
		if !strings.Contains(raw, want) {
			t.Errorf("data 缺少片段 %s，data=%s", want, raw)
		}
	}
	snap := mustFileCall(t, env)
	in, ok := snap.req.(*blogv1.ListFilesRequest)
	if !ok {
		t.Fatalf("收到请求类型 %T, 期望 *blogv1.ListFilesRequest", snap.req)
	}
	if in.Page != 2 || in.PageSize != 20 {
		t.Errorf("fake 收到 page=%d page_size=%d, 期望 2/20（query 绑定失败）", in.Page, in.PageSize)
	}
}

// TestFile_ListFiles_BadQuery：Given page 非整数；When 转发；
// Then 400 信封并短路——不触发下游调用。
func TestFile_ListFiles_BadQuery(t *testing.T) {
	// Given
	env := newFileTestEnv(t)
	registerFileHandler(env, withAuthMeta(7, "author1", "author"))
	// When
	w := doRequest(t, env.engine, http.MethodGet, "/api/v1/files?page=abc", "")
	// Then
	wantFail(t, w, http.StatusBadRequest, "查询参数 page 必须为整数")
	if env.fake.callCount() != 0 {
		t.Errorf("非法 query 不应触发下游调用，实际调用 %d 次", env.fake.callCount())
	}
}

// TestFile_GetPresignedPutURL_QueryAndAnonymous：Given 匿名 GET query
// filename/mime_type（本接口可不带认证中间件）；When 转发；Then query 绑定
// 正确、成功信封携带 url/object_key、server 端不透传认证键但真实 IP 到达。
func TestFile_GetPresignedPutURL_QueryAndAnonymous(t *testing.T) {
	// Given：不挂 withAuthMeta，模拟公开路由的匿名请求
	env := newFileTestEnv(t)
	registerFileHandler(env)
	// When
	w := doRequest(t, env.engine, http.MethodGet,
		"/api/v1/files/presigned-upload?filename=cover.png&mime_type=image/png", "")
	// Then：成功信封 + snake_case 回复字段
	resp := wantOK(t, w)
	raw := string(resp.Data)
	for _, want := range []string{`"url":"https://minio.example.com/presign/cover.png"`, `"object_key":"uploads/cover.png"`} {
		if !strings.Contains(raw, want) {
			t.Errorf("data 缺少片段 %s，data=%s", want, raw)
		}
	}
	// Then：query 绑定结果
	snap := mustFileCall(t, env)
	in, ok := snap.req.(*blogv1.GetPresignedPutURLRequest)
	if !ok {
		t.Fatalf("收到请求类型 %T, 期望 *blogv1.GetPresignedPutURLRequest", snap.req)
	}
	if in.Filename != "cover.png" || in.MimeType != "image/png" {
		t.Errorf("fake 收到 filename=%q mime_type=%q, 期望 cover.png/image/png（query 绑定失败）", in.Filename, in.MimeType)
	}
	// Then：匿名请求不透传认证键，真实 IP 仍透传（BuildRequestMeta 回填）
	for _, key := range []string{pkgmeta.AuthUserIDKey, pkgmeta.AuthUserNameKey, pkgmeta.AuthUserRoleKey} {
		mdAbsent(t, snap.md, key)
	}
	if got := mdVal(t, snap.md, pkgmeta.AuthRealClientIpKey); got != "203.0.113.9" {
		t.Errorf("x-md-global-auth-real-ip = %q, 期望 %q", got, "203.0.113.9")
	}
}

// ===================== 4. 预签名创建/完成：body 绑定 =====================

// TestFile_CreatePresignedUpload_Body：Given 认证请求 + body；
// When 转发；Then fake 收到全部字段，成功信封携带 presigned_url/object_key/expires_in。
func TestFile_CreatePresignedUpload_Body(t *testing.T) {
	// Given
	env := newFileTestEnv(t)
	registerFileHandler(env, withAuthMeta(7, "author1", "author"))
	// When
	w := doRequest(t, env.engine, http.MethodPost, "/api/v1/files/presigned-uploads",
		`{"filename":"demo.zip","mime_type":"application/zip","size":4096}`)
	// Then
	resp := wantOK(t, w)
	raw := string(resp.Data)
	for _, want := range []string{`"presigned_url":"https://minio.example.com/presign/abc123.png"`, `"object_key":"uploads/abc123.png"`, `"expires_in":"3600"`} {
		if !strings.Contains(raw, want) {
			t.Errorf("data 缺少片段 %s，data=%s", want, raw)
		}
	}
	snap := mustFileCall(t, env)
	in, ok := snap.req.(*blogv1.CreatePresignedUploadRequest)
	if !ok {
		t.Fatalf("收到请求类型 %T, 期望 *blogv1.CreatePresignedUploadRequest", snap.req)
	}
	if in.Filename != "demo.zip" || in.MimeType != "application/zip" || in.Size != 4096 {
		t.Errorf("fake 收到 filename=%q mime_type=%q size=%d, 期望 demo.zip/application/zip/4096", in.Filename, in.MimeType, in.Size)
	}
}

// TestFile_CompletePresignedUpload_Body：Given 认证请求 + body（object_key）；
// When 转发；Then fake 收到 object_key 等字段，成功信封携带登记后的 FileInfo。
func TestFile_CompletePresignedUpload_Body(t *testing.T) {
	// Given
	env := newFileTestEnv(t)
	registerFileHandler(env, withAuthMeta(7, "author1", "author"))
	// When
	w := doRequest(t, env.engine, http.MethodPost, "/api/v1/files/presigned-uploads/complete",
		`{"object_key":"uploads/abc123.png","filename":"abc.png","mime_type":"image/png"}`)
	// Then
	resp := wantOK(t, w)
	if raw := string(resp.Data); !strings.Contains(raw, `"id":"77"`) {
		t.Errorf("data 缺少登记后的文件 id，data=%s", raw)
	}
	snap := mustFileCall(t, env)
	in, ok := snap.req.(*blogv1.CompletePresignedUploadRequest)
	if !ok {
		t.Fatalf("收到请求类型 %T, 期望 *blogv1.CompletePresignedUploadRequest", snap.req)
	}
	if in.ObjectKey != "uploads/abc123.png" || in.Filename != "abc.png" {
		t.Errorf("fake 收到 object_key=%q filename=%q, 期望 uploads/abc123.png/abc.png", in.ObjectKey, in.Filename)
	}
}

// ===================== 5. 错误映射 =====================

// TestFile_GetFile_KratosNotFound：Given fake GetFile 返回 kratos 业务错误；
// When 转发；Then 404 信封、msg 取业务消息（中文透出）、data 为 null。
func TestFile_GetFile_KratosNotFound(t *testing.T) {
	// Given
	env := newFileTestEnv(t)
	env.fake.setErr("GetFile", kerrors.NotFound("FILE_NOT_FOUND", "文件不存在"))
	registerFileHandler(env, withAuthMeta(7, "author1", "author"))
	// When
	w := doRequest(t, env.engine, http.MethodGet, "/api/v1/files/99", "")
	// Then
	wantFail(t, w, http.StatusNotFound, "文件不存在")
}

// TestFile_ListFiles_PermissionDenied：Given fake ListFiles 返回原始 gRPC
// PermissionDenied；When 转发；Then 403 信封、msg 取 gRPC 状态消息。
func TestFile_ListFiles_PermissionDenied(t *testing.T) {
	// Given
	env := newFileTestEnv(t)
	env.fake.setErr("ListFiles", status.Error(codes.PermissionDenied, "无权访问该文件列表"))
	registerFileHandler(env, withAuthMeta(7, "author1", "author"))
	// When
	w := doRequest(t, env.engine, http.MethodGet, "/api/v1/files?page=1&page_size=10", "")
	// Then
	wantFail(t, w, http.StatusForbidden, "无权访问该文件列表")
}
