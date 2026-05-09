package service

import (
	"context"

	authv1 "ley/api/auth/v1"
	commonv1 "ley/api/common/v1"
	"ley/app/auth/internal/biz"
	"ley/pkg/jwt"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/transport"
)

type AuthService struct {
	authv1.UnimplementedAuthServiceServer
	authUC *biz.AuthUseCase
	userUC *biz.UserUseCase
	log    *log.Helper
}

func NewAuthService(authUC *biz.AuthUseCase, userUC *biz.UserUseCase, logger log.Logger) *AuthService {
	return &AuthService{authUC: authUC, userUC: userUC, log: log.NewHelper(logger)}
}

func (s *AuthService) Register(ctx context.Context, req *authv1.RegisterRequest) (*authv1.RegisterReply, error) {
	pair, user, err := s.authUC.Register(ctx, req.Username, req.Email, req.Password)
	if err != nil {
		return nil, err
	}
	return &authv1.RegisterReply{
		User:      toUserInfo(user),
		TokenPair: toTokenPair(pair),
	}, nil
}

func (s *AuthService) Login(ctx context.Context, req *authv1.LoginRequest) (*authv1.LoginReply, error) {
	pair, user, err := s.authUC.Login(ctx, req.Account, req.Password)
	if err != nil {
		return nil, err
	}
	return &authv1.LoginReply{
		User:      toUserInfo(user),
		TokenPair: toTokenPair(pair),
	}, nil
}

func (s *AuthService) RefreshToken(ctx context.Context, req *authv1.RefreshTokenRequest) (*authv1.RefreshTokenReply, error) {
	pair, user, err := s.authUC.RefreshToken(ctx, req.RefreshToken)
	if err != nil {
		return nil, err
	}
	return &authv1.RefreshTokenReply{
		User:      toUserInfo(user),
		TokenPair: toTokenPair(pair),
	}, nil
}

func (s *AuthService) Logout(ctx context.Context, req *authv1.LogoutRequest) (*authv1.LogoutReply, error) {
	accessToken := extractToken(ctx)
	if err := s.authUC.Logout(ctx, accessToken, req.RefreshToken); err != nil {
		return nil, err
	}
	return &authv1.LogoutReply{}, nil
}

func (s *AuthService) GetProfile(ctx context.Context, _ *authv1.GetProfileRequest) (*authv1.GetProfileReply, error) {
	user, err := s.userUC.GetProfile(ctx, getUserID(ctx))
	if err != nil {
		return nil, err
	}
	return &authv1.GetProfileReply{User: toUserInfo(user)}, nil
}

func (s *AuthService) UpdateProfile(ctx context.Context, req *authv1.UpdateProfileRequest) (*authv1.UpdateProfileReply, error) {
	user, err := s.userUC.UpdateProfile(ctx, getUserID(ctx), req.Avatar, req.Bio)
	if err != nil {
		return nil, err
	}
	return &authv1.UpdateProfileReply{User: toUserInfo(user)}, nil
}

// =============================================================================
// 类型转换
// =============================================================================

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
		CreatedAt: u.CreatedAt.Format("2006-01-02T15:04:05Z"),
		UpdatedAt: u.UpdatedAt.Format("2006-01-02T15:04:05Z"),
	}
}

func toTokenPair(pair *jwt.TokenPair) *commonv1.TokenPair {
	return &commonv1.TokenPair{
		AccessToken:  pair.AccessToken,
		RefreshToken: pair.RefreshToken,
		ExpiresIn:    900,
	}
}

// =============================================================================
// Transport 工具
// =============================================================================

func getUserID(ctx context.Context) uint {
	if v, ok := ctx.Value("user_id").(uint64); ok {
		return uint(v)
	}
	return 0
}

func extractToken(ctx context.Context) string {
	if header, ok := transport.FromServerContext(ctx); ok {
		auth := header.RequestHeader().Get("Authorization")
		const prefix = "Bearer "
		if len(auth) > len(prefix) && auth[:len(prefix)] == prefix {
			return auth[len(prefix):]
		}
		return auth
	}
	return ""
}
