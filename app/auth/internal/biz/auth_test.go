package biz

import (
	"context"
	"strings"
	"testing"

	"github.com/CycleZero/ley/pkg/jwt"
	"github.com/CycleZero/ley/pkg/security"
	"github.com/go-kratos/kratos/v2/log"
	kerrors "github.com/go-kratos/kratos/v2/errors"
)

// =============================================================================
// Register
// =============================================================================

func TestRegisterSuccess(t *testing.T) {
	uc, repo, eb, _ := setupAuthUseCase()
	ctx := context.Background()

	pair, user, err := uc.Register(ctx, "tester", "t@example.com", "Password123")
	if err != nil {
		t.Fatalf("Register 失败: %v", err)
	}
	if user.ID == 0 {
		t.Error("user.ID 不应为 0")
	}
	if user.Role != RoleReader {
		t.Errorf("默认角色应为 reader: %v", user.Role)
	}
	if user.Status != UserStatusActive {
		t.Errorf("默认状态应为 active: %v", user.Status)
	}
	// 密码必须哈希存储
	if user.Password == "Password123" || !security.VerifyPassword("Password123", user.Password) {
		t.Error("密码应以 bcrypt 哈希存储且可验证")
	}
	// token pair 应可解析
	if pair.AccessToken == "" || pair.RefreshToken == "" {
		t.Error("token pair 不应为空")
	}
	// 注册事件应发布
	topics := eb.PublishedTopics()
	if len(topics) != 1 || topics[0] != TopicUserRegistered {
		t.Errorf("应发布 user.registered 事件: %v", topics)
	}
	// 用户应可查
	if _, err := repo.FindByUsername(ctx, "tester"); err != nil {
		t.Errorf("注册后应能查到用户: %v", err)
	}
}

func TestRegisterInvalidUsername(t *testing.T) {
	uc, _, _, _ := setupAuthUseCase()
	cases := []string{
		"ab",              // 太短
		strings.Repeat("a", 33), // 太长
		"中文名",            // 非法字符
		"has space",       // 空格
		"has@symbol",      // 特殊符号
		"",                // 空
	}
	for _, name := range cases {
		if _, _, err := uc.Register(context.Background(), name, "t@example.com", "Password123"); err == nil {
			t.Errorf("用户名 %q 应注册失败", name)
		} else if !kerrors.IsBadRequest(err) {
			t.Errorf("用户名 %q 应返回 400: %v", name, err)
		}
	}
}

func TestRegisterInvalidPassword(t *testing.T) {
	uc, _, _, _ := setupAuthUseCase()
	cases := []struct {
		pwd  string
		want string
	}{
		{"short1", "PASSWORD_TOO_SHORT"},
		{strings.Repeat("A1b", 30), "PASSWORD_TOO_LONG"}, // 90 字符
		{"alllower123", "PASSWORD_WEAK"},                 // 缺大写
		{"ALLUPPER123", "PASSWORD_WEAK"},                 // 缺小写
		{"OnlyLetters", "PASSWORD_WEAK"},                 // 缺数字
	}
	for _, c := range cases {
		_, _, err := uc.Register(context.Background(), "tester", "t@example.com", c.pwd)
		if err == nil {
			t.Errorf("密码 %q 应注册失败", c.pwd)
			continue
		}
		ke := kerrors.FromError(err)
		if ke.Reason != c.want {
			t.Errorf("密码 %q 期望错误 %s, got %s", c.pwd, c.want, ke.Reason)
		}
	}
}

func TestRegisterUsernameTaken(t *testing.T) {
	uc, repo, _, _ := setupAuthUseCase()
	repo.Seed(&User{Username: "taken", Email: "other@example.com", Password: "hash", Role: RoleReader, Status: UserStatusActive})

	_, _, err := uc.Register(context.Background(), "taken", "new@example.com", "Password123")
	if err == nil || !kerrors.IsConflict(err) {
		t.Errorf("用户名占用应返回 409: %v", err)
	}
}

func TestRegisterEmailTaken(t *testing.T) {
	uc, repo, _, _ := setupAuthUseCase()
	repo.Seed(&User{Username: "other", Email: "dup@example.com", Password: "hash", Role: RoleReader, Status: UserStatusActive})

	_, _, err := uc.Register(context.Background(), "newuser", "dup@example.com", "Password123")
	if err == nil || !kerrors.IsConflict(err) {
		t.Errorf("邮箱占用应返回 409: %v", err)
	}
}

