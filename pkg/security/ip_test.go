package security

import (
	"context"
	"testing"

	"github.com/CycleZero/ley/pkg/meta"
	"github.com/go-kratos/kratos/v2/metadata"
	"github.com/go-kratos/kratos/v2/transport"
)

// fakeHeader / fakeTransporter：构造 Kratos transport 上下文
type fakeHeader struct {
	h map[string]string
}

func (f *fakeHeader) Get(key string) string { return f.h[key] }
func (f *fakeHeader) Set(key, value string) { f.h[key] = value }
func (f *fakeHeader) Add(key, value string) { f.h[key] = value }
func (f *fakeHeader) Values(key string) []string {
	if v, ok := f.h[key]; ok {
		return []string{v}
	}
	return nil
}
func (f *fakeHeader) Keys() []string {
	keys := make([]string, 0, len(f.h))
	for k := range f.h {
		keys = append(keys, k)
	}
	return keys
}

type fakeTransporter struct {
	kind      transport.Kind
	reqH      transport.Header
	replyH    transport.Header
	endpoint  string
	operation string
}

func (t *fakeTransporter) Kind() transport.Kind        { return t.kind }
func (t *fakeTransporter) Endpoint() string            { return t.endpoint }
func (t *fakeTransporter) Operation() string           { return t.operation }
func (t *fakeTransporter) RequestHeader() transport.Header { return t.reqH }
func (t *fakeTransporter) ReplyHeader() transport.Header   { return t.replyH }

func newHeaderContext(kv map[string]string) context.Context {
	h := &fakeHeader{h: kv}
	tr := &fakeTransporter{kind: transport.KindHTTP, reqH: h, replyH: &fakeHeader{h: map[string]string{}}}
	return transport.NewServerContext(context.Background(), tr)
}

func TestIsValidIp(t *testing.T) {
	valid := []string{
		"8.8.8.8", "1.1.1.1", "192.168.1.1", "10.0.0.1", "172.16.0.1",
		"2001:4860:4860::8888", "::1", "fe80::1",
	}
	for _, ip := range valid {
		if !IsValidIp(ip) {
			t.Errorf("IsValidIp(%q) 应为 true", ip)
		}
	}
	invalid := []string{"", "999.1.1.1", "abc", "8.8.8", "1.2.3.4.5", "not-an-ip"}
	for _, ip := range invalid {
		if IsValidIp(ip) {
			t.Errorf("IsValidIp(%q) 应为 false", ip)
		}
	}
}

func TestIsPrivateIp(t *testing.T) {
	private := []string{"10.0.0.1", "172.16.0.1", "172.31.255.255", "192.168.1.1", "127.0.0.1", "::1", "fc00::1"}
	for _, ip := range private {
		if !IsPrivateIp(ip) {
			t.Errorf("IsPrivateIp(%q) 应为 true", ip)
		}
	}
	public := []string{"8.8.8.8", "114.114.114.114", "2001:4860:4860::8888"}
	for _, ip := range public {
		if IsPrivateIp(ip) {
			t.Errorf("IsPrivateIp(%q) 应为 false", ip)
		}
	}
	if IsPrivateIp("invalid") {
		t.Error("非法 IP 应返回 false")
	}
}

func TestGetRealIpFromMetaFirst(t *testing.T) {
	// meta 中透传的真实 IP 优先级最高
	ctx := newHeaderContext(map[string]string{"X-Forwarded-For": "8.8.8.8"})
	md := metadata.New(map[string][]string{meta.AuthRealClientIpKey: {"9.9.9.9"}})
	ctx = metadata.NewServerContext(ctx, md)
	if got := GetRealIp(ctx); got != "9.9.9.9" {
		t.Errorf("meta 透传 IP 应优先: got %q", got)
	}
}

func TestGetRealIpFromHeaders(t *testing.T) {
	cases := []struct {
		name string
		kv   map[string]string
		want string
	}{
		{"EO-Client-Ip 最高优先", map[string]string{"EO-Client-Ip": "1.1.1.1", "X-Real-Ip": "2.2.2.2"}, "1.1.1.1"},
		{"Cf-Connecting-Ip", map[string]string{"Cf-Connecting-Ip": "3.3.3.3"}, "3.3.3.3"},
		{"X-Real-Ip", map[string]string{"X-Real-Ip": "4.4.4.4"}, "4.4.4.4"},
		{"XFF 取第一个", map[string]string{"X-Forwarded-For": "5.5.5.5, 6.6.6.6, 7.7.7.7"}, "5.5.5.5"},
		{"XFF 带空格", map[string]string{"X-Forwarded-For": " 8.8.8.8 , 9.9.9.9 "}, "8.8.8.8"},
		// 实现局限：每个头只取第一个 IP，私有则放弃该头；XFF 首个私有时走回退路径
		{"XFF 首个私有走回退", map[string]string{"X-Real-Ip": "192.168.1.1", "X-Forwarded-For": "10.0.0.1, 8.8.8.8"}, "10.0.0.1"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := GetRealIp(newHeaderContext(c.kv)); got != c.want {
				t.Errorf("GetRealIp = %q, want %q", got, c.want)
			}
		})
	}
}

func TestGetRealIpFallbackToXFFPrivate(t *testing.T) {
	// 全私有时回退 XFF 第一个（即使内网）
	ctx := newHeaderContext(map[string]string{"X-Forwarded-For": "10.0.0.5, 192.168.1.1"})
	if got := GetRealIp(ctx); got != "10.0.0.5" {
		t.Errorf("应回退到 XFF 第一个: got %q", got)
	}
}

func TestGetRealIpDefault(t *testing.T) {
	// 无任何 header → 默认 0.0.0.0
	if got := GetRealIp(context.Background()); got != "0.0.0.0" {
		t.Errorf("默认应返回 0.0.0.0: got %q", got)
	}
}
