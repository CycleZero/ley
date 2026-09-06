// Package conf 提供 entry 入口服务的配置加载。
//
// 与 auth/blog 微服务一致采用 Kratos 两级配置体系：
//  1. 本地引导配置（Bootstrap，conf/common.proto）：本文件 LoadBootstrap 加载
//  2. etcd 远程业务配置：RemoteConfigPath 常量预留给后续 wave 的 remote.go
package conf

import (
	"fmt"

	commonconf "github.com/CycleZero/ley/conf"
	"github.com/CycleZero/ley/pkg/constant"
	"github.com/go-kratos/kratos/v2/config"
	"github.com/go-kratos/kratos/v2/config/file"
)

const (
	// ServiceName 服务名（日志文件名、目录名、trace 服务名均基于它）
	ServiceName = "entry"
	// ServiceDataDir 服务数据目录
	ServiceDataDir = "./data/" + ServiceName
	// LocalConfigDir 本地引导配置目录（main.go -conf flag 的默认值）
	LocalConfigDir = ServiceDataDir + "/configs"
	// RemoteConfigPath etcd 远程业务配置路径（ley/configs/entry/config.yaml）
	// 本 wave 尚未接线，后续 wave 在 remote.go 中细化加载逻辑
	RemoteConfigPath = constant.AppName + "/configs/" + ServiceName + "/config.yaml"
)

// LoadBootstrap 从本地路径加载引导配置（conf/common.proto 的 Bootstrap）。
//
// path 为目录或单个 YAML 文件均可：kratos file source 会扫描目录下配置文件，
// 并按扩展名选择 yaml codec 解析，最终 Scan 进 proto 结构。
func LoadBootstrap(path string) (*commonconf.Bootstrap, error) {
	c := config.New(config.WithSource(file.NewSource(path)))
	defer c.Close()

	var bc commonconf.Bootstrap
	if err := c.Load(); err != nil {
		return nil, fmt.Errorf("加载引导配置失败：%w", err)
	}
	if err := c.Scan(&bc); err != nil {
		return nil, fmt.Errorf("解析引导配置失败：%w", err)
	}
	return &bc, nil
}
