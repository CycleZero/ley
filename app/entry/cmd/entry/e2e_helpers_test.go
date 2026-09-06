package main

// ===================== E2E 环境装配（e2e_helpers_test.go） =====================
//
// 端到端测试环境：bufconn fake gRPC server（见 e2e_fakes_test.go）+ 与生产同款
// 拨号的 client 连接 + 真实路由注册装配。核心是 newE2EEnv——绕过 wire（真实
// initApp 依赖 etcd 客户端/服务发现，单测不可用），按 app.go NewMainApp 签名
// 手动注入全部 fake 依赖：
//
//	bootstrap（Prod 日志模式）→ NewMainApp(holder 降级默认配置 + 内存黑名单 + JWT)
//
// 链路与生产完全一致：Recovery → RegisteredMiddleWire.Register()（发布 JWT 认证
// 工厂 + 按默认远程配置组装全局链：RequestLogger → AddMetaData → CORS）→
// RegisterRouter（全局链挂载 + 39 条业务路由 + healthz + 404/405 兜底）→
// handler → gRPC client（kratos metadata.Client() 透传 x-md-global-*）→ fake server。

import (
	"context"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	authv1 "github.com/CycleZero/ley/api/auth/v1"
	blogv1 "github.com/CycleZero/ley/api/blog/v1"
	"github.com/CycleZero/ley/app/entry/conf"
	"github.com/CycleZero/ley/app/entry/internal/domain/proxy"
	"github.com/CycleZero/ley/app/entry/internal/router"
	commonconf "github.com/CycleZero/ley/conf"
	jwtpkg "github.com/CycleZero/ley/pkg/jwt"
	"github.com/CycleZero/ley/pkg/log"
	"github.com/CycleZero/ley/pkg/testutil/datatest"

	"github.com/gin-gonic/gin"
	kratosmd "github.com/go-kratos/kratos/v2/middleware/metadata"
	grpcx "github.com/go-kratos/kratos/v2/transport/grpc"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/test/bufconn"
)

// e2eSigningKey 测试签名密钥（>=32 字节，对齐生产的 256 位要求）。
const e2eSigningKey = "entry-e2e-test-secret-0123456789abcdef"

// e2eAccessTTL 默认 access token 有效期（15 分钟，与生产约定一致）。
const e2eAccessTTL = 15 * time.Minute

// e2eClientIP 所有 E2E 请求统一携带的 X-Forwarded-For（真实 IP 透传断言基准）。
const e2eClientIP = "203.0.113.9"

// e2eEnv 一套完整 E2E 环境：fake servers + ServiceHub + 手动装配的 MainApp。
type e2eEnv struct {
	t         *testing.T
	jwt       jwtpkg.JWT            // 测试 JWT 解析器（签发测试令牌用）
	blacklist jwtpkg.BlackListCache // 启用的内存黑名单（吊销令牌场景预置用）
	fakeAuth  *e2eFakeAuth          // fake auth server 句柄（快照断言）
	fakeBlog  *e2eFakeBlog          // fake blog server 句柄（快照断言）
	app       *MainApp              // 完整应用（Engine 即被测入口）
}

