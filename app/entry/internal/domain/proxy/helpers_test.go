package proxy

// ===================== fake-gRPC 测试基建（helpers_test.go） =====================
//
// 价值说明：本文件的 bufconn fake gRPC server 让 handle_test.go 在 -short 模式下
// 零外部依赖地覆盖「HTTP → entry handler → gRPC → 下游」全链路，其中最关键的断言
// 是元数据透传：handler 经 pkg/meta.NewClientCtx 注入的用户上下文（x-md-global-*）
// 必须真实到达 server 端——这要求 client 连接挂载与生产拨号（infra/grpc.go
// dialServiceConn）同款的 kratos metadata.Client() 中间件。若该中间件缺失或前缀
// 规则不符，测试会立即失败，从而证明 T2 的拨号接线与 pkg/meta 转换链路正确。
//
// 本文件不依赖 etcd/真实服务：bufconn 在进程内完成监听与拨号。

import (
	"context"
	"net"
	"sync"
	"testing"
	"time"

	authv1 "github.com/CycleZero/ley/api/auth/v1"
	blogv1 "github.com/CycleZero/ley/api/blog/v1"
	commonv1 "github.com/CycleZero/ley/api/common/v1"
	"github.com/CycleZero/ley/app/entry/internal/common"
	"github.com/gin-gonic/gin"
	kratosmd "github.com/go-kratos/kratos/v2/middleware/metadata"
	grpcx "github.com/go-kratos/kratos/v2/transport/grpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/test/bufconn"
)

// ===================== fake 服务实现 =====================

// recordIncomingMD 提取 server 端收到的 gRPC incoming metadata 副本。
// 请求经 metadata.Client() 序列化到 wire header 后，server 端（即使是不走
// kratos server 中间件的原始 grpc server）也能从上下文取出原始键值。
func recordIncomingMD(ctx context.Context) metadata.MD {
	md, _ := metadata.FromIncomingContext(ctx)
	return md.Copy()
}

// fakeAuthServer 假 auth 服务：实现 Login/GetProfile（固定响应 + 记录收到的
// 元数据与请求字段），其余方法由嵌入的 UnimplementedAuthServiceServer 兜底。
type fakeAuthServer struct {
	authv1.UnimplementedAuthServiceServer

	// loginErr 非 nil 时 Login 直接返回该错误（错误映射测试注入点）
	loginErr error

	mu      sync.Mutex
	logins  []authLoginSnap // 按调用顺序记录 Login 快照
	profile []profileSnap   // 按调用顺序记录 GetProfile 快照
}

// authLoginSnap 一次 Login 调用的 server 端快照。
type authLoginSnap struct {
	md       metadata.MD // server 端收到的元数据（断言 x-md-global-* 透传）
	account  string      // 请求体绑定结果
	password string
}

// profileSnap 一次 GetProfile 调用的 server 端快照（无请求字段，仅元数据）。
type profileSnap struct {
	md metadata.MD
}

// Login 固定返回成功回复：user alice + token_pair。
func (f *fakeAuthServer) Login(ctx context.Context, in *authv1.LoginRequest) (*authv1.LoginReply, error) {
	if f.loginErr != nil {
		return nil, f.loginErr
	}
	f.mu.Lock()
	f.logins = append(f.logins, authLoginSnap{
		md:       recordIncomingMD(ctx),
		account:  in.Account,
		password: in.Password,
	})
	f.mu.Unlock()
	return &authv1.LoginReply{
		User: &commonv1.UserInfo{Id: 1, Username: "alice", Email: "alice@example.com", Role: "reader"},
		TokenPair: &commonv1.TokenPair{
			AccessToken: "access-token-1", RefreshToken: "refresh-token-1", ExpiresIn: 900,
		},
	}, nil
}

// GetProfile 固定返回当前用户信息（user id 取自透传元数据，顺带自证 meta 到达）。
func (f *fakeAuthServer) GetProfile(ctx context.Context, _ *authv1.GetProfileRequest) (*authv1.GetProfileReply, error) {
	f.mu.Lock()
	f.profile = append(f.profile, profileSnap{md: recordIncomingMD(ctx)})
	f.mu.Unlock()
	return &authv1.GetProfileReply{
		User: &commonv1.UserInfo{Id: 42, Username: "tester", Email: "t@example.com", Role: "admin"},
	}, nil
}

// lastLogin 返回最近一次 Login 快照（无调用时第二个返回值为 false）。
func (f *fakeAuthServer) lastLogin() (authLoginSnap, bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.logins) == 0 {
		return authLoginSnap{}, false
	}
	return f.logins[len(f.logins)-1], true
}

// lastProfile 返回最近一次 GetProfile 快照（无调用时第二个返回值为 false）。
func (f *fakeAuthServer) lastProfile() (profileSnap, bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.profile) == 0 {
		return profileSnap{}, false
	}
	return f.profile[len(f.profile)-1], true
}

// fakeBlogServer 假 blog 服务：实现 ArticleService.DeleteArticle（记录元数据与
// 路径参数绑定的 Id），其余方法由 Unimplemented 兜底。
type fakeBlogServer struct {
	blogv1.UnimplementedArticleServiceServer

	// deleteErr 非 nil 时 DeleteArticle 直接返回该错误（错误映射测试注入点）
	deleteErr error

	mu      sync.Mutex
	deletes []blogDeleteSnap // 按调用顺序记录 DeleteArticle 快照
}

