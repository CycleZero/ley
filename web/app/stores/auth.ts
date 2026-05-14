/**
 * 认证状态管理 Store（auth.ts）
 *
 * 基于 Pinia 的 Setup Store 语法（defineStore + 组合式函数）。
 * 管理用户认证相关的所有状态和操作，是前端认证体系的核心。
 *
 * 核心职责：
 * - 维护认证状态（用户信息、访问令牌、刷新令牌）
 * - 提供登录 / 注册 / 登出 / Token 刷新 / 获取用户资料等异步操作
 * - 将 Token 持久化到 localStorage 并从其中恢复状态
 *
 * 设计决策：
 * - 使用 Pinia Setup Store 而非 Options Store，更符合 Vue 3 组合式 API 风格
 * - Token 存储在 localStorage 而非 Cookie，避免 CSRF 风险（前端显式附带 Bearer Token）
 * - 刷新令牌与访问令牌分离，支持静默续期（doRefresh）
 * - isLoggedIn / isAdmin 使用 computed 派生，保证状态一致性
 */

import type { UserInfo } from '~~/lib/types'
import { authApi } from '~~/lib/api/auth'

/**
 * useAuthStore: Pinia Store 实例获取函数
 *
 * 使用 defineStore('auth', ...) 定义一个 ID 为 'auth' 的 Store，
 * 在组件中通过 const authStore = useAuthStore() 获取单例实例。
 */
