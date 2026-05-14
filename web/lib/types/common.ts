/**
 * 通用类型定义（common.ts）
 *
 * 本文件定义了在多个 API 模块之间共享的基础消息类型。
 * 与后端 proto 文件 api/common/v1/common.proto 严格对齐，
 * JSON 字段名使用 snake_case，与 Go 语言的 json tag 保持一致。
 *
 * 包含：
 * - TokenPair: JWT 令牌对（访问令牌 + 刷新令牌）
 * - UserInfo:  用户公开信息
 * - AuthorInfo: 作者简要信息（用于文章列表展示）
 */

// =============================================================================
// 与 api/common/v1/common.proto 严格对齐
// JSON key: snake_case（与 Go json tag 一致）
// =============================================================================

/**
 * TokenPair: JWT 令牌对
 *
 * 对应 proto 消息：api.common.v1.TokenPair
 *
 * 字段说明：
 * - access_token:  访问令牌（JWT），短有效期（如 15 分钟），
 *                  通过 Authorization: Bearer <token> 头携带
 * - refresh_token: 刷新令牌（JWT），长有效期（如 7 天），
 *                  访问令牌过期后用于获取新令牌，无需重新登录
 * - expires_in:    访问令牌的有效时长（单位：秒），int64 类型
 *                  前端可据此计算 token 过期时间，实现提前刷新
 */
export interface TokenPair {
  access_token: string
  refresh_token: string
  /** 访问令牌过期时间（秒），int64 */
  expires_in: number
}

/**
 * UserInfo: 用户公开信息
 *
 * 对应 proto 消息：api.common.v1.UserInfo
 *
 * 字段说明：
 * - id:         用户唯一标识（uint64）
 * - username:   用户名（登录和展示用）
 * - email:      邮箱地址
 * - avatar:     头像 URL（可为空字符串）
 * - bio:        个人简介/签名（可为空字符串）
 * - role:       用户角色（如 "admin"、"user"），用于权限判断
 * - created_at: 账号创建时间（ISO 8601 格式字符串）
 * - updated_at: 信息最后更新时间（ISO 8601 格式字符串）
 *
 * 注意：不包含密码哈希等敏感信息，仅用于展示和权限判断。
 */
export interface UserInfo {
  /** 用户 ID，uint64 */
  id: number
  username: string
  email: string
  avatar: string
  bio: string
  role: string
  /** 创建时间，ISO 8601 格式 */
  created_at: string
  /** 最后更新时间，ISO 8601 格式 */
  updated_at: string
}

/**
 * AuthorInfo: 作者简要信息
 *
 * 对应 proto 消息：api.common.v1.AuthorInfo
 *
 * 字段说明：
 * - id:       作者（用户）唯一标识（uint64）
 * - username: 作者用户名
 * - avatar:   作者头像 URL
 *
 * 设计说明：相比 UserInfo，AuthorInfo 仅包含文章列表中需要展示的少量字段，
 * 减少了网络传输体积。当需要完整用户信息时，使用 UserInfo 类型。
 */
export interface AuthorInfo {
  /** 作者 ID，uint64 */
  id: number
  username: string
  avatar: string
}
