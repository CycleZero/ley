package meta

import (
	"context"
	"testing"

	"github.com/go-kratos/kratos/v2/metadata"
)

func TestIntoMetadataAndParseRoundTrip(t *testing.T) {
	reqMeta := &RequestMetaData{
		Auth:         Auth{UserID: 123, UserName: "tester"},
		RealClientIp: "8.8.8.8",
	}

	md := reqMeta.IntoMetadata()
	if md.Get(AuthUserIDKey) != "123" {
		t.Errorf("UserID 键值错误: %q", md.Get(AuthUserIDKey))
	}
	if md.Get(AuthUserNameKey) != "tester" {
		t.Errorf("UserName 键值错误: %q", md.Get(AuthUserNameKey))
	}
	if md.Get(AuthRealClientIpKey) != "8.8.8.8" {
		t.Errorf("IP 键值错误: %q", md.Get(AuthRealClientIpKey))
	}

	parsed := ParseMetadata(md)
	if parsed.Auth.UserID != 123 || parsed.Auth.UserName != "tester" || parsed.RealClientIp != "8.8.8.8" {
		t.Errorf("往返解析不一致: %+v", parsed)
	}
}

func TestIntoMetadataSkipsZeroValues(t *testing.T) {
	// UserID=0、空串不应写入 metadata
	reqMeta := &RequestMetaData{Auth: Auth{UserName: "only-name"}}
	md := reqMeta.IntoMetadata()
	if md.Get(AuthUserIDKey) != "" {
		t.Errorf("UserID=0 不应写入: %q", md.Get(AuthUserIDKey))
	}
	if md.Get(AuthRealClientIpKey) != "" {
		t.Errorf("空 IP 不应写入: %q", md.Get(AuthRealClientIpKey))
	}
}

func TestNilMetaSafe(t *testing.T) {
	var m *RequestMetaData
	md := m.IntoMetadata() // nil 不 panic
	if len(md) != 0 {
		t.Errorf("nil meta 应返回空 metadata: %v", md)
	}
}

func TestParseMetadataInvalidUserID(t *testing.T) {
	// 非法 UserID 不应 panic，解析为 0
	md := metadata.New(map[string][]string{
		AuthUserIDKey:   {"not-a-number"},
		AuthUserNameKey: {"tester"},
	})
	parsed := ParseMetadata(md)
	if parsed.Auth.UserID != 0 {
		t.Errorf("非法 UserID 应解析为 0: %d", parsed.Auth.UserID)
	}
	if parsed.Auth.UserName != "tester" {
		t.Errorf("UserName 解析错误: %q", parsed.Auth.UserName)
	}
}

func TestNewClientCtxAndGetClientMeta(t *testing.T) {
	reqMeta := &RequestMetaData{Auth: Auth{UserID: 7, UserName: "alice"}}
	ctx := NewClientCtx(context.Background(), reqMeta)

	got := GetClientMeta(ctx)
	if got.Auth.UserID != 7 || got.Auth.UserName != "alice" {
		t.Errorf("客户端上下文往返不一致: %+v", got)
	}
}

func TestGetServerMeta(t *testing.T) {
	md := metadata.New(map[string][]string{AuthUserIDKey: []string{"9"}})
	ctx := metadata.NewServerContext(context.Background(), md)

	got := GetServerMeta(ctx)
	if got.Auth.UserID != 9 {
		t.Errorf("服务端上下文解析错误: %+v", got)
	}
}

func TestGetRequestMetaDataPriority(t *testing.T) {
	// 同时存在 server 与 client 上下文时，server 优先
	serverMD := metadata.New(map[string][]string{AuthUserIDKey: []string{"1"}})
	clientMD := metadata.New(map[string][]string{AuthUserIDKey: []string{"2"}})
	ctx := metadata.NewClientContext(context.Background(), clientMD)
	ctx = metadata.NewServerContext(ctx, serverMD)

	got := GetRequestMetaData(ctx)
	if got.Auth.UserID != 1 {
		t.Errorf("server 上下文应优先: got %d want 1", got.Auth.UserID)
	}
}

func TestGetRequestMetaDataEmpty(t *testing.T) {
	got := GetRequestMetaData(context.Background())
	if got == nil || got.Auth.UserID != 0 {
		t.Errorf("无上下文时应返回空结构: %+v", got)
	}
}

// B-103: 用户角色须随业务元数据透传（x-md-global-auth-role），供下游服务做 admin 判定
func TestMetaCarriesRole(t *testing.T) {
	reqMeta := &RequestMetaData{Auth: Auth{UserID: 1, UserName: "boss", Role: "admin"}}

	md := reqMeta.IntoMetadata()
	if got := md.Get(AuthUserRoleKey); got != "admin" {
		t.Errorf("AuthUserRoleKey 缺失或值错误: got %q want admin", got)
	}

	parsed := ParseMetadata(md)
	if parsed.Auth.Role != "admin" {
		t.Errorf("Role 往返不一致: %+v", parsed.Auth)
	}
	if parsed.Auth.UserID != 1 || parsed.Auth.UserName != "boss" {
		t.Errorf("Role 携带不应破坏既有字段: %+v", parsed.Auth)
	}
}

// B-103: 角色为空时不写入 metadata（与其它零值字段一致）
func TestMetaSkipsEmptyRole(t *testing.T) {
	reqMeta := &RequestMetaData{Auth: Auth{UserID: 5, UserName: "reader"}}
	md := reqMeta.IntoMetadata()
	if got := md.Get(AuthUserRoleKey); got != "" {
		t.Errorf("空 Role 不应写入: %q", got)
	}
}
