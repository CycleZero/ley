package infra

//
// 通用 ProviderSet 已废弃 —— DB/Cache/Registrar 的构造函数因各服务使用
// 不同的 conf.Data 类型，改为在各服务 cmd/wire.go 中通过 inline Wire 函数提供。
// 参见: app/{service}/cmd/wire.go
//
