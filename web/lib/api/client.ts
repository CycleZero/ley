/**
 * HTTP API 客户端（client.ts）
 *
 * 本项目 API 层的核心模块，封装了所有与后端通信的逻辑：
 * - 统一的请求方法（GET / POST / PUT / DELETE）
 * - 自动附加 Bearer Token 认证头
 * - 401 响应自动触发 Token 刷新并重试请求
 * - Token 刷新并发控制（多个并发请求共享同一个刷新过程）
 *
 * 导出的 api 对象被注册为 Nuxt 插件（见 app/plugins/api.ts），
 * 通过 $api 在组件中访问。上层的业务 API 模块（auth.ts、articles.ts 等）
 * 通过 import { api } from './client' 直接使用。
 *
 * 关键设计决策：
 * 1. Token 存储在 localStorage 而非 Cookie，避免 CSRF 自动附带
 * 2. Token 刷新采用单 Promise 模式，防止并发 401 导致多次刷新
 * 3. 服务端渲染时 getAccessToken/getRefreshToken 返回 null（SSR 无 localStorage）
 * 4. 所有请求函数为泛型，调用方可指定返回类型
 */

import type { RefreshTokenReply } from '~~/lib/types'

/**
 * refreshPromise: 全局 Token 刷新 Promise 引用
 *
 * 用于实现"刷新去重"——当多个并发请求同时收到 401 时，
 * 只有第一个请求会发起实际的刷新请求，其他请求等待同一个 Promise 完成。
 * 刷新完成后（成功或失败）设为 null，允许后续的刷新。
 *
 * 类型为 Promise<void> | null，null 表示当前没有正在进行的刷新。
 */
let refreshPromise: Promise<void> | null = null

/**
 * tryRefresh: 尝试刷新访问令牌
 *
 * 返回值：Promise<boolean> —— true 表示刷新成功，false 表示失败
 *
 * 执行逻辑：
 * 1. 检查是否存在 refreshToken，无则直接返回 false
 * 2. 如果 refreshPromise 已存在（有正在进行的刷新），等待其完成并返回 true
 * 3. 否则创建新的刷新请求（POST /api/v1/auth/refresh）
 * 4. 刷新成功后更新 localStorage 中的 token
 * 5. 无论成功或失败，finally 中将 refreshPromise 重置为 null
 *
 * 注意：此函数直接使用 $fetch 而非调用 authApi.refresh，
 * 因为 authApi 内部又依赖 api（即本模块），会造成循环依赖。
 * 同时不使用 request() 封装，因为 401 处理会递归调用自身。
 */
async function tryRefresh(): Promise<boolean> {
  const token = getRefreshToken()
  if (!token) return false

  // 并发去重：已有刷新进行中，等待其完成即可
  if (refreshPromise) {
    await refreshPromise
    return true
  }

  refreshPromise = $fetch<RefreshTokenReply>('/api/v1/auth/refresh', {
    baseURL: useRuntimeConfig().public.apiBaseUrl as string,
    method: 'POST',
    body: { refresh_token: token },
  }).then((res) => {
    // 刷新成功：更新 localStorage 中的两个 token
    localStorage.setItem('ley_access_token', res.token_pair.access_token)
    localStorage.setItem('ley_refresh_token', res.token_pair.refresh_token)
  }).catch(() => {
    // 刷新失败（如 refreshToken 也过期）：不抛出，返回 false
    return false
  }).finally(() => {
    // 无论成功失败，重置全局 Promise 引用，允许后续刷新
    refreshPromise = null
  }) as unknown as Promise<void>

  await refreshPromise
  // 返回时再次检查 accessToken 是否已成功保存（双重保险）
  return getAccessToken() !== null
}

/**
 * request: 发送 HTTP 请求的核心函数
 *
 * 类型参数：
 * @template T - 期望的响应体类型
 *
 * 参数：
 * @param method - HTTP 方法：'GET' | 'POST' | 'PUT' | 'DELETE'
 * @param path   - 请求路径（相对于 baseURL），例如 '/api/v1/articles'
 * @param extra  - 可选参数：
 *   - query: URL 查询参数对象，自动序列化为 query string
 *   - body:  请求体对象（POST / PUT 时使用），JSON 序列化
 *
 * 返回值：Promise<T> —— 解析后的响应体数据
 *
 * 401 自动重试机制：
 * 1. 首次请求返回 401
 * 2. 检查是否存在 refreshToken（无则直接抛错）
 * 3. 调用 tryRefresh() 尝试刷新 token
 * 4. 刷新成功：用新 token 重新发送原始请求（仅重试一次）
 * 5. 刷新失败：将原始 401 错误重新抛出给调用方处理
 */
