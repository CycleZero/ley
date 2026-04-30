//go:build wireinject
// +build wireinject

package main

import (
	"ley/app/post/internal/biz"
	"ley/app/post/internal/conf"
	"ley/app/post/internal/data"
	"ley/app/post/internal/server"
	"ley/app/post/internal/service"
	commonconf "ley/conf"
	"ley/pkg/cache"
	"ley/pkg/infra"

	"github.com/go-kratos/kratos/v2"
	etcdreg "github.com/go-kratos/kratos/contrib/registry/etcd/v2"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/registry"
	"github.com/google/wire"
	clientv3 "go.etcd.io/etcd/client/v3"
	"gorm.io/gorm"
)

func wireApp(
	bc *commonconf.Bootstrap,
	sc *conf.Config,
	confServer *commonconf.Server,
	confData *conf.Data,
	logger log.Logger,
	etcdClient *clientv3.Client,
) (*kratos.App, func(), error) {
	panic(wire.Build(
		server.ProviderSet,
		data.ProviderSet,
		biz.ProviderSet,
		service.ProviderSet,
		newApp,
		provideDB,
		provideCache,
		provideRegistrar,
	))
}

func provideDB(confData *conf.Data) *gorm.DB {
	db := confData.Database
	return infra.NewDB(infra.NewDbParams{
		Driver: db.Driver,
		Host:   db.Host,
		Port:   int(db.Port),
		User:   db.Username,
		Pass:   db.Password,
		DBName: db.Database,
	})
}

func provideCache(confData *conf.Data) cache.Cache {
	red := confData.Redis
	return cache.NewRedisCache(
		red.Host, int(red.Port), red.Password, int(red.Db),
	)
}

func provideRegistrar(client *clientv3.Client) registry.Registrar {
	return etcdreg.New(client)
}
