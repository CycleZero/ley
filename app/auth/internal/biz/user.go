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
	UpdateProfile(ctx context.Context, id uint, avatar, bio string) error
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

// NewUserUseCase 构造用户资料用例。
func NewUserUseCase(repo UserRepo, logger log.Logger) *UserUseCase {
	return &UserUseCase{repo: repo, log: log.NewHelper(logger)}
}

// GetProfile 获取用户资料；用户不存在时统一返回 ErrUserNotFound（不泄漏底层错误）。
func (uc *UserUseCase) GetProfile(ctx context.Context, userID uint) (*User, error) {
	user, err := uc.repo.FindByID(ctx, userID)
	if err != nil {
		uc.log.WithContext(ctx).Warnf("获取资料失败：用户不存在 id=%d", userID)
		return nil, ErrUserNotFound
	}
	return user, nil
}

// UpdateProfile 更新用户资料（仅头像与简介）。
// 先校验简介长度，再确认用户存在，最后只写 avatar/bio 两列，避免全字段回写竞态。
func (uc *UserUseCase) UpdateProfile(ctx context.Context, userID uint, avatar, bio string) (*User, error) {
	if len(bio) > MaxBioLength {
		uc.log.WithContext(ctx).Warnf("更新资料失败：简介超长 id=%d len=%d", userID, len(bio))
		return nil, ErrBioTooLong
	}
	user, err := uc.repo.FindByID(ctx, userID)
	if err != nil {
		uc.log.WithContext(ctx).Warnf("更新资料失败：用户不存在 id=%d", userID)
		return nil, ErrUserNotFound
	}
	if err := uc.repo.UpdateProfile(ctx, userID, avatar, bio); err != nil {
		uc.log.WithContext(ctx).Errorf("更新资料写入失败 id=%d: %v", userID, err)
		return nil, fmt.Errorf("update profile: %w", err)
	}
	user.Avatar = avatar
	user.Bio = bio
	uc.log.WithContext(ctx).Infof("更新资料成功 id=%d", userID)
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

// UpdateStatus 更新用户状态（启用/禁用）；禁用属关键状态迁移，记录 Info 审计日志。
func (uc *UserUseCase) UpdateStatus(ctx context.Context, id uint, status UserStatus) error {
	if err := uc.repo.UpdateStatus(ctx, id, status); err != nil {
		uc.log.WithContext(ctx).Errorf("更新用户状态失败 id=%d status=%s: %v", id, status, err)
		return err
	}
	uc.log.WithContext(ctx).Infof("更新用户状态成功 id=%d status=%s", id, status)
	return nil
}

// Delete 删除用户；删除属关键状态迁移，记录 Info 审计日志。
func (uc *UserUseCase) Delete(ctx context.Context, id uint) error {
	if err := uc.repo.Delete(ctx, id); err != nil {
		uc.log.WithContext(ctx).Errorf("删除用户失败 id=%d: %v", id, err)
		return err
	}
	uc.log.WithContext(ctx).Infof("删除用户成功 id=%d", id)
	return nil
}

// =============================================================================
// 校验函数
// =============================================================================

// validateUsername 校验用户名：长度符合且仅允许字母、数字、下划线、连字符。
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

// validatePassword 校验密码强度：长度合规且同时包含大写、小写、数字。
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
