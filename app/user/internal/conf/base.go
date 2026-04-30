// Package conf — 用户服务配置常量与远程配置路径
//
// 定义服务名称、本地/远程配置路径等常量和辅助函数。
// 通过 Wire 注入到 Data 构造函数中。
package conf

import "ley/pkg/constant"

// ServiceName 服务在 etcd 注册中心中的名称
const ServiceName = constant.ServiceNameUser

// ServiceDataDir 服务数据目录（日志、本地配置等）
const ServiceDataDir = constant.DataDir + "/" + ServiceName

// LocalConfigDir 本地配置文件目录
const LocalConfigDir = ServiceDataDir + "/" + "configs"

// RemoteConfigPath etcd 远程配置路径
// 格式: ley/configs/user/config.yaml
const RemoteConfigPath = constant.RemoteConfigPathBase +
	constant.RemoteConfigPathSep + ServiceName +
	constant.RemoteConfigPathSep + "config.yaml"
