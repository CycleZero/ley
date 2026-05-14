/**
 * 认证相关类型定义（auth.ts）
 *
 * 与后端 proto 文件 api/auth/v1/auth.proto 严格对齐，
 * 定义了认证模块所有请求（Request）和响应（Reply）消息的类型。
 *
 * JSON 字段名使用 snake_case，与 Go 语言的 json tag 保持一致。
 *
 * 消息分类：
 * - Request 类型：前端发送给后端的请求体结构
 * - Reply 类型：后端返回给前端的响应体结构
 */

// =============================================================================
// 与 api/auth/v1/auth.proto 严格对齐
// JSON key: snake_case（与 Go json tag 一致）
// =============================================================================

import type { UserInfo, TokenPair } from './common'

// ---- Request 请求消息 ----

/**
 * RegisterRequest: 用户注册请求
 *
 * 对应 proto 消息：api.auth.v1.RegisterRequest
 *
 * 字段：
 * - username: 用户名（需唯一，用于登录和展示）
 * - email:    邮箱地址（需唯一，用于找回密码等）
 * - password: 明文密码（长度/复杂度由前端和后端双重校验）
 */
export interface RegisterRequest {
  username: string
  email: string
  password: string
}

/**
 * LoginRequest: 用户登录请求
 *
 * 对应 proto 消息：api.auth.v1.LoginRequest
 *
 * 字段：
 * - account:  用户名或邮箱（后端自动判断是哪种类型）
 * - password: 密码
 */
export interface LoginRequest {
  /** 用户名或邮箱 */
  account: string
  password: string
}

/**
 * RefreshTokenRequest: Token 刷新请求
 *
 * 对应 proto 消息：api.auth.v1.RefreshTokenRequest
 *
 * 字段：
 * - refresh_token: 当前持有的刷新令牌（客户端从 localStorage 读取）
 */
export interface RefreshTokenRequest {
  refresh_token: string
}

/**
 * LogoutRequest: 登出请求
 *
 * 对应 proto 消息：api.auth.v1.LogoutRequest
 *
 * 字段：
 * - refresh_token: 需要失效的刷新令牌
 *   后端收到后将该 token 标记为无效，防止被继续使用
 */
export interface LogoutRequest {
  refresh_token: string
}

/**
 * UpdateProfileRequest: 更新个人资料请求
 *
 * 对应 proto 消息：api.auth.v1.UpdateProfileRequest
 *
 * 字段：
 * - avatar: 新头像 URL（覆盖旧值）
 * - bio:    新个人简介（覆盖旧值）
 */
export interface UpdateProfileRequest {
  avatar: string
  bio: string
}

// ---- Reply 响应消息 ----

/**
 * RegisterReply: 注册响应
 *
 * 对应 proto 消息：api.auth.v1.RegisterReply
 *
 * 注册成功后直接返回 TokenPair 和 UserInfo，
 * 前端无需再调用登录接口即可完成认证。
 */
export interface RegisterReply {
  user: UserInfo
  token_pair: TokenPair
}

/**
 * LoginReply: 登录响应
 *
 * 对应 proto 消息：api.auth.v1.LoginReply
 *
 * 结构同 RegisterReply，包含用户信息和令牌对。
 */
export interface LoginReply {
  user: UserInfo
  token_pair: TokenPair
}

/**
 * RefreshTokenReply: Token 刷新响应
 *
 * 对应 proto 消息：api.auth.v1.RefreshTokenReply
 *
 * 返回新的 TokenPair 和最新的用户信息。
 * 旧 refreshToken 在刷新成功后立即失效（Token 轮转机制）。
 */
export interface RefreshTokenReply {
  user: UserInfo
  token_pair: TokenPair
}

/**
 * LogoutReply: 登出响应
 *
 * 对应 proto 消息：api.auth.v1.LogoutReply
 *
 * 空响应体 —— 登出操作无需返回数据。
 * 使用 type 定义而非 interface 以避免 TypeScript 的空对象警告。
 */
// eslint-disable-next-line @typescript-eslint/no-empty-object-type
export type LogoutReply = {}

/**
 * GetProfileReply: 获取个人资料响应
 *
 * 对应 proto 消息：api.auth.v1.GetProfileReply
 *
 * 与 LoginReply 不同，仅返回用户信息不返回 TokenPair，
 * 因为获取资料不需重新颁发令牌。
 */
export interface GetProfileReply {
  user: UserInfo
}

/**
 * UpdateProfileReply: 更新个人资料响应
 *
 * 对应 proto 消息：api.auth.v1.UpdateProfileReply
 *
 * 返回更新后的完整用户信息，前端用于更新本地状态。
 */
export interface UpdateProfileReply {
  user: UserInfo
}
