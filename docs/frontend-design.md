# Ley Blog — 前端设计方案 (Vue 3 / Nuxt 4)

> 版本: v2.1 | 日期: 2026-05 | 状态: 实施中

---

## 1. 技术选型

| 类别 | 选型 | 理由 |
|------|------|------|
| 框架 | **Nuxt 4** (Vue 3 + Composition API) | 基于 Vite，内置 SSR/SSG/ISR；文件路由 + 自动导入，开发效率最高 |
| 语言 | **TypeScript** | 与后端 Proto 类型对齐，`<script setup lang="ts">` 原生支持 |
| 样式 | **TailwindCSS 3** | Nuxt 官方 `@nuxtjs/tailwindcss` 模块，零配置 |
| UI 组件 | **Radix Vue** + **Headless UI (Vue)** | 无头组件 + Tailwind，灵活性最强，a11y 内置 |
| HTTP 客户端 | **ofetch** (Nuxt 内置 `$fetch` / `useFetch`) | SSR 端自动附加 Cookie，浏览器端自动 polyfill，拦截器简洁 |
| 状态管理 | **Pinia** | Vue 官方，TypeScript 一等公民，Devtools 支持，模块化 Store |
| 表单 | **VeeValidate + Zod** | `useField` / `useForm` 配合 Zod schema，声明式校验 |
| 富文本 | **TipTap (Vue 3 bindings)** | ProseMirror 核心，Vue 3 第一方支持，扩展丰富 |
| Markdown 渲染 | **nuxt-mdc** (`@nuxtjs/mdc`) | 服务端渲染 Markdown → HTML，shiki 代码高亮，零客户端开销 |
| 图标 | **Lucide Vue Next** | 轻量，tree-shakable |
| Testing | **Vitest** + **Playwright** | Vitest 原生 Vite 集成；Playwright 跨浏览器 E2E |
| 包管理 | **pnpm** | 磁盘高效，lockfile 稳定 |

---

## 2. 项目结构

```
web/
├── package.json
├── pnpm-lock.yaml
├── tsconfig.json
├── nuxt.config.ts                # Nuxt 配置（模块/SSR/运行时）
├── Dockerfile
│
├── public/                       # 静态资源（直接映射到 /）
│   ├── favicon.ico
│   ├── robots.txt
│   └── images/
│       └── default-cover.jpg
│
├── server/                       # Nitro 服务端（SSR + API 代理）
│   ├── api/                      # 可选：服务端 API 代理
│   │   └── proxy/                # 将 /api/* 代理到 Gateway
│   └── routes/
│       ├── sitemap.xml.ts        # 动态 sitemap 生成
│       └── rss.xml.ts            # RSS Feed
│
├── app/                          # Nuxt 4 应用目录（source root）
│   ├── app.vue                   # 根组件（NuxtLayout → NuxtPage）
│   │
│   ├── pages/                    # 基于文件的路由
│   │   ├── index.vue             # 首页（ISR）
│   │   │
│   │   ├── articles/
│   │   │   ├── index.vue         # 文章列表（ISR + 筛选）
│   │   │   └── [slug].vue        # 文章详情（SSG + 增量 ISR）
│   │   │
│   │   ├── tags/
│   │   │   ├── index.vue         # 标签云
│   │   │   └── [slug].vue        # 标签下文章列表
│   │   │
│   │   ├── categories/
│   │   │   └── [slug].vue        # 分类下文章列表
│   │   │
│   │   ├── search.vue            # 搜索页
│   │   │
│   │   ├── auth/
│   │   │   ├── login.vue         # 登录（CSR）
│   │   │   └── register.vue      # 注册（CSR）
│   │   │
│   │   └── admin/                # 管理后台（全部 CSR + 认证守卫）
│   │       ├── index.vue         # Dashboard
│   │       ├── articles/
│   │       │   ├── index.vue     # 文章管理列表
│   │       │   ├── new.vue       # 新建文章
│   │       │   └── [id]/
│   │       │       └── edit.vue  # 编辑文章
│   │       ├── comments/
│   │       │   └── index.vue     # 评论管理
│   │       ├── media/
│   │       │   └── index.vue     # 媒体库
│   │       ├── tags/
│   │       │   └── index.vue     # 标签管理
│   │       ├── categories/
│   │       │   └── index.vue     # 分类管理
│   │       ├── settings/
│   │       │   ├── index.vue     # 站点配置
│   │       │   └── appearance.vue# 背景图 & 歌单管理
│   │       └── profile/
│   │           └── index.vue     # 个人资料
│   │
│   ├── layouts/                  # 布局组件（Nuxt 自动应用）
│   │   ├── default.vue           # 公开页布局（Header + Content + Footer）
│   │   └── admin.vue             # 管理后台布局（Sidebar + Topbar + Content）
│   │
│   ├── components/               # 组件（Nuxt 自动导入，无需 import）
│   │   ├── ui/                   # 基础 UI 组件
│   │   │   ├── AppButton.vue
│   │   │   ├── AppInput.vue
│   │   │   ├── AppModal.vue      # Teleport 弹窗
│   │   │   ├── AppToast.vue      # 全局 Toast（Pinia 驱动）
│   │   │   ├── AppPagination.vue
│   │   │   ├── AppEmpty.vue      # 空状态占位
│   │   │   ├── AppLoading.vue    # 骨架屏 / 加载态
│   │   │   ├── AppConfirm.vue    # 确认对话框
│   │   │   └── AppToggle.vue     # 开关组件
│   │   │
│   │   ├── layout/
│   │   │   ├── PublicHeader.vue
│   │   │   ├── PublicFooter.vue
│   │   │   ├── AdminSidebar.vue
│   │   │   └── AdminTopbar.vue
│   │   │
│   │   ├── article/
│   │   │   ├── ArticleCard.vue
│   │   │   ├── ArticleList.vue
│   │   │   ├── ArticleContent.vue
│   │   │   ├── ArticleMeta.vue
│   │   │   ├── ArticleTOC.vue
│   │   │   ├── LikeButton.vue
│   │   │   └── ShareButtons.vue
│   │   │
│   │   ├── comment/
│   │   │   ├── CommentList.vue
│   │   │   ├── CommentItem.vue
│   │   │   └── CommentForm.vue
│   │   │
│   │   ├── tag/
│   │   │   ├── TagBadge.vue
│   │   │   └── TagCloud.vue
│   │   │
│   │   ├── category/
│   │   │   └── CategoryTree.vue
│   │   │
│   │   ├── editor/
│   │   │   ├── ArticleEditor.vue
│   │   │   ├── EditorToolbar.vue
│   │   │   └── ImageUploader.vue
│   │   │
│   │   └── auth/
│   │       ├── LoginForm.vue
│   │       └── RegisterForm.vue
│   │
│   ├── composables/              # 组合式函数（Nuxt 自动导入）
│   │   ├── useAuth.ts
│   │   ├── usePagination.ts
│   │   ├── useDebounce.ts
│   │   ├── useMediaQuery.ts
│   │   └── useTheme.ts
│   │
│   ├── middleware/                # Nuxt 路由中间件
│   │   ├── auth.global.ts
│   │   └── admin.ts
│   │
│   ├── plugins/                  # Nuxt 插件
│   │   └── api.ts                # 注册 $api（NuxtApp 全局注入）
│   │
│   └── assets/
│       └── css/
│           └── main.css          # TailwindCSS 指令 + 全局 CSS 变量
│
├── lib/                          # 工具库（不在 app/ 下，Nuxt 仍可解析 imports）
│   ├── api/                      # API 调用函数（按服务模块拆分）
│   │   ├── client.ts             # $fetch 封装：baseURL / 错误处理 / 401 刷新
│   │   ├── auth.ts               # register / login / refresh / logout / getProfile
│   │   ├── articles.ts           # CRUD + publish + archive + search + like
│   │   ├── comments.ts           # CRUD + approve + markSpam
│   │   ├── tags.ts               # CRUD
│   │   ├── categories.ts         # CRUD
│   │   ├── files.ts              # upload / list / delete / presignedUrl
│   │   └── site.ts               # config / backgrounds / playlist
│   │
│   ├── types/                    # 类型定义，与 Proto 一一对应
│   │   ├── index.ts              # 统一导出
│   │   ├── common.ts             # TokenPair / UserInfo / AuthorInfo
│   │   ├── auth.ts               # 认证请求/响应
│   │   ├── article.ts            # ArticleInfo / ListArticlesRequest
│   │   ├── comment.ts            # CommentInfo / CommentNode
│   │   ├── tag.ts                # TagInfo
│   │   ├── category.ts           # CategoryInfo（递归 children）
│   │   ├── file.ts               # FileInfo
│   │   └── site.ts               # SiteConfig / SiteBackground / MusicPlaylist
│   │
│   ├── validators/               # Zod Schema（与 Proto 校验对齐）
│   │   ├── auth.ts               # RegisterSchema / LoginSchema
│   │   ├── article.ts            # ArticleSchema / UpdateArticleSchema
│   │   └── site.ts               # SiteConfigSchema
│   │
│   └── utils/
│       ├── format.ts             # 日期 / 数字 / 字节格式化
│       ├── cn.ts                # clsx + tailwind-merge 类名合并
│       └── jwt.ts                # JWT decode（仅解析 exp，不校验签名）
│
├── stores/                       # Pinia Store（状态管理）
│   ├── auth.ts                   # authStore: Token / UserInfo / login / logout
│   ├── toast.ts                  # toastStore: 全局通知队列
│   └── theme.ts                  # themeStore: 暗色模式持久化
│
├── __tests__/                    # 测试
│   ├── unit/                     # Vitest 单元测试
│   └── e2e/                      # Playwright E2E
│
├── .env                          # NUXT_PUBLIC_API_BASE_URL=http://localhost:8000
└── .dockerignore
```

