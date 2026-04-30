// Package biz 文章服务业务逻辑层
// 实现文章的创建、发布、编辑、删除等核心业务逻辑，
// 以及标签和分类的管理业务逻辑。
package biz

import (
	"github.com/google/wire"
)

// ProviderSet Wire 依赖注入集合
// 将三个 UseCase 构造函数注册到 Wire 依赖注入容器中。
var ProviderSet = wire.NewSet(
	NewPostUseCase,     // 文章业务用例构造函数
	NewTagUseCase,      // 标签业务用例构造函数
	NewCategoryUseCase, // 分类业务用例构造函数
)
