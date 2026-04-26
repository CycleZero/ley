// Package testutil 提供跨微服务公用的集成测试基础设施。
//
// 用法（以 user 服务为例）：
//
//	func TestRegister(t *testing.T) {
//	    s := testutil.NewSuite(t)
//	    resp := s.POST(t, "/api/auth/register", map[string]any{
//	        "username": "e2e_test_user",
//	        "password": "Aa123456",
//	    })
//	    require.Equal(t, 200, resp.StatusCode)
//	}
//
// 环境变量：
//   - GATEWAY_URL   网关 HTTP 地址（如 http://127.0.0.1:8000）
//   - ETCD_ADDR     etcd 地址（如 127.0.0.1:2379）
//   - SERVICE_NAME  微服务注册名（如 vcyuan-backend-app-user）
//   - GATEWAY_TIMEOUT HTTP 请求超时（默认 30s）
package testutil

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	etcdregistry "github.com/go-kratos/kratos/contrib/registry/etcd/v2"
	grpcx "github.com/go-kratos/kratos/v2/transport/grpc"
	"go.etcd.io/etcd/client/v3"
	"google.golang.org/grpc"
)

// Suite 公共集成测试套件。
// 每个微服务的集成测试通过 NewSuite 创建，共享 HTTP 客户端和 etcd gRPC 连接。
type Suite struct {
	// GatewayURL 网关 HTTP 地址，从环境变量 GATEWAY_URL 读取
	GatewayURL string
	// EtcdAddrs etcd 地址列表，从环境变量 ETCD_ADDR 读取（逗号分隔）
	EtcdAddrs []string
	// ServiceName 微服务注册名，从环境变量 SERVICE_NAME 读取
	ServiceName string
	// HTTPClient 可复用的 HTTP 客户端
	HTTPClient *http.Client

	// 内部状态
	etcdClient *clientv3.Client
	discovery  *etcdregistry.Registry
	grpcConn   *grpc.ClientConn
}

// NewSuite 创建集成测试套件。
// 如果必要环境变量未设置，自动 t.Skip。
func NewSuite(t *testing.T) *Suite {
	t.Helper()

	gw := envOr("GATEWAY_URL", "")
	etcdAddr := envOr("ETCD_ADDR", "")
	svcName := envOr("SERVICE_NAME", "")
	timeoutStr := envOr("GATEWAY_TIMEOUT", "30s")

	if gw == "" {
		t.Skip("GATEWAY_URL 未设置，跳过集成测试")
	}

	timeout, err := time.ParseDuration(timeoutStr)
	if err != nil {
		timeout = 30 * time.Second
	}

	s := &Suite{
		GatewayURL:  strings.TrimRight(gw, "/"),
		EtcdAddrs:   strings.Split(etcdAddr, ","),
		ServiceName: svcName,
		HTTPClient: &http.Client{
			Timeout: timeout,
			Transport: &http.Transport{
				MaxIdleConns:    100,
				IdleConnTimeout: 90 * time.Second,
			},
		},
	}

	// etcd gRPC 连接（仅 ETCD_ADDR 设置时生效）
	if etcdAddr != "" && svcName != "" {
		s.initEtcdGRPC(t)
	}

	return s
}

// initEtcdGRPC 初始化 etcd 服务发现连接。
func (s *Suite) initEtcdGRPC(t *testing.T) {
	t.Helper()

	var err error
	s.etcdClient, err = clientv3.New(clientv3.Config{
		Endpoints:   s.EtcdAddrs,
		DialTimeout: 5 * time.Second,
	})
	if err != nil {
		t.Skipf("etcd 连接失败: %v", err)
	}
	s.discovery = etcdregistry.New(s.etcdClient)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	s.grpcConn, err = grpcx.DialInsecure(ctx,
		grpcx.WithEndpoint("discovery:///"+s.ServiceName),
		grpcx.WithDiscovery(s.discovery),
	)
	if err != nil {
		t.Skipf("gRPC 服务发现连接失败: %v", err)
	}
}

