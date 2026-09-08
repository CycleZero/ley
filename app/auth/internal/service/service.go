package service

import (
	"context"
	"time"

	authv1 "github.com/CycleZero/ley/api/auth/v1"
	commonv1 "github.com/CycleZero/ley/api/common/v1"
	"github.com/CycleZero/ley/app/auth/internal/biz"
	"github.com/CycleZero/ley/pkg/jwt"
	"github.com/CycleZero/ley/pkg/meta"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/transport"
)

// AuthService 认证服务传输层：把 gRPC/HTTP 请求转为 biz 用例调用并做 proto ↔ DTO 转换。
//
// 日志纪律：本层仅在请求入口打 Debug（细节默认关闭）；业务结果与失败由 biz 层
// 记录一次（遵循「一次事件一行日志」，避免 log-and-rethrow 造成重复告警）。
type AuthService struct {
	authv1.UnimplementedAuthServiceServer
	authUC *biz.AuthUseCase
	userUC *biz.UserUseCase
	log    *log.Helper
}

// NewAuthService 构造认证服务（依赖由 Wire 注入）。
func NewAuthService(authUC *biz.AuthUseCase, userUC *biz.UserUseCase, logger log.Logger) *AuthService {
	return &AuthService{authUC: authUC, userUC: userUC, log: log.NewHelper(logger)}
}

// Register 用户注册：调用注册用例并返回用户信息 + 令牌对。
func (s *AuthService) Register(ctx context.Context, req *authv1.RegisterRequest) (*authv1.RegisterReply, error) {
	s.log.WithContext(ctx).Debug("收到用户注册请求")
	pair, user, err := s.authUC.Register(ctx, req.Username, req.Email, req.Password)
	if err != nil {
		return nil, err
	}
	return &authv1.RegisterReply{
		User:      toUserInfo(user),
		TokenPair: toTokenPair(pair, s.authUC.AccessTTL()),
	}, nil
}

// Login 用户登录：校验凭证并返回用户信息 + 令牌对。
func (s *AuthService) Login(ctx context.Context, req *authv1.LoginRequest) (*authv1.LoginReply, error) {
	s.log.WithContext(ctx).Debug("收到用户登录请求")
	pair, user, err := s.authUC.Login(ctx, req.Account, req.Password)
	if err != nil {
		return nil, err
	}
	return &authv1.LoginReply{
		User:      toUserInfo(user),
		TokenPair: toTokenPair(pair, s.authUC.AccessTTL()),
	}, nil
}

// RefreshToken 刷新令牌：以 refresh token 换取新的令牌对（旧 refresh 立即失效）。
func (s *AuthService) RefreshToken(ctx context.Context, req *authv1.RefreshTokenRequest) (*authv1.RefreshTokenReply, error) {
	s.log.WithContext(ctx).Debug("收到令牌刷新请求")
	pair, user, err := s.authUC.RefreshToken(ctx, req.RefreshToken)
	if err != nil {
		return nil, err
	}
	return &authv1.RefreshTokenReply{
		User:      toUserInfo(user),
		TokenPair: toTokenPair(pair, s.authUC.AccessTTL()),
	}, nil
}

// Logout 登出：吊销 access token（取自 Authorization 或 pkg/meta 透传）与 refresh token。
func (s *AuthService) Logout(ctx context.Context, req *authv1.LogoutRequest) (*authv1.LogoutReply, error) {
	s.log.WithContext(ctx).Debug("收到登出请求")
	accessToken := extractToken(ctx)
	if err := s.authUC.Logout(ctx, accessToken, req.RefreshToken); err != nil {
		return nil, err
	}
	return &authv1.LogoutReply{}, nil
}

// GetProfile 获取当前登录用户资料（用户身份由 pkg/meta 从认证中间件透传）。
func (s *AuthService) GetProfile(ctx context.Context, _ *authv1.GetProfileRequest) (*authv1.GetProfileReply, error) {
	s.log.WithContext(ctx).Debug("收到获取资料请求")
	user, err := s.userUC.GetProfile(ctx, getUserID(ctx))
	if err != nil {
		return nil, err
	}
	return &authv1.GetProfileReply{User: toUserInfo(user)}, nil
}

// UpdateProfile 更新当前登录用户资料（仅头像与简介，防全字段回写覆盖状态/角色）。
func (s *AuthService) UpdateProfile(ctx context.Context, req *authv1.UpdateProfileRequest) (*authv1.UpdateProfileReply, error) {
	s.log.WithContext(ctx).Debug("收到更新资料请求")
	user, err := s.userUC.UpdateProfile(ctx, getUserID(ctx), req.Avatar, req.Bio)
	if err != nil {
		return nil, err
	}
	return &authv1.UpdateProfileReply{User: toUserInfo(user)}, nil
}

// =============================================================================
// 类型转换
// =============================================================================

// toUserInfo 将 biz 用户对象转为 proto UserInfo（时间统一为 UTC RFC3339）。
func toUserInfo(u *biz.User) *commonv1.UserInfo {
	if u == nil {
		return nil
	}
	return &commonv1.UserInfo{
		Id:        uint64(u.ID),
		Username:  u.Username,
		Email:     u.Email,
		Avatar:    u.Avatar,
		Bio:       u.Bio,
		Role:      string(u.Role),
		CreatedAt: u.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt: u.UpdatedAt.UTC().Format(time.RFC3339),
	}
}

// toTokenPair 将 JWT 令牌对转为 proto TokenPair，并回填 expires_in（秒）。
func toTokenPair(pair *jwt.TokenPair, accessTTL time.Duration) *commonv1.TokenPair {
	return &commonv1.TokenPair{
		AccessToken:  pair.AccessToken,
		RefreshToken: pair.RefreshToken,
		ExpiresIn:    int64(accessTTL.Seconds()),
	}
}

// =============================================================================
// Transport 工具
// =============================================================================

// getUserID 从 pkg/meta 读取认证用户 ID；未认证时返回 0（由 biz 层判定未登录）。
func getUserID(ctx context.Context) uint {
	reqMeta := meta.GetRequestMetaData(ctx)
	if reqMeta != nil && reqMeta.Auth.UserID > 0 {
		return uint(reqMeta.Auth.UserID)
	}
	return 0
}

// extractToken 提取原始 access token：优先 Authorization 头，回退 pkg/meta 透传值
// （entry 经 gRPC 转发时不带原始 Authorization，access token 随 x-md-global-* 透传）。
func extractToken(ctx context.Context) string {
	if header, ok := transport.FromServerContext(ctx); ok {
		auth := header.RequestHeader().Get("Authorization")
		const prefix = "Bearer "
		if len(auth) > len(prefix) && auth[:len(prefix)] == prefix {
			return auth[len(prefix):]
		}
		if auth != "" {
			return auth
		}
	}
	if md := meta.GetRequestMetaData(ctx); md != nil {
		return md.AccessToken
	}
	return ""
}
