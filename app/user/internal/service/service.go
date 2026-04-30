// Package service — 用户服务 HTTP/gRPC Handler 层
//
// 负责：
//   1. 实现 api/user/v1 生成的 UserServiceServer 接口
//   2. Proto 消息 ↔ 业务模型（model.User）转换
//   3. 调用 biz 层 UseCase，映射错误码到 gRPC status
//   4. 不包含业务逻辑（全部委托给 biz 层）
package service

import (
	"context"
	"errors"

	userv1 "ley/api/user/v1"
	"ley/app/user/internal/biz"
	"ley/app/user/internal/model"

	"github.com/go-kratos/kratos/v2/log"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// =============================================================================
// UserService — 实现 api/user/v1.UserServiceServer
// =============================================================================

// UserService 用户服务 Handler
// 嵌入 UnimplementedUserServiceServer 保证前向兼容：
// 若 proto 新增 RPC 方法但未实现，编译不会报错，运行时会返回 Unimplemented 错误
type UserService struct {
	userv1.UnimplementedUserServiceServer // 嵌入未实现接口，保证前向兼容

	uc     *biz.UserUseCase // 用户资料业务逻辑（注册、登录、资料编辑）
	authUC *biz.AuthUseCase  // 认证 + Token 管理（登录出/令牌刷新/登出）
	logger log.Logger        // Kratos 日志接口
}

// NewUserService 创建 UserService
// uc: UserUseCase（由 Wire 自动注入）
// authUC: AuthUseCase（由 Wire 自动注入）
// logger: Kratos 日志接口
func NewUserService(uc *biz.UserUseCase, authUC *biz.AuthUseCase, logger log.Logger) *UserService {
	return &UserService{uc: uc, authUC: authUC, logger: logger}
}

// log 创建携带链路追踪上下文信息的日志 Helper
func (s *UserService) log(ctx context.Context) *log.Helper {
	return log.NewHelper(log.WithContext(ctx, s.logger))
}

// =============================================================================
// Auth Handlers — 认证相关 RPC 方法
// =============================================================================

// Register 用户注册
// 将 Proto 请求参数传递给 biz 层 UserUseCase.Register
// 注册成功后返回用户信息
func (s *UserService) Register(ctx context.Context, req *userv1.RegisterRequest) (*userv1.RegisterReply, error) {
	s.log(ctx).Debugw("服务层：收到注册请求",
		"username", req.Username,
		"email", req.Email,
	)

	// 委托 biz 层执行注册逻辑（校验、唯一性检查、密码哈希、入库）
	user, err := s.uc.Register(ctx, req.Username, req.Email, req.Password, req.Nickname)
	if err != nil {
		s.log(ctx).Warnw("服务层：注册失败", "username", req.Username, "error", err)
		return nil, s.mapError(err)
	}

	s.log(ctx).Infow("服务层：用户注册完成",
		"id", user.ID, "username", user.Username,
	)

	// 将 model.User 转换为 Proto 消息返回
	return &userv1.RegisterReply{User: toUserInfo(user)}, nil
}

// Login 用户登录 → TokenPair
// 委托 AuthUseCase.Login 完成凭证验证和令牌生成
// 登录成功后从生成的 AccessToken 解析用户信息返回
func (s *UserService) Login(ctx context.Context, req *userv1.LoginRequest) (*userv1.LoginReply, error) {
	s.log(ctx).Debugw("服务层：收到登录请求", "account", req.Account)

	// 委托 AuthUseCase 验证凭证并生成令牌对
	pair, err := s.authUC.Login(ctx, req.Account, req.Password)
	if err != nil {
		s.log(ctx).Warnw("服务层：登录失败", "account", req.Account, "error", err)
		return nil, s.mapError(err)
	}

	// 登录成功后获取用户信息（Login 内部已做过凭证校验，此处直接查）
	// 使用 AccessToken 解析获取 user_id
	claims, parseErr := s.authUC.ParseAccessToken(pair.AccessToken)
	if parseErr != nil {
		s.log(ctx).Errorw("服务层：解析生成的令牌失败", "error", parseErr)
		return nil, status.Error(codes.Internal, "internal error")
	}

	s.log(ctx).Debugw("服务层：令牌解析成功", "user_id", claims.UserId)

	// 尝试从 context 获取完整用户信息，失败则使用 claims 中的最小信息
	user, findErr := s.uc.GetProfile(ctx) // GetProfile 从 ctx 提取 user_id
	if findErr != nil {
		s.log(ctx).Warnw("服务层：登录后获取用户资料失败，使用最小信息", "user_id", claims.UserId)
		user = &model.User{Username: claims.UserName}
	}

	return &userv1.LoginReply{
		User: toUserInfo(user),
		TokenPair: &userv1.TokenPair{
			AccessToken:  pair.AccessToken,
			RefreshToken: pair.RefreshToken,
			ExpiresIn:    900, // AccessToken 过期时间 15 分钟
		},
	}, nil
}

// RefreshToken 令牌刷新
// 委托 AuthUseCase.RefreshToken 校验 RefreshToken 并返回新令牌对
func (s *UserService) RefreshToken(ctx context.Context, req *userv1.RefreshTokenRequest) (*userv1.LoginReply, error) {
	s.log(ctx).Debugw("服务层：收到刷新令牌请求")

	// 委托 AuthUseCase 执行令牌刷新（解析旧令牌 → 查用户状态 → 生成新对 → 旧令牌加黑名单）
	pair, err := s.authUC.RefreshToken(ctx, req.RefreshToken)
	if err != nil {
		s.log(ctx).Warnw("服务层：刷新令牌失败", "error", err)
		return nil, s.mapError(err)
	}

	// 解析新 AccessToken 获取用户信息
	claims, parseErr := s.authUC.ParseAccessToken(pair.AccessToken)
	if parseErr != nil {
		s.log(ctx).Errorw("服务层：刷新后解析新令牌失败", "error", parseErr)
		return nil, status.Error(codes.Internal, "internal error")
	}

	s.log(ctx).Debugw("服务层：新令牌解析成功", "user_id", claims.UserId)

	// 获取用户信息，失败则用最小信息
	user, _ := s.uc.GetProfile(ctx)
	if user == nil {
		s.log(ctx).Debugw("服务层：刷新后获取用户资料失败，使用最小信息", "user_id", claims.UserId)
		user = &model.User{Username: claims.UserName}
	}

	return &userv1.LoginReply{
		User: toUserInfo(user),
		TokenPair: &userv1.TokenPair{
			AccessToken:  pair.AccessToken,
			RefreshToken: pair.RefreshToken,
			ExpiresIn:    900,
		},
	}, nil
}

// Logout 登出
// 从上下文提取 AccessToken，与 RefreshToken 一起加入黑名单
func (s *UserService) Logout(ctx context.Context, req *userv1.LogoutRequest) (*userv1.LogoutReply, error) {
	s.log(ctx).Debugw("服务层：收到登出请求")

	// 从 context 中提取 Bearer token（由上游 middleware 注入）
	accessToken := extractToken(ctx)
	if err := s.authUC.Logout(ctx, accessToken, req.RefreshToken); err != nil {
		s.log(ctx).Warnw("服务层：登出失败", "error", err)
		return nil, s.mapError(err)
	}

	s.log(ctx).Debugw("服务层：登出完成")
	return &userv1.LogoutReply{}, nil
}

// =============================================================================
// Profile Handlers — 用户资料相关 RPC 方法
// =============================================================================

// GetProfile 获取当前用户资料
// 从 context 提取 userID（由上游认证中间件注入）查询用户信息
func (s *UserService) GetProfile(ctx context.Context, _ *userv1.GetProfileRequest) (*userv1.GetProfileReply, error) {
	s.log(ctx).Debugw("服务层：收到获取用户资料请求")

	user, err := s.uc.GetProfile(ctx)
	if err != nil {
		s.log(ctx).Warnw("服务层：获取用户资料失败", "error", err)
		return nil, s.mapError(err)
	}

	s.log(ctx).Debugw("服务层：获取用户资料成功", "id", user.ID, "username", user.Username)
	return &userv1.GetProfileReply{User: toUserInfo(user)}, nil
}

// UpdateProfile 更新当前用户资料
// 只允许编辑昵称、头像、个人简介
func (s *UserService) UpdateProfile(ctx context.Context, req *userv1.UpdateProfileRequest) (*userv1.GetProfileReply, error) {
	s.log(ctx).Debugw("服务层：收到更新用户资料请求",
		"nickname", req.Nickname,
	)

	user, err := s.uc.UpdateProfile(ctx, req.Nickname, req.Avatar, req.Bio)
	if err != nil {
		s.log(ctx).Warnw("服务层：更新用户资料失败", "error", err)
		return nil, s.mapError(err)
	}

	s.log(ctx).Debugw("服务层：更新用户资料成功", "id", user.ID, "username", user.Username)
	return &userv1.GetProfileReply{User: toUserInfo(user)}, nil
}

// =============================================================================
// Internal RPC Handlers — 内部 gRPC 方法（供其他微服务调用）
// =============================================================================

// GetUser 按 UUID 获取用户（内部 gRPC）
func (s *UserService) GetUser(ctx context.Context, req *userv1.GetUserRequest) (*userv1.GetUserReply, error) {
	s.log(ctx).Debugw("服务层：内部RPC 收到获取用户请求", "user_id", req.UserId)

	user, err := s.uc.FindByUUID(ctx, req.UserId)
	if err != nil {
		s.log(ctx).Warnw("服务层：内部RPC 获取用户失败", "user_id", req.UserId, "error", err)
		return nil, s.mapError(err)
	}

	s.log(ctx).Debugw("服务层：内部RPC 获取用户成功", "uuid", req.UserId, "id", user.ID)
	return &userv1.GetUserReply{User: toUserInfo(user)}, nil
}

// BatchGetUsers 批量获取用户（内部 gRPC）
// 遍历 userIds 逐一查询，不存在的用户静默跳过（不返回错误）
func (s *UserService) BatchGetUsers(ctx context.Context, req *userv1.BatchGetUsersRequest) (*userv1.BatchGetUsersReply, error) {
	s.log(ctx).Debugw("服务层：内部RPC 收到批量获取用户请求", "count", len(req.UserIds))

	users := make([]*userv1.UserInfo, 0, len(req.UserIds))
	missCount := 0
	for _, id := range req.UserIds {
		user, err := s.uc.FindByUUID(ctx, id)
		if err == nil {
			users = append(users, toUserInfo(user))
		} else {
			missCount++
			s.log(ctx).Debugw("服务层：批量获取中用户不存在，跳过", "user_id", id)
		}
	}

	s.log(ctx).Debugw("服务层：批量获取用户完成",
		"requested", len(req.UserIds),
		"found", len(users),
		"missed", missCount,
	)
	return &userv1.BatchGetUsersReply{Users: users}, nil
}

// =============================================================================
// Helpers: 类型转换 — 业务模型 ↔ Proto 消息
// =============================================================================

// toUserInfo 将 model.User 转换为 proto UserInfo
// 对外使用 UUID 作为用户 ID，不暴露数据库自增主键
func toUserInfo(user *model.User) *userv1.UserInfo {
	if user == nil {
		return nil
	}
	return &userv1.UserInfo{
		Id:        user.UUID,
		Username:  user.Username,
		Email:     user.Email,
		Nickname:  user.Nickname,
		Avatar:    user.Avatar,
		Bio:       user.Bio,
		Role:      string(user.Role),
		CreatedAt: user.CreatedAt.Format("2006-01-02T15:04:05Z"),
		UpdatedAt: user.UpdatedAt.Format("2006-01-02T15:04:05Z"),
	}
}

// extractToken 从 context 中提取 Bearer token
// 由上游认证 middleware 在请求处理前注入到 context value 中
func extractToken(ctx context.Context) string {
	if token, ok := ctx.Value("access_token").(string); ok {
		return token
	}
	return ""
}

// =============================================================================
// Helpers: 错误映射 — biz 层错误 → gRPC Status Code
// =============================================================================

// mapError 将 biz 层错误映射为 gRPC status
// 对外不暴露内部错误详情，仅返回标准 HTTP/gRPC 状态码
// 映射规则：
//   - ErrUserNotFound            → NotFound (5)
//   - ErrUserDuplicate/Username/Email → AlreadyExists (6)
//   - ErrBadCredentials          → Unauthenticated (16)
//   - ErrAccountDisabled         → PermissionDenied (7)
//   - 输入校验错误                → InvalidArgument (3)
//   - 其他未映射错误             → Internal (13)
func (s *UserService) mapError(err error) error {
	switch {
	case errors.Is(err, biz.ErrUserNotFound):
		return status.Error(codes.NotFound, "user not found")
	case errors.Is(err, biz.ErrUserDuplicate),
		errors.Is(err, biz.ErrUsernameTaken),
		errors.Is(err, biz.ErrEmailTaken):
		return status.Error(codes.AlreadyExists, err.Error())
	case errors.Is(err, biz.ErrBadCredentials):
		return status.Error(codes.Unauthenticated, "invalid credentials")
	case errors.Is(err, biz.ErrAccountDisabled):
		return status.Error(codes.PermissionDenied, "account disabled")
	case errors.Is(err, biz.ErrPasswordTooShort),
		errors.Is(err, biz.ErrPasswordTooLong),
		errors.Is(err, biz.ErrPasswordWeak),
		errors.Is(err, biz.ErrUsernameInvalid),
		errors.Is(err, biz.ErrNicknameTooLong),
		errors.Is(err, biz.ErrBioTooLong):
		return status.Error(codes.InvalidArgument, err.Error())
	default:
		s.logger.Log(log.LevelError, "未映射的错误", "error", err)
		return status.Error(codes.Internal, "internal server error")
	}
}
