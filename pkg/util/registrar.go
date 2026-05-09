package util

import (
	"github.com/go-kratos/kratos/contrib/registry/etcd/v2"
	"github.com/go-kratos/kratos/v2/registry"
	clientv3 "go.etcd.io/etcd/client/v3"
)

// 服务注册器与服务发现器
// 两者都基于相同的etcd客户端

func NewRegistrar(etcdClient *clientv3.Client) registry.Registrar {
	return etcd.New(etcdClient)
}

func NewDiscovery(etcdClient *clientv3.Client) registry.Discovery {
	return etcd.New(etcdClient)
}
