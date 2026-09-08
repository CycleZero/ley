package biz

import (
	"context"
	"strings"
	"testing"

	kerrors "github.com/go-kratos/kratos/v2/errors"
)

func seedUser(t *testing.T, repo *mockUserRepo) *User {
	t.Helper()
	return repo.Seed(&User{
		Username: "tester",
		Email:    "t@example.com",
		Password: "hash",
		Role:     RoleReader,
		Status:   UserStatusActive,
	})
}

// =============================================================================
// validateUsername / validatePassword 单元测试
// =============================================================================

func TestValidateUsername(t *testing.T) {
	valid := []string{"abc", "abc123", "user_name", "user-name", strings.Repeat("a", 32)}
	for _, name := range valid {
		if err := validateUsername(name); err != nil {
			t.Errorf("validateUsername(%q) 不应报错: %v", name, err)
		}
	}
	invalid := []string{"", "ab", strings.Repeat("a", 33), "中文", "a b", "a@b", "a.b"}
	for _, name := range invalid {
		if err := validateUsername(name); err == nil {
			t.Errorf("validateUsername(%q) 应报错", name)
		} else if !kerrors.IsBadRequest(err) {
			t.Errorf("validateUsername(%q) 应返回 400: %v", name, err)
		}
	}
}

func TestValidatePassword(t *testing.T) {
	valid := []string{"Password1", "Aa1" + strings.Repeat("x", 60), "pA1!@#xyz"}
	for _, pwd := range valid {
		if err := validatePassword(pwd); err != nil {
			t.Errorf("validatePassword(%q) 不应报错: %v", pwd, err)
		}
	}
	invalid := []struct {
		pwd  string
		want string
	}{
		{"", "PASSWORD_TOO_SHORT"},
		{"Short1", "PASSWORD_TOO_SHORT"}, // 6 位
		{strings.Repeat("Aa1", 25), "PASSWORD_TOO_LONG"}, // 75 位
		{"lowercase1", "PASSWORD_WEAK"},  // 缺大写
		{"UPPERCASE1", "PASSWORD_WEAK"},  // 缺小写
		{"OnlyLetters", "PASSWORD_WEAK"}, // 缺数字
	}
	for _, c := range invalid {
		err := validatePassword(c.pwd)
		if err == nil {
			t.Errorf("validatePassword(%q) 应报错", c.pwd)
			continue
		}
		if kerrors.FromError(err).Reason != c.want {
			t.Errorf("validatePassword(%q) 期望 %s, got %s", c.pwd, c.want, kerrors.FromError(err).Reason)
		}
	}
}

// =============================================================================
// GetProfile
// =============================================================================

func TestGetProfileSuccess(t *testing.T) {
	uc, repo := setupUserUseCase()
	seedUser(t, repo)

	user, err := uc.GetProfile(context.Background(), 1)
	if err != nil {
		t.Fatalf("GetProfile 失败: %v", err)
	}
	if user.Username != "tester" {
		t.Errorf("用户不匹配: %v", user.Username)
	}
}

func TestGetProfileNotFound(t *testing.T) {
	uc, _ := setupUserUseCase()
	_, err := uc.GetProfile(context.Background(), 999)
	if err == nil || !kerrors.IsNotFound(err) {
		t.Errorf("未知用户应返回 404: %v", err)
	}
}

// =============================================================================
// UpdateProfile
// =============================================================================

func TestUpdateProfileSuccess(t *testing.T) {
	uc, repo := setupUserUseCase()
	seedUser(t, repo)

	user, err := uc.UpdateProfile(context.Background(), 1, "https://avatar.png", "你好，我是测试用户")
	if err != nil {
		t.Fatalf("UpdateProfile 失败: %v", err)
	}
	if user.Avatar != "https://avatar.png" || user.Bio != "你好，我是测试用户" {
		t.Errorf("资料未更新: %+v", user)
	}
	// 持久化验证
	stored, _ := repo.FindByID(context.Background(), 1)
	if stored.Bio != "你好，我是测试用户" {
		t.Errorf("资料未持久化: %+v", stored)
	}
}

func TestUpdateProfileBioTooLong(t *testing.T) {
	uc, repo := setupUserUseCase()
	seedUser(t, repo)

	longBio := strings.Repeat("字", MaxBioLength+1)
	_, err := uc.UpdateProfile(context.Background(), 1, "", longBio)
	if err == nil || kerrors.FromError(err).Reason != "BIO_TOO_LONG" {
		t.Errorf("超长简介应返回 BIO_TOO_LONG: %v", err)
	}
}

