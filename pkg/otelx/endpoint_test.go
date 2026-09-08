package otelx

import "testing"

func TestParseEndpoint(t *testing.T) {
	tests := []struct {
		name         string
		raw          string
		insecure     bool
		defaultPath  string
		wantEndpoint string
		wantPath     string
		wantInsecure bool
	}{
		{"空值交环境变量", "", true, "/v1/traces", "", "", true},
		{"裸 host:port", "127.0.0.1:4318", true, "/v1/traces", "127.0.0.1:4318", "", true},
		{"http 强制明文", "http://127.0.0.1:4318", false, "/v1/traces", "127.0.0.1:4318", "", true},
		{"https 强制 TLS", "https://collector.example.com", true, "/v1/traces", "collector.example.com", "", false},
		{"trace 默认路径剥离", "http://127.0.0.1:4318/v1/traces", true, "/v1/traces", "127.0.0.1:4318", "", true},
		{"自定义路径保留", "http://example.com/otlp/v1/traces", true, "/v1/traces", "example.com", "/otlp/v1/traces", true},
		{"metrics 默认路径", "http://127.0.0.1:4318/v1/metrics", true, "/v1/metrics", "127.0.0.1:4318", "", true},
		{"logs 路径不误判", "http://127.0.0.1:4318/v1/logs", true, "/v1/traces", "127.0.0.1:4318", "/v1/logs", true},
		{"尾部斜杠", "http://127.0.0.1:4318/v1/traces/", true, "/v1/traces", "127.0.0.1:4318", "", true},
		{"非法 URL 退化剥离", "https://", true, "/v1/traces", "", "", true},
		{"空白裁剪", "  collector:4317  ", true, "/v1/traces", "collector:4317", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ep, path, insecure := ParseEndpoint(tt.raw, tt.insecure, tt.defaultPath)
			if ep != tt.wantEndpoint || path != tt.wantPath || insecure != tt.wantInsecure {
				t.Errorf("ParseEndpoint(%q) = (%q, %q, %v)，期望 (%q, %q, %v)",
					tt.raw, ep, path, insecure, tt.wantEndpoint, tt.wantPath, tt.wantInsecure)
			}
		})
	}
}
