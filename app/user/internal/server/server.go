// Package server — 用户服务 gRPC + HTTP 传输层
//
// 负责：
//   1. 创建 Kratos gRPC Server 并注册用户服务
//   2. 创建 Kratos HTTP Server 并注册用户服务
//   3. 组装通用中间件链（recovery/tracing/logging/metadata）
//
// 中间件执行顺序：Recovery → Tracing → Logging → Metadata → Handler
package server

import (
	"github.com/google/wire"
)

var ProviderSet = wire.NewSet(NewGRPCServer, NewHTTPServer)
