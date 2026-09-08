// Package otelx 提供 OTel 相关的小工具，供 trace / metrics / log 三类信号共用。
package otelx

import (
	"net/url"
	"strings"
)

// ParseEndpoint 规范化 OTLP/HTTP 端点，兼容带 scheme 的配置写法。
//
// 返回 host:port、URL 路径（与 exporter 默认路径 defaultPath 相同时为空）以及是否明文传输。
//
// 背景：历史配置里端点被写成 "http://127.0.0.1:4318"，而 OTel 各 exporter 的
// WithEndpoint 只接受 host:port，会二次拼接协议头，生成
// "http://http:%2F%2F127.0.0.1:4318/v1/traces" 这类畸形 URL。
// 规则：
//   - 空值：返回空端点，交由 OTel 环境变量兜底；
//   - 无 scheme：按 host:port 处理，沿用调用方 insecure；
//   - http:// ：剥离 scheme 并强制明文；
//   - https://：剥离 scheme 并强制 TLS（忽略调用方 insecure）。
func ParseEndpoint(raw string, insecure bool, defaultPath string) (endpoint, path string, useInsecure bool) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", "", insecure
	}
	if !strings.Contains(trimmed, "://") {
		return trimmed, "", insecure
	}
	u, err := url.Parse(trimmed)
	if err != nil || u.Host == "" {
		// 解析失败时退化为裸剥离 scheme，保证进程仍能启动（由 exporter 上报错误）
		fallback := strings.TrimPrefix(strings.TrimPrefix(trimmed, "https://"), "http://")
		return fallback, "", insecure
	}
	switch strings.ToLower(u.Scheme) {
	case "https":
		insecure = false
	case "http":
		insecure = true
	}
	p := strings.TrimSuffix(u.Path, "/")
	if p == strings.TrimSuffix(defaultPath, "/") {
		p = "" // exporter 默认路径，无需重复设置
	}
	return u.Host, p, insecure
}
