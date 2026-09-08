package biz

import (
	"context"
	"fmt"
	"time"

	"github.com/CycleZero/ley/pkg/eventbus"
	"github.com/CycleZero/ley/pkg/jwt"
	"github.com/CycleZero/ley/pkg/metrics"
	"github.com/CycleZero/ley/pkg/security"

	kerrors "github.com/go-kratos/kratos/v2/errors"
	"github.com/go-kratos/kratos/v2/log"
	"go.opentelemetry.io/otel/metric"
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
	metrics   *authMetrics
}

// authMetrics 认证域业务指标：按 result 标签（success/error）区分成败，标签基数有界。
type authMetrics struct {
	register metric.Int64Counter
	login    metric.Int64Counter
	refresh  metric.Int64Counter
	logout   metric.Int64Counter
}

// NewAuthUseCase 构造认证业务用例；业务指标仪器在构造期创建
// （Wire 阶段执行，此时 metrics.New 已完成，仪器绑定到真实 MeterProvider）。
func NewAuthUseCase(repo UserRepo, j jwt.JWT, bl jwt.BlackListCache, eb eventbus.EventBus, logger log.Logger) *AuthUseCase {
	return &AuthUseCase{
		repo:      repo,
		jwt:       j,
		blacklist: bl,
		eb:        eb,
		log:       log.NewHelper(logger),
		metrics: &authMetrics{
			register: metrics.Counter("auth_register_total", "用户注册次数"),
			login:    metrics.Counter("auth_login_total", "用户登录次数"),
			refresh:  metrics.Counter("auth_token_refresh_total", "令牌刷新次数"),
			logout:   metrics.Counter("auth_logout_total", "登出次数"),
		},
	}
}

// AccessTTL 返回 access token 有效期，供 service 层回填响应 expires_in
func (uc *AuthUseCase) AccessTTL() time.Duration {
	return uc.jwt.AccessTTL()
}

// =============================================================================
// Register — 用户注册
// =============================================================================