### Nuxt 4 核心特性说明

| 特性 | 说明 |
|------|------|
| **app/ 源目录** | `app/` 统一管理 pages/layouts/components/plugins/middleware/composables/stores，工程配置与业务代码分离 |
| **文件路由** | `app/pages/` 下文件自动生成路由，`[slug]` 动态参数，无需手写 router |
| **别名约定** | `~` → `app/`，`~~` → 项目根。`lib/` 等放在项目根的模块用 `~~/lib/` 引用 |
| **自动导入** | `app/components/` 下组件在模板中直接使用，无需 import；`app/composables/` 同理 |
| **布局系统** | `app/layouts/` 通过 `<NuxtLayout>` 切换；公开页用 `default.vue`，管理页用 `admin.vue` |
| **服务端引擎** | Nitro 提供 API Route、缓存、ISR、SWR 等，`server/` 目录下文件自动注册 |
| **SSR 控制** | `routeRules` 精确控制每页渲染策略；`<ClientOnly>` 包裹纯客户端组件 |
| **中间件** | `app/middleware/` 在路由切换时执行，`*.global.ts` 全局生效 |

---

## 3. 路由设计

### 3.1 nuxt.config.ts 核心配置

```typescript
// nuxt.config.ts
export default defineNuxtConfig({
  modules: ['@nuxtjs/tailwindcss', '@nuxtjs/mdc', 'nuxt-icon'],
  runtimeConfig: {
    public: {
      apiBaseUrl: process.env.NUXT_PUBLIC_API_BASE_URL || 'http://localhost:8000',
    },
  },
  routeRules: {
    '/':                          { swr: 60 },            // 首页 ISR 60s
    '/articles':                  { swr: 60 },
    '/articles/**':               { swr: 300 },           // 文章详情 ISR 5min
    '/tags':                      { swr: 300 },
    '/tags/**':                   { swr: 60 },
    '/categories/**':             { swr: 60 },
    '/search':                    { ssr: false },         // 搜索纯 CSR
    '/rss.xml':                   { swr: 300 },
    '/sitemap.xml':               { swr: 300 },
    '/auth/**':                   { ssr: false },         // 登录/注册纯 CSR
    '/admin/**':                  { ssr: false },         // 管理后台纯 CSR
  },
})
```

### 3.2 公开页面

| 路由 | 渲染策略 | 数据依赖 | 说明 |
|------|----------|----------|------|
| `/` | ISR 60s | `useFetch('/articles')` + `useFetch('/site/config')` | 首页：Banner + 最新 10 篇文章 |
| `/articles` | ISR 60s | `useFetch('/articles', { query })` | 支持 `?category_id=&tag=&page=` 筛选 |
| `/articles/[slug]` | ISR 300s | `useFetch('/articles/${slug}')` | 正文 Markdown + 评论区（客户端 interact） |
| `/tags` | ISR 300s | `useFetch('/tags')` | 标签云 |
| `/tags/[slug]` | ISR 60s | `useFetch('/articles?status=published&tags=${slug}')` | 标签下文章 |
| `/categories/[slug]` | ISR 60s | `useFetch('/articles?category_id=${id}')` | 分类下文章 |
| `/search` | CSR | 客户端 `useFetch('/articles/search?keyword=')` | 防抖 300ms，无缓存 |
| `/rss.xml` | ISR 300s | `server/routes/rss.xml.ts` | Nitro Route 动态生成 |

### 3.3 认证页面（纯 CSR）

| 路由 | 说明 |
|------|------|
| `/auth/login` | 登录表单，成功后跳转 `/admin` |
| `/auth/register` | 注册表单，成功后自动登录跳转 `/admin` |

