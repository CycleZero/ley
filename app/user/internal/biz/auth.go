package biz

import (
	"context"
	"fmt"
	"time"

	"ley/app/user/internal/model"
	"ley/pkg/jwt"

	"github.com/go-kratos/kratos/v2/log"
)

// JWTProvider 令牌服务接口（biz 层只依赖接口，不依赖具体实现）
// 提供令牌生成和解析能力，由 pkg/jwt 包实现
type JWTProvider interface {
	GenerateTokenPair(payload jwt.Payload) (*jwt.TokenPair, error) // 生成 AccessToken + RefreshToken 对
	ParseAccessToken(tokenString string) (*jwt.Claims, error)      // 解析并验证 AccessToken
	ParseRefreshToken(tokenString string) (*jwt.Claims, error)     // 解析并验证 RefreshToken
	GenerateToken(payload jwt.Payload) (string, error)             // 生成单个令牌
}

// AuthUseCase 认证业务用例
// 负责登录认证、令牌刷新、登出等认证流程
// 持有 UserRepo（查询用户）、JWTProvider（令牌管理）、BlackListCache（黑名单）
type AuthUseCase struct {
	userRepo  UserRepo           // 用户数据访问接口（用于查询用户信息）
	jwt       JWTProvider        // JWT 令牌生成与解析接口
	blacklist jwt.BlackListCache // 令牌黑名单缓存（防重放攻击）
	jwtExpire time.Duration      // JWT 令牌过期时间
	logger    log.Logger         // Kratos 日志接口
}

// NewAuthUseCase 创建 AuthUseCase
// userRepo: UserRepo 接口实现
// jwtProv: JWTProvider 接口实现
// bl: BlackListCache 接口实现（黑名单缓存）
// jwtExpire: 令牌过期时长配置
// logger: Kratos 日志接口
func NewAuthUseCase(
	userRepo UserRepo,
	jwtProv JWTProvider,
	bl jwt.BlackListCache,
	jwtExpire time.Duration,
	logger log.Logger,
) *AuthUseCase {
	return &AuthUseCase{
		userRepo:  userRepo,
		jwt:       jwtProv,
		blacklist: bl,
		jwtExpire: jwtExpire,
		logger:    logger,
	}
}

// log 创建携带链路追踪上下文信息的日志 Helper
func (uc *AuthUseCase) log(ctx context.Context) *log.Helper {
	return log.NewHelper(log.WithContext(ctx, uc.logger))
}

// Login 登录并返回 TokenPair
// 流程：验证用户凭证 → 生成 AccessToken + RefreshToken
// 返回令牌对供 service 层返回给客户端
func (uc *AuthUseCase) Login(ctx context.Context, account, password string) (*jwt.TokenPair, error) {
	uc.log(ctx).Debugw("认证登录开始", "account", account)

	// 复用 UserUseCase.Login 验证用户凭证（account + password）
	// 传入 AuthUseCase 的 logger，确保日志链路完整
	loginUC := &UserUseCase{repo: uc.userRepo, logger: uc.logger}
	user, err := loginUC.Login(ctx, account, password)
	if err != nil {
		uc.log(ctx).Warnw("认证登录：凭证验证失败", "account", account, "error", err)
		return nil, err
	}

	uc.log(ctx).Debugw("认证登录：凭证验证通过，准备生成令牌", "id", user.ID, "username", user.Username)

	// 生成令牌对（AccessToken + RefreshToken）
	pair, err := uc.jwt.GenerateTokenPair(jwt.Payload{
		UserId:   uint64(user.ID),
		UserName: user.Username,
	})
	if err != nil {
		uc.log(ctx).Errorw("认证登录：生成令牌失败", "id", user.ID, "error", err)
		return nil, fmt.Errorf("login: generate token: %w", err)
	}

	uc.log(ctx).Infow("令牌对生成成功", "id", user.ID, "username", user.Username)
	return pair, nil
}