func TestUpdateProfileNotFound(t *testing.T) {
	uc, _ := setupUserUseCase()
	_, err := uc.UpdateProfile(context.Background(), 999, "", "bio")
	if err == nil || !kerrors.IsNotFound(err) {
		t.Errorf("未知用户应返回 404: %v", err)
	}
}

// FIX-6: UpdateProfile 只能写 avatar/bio——不得回写加载快照中的其它字段。
// 缓存陈旧时（如管理员刚禁用账号而缓存尚未失效）全字段回写会把 status/role
// 旧值覆盖回库，禁用账号可借自身资料更新复活。
func TestUpdateProfileDoesNotRevertOtherFields(t *testing.T) {
	uc, repo := setupUserUseCase()
	// 库中真实状态：禁用 + reader
	stored := repo.Seed(&User{
		Username: "tester", Email: "t@example.com", Password: "hash",
		Role: RoleReader, Status: UserStatusDisabled,
	})
	// 模拟陈旧缓存快照：FindByID 返回禁用前的快照（active + admin）
	repo.stale = &User{
		ID: stored.ID, Username: "tester", Email: "t@example.com", Password: "hash",
		Role: RoleAdmin, Status: UserStatusActive,
	}

	if _, err := uc.UpdateProfile(context.Background(), stored.ID, "new-avatar", "new-bio"); err != nil {
		t.Fatalf("UpdateProfile 失败: %v", err)
	}

	got := repo.users[stored.ID]
	if got.Status != UserStatusDisabled {
		t.Errorf("UpdateProfile 不应回写 status: got %v want %v", got.Status, UserStatusDisabled)
	}
	if got.Role != RoleReader {
		t.Errorf("UpdateProfile 不应回写 role: got %v want %v", got.Role, RoleReader)
	}
	if got.Avatar != "new-avatar" || got.Bio != "new-bio" {
		t.Errorf("avatar/bio 应更新: %+v", got)
	}
}

// =============================================================================
// 其他用例
// =============================================================================

func TestUserFindByID(t *testing.T) {
	uc, repo := setupUserUseCase()
	seedUser(t, repo)

	if _, err := uc.FindByID(context.Background(), 1); err != nil {
		t.Errorf("FindByID 失败: %v", err)
	}
	if _, err := uc.FindByID(context.Background(), 999); err == nil {
		t.Error("未知 ID 应返回错误")
	}
}

func TestUserList(t *testing.T) {
	uc, repo := setupUserUseCase()
	repo.Seed(&User{Username: "u1", Email: "u1@example.com", Password: "h", Role: RoleReader, Status: UserStatusActive})
	repo.Seed(&User{Username: "u2", Email: "u2@example.com", Password: "h", Role: RoleReader, Status: UserStatusActive})

	users, total, err := uc.List(context.Background(), 1, 10)
	if err != nil {
		t.Fatalf("List 失败: %v", err)
	}
	if total != 2 || len(users) != 2 {
		t.Errorf("期望 2 个用户: total=%d len=%d", total, len(users))
	}
}

func TestUserUpdateStatus(t *testing.T) {
	uc, repo := setupUserUseCase()
	seedUser(t, repo)

	if err := uc.UpdateStatus(context.Background(), 1, UserStatusDisabled); err != nil {
		t.Fatalf("UpdateStatus 失败: %v", err)
	}
	user, _ := repo.FindByID(context.Background(), 1)
	if user.Status != UserStatusDisabled {
		t.Errorf("状态未更新: %v", user.Status)
	}
	if err := uc.UpdateStatus(context.Background(), 999, UserStatusDisabled); err == nil {
		t.Error("未知用户应返回错误")
	}
}

func TestUserDelete(t *testing.T) {
	uc, repo := setupUserUseCase()
	seedUser(t, repo)

	if err := uc.Delete(context.Background(), 1); err != nil {
		t.Fatalf("Delete 失败: %v", err)
	}
	if _, err := repo.FindByID(context.Background(), 1); err == nil {
		t.Error("删除后不应能查到")
	}
	if err := uc.Delete(context.Background(), 999); err == nil {
		t.Error("删除未知用户应报错")
	}
}