### 3.4 管理后台（纯 CSR + `admin` 中间件守卫）

| 路由 | 中间件 | 说明 |
|------|--------|------|
| `/admin` | `auth` | Dashboard：统计概览（文章/评论/草稿计数） |
| `/admin/articles` | `auth` | 文章列表 Table：按状态筛选 + 批量操作 |
| `/admin/articles/new` | `auth` | 文章编辑器：TipTap + 右侧元信息面板 |
| `/admin/articles/[id]/edit` | `auth` | 编辑已有文章（复用 ArticleEditor） |
| `/admin/comments` | `auth` | 评论管理：表格 + 审核/垃圾操作 |
| `/admin/media` | `auth` | 媒体库：网格缩略图 + 上传 + 删除 |
| `/admin/tags` | `auth` | 标签 CRUD |
| `/admin/categories` | `auth` | 分类树 CRUD + 拖拽排序 |
| `/admin/settings` | `auth` | 站点配置表单（全部 15 个字段） |
| `/admin/settings/appearance` | `auth` | 背景图管理 + 歌单编辑 |
| `/admin/profile` | `auth` | 修改头像/简介 |

---

## 4. 状态管理 & 数据流

### 4.1 Pinia Store 设计

```
┌─────────────────────────────────────────────┐
│  authStore                                   │
│  ┌─────────────────────────────────────┐     │
│  │ state:  user, accessToken,          │     │
│  │         refreshToken, isLoggedIn    │     │
│  │ getters: isAdmin, userDisplayName   │     │
│  │ actions: login, register, logout,   │     │
│  │          refreshToken, fetchProfile │     │
│  └─────────────────────────────────────┘     │
│  → Token 写入 localStorage                   │
│  → 初始化时从 localStorage 恢复               │
│  → accessToken 过期自动 refresh              │
└─────────────────────────────────────────────┘

┌─────────────────────────────────────────────┐
│  themeStore                                  │
│  ┌─────────────────────────────────────┐     │
│  │ state:  mode ('light' | 'dark')     │     │
│  │ actions: toggle(), set(mode)         │     │
│  │ persist: localStorage                │     │
│  └─────────────────────────────────────┘     │
│  → 通过 <html class="dark"> 驱动 Tailwind   │
└─────────────────────────────────────────────┘

┌─────────────────────────────────────────────┐
│  toastStore                                  │
│  ┌─────────────────────────────────────┐     │
│  │ state:  toasts[]                     │     │
│  │ actions: success(msg), error(msg),   │     │
│  │          info(msg), dismiss(id)      │     │
│  └─────────────────────────────────────┘     │
│  → AppToast 组件监听 store 自动渲染          │
└─────────────────────────────────────────────┘
```

### 4.2 Auth 流程

```
┌──────────┐     POST /api/v1/auth/login     ┌──────────┐
│  Browser  │ ───────────────────────────────> │ Gateway   │
│           │ <── { user, access_token,       │           │
│           │       refresh_token, expires }  │ /auth     │
└────┬─────┘                                  └──────────┘
     │
     │  authStore.login() 保存:
     │  ├── localStorage.set('ley_access_token', ...)
     │  ├── localStorage.set('ley_refresh_token', ...)
     │  └── state.user = user
     │
     ▼
  所有 API 请求自动附加（composables/useAuth.ts）:
  Authorization: Bearer <accessToken>
```

**Token 刷新策略**（`composables/useAuth.ts`）：

```typescript
// $fetch 全局拦截器 (plugins/api.ts)
export default defineNuxtPlugin(() => {
  const auth = useAuthStore()
  const config = useRuntimeConfig()

  globalThis.$fetch = $fetch.create({
    baseURL: config.public.apiBaseUrl,
    onRequest({ options }) {
      if (auth.accessToken)
        options.headers.set('Authorization', `Bearer ${auth.accessToken}`)
    },
    onResponseError({ response }) {
      if (response.status === 401 && auth.refreshToken) {
        return auth.refresh().then(() => {
          // 重试原请求
        }).catch(() => {
          auth.logout()
          navigateTo('/auth/login')
        })
      }
    },
  })
})
```

### 4.3 服务端数据获取（`useFetch` / `useAsyncData`）

```vue
<script setup lang="ts">
// 首页 — ISR 预取 + 客户端 hydration
const { data: site } = await useFetch('/api/v1/site/config', {
  key: 'site-config',        // 跨页面缓存复用
  transform: (res: SiteConfigReply) => res.config,
})

const { data: articles } = await useFetch('/api/v1/articles', {
  query: { status: 'published', page: 1, page_size: 10 },
  key: 'home-articles',
})

// 客户端分页（不触发 SSR）
const page = ref(1)
const { data, refresh } = await useFetch('/api/v1/articles', {
  query: computed(() => ({ page: page.value, page_size: 10 })),
  lazy: true,                // 不阻塞导航
  watch: [page],             // 自动跟踪 page ref
})
</script>
```

**缓存策略总结**：

| 数据 | Key | TTL | 策略 |
|------|-----|-----|------|
| 站点配置 | `site-config` | 60s | `useFetch` + `swr: 60` route rule |
| 文章列表（首页） | `home-articles` | 60s | 同上 |
| 文章详情 | `article-${slug}` | 300s | 同上 |
| 标签列表 | `tags-all` | 300s | ISR |
| 分类树 | `categories-all` | 300s | ISR |
| 评论列表 | — | 客户端 only | `useFetch` + `lazy: true` |
| 管理后台列表 | — | 纯客户端 | `useFetch` + `lazy: true` + `ssr: false` |

### 4.4 表单状态（VeeValidate + Zod）

```vue
<script setup lang="ts">
import { useForm } from 'vee-validate'
import { z } from 'zod'

const articleSchema = z.object({
  title: z.string().min(2, '标题至少 2 个字符').max(200),
  content: z.string().min(1, '正文不能为空'),
  excerpt: z.string().max(500).optional(),
  coverImage: z.string().url('请输入有效 URL').optional(),
  categoryId: z.number().positive().nullable(),
  tagNames: z.array(z.string()).default([]),
})

const { handleSubmit, errors, isSubmitting } = useForm({
  validationSchema: toTypedSchema(articleSchema),
})

const onSubmit = handleSubmit(async (values) => {
  await $api.articles.create(values)
  toast.success('文章创建成功')
  await navigateTo('/admin/articles')
})
</script>

<template>
  <form @submit="onSubmit">
    <AppInput name="title" label="标题" />
    <p v-if="errors.title" class="text-red-500">{{ errors.title }}</p>
    <!-- ... -->
  </form>
</template>
```

---

## 5. API 客户端层设计

### 5.1 基础封装 (`lib/api/client.ts`)

