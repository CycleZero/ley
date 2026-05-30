/**
 * API 客户端封装
 *
 * 基于 Nuxt 4 的 $fetch（ofetch），提供：
 * - 自动注入 Bearer Token
 * - 统一 baseURL
 *
 * 注意：auth store 中内联了 login/register/refresh/logout API，
 * 避免与 useApiClient 形成循环依赖。
 */
export function useApiClient() {
  const config = useRuntimeConfig()
  const auth = useAuthStore()

  return $fetch.create({
    baseURL: config.public.apiBase,

    onRequest({ options }) {
      const token = auth.accessToken
      if (token) {
        options.headers = {
          ...(options.headers || {}),
          Authorization: `Bearer ${token}`,
        }
      }
    },
  })
}