// Close 释放资源。
func (s *Suite) Close() {
	if s.grpcConn != nil {
		s.grpcConn.Close()
	}
	if s.etcdClient != nil {
		s.etcdClient.Close()
	}
}

// GRPCConn 返回 etcd 服务发现的 gRPC 连接。
// 仅在 ETCD_ADDR 和 SERVICE_NAME 都设置时可用，否则 panic。
func (s *Suite) GRPCConn() *grpc.ClientConn {
	if s.grpcConn == nil {
		panic("gRPC 连接未初始化：请设置 ETCD_ADDR 和 SERVICE_NAME 环境变量")
	}
	return s.grpcConn
}

// ============================================================
// HTTP 请求方法
// ============================================================

// GET 发送 HTTP GET 请求过网关。
// token 为空时不带 Authorization 头。
func (s *Suite) GET(t *testing.T, path string, token string) *http.Response {
	t.Helper()
	return s.doRequest(t, http.MethodGet, path, nil, token)
}

// POST 发送 HTTP POST 请求过网关。
// body 会被序列化为 JSON。
func (s *Suite) POST(t *testing.T, path string, body any, token string) *http.Response {
	t.Helper()
	return s.doRequest(t, http.MethodPost, path, body, token)
}

// PUT 发送 HTTP PUT 请求过网关。
func (s *Suite) PUT(t *testing.T, path string, body any, token string) *http.Response {
	t.Helper()
	return s.doRequest(t, http.MethodPut, path, body, token)
}

// DELETE 发送 HTTP DELETE 请求过网关。
func (s *Suite) DELETE(t *testing.T, path string, token string) *http.Response {
	t.Helper()
	return s.doRequest(t, http.MethodDelete, path, nil, token)
}

func (s *Suite) doRequest(t *testing.T, method, path string, body any, token string) *http.Response {
	t.Helper()

	url := s.GatewayURL + path
	var reqBody io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("序列化请求体失败: %v", err)
		}
		reqBody = bytes.NewReader(data)
	}

	req, err := http.NewRequest(method, url, reqBody)
	if err != nil {
		t.Fatalf("创建 HTTP 请求失败: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := s.HTTPClient.Do(req)
	if err != nil {
		t.Fatalf("HTTP 请求失败: %v", err)
	}
	return resp
}

// ============================================================
// 响应解析
// ============================================================

// ParseBody 解析 HTTP 响应体为 map。
func ParseBody(t *testing.T, resp *http.Response) map[string]any {
	t.Helper()
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("读取响应体失败: %v", err)
	}
	var result map[string]any
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("解析 JSON 响应失败 (body=%s): %v", string(data), err)
	}
	return result
}

// ReadBody 读取原始响应体字符串。
func ReadBody(t *testing.T, resp *http.Response) string {
	t.Helper()
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("读取响应体失败: %v", err)
	}
	return string(data)
}

// ============================================================
// 辅助函数
// ============================================================

// envOr 返回环境变量值，未设置时返回 fallback。
func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// RandomString 生成长度为 n 的随机字母串，用于测试资源命名。
func RandomString(n int) string {
	const letters = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, n)
	for i := range b {
		b[i] = letters[time.Now().UnixNano()%int64(len(letters))]
		time.Sleep(1) // 避免纳秒碰撞
	}
	return string(b)
}

// StrVal 从 map[string]any 中安全提取 string 值。
func StrVal(m map[string]any, key string) string {
	if m == nil {
		return ""
	}
	v, _ := m[key].(string)
	return v
}

// FloatVal 从 map[string]any 中安全提取 float64 值（JSON 数字默认解析为 float64）。
func FloatVal(m map[string]any, key string) float64 {
	if m == nil {
		return 0
	}
	v, _ := m[key].(float64)
	return v
}