```typescript
// lib/api/client.ts
import type { ApiError } from '~/lib/types/common'

const API_BASE = process.env.NUXT_PUBLIC_API_BASE_URL || '/api/v1'

export function createApiClient() {
  const auth = useAuthStore()

  return {
    get<T>(path: string, query?: Record<string, any>): Promise<T> {
      return $fetch<T>(path, {
        baseURL: API_BASE,
        method: 'GET',
        query,
        onResponseError: handleError,
      })
    },
    post<T>(path: string, body?: unknown): Promise<T> {
      return $fetch<T>(path, {
        baseURL: API_BASE,
        method: 'POST',
        body,
        onResponseError: handleError,
      })
    },
    put<T>(path: string, body?: unknown): Promise<T> {
      return $fetch<T>(path, {
        baseURL: API_BASE,
        method: 'PUT',
        body,
        onResponseError: handleError,
      })
    },
    delete<T>(path: string): Promise<T> {
      return $fetch<T>(path, {
        baseURL: API_BASE,
        method: 'DELETE',
        onResponseError: handleError,
      })
    },
    upload<T>(path: string, file: File): Promise<T> {
      const form = new FormData()
      form.append('file', file)
      return $fetch<T>(path, {
        baseURL: API_BASE,
        method: 'POST',
        body: form,
        onResponseError: handleError,
      })
    },
  }
}

function handleError({ response }: { response: any }) {
  const err: ApiError = response._data ?? { code: response.status, message: '未知错误', reason: '' }
  if (response.status === 429) {
    // 限流：显示 Toast 提示
    const toast = useToastStore()
    toast.error('请求过于频繁，请稍后再试')
  }
  throw err
}

// 全局注入（plugins/api.ts）
export default defineNuxtPlugin(() => {
  return { provide: { api: createApiClient() } }
})
```

使用方式：
```vue
<script setup lang="ts">
const { $api } = useNuxtApp()
const data = await $api.get('/articles', { page: 1 })
</script>
```

### 5.2 类型定义与 Proto 对齐

```typescript
// lib/types/article.ts
export interface ArticleInfo {
  id: number
  title: string
  slug: string
  content: string            // Markdown 原文
  excerpt: string
  coverImage: string
  status: 'draft' | 'published' | 'archived'
  authorId: number
  author: AuthorInfo
  categoryId: number
  category: CategoryInfo
  tags: TagInfo[]
  viewCount: number
  likeCount: number
  commentCount: number
  isTop: boolean
  isLiked: boolean
  publishedAt: string        // ISO 8601
  createdAt: string
  updatedAt: string
}

export interface CommentInfo {
  id: number
  articleId: number
  authorId: number
  authorName: string
  authorAvatar: string
  parentId: number
  depth: number
  content: string
  status: string
  createdAt: string
  updatedAt: string
}

// 递归类型（评论树）
export interface CommentNode {
  comment: CommentInfo
  children: CommentNode[]
}

export interface ListArticlesRequest {
  status?: string            // 'draft' | 'published' | 'archived'
  categoryId?: number
  tags?: string[]
  authorId?: number
  sortBy?: string            // 'created_at' | 'view_count' | 'published_at'
  sortOrder?: 'asc' | 'desc'
  page?: number
  pageSize?: number
}

// ... 与 Proto 完整对齐（TagInfo / CategoryInfo / FileInfo / SiteConfig / ...）
```

---

## 6. 页面设计

### 6.1 公开首页 `/`

```
┌─────────────────────────────────────────────────────────┐
│  [Logo]   首页   文章   标签   🔍[搜索框]    [登录]       │  ← PublicHeader
├─────────────────────────────────────────────────────────┤
│                                                         │
│                   站点标题 (SiteTitle)                    │
│                   站点副标题 (SiteSubtitle)                │
│               [背景大图 Banner / 活跃背景图]               │
│                                                         │
├─────────────────────────────────────────────────────────┤
│                                                         │
│  最新文章（分页，每页 10 条）                              │
│  ┌───────────────────────────────────────────────┐      │
│  │ ┌──────┐  文章标题                             │      │
│  │ │ 封面  │  摘要...（最多 3 行截断）              │      │
│  │ │      │  📅 2026-05-13   🏷 Vue   🏷 TypeScript│      │
│  │ └──────┘  👁 1.2k  ❤️ 42  💬 15                │      │
│  └───────────────────────────────────────────────┘      │
│  ┌───────────────────────────────────────────────┐      │
│  │ ...（重复）                                     │      │
│  └───────────────────────────────────────────────┘      │
│                                                         │
│       [< 上一页]    [1]  [2]  [3] ... [10]    [下一页 >] │
│                                                         │
├─────────────────────────────────────────────────────────┤
│  Footer: © 2026  Ley Blog  |  京ICP备xxxxxx             │
│  GitHub   Twitter   Email                               │
│  友情链接                                                │
└─────────────────────────────────────────────────────────┘
```

**数据依赖**：
- `useFetch('/api/v1/site/config')` → 标题/副标题/Logo/Footer 文案
- `useFetch('/api/v1/site/backgrounds')` → 活跃背景图
- `useFetch('/api/v1/articles', { query: { status: 'published', page, page_size: 10 } })`

### 6.2 文章详情 `/articles/[slug]`

```
┌─────────────────────────────────────────────────────────┐
│  PublicHeader                                           │
├──────────────┬──────────────────────────────────────────┤
│  📑 目录      │  文章标题                                │
│  (sticky)     │                                         │
│               │  👤 作者名  ·  📅 2026-05-13             │
│  一、引言      │  🏷 Vue  TypeScript  TailwindCSS        │
│  二、正文      │  ──────────────────────────────────     │
│  2.1 概念     │                                         │
│  2.2 实现     │  ## 一、引言                             │
│  三、总结      │  ...Markdown 渲染内容...                 │
│               │                                         │
│  ◆ 当前位置    │  ## 二、正文                             │
│               │  ...                                    │
│               │                                         │
│               │  ──────────────────────────────────     │
│               │  ❤️ 点赞 42    📤 分享                  │
│               ├──────────────────────────────────────────┤
│               │  评论区                                   │
│               │  ┌──────────────────────────────────┐    │
│               │  │ [textarea]         [发送评论]    │    │
│               │  └──────────────────────────────────┘    │
│               │                                          │
│               │  💬 alice  2 天前                        │
│               │     文章写得真好！                        │
│               │     💬 bob  1 天前（回复 alice）          │
│               │         +1，学习了！                      │
│               │         💬 alice（回复 bob）              │
│               │             谢谢！                        │
│               │             [已达到最大嵌套深度，不能再回复]│
│               │     [回复 alice]                         │
│               │                                          │
└──────────────┴──────────────────────────────────────────┘
```