// =============================================================================
// Login
// =============================================================================

func TestLoginSuccessByUsername(t *testing.T) {
	uc, repo, _, _ := setupAuthUseCase()
	hash, _ := security.HashPassword("Password123")
	repo.Seed(&User{Username: "tester", Email: "t@example.com", Password: hash, Role: RoleReader, Status: UserStatusActive})

	pair, user, err := uc.Login(context.Background(), "tester", "Password123")
	if err != nil {
		t.Fatalf("Login 失败: %v", err)
	}
	if user.Username != "tester" {
		t.Errorf("用户不匹配: %v", user.Username)
	}
	if pair.AccessToken == "" {
		t.Error("应返回 access token")
	}
}

func TestLoginSuccessByEmail(t *testing.T) {
	uc, repo, _, _ := setupAuthUseCase()
	hash, _ := security.HashPassword("Password123")
	repo.Seed(&User{Username: "tester", Email: "t@example.com", Password: hash, Role: RoleReader, Status: UserStatusActive})

	if _, _, err := uc.Login(context.Background(), "t@example.com", "Password123"); err != nil {
		t.Errorf("邮箱登录应成功: %v", err)
	}
}

func TestLoginWrongPassword(t *testing.T) {
	uc, repo, _, _ := setupAuthUseCase()
	hash, _ := security.HashPassword("Password123")
	repo.Seed(&User{Username: "tester", Email: "t@example.com", Password: hash, Role: RoleReader, Status: UserStatusActive})

	_, _, err := uc.Login(context.Background(), "tester", "WrongPass1")
	if err == nil || !kerrors.IsUnauthorized(err) {
		t.Errorf("错误密码应返回 401: %v", err)
	}
}

func TestLoginUnknownAccount(t *testing.T) {
	uc, _, _, _ := setupAuthUseCase()
	// 未知账号与错误密码返回同一错误（防用户枚举）
	_, _, err := uc.Login(context.Background(), "ghost", "Password123")
	if err == nil || !kerrors.IsUnauthorized(err) {
		t.Errorf("未知账号应返回 401: %v", err)
	}
}

func TestLoginDisabledAccount(t *testing.T) {
	uc, repo, _, _ := setupAuthUseCase()
	hash, _ := security.HashPassword("Password123")
	repo.Seed(&User{Username: "banned", Email: "b@example.com", Password: hash, Role: RoleReader, Status: UserStatusDisabled})

	_, _, err := uc.Login(context.Background(), "banned", "Password123")
	if err == nil || !kerrors.IsForbidden(err) {
		t.Errorf("禁用账号应返回 403: %v", err)
	}
}

// =============================================================================
// RefreshToken（Token 轮换）
// =============================================================================

func TestRefreshTokenSuccess(t *testing.T) {
	uc, repo, _, bl := setupAuthUseCase()
	hash, _ := security.HashPassword("Password123")
	repo.Seed(&User{Username: "tester", Email: "t@example.com", Password: hash, Role: RoleReader, Status: UserStatusActive})

	// 先登录拿 refresh token
	pair, _, err := uc.Login(context.Background(), "tester", "Password123")
	if err != nil {
		t.Fatalf("Login 失败: %v", err)
	}

	newPair, user, err := uc.RefreshToken(context.Background(), pair.RefreshToken)
	if err != nil {
		t.Fatalf("RefreshToken 失败: %v", err)
	}
	if newPair.AccessToken == pair.AccessToken {
		t.Error("刷新后 access token 应轮换")
	}
	if newPair.RefreshToken == pair.RefreshToken {
		t.Error("刷新后 refresh token 应轮换")
	}
	if user.Username != "tester" {
		t.Errorf("用户不匹配: %v", user.Username)
	}
	// 旧 refresh token 应已入黑名单（防重放）
	if !bl.IsTokenBlackListed(pair.RefreshToken) {
		t.Error("旧 refresh token 应入黑名单")
	}
	// 新 token 未被拉黑
	if bl.IsTokenBlackListed(newPair.RefreshToken) {
		t.Error("新 refresh token 不应被拉黑")
	}
}

