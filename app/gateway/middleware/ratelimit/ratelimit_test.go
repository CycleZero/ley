package ratelimit

import (
	"testing"

	config "github.com/CycleZero/ley/api/gateway/config/v1"
)

// B-304: 429 限流响应须为统一信封格式（与 wrapresp 错误形态一致）
func TestBuildLimitEnvelope(t *testing.T) {
	got := string(buildLimitEnvelope(429, "请求过于频繁，请稍后再试"))
	expect := `{"code":429,"msg":"请求过于频繁，请稍后再试","data":null}`
	if got != expect {
		t.Errorf("信封格式不符:\n got %s\nwant %s", got, expect)
	}
}

// B-305: 无规则配置时中间件退化为直通（不创建 Redis 连接）
func TestMiddlewarePassthroughWithoutRules(t *testing.T) {
	_, err := Middleware(&config.Middleware{})
	if err != nil {
		t.Fatalf("Middleware 不应报错: %v", err)
	}
}
