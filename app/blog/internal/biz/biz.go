package biz

import (
	"github.com/CycleZero/ley/app/blog/internal/conf"
	"github.com/CycleZero/ley/pkg/eventbus"
	"github.com/CycleZero/ley/pkg/mq"

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
