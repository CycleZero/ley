package biz

import "github.com/google/wire"

var ProviderSet = wire.NewSet(
	NewArticleUseCase,
	NewCommentUseCase,
	NewTagUseCase,
	NewCategoryUseCase,
	NewFileUseCase,
	NewSiteUseCase,
)
