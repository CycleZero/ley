package conf

import "ley/pkg/constant"

const ServiceDataDir = constant.DataDir + "/" + ServiceName

const LocalConfigDir = ServiceDataDir + "/" + "configs"
const ServiceName = constant.ServiceNameBlog
const (
	RemoteConfigPath = constant.RemoteConfigPathBase +
		constant.RemoteConfigPathSep + ServiceName +
		constant.RemoteConfigPathSep + "config.yaml"
)
