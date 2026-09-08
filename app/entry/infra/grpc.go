package infra

import (
	"context"
	"time"

	"github.com/CycleZero/ley/pkg/metrics"
	"github.com/CycleZero/ley/pkg/util"
	kratosmd "github.com/go-kratos/kratos/v2/middleware/metadata"
	kratostracing "github.com/go-kratos/kratos/v2/middleware/tracing"
	"github.com/go-kratos/kratos/v2/registry"
	grpcx "github.com/go-kratos/kratos/v2/transport/grpc"
	"google.golang.org/grpc"
)

// blogMaxMsgSize Blog 服务消息上限：32MiB。
//
// 博客文章内容较长且文件上传走 base64 载荷（HTTP → gRPC 转发），
// 需超过 gRPC 默认 4MiB 接收上限，收发两侧同步放大。
const blogMaxMsgSize = 32 << 20

// dialServiceConn 按服务发现拨号 gRPC 连接（auth/blog 共用骨架）。
//
// ⚠️ 不复用 pkg/util.NewGrpcConn 的原因（关键设计点）：
// pkg/util.NewGrpcConn 只挂 discovery + endpoint，不带 kratos metadata 客户端中间件；
// 而 entry handler 内 meta.NewClientCtx 注入的用户上下文（x-md-global- 前缀）
// 必须经 metadata.Client() 中间件（默认透传 x-md-global- 前缀，见 kratos
// middleware/metadata 源码）序列化到 gRPC wire header，下游 auth/blog 才能收到。
// 因此此处显式挂载 kratosmd.Client()（本 wave 骨架即埋入，避免后续 wave 逐一补齐）。
func dialServiceConn(dis registry.Discovery, endpoint string, extras ...grpcx.ClientOption) (*grpc.ClientConn, error) {
	opts := []grpcx.ClientOption{
		grpcx.WithDiscovery(dis),
		grpcx.WithEndpoint(endpoint),
		// 链路续接：把当前请求 span 上下文注入 gRPC metadata（traceparent），
		// 下游 auth/blog 据此挂到同一 trace 上；metrics 记录服务间调用指标。
		grpcx.WithMiddleware(kratostracing.Client(), metrics.ClientMiddleware(), kratosmd.Client()),
		// 单次调用超时（kratos interceptor 语义），防下游无响应时请求悬挂
		grpcx.WithTimeout(3 * time.Second),
	}
	opts = append(opts, extras...)
	return grpcx.DialInsecure(context.Background(), opts...)
}

// =====================  连接命名包装（wire 绑定消歧） =====================

// AuthClientConn auth 服务（ley.auth）gRPC 连接。
//
// 与 BlogClientConn 同源于 *grpc.ClientConn：wire 按静态类型绑定 Provider，
// 两个同类型 Provider 会造成绑定歧义，故以命名结构包装（见 provider.go InfraProviderSet）。
// 消费方通过 Conn() 取出底层连接构建 authv1.AuthClient 等生成客户端。
type AuthClientConn struct {
	conn *grpc.ClientConn
}

// Conn 返回底层 *grpc.ClientConn（构建 api 生成客户端时使用）。
func (c *AuthClientConn) Conn() *grpc.ClientConn { return c.conn }

// Close 关闭连接（wire cleanup 链调用）。
func (c *AuthClientConn) Close() error { return c.conn.Close() }

// BlogClientConn blog 服务（ley.blog）gRPC 连接（作用同 AuthClientConn，独立类型用于 wire 消歧）。
type BlogClientConn struct {
	conn *grpc.ClientConn
}

// Conn 返回底层 *grpc.ClientConn（构建 blogv1.BlogClient 等生成客户端时使用）。
func (c *BlogClientConn) Conn() *grpc.ClientConn { return c.conn }

// Close 关闭连接（wire cleanup 链调用）。
func (c *BlogClientConn) Close() error { return c.conn.Close() }

// NewAuthClientConn 创建 auth 服务连接。
//
// 服务名 ley.auth 与 pkg/util.DisServiceName("auth") 生成的注册名一致
// （DiscoveryEndpoint 统一走 discovery:// 协议）。
func NewAuthClientConn(dis registry.Discovery) (*AuthClientConn, func(), error) {
	conn, err := dialServiceConn(dis, util.DiscoveryEndpoint("ley.auth"))
	if err != nil {
		return nil, nil, err
	}
	client := &AuthClientConn{conn: conn}
	return client, func() { _ = client.Close() }, nil
}

// NewBlogClientConn 创建 blog 服务连接（ley.blog）。
//
// 在公共拨号参数之上追加 32MiB 消息上限（文章内容/文件 base64 载荷，见 blogMaxMsgSize）。
func NewBlogClientConn(dis registry.Discovery) (*BlogClientConn, func(), error) {
	conn, err := dialServiceConn(dis, util.DiscoveryEndpoint("ley.blog"),
		grpcx.WithOptions(grpc.WithDefaultCallOptions(
			grpc.MaxCallRecvMsgSize(blogMaxMsgSize),
			grpc.MaxCallSendMsgSize(blogMaxMsgSize),
		)),
	)
	if err != nil {
		return nil, nil, err
	}
	client := &BlogClientConn{conn: conn}
	return client, func() { _ = client.Close() }, nil
}
