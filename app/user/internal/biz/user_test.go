package biz

import (
	"context"
	"errors"
	"testing"

	"ley/app/user/internal/model"

	klog "github.com/go-kratos/kratos/v2/log"
	"io"
)

// =============================================================================
// Mock Repo — 内存实现，返回预设数据/错误
// =============================================================================

type mockUserRepo struct {
	users       map[uint]*model.User
	byUsername  map[string]*model.User
	byEmail     map[string]*model.User
	byUUID      map[string]*model.User
	byAccount   map[string]*model.User
	createErr   error
	findErr     error
}

func newMockUserRepo() *mockUserRepo {
	return &mockUserRepo{
		users:      make(map[uint]*model.User),
		byUsername: make(map[string]*model.User),
		byEmail:    make(map[string]*model.User),
		byUUID:     make(map[string]*model.User),
		byAccount:  make(map[string]*model.User),
	}
}

func (m *mockUserRepo) Create(ctx context.Context, user *model.User) error {
	if m.createErr != nil {
		return m.createErr
	}
	user.ID = uint(len(m.users) + 1)
	return nil
}
func (m *mockUserRepo) Update(ctx context.Context, user *model.User) error { return nil }
func (m *mockUserRepo) Delete(ctx context.Context, id uint) error           { return nil }
func (m *mockUserRepo) FindByID(ctx context.Context, id uint) (*model.User, error) {
	if u, ok := m.users[id]; ok {
		return u, nil
	}
	return nil, ErrUserNotFound
}
func (m *mockUserRepo) FindByUUID(ctx context.Context, uuid string) (*model.User, error) {
	if u, ok := m.byUUID[uuid]; ok {
		return u, nil
	}
	return nil, ErrUserNotFound
}
func (m *mockUserRepo) FindByUsername(ctx context.Context, username string) (*model.User, error) {
	if u, ok := m.byUsername[username]; ok {
		return u, nil
	}
	return nil, ErrUserNotFound
}
func (m *mockUserRepo) FindByEmail(ctx context.Context, email string) (*model.User, error) {
	if u, ok := m.byEmail[email]; ok {
		return u, nil
	}
	return nil, ErrUserNotFound
}
func (m *mockUserRepo) FindByAccount(ctx context.Context, account string) (*model.User, error) {
	if u, ok := m.byAccount[account]; ok {
		return u, nil
	}
	return nil, ErrUserNotFound
}
func (m *mockUserRepo) List(ctx context.Context, page, pageSize int) ([]*model.User, int64, error) {
	return nil, 0, nil
}
func (m *mockUserRepo) UpdateStatus(ctx context.Context, id uint, status model.UserStatus) error {
	return nil
}

// =============================================================================
// createTestUserUseCase creates a UserUseCase with mock repo and discard logger
// =============================================================================
func createTestUserUseCase(repo *mockUserRepo) *UserUseCase {
	return &UserUseCase{repo: repo, logger: klog.NewStdLogger(io.Discard)}
}

// =============================================================================
// Password Validation Tests
// =============================================================================

func TestValidatePassword(t *testing.T) {
	tests := []struct {
		name     string
		password string
		wantErr  error
	}{
		{name: "合法密码", password: "Abc12345", wantErr: nil},
		{name: "合法密码含特殊字符", password: "Str0ng!Pass", wantErr: nil},
		{name: "过短 7 位", password: "Abc1234", wantErr: ErrPasswordTooShort},
		{name: "过短 1 位", password: "A", wantErr: ErrPasswordTooShort},
		{name: "空密码", password: "", wantErr: ErrPasswordTooShort},
		{name: "超长 65 位", password: string(make([]byte, 65)), wantErr: ErrPasswordTooLong},
		{name: "仅小写", password: "abcdefgh", wantErr: ErrPasswordWeak},
		{name: "仅大写", password: "ABCDEFGH", wantErr: ErrPasswordWeak},
		{name: "仅数字", password: "12345678", wantErr: ErrPasswordWeak},
		{name: "缺大写", password: "abc12345", wantErr: ErrPasswordWeak},
		{name: "缺小写", password: "ABC12345", wantErr: ErrPasswordWeak},
		{name: "缺数字", password: "Abcdefgh", wantErr: ErrPasswordWeak},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validatePassword(tt.password)
			if tt.wantErr == nil {
				if err != nil {
					t.Errorf("validatePassword(%q) = %v, want nil", tt.password, err)
				}
			} else {
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("validatePassword(%q) = %v, want %v", tt.password, err, tt.wantErr)
				}
			}
		})
	}
}

// =============================================================================
// Username Validation Tests
// =============================================================================

