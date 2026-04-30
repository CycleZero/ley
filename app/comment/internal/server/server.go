// Package server — 评论服务 gRPC + HTTP 传输层
package server

import (
	"github.com/google/wire"
)

var ProviderSet = wire.NewSet(NewGRPCServer, NewHTTPServer)
