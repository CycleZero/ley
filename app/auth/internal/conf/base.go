package conf

const (
	ServiceName      = "auth"
	ServiceDataDir   = "./data/" + ServiceName
	LocalConfigDir   = ServiceDataDir + "/configs"
	RemoteConfigPath = "/ley/configs/" + ServiceName + "/config.yaml"
)