async function request<T>(
  method: 'GET' | 'POST' | 'PUT' | 'DELETE',
  path: string,
  extra?: { query?: Record<string, unknown>; body?: Record<string, unknown> },
): Promise<T> {
  const config = useRuntimeConfig()
  const baseURL = config.public.apiBaseUrl as string

  // 构建请求头，如果存在 accessToken 则附加 Bearer 认证头
  const headers: Record<string, string> = {}

  const token = getAccessToken()
  if (token) {
    headers['Authorization'] = `Bearer ${token}`
  }

  /**
   * doFetch: 实际发送请求的闭包
   * 抽取为函数便于 401 重试时复用
   */
  const doFetch = () => $fetch<T>(path, {
    baseURL,
    method,
    headers,
    query: extra?.query,
    body: extra?.body,
  })

  try {
    return await doFetch()
  } catch (err: unknown) {
    const error = err as { response?: { status: number } }
    // 仅当响应状态码为 401 且存在 refreshToken 时才尝试刷新
    if (error?.response?.status === 401 && getRefreshToken()) {
      const ok = await tryRefresh()
      if (ok) {
        // 刷新成功：用新 token 更新 Authorization 头并重试
        const token = getAccessToken()
        if (token) headers['Authorization'] = `Bearer ${token}`
        return doFetch()
      }
    }
    // 非 401 错误、无 refreshToken、或刷新失败：直接抛出原始错误
    throw err
  }
}

/**
 * getAccessToken: 从 localStorage 获取访问令牌
 *
 * 返回值：string | null
 *
 * 服务端渲染保护：import.meta.server 为 true 时直接返回 null，
 * 因为 Node.js 环境没有 localStorage API。
 *
 * 注意：此处直接读取 localStorage 而非通过 useAuthStore，
 * 是因为 api 模块在 Pinia Store 初始化前就可能被调用，
 * 且避免建立双向依赖。
 */
function getAccessToken(): string | null {
  if (import.meta.server) return null
  return localStorage.getItem('ley_access_token')
}

/**
 * getRefreshToken: 从 localStorage 获取刷新令牌
 *
 * 返回值：string | null
 *
 * 同样包含 SSR 保护逻辑。
 */
function getRefreshToken(): string | null {
  if (import.meta.server) return null
  return localStorage.getItem('ley_refresh_token')
}

/**
 * api: 导出的 HTTP 客户端对象
 *
 * 提供四个便捷方法，对应 RESTful 的 CRUD 操作：
 * - api.get<T>(path, query?)        → GET 请求
 * - api.post<T>(path, body?)        → POST 请求
 * - api.put<T>(path, body?)         → PUT 请求
 * - api.delete<T>(path)             → DELETE 请求
 *
 * 所有方法均返回 Promise<T>，T 为期望的响应类型。
 * 调用示例：
 *   const res = await api.get<GetArticleReply>(`/api/v1/articles/${id}`)
 */
export const api = {
  /**
   * 发送 GET 请求
   * @param path  - 请求路径
   * @param query - 可选的查询参数对象，如 { page: 1, page_size: 10 }
   */
  get<T>(path: string, query?: Record<string, unknown>) {
    return request<T>('GET', path, { query })
  },

  /**
   * 发送 POST 请求
   * @param path - 请求路径
   * @param body - 可选的请求体对象，JSON 序列化
   */
  post<T>(path: string, body?: Record<string, unknown>) {
    return request<T>('POST', path, { body })
  },

  /**
   * 发送 PUT 请求
   * @param path - 请求路径
   * @param body - 可选的请求体对象，JSON 序列化
   */
  put<T>(path: string, body?: Record<string, unknown>) {
    return request<T>('PUT', path, { body })
  },

  /**
   * 发送 DELETE 请求
   * @param path - 请求路径（DELETE 请求通常不带 body，参数通过 URL 传递）
   */
  delete<T>(path: string) {
    return request<T>('DELETE', path)
  },
}
