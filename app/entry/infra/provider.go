package infra

import (
	"errors"
	"fmt"

	"github.com/CycleZero/ley/app/entry/conf"
	commonconf "github.com/CycleZero/ley/conf"
	pkginfra "github.com/CycleZero/ley/pkg/infra"
	"github.com/google/wire"
	clientv3 "go.etcd.io/etcd/client/v3"
)

// InfraProviderSet 基础设施层 Wire ProviderSet。
//
// 绑定关系（依赖方向：引导配置 → etcd → 服务发现/远程配置 → 连接/黑名单）：
//
//	bc *commonconf.Bootstrap
//	 ├─ ProvideEtcdClient ──────────► *clientv3.Client
//	 │    ├─ NewDiscovery ──────────► registry.Discovery ──► NewAuthClientConn/NewBlogClientConn
//	 │    └─ conf.NewRemoteConfigHolder ──► *conf.RemoteConfigHolder
//	 │          └─ ProvideRedisSnapshot ──► conf.RedisConfig ──► NewBlacklistCache
//	 └─（wire cleanup 链：holder stop → etcd client Close，逆序执行）
//
// 各连接 Provider 返回独立命名类型（AuthClientConn/BlogClientConn），
// 避免两个同源 *grpc.ClientConn 在 wire 绑定中产生歧义。
var InfraProviderSet = wire.NewSet(
	ProvideEtcdClient,
	NewDiscovery,
	NewAuthClientConn,
	NewBlogClientConn,
	ProvideRedisSnapshot,
	NewBlacklistCache,
	conf.NewRemoteConfigHolder,
)

// ProvideEtcdClient 从引导配置创建 etcd 客户端（wire Provider，带 cleanup）。
//
// 引导配置缺少 etcd.endpoints 时视为启动配置错误（与 auth main.go 的处理一致，panic 语义
// 收敛到此处返回错误，由 wire cleanup 保证异常路径不泄漏资源）。
func ProvideEtcdClient(bc *commonconf.Bootstrap) (*clientv3.Client, func(), error) {
	if bc == nil || bc.Etcd == nil || len(bc.Etcd.Endpoints) == 0 {
		return nil, nil, errors.New("引导配置缺少 etcd.endpoints，无法初始化 etcd 客户端")
	}
	// pkg/infra.NewEtcdClient 内部失败时 panic（既有约定），此处不复写其语义
	etcdClient := pkginfra.NewEtcdClient(bc.Etcd.Endpoints)
	cleanup := func() {
		if err := etcdClient.Close(); err != nil {
			fmt.Printf("[entry/infra] 关闭 etcd 客户端失败：%v\n", err)
		}
	}
	return etcdClient, cleanup, nil
}

// ProvideRedisSnapshot 提供启动时固化的 Redis 配置快照（黑名单等启动期依赖使用）。
//
// 远程业务配置热更后，仅持有者中需要随配置重建的依赖才受影响；Redis 连接在启动期
// 一次性建立（重建代价高且本 wave 无重建诉求），故取构造时刻快照并在注释中明示该约定。
// 需要读最新配置的后续 wave 组件请直接依赖 *conf.RemoteConfigHolder 调 Get()。
func ProvideRedisSnapshot(h *conf.RemoteConfigHolder) conf.RedisConfig {
	return h.Get().Redis
}
