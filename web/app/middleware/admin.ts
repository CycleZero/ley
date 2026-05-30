import type { NavigationGuardNext } from 'vue-router'

/**
 * 管理员权限中间件
 *
 * 检查用户是否已登录且为管理员角色。
 * 未登录跳转到登录页，非管理员跳转到首页。
 */
export default defineNuxtRouteMiddleware(async (to, from) => {
  const auth = useAuthStore()

  // 等待登录态恢复
  if (!auth.user && auth.accessToken) {
    await auth.init()
  }

  // 未登录
  if (!auth.isLoggedIn) {
    return navigateTo('/login')
  }

  // 已登录但非管理员
  if (!auth.isAdmin) {
    return navigateTo('/')
  }
})
