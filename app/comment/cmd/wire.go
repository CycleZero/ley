//go:build wireinject
// +build wireinject

// Package main 评论服务 Wire 依赖注入声明
//
// 定义 wireApp 注入函数及其所需的底层 Provider 函数。
// Wire 根据此文件 + ProviderSet 自动生成 wire_gen.go。
//
// 生成命令: wire gen ./app/comment/cmd
// 或通过: make wire
package main

import (
	"ley/app/comment/internal/biz"
	"ley/app/comment/internal/conf"
	"ley/app/comment/internal/data"
	"ley/app/comment/internal/server"
	"ley/app/comment/internal/service"
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

// wireApp Wire 依赖注入入口
// 参数由 main.go 从配置文件和 etcd 加载后传入
func wireApp(
	bc *commonconf.Bootstrap,        // 根引导配置（Server/Log/Etcd/Trace）
	sc *conf.Config,                  // 评论服务远程配置（Data/MinIO）
	confServer *commonconf.Server,    // Server 配置段
	confData *conf.Data,              // Data 配置段（DB/Redis）
	logger log.Logger,                // 日志器（main 中已初始化）
	etcdClient *clientv3.Client,      // etcd 客户端
) (*kratos.App, func(), error) {
	panic(wire.Build(
		// 各层 ProviderSet
		server.ProviderSet,
		data.ProviderSet,
		biz.ProviderSet,
		service.ProviderSet,
		// App 构造函数
		newApp,
		// 底层组件 Provider（由配置创建）
		provideDB,
		provideCache,
		provideRegistrar,
	))
}

// ---- Wire Provider 函数（仅 wireinject 编译标签下编译） ----

// provideDB 从评论服务 Data 配置创建 PostgreSQL GORM 连接
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

// provideCache 从评论服务 Data 配置创建 Redis Cache 客户端
func provideCache(confData *conf.Data) cache.Cache {
	red := confData.Redis
	return cache.NewRedisCache(
		red.Host, int(red.Port), red.Password, int(red.Db),
	)
}

// provideRegistrar 从 etcd Client 创建服务注册器
func provideRegistrar(client *clientv3.Client) registry.Registrar {
	return etcdreg.New(client)
}
