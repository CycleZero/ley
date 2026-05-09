package biz

import (
	"context"
	"fmt"

	"ley/pkg/eventbus"
	"ley/pkg/jwt"
	"ley/pkg/security"

	kerrors "github.com/go-kratos/kratos/v2/errors"
	"github.com/go-kratos/kratos/v2/log"
)

// =============================================================================
// AuthUseCase — 认证业务用例（注册、登录、Token 管理）
// =============================================================================

type AuthUseCase struct {
	repo      UserRepo
	jwt       jwt.JWT
	blacklist jwt.BlackListCache
	eb        eventbus.EventBus
	log       *log.Helper
}

func NewAuthUseCase(repo UserRepo, j jwt.JWT, bl jwt.BlackListCache, eb eventbus.EventBus, logger log.Logger) *AuthUseCase {
	return &AuthUseCase{
		repo:      repo,
		jwt:       j,
		blacklist: bl,
		eb:        eb,
		log:       log.NewHelper(logger),
	}
}

// =============================================================================
// Register — 用户注册
// =============================================================================

func (uc *AuthUseCase) Register(ctx context.Context, username, email, password string) (*jwt.TokenPair, *User, error) {
	if err := validateUsername(username); err != nil {
		return nil, nil, err
	}
	if err := validatePassword(password); err != nil {
		return nil, nil, err
	}

	if _, err := uc.repo.FindByUsername(ctx, username); err == nil {
		return nil, nil, ErrUsernameTaken
	}
	if _, err := uc.repo.FindByEmail(ctx, email); err == nil {
		return nil, nil, ErrEmailTaken
	}

	hashed, err := security.HashPassword(password)
	if err != nil {
		uc.log.WithContext(ctx).Errorf("密码哈希失败: %v", err)
		return nil, nil, kerrors.InternalServer("HASH_FAILED", "密码处理失败")
	}

	user := &User{
		Username: username,
		Email:    email,
		Password: hashed,
		Status:   UserStatusActive,
		Role:     RoleReader,
	}

	if err := uc.repo.Create(ctx, user); err != nil {
		if kerrors.IsConflict(err) || kerrors.Code(err) == 409 {
			return nil, nil, err
		}
		uc.log.WithContext(ctx).Errorf("创建用户失败: %v", err)
		return nil, nil, fmt.Errorf("register: %w", err)
	}

	pair, err := uc.jwt.GenerateTokenPair(jwt.Payload{
		UserId:   uint64(user.ID),
		UserName: user.Username,
	})
	if err != nil {
		uc.log.WithContext(ctx).Errorf("生成令牌失败 id=%d: %v", user.ID, err)
		return nil, nil, kerrors.InternalServer("TOKEN_FAILED", "令牌生成失败")
	}

	uc.log.WithContext(ctx).Infof("用户注册成功 id=%d username=%s", user.ID, user.Username)

	_ = uc.eb.PublishAsync(ctx, TopicUserRegistered, &UserRegisteredEvent{
		UserID:   uint64(user.ID),
		Username: user.Username,
		Email:    user.Email,
	})

	return pair, user, nil
}

// =============================================================================
// Login — 登录：验证凭证 + 生成 TokenPair
// =============================================================================

func (uc *AuthUseCase) Login(ctx context.Context, account, password string) (*jwt.TokenPair, *User, error) {
	user, err := uc.repo.FindByAccount(ctx, account)
	if err != nil {
		return nil, nil, ErrBadCredentials
	}

	if user.Status != UserStatusActive {
		return nil, nil, ErrAccountDisabled
	}

	if !security.VerifyPassword(password, user.Password) {
		return nil, nil, ErrBadCredentials
	}

	pair, err := uc.jwt.GenerateTokenPair(jwt.Payload{
		UserId:   uint64(user.ID),
		UserName: user.Username,
	})
	if err != nil {
		uc.log.WithContext(ctx).Errorf("生成令牌失败 id=%d: %v", user.ID, err)
		return nil, nil, kerrors.InternalServer("TOKEN_FAILED", "令牌生成失败")
	}

	uc.log.WithContext(ctx).Infof("登录成功 id=%d username=%s", user.ID, user.Username)
	return pair, user, nil
}

// =============================================================================
// RefreshToken — 刷新令牌（Token Rotation）
// =============================================================================

func (uc *AuthUseCase) RefreshToken(ctx context.Context, refreshToken string) (*jwt.TokenPair, *User, error) {
	claims, err := uc.jwt.ParseRefreshToken(refreshToken)
	if err != nil {
		return nil, nil, kerrors.Unauthorized("TOKEN_INVALID", "令牌无效或已过期")
	}

	if uc.blacklist.IsEnabled() && uc.blacklist.IsTokenBlackListed(refreshToken) {
		return nil, nil, kerrors.Unauthorized("TOKEN_BLACKLISTED", "令牌已登出")
	}

	user, err := uc.repo.FindByID(ctx, uint(claims.UserId))
	if err != nil {
		return nil, nil, ErrUserNotFound
	}

	if user.Status != UserStatusActive {
		return nil, nil, ErrAccountDisabled
	}

	pair, err := uc.jwt.GenerateTokenPair(jwt.Payload{
		UserId:   uint64(user.ID),
		UserName: user.Username,
	})
	if err != nil {
		uc.log.WithContext(ctx).Errorf("刷新令牌生成失败 id=%d: %v", user.ID, err)
		return nil, nil, fmt.Errorf("refresh token: %w", err)
	}

	if uc.blacklist.IsEnabled() {
		if err := uc.blacklist.Add(refreshToken); err != nil {
			uc.log.WithContext(ctx).Warnf("旧令牌加入黑名单失败: %v", err)
		}
	}

	uc.log.WithContext(ctx).Infof("令牌刷新成功 id=%d", user.ID)
	return pair, user, nil
}

// =============================================================================
// Logout — 登出：令牌加入黑名单
// =============================================================================

func (uc *AuthUseCase) Logout(ctx context.Context, accessToken, refreshToken string) error {
	if !uc.blacklist.IsEnabled() {
		return nil
	}

	if accessToken != "" {
		if _, err := uc.jwt.ParseAccessToken(accessToken); err == nil {
			_ = uc.blacklist.Add(accessToken)
		}
	}

	if refreshToken != "" {
		if _, err := uc.jwt.ParseRefreshToken(refreshToken); err == nil {
			_ = uc.blacklist.Add(refreshToken)
		}
	}

	uc.log.WithContext(ctx).Info("登出成功")
	return nil
}

// =============================================================================
// 事件定义
// =============================================================================

const TopicUserRegistered = "user.registered"

type UserRegisteredEvent struct {
	UserID   uint64 `json:"user_id"`
	Username string `json:"username"`
	Email    string `json:"email"`
}
