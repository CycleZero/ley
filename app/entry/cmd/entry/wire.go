//go:build wireinject
// +build wireinject

package main

import (
	"github.com/CycleZero/ley/app/entry/infra"
	"github.com/CycleZero/ley/app/entry/internal/domain/proxy"
	"github.com/CycleZero/ley/app/entry/internal/router"
	commonconf "github.com/CycleZero/ley/conf"
	"github.com/google/wire"
)

// initApp 构建应用依赖图（wire 生成 wire_gen.go，gitignored）。
//
// 完整图谱（wave 5，proxy 核心接线后）：
//
//	bc *commonconf.Bootstrap（注入参数）
//	 ├─ infra.InfraProviderSet
//	 │    ├─ ProvideEtcdClient ──► *clientv3.Client
//	 │    │    └─ NewDiscovery ──► registry.Discovery
//	 │    │         ├─ NewAuthClientConn ──► *infra.AuthClientConn
//	 │    │         └─ NewBlogClientConn ──► *infra.BlogClientConn
//	 │    └─（黑名单/远程配置等 Provider 供后续 wave 消费，wire 只装配被依赖项）
//	 ├─ proxy.ProxyProviderSet
//	 │    └─ NewProxyHub(authConn, blogConn) ──► *proxy.ServiceHub
//	 ├─ router.RouterProviderSet
//	 │    ├─ NewRegisterFunc ──► router.RegisterFunc
//	 │    └─ NewRegisterMiddleWire ──► router.RegisteredMiddleWire
//	 └─ NewMainApp(bc, hub, registerFunc, middleWire) ──► *MainApp
//
// cleanup 链由 wire 逆序合成：etcd 客户端等带 cleanup 的 Provider 会在
// 应用退出时被释放（main.go defer cleanup() 调用）。
func initApp(bc *commonconf.Bootstrap) (*MainApp, func(), error) {
	panic(wire.Build(
		NewMainApp,
		router.RouterProviderSet,
		proxy.ProxyProviderSet,
		infra.InfraProviderSet,
	))
}
