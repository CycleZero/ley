/**
 * 管理员权限中间件（admin.ts）
 *
 * 非全局中间件，需要在页面中使用 definePageMeta({ middleware: 'admin' }) 显式声明。
 * 通常用于 /admin/** 路由的保护。
 *
 * 功能：
 * 1. 检查用户是否已登录
 * 2. 检查已登录用户的角色是否为 'admin'
 *
 * 重定向逻辑：
 * - 未登录 → 跳转到 /auth/login 登录页
 * - 已登录但非管理员 → 跳转到首页 /
 *
 * 返回值含义：
 * - navigateTo() 返回一个导航对象，Nuxt 会执行客户端重定向
 * - void（无返回值）表示放行，允许继续访问目标路由
 *
 * 安全边界：
 * - 前端中间件仅作为用户体验优化（避免看到无权限的页面），
 *   真正的权限校验必须由后端 API 完成。
 * - 中间件执行在客户端和服务端均可能发生，但 admin 路由
 *   在 nuxt.config.ts 中已设置 ssr: false，确保仅在客户端运行。
 */

export default defineNuxtRouteMiddleware(() => {
  const auth = useAuthStore()

  /**
   * 第一步：检查登录状态
   * isLoggedIn 是一个 computed 属性，基于 accessToken 是否存在判断
   */
  if (!auth.isLoggedIn) {
    // 未登录：重定向到登录页
    return navigateTo('/auth/login')
  }

  /**
   * 第二步：检查管理员角色
   * user 对象可能为 null（如 token 存在但未拉取用户信息），
   * 使用可选链 user?.role 安全访问。
   */
  if (auth.user && auth.user.role !== 'admin') {
    // 非管理员：重定向到首页
    return navigateTo('/')
  }

  // 通过校验，正常放行
})
