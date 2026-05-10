package conf

const (
	ServiceName      = "blog"
	ServiceDataDir   = "./data/" + ServiceName
	LocalConfigDir   = ServiceDataDir + "/configs"
	RemoteConfigPath = "/ley/configs/" + ServiceName + "/config.yaml"
)
