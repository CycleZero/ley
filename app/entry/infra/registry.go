// Package infra 提供 entry 入口服务的基础设施组装：etcd 服务发现、auth/blog gRPC
// 客户端连接、JWT 黑名单 Redis 缓存与 Wire ProviderSet。
//
// 设计约束（AGENTS.md「Entry 入口服务特别说明」）：entry 不直接访问 gorm/minio/oss 等
// 存储，本包只做「客户端侧基础设施」——发现下游服务端点并建立转发通道。
package infra

import (
	etcdreg "github.com/go-kratos/kratos/contrib/registry/etcd/v2"
	"github.com/go-kratos/kratos/v2/registry"
	clientv3 "go.etcd.io/etcd/client/v3"
)

// NewDiscovery 基于 etcd 客户端创建服务发现（registry.Discovery）。
//
// contrib registry etcd v2 的 Registry 同时实现 Registrar 与 Discovery（单返回值），
// entry 仅消费 Discovery 角色（解析 auth/blog 的 gRPC 端点），注册职责由各微服务自身承担。
// 函数返回类型 registry.Discovery 本身即编译期约束：contrib API 若不再满足该接口将无法编译。
func NewDiscovery(etcdClient *clientv3.Client) registry.Discovery {
	return etcdreg.New(etcdClient)
}