**关键交互**：
- **目录生成**：客户端从 `article.content` 提取 `##` 标题，渲染为 sticky sidebar
- **点赞**：`<LikeButton>` 乐观更新 `likeCount ± 1` + `isLiked` toggle
- **评论树**：`<CommentList>` → `<CommentItem>` 递归，深度 >= 5 禁用回复按钮
- **分享**：`navigator.clipboard` 复制链接，或 Web Share API

### 6.3 管理后台 Dashboard `/admin`

```
┌──────────────────────────────────────────────────────────┐
│ ┌──────────┐  AdminTopbar [博客名]  [👤 admin ▼]         │
│ │ 📊 概览    │────────────────────────────────────────────┤
│ │ 文章管理   │                                            │
│ │ 评论管理   │  统计卡片                                    │
│ │ 媒体库    │  ┌──────────┐ ┌──────────┐ ┌──────────┐   │
│ │ 标签管理   │  │ 总文章   │ │ 总评论   │ │ 草稿     │   │
│ │ 分类管理   │  │   128    │ │   356    │ │   12     │   │
│ │            │  └──────────┘ └──────────┘ └──────────┘   │
│ │ ⚙ 站点配置  │                                            │
│ │ 🎨 外观设置  │  最近文章                                    │
│ │ 👤 个人资料  │  ┌───────────────────────────────────────┐ │
│ │            │  │ 标题          状态    日期     操作    │ │
│ │            │  │ 搭建个人博客...  已发布  05-13  编辑   │ │
│ │            │  │ Vue3 实战...    已发布  05-10  编辑   │ │
│ │            │  │ 项目规划...      草稿   05-08  编辑   │ │
│ └────────────┘  └───────────────────────────────────────┘ │
└──────────────────────────────────────────────────────────┘
```

### 6.4 文章编辑器 `/admin/articles/new`

```
┌──────────────────────────────────────────────────────────┐
│  AdminTopbar            [保存草稿]  [发布文章]              │
├─────────────────┬────────────────────────────────────────┤
│ AdminSidebar    │  ┌───────────────────────────────────┐ │
│                 │  │ 标题：[___________________________]│ │
│                 │  └───────────────────────────────────┘ │
│                 │                                        │
│                 │  ┌─────────────┐  ┌────────────────┐  │
│                 │  │ 编辑区 (70%)│  │  元信息面板(30%) │  │
│                 │  │             │  │                │  │
│                 │  │ [B][I][H1]  │  │ 📝 状态        │  │
│                 │  │ [H2][link]  │  │  [草稿 ▼]      │  │
│                 │  │ [code][img] │  │                │  │
│                 │  │ [quote]     │  │ 📂 分类        │  │
│                 │  │ ─────────── │  │  [前端技术 ▼]  │  │
│                 │  │             │  │                │  │
│                 │  │ # 标题      │  │ 🏷 标签        │  │
│                 │  │             │  │  [Vue ×]        │  │
│                 │  │ 正文内容...  │  │  [TS ×]         │  │
│                 │  │             │  │  [+ 添加标签]   │  │
│                 │  │             │  │                │  │
│                 │  │             │  │ 📝 摘要        │  │
│                 │  │             │  │ [textarea...]  │  │
│                 │  │             │  │                │  │
│                 │  │             │  │ 🖼 封面图       │  │
│                 │  │             │  │ [上传图片]      │  │
│                 │  │             │  │                │  │
│                 │  │             │  │ 📌 置顶        │  │
│                 │  │             │  │ [开关]         │  │
│                 │  └─────────────┘  └────────────────┘  │
└─────────────────┴────────────────────────────────────────┘
```

**编辑器核心流程**：
1. TipTap 编辑器 → 内部 HTML 状态 → 提交时 `editor.getHTML()` 获取
2. 图片粘贴/拖拽 → `ImageUploader` → `$api.files.upload(file)` → 返回 URL → 插入 `![alt](url)`
3. Markdown 快捷键：`#` + `Space` → H1，`**粗体**` 自动转换

### 6.5 站点配置 `/admin/settings`

```
┌──────────────────────────────────────────────────────────┐
│ AdminSidebar    │  站点配置                                │
│                 │────────────────────────────────────────┤
│                 │                                        │
│                 │  ── 基本信息 ──                          │
│                 │  站点标题      [____________________]   │
│                 │  副标题         [____________________]   │
│                 │  站点描述       [____________________]   │
│                 │  Logo URL      [____________________]   │
│                 │  Favicon URL   [____________________]   │
│                 │                                        │
│                 │  ── SEO 设置 ──                          │
│                 │  SEO 关键词     [____________________]   │
│                 │  SEO 描述       [____________________]   │
│                 │                                        │
│                 │  ── 社交媒体 ──                          │
│                 │  GitHub         [____________________]   │
│                 │  Twitter        [____________________]   │
│                 │  Email          [____________________]   │
│                 │                                        │
│                 │  ── 页脚 ──                              │
│                 │  版权文案       [____________________]   │
│                 │  ICP 备案号     [____________________]   │
│                 │                                        │
│                 │  ── 功能开关 ──                          │
│                 │  [✓] 全站评论   [✓] 全站点赞             │
│                 │  [✓] 评论无需审核                        │
│                 │                                        │
│                 │                [保存配置]                │
└─────────────────┴────────────────────────────────────────┘
```

---

## 7. 组件树 & 数据流

### 7.1 文章详情页完整组件树

```
pages/articles/[slug].vue              ← setup: useFetch(article)
│
├─ PublicHeader.vue                    ← useFetch(site-config) → 站点名/导航
│
├─ <main class="grid grid-cols-[240px_1fr]">
│  ├─ ArticleTOC.vue                   ← computed: 从 article.content 解析标题
│  │   └─ 点击跳转 → document.querySelector(href)
│  │
│  └─ <article>
│     ├─ <h1>{{ article.title }}</h1>
│     ├─ ArticleMeta.vue               ← props: author / date / tags / stats
│     │   ├─ TagBadge.vue × N
│     │   └─ LikeButton.vue            ← emit: like → $api.articles.like(id)
│     │
│     ├─ ArticleContent.vue            ← <MDC :value="article.content" />
│     │                                  (服务端渲染 Markdown → HTML)
│     │
│     ├─ <hr />
│     └─ <section aria-label="评论">
│        ├─ CommentForm.vue            ← 顶层评论（parentId = 0）
│        └─ CommentList.vue            ← props: nodes (CommentNode[])
│           └─ CommentItem.vue × N     ← 递归组件（自身引用）
│              ├─ <p>{{ comment.content }}</p>
│              ├─ CommentForm.vue       ← 回复表单（v-if depth < 5）
│              └─ CommentList.vue       ← 递归 children
│     </section>
│  </article>
│  </main>
│
└─ PublicFooter.vue                    ← useFetch(site-config) → 备案号/社交链接
```

