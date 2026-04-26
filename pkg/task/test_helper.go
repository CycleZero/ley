package task

import (
	"go.uber.org/zap"
)

// initTestLogger 初始化测试日志器
func initTestLogger() {
	// 创建一个简单的控制台日志器用于测试
	config := zap.NewDevelopmentConfig()
	config.OutputPaths = []string{"stdout"}
	config.ErrorOutputPaths = []string{"stderr"}

	logger, err := config.Build()
	if err != nil {
		return
	}

	// 设置全局日志器
	zap.ReplaceGlobals(logger)
}