// Register 用户注册：校验用户名/密码 → 唯一性检查 → 哈希落库 → 签发令牌 → 发布注册事件。
// 返回命名参数以便统一记录业务指标（result 标签区分成败）。
func (uc *AuthUseCase) Register(ctx context.Context, username, email, password string) (pair *jwt.TokenPair, user *User, err error) {
	defer func() { recordResult(ctx, uc.metrics.register, resultOf(err)) }()

	if err = validateUsername(username); err != nil {
		uc.log.WithContext(ctx).Warnf("注册失败：用户名不合法 username=%s", username)
		return nil, nil, err
	}
	if err = validatePassword(password); err != nil {
		uc.log.WithContext(ctx).Warn("注册失败：密码强度不足")
		return nil, nil, err
	}

	if _, err = uc.repo.FindByUsername(ctx, username); err == nil {
		uc.log.WithContext(ctx).Warnf("注册失败：用户名已存在 username=%s", username)
		return nil, nil, ErrUsernameTaken
	}
	if _, err = uc.repo.FindByEmail(ctx, email); err == nil {
		uc.log.WithContext(ctx).Warnf("注册失败：邮箱已注册 email=%s", email)
		return nil, nil, ErrEmailTaken
	}

	hashed, hashErr := security.HashPassword(password)
	if hashErr != nil {
		err = kerrors.InternalServer("HASH_FAILED", "密码处理失败")
		uc.log.WithContext(ctx).Errorf("密码哈希失败: %v", hashErr)
		return nil, nil, err
	}

	user = &User{
		Username: username,
		Email:    email,
		Password: hashed,
		Status:   UserStatusActive,
		Role:     RoleReader,
	}

	if err = uc.repo.Create(ctx, user); err != nil {
		if kerrors.IsConflict(err) || kerrors.Code(err) == 409 {
			uc.log.WithContext(ctx).Warnf("注册失败：唯一约束冲突 username=%s", username)
			return nil, nil, err
		}
		uc.log.WithContext(ctx).Errorf("创建用户失败: %v", err)
		return nil, nil, fmt.Errorf("register: %w", err)
	}

	pair, err = uc.jwt.GenerateTokenPair(jwt.Payload{
		UserId:   uint64(user.ID),
		UserName: user.Username,
		Role:     string(user.Role),
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

// Login 登录：按账号查用户 → 校验状态与密码 → 签发令牌。
// 失败路径统一 Warn（不记录密码/明文账号，避免敏感信息入日志），并计入业务指标。
func (uc *AuthUseCase) Login(ctx context.Context, account, password string) (pair *jwt.TokenPair, user *User, err error) {
	defer func() { recordResult(ctx, uc.metrics.login, resultOf(err)) }()

	user, err = uc.repo.FindByAccount(ctx, account)
	if err != nil {
		uc.log.WithContext(ctx).Warn("登录失败：账号不存在")
		return nil, nil, ErrBadCredentials
	}

	if user.Status != UserStatusActive {
		uc.log.WithContext(ctx).Warnf("登录失败：账号已禁用 id=%d", user.ID)
		return nil, nil, ErrAccountDisabled
	}

	if !security.VerifyPassword(password, user.Password) {
		uc.log.WithContext(ctx).Warnf("登录失败：密码错误 id=%d", user.ID)
		return nil, nil, ErrBadCredentials
	}

	pair, err = uc.jwt.GenerateTokenPair(jwt.Payload{
		UserId:   uint64(user.ID),
		UserName: user.Username,
		Role:     string(user.Role),
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

// RefreshToken 刷新令牌（Token Rotation）：解析并校验 refresh → 黑名单检查 →
// 重新签发令牌对 → 旧 refresh 加入黑名单。失败路径 Warn + 业务指标。
func (uc *AuthUseCase) RefreshToken(ctx context.Context, refreshToken string) (pair *jwt.TokenPair, user *User, err error) {
	defer func() { recordResult(ctx, uc.metrics.refresh, resultOf(err)) }()

	claims, err := uc.jwt.ParseRefreshToken(refreshToken)
	if err != nil {
		uc.log.WithContext(ctx).Warn("刷新失败：refresh token 无效或已过期")
		return nil, nil, kerrors.Unauthorized("TOKEN_INVALID", "令牌无效或已过期")
	}

	if uc.blacklist.IsEnabled() && uc.blacklist.IsTokenBlackListed(refreshToken) {
		uc.log.WithContext(ctx).Warnf("刷新失败：refresh token 已登出 id=%d", claims.UserId)
		return nil, nil, kerrors.Unauthorized("TOKEN_BLACKLISTED", "令牌已登出")
	}

	user, err = uc.repo.FindByID(ctx, uint(claims.UserId))
	if err != nil {
		uc.log.WithContext(ctx).Warnf("刷新失败：用户不存在 id=%d", claims.UserId)
		return nil, nil, ErrUserNotFound
	}

	if user.Status != UserStatusActive {
		uc.log.WithContext(ctx).Warnf("刷新失败：账号已禁用 id=%d", user.ID)
		return nil, nil, ErrAccountDisabled
	}

	pair, err = uc.jwt.GenerateTokenPair(jwt.Payload{
		UserId:   uint64(user.ID),
		UserName: user.Username,
		Role:     string(user.Role),
	})
	if err != nil {
		uc.log.WithContext(ctx).Errorf("刷新令牌生成失败 id=%d: %v", user.ID, err)
		return nil, nil, fmt.Errorf("refresh token: %w", err)
	}

	if uc.blacklist.IsEnabled() {
		if blErr := uc.blacklist.AddWithTTL(refreshToken, tokenRemainingTTL(claims)); blErr != nil {
			uc.log.WithContext(ctx).Warnf("旧令牌加入黑名单失败: %v", blErr)
		}
	}

	uc.log.WithContext(ctx).Infof("令牌刷新成功 id=%d", user.ID)
	return pair, user, nil
}

// tokenRemainingTTL 返回 token 距过期的剩余时长，供黑名单按剩余寿命设置 TTL，
// 避免黑名单键永久驻留 Redis。已过期时兜底 1 秒（解析成功路径理论上不会出现）。
func tokenRemainingTTL(claims *jwt.Claims) time.Duration {
	ttl := time.Until(claims.ExpiresAt.Time)
	if ttl <= 0 {
		return time.Second
	}
	return ttl
}

// =============================================================================
// Logout — 登出：令牌加入黑名单
// =============================================================================

// Logout 登出：把 access/refresh token 加入黑名单（按剩余寿命设 TTL）。
// 黑名单未启用时直接返回；加入失败仅 Warn（不阻断登出，最坏情况令牌自然过期）。
func (uc *AuthUseCase) Logout(ctx context.Context, accessToken, refreshToken string) (err error) {
	defer func() { recordResult(ctx, uc.metrics.logout, resultOf(err)) }()

	if !uc.blacklist.IsEnabled() {
		return nil
	}

	if accessToken != "" {
		if claims, perr := uc.jwt.ParseAccessToken(accessToken); perr == nil {
			if blErr := uc.blacklist.AddWithTTL(accessToken, tokenRemainingTTL(claims)); blErr != nil {
				uc.log.WithContext(ctx).Warnf("access token 加入黑名单失败: %v", blErr)
			}
		}
	}

	if refreshToken != "" {
		if claims, perr := uc.jwt.ParseRefreshToken(refreshToken); perr == nil {
			if blErr := uc.blacklist.AddWithTTL(refreshToken, tokenRemainingTTL(claims)); blErr != nil {
				uc.log.WithContext(ctx).Warnf("refresh token 加入黑名单失败: %v", blErr)
			}
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
