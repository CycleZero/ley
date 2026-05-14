/**
 * 认证相关 API 模块（auth.ts）
 *
 * 封装所有与用户认证相关的后端 API 调用，包括：
 * - 用户注册（register）
 * - 用户登录（login）
 * - Token 刷新（refresh）
 * - 登出（logout）
 * - 获取/更新个人资料（getProfile / updateProfile）
 *
 * 每个方法均返回泛型 Promise<T>，T 为对应的 Reply 类型，
 * 类型定义来自 ../types（lib/types/auth.ts）。
 *
 * 依赖关系：本模块依赖 ./client 中的 api 对象，
 * api 对象内部处理了 Token 附加和 401 自动刷新。
 */

import type { LoginReply, RegisterReply, RefreshTokenReply, GetProfileReply, UpdateProfileReply } from '../types'
import { api } from './client'

/**
 * authApi: 认证 API 命名空间对象
 *
 * 将所有认证相关接口集中管理，调用方式：
 * import { authApi } from '~~/lib/api/auth'
 * const res = await authApi.login({ account: 'user', password: 'pass' })
 */
export const authApi = {
  /**
   * register: 用户注册
   *
   * 参数：
   * @param body.username - 用户名（唯一标识，用于登录和展示）
   * @param body.email    - 邮箱地址（用于找回密码等）
   * @param body.password - 明文密码（后端负责加密存储）
   *
   * 返回值：Promise<RegisterReply>
   * RegisterReply 包含 user (UserInfo) 和 token_pair (TokenPair)
   *
   * 注意：注册成功后后端自动返回 TokenPair，前端无需额外调用登录
   */
  register(body: { username: string; email: string; password: string }) {
    return api.post<RegisterReply>('/api/v1/auth/register', body)
  },

  /**
   * login: 用户登录
   *
   * 参数：
   * @param body.account  - 账号（支持用户名或邮箱两种方式）
   * @param body.password - 密码
   *
   * 返回值：Promise<LoginReply>
   * 同 register 一样包含 user 和 token_pair
   */
  login(body: { account: string; password: string }) {
    return api.post<LoginReply>('/api/v1/auth/login', body)
  },

  /**
   * refresh: 刷新 Token
   *
   * 参数：
   * @param refreshToken - 当前有效的刷新令牌
   *
   * 返回值：Promise<RefreshTokenReply>
   * 返回新的 TokenPair（新的 accessToken 和 refreshToken）
   *
   * 注意：调用后旧的 refreshToken 将失效（Token 轮转机制）
   */
  refresh(refreshToken: string) {
    return api.post<RefreshTokenReply>('/api/v1/auth/refresh', { refresh_token: refreshToken })
  },

  /**
   * logout: 登出
   *
   * 参数：
   * @param refreshToken - 需要失效的刷新令牌
   *
   * 返回值：Promise<void> —— 空响应体
   *
   * 作用：通知后端将指定的 refreshToken 标记为无效，
   * 防止该 token 被继续用于刷新。
   */
  logout(refreshToken: string) {
    return api.post<void>('/api/v1/auth/logout', { refresh_token: refreshToken })
  },

  /**
   * getProfile: 获取当前登录用户的个人资料
   *
   * 返回值：Promise<GetProfileReply>
   * GetProfileReply 仅包含 user 字段（不含 TokenPair，因为无需重新颁发 Token）
   *
   * 认证要求：需要在 Authorization 头中携带有效的 accessToken
   */
  getProfile() {
    return api.get<GetProfileReply>('/api/v1/users/me')
  },

  /**
   * updateProfile: 更新当前用户的个人资料
   *
   * 参数：
   * @param body.avatar - 头像 URL（可选）
   * @param body.bio    - 个人简介（可选）
   *
   * 返回值：Promise<UpdateProfileReply>
   * 返回更新后的完整 UserInfo
   *
   * 注意：目前仅支持更新头像和简介两个字段，
   * 用户名和邮箱修改请通过其他专门的 API。
   */
  updateProfile(body: { avatar?: string; bio?: string }) {
    return api.put<UpdateProfileReply>('/api/v1/users/me', body)
  },
}
