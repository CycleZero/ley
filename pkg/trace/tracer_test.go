package trace

import "testing"

// 场景：OTLP/HTTP 端点兼容带 scheme 的配置写法。
// 历史上 trace.endpoint 被写成 "http://127.0.0.1:4318"，
// 会被 exporter 二次拼接协议头生成畸形 URL，导致 trace 上报静默失败。
func TestNormalizeEndpoint(t *testing.T) {
	tests := []struct {
		name         string
		raw          string
		insecure     bool
		wantEndpoint string
		wantPath     string
		wantInsecure bool
	}{
		{name: "空值交由环境变量兜底", raw: "", insecure: true, wantEndpoint: "", wantPath: "", wantInsecure: true},
		{name: "裸 host:port 沿用调用方配置", raw: "127.0.0.1:4318", insecure: true, wantEndpoint: "127.0.0.1:4318", wantPath: "", wantInsecure: true},
		{name: "http scheme 强制明文", raw: "http://127.0.0.1:4318", insecure: false, wantEndpoint: "127.0.0.1:4318", wantPath: "", wantInsecure: true},
		{name: "https scheme 强制 TLS", raw: "https://otel.example.com:4318", insecure: true, wantEndpoint: "otel.example.com:4318", wantPath: "", wantInsecure: false},
		{name: "尾部斜杠归一化", raw: "http://127.0.0.1:4318/", insecure: true, wantEndpoint: "127.0.0.1:4318", wantPath: "", wantInsecure: true},
		{name: "默认路径不重复设置", raw: "http://127.0.0.1:4318/v1/traces", insecure: true, wantEndpoint: "127.0.0.1:4318", wantPath: "", wantInsecure: true},
		{name: "自定义路径透传", raw: "http://127.0.0.1:5080/api/default/v1/traces", insecure: true, wantEndpoint: "127.0.0.1:5080", wantPath: "/api/default/v1/traces", wantInsecure: true},
		{name: "首尾空白忽略", raw: "  http://127.0.0.1:4318  ", insecure: true, wantEndpoint: "127.0.0.1:4318", wantPath: "", wantInsecure: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			endpoint, path, insecure := normalizeEndpoint(tt.raw, tt.insecure)
			if endpoint != tt.wantEndpoint {
				t.Errorf("endpoint = %q，期望 %q", endpoint, tt.wantEndpoint)
			}
			if path != tt.wantPath {
				t.Errorf("path = %q，期望 %q", path, tt.wantPath)
			}
			if insecure != tt.wantInsecure {
				t.Errorf("insecure = %v，期望 %v", insecure, tt.wantInsecure)
			}
		})
	}
}
