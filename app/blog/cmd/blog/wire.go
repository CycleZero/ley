//go:build wireinject
// +build wireinject

package main

import (
	"ley/app/blog/internal/biz"
	"ley/app/blog/internal/conf"
	"ley/app/blog/internal/data"
	"ley/app/blog/internal/server"
	"ley/app/blog/internal/service"
	commonconf "ley/conf"

	"github.com/go-kratos/kratos/v2"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/wire"
	"go.etcd.io/etcd/client/v3"
)

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
