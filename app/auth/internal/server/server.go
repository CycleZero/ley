package server

import (
	"github.com/CycleZero/ley/pkg/util"

	"github.com/google/wire"
)

var ProviderSet = wire.NewSet(NewGRPCServer, NewHTTPServer, util.NewDiscovery, util.NewRegistrar)
