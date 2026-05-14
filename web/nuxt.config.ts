/**
 * Nuxt 应用配置文件
 *
 * 本文件定义了整个前端项目的框架级配置，包括：
 * - 构建兼容性日期与开发工具
 * - 使用的 Nuxt 模块（TailwindCSS、MDC、Pinia）
 * - 运行时环境变量（后端 API 地址）
 * - 路由级别的渲染与缓存策略（SSR / SWR）
 * - Nitro 服务器优化选项
 */

export default defineNuxtConfig({
  /**
   * compatibilityDate: 指定兼容性日期，确保 Nuxt 使用该日期对应的默认行为。
   * 当升级 Nuxt 版本时，该日期决定了向后兼容策略。
   */
  compatibilityDate: '2025-07-15',

  /**
   * devtools: 启用 Nuxt DevTools 调试面板，开发时可在浏览器中查看组件树、状态、
   * 路由、请求等信息。
   */
  devtools: { enabled: true },

  /**
   * modules: Nuxt 模块列表，模块会自动注入插件、组件、组合式函数等。
   * - @nuxtjs/tailwindcss: 集成 Tailwind CSS，自动检测并加载 tailwind.config
   * - @nuxtjs/mdc: 集成 Markdown Components，支持在 Nuxt Content 中使用 Vue 组件
   * - @pinia/nuxt: 集成 Pinia 状态管理库，自动注册 stores/ 下的 store 文件
   */
  modules: [
    '@nuxtjs/tailwindcss',
    '@nuxtjs/mdc',
    '@pinia/nuxt',
  ],

  /**
   * runtimeConfig: 运行时配置，可在服务端和客户端通过 useRuntimeConfig() 访问。
   * public 下的字段同时暴露给客户端和服务端，非 public 的仅服务端可用。
   */
  runtimeConfig: {
    public: {
      /**
       * apiBaseUrl: 后端 API 的基础地址。
       * 优先从环境变量 NUXT_PUBLIC_API_BASE_URL 读取，未设置时回退到本地 8000 端口。
       * 前导下划线避免在客户端通过 useRuntimeConfig() 直接暴露敏感内部配置。
       */
      apiBaseUrl: process.env.NUXT_PUBLIC_API_BASE_URL || 'http://localhost:8000',
    },
  },

  /**
   * routeRules: 路由规则——按路由模式匹配，配置每类路由的渲染策略。
   *
   * swr (Stale-While-Revalidate): 首次请求 SSR 渲染后，后续请求在 N 秒内
   * 直接返回缓存的页面（stale），同时在后台重新生成（revalidate）。
   * 适用于内容变化不频繁的页面，兼顾首屏性能与内容新鲜度。
   *
   * ssr: false 意味着该路由完全在客户端渲染（CSR），不进行服务端渲染。
   * 适用于认证页面和管理后台等不需要 SEO 的路由。
   */
  routeRules: {
    '/':                          { swr: 60 },   // 首页：60 秒 SWR 缓存
    '/articles':                  { swr: 60 },   // 文章列表页：60 秒 SWR 缓存
    '/articles/**':               { swr: 300 },  // 文章详情页：5 分钟 SWR 缓存（内容变化少）
    '/tags':                      { swr: 300 },  // 标签列表页：5 分钟 SWR 缓存
    '/tags/**':                   { swr: 60 },   // 标签详情页：60 秒 SWR 缓存
    '/categories/**':             { swr: 60 },   // 分类详情页：60 秒 SWR 缓存
    '/search':                    { ssr: false }, // 搜索页：纯客户端渲染（依赖用户输入）
    '/rss.xml':                   { swr: 300 },  // RSS Feed：5 分钟 SWR 缓存
    '/sitemap.xml':               { swr: 300 },  // Sitemap：5 分钟 SWR 缓存
    '/auth/**':                   { ssr: false }, // 认证相关页面：纯客户端渲染（不含 SEO 需求）
    '/admin/**':                  { ssr: false }, // 管理后台：纯客户端渲染（需要认证状态）
  },

  /**
   * nitro: Nitro 服务器引擎配置。
   * compressPublicAssets: 对 public/ 目录下的静态资源启用 gzip/brotli 压缩，
   * 减少传输体积，加快资源加载速度。
   */
  nitro: {
    compressPublicAssets: true,
  },
})