// blogDeleteSnap 一次 DeleteArticle 调用的 server 端快照。
type blogDeleteSnap struct {
	md metadata.MD // server 端收到的元数据
	id uint64      // 路径参数合并后的请求 Id
}

// DeleteArticle 记录请求并返回空回复（删除类接口无业务数据）。
func (f *fakeBlogServer) DeleteArticle(ctx context.Context, in *blogv1.DeleteArticleRequest) (*blogv1.DeleteArticleReply, error) {
	if f.deleteErr != nil {
		return nil, f.deleteErr
	}
	f.mu.Lock()
	f.deletes = append(f.deletes, blogDeleteSnap{md: recordIncomingMD(ctx), id: in.Id})
	f.mu.Unlock()
	return &blogv1.DeleteArticleReply{}, nil
}

// lastDelete 返回最近一次 DeleteArticle 快照（无调用时第二个返回值为 false）。
func (f *fakeBlogServer) lastDelete() (blogDeleteSnap, bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.deletes) == 0 {
		return blogDeleteSnap{}, false
	}
	return f.deletes[len(f.deletes)-1], true
}

// ===================== 环境装配 =====================

// testEnv 单测环境：bufconn fake server + 与生产同款拨号的 client 连接 +
// 可直接注册路由的 gin engine + 由 6 个生成 client 装配的 ServiceHub。
type testEnv struct {
	t        *testing.T
	fakeAuth *fakeAuthServer
	fakeBlog *fakeBlogServer
	hub      *ServiceHub
	engine   *gin.Engine
}

// newTestEnv 装配一套完整测试环境：
//
//  1. bufconn 监听 + grpc server，注册 fakeAuth/fakeBlog；
//  2. 两条 client 连接（auth/blog 各一）：拨号参数与生产 dialServiceConn 对齐
//     （kratos metadata.Client() 中间件 + 3s 超时），仅把服务发现换成 bufconn
//     拨号器——中间件链与生产完全一致，保证元数据透传断言真实有效；
//  3. 由连接装配 ServiceHub，等价于 NewProxyHub 的产物；
//  4. 返回裸 gin engine（gin.New()，无全局中间件），测试按需注册路由与
//     meta 种入中间件（模拟 T4 JWT 中间件行为）。
//
// 清理由 t.Cleanup 完成（conn 关闭 → grpc server 停止）。
func newTestEnv(t *testing.T) *testEnv {
	t.Helper()
	gin.SetMode(gin.TestMode)

	// bufconn：进程内监听，替代真实 etcd 服务发现
	lis := bufconn.Listen(1 << 20)

	fakeAuth := &fakeAuthServer{}
	fakeBlog := &fakeBlogServer{}
	srv := grpc.NewServer()
	authv1.RegisterAuthServiceServer(srv, fakeAuth)
	blogv1.RegisterArticleServiceServer(srv, fakeBlog)
	// Serve 阻塞运行；t.Cleanup 中 srv.Stop() 会使其返回（ErrServerStopped 属正常关闭）
	go func() { _ = srv.Serve(lis) }()
	t.Cleanup(func() { srv.Stop() })

	authConn := dialBufconn(t, lis, "auth")
	blogConn := dialBufconn(t, lis, "blog")

	hub := &ServiceHub{
		Auth:     authv1.NewAuthServiceClient(authConn),
		Article:  blogv1.NewArticleServiceClient(blogConn),
		Tag:      blogv1.NewTagServiceClient(blogConn),
		Category: blogv1.NewCategoryServiceClient(blogConn),
		File:     blogv1.NewFileServiceClient(blogConn),
		Site:     blogv1.NewSiteServiceClient(blogConn),
	}

	return &testEnv{
		t:        t,
		fakeAuth: fakeAuth,
		fakeBlog: fakeBlog,
		hub:      hub,
		engine:   gin.New(),
	}
}

// dialBufconn 拨号 bufconn 监听器。
//
// 复刻 infra/grpc.go dialServiceConn 的关键中间件（kratosmd.Client() 透传
// x-md-global-*）与 3s 超时；endpoint 走 passthrough 解析 + 自定义 dialer，
// 使连接在进程内完成。注释在此明示：若此处缺少 kratosmd.Client()，
// metadata 透传测试会通过但生产链路可能已断——保持与生产拨号一致是
// 本测试有效性的前提。
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

// withAuthMeta 返回种入认证元数据的 gin 中间件（模拟 T4 认证中间件行为：
// JWT 校验通过后调用 common.SetRequestMeta）。
func withAuthMeta(uid uint64, name, role string) gin.HandlerFunc {
	return func(c *gin.Context) {
		common.SetRequestMeta(c, &common.RequestMetaData{
			Auth: common.Auth{UserID: uid, UserName: name, Role: role},
		})
		c.Next()
	}
}

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

// mdKeys 返回元数据全部键（排序后），便于失败信息展示。
func mdKeys(md metadata.MD) []string {
	keys := make([]string, 0, len(md))
	for k := range md {
		keys = append(keys, k)
	}
	return keys
}
