package service

import (
	"errors"
	"testing"
	"time"

	userv1 "ley/api/user/v1"
	"ley/app/user/internal/biz"
	"ley/app/user/internal/model"

	klog "github.com/go-kratos/kratos/v2/log"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"io"
)

func testUserService() *UserService {
	return &UserService{logger: klog.NewStdLogger(io.Discard)}
}

// =============================================================================
// mapError Tests
// =============================================================================

func TestUserService_MapError(t *testing.T) {
	svc := testUserService()

	tests := []struct {
		name     string
		err      error
		wantCode codes.Code
	}{
		{name: "用户不存在→NotFound", err: biz.ErrUserNotFound, wantCode: codes.NotFound},
		{name: "重复用户→AlreadyExists", err: biz.ErrUserDuplicate, wantCode: codes.AlreadyExists},
		{name: "用户名占用→AlreadyExists", err: biz.ErrUsernameTaken, wantCode: codes.AlreadyExists},
		{name: "邮箱占用→AlreadyExists", err: biz.ErrEmailTaken, wantCode: codes.AlreadyExists},
		{name: "密码错误→Unauthenticated", err: biz.ErrBadCredentials, wantCode: codes.Unauthenticated},
		{name: "账号禁用→PermissionDenied", err: biz.ErrAccountDisabled, wantCode: codes.PermissionDenied},
		{name: "密码过短→InvalidArgument", err: biz.ErrPasswordTooShort, wantCode: codes.InvalidArgument},
		{name: "密码过弱→InvalidArgument", err: biz.ErrPasswordWeak, wantCode: codes.InvalidArgument},
		{name: "用户名非法→InvalidArgument", err: biz.ErrUsernameInvalid, wantCode: codes.InvalidArgument},
		{name: "昵称过长→InvalidArgument", err: biz.ErrNicknameTooLong, wantCode: codes.InvalidArgument},
		{name: "简介过长→InvalidArgument", err: biz.ErrBioTooLong, wantCode: codes.InvalidArgument},
		{name: "未知错误→Internal", err: errors.New("some random error"), wantCode: codes.Internal},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := svc.mapError(tt.err)
			if got == nil {
				t.Fatal("mapError returned nil")
			}
			s, ok := status.FromError(got)
			if !ok {
				t.Fatalf("not gRPC status: %v", got)
			}
			if s.Code() != tt.wantCode {
				t.Errorf("mapError(%v) code = %s, want %s", tt.err, s.Code(), tt.wantCode)
			}
		})
	}
}

// =============================================================================
// toUserInfo Tests
// =============================================================================

func TestToUserInfo_Normal(t *testing.T) {
	now := time.Date(2026, 1, 15, 10, 30, 0, 0, time.UTC)
	user := &model.User{
		Username: "alice", Email: "alice@example.com", Nickname: "Alice",
		Avatar: "https://minio/avatars/alice.png", Bio: "Developer", Role: model.RoleAuthor,
	}
	user.UUID = "uuid-001"
	user.ID = 42
	user.CreatedAt = now
	user.UpdatedAt = now

	info := toUserInfo(user)
	if info == nil {
		t.Fatal("toUserInfo returned nil")
	}
	if info.Id != "uuid-001" {
		t.Errorf("Id = %q, want %q", info.Id, "uuid-001")
	}
	if info.Role != "author" {
		t.Errorf("Role = %q, want author", info.Role)
	}
	if info.CreatedAt != "2026-01-15T10:30:00Z" {
		t.Errorf("CreatedAt = %q", info.CreatedAt)
	}
}

func TestToUserInfo_Nil(t *testing.T) {
	if info := toUserInfo(nil); info != nil {
		t.Errorf("toUserInfo(nil) = %v, want nil", info)
	}
}

func TestToUserInfo_ZeroValue(t *testing.T) {
	info := toUserInfo(&model.User{})
	if info == nil {
		t.Fatal("zero-value should not return nil")
	}
	if info.Id != "" {
		t.Errorf("Id should be empty, got %q", info.Id)
	}
	if info.Username != "" {
		t.Errorf("Username should be empty, got %q", info.Username)
	}
}

var _ userv1.UserInfo