export const useAuthStore = defineStore('auth', () => {
  /**
   * user: 当前登录用户信息
   * null 表示未获取用户资料或未登录
   */
  const user = ref<UserInfo | null>(null)

  /**
   * accessToken: JWT 访问令牌（短有效期，如 15 分钟）
   * 每次 API 请求通过 Authorization: Bearer <token> 头携带
   */
  const accessToken = ref<string | null>(null)

  /**
   * refreshToken: JWT 刷新令牌（长有效期，如 7 天）
   * 访问令牌过期后，使用刷新令牌获取新的访问令牌，无需重新登录
   */
  const refreshToken = ref<string | null>(null)

  /**
   * isLoggedIn: 派生状态，判断用户是否已登录
   * 仅基于 accessToken 是否存在，不验证 token 是否过期
   * （过期检测由 API 层的 401 拦截器处理）
   */
  const isLoggedIn = computed(() => !!accessToken.value)

  /**
   * isAdmin: 派生状态，判断当前用户是否为管理员
   * 用于前端权限控制（路由守卫、按钮显示/隐藏）
   */
  const isAdmin = computed(() => user.value?.role === 'admin')

  /**
   * restoreFromStorage: 从 localStorage 恢复认证状态
   *
   * 调用时机：
   * - auth.global.ts 全局中间件在每次路由切换时调用
   * - 应用首次加载时需要恢复之前保存的登录状态
   *
   * 实现细节：
   * - 服务端渲染时（import.meta.server === true）直接返回，
   *   因为 SSR 环境没有 localStorage API
   * - 仅恢复 token，不恢复 user 对象（user 需要通过 fetchProfile 重新获取）
   */
  function restoreFromStorage() {
    if (import.meta.server) return
    accessToken.value = localStorage.getItem('ley_access_token')
    refreshToken.value = localStorage.getItem('ley_refresh_token')
  }

  /**
   * login: 用户登录
   *
   * 参数：
   * @param account  - 用户名或邮箱（支持两种方式登录）
   * @param password - 密码
   *
   * 返回值：登录成功后的 UserInfo 对象
   *
   * 流程：
   * 1. 调用后端登录 API
   * 2. 将返回的 Token 保存到响应式状态和 localStorage
   * 3. 更新 user 信息
   */
  async function login(account: string, password: string) {
    const res = await authApi.login({ account, password })
    accessToken.value = res.token_pair.access_token
    refreshToken.value = res.token_pair.refresh_token
    user.value = res.user
    localStorage.setItem('ley_access_token', res.token_pair.access_token)
    localStorage.setItem('ley_refresh_token', res.token_pair.refresh_token)
    return res.user
  }

  /**
   * register: 用户注册
   *
   * 参数：
   * @param username - 用户名
   * @param email    - 邮箱地址
   * @param password - 密码
   *
   * 返回值：注册成功后的 UserInfo 对象
   *
   * 设计说明：注册成功后自动完成登录（无需再调用 login），
   * 后端直接在注册响应中返回 TokenPair 和 UserInfo。
   */
  async function register(username: string, email: string, password: string) {
    const res = await authApi.register({ username, email, password })
    accessToken.value = res.token_pair.access_token
    refreshToken.value = res.token_pair.refresh_token
    user.value = res.user
    localStorage.setItem('ley_access_token', res.token_pair.access_token)
    localStorage.setItem('ley_refresh_token', res.token_pair.refresh_token)
    return res.user
  }

  /**
   * logout: 用户登出
   *
   * 流程：
   * 1. 如果存在刷新令牌，调用后端 logout API 使其失效（可选，.catch(() => {}) 忽略网络错误）
   * 2. 清空所有响应式状态（user、accessToken、refreshToken）
   * 3. 清除 localStorage 中的持久化数据
   *
   * 边界情况：
   * - 即使后端 logout API 调用失败（网络断开、服务端错误），
   *   前端仍会清除本地状态，确保用户能正常退出
   */
  async function logout() {
    const rt = refreshToken.value
    if (rt) {
      // 尝试通知后端使 Token 失效，但即使失败也不阻塞登出流程
      await authApi.logout(rt).catch(() => {})
    }
    accessToken.value = null
    refreshToken.value = null
    user.value = null
    localStorage.removeItem('ley_access_token')
    localStorage.removeItem('ley_refresh_token')
  }

  /**
   * fetchProfile: 获取当前用户资料
   *
   * 返回值：最新的 UserInfo 对象
   *
   * 调用时机：
   * - 页面刷新后恢复 token 但丢失 user 对象时
   * - 用户编辑个人资料后需要同步最新信息
   *
   * 注意：此方法需要有效的 accessToken（Authorization 头），
   * 如果 token 已过期且刷新失败，API 层会抛出异常。
   */
  async function fetchProfile() {
    const res = await authApi.getProfile()
    user.value = res.user
    return res.user
  }

  /**
   * doRefresh: 执行 Token 刷新（静默续期）
   *
   * 使用场景：
   * - API 客户端在收到 401 响应时自动调用（见 lib/api/client.ts）
   * - 用户手动触发 Token 续期
   *
   * 前置条件：refreshToken 必须存在，否则抛出 Error
   *
   * 流程：
   * 1. 用当前的 refreshToken 向 /api/v1/auth/refresh 请求新的 TokenPair
   * 2. 更新内存和 localStorage 中的 token
   * 3. 更新 user 信息（后端可能返回最新的用户信息）
   *
   * 异常处理：由调用方捕获，通常 API 客户端会在刷新失败后重新抛出原始 401 错误
   */
  async function doRefresh() {
    if (!refreshToken.value) throw new Error('no refresh token')
    const res = await authApi.refresh(refreshToken.value)
    accessToken.value = res.token_pair.access_token
    refreshToken.value = res.token_pair.refresh_token
    user.value = res.user
    localStorage.setItem('ley_access_token', res.token_pair.access_token)
    localStorage.setItem('ley_refresh_token', res.token_pair.refresh_token)
  }

  /**
   * 导出供外部使用的状态和操作方法。
   * 注意：accessToken 和 refreshToken 也导出，供 API 客户端中的工具函数
   * getAccessToken() / getRefreshToken() 读取（但推荐通过此 Store 访问）。
   */
  return {
    user,
    accessToken,
    refreshToken,
    isLoggedIn,
    isAdmin,
    restoreFromStorage,
    login,
    register,
    logout,
    fetchProfile,
    doRefresh,
  }
})
