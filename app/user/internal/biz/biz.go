package biz

import (
	"github.com/google/wire"
)

// ProviderSet Wire 依赖注入集合
// 包含所有 UseCase 构造函数（UserUseCase、AuthUseCase）
// Wire 在编译期根据构造函数签名自动解析依赖链并生成初始化代码
// 新增 UseCase 时在此集合中注册即可
var ProviderSet = wire.NewSet(
	NewUserUseCase,  // 用户资料业务用例
	NewAuthUseCase,  // 认证与令牌业务用例
)
