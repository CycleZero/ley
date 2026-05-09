package biz

import (
	"ley/app/auth/internal/conf"
	"ley/pkg/cache"
	"ley/pkg/eventbus"
	"ley/pkg/jwt"
	"ley/pkg/mq"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/wire"
)

func ProvideJWT(sc *conf.Config) jwt.JWT {
	return jwt.NewJWT(&jwt.Config{
		SigningKey:  sc.Jwt.Secret,
		ExpiredTime: sc.Jwt.AccessTtl.AsDuration(),
		Issuer:      sc.Jwt.Issuer,
	}, nil)
}

func ProvideBlackList(c cache.Cache) jwt.BlackListCache {
	return jwt.NewBlackList(c)
}

func ProvideEventBus(logger log.Logger) eventbus.EventBus {
	return eventbus.NewEventBus(
		mq.NewMemoryConnection(256),
		eventbus.EventBusConfig{Source: "auth-service"},
		logger,
	)
}

var ProviderSet = wire.NewSet(
	NewUserUseCase,
	NewAuthUseCase,
	ProvideJWT,
	ProvideBlackList,
	ProvideEventBus,
)
