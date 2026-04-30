// Package conf — 文章服务配置常量与远程配置路径
package conf

import "ley/pkg/constant"

// ServiceName 服务在 etcd 注册中心中的名称
const ServiceName = constant.ServiceNamePost

// ServiceDataDir 服务数据目录
const ServiceDataDir = constant.DataDir + "/" + ServiceName

// LocalConfigDir 本地配置文件目录
const LocalConfigDir = ServiceDataDir + "/" + "configs"

// RemoteConfigPath etcd 远程配置路径
// 格式: ley/configs/post/config.yaml
const RemoteConfigPath = constant.RemoteConfigPathBase +
	constant.RemoteConfigPathSep + ServiceName +
	constant.RemoteConfigPathSep + "config.yaml"