### 7.2 状态流转

```
用户操作
   │
   ├── GET 请求 → useFetch / useAsyncData → SSR 预取 + 客户端 hydration
   │                │
   │                ├── Nitro 缓存命中 → 直接返回（ISR/SWR）
   │                └── 缓存未命中 → $fetch → Gateway → Blog/Auth 服务
   │
   └── 写操作 → $api.xxx.post() → 乐观更新（可选）
                    │
                    ├── Pinia Store 立即 mutation（乐观）
                    ├── $fetch → Gateway
                    ├── 成功 → 无需操作（或 refresh 列表）
                    └── 失败 → Pinia Store 回滚 + Toast 错误提示
```

---

## 8. SEO & 性能优化

### 8.1 SEO

| 手段 | 实现 |
|------|------|
| `<title>` / `<meta description>` | `useHead()` / `useSeoMeta()` 在每页 `setup` 中动态设置 |
| Open Graph | `defineOgImage()` 动态生成封面图（字体/颜色/标题叠加），`og:image` 配置 |
| Twitter Card | `twitter:card: 'summary_large_image'` |
| JSON-LD 结构化数据 | `<script type="application/ld+json">` 注入 Article / BreadcrumbList Schema |
| Sitemap | `server/routes/sitemap.xml.ts` 动态生成（<lastmod> 自动填充） |
| RSS Feed | `server/routes/rss.xml.ts` 最新 20 篇 |
| Canonical URL | `<link rel="canonical">` 统一 `/articles/[slug]` |
| 语义化 HTML | `<article>` / `<time>` / `<nav>` / `<header>` / `<footer>` |

### 8.2 性能

| 手段 | 实现 |
|------|------|
| ISR / SWR | `routeRules` 配置每页缓存策略，Nitro 自动处理 |
| 图片优化 | Nuxt Image 模块（`@nuxt/image`）：自动 WebP/AVIF 转换 + 响应式尺寸 |
| 代码分割 | 管理后台（admin/）组件惰性加载 `defineAsyncComponent` |
| 字体优化 | `@nuxt/fonts` 或手写 `@font-face` + `font-display: swap` |
| 首屏最小化 | 文章列表首屏仅 5 条 SSR，余下客户端 `lazy: true` |
| Bundle 分析 | `nuxt build --analyze` 查看各 chunk 大小 |
| CSS 优化 | TailwindCSS JIT 模式按需生成；nuxt-purgecss 移除未使用样式 |

---

## 9. 认证 & 权限

### 9.1 Token 存储

```
AccessToken:   localStorage.getItem('ley_access_token')   — 15min 有效期
RefreshToken:  localStorage.getItem('ley_refresh_token')  — 7d 有效期
UserInfo:      authStore.user                             — 内存（不持久化）
```

### 9.2 路由守卫

```typescript
// middleware/admin.ts
export default defineNuxtRouteMiddleware((to) => {
  const auth = useAuthStore()

  // 1. 未登录
  if (!auth.accessToken) {
    return navigateTo('/auth/login')
  }

  // 2. AccessToken 即将过期（剩余 < 2min）
  const exp = jwtDecode(auth.accessToken).exp
  if (exp && (exp - Date.now() / 1000) < 120) {
    auth.refreshToken()
  }

  // 3. 非管理员角色
  if (auth.user?.role !== 'admin') {
    return navigateTo('/')  // 跳转首页
  }
})
```

页面应用中间件：
```vue
<script setup lang="ts">
// pages/admin/index.vue
definePageMeta({ middleware: 'admin' })   // Nuxt 自动执行 middleware/admin.ts
</script>
```

全局认证中间件（自动注入 Token）：
```typescript
// middleware/auth.global.ts
export default defineNuxtRouteMiddleware(() => {
  const auth = useAuthStore()
  auth.restoreFromStorage()  // 页面刷新后恢复 Token
})
```

### 9.3 API 权限模型

```
公开接口（无需 Token）:
  GET  /api/v1/articles / articles/{slug}
  GET  /api/v1/articles/{id}/comments
  GET  /api/v1/tags / categories
  GET  /api/v1/site/config / site/backgrounds / site/music/playlist

需认证（Bearer Token）:
  POST /api/v1/articles/{id}/comments     ← 发表评论
  POST /api/v1/articles/{id}/like         ← 点赞
  DELETE /api/v1/articles/{id}/like       ← 取消点赞
  GET  /api/v1/users/me                    ← 获取个人资料
  PUT  /api/v1/users/me                    ← 更新个人资料

需管理员角色（在管理后台使用）:
  全部 CRUD:   articles / comments / tags / categories / files
  站点管理:     site/config / site/backgrounds / site/music/playlist
```

---

## 10. 部署方案

### 10.1 部署架构总览

```
                         ┌─────────────┐
                         │   CDN / DNS  │
                         └──────┬──────┘
                                │
                         ┌──────▼──────┐
                         │   Nginx      │  ← TLS 终结 / 静态资源缓存 / 反向代理
                         │   :80/:443   │
                         └──┬──────┬───┘
                  / (前端)  │      │  /api/* (后端)
                            │      │
              ┌─────────────▼┐  ┌──▼──────────────┐
              │  Nuxt Nitro  │  │  Gateway :8000    │
              │  :3000       │  │                   │
              │  (web 容器)   │  │  路由 → Auth/Blog │
              └──────────────┘  └───────────────────┘
```

**核心原则**：
- **Nginx 统一入口** — TLS/SSL 全部在 Nginx 终止，内部走 HTTP；`/` 转发到 Nitro Server，`/api/` 转发到 Gateway
- **前后端分离部署** — 各自独立容器，Nginx 按路径分发，互不影响
- **可降级** — 静态资源可推 CDN；后端不可用时前端仍可展示已缓存页面

### 10.2 三种部署方案

| 方案 | 适用场景 | 服务器需求 |
|------|----------|------------|
| **A. Docker Compose 一体化** | 单机部署 / 开发验证 | 1 台 VPS |
| **B. Nginx 反向代理** | 生产环境 | 1 台 VPS + 域名 |
| **C. 静态导出 + CDN** | 极致性能 / 无服务端渲染需求 | CDN（管理后台需额外处理） |

---

### 10.3 方案 A：Docker Compose 一体化

适合本地开发或单机全栈部署。后端服务使用 `network_mode: host`，web 容器通过 `extra_hosts` 访问宿主机的 Gateway。

**Dockerfile**：