func TestValidateUsername(t *testing.T) {
	tests := []struct {
		name     string
		username string
		wantErr  error
	}{
		{name: "合法用户名", username: "alice_123", wantErr: nil},
		{name: "合法纯字母", username: "bob", wantErr: nil},
		{name: "合法含连字符", username: "hello-world", wantErr: nil},
		{name: "合法含下划线", username: "test_user", wantErr: nil},
		{name: "过短 2 位", username: "ab", wantErr: ErrUsernameInvalid},
		{name: "超长 33 位", username: "abcdefghijklmnopqrstuvwxyz1234567", wantErr: ErrUsernameInvalid},
		{name: "含空格", username: "hello world", wantErr: ErrUsernameInvalid},
		{name: "含特殊字符", username: "hello@world", wantErr: ErrUsernameInvalid},
		{name: "中文", username: "用户", wantErr: ErrUsernameInvalid},
		{name: "空字符串", username: "", wantErr: ErrUsernameInvalid},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateUsername(tt.username)
			if tt.wantErr == nil {
				if err != nil {
					t.Errorf("validateUsername(%q) = %v, want nil", tt.username, err)
				}
			} else {
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("validateUsername(%q) = %v, want %v", tt.username, err, tt.wantErr)
				}
			}
		})
	}
}

// =============================================================================
// Register Tests
// =============================================================================

func TestRegister(t *testing.T) {
	ctx := context.Background()

	t.Run("成功注册", func(t *testing.T) {
		repo := newMockUserRepo()
		uc := createTestUserUseCase(repo)

		user, err := uc.Register(ctx, "alice_12", "alice@test.com", "Abc12345", "Alice")
		if err != nil {
			t.Fatalf("Register failed: %v", err)
		}
		if user.Username != "alice_12" {
			t.Errorf("Username = %q, want %q", user.Username, "alice_12")
		}
		if user.Status != model.UserStatusActive {
			t.Errorf("Status = %d, want Active(0)", user.Status)
		}
		if user.Role != model.RoleReader {
			t.Errorf("Role = %s, want reader", user.Role)
		}
		if user.UUID == "" {
			t.Error("UUID should not be empty")
		}
	})

	t.Run("用户名已被占用", func(t *testing.T) {
		repo := newMockUserRepo()
		repo.byUsername["alice"] = &model.User{Username: "alice"}
		uc := createTestUserUseCase(repo)

		_, err := uc.Register(ctx, "alice", "new@test.com", "Abc12345", "Alice")
		if !errors.Is(err, ErrUsernameTaken) {
			t.Errorf("expected ErrUsernameTaken, got %v", err)
		}
	})

	t.Run("邮箱已被注册", func(t *testing.T) {
		repo := newMockUserRepo()
		repo.byEmail["alice@test.com"] = &model.User{Email: "alice@test.com"}
		uc := createTestUserUseCase(repo)

		_, err := uc.Register(ctx, "new_user", "alice@test.com", "Abc12345", "Alice")
		if !errors.Is(err, ErrEmailTaken) {
			t.Errorf("expected ErrEmailTaken, got %v", err)
		}
	})

	t.Run("密码强度不足", func(t *testing.T) {
		repo := newMockUserRepo()
		uc := createTestUserUseCase(repo)

		_, err := uc.Register(ctx, "alice", "alice@test.com", "abcdefgh", "Alice")
		if !errors.Is(err, ErrPasswordWeak) {
			t.Errorf("expected ErrPasswordWeak, got %v", err)
		}
	})

	t.Run("用户名格式非法", func(t *testing.T) {
		repo := newMockUserRepo()
		uc := createTestUserUseCase(repo)

		_, err := uc.Register(ctx, "a b", "alice@test.com", "Abc12345", "Alice")
		if !errors.Is(err, ErrUsernameInvalid) {
			t.Errorf("expected ErrUsernameInvalid, got %v", err)
		}
	})
}

// =============================================================================
// Login Tests
// =============================================================================

func TestLogin(t *testing.T) {
	ctx := context.Background()

	t.Run("账号不存在返回 ErrBadCredentials", func(t *testing.T) {
		repo := newMockUserRepo()
		uc := createTestUserUseCase(repo)

		_, err := uc.Login(ctx, "noone", "Abc12345")
		if !errors.Is(err, ErrBadCredentials) {
			t.Errorf("expected ErrBadCredentials, got %v", err)
		}
	})
}

// =============================================================================
// Error Sentinel Isolation
// =============================================================================

func TestErrorSentinels(t *testing.T) {
	tests := []struct {
		name        string
		parent      error
		child       error
		shouldMatch bool
	}{
		{name: "ErrBadCredentials wraps correctly", parent: ErrBadCredentials, child: ErrBadCredentials, shouldMatch: true},
		{name: "ErrUserNotFound ≠ ErrBadCredentials", parent: ErrUserNotFound, child: ErrBadCredentials, shouldMatch: false},
		{name: "ErrUserNotFound ≠ ErrUserDuplicate", parent: ErrUserNotFound, child: ErrUserDuplicate, shouldMatch: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if errors.Is(tt.parent, tt.child) != tt.shouldMatch {
				t.Errorf("errors.Is(%v, %v) = %v, want %v",
					tt.parent, tt.child, !tt.shouldMatch, tt.shouldMatch)
			}
		})
	}
}
