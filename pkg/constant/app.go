package constant

const AppName = "ley"

const DataDir = "./data"
const (
	ServiceNameComment    = "comment"
	ServiceNameUser       = "user"
	ServiceNamePost       = "post"
	ServiceNameGreeter    = "greeter"
	ServiceNameAttachment = "attachment"
)

const (

	// RemoteConfigPathSep 路径分隔符
	RemoteConfigPathSep = "/"
	// RemoteConfigPathBase etcd远程配置文件根路径
	RemoteConfigPathBase = AppName + RemoteConfigPathSep + "configs"
)
