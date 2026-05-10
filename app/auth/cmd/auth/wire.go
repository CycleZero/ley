//go:build wireinject
// +build wireinject

// The build tag makes sure the stub is not built in the final build.

package main

import (
	"github.com/CycleZero/ley/app/auth/internal/biz"
	"github.com/CycleZero/ley/app/auth/internal/conf"
	"github.com/CycleZero/ley/app/auth/internal/data"
	"github.com/CycleZero/ley/app/auth/internal/server"
	"github.com/CycleZero/ley/app/auth/internal/service"
	commonconf "github.com/CycleZero/ley/conf"

	"github.com/go-kratos/kratos/v2"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/wire"
	"go.etcd.io/etcd/client/v3"
)

// wireApp init kratos application.
func wireApp(
	bc *commonconf.Bootstrap,
	sc *conf.Config,
	confServer *commonconf.Server,
	confData *conf.Data,
	logger log.Logger,
	etcdClient *clientv3.Client,
) (*kratos.App, func(), error) {
	panic(wire.Build(server.ProviderSet, data.ProviderSet, biz.ProviderSet, service.ProviderSet, newApp))
}