```dockerfile
# ================================
# 构建阶段
# ================================
FROM node:20-alpine AS builder
WORKDIR /app
RUN corepack enable

# 依赖缓存层（利用 Docker 层缓存）
COPY package.json pnpm-lock.yaml ./
RUN pnpm install --frozen-lockfile

# 构建
COPY . .
ARG NUXT_PUBLIC_API_BASE_URL
ENV NUXT_PUBLIC_API_BASE_URL=$NUXT_PUBLIC_API_BASE_URL
RUN pnpm build                      # Nuxt Build → .output/

# ================================
# 运行阶段
# ================================
FROM node:20-alpine AS runner
WORKDIR /app
ENV NODE_ENV=production
ENV HOST=0.0.0.0
ENV PORT=3000

COPY --from=builder /app/.output ./.output
EXPOSE 3000

# 健康检查
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD wget --no-verbose --tries=1 --spider http://localhost:3000/healthz || exit 1

CMD ["node", ".output/server/index.mjs"]
```

**docker-compose.yml 追加**（项目根已有后端服务定义）：

```yaml
  # ================================
  # Web — 前端服务 :3000
  # ================================
  web:
    build:
      context: ./web
      args:
        - NUXT_PUBLIC_API_BASE_URL=http://localhost:8000    # Gateway 端口
    image: ley-web:latest
    container_name: ley-web
    network_mode: host            # 与后端共享 host 网络，可直接 localhost 通信
    volumes:
      - ./data:/app/data
    environment:
      - TZ=Asia/Shanghai
    restart: unless-stopped
    deploy:
      resources:
        limits:
          cpus: '1.00'
          memory: '512M'
```

**注意**：`network_mode: host` 下容器端口直接绑定宿主机，`ports` 映射无效。`NUXT_PUBLIC_API_BASE_URL` 必须以浏览器能访问到的地址为准（本地用 `localhost:8000`，生产用实际域名）。

**.dockerignore**（减少构建上下文）：

```
node_modules
.output
dist
.env
.env.*
!.env.example
*.md
.git
.cache
```

---

### 10.4 方案 B：Nginx 反向代理（推荐生产方案）

Nginx 作为统一入口，提供 TLS 终止、静态资源缓存、安全头、请求限速、日志等生产级能力。

**完整 nginx.conf**：

```nginx
# ================================
# Ley Blog — Nginx 生产配置
# ================================

# 限流区域定义（10 req/s，突发 20）
limit_req_zone $binary_remote_addr zone=api_limit:10m rate=10r/s;
limit_req_zone $binary_remote_addr zone=global_limit:10m rate=30r/s;

upstream web_upstream {
    server 127.0.0.1:3000;
    keepalive 64;
}

upstream gateway_upstream {
    server 127.0.0.1:8000;
    keepalive 64;
}

# HTTP → HTTPS 重定向
server {
    listen 80;
    server_name blog.example.com;
    return 301 https://$host$request_uri;
}

server {
    listen 443 ssl http2;
    server_name blog.example.com;

    # ============================
    # TLS
    # ============================
    ssl_certificate     /etc/nginx/certs/fullchain.pem;
    ssl_certificate_key /etc/nginx/certs/privkey.pem;
    ssl_protocols       TLSv1.2 TLSv1.3;
    ssl_ciphers         ECDHE-ECDSA-AES128-GCM-SHA256:ECDHE-RSA-AES128-GCM-SHA256;
    ssl_session_cache   shared:SSL:10m;
    ssl_session_timeout 10m;

    # ============================
    # 安全头
    # ============================
    add_header X-Content-Type-Options    "nosniff"       always;
    add_header X-Frame-Options           "DENY"           always;
    add_header X-XSS-Protection          "1; mode=block"  always;
    add_header Referrer-Policy           "strict-origin"  always;
    add_header Permissions-Policy        "camera=(), microphone=(), geolocation=()" always;

    # ============================
    # 日志
    # ============================
    access_log /var/log/nginx/ley_access.log combined buffer=16k;
    error_log  /var/log/nginx/ley_error.log  warn;

    # ============================
    # 前端静态资源（长缓存 + Gzip）
    # ============================
    location /_nuxt/ {
        proxy_pass http://web_upstream;
        proxy_set_header Host $host;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_cache_valid 200 30d;
        add_header Cache-Control "public, max-age=2592000, immutable";
    }

    location /images/ {
        proxy_pass http://web_upstream;
        proxy_set_header Host $host;
        proxy_cache_valid 200 7d;
        add_header Cache-Control "public, max-age=604800";
    }

    # ============================
    # SSR 页面（前端 Nitro Server）
    # ============================
    location / {
        limit_req zone=global_limit burst=30 nodelay;
        proxy_pass http://web_upstream;
        proxy_http_version 1.1;
        proxy_set_header Upgrade           $http_upgrade;
        proxy_set_header Connection        "upgrade";
        proxy_set_header Host              $host;
        proxy_set_header X-Real-IP         $remote_addr;
        proxy_set_header X-Forwarded-For   $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_read_timeout 30s;
    }

    # ============================
    # API 反向代理到 Gateway
    # ============================
    location /api/ {
        limit_req zone=api_limit burst=20 nodelay;
        proxy_pass http://gateway_upstream;
        proxy_http_version 1.1;
        proxy_set_header Host              $host;
        proxy_set_header X-Real-IP         $remote_addr;
        proxy_set_header X-Forwarded-For   $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_read_timeout 60s;
        client_max_body_size 10m;          # 允许文件上传
    }

    # ============================
    # 健康检查端点（不记录日志）
    # ============================
    location /healthz {
        proxy_pass http://web_upstream;
        access_log off;
    }
}
```

**SSL 证书**（Let's Encrypt 自动续期）：

```bash
# certbot 自动化
sudo certbot certonly --webroot \
  -w /var/www/certbot \
  -d blog.example.com \
  --email admin@example.com \
  --agree-tos --non-interactive

# crontab 自动续期
# 0 3 * * * certbot renew --quiet && nginx -s reload
```

---

### 10.5 方案 C：静态导出 + CDN

对纯静态公开页面（舍弃 SSR/ISR，管理后台另行部署），可将 Nuxt 预渲染为纯静态文件推送到 CDN。

```bash
# nuxt.config.ts
export default defineNuxtConfig({
  nitro: {
    prerender: {
      crawlLinks: true,        # 追踪所有内部链接，自动发现需预渲染的页面
      routes: [
        '/',
        '/articles',
        '/tags',
        // 动态路由需要手动列出或通过 API 自动扫描
      ],
    },
  },
})
```

```bash
pnpm generate                  # nuxi generate → .output/public/
# 产物为纯 HTML/CSS/JS，可直接部署到：
# - Nginx/Alpine (Docker)
# - GitHub Pages
# - 阿里云 OSS + CDN
# - Vercel / Netlify（零配置）
```

**限制**：静态导出后 `/admin/*` 管理后台需要额外部署 SPA 版本（单独构建入口），或另起一个 BFF 服务。

---

### 10.6 环境变量管理

