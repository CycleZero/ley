package proxy

// ===================== 认证域代理 handler（auth.go） =====================
//
// AuthHandler 聚合 auth 服务的 6 个转发 handler：注册/登录/刷新令牌/登出/
// 资料查询/资料更新。每个 handler 都是 callProto 骨架上的薄封装——分配请求
// 消息、闭包内做具体 client 方法与类型断言；请求体绑定、元数据注入、信封
// 序列化与错误映射全部由 callProto 统一完成（设计约定见 handle.go 顶部）。
//
// 本文件不注册任何路由：路径与中间件（哪些路由需 JWT 认证）由 router 域
// 统一挂载，注册形态为 NewAuthHandler(hub) 后按方法取地址，如
//
//	auth := proxy.NewAuthHandler(serviceHub)
//	group.POST("/api/v1/auth/register", auth.Register)
//
// swag 注释中的 @Router 与 @Param 与 proto google.api.http 注解的完整路径一致，
// swag 生成时据此产出 OpenAPI 文档（auth 服务 proto 的 openapi 文档为对照）。

import (
	"context"

	authv1 "github.com/CycleZero/ley/api/auth/v1"
	"github.com/gin-gonic/gin"
	"google.golang.org/protobuf/proto"
)

// AuthHandler 认证域代理 handler 集合：hub 为下游 gRPC 客户端聚合
// （ServiceHub.Auth），handler 只做转发，不承载领域逻辑。
type AuthHandler struct {
	hub *ServiceHub
}

// NewAuthHandler 以 ServiceHub 构造认证域 handler 集合。
func NewAuthHandler(hub *ServiceHub) *AuthHandler {
	return &AuthHandler{hub: hub}
}

// Register 用户注册：POST /api/v1/auth/register（匿名）。
//
// 请求体 JSON（snake_case）→ RegisterRequest；成功后返回 user + token_pair
// （注册即登录，见 proto openapi 描述）。
// @Summary 用户注册
// @Tags 认证
// @Accept json
// @Produce json
// @Param req body authv1.RegisterRequest true "注册请求"
// @Success 200 {object} common.Response{data=authv1.RegisterReply} "成功"
// @Failure 400 {object} common.Response "参数错误"
// @Router /api/v1/auth/register [post]
func (h *AuthHandler) Register(c *gin.Context) {
	callProto(c, &authv1.RegisterRequest{}, func(ctx context.Context, req proto.Message) (proto.Message, error) {
		return h.hub.Auth.Register(ctx, req.(*authv1.RegisterRequest))
	})
}

// Login 用户登录：POST /api/v1/auth/login（匿名）。
//
// 通过用户名/邮箱（account）+ 密码登录，返回 token_pair 与用户信息。
// @Summary 用户登录
// @Tags 认证
// @Accept json
// @Produce json
// @Param req body authv1.LoginRequest true "登录请求"
// @Success 200 {object} common.Response{data=authv1.LoginReply} "成功"
// @Failure 400 {object} common.Response "参数错误"
// @Router /api/v1/auth/login [post]
func (h *AuthHandler) Login(c *gin.Context) {
	callProto(c, &authv1.LoginRequest{}, func(ctx context.Context, req proto.Message) (proto.Message, error) {
		return h.hub.Auth.Login(ctx, req.(*authv1.LoginRequest))
	})
}

// RefreshToken 刷新令牌：POST /api/v1/auth/refresh（匿名，凭 refresh_token）。
//
// 旧 RefreshToken 会被下游加入黑名单，换取新的 token_pair。
// @Summary 刷新令牌
// @Tags 认证
// @Accept json
// @Produce json
// @Param req body authv1.RefreshTokenRequest true "刷新令牌请求"
// @Success 200 {object} common.Response{data=authv1.RefreshTokenReply} "成功"
// @Failure 400 {object} common.Response "参数错误"
// @Router /api/v1/auth/refresh [post]
func (h *AuthHandler) RefreshToken(c *gin.Context) {
	callProto(c, &authv1.RefreshTokenRequest{}, func(ctx context.Context, req proto.Message) (proto.Message, error) {
		return h.hub.Auth.RefreshToken(ctx, req.(*authv1.RefreshTokenRequest))
	})
}

// Logout 登出：POST /api/v1/auth/logout（匿名，凭 refresh_token）。
//
// 下游将当前 AccessToken 与 RefreshToken 加入黑名单，令牌不可再使用。
// @Summary 登出
// @Tags 认证
// @Accept json
// @Produce json
// @Param req body authv1.LogoutRequest true "登出请求"
// @Success 200 {object} common.Response{data=authv1.LogoutReply} "成功"
// @Failure 400 {object} common.Response "参数错误"
// @Router /api/v1/auth/logout [post]
func (h *AuthHandler) Logout(c *gin.Context) {
	callProto(c, &authv1.LogoutRequest{}, func(ctx context.Context, req proto.Message) (proto.Message, error) {
		return h.hub.Auth.Logout(ctx, req.(*authv1.LogoutRequest))
	})
}

// GetProfile 获取当前用户资料：GET /api/v1/users/me（需 JWT，无请求体）。
//
// 用户身份不随 body 传递：T4 认证中间件解析 JWT 后经 pkg/meta 注入 gRPC
// 元数据（x-md-global-auth-*），下游据此识别当前用户——GET 无 body，
// bindRequestBody 直接跳过绑定。
// @Summary 获取当前用户资料
// @Tags 用户
// @Produce json
// @Success 200 {object} common.Response{data=authv1.GetProfileReply} "成功"
// @Failure 401 {object} common.Response "未认证或令牌失效"
// @Router /api/v1/users/me [get]
func (h *AuthHandler) GetProfile(c *gin.Context) {
	callProto(c, &authv1.GetProfileRequest{}, func(ctx context.Context, req proto.Message) (proto.Message, error) {
		return h.hub.Auth.GetProfile(ctx, req.(*authv1.GetProfileRequest))
	})
}

// UpdateProfile 更新用户资料：PUT /api/v1/users/me（需 JWT）。
//
// 只更新头像与简介两个字段（proto 消息即这两个字段）；返回更新后的完整用户信息。
// @Summary 更新用户资料
// @Tags 用户
// @Accept json
// @Produce json
// @Param req body authv1.UpdateProfileRequest true "更新资料请求"
// @Success 200 {object} common.Response{data=authv1.UpdateProfileReply} "成功"
// @Failure 400 {object} common.Response "参数错误"
// @Failure 401 {object} common.Response "未认证或令牌失效"
// @Router /api/v1/users/me [put]
func (h *AuthHandler) UpdateProfile(c *gin.Context) {
	callProto(c, &authv1.UpdateProfileRequest{}, func(ctx context.Context, req proto.Message) (proto.Message, error) {
		return h.hub.Auth.UpdateProfile(ctx, req.(*authv1.UpdateProfileRequest))
	})
}
