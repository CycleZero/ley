package biz

import (
	"github.com/google/wire"
)

// ProviderSet Wire 依赖注入集合
// 声明 biz 层需要由 Wire 生成的 Provider，用于自动组装依赖关系
// 当前仅包含 CommentUseCase 的构造函数
var ProviderSet = wire.NewSet(
	NewCommentUseCase,
)