func TestRefreshTokenReplayRejected(t *testing.T) {
	uc, repo, _, _ := setupAuthUseCase()
	hash, _ := security.HashPassword("Password123")
	repo.Seed(&User{Username: "tester", Email: "t@example.com", Password: hash, Role: RoleReader, Status: UserStatusActive})

	pair, _, _ := uc.Login(context.Background(), "tester", "Password123")
	// 第一次刷新成功
	if _, _, err := uc.RefreshToken(context.Background(), pair.RefreshToken); err != nil {
		t.Fatalf("首次刷新失败: %v", err)
	}
	// 第二次用同一 refresh token 重放必须被拒绝
	_, _, err := uc.RefreshToken(context.Background(), pair.RefreshToken)
	if err == nil || !kerrors.IsUnauthorized(err) {
		t.Errorf("重放旧 refresh token 应返回 401: %v", err)
	}
}

func TestRefreshTokenInvalid(t *testing.T) {
	uc, _, _, _ := setupAuthUseCase()
	// 用 access token 冒充 refresh token
	j := newTestJWT()
	pair, _ := j.GenerateTokenPair(jwt.Payload{UserId: 1})
	_, _, err := uc.RefreshToken(context.Background(), pair.AccessToken)
	if err == nil || !kerrors.IsUnauthorized(err) {
		t.Errorf("access token 冒充 refresh 应返回 401: %v", err)
	}
	// 垃圾字符串
	if _, _, err := uc.RefreshToken(context.Background(), "garbage"); err == nil {
		t.Error("垃圾字符串应刷新失败")
	}
}

func TestRefreshTokenUnknownUser(t *testing.T) {
	uc, _, _, _ := setupAuthUseCase()
	j := newTestJWT()
	pair, _ := j.GenerateTokenPair(jwt.Payload{UserId: 999})

	_, _, err := uc.RefreshToken(context.Background(), pair.RefreshToken)
	if err == nil || !kerrors.IsNotFound(err) {
		t.Errorf("未知用户应返回 404: %v", err)
	}
}

func TestRefreshTokenDisabledUser(t *testing.T) {
	uc, repo, _, _ := setupAuthUseCase()
	hash, _ := security.HashPassword("Password123")
	repo.Seed(&User{Username: "banned", Email: "b@example.com", Password: hash, Role: RoleReader, Status: UserStatusDisabled})

	j := newTestJWT()
	pair, _ := j.GenerateTokenPair(jwt.Payload{UserId: 1}) // Seed 的 ID 是 1

	_, _, err := uc.RefreshToken(context.Background(), pair.RefreshToken)
	if err == nil || !kerrors.IsForbidden(err) {
		t.Errorf("禁用账号刷新应返回 403: %v", err)
	}
}

// =============================================================================
// Logout
// =============================================================================

func TestLogoutBlacklistsBothTokens(t *testing.T) {
	uc, repo, _, bl := setupAuthUseCase()
	hash, _ := security.HashPassword("Password123")
	repo.Seed(&User{Username: "tester", Email: "t@example.com", Password: hash, Role: RoleReader, Status: UserStatusActive})

	pair, _, _ := uc.Login(context.Background(), "tester", "Password123")

	if err := uc.Logout(context.Background(), pair.AccessToken, pair.RefreshToken); err != nil {
		t.Fatalf("Logout 失败: %v", err)
	}
	if !bl.IsTokenBlackListed(pair.AccessToken) {
		t.Error("access token 应入黑名单")
	}
	if !bl.IsTokenBlackListed(pair.RefreshToken) {
		t.Error("refresh token 应入黑名单")
	}
	// 登出后 refresh 应失败
	if _, _, err := uc.RefreshToken(context.Background(), pair.RefreshToken); err == nil {
		t.Error("登出后 refresh 应失败")
	}
}

func TestLogoutIgnoresInvalidTokens(t *testing.T) {
	uc, _, _, _ := setupAuthUseCase()
	// 无效 token 不应报错
	if err := uc.Logout(context.Background(), "invalid-token", "garbage"); err != nil {
		t.Errorf("无效 token 登出不应报错: %v", err)
	}
	// 空 token 也不应报错
	if err := uc.Logout(context.Background(), "", ""); err != nil {
		t.Errorf("空 token 登出不应报错: %v", err)
	}
}

func TestLogoutWithDisabledBlacklist(t *testing.T) {
	repo := newMockUserRepo()
	eb := &mockEventBus{}
	bl := jwt.NewBlackList(nil) // 禁用黑名单
	uc := NewAuthUseCase(repo, newTestJWT(), bl, eb, log.DefaultLogger)

	if err := uc.Logout(context.Background(), "any", "any"); err != nil {
		t.Errorf("黑名单禁用时登出不应报错: %v", err)
	}
}
