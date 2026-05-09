package biz

import (
	"context"
	"fmt"
	"time"

	kerrors "github.com/go-kratos/kratos/v2/errors"
	"github.com/go-kratos/kratos/v2/log"
)

// =============================================================================
// UserStatus / UserRole
// =============================================================================

type UserStatus int8

const (
	UserStatusActive   UserStatus = 0
	UserStatusDisabled UserStatus = 1
)

type UserRole string

const (
	RoleReader UserRole = "reader"
	RoleAuthor UserRole = "author"
	RoleAdmin  UserRole = "admin"
)

// =============================================================================
// User — 用户业务模型
// =============================================================================

type User struct {
	ID        uint
	Username  string
	Email     string
	Password  string
	Avatar    string
	Bio       string
	Status    UserStatus
	Role      UserRole
	CreatedAt time.Time
	UpdatedAt time.Time
}

// =============================================================================
// 业务错误定义
// =============================================================================

var (
	ErrPasswordTooShort = kerrors.BadRequest("PASSWORD_TOO_SHORT", "密码至少 8 个字符")
	ErrPasswordTooLong  = kerrors.BadRequest("PASSWORD_TOO_LONG", "密码最多 64 个字符")
	ErrPasswordWeak     = kerrors.BadRequest("PASSWORD_WEAK", "密码必须包含大写字母、小写字母和数字")
	ErrUsernameInvalid  = kerrors.BadRequest("USERNAME_INVALID", "用户名须为 3-32 位字母、数字、下划线或连字符")
	ErrBioTooLong       = kerrors.BadRequest("BIO_TOO_LONG", "个人简介最多 500 个字符")

	ErrUserNotFound    = kerrors.NotFound("USER_NOT_FOUND", "用户不存在")
	ErrUserDuplicate   = kerrors.Conflict("USER_DUPLICATE", "用户名或邮箱已存在")
	ErrUsernameTaken   = kerrors.Conflict("USERNAME_TAKEN", "用户名已被占用")
	ErrEmailTaken      = kerrors.Conflict("EMAIL_TAKEN", "邮箱已被注册")
	ErrBadCredentials  = kerrors.Unauthorized("BAD_CREDENTIALS", "用户名/邮箱或密码错误")
	ErrAccountDisabled = kerrors.Forbidden("ACCOUNT_DISABLED", "账号已被禁用")
)

// =============================================================================
// 字段长度常量
// =============================================================================

const (
	MinPasswordLength = 8
	MaxPasswordLength = 64
	MinUsernameLength = 3
	MaxUsernameLength = 32
	MaxBioLength      = 500
)

// =============================================================================
// UserRepo — 数据访问接口
// =============================================================================

type UserRepo interface {
	Create(ctx context.Context, user *User) error
	Update(ctx context.Context, user *User) error
	Delete(ctx context.Context, id uint) error
	FindByID(ctx context.Context, id uint) (*User, error)
	FindByUsername(ctx context.Context, username string) (*User, error)
	FindByEmail(ctx context.Context, email string) (*User, error)
	FindByAccount(ctx context.Context, account string) (*User, error)
	List(ctx context.Context, page, pageSize int) ([]*User, int64, error)
	UpdateStatus(ctx context.Context, id uint, status UserStatus) error
}

// =============================================================================
// UserUseCase — 用户资料管理
// =============================================================================

type UserUseCase struct {
	repo UserRepo
	log  *log.Helper
}

func NewUserUseCase(repo UserRepo, logger log.Logger) *UserUseCase {
	return &UserUseCase{repo: repo, log: log.NewHelper(logger)}
}

// GetProfile 获取用户资料
func (uc *UserUseCase) GetProfile(ctx context.Context, userID uint) (*User, error) {
	user, err := uc.repo.FindByID(ctx, userID)
	if err != nil {
		return nil, ErrUserNotFound
	}
	return user, nil
}

// UpdateProfile 更新用户资料
func (uc *UserUseCase) UpdateProfile(ctx context.Context, userID uint, avatar, bio string) (*User, error) {
	if len(bio) > MaxBioLength {
		return nil, ErrBioTooLong
	}
	user, err := uc.repo.FindByID(ctx, userID)
	if err != nil {
		return nil, ErrUserNotFound
	}
	user.Avatar = avatar
	user.Bio = bio
	if err := uc.repo.Update(ctx, user); err != nil {
		return nil, fmt.Errorf("update profile: %w", err)
	}
	return user, nil
}

// FindByID 按 ID 查询用户
func (uc *UserUseCase) FindByID(ctx context.Context, id uint) (*User, error) {
	user, err := uc.repo.FindByID(ctx, id)
	if err != nil {
		return nil, ErrUserNotFound
	}
	return user, nil
}

// List 分页查询用户列表
func (uc *UserUseCase) List(ctx context.Context, page, pageSize int) ([]*User, int64, error) {
	return uc.repo.List(ctx, page, pageSize)
}

// UpdateStatus 更新用户状态（启用/禁用）
func (uc *UserUseCase) UpdateStatus(ctx context.Context, id uint, status UserStatus) error {
	return uc.repo.UpdateStatus(ctx, id, status)
}

// Delete 删除用户
func (uc *UserUseCase) Delete(ctx context.Context, id uint) error {
	return uc.repo.Delete(ctx, id)
}

// =============================================================================
// 校验函数
// =============================================================================

func validateUsername(username string) error {
	if len(username) < MinUsernameLength || len(username) > MaxUsernameLength {
		return ErrUsernameInvalid
	}
	for _, r := range username {
		if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') ||
			(r >= '0' && r <= '9') || r == '_' || r == '-') {
			return ErrUsernameInvalid
		}
	}
	return nil
}

func validatePassword(password string) error {
	if len(password) < MinPasswordLength {
		return ErrPasswordTooShort
	}
	if len(password) > MaxPasswordLength {
		return ErrPasswordTooLong
	}
	var hasUpper, hasLower, hasDigit bool
	for _, r := range password {
		switch {
		case r >= 'A' && r <= 'Z':
			hasUpper = true
		case r >= 'a' && r <= 'z':
			hasLower = true
		case r >= '0' && r <= '9':
			hasDigit = true
		}
	}
	if !hasUpper || !hasLower || !hasDigit {
		return ErrPasswordWeak
	}
	return nil
}