// RefreshToken 刷新令牌
// 流程：解析并验证 RefreshToken → 检查黑名单 → 验证用户状态 → 生成新令牌对 → 将旧 RefreshToken 加入黑名单
// 防止 RefreshToken 被重复使用（Token Rotation）
func (uc *AuthUseCase) RefreshToken(ctx context.Context, refreshToken string) (*jwt.TokenPair, error) {
	uc.log(ctx).Debugw("刷新令牌开始", "token_prefix", refreshToken[:min(len(refreshToken), 10)])

	// 第一步：解析并验证 RefreshToken
	claims, err := uc.jwt.ParseRefreshToken(refreshToken)
	if err != nil {
		uc.log(ctx).Warnw("刷新令牌：解析RefreshToken失败", "error", err)
		return nil, ErrBadCredentials
	}

	uc.log(ctx).Debugw("刷新令牌：RefreshToken解析成功", "user_id", claims.UserId)

	// 第二步：检查 RefreshToken 是否在黑名单中（防重放）
	if uc.blacklist.IsEnabled() && uc.blacklist.IsTokenBlackListed(refreshToken) {
		uc.log(ctx).Warnw("刷新令牌：令牌已加入黑名单", "user_id", claims.UserId)
		return nil, ErrBadCredentials
	}

	// 第三步：查询用户并验证状态
	user, err := uc.userRepo.FindByID(ctx, uint(claims.UserId))
	if err != nil {
		uc.log(ctx).Warnw("刷新令牌：查询用户失败", "user_id", claims.UserId)
		return nil, ErrBadCredentials
	}
	if user.Status != model.UserStatusActive {
		uc.log(ctx).Warnw("刷新令牌：用户账号已禁用", "id", user.ID, "username", user.Username)
		return nil, ErrAccountDisabled
	}

	// 第四步：生成新的令牌对
	pair, err := uc.jwt.GenerateTokenPair(jwt.Payload{
		UserId:   uint64(user.ID),
		UserName: user.Username,
	})
	if err != nil {
		uc.log(ctx).Errorw("刷新令牌：生成新令牌失败", "id", user.ID, "error", err)
		return nil, fmt.Errorf("refresh token: %w", err)
	}

	// 第五步：将当前 RefreshToken 加入黑名单（Token Rotation 安全策略）
	if uc.blacklist.IsEnabled() {
		if addErr := uc.blacklist.Add(refreshToken); addErr != nil {
			uc.log(ctx).Warnw("刷新令牌：旧令牌加入黑名单失败", "error", addErr)
		} else {
			uc.log(ctx).Debugw("刷新令牌：旧令牌已加入黑名单", "user_id", user.ID)
		}
	}

	uc.log(ctx).Infow("令牌刷新成功", "id", user.ID, "username", user.Username)
	return pair, nil
}

// Logout 登出
// 将当前 AccessToken 和 RefreshToken 加入黑名单
// 仅在黑名单启用时生效，未启用时直接返回成功
func (uc *AuthUseCase) Logout(ctx context.Context, accessToken, refreshToken string) error {
	uc.log(ctx).Debugw("用户登出开始")

	// 黑名单未启用则跳过
	if !uc.blacklist.IsEnabled() {
		uc.log(ctx).Debugw("用户登出：黑名单未启用，跳过")
		return nil
	}

	// 将 AccessToken 加入黑名单
	if accessToken != "" {
		if _, err := uc.jwt.ParseAccessToken(accessToken); err != nil {
			uc.log(ctx).Warnw("用户登出：AccessToken无效，跳过加入黑名单", "error", err)
		} else {
			_ = uc.blacklist.Add(accessToken)
			uc.log(ctx).Debugw("用户登出：AccessToken已加入黑名单")
		}
	}

	// 将 RefreshToken 加入黑名单
	if refreshToken != "" {
		if _, err := uc.jwt.ParseRefreshToken(refreshToken); err != nil {
			uc.log(ctx).Warnw("用户登出：RefreshToken无效，跳过加入黑名单", "error", err)
		} else {
			_ = uc.blacklist.Add(refreshToken)
			uc.log(ctx).Debugw("用户登出：RefreshToken已加入黑名单")
		}
	}

	uc.log(ctx).Infow("用户已登出")
	return nil
}

// ParseAccessToken 解析 AccessToken（供 service 层获取登录用户信息）
// 直接委托给 JWTProvider，不添加额外业务逻辑
func (uc *AuthUseCase) ParseAccessToken(tokenString string) (*jwt.Claims, error) {
	return uc.jwt.ParseAccessToken(tokenString)
}
