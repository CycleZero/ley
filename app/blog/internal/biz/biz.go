package biz

import (
	"ley/app/blog/internal/conf"
	"ley/pkg/eventbus"
	"ley/pkg/mq"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/wire"
)

func ProvideEventBus(logger log.Logger, sc *conf.Config) eventbus.EventBus {
	return eventbus.NewEventBus(
		mq.NewMemoryConnection(256),
		eventbus.EventBusConfig{Source: "blog-service"},
		logger,
	)
}

var ProviderSet = wire.NewSet(
	NewArticleUseCase,
	NewCommentUseCase,
	NewTagUseCase,
	NewCategoryUseCase,
	NewFileUseCase,
	NewSiteUseCase,
	ProvideEventBus,
)
