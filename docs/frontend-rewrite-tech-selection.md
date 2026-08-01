# 前端重构 — React SPA 技术选型

> 记录日期：2026-07-03
> 重构目标：从 Nuxt 4 (Vue 3) 全栈重写为 React 纯 SPA，视觉风格从侘寂和風改为现代极简。
> 视觉参考：[Cloudreve](https://github.com/cloudreve/Cloudreve) 前端风格 — MUI v6 + 灰底白卡 + 12px 统一圆角 + 扁平无阴影按钮 + 自定义滚动条。

---

## 核心决策

| 决策 | 选择 | 被排除的选项 |
|------|------|------------|
| **SSR vs SPA** | **纯 SPA** (Vite) | Next.js (SPA 模式下运行时开销大，与框架设计相悖) |
| **路由** | **React Router v7** (library mode) | Next.js App Router、Remix、TanStack Router |
| **数据获取** | **TanStack Query v5** | SWR、RTK Query、手写 fetch |
| **状态管理** | **Zustand** | Redux Toolkit、Jotai、MobX |
| **样式方案** | **TailwindCSS v4** | CSS Modules、Styled Components、Panda CSS |

---

## 技术栈全表

| 类别 | 选型 | 版本 | 对标当前 (Nuxt/Vue) | 选择理由 |
|------|------|------|---------------------|----------|
| **构建工具** | Vite | ^6 | Nuxt CLI | 最快构建速度，React 官方推荐，零配置 JSX/TSX |
| **UI 框架** | React | ^19 | Vue 3 | 团队偏好，生态丰富 |
| **语言** | TypeScript | ^5.7 | 同 | 保持不变 |
| **路由** | React Router | ^7 (library) | Nuxt file-based routes | 声明式路由树，layout routes 天然实现多布局，loader 数据预取 |
| **状态管理** | Zustand | ^5 | Pinia | ~1KB、无 Provider 包裹、不可变更新、devtools 支持 |
| **服务端缓存** | TanStack Query | ^5 | 手写 store 缓存 | 缓存去重、后台刷新、乐观更新、分页、SSR 预取（未来扩展） |
| **HTTP 客户端** | ofetch | ^1 | $fetch (Nuxt 内置) | $fetch 的底层库，API 完全兼容 |
| **样式** | TailwindCSS | ^4 | Tailwind 3.4 | 仅更换设计 token |
| **组件变体** | CVA | ^0.7 | — | 类型安全的组件 variant 管理 (对标 Was 前缀的 variant prop) |
| **主题切换** | Zustand + CSS 变量 | — | @nuxtjs/color-mode | SPA 无 FOUC 问题，几行代码搞定 |
| **国际化** | react-i18next | ^15 | @nuxtjs/i18n | SPA 最成熟方案，命名空间分文件，ICU 支持 |
| **Markdown 渲染** | react-markdown | ^10 | @nuxtjs/mdc | 纯客户端渲染，配合 rehype/remark 插件 |
| **代码高亮** | Shiki | ^1 | Shiki (Nuxt 同款) | 双主题高亮，不变 |
| **动画** | framer-motion | ^12 | 手写 IntersectionObserver | 声明式 API，whileInView 替代 ScrollReveal，AnimatePresence 替代 PageTransition |
| **图标** | lucide-react | ^0.4 | lucide-vue-next | 图标集不变，换 React 绑定 |
| **表单** | react-hook-form + zod | ^7 + ^3 | — (当前无) | 管理后台大量表单需求，性能最优方案 |
| **Toast** | sonner | ^2 | vue-sonner | API 几乎相同 |
| **字体** | Geist + Geist Mono | — | Noto Serif SC + LXGW WenKai | Vercel 现代无衬线字体，从衬线转向无衬线 |
| **远程图片** | unpic | ^3 | @nuxt/image | 支持 MinIO 等任意远程源，无框架绑定 |
| **工具集** | @reactuses/core | ^3 | @vueuse/core | Composition API hooks |
| **部署** | `vite build` → Nginx 静态托管 | — | `pnpm generate` → rsync | `try_files $uri /index.html` |

---

## Store 拆分 — 从 6 个减到 3 个

TanStack Query 接管了服务端数据的缓存层，Zustand 只负责纯客户端状态。

| 当前 Pinia Store | 重构方案 | 说明 |
|---|---|---|
| `auth` (token + user) | **Zustand `authStore`** | Token 存 Cookie (`ley_at`/`ley_rt`)，Zustand 只存 `user` 引用 |
| `ui` (toast, loading, title) | **Zustand `uiStore`** | 不变 |
| `article` (list, detail, draft, search) | **Zustand `draftStore`** + **TanStack Query** | draft(本地) → Zustand；list/detail/search → TanStack Query |
| `tag` | **TanStack Query** | `useQuery(['tags'], fetchTags)` → 缓存 60min |
| `category` | **TanStack Query** | `useQuery(['categories'], fetchCategories)` → 缓存 60min |
| `site` (config, backgrounds, playlist) | **TanStack Query** + Zustand `siteStore` | 配置数据 → Query；当前活跃背景等 UI 状态 → Zustand |

**结果：Zustand 3 个 store (auth, ui, draft)，其余全部由 TanStack Query 管理。**

---

## 路由架构

### 布局对照

| 当前 Nuxt Layout | React Router 实现 | 视觉参考 |
|---|---|---|
| `layouts/default.vue` (导航头+页脚) | `<DefaultLayout>` layout route | 顶栏 + 内容 + 页脚，灰色全局背景 |
| `layouts/admin.vue` (固定侧边栏) | `<AdminLayout>` layout route | Cloudreve NavBarFrame 风格：TopBar + 侧边抽屉 + Main |
| `layouts/clean.vue` (悬浮头) | `<CleanLayout>` layout route | 无侧边栏，悬浮顶栏，专注阅读 |
| `layouts/blank.vue` (全屏居中) | `<BlankLayout>` layout route | Cloudreve HeadlessFrame 风格：灰色全屏 + 居中 Paper 卡片 |

### 路由守卫

| 当前 Nuxt Middleware | React Router 实现 |
|---|---|
| `middleware/auth.ts` (需登录) | `<RequireAuth>` 组件包裹路由，检查 Zustand `authStore.isLoggedIn` |
| `middleware/admin.ts` (需管理员) | `<RequireAdmin>` 组件包裹路由，检查 `authStore.isAdmin` |

**无 `middleware.ts` 文件** — SPA 中守卫就是普通 React 组件，通过 `<Navigate>` 重定向。

### 路由树

```
/                                     → HomePage         (DefaultLayout)
/articles                             → ArticleListPage  (DefaultLayout)
/articles/:slug                       → ArticleDetail    (CleanLayout)
/about                                → AboutPage        (DefaultLayout)
/categories                           → CategoriesPage   (DefaultLayout)
/tags                                 → TagsPage         (DefaultLayout)
/login                                → LoginPage        (BlankLayout)
/profile                              → ProfilePage      (DefaultLayout + RequireAuth)
/admin                                → AdminDashboard   (AdminLayout + RequireAdmin)
/admin/articles                       → ArticleManage    (AdminLayout + RequireAdmin)
/admin/articles/edit/:id              → ArticleEditor    (AdminLayout + RequireAdmin)
/admin/categories                     → CategoryManage   (AdminLayout + RequireAdmin)
/admin/tags                           → TagManage        (AdminLayout + RequireAdmin)
/admin/site                           → SiteConfig       (AdminLayout + RequireAdmin)
```

---

## 认证流

保持当前策略不变：**Token 存 Cookie 为唯一真实来源**。

```
STORAGE:
  Cookie:  ley_at (Access Token, 15min)
           ley_rt (Refresh Token, 7d)
  Zustand: user ref only (id, username, email, avatar, bio, role)

FLOW:
  1. App 挂载 → AuthProvider → authStore.init()
  2. 读 ley_at cookie → GET /api/v1/users/me → 设置 user
  3. 401 → 读 ley_rt → POST /api/v1/auth/refresh → 重置 cookie → 重试
  4. Refresh 也失败 → 清除 cookie + user → 未登录状态

API 调用:
  ofetch.create() → onRequest hook 从 cookie 读 ley_at → Authorization header
  (auth store 的 login/register/refresh/logout 直接调 ofetch，不依赖 useApiClient)
```

---

## 项目目录结构

```
ley-web/
├── index.html
├── package.json
├── pnpm-lock.yaml
├── tsconfig.json
├── vite.config.ts
├── tailwind.config.ts
├── public/
│   └── favicon.ico
└── src/
    ├── main.tsx                    # createRoot + providers
    ├── routes.ts                   # createBrowserRouter 路由树
    ├── app.tsx                     # 根组件 (QueryClient + Theme + Auth + Router)
    ├── lib/
    │   ├── api-client.ts           # ofetch 封装
    │   └── i18n.ts                 # react-i18next 配置
    ├── hooks/                      # TanStack Query hooks
    │   ├── use-articles.ts
    │   ├── use-tags.ts
    │   ├── use-categories.ts
    │   ├── use-files.ts
    │   └── use-site.ts
    ├── stores/                     # Zustand (仅客户端状态)
    │   ├── auth.ts
    │   ├── ui.ts
    │   └── draft.ts
    ├── components/
    │   ├── ui/                     # Button, Modal, Input, Textarea, Tag, Pagination, Skeleton, Empty, Badge, Divider, Loading
    │   ├── layout/                 # DefaultLayout, AdminLayout, CleanLayout, BlankLayout
    │   ├── article/                # ArticleCard, ArticleList, ArticleContent, ArticleMeta, ArticleToc
    │   ├── admin/                  # 管理后台专用: StatCard, Sidebar, TopBar, DataTable, ConfirmDialog
    │   ├── animation/              # FadeIn, ScrollReveal, StaggerList, PageTransition, InkSpread
    │   ├── require-auth.tsx        # 路由守卫
    │   └── require-admin.tsx
    ├── pages/                      # 页面组件 (lazy loaded)
    │   ├── home.tsx
    │   ├── article-list.tsx
    │   ├── article-detail.tsx
    │   ├── about.tsx
    │   ├── categories.tsx
    │   ├── tags.tsx
    │   ├── login.tsx
    │   ├── profile.tsx
    │   └── admin/
    │       ├── dashboard.tsx
    │       ├── article-manage.tsx
    │       ├── article-editor.tsx
    │       ├── category-manage.tsx
    │       ├── tag-manage.tsx
    │       └── site-config.tsx
    ├── i18n/
    │   ├── locales/
    │   │   ├── zh-CN.json
    │   │   └── en-US.json
    │   └── config.ts
    └── styles/
        └── globals.css             # CSS 变量 + Tailwind
```

---

## 视觉设计系统 (参考 Cloudreve 风格)

### 核心理念：灰底白卡

借鉴 Cloudreve 的 Material Design 风格，用 TailwindCSS 实现：

> **永远不要让白色卡片直接贴在白色背景上。** 全局灰色底板 + 白色圆角卡片悬浮其上。

### 色彩体系

```
全局背景:
  light: gray-100 (#f5f5f5)    // Cloudreve: grey[100]
  dark:  gray-900 (#212121)    // Cloudreve: grey[900]

卡片/Paper:
  light: white (#ffffff)       // Cloudreve: background.paper
  dark:  gray-800 (#1e1e1e)

品牌色:
  light: blue-600 (#2563eb)    // 对标 Cloudreve 的 primary.main
  dark:  blue-400 (#60a5fa)

边框:
  light: gray-200 (#e5e7eb)    // 卡片可选 1px 边框
  dark:  gray-700 (#374151)
```

### 圆角系统

```
全局默认:   12px (rounded-xl)      // Cloudreve: shape.borderRadius = 12
  - Button:     rounded-xl
  - Card/Paper: rounded-xl
  - Input:      rounded-xl
  - Modal:      rounded-xl
  - Snackbar:   rounded-xl

小型组件:   8px (rounded-lg)
  - Menu:       rounded-lg
  - MenuItem:   rounded-lg
  - Dropdown:   rounded-lg
  - Badge:      rounded-lg

滚动条:     4px (rounded)
```

### 按钮规范

```
所有按钮:
  - 扁平无阴影: shadow-none (对标 disableElevation: true)
  - 原文大小写: normal-case (对标 textTransform: "none")
  - 禁用态:     opacity-50 cursor-not-allowed

变体:
  primary:   bg-blue-600 text-white hover:bg-blue-700
  secondary: bg-gray-100 text-gray-700 hover:bg-gray-200 border border-gray-200
  ghost:     text-gray-600 hover:bg-gray-100
  danger:    bg-red-500 text-white hover:bg-red-600
```

### 输入框规范

```
所有输入框:
  - 圆角 12px:  rounded-xl
  - 无底部边框: border-0 (对标 FilledInput before/after: none)
  - 浅灰填充:    bg-gray-50 dark:bg-gray-800
  - Focus:       ring-2 ring-blue-500

高度统一:
  sm: h-8   (32px)
  md: h-10  (40px)
  lg: h-12  (48px)
```

### 菜单规范

```
Dropdown menu:
  - 圆角 8px:    rounded-lg
  - 阴影:        shadow-lg
  - 内边距:      p-1 (4px)
  - 容器:        bg-white dark:bg-gray-800 border

MenuItem:
  - 圆角 8px:    rounded-lg
  - 水平 margin: mx-1
  - 内边距:      px-2 py-1.5
  - Hover:       hover:bg-gray-100 dark:hover:bg-gray-700
```

### 滚动条

```css
/* 对标 Cloudreve 自定义滚动条 — hover 变主题色 */
::-webkit-scrollbar { width: 8px; height: 8px; }
::-webkit-scrollbar-track { background: transparent; }
::-webkit-scrollbar-track:hover { background: #e5e7eb; }
::-webkit-scrollbar-thumb { border-radius: 4px; background: #d1d5db; }
::-webkit-scrollbar-thumb:hover { background: #2563eb; }
```

### 字体

```
正文/标题:  Geist (Vercel 无衬线字体，对标 Roboto)
代码:       Geist Mono

字重:
  300 → 辅助文字 / 描述
  400 → 正文
  500 → 加粗正文 / 导航
  700 → 标题
```

### 间距系统

```
放弃日式 ma-1 ~ ma-8，改用 Tailwind 原生间距:
  p-2  (8px)   → 紧凑内边距
  p-4  (16px)  → 标准卡片内边距
  p-6  (24px)  → 宽松区域
  gap-4 (16px) → 标准栅格间距
```

### 动效

```
微交互:
  duration: 150-300ms
  easing:   ease-out

页面过渡:
  fade-in + slide-up (framer-motion AnimatePresence)

Skeleton:
  pulse 动画 (对标 Cloudreve wave)
```

### 暗色模式

```css
/* 对标 Cloudreve: 系统偏好 > 用户选择 */
[data-theme='dark'] {
  --bg-global:  #212121;
  --bg-card:    #1e1e1e;
  --bg-input:   #374151;
  --text-primary: #f3f4f6;
  --text-secondary: #9ca3af;
  --border:     #374151;
}
```

### 设计 Token 对照总表

| 维度 | 当前 (侘寂和風) | 新 (Cloudreve 风格) |
|------|-----------------|---------------------|
| **色彩主调** | 苔绿 `#5d8c8c` | 蓝色 `#2563eb` (blue-600) |
| **全局背景** | 暖纸色 `#fdfcfa` | 浅灰 `#f5f5f5` (gray-100) → 灰底白卡 |
| **卡片** | 无层级区分 | `white` 浮动于灰底之上 |
| **字体** | Noto Serif SC (衬线) | **Geist** (无衬线) |
| **代码字体** | LXGW WenKai | **Geist Mono** |
| **圆角** | `rounded-sm` (~2px) | **`rounded-xl` (12px)** 全局统一 |
| **按钮** | 有阴影、全大写 | **扁平无阴影、自然大小写** |
| **输入框** | 底部边框线 | **无边框、浅灰填充、圆角** |
| **阴影** | 极淡单层 | 分层 shadow (sm/md/lg) |
| **滚动条** | 系统默认 | **自定义 8px、hover 变主题色** |
| **动效** | 慢节奏、InkSpread | **150-300ms 微交互** |
| **暗色模式** | 无 | **支持** (CSS 变量 + 系统偏好) |
| **组件前缀** | `Was` (和風) | 无前缀，目录命名空间 |

---

## 管理后台设计

### 布局：Cloudreve NavBarFrame 风格

管理后台与前台的博客渲染是**同一个 SPA**（非独立子应用）。布局借鉴 Cloudreve 的 NavBarFrame：

```
┌──────────────────────────────────────────┐
│ TopBar (h-14)                            │
│ Logo + 面包屑 + 用户头像/退出              │
├────────┬─────────────────────────────────┤
│        │                                 │
│ Sidebar│ Main (p-6)                     │
│ (w-56) │  ┌──────────────────────────┐   │
│        │  │ Card (rounded-xl p-4)    │   │
│ 导航    │  │                          │   │
│ · 仪表盘│  │  内容区                   │   │
│ · 文章  │  │                          │   │
│ · 分类  │  └──────────────────────────┘   │
│ · 标签  │                                 │
│ · 站点  │                                 │
│        │                                 │
│ ← 站点  │                                 │
└────────┴─────────────────────────────────┘

背景: gray-100  dark:gray-900
Sidebar: white  dark:gray-800
TopBar: white border-b dark:gray-800 dark:border-gray-700
```

**响应式：** `< sm` 时 Sidebar 变为 Drawer（从左侧滑出），TopBar 左侧显示 hamburger 按钮。

### 页面清单

| 路由 | 页面 | 核心组件 |
|------|------|----------|
| `/admin` | **仪表盘** | 统计卡片 (文章数、评论数、浏览量) + 最近文章列表 |
| `/admin/articles` | **文章管理** | 搜索 + 状态筛选 + 表格 (id/标题/分类/状态/时间) + 行操作 (编辑/发布/归档/删除) |
| `/admin/articles/edit/:id` | **文章编辑** | react-hook-form 表单: 标题/内容(Markdown编辑器)/摘要/封面/分类/标签 |
| `/admin/categories` | **分类管理** | 树形结构展示 + 行内编辑 + 新增/删除 (校验子节点和文章数) |
| `/admin/tags` | **标签管理** | 标签列表 + 行内新增 + 删除 |
| `/admin/site` | **站点配置** | 表单: 标题/副标题/描述/Logo/Favicon/SEO/社交链接/ICP/页脚 |

### 管理后台与前台的关系

```
同一个 SPA:
  /             → DefaultLayout (博客前台)
  /articles/:slug → CleanLayout (文章阅读)
  /admin/*      → AdminLayout   (管理后台)

布局切换:
  React Router layout routes 自然隔离
  前台: DefaultLayout / CleanLayout / BlankLayout
  后台: AdminLayout (RequireAdmin 守卫 + 完全不同的布局结构)

状态共享:
  auth 和 site Zustand store 全局共享
  TanStack Query 缓存跨页面复用
  管理后台操作后 → queryClient.invalidateQueries 自动刷新前台数据
```

### 表单策略

管理后台是表单密集型，使用 `react-hook-form` + `zod`：

```
所有表单统一模式:
  1. zod schema 定义校验规则
  2. useForm<z.infer<typeof schema>>({ resolver: zodResolver(schema) })
  3. Controller 包裹 MUI 风格输入框组件
  4. 提交: TanStack Query useMutation → onSuccess → toast + navigate

示例 (站点配置):
  const siteSchema = z.object({
    siteTitle: z.string().min(1).max(50),
    siteSubtitle: z.string().max(100).optional(),
    seoKeywords: z.string().optional(),
    enableLikes: z.boolean(),
  });
```

### 表格规范

```
文章管理表格:
  - 表头: gray-50 背景, 12px 小字, font-medium
  - 行: hover:bg-gray-50, border-b border-gray-100
  - 单元格: py-3 px-4, text-sm
  - 操作列: 图标按钮 (编辑/发布/删除), gap-1
  - 空状态: WasEmpty 风格占位

分页:
  - 底部: flex justify-between items-center py-3 px-4 border-t
  - 左侧: "共 X 条"
  - 右侧: 页码按钮组 (rounded-lg, 当前页 bg-blue-600 text-white)
```

### 状态反馈

```
所有写操作后统一反馈:
  - 成功: sonner.toast.success("操作成功")
  - 失败: sonner.toast.error("操作失败: {原因}")
  - 删除: Dialog 二次确认 (title + description + cancel/confirm)

Loading 态:
  - 页面级: Skeleton 占位
  - 按钮级: disabled + spinner 图标
  - 表格: Skeleton rows
```

---

## 部署

```
vite build → dist/ → Nginx 静态托管

Nginx 配置要点:
  try_files $uri /index.html;    # SPA fallback
  gzip_static on;                 # Vite 预压缩
  location /assets/ { expires 1y; }  # 静态资源长期缓存
```

**CD 流水线对标当前：**

```
当前: pnpm generate → rsync → nginx reload
新:   pnpm build   → rsync → nginx reload
```

流水线结构不变，仅构建命令改变。

---

## 与当前架构的关键差异

| 维度 | 当前 (Nuxt) | 新 (React SPA) |
|------|------------|-----------------|
| 渲染模式 | SSG (构建时静态生成) | CSR (客户端渲染) |
| SEO | 天然支持 | 非核心需求 (个人博客) |
| 首屏速度 | 静态 HTML 直出 | 空白 → Skeleton → 数据渲染 |
| 中间件 | 页面级 middleware | React 组件路由守卫 |
| 数据获取 | build 时 useAsyncData + 客户端 hydration | 纯客户端 TanStack Query lazy fetch |
| Store 数量 | 6 个 Pinia | 3 个 Zustand + TanStack Query |
| 组件复用 | Auto-import | 显式 import (更可控) |
| 动画 | 6 个手写 IntersectionObserver 组件 | framer-motion 声明式动画 |
| 暗色模式 | 无 | 支持 light/dark 双主题 |
