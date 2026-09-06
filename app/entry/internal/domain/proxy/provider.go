package proxy

import "github.com/google/wire"

// ProxyProviderSet 代理域 Wire ProviderSet。
//
// 绑定关系：NewProxyHub 依赖 *infra.AuthClientConn / *infra.BlogClientConn，
// 两者由 infra.InfraProviderSet 提供（见 infra/provider.go 绑定说明）。
var ProxyProviderSet = wire.NewSet(NewProxyHub)
