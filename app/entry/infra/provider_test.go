package infra

import (
	"os"
	"testing"

	"github.com/CycleZero/ley/app/entry/conf"
	jwtpkg "github.com/CycleZero/ley/pkg/jwt"
	"github.com/CycleZero/ley/pkg/log"
	"github.com/go-kratos/kratos/v2/registry"
	clientv3 "go.etcd.io/etcd/client/v3"
	"go.uber.org/zap"
)

// TestMain 初始化全局日志器为丢弃日志（对齐生产「先 SetGlobalLogger 后使用」的顺序；
// pkg/log.GetLogger 在未初始化时直接 panic；redis 降级路径会输出 Warn/Info）
func TestMain(m *testing.M) {
	log.SetGlobalLogger(&log.Logger{Logger: zap.NewNop()})
	os.Exit(m.Run())
}

// =====================  编译期/签名断言（不依赖真实 etcd，-short 可跑） =====================

// TestProviderSignatures 以编译期函数值赋值断言各 Provider 的静态签名，
// 防止后续改动破坏 wire 绑定所需类型（如 NewDiscovery 返回值类型）。
//
// 本测试不执行任何 Provider（均需真实 etcd 连接），仅验证签名与类型契约。
func TestProviderSignatures(t *testing.T) {
	var (
		_ func(*clientv3.Client) registry.Discovery                 = NewDiscovery
		_ func(registry.Discovery) (*AuthClientConn, func(), error) = NewAuthClientConn
		_ func(registry.Discovery) (*BlogClientConn, func(), error) = NewBlogClientConn
		_ func(conf.RedisConfig) jwtpkg.BlackListCache              = NewBlacklistCache
		_ func(*clientv3.Client) (*conf.RemoteConfigHolder, func()) = conf.NewRemoteConfigHolder
	)
	// 编译通过即契约成立；运行期无事可做
}

// =====================  黑名单降级路径（无需真实 Redis，-short 可跑） =====================

func TestNewBlacklistCacheDisabledWhenNoHost(t *testing.T) {
	// Given 未配置 Redis 地址（dev 无 redis 的典型形态）
	bl := NewBlacklistCache(conf.RedisConfig{})

	// Then 返回禁用的黑名单：IsEnabled=false 且各方法安全（nil cache 语义）
	if bl.IsEnabled() {
		t.Fatal("未配置 Redis 时黑名单应处于禁用态")
	}
	if bl.IsTokenBlackListed("any-token") {
		t.Fatal("禁用态不应判定任何 token 已拉黑")
	}
}

func TestNewBlacklistCacheDisabledWhenRedisUnreachable(t *testing.T) {
	// Given 配置了 Redis 地址但指向不可达端口（127.0.0.1:1 拒绝连接，即时失败）
	bl := NewBlacklistCache(conf.RedisConfig{Host: "127.0.0.1", Port: 1})

	// Then 探测失败后降级为禁用态（IsEnabled=false），而非启动即报错
	if bl.IsEnabled() {
		t.Fatal("Redis 不可达时黑名单应降级为禁用态")
	}
}
