package conf

import "github.com/CycleZero/ley/pkg/constant"

const (
	ServiceName      = constant.ServiceNameBlog
	ServiceDataDir   = "./data/" + ServiceName
	LocalConfigDir   = ServiceDataDir + "/configs"
	RemoteConfigPath = constant.AppName + "/configs/" + ServiceName + "/config.yaml"
)