// newE2EEnv 装配一套完整 E2E 环境（每测试独立，互不串扰）：
//
//	accessTTL：JWT access 有效期；传负值（如 -time.Minute）可签发即过期的令牌
//	（供「过期 token 401」场景——需在装配前决定，故作为参数而非事后换 JWT）。
//
// 组装步骤：
//  1. bufconn 监听 + grpc server，注册 fakeAuth/fakeBlog（Tag/Article/Site 三服务）；
//  2. 两条 client 连接（auth/blog 各一）：拨号参数与生产 dialServiceConn 对齐
//     （kratos metadata.Client() 中间件 + 3s 超时），仅把服务发现换成 bufconn
//     拨号器——元数据透传断言因此与生产链路等价；
//  3. 手动 NewMainApp：JWT（etcd 远程配置由 holder 降级默认，测试直接构造解析器）、
//     内存黑名单（IsEnabled=true，验证黑名单前置拦截）、holder 零值配置（rps=0
//     默认不挂限流、CORS 放行所有来源）。
//
// 清理由 t.Cleanup 完成（conn 关闭 → grpc server 停止 → holder watch 停止）。
func newE2EEnv(t *testing.T, accessTTL time.Duration) *e2eEnv {
	t.Helper()
	gin.SetMode(gin.TestMode)

	if accessTTL == 0 {
		accessTTL = e2eAccessTTL
	}

	// 1. bufconn：进程内监听，替代真实 etcd 服务发现
	lis := bufconn.Listen(1 << 20)
	fakeAuth := &e2eFakeAuth{}
	fakeBlog := &e2eFakeBlog{}
	srv := grpc.NewServer()
	authv1.RegisterAuthServiceServer(srv, fakeAuth)
	blogv1.RegisterArticleServiceServer(srv, fakeBlog)
	blogv1.RegisterTagServiceServer(srv, fakeBlog)
	blogv1.RegisterSiteServiceServer(srv, fakeBlog)
	// Serve 阻塞运行；t.Cleanup 中 srv.Stop() 使其返回（ErrServerStopped 属正常关闭）
	go func() { _ = srv.Serve(lis) }()
	t.Cleanup(func() { srv.Stop() })

	// 2. 与生产同款拨号的 client 连接（中间件链一致，透传断言才真实有效）
	authConn := dialBufconn(t, lis, "auth")
	blogConn := dialBufconn(t, lis, "blog")
	hub := &proxy.ServiceHub{
		Auth:     authv1.NewAuthServiceClient(authConn),
		Article:  blogv1.NewArticleServiceClient(blogConn),
		Tag:      blogv1.NewTagServiceClient(blogConn),
		Category: blogv1.NewCategoryServiceClient(blogConn),
		File:     blogv1.NewFileServiceClient(blogConn),
		Site:     blogv1.NewSiteServiceClient(blogConn),
	}

	// 3. 手动装配 MainApp（不走 wire：真实 initApp 依赖 etcd，单测无法 fake）
	jw := jwtpkg.NewJWT(&jwtpkg.Config{
		SigningKey:         e2eSigningKey,
		Issuer:             "entry-e2e",
		ExpiredTime:        accessTTL,
		RefreshExpiredTime: accessTTL * 7,
	})
	blacklist := jwtpkg.NewBlackList(datatest.NewInMemoryCache())
	if !blacklist.IsEnabled() {
		t.Fatalf("测试前置失败：黑名单应处于启用状态")
	}
	// etcd 为空 → 降级为默认配置 holder（rps=0 无限流、CORS 放行所有来源）
	holder, stop := conf.NewRemoteConfigHolder(nil)
	t.Cleanup(stop)
	reg := router.NewRegisterMiddleWire(jw, blacklist, holder)
	// Prod 日志模式：gin ReleaseMode，抑制路由调试输出；请求链路与 Dev 完全一致
	bc := &commonconf.Bootstrap{Log: &commonconf.Log{Mode: commonconf.LogMode_Prod}}
	app := NewMainApp(bc, hub, router.NewRegisterFunc(), reg)

	return &e2eEnv{
		t: t, jwt: jw, blacklist: blacklist,
		fakeAuth: fakeAuth, fakeBlog: fakeBlog, app: app,
	}
}

