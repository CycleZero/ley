// https://nuxt.com/docs/api/configuration/nuxt-config
export default defineNuxtConfig({
  compatibilityDate: '2025-07-15',
  devtools: { enabled: false },

  modules: [
    '@pinia/nuxt',
    '@nuxtjs/tailwindcss',
    '@vueuse/nuxt',
    '@nuxtjs/mdc',
    '@nuxt/image',
    '@nuxtjs/color-mode',
  ],

  // 组件自动导入配置：子目录组件不带路径前缀
  components: [
    { path: '~/components/ui', pathPrefix: false },
    { path: '~/components/animation', pathPrefix: false },
    { path: '~/components/article', pathPrefix: false },
    { path: '~/components/layout', pathPrefix: false },
  ],

  css: ['~/assets/css/main.css'],

  runtimeConfig: {
    public: {
      apiBase: process.env.NUXT_PUBLIC_API_BASE || 'http://172.18.240.1:8080',
    },
  },

  // 颜色模式配置
  colorMode: {
    classSuffix: '',
    preference: 'system',
    fallback: 'light',
    dataValue: 'theme',
  },

  // MDC 配置
  mdc: {
    highlight: {
      theme: {
        dark: 'github-dark',
        light: 'github-light',
      },
    },
  },

  // Image 配置
  image: {
    quality: 80,
    format: ['webp'],
  },

  // PostCSS 配置（Nuxt 推荐在配置中定义）
  postcss: {
    plugins: {
      tailwindcss: {},
      autoprefixer: {},
    },
  },

  // Nuxt 4 App 目录结构
  future: {
    compatibilityVersion: 4,
  },

  app: {
    head: {
      htmlAttrs: {
        lang: 'zh-CN',
      },
      meta: [
        { charset: 'utf-8' },
        { name: 'viewport', content: 'width=device-width, initial-scale=1' },
      ],
      link: [
        { rel: 'icon', type: 'image/svg+xml', href: '/logo.svg' },
        // 霞鹜文楷
        {
          rel: 'stylesheet',
          href: 'https://cdn.jsdelivr.net/npm/lxgw-wenkai-webfont@1.7.0/style.css',
        },
        // 思源宋体
        {
          rel: 'stylesheet',
          href: 'https://fonts.googleapis.com/css2?family=Noto+Serif+SC:wght@400;600;700&display=swap',
        },
        // Inter
        {
          rel: 'stylesheet',
          href: 'https://fonts.googleapis.com/css2?family=Inter:wght@400;500;600&display=swap',
        },
        // JetBrains Mono
        {
          rel: 'stylesheet',
          href: 'https://fonts.googleapis.com/css2?family=JetBrains+Mono:wght@400;500&display=swap',
        },
      ],
    },
  },
})
