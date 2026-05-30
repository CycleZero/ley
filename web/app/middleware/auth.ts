import type { NavigationGuardNext } from 'vue-router'

/**
 * 认证中间件
 *
 * 检查用户是否已登录，未登录则重定向到登录页。
 * 有 Token 但 user 未恢复时，先等待 init() 完成再判断。
 */
export default defineNuxtRouteMiddleware(async (to, from) => {
  const auth = useAuthStore()

  // 没有 accessToken cookie，直接跳转
  if (!auth.accessToken) {
    return navigateTo('/login')
  }

  // 有 token 但还没恢复 user（刷新页面后），等待 init
  if (!auth.user) {
    await auth.init()
  }

  // init 后仍未登录（Token 无效/过期且刷新失败）
  if (!auth.isLoggedIn) {
    return navigateTo('/login')
  }
})
