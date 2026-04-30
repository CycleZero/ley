//go:build wireinject
// +build wireinject

package main

import (
	"time"

	"ley/app/user/internal/biz"
	"ley/app/user/internal/conf"
	"ley/app/user/internal/data"
	"ley/app/user/internal/server"
	"ley/app/user/internal/service"
	commonconf "ley/conf"
	"ley/pkg/cache"
	"ley/pkg/infra"
	"ley/pkg/jwt"

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
		// 用户服务特有：JWT 组件（从 sc *Config 提取，sc 是完整服务配置）
		provideJWT,
		provideJWTBlacklist,
		provideJWTExpire,
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

// provideJWT 从完整服务配置（Config）创建 JWTProvider
func provideJWT(sc *conf.Config) biz.JWTProvider {
	jwtCfg := &jwt.Config{
		SigningKey:  sc.Jwt.Secret,
		ExpiredTime: sc.Jwt.AccessTtl.AsDuration(),
		Issuer:      sc.Jwt.Issuer,
	}
	return jwt.NewJWT(jwtCfg)
}

// provideJWTExpire 从完整服务配置提取令牌过期时间
func provideJWTExpire(sc *conf.Config) time.Duration {
	return sc.Jwt.AccessTtl.AsDuration()
}

// provideJWTBlacklist 从 Cache 创建 JWT 黑名单缓存
func provideJWTBlacklist(c cache.Cache) jwt.BlackListCache {
	return jwt.NewBlackList(c)
}
