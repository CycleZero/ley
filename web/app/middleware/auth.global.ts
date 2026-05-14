/**
 * 全局认证中间件（auth.global.ts）
 *
 * 文件名中的 .global 后缀表示这是一个全局中间件，
 * Nuxt 会在每次路由切换时自动执行，无需在页面中手动声明。
 *
 * 功能：
 * - 在每次页面导航前从 localStorage 恢复认证状态（token、用户信息）
 * - 确保用户刷新页面或重新打开标签页后仍然保持登录状态
 *
 * 执行时机：
 * - Nuxt 客户端导航时：每次路由切换前执行
 * - 首次加载应用时：在页面渲染前执行
 *
 * 注意：全局中间件会在每次路由变化时都执行，因此其逻辑应保持轻量。
 * restoreFromStorage 仅从 localStorage 读取数据，无网络请求，开销极小。
 */

export default defineNuxtRouteMiddleware(() => {
  /**
   * 获取 Pinia 认证 Store 实例
   * useAuthStore 定义在 app/stores/auth.ts
   */
  const auth = useAuthStore()

  /**
   * 从 localStorage 恢复认证数据：
   * - accessToken (ley_access_token): 短期访问令牌
   * - refreshToken (ley_refresh_token): 长期刷新令牌
   *
   * restoreFromStorage 内部已检查 import.meta.server，
   * 服务端渲染时跳过（SSR 无 localStorage），避免报错。
   */
  auth.restoreFromStorage()
})