// dialBufconn 拨号 bufconn 监听器。
//
// 复刻 infra/grpc.go dialServiceConn 的关键中间件（kratosmd.Client() 透传
// x-md-global-*）与 3s 超时；endpoint 走 passthrough 解析 + 自定义 dialer，
// 使连接在进程内完成（模式同 domain/proxy/helpers_test.go，跨包复制）。
func dialBufconn(t *testing.T, lis *bufconn.Listener, name string) *grpc.ClientConn {
	t.Helper()
	conn, err := grpcx.DialInsecure(context.Background(),
		grpcx.WithEndpoint("passthrough:///"+name),
		grpcx.WithMiddleware(kratosmd.Client()),
		grpcx.WithTimeout(3*time.Second),
		grpcx.WithOptions(grpc.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) {
			return lis.DialContext(ctx)
		})),
	)
	if err != nil {
		t.Fatalf("拨号 bufconn(%s) 失败: %v", name, err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	return conn
}

// ===================== 请求/响应辅助 =====================

// e2eEnvelope 测试用信封结构（对齐 common.Response：{code,msg,data}）。
type e2eEnvelope struct {
	Code int             `json:"code"`
	Msg  string          `json:"msg"`
	Data json.RawMessage `json:"data"`
}

// e2eDo 向被测应用引擎发起请求。
//
//	token：Authorization: Bearer <token>；空串则不携带认证头；
//	body：请求体 JSON；空串则无请求体。
//
// 统一携带 X-Forwarded-For: e2eClientIP（真实客户端 IP 透传断言基准）。
func e2eDo(t *testing.T, env *e2eEnv, method, path, token, body string) *httptest.ResponseRecorder {
	t.Helper()
	var rdr io.Reader
	if body != "" {
		rdr = strings.NewReader(body)
	}
	req := httptest.NewRequest(method, path, rdr)
	req.Header.Set("X-Forwarded-For", e2eClientIP)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	w := httptest.NewRecorder()
	env.app.Engine.ServeHTTP(w, req)
	return w
}

// decodeEnvelope 解析响应为信封结构。
func decodeEnvelope(t *testing.T, w *httptest.ResponseRecorder) e2eEnvelope {
	t.Helper()
	var env e2eEnvelope
	if err := json.Unmarshal(w.Body.Bytes(), &env); err != nil {
		t.Fatalf("解析响应体失败: %v，body=%s", err, w.Body.String())
	}
	return env
}

// dataStr 将信封 data 转为字符串（失败信封 data=null 时返回 "null"）。
func dataStr(t *testing.T, env e2eEnvelope) string {
	t.Helper()
	if len(env.Data) == 0 {
		t.Fatalf("信封缺少 data 字段：%+v", env)
	}
	return string(env.Data)
}

// wantOK 断言 200 + 成功信封 {code:0,msg:ok}。
func wantOK(t *testing.T, w *httptest.ResponseRecorder) e2eEnvelope {
	t.Helper()
	if w.Code != http.StatusOK {
		t.Fatalf("HTTP 状态 = %d, 期望 200，body=%s", w.Code, w.Body.String())
	}
	env := decodeEnvelope(t, w)
	if env.Code != 0 || env.Msg != "ok" {
		t.Fatalf("成功信封 = {code:%d msg:%q}, 期望 {code:0 msg:ok}，body=%s", env.Code, env.Msg, w.Body.String())
	}
	return env
}

// wantFail 断言失败信封：HTTP 状态与业务码镜像一致（T3 约定 code 镜像 httpStatus），
// 消息精确匹配、data 为 null。
func wantFail(t *testing.T, w *httptest.ResponseRecorder, status int, msg string) {
	t.Helper()
	if w.Code != status {
		t.Fatalf("HTTP 状态 = %d, 期望 %d，body=%s", w.Code, status, w.Body.String())
	}
	env := decodeEnvelope(t, w)
	if env.Code != status {
		t.Fatalf("业务码 = %d, 期望 %d（镜像 HTTP 状态），body=%s", env.Code, status, w.Body.String())
	}
	if env.Msg != msg {
		t.Fatalf("错误消息 = %q, 期望 %q，body=%s", env.Msg, msg, w.Body.String())
	}
	if string(env.Data) != "null" {
		t.Fatalf("失败信封 data 应为 null，实际 %s，body=%s", env.Data, w.Body.String())
	}
}

// wantCommonHeaders 断言请求真实走完整全局链：RequestID + CORS 响应头齐备。
// （globalMiddleWires = RequestLogger → AddMetaData → CORS，缺任一即断言失败）
func wantCommonHeaders(t *testing.T, w *httptest.ResponseRecorder) {
	t.Helper()
	if got := w.Header().Get("X-Request-ID"); got == "" {
		t.Fatalf("响应应携带 X-Request-ID（AddMetaData 中间件产出）")
	}
	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "*" {
		t.Fatalf("默认 CORS（空来源列表）应放行所有来源，实际 %q", got)
	}
}

// issueToken 用 env 的 JWT 签发指定身份的 access token。
func issueToken(t *testing.T, env *e2eEnv, uid uint64, name, role string) string {
	t.Helper()
	token, err := env.jwt.GenerateToken(jwtpkg.Payload{UserId: uid, UserName: name, Role: role})
	if err != nil {
		t.Fatalf("签发测试令牌失败: %v", err)
	}
	return token
}

// ===================== 测试日志初始化 =====================

// TestMain 初始化全局日志器为丢弃日志（对齐 conf/infra/router 测试约定：
// pkg/log.GetLogger 未初始化时无日志兜底，NewMainApp.printRoutes 会落日志）。
func TestMain(m *testing.M) {
	log.SetGlobalLogger(&log.Logger{Logger: zap.NewNop()})
	gin.SetMode(gin.TestMode)
	os.Exit(m.Run())
}