| 变量 | 开发 | 生产 | 说明 |
|------|------|------|------|
| `NUXT_PUBLIC_API_BASE_URL` | `http://localhost:8000` | `https://blog.example.com` | 浏览器端 API 请求基址 |
| `NITRO_HOST` | `0.0.0.0` | `0.0.0.0` | Nitro 监听地址 |
| `NITRO_PORT` | `3000` | `3000` | Nitro 监听端口 |

**`.env` 不提交**（`.gitignore`）；`.env.example` 提供模板。

Nuxt 4 的 `NUXT_PUBLIC_*` 前缀变量会**编译期内联到客户端 JS**，因此**必须在构建时确定**（不同环境需分别构建镜像）。生产构建命令：

```bash
docker build \
  --build-arg NUXT_PUBLIC_API_BASE_URL=https://blog.example.com \
  -t ley-web:latest \
  ./web
```

---

### 10.7 CI/CD（GitHub Actions 示例）

```yaml
name: Deploy Web

on:
  push:
    branches: [main]
    paths: ['web/**']

jobs:
  build-and-deploy:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - uses: docker/setup-buildx-action@v3

      - name: Build image
        run: |
          docker build \
            --build-arg NUXT_PUBLIC_API_BASE_URL=https://blog.example.com \
            -t ley-web:${{ github.sha }} \
            ./web

      - name: Deploy to server
        uses: appleboy/ssh-action@v1
        with:
          host: ${{ secrets.SSH_HOST }}
          username: ${{ secrets.SSH_USER }}
          key: ${{ secrets.SSH_KEY }}
          script: |
            cd /opt/ley
            docker compose pull web
            docker compose up -d --no-deps web
            docker image prune -f
```

---

## 11. 实施计划

### Phase 1 — 基础框架 + 公开页面（第 1-2 周）

- [ ] Nuxt 4 项目初始化：TailwindCSS / TypeScript / 目录结构
- [ ] API 客户端封装 + 全局插件注入（`$api`）
- [ ] 类型定义（`lib/types/`）与 Proto 对齐
- [ ] Pinia 认证 Store + 登录/注册页
- [ ] 公开布局（`default.vue`：Header + Footer）
- [ ] 首页文章列表（ISR + 分页）
- [ ] 文章详情页 + Markdown 渲染（nuxt-mdc）
- [ ] 标签云 + 分类列表页
- [ ] 搜索页（防抖 + CSR）

### Phase 2 — 管理后台（第 3-4 周）

- [ ] Admin 布局（`admin.vue`：Sidebar + Topbar）
- [ ] `admin` 路由中间件（JWT 校验 + 角色检查）
- [ ] 文章编辑器（TipTap Vue + ImageUploader）
- [ ] 文章管理列表（筛选 + 表格 + 批量操作）
- [ ] 媒体库（拖拽上传 + 缩略图网格 + 删除确认）
- [ ] 标签管理 CRUD
- [ ] 分类树管理 CRUD
- [ ] 站点配置编辑表单

### Phase 3 — 交互增强（第 5-6 周）

- [ ] 评论系统（发布 / 树形渲染 / 嵌套回复 / 深度限制）
- [ ] 评论管理（审核 / 垃圾标记）
- [ ] 点赞功能（乐观更新）
- [ ] 背景图管理（上传 / 激活 / 删除）
- [ ] 歌单编辑（添加 / 删除 / 排序）
- [ ] SEO 完善（useSeoMeta / OG / sitemap / RSS / JSON-LD）

### Phase 4 — 打磨上线（第 7-8 周）

- [ ] 暗色模式（Pinia themeStore + `class="dark"`）
- [ ] 移动端响应式（hamburger menu / 侧边栏折叠 / 表格横滚）
- [ ] E2E 测试（Playwright：注册→登录→发表文章→评论→点赞）
- [ ] 性能优化（Lighthouse 评分 > 90）
- [ ] Dockerfile + Docker Compose 集成联调
- [ ] Nginx 配置 + HTTPS / CDN

---

## 12. 关键技术决策说明

### 12.1 为何选择 Nuxt 4 而非 SPA（Vite + Vue Router）

- 博客 SEO 是核心需求，SSR/SSG 让搜索引擎能直接抓取文章内容
- Nuxt 的 `routeRules` 可按页面粒度混合 SSG / ISR / SSR / CSR，公开页静态生成，管理后台纯 CSR
- Nitro 服务端引擎内置缓存层（SWR），无需额外配置 Redis/varnish
- 文件路由 + 自动导入让代码量减少 30%+

### 12.2 为何不使用 gRPC-Web

- 后端 Kratos 已通过 `google.api.http` 注解暴露标准 REST API，直接 `fetch` 即可
- gRPC-Web 需要 Envoy 代理层，增加部署复杂度和延迟
- 博客场景请求频率低（个人博客 PV 量级），Protobuf 二进制序列化优势不显著
- `openapi.yaml` 已由 protoc 生成，可直接用于 API 文档和类型参考

### 12.3 Rich Text 方案选择

- **TipTap** 是唯一对 Vue 3 有第一方支持的 ProseMirror 封装，扩展生态成熟
- 文章存储格式为 Markdown（后端不变），编辑器内部使用 HTML 编辑，提交时 HTML → Markdown 转换
- Markdown 渲染使用 Nuxt 的 `@nuxtjs/mdc`（MDC 语法增强），服务端渲染，零客户端 JS
- 编辑器仅用于管理后台（admin），可作为异步 chunk 延迟加载

### 12.4 评论嵌套深度限制

- 后端 `Comment.depth` 字段硬限制最大 5 层
- 前端 `CommentForm` 在 `depth >= 5` 时隐藏回复按钮，替换为提示文字
- 递归渲染 `CommentItem` → `CommentList` 通过 `<slot>` 实现无限递归，每层缩进 `padding-left: 1.5rem`
- 移动端评论列表使用手风琴折叠（默认展开前 3 层）

### 12.5 状态管理边界

- **Pinia Store**：仅管理全局状态（auth / theme / toast），跨页面共享
- **组件内部 ref**：局部状态（表单数据 / 分页 / 筛选条件）存放在组件内
- **服务端数据**：`useFetch` / `useAsyncData` 自动管理请求状态和缓存，不放入 Store
- **路由参数**：`useRoute().params` / `useRoute().query` 驱动数据查询

---

> 本设计方案覆盖 Vue 3 / Nuxt 4 全栈技术选型、项目结构、路由、状态管理（Pinia）、数据流（useFetch）、页面布局、组件树、认证权限（middleware）、构建部署（Docker + Nginx）和实施计划（4 个 Phase / 8 周）。设计对齐后端 Proto API（6 个 Service / 35 个 RPC），具体实现细节待各阶段开发时细化。
