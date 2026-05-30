# Ley 前端设计文档

> 日式简约风格（Wabi-Sabi）+ 流畅动画效果

---

## 一、项目概述

Ley 前端采用 **Vue 3 + Nuxt 4** 技术栈，以**日式简约美学**为核心设计语言，追求极致的留白、克制的色彩和流畅的动画体验。

### 设计关键词

- **間（Ma）** — 留白，元素间的大面积呼吸空间
- **渋み（Shibumi）** — 素雅，低饱和度、不刺眼的和彩色调
- **侘寂（Wabi-Sabi）** — 质朴，接受自然的不完美，如手写体的温度
- **筆致（Hitsuchi）** — 笔意，动画如水墨般流动，有起承转合

---

## 二、技术选型

| 层面 | 技术 | 理由 |
|------|------|------|
| 框架 | **Vue 3 + Nuxt 4** | 与后端 Kratos 网关完美配合，支持 SSR/SSG/SPA 多种渲染模式 |
| 语言 | **TypeScript** | 与后端 Protobuf 类型对齐，提供全链路类型安全 |
| 样式 | **TailwindCSS + 自定义 Design Tokens** | 原子化 CSS + 日式色彩系统，实现设计一致性 |
| 状态管理 | **Pinia** | 轻量、TypeScript 友好、与 Vue 生态深度集成 |
| 动画 | **GSAP + @vueuse/motion + CSS Transitions** | 流畅、可控、性能优异，支持复杂时间线编排 |
| 图标 | **Lucide Vue** | 线条简洁、SVG 矢量、符合日式极简审美 |
| 字体 | **思源宋体（Noto Serif SC）** + **霞鹜文楷（LXGW WenKai）** | 中文日系排版质感，阅读温度 |
| Markdown | **@nuxtjs/mdc** | 原生 Nuxt 支持，可深度定制渲染组件 |
| 网络请求 | **Nuxt `$fetch`** | Nuxt 原生封装，支持 SSR/客户端同构、拦截器 |

---

## 三、日式简约设计系统

### 3.1 色彩系统（和色 WASHOKU）

```typescript
// tailwind.config.ts
import type { Config } from 'tailwindcss'

const config: Config = {
  theme: {
    extend: {
      colors: {
        // 纸张色 — 不用纯白，用暖白/米白，营造和纸质感
        washi: {
          50:  '#fdfcfa',   // 宣紙
          100: '#f7f5f0',   // 生成色（kinari）
          200: '#efece4',   // 胡粉（gofun）
          300: '#e8e3d9',   // 象牙色（zoge）
        },

        // 墨色系 — 不用纯黑，用温润的墨色调
        sumi: {
          50:  '#f5f5f5',
          100: '#e0e0e0',
          200: '#bdbdbd',
          300: '#9e9e9e',
          400: '#757575',
          500: '#616161',   // 墨色
          600: '#424242',
          700: '#2c2c2c',   // 濡羽色（nurebairo）
          800: '#1a1a1a',
          900: '#0f0f0f',
        },

        // 和彩 — 极低饱和度点缀色，克制使用
        enji:   '#b4715f',   // 臙脂 — 点赞、重要标记、警示
        ai:     '#5d8c8c',   // 藍 — 链接、交互、主色调
        matcha: '#8b9d7b',   // 抹茶 — 成功、发布、正向反馈
        sakura: '#e6d5d0',   // 桜 — hover 背景、柔和高亮
        kiiro:  '#c4a265',   // 黄土 — 标签、分类、次要强调
        sora:   '#8ba4be',   // 空 — 信息、辅助
      },
    },
  },
}

export default config
```

**使用原则**：
- 背景层 90% 使用 `washi-50/100`
- 文字主体使用 `sumi-700`，次要信息用 `sumi-400`
- 每页最多出现 2 种点缀色（`enji`/`ai`/`matcha`）
- 禁用高饱和度颜色（如纯红 `#ff0000`、纯蓝 `#0000ff`）

### 3.2 字体排版

```css
/* assets/css/fonts.css */

/* 标题：思源宋体 — 衬线带来的书卷气 */
.font-serif-jp {
  font-family: 'Noto Serif SC', 'Source Han Serif SC', 'Songti SC', serif;
}

/* 正文：霞鹜文楷 — 手写温度 + 优秀可读性 */
.font-body-jp {
  font-family: 'LXGW WenKai', 'PingFang SC', 'Microsoft YaHei', 'Hiragino Sans GB', sans-serif;
}

/* 英文/数字：Inter — 现代几何感 */
.font-sans {
  font-family: 'Inter', 'LXGW WenKai', sans-serif;
}

/* 代码：JetBrains Mono — 等宽、清晰 */
.font-mono {
  font-family: 'JetBrains Mono', 'LXGW WenKai', 'PingFang SC', monospace;
}
```

**排版尺度**：

| Token | 尺寸 | 用途 |
|-------|------|------|
| `text-display` | `2.5rem` (40px) | 首页大标题 |
| `text-h1` | `1.75rem` (28px) | 页面标题 |
| `text-h2` | `1.375rem` (22px) | 区块标题 |
| `text-h3` | `1.125rem` (18px) | 卡片标题 |
| `text-body` | `1rem` (16px) | 正文 |
| `text-sm` | `0.875rem` (14px) | 辅助文字、元信息 |
| `text-xs` | `0.75rem` (12px) | 时间戳、标签 |

**行高**：正文 `leading-[1.9]`，标题 `leading-[1.4]`，英文 `leading-[1.6]`

**字间距**：标题 `tracking-[0.05em]`，增加空气感

### 3.3 间距系统（間 MA）

日式设计的灵魂是**留白**。不使用紧凑的 4px 基线，而是更宽松的 8px 基线 + 大留白：

```typescript
// tailwind.config.ts — spacing 扩展
spacing: {
  ma: {
    1: '0.25rem',   // 4px   — 字距微调
    2: '0.5rem',    // 8px   — 行内间距
    3: '1rem',      // 16px  — 组件内 padding
    4: '1.5rem',    // 24px  — 组件间
    5: '2.5rem',    // 40px  — 模块间
    6: '4rem',      // 64px  — 区块间
    7: '6rem',      // 96px  — 页面级留白
    8: '10rem',     // 160px — 大区块呼吸
  }
}
```

**使用原则**：
- 段落间距 `> 行高`（如行高 1.9，段落间距 2.5em）
- 区块上下留白至少 `ma-6`
- 卡片内部 padding 至少 `ma-4`

### 3.4 圆角与阴影（无阴影哲学）

日式 UI 不使用投影制造层级，而是用**线**、**留白**和**微妙的 border** 区分：

```typescript
// tailwind.config.ts
borderRadius: {
  none: '0',
  sm: '2px',       // 几乎不圆 — 极度克制
  md: '4px',
  lg: '6px',
},

boxShadow: {
  // 不使用阴影制造层级
  // 如需强调，用 1px 实线或内阴影
  line: 'inset 0 -1px 0 0 #e0e0e0',
  subtle: '0 1px 2px rgba(0,0,0,0.04)',  // 极少使用
}
```

**边框风格**：
- 分割线使用 `1px solid sumi-100` 或 `sumi-200`
- 重要分隔使用双边框（上 1px + 下 1px）
- 卡片边框使用 `1px solid washi-200`

---

## 四、页面结构与路由设计

```
/                          首页 — 文章流 + 站点介绍 + 背景图
/about                     关于 — 作者信息、站点背景
/articles                  文章列表 — 瀑布流/时间线卡片
/articles/[slug]           文章详情 — 沉浸阅读页 + 评论区
/articles/[slug]/edit      文章编辑（管理员）
/write                     写文章（管理员）
/tags                      标签云 — 和式散点布局
/tags/[name]               标签文章聚合
/categories                分类树浏览 — 层级展开
/search                    全文搜索 — 实时结果
/login                     登录 — 极简表单
/register                  注册 — 极简表单
/profile                   个人中心 — 信息编辑
/admin                     管理后台仪表盘
/admin/articles            文章管理 — 表格 + 批量操作
/admin/comments            评论管理 — 审核列表
/admin/categories          分类管理 — 树形编辑器
/admin/tags                标签管理
/admin/files               文件管理 — 网格图库
/admin/site                站点配置 — 背景图、元信息
```

---

## 五、组件架构

### 5.1 布局组件（layouts/）

```
layouts/
├── default.vue          # 默认布局：顶部导航 + 主内容 + 页脚
├── clean.vue            # 纯净布局：文章阅读页专用（导航滚动浮现）
├── admin.vue            # 管理后台：固定侧边栏 + 顶部面包屑 + 内容区
└── blank.vue            # 空白：登录/注册页，全屏居中
```

### 5.2 核心组件目录结构

```
components/
├── layout/
│   ├── AppHeader.vue         # 智能导航 — 滚动隐藏/浮现（GSAP）
│   ├── AppFooter.vue         # 极简页脚 — 一行版权 + 链接
│   ├── AppSidebar.vue        # 文章页侧边目录 — 锚点跟踪
│   └── AppScrollTop.vue      # 回到顶部 — 和纸圆形按钮
├── article/
│   ├── ArticleCard.vue       # 文章卡片 — 悬停上浮 + 墨迹扩散
│   ├── ArticleList.vue       # 文章列表 — Stagger 入场动画容器
│   ├── ArticleContent.vue    # 正文渲染 — mdc 组件 + 和纸样式
│   ├── ArticleMeta.vue       # 元信息 — 时间、标签、阅读时长、点赞
│   ├── ArticleToc.vue        # 目录树 — 平滑滚动 + 当前高亮
│   ├── ArticleEditor.vue     # Markdown 编辑器 — 分屏预览（codemirror / milkdown）
│   └── ArticleToolbar.vue    # 阅读工具栏 — 字体大小、主题切换
├── comment/
│   ├── CommentTree.vue       # 嵌套评论树 — 缩进线动画
│   ├── CommentItem.vue       # 单条评论 — 悬停显示回复按钮
│   └── CommentEditor.vue     # 回复输入 — 点击展开动画
├── ui/                         # 和纸原子组件（Washi UI）
│   ├── WasButton.vue           # 和纸按钮 — 按压下沉效果
│   ├── WasInput.vue            # 输入框 — 下划线聚焦展开动画
│   ├── WasTextarea.vue         # 文本域 — 边框呼吸效果
│   ├── WasTag.vue              # 标签 — 细边框 + 悬停填充
│   ├── WasBadge.vue            # 徽标 — 小圆点/数字
│   ├── WasDivider.vue          # 分割线 — 实线/虚线/花结三种
│   ├── WasLoading.vue          # 加载 — 墨滴下落 + 晕染动画
│   ├── WasSkeleton.vue         # 骨架屏 — 和纸线条 shimmer
│   ├── WasEmpty.vue            # 空状态 — 枯山水 SVG 插画
│   ├── WasModal.vue            # 模态框 — 淡入 + 轻微缩放
│   ├── WasToast.vue            # 提示 — 右上角滑入，墨痕背景
│   ├── WasPagination.vue       # 分页 — 极简数字 + 箭头
│   ├── WasDropdown.vue         # 下拉菜单 — 淡入 + 位移
│   ├── WasSelect.vue           # 选择器 — 和纸风格
│   ├── WasTable.vue            # 表格 — 极简线框
│   └── WasTabs.vue             # 标签页 — 下划线滑动指示器
├── animation/
│   ├── FadeIn.vue              # 通用淡入封装（可配置方向、延迟）
│   ├── StaggerList.vue         # 列表 stagger 动画容器
│   ├── ScrollReveal.vue        # 滚动触发显示（IntersectionObserver）
│   ├── InkSpread.vue           # 墨迹扩散 hover 效果
│   ├── PageTransition.vue      # 页面切换动画（淡墨效果）
│   ├── SmoothScroll.vue        # 平滑滚动锚点
│   └── ParallaxImage.vue       # 视差图片（背景/头图）
├── form/
│   ├── FormField.vue           # 表单字段 — Label + Input + Error
│   ├── FormEditor.vue          # 富文本/Markdown 编辑器封装
│   └── ImageUploader.vue       # 图片上传 — 拖拽 + 预览 + MinIO 直传
└── search/
    ├── SearchInput.vue         # 搜索输入 — 聚焦展开动画
    ├── SearchResult.vue        # 搜索结果卡片
    └── SearchFilter.vue        # 筛选器 — 标签/分类/时间
```

---

## 六、动画设计方案

### 6.1 动画哲学

- **缓动优先使用 `ease-out`（进入）和 `ease-in`（离开）**
- **时长原则**：微交互 150-300ms，页面级 400-700ms，氛围动画可更长
- **不阻塞交互**：所有动画使用 `transform` 和 `opacity`，触发 GPU 加速
- **尊重用户**：支持 `prefers-reduced-motion` 媒体查询，自动降级为简单淡入淡出

### 6.2 页面级动画 — 淡墨（Fade Ink）

路由切换时如水墨在宣纸上晕开：

```typescript
// composables/usePageTransition.ts
export const pageTransition = {
  mode: 'out-in' as const,

  onBeforeEnter(el: HTMLElement) {
    gsap.set(el, {
      opacity: 0,
      y: 24,
      filter: 'blur(6px)',
    })
  },

  onEnter(el: HTMLElement, done: () => void) {
    gsap.to(el, {
      opacity: 1,
      y: 0,
      filter: 'blur(0px)',
      duration: 0.6,
      ease: 'power2.out',
      onComplete: done,
    })
  },

  onLeave(el: HTMLElement, done: () => void) {
    gsap.to(el, {
      opacity: 0,
      y: -16,
      filter: 'blur(4px)',
      duration: 0.35,
      ease: 'power2.in',
      onComplete: done,
    })
  },
}
```

### 6.3 列表级动画 — 落樱（Falling Petals）

文章卡片列表的 stagger 入场：

```vue
<!-- components/animation/StaggerList.vue -->
<template>
  <TransitionGroup
    tag="div"
    :css="false"
    @before-enter="onBeforeEnter"
    @enter="onEnter"
    @leave="onLeave"
  >
    <slot />
  </TransitionGroup>
</template>

<script setup lang="ts">
import gsap from 'gsap'

const onBeforeEnter = (el: HTMLElement) => {
  gsap.set(el, {
    opacity: 0,
    y: 32,
    rotateX: -6,
    transformOrigin: 'center top',
  })
}

const onEnter = (el: HTMLElement, done: () => void) => {
  const index = Number(el.dataset.index) || 0
  gsap.to(el, {
    opacity: 1,
    y: 0,
    rotateX: 0,
    duration: 0.7,
    delay: index * 0.08,      // 0.08s 级联，如水滴涟漪
    ease: 'power3.out',
    onComplete: done,
  })
}

const onLeave = (el: HTMLElement, done: () => void) => {
  gsap.to(el, {
    opacity: 0,
    y: -16,
    duration: 0.3,
    ease: 'power2.in',
    onComplete: done,
  })
}
</script>
```

### 6.4 交互级动画

#### 和纸按钮 — 按压凹陷（Washi Press）

```vue
<!-- components/ui/WasButton.vue -->
<template>
  <button
    :class="[
      'relative inline-flex items-center justify-center',
      'px-5 py-2.5 text-sm font-medium',
      'transition-all duration-150 ease-out',
      'rounded-sm select-none',
      // 默认：轻微浮起
      'transform -translate-y-px',
      'shadow-[inset_0_-2px_0_0_rgba(0,0,0,0.06)]',
      // 颜色变体
      variantClasses[variant],
      // 状态
      { 'opacity-50 cursor-not-allowed': disabled },
    ]"
    :disabled="disabled"
    @mousedown="isPressed = true"
    @mouseup="isPressed = false"
    @mouseleave="isPressed = false"
    :style="isPressed ? pressedStyle : {}"
  >
    <slot />
  </button>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'

const props = withDefaults(defineProps<{
  variant?: 'primary' | 'secondary' | 'ghost'
  disabled?: boolean
}>(), {
  variant: 'primary',
})

const isPressed = ref(false)

const variantClasses = {
  primary:   'bg-sumi-800 text-washi-50 hover:bg-sumi-700',
  secondary: 'bg-washi-100 text-sumi-700 border border-sumi-200 hover:bg-washi-200',
  ghost:     'bg-transparent text-sumi-600 hover:bg-sumi-50 hover:text-sumi-800',
}

const pressedStyle = computed(() => ({
  transform: 'translateY(0)',
  boxShadow: 'inset 0 1px 3px rgba(0,0,0,0.10)',
}))
</script>
```

#### 输入框聚焦 — 墨线延伸（Ink Line）

```vue
<!-- components/ui/WasInput.vue -->
<template>
  <div class="relative">
    <input
      v-model="modelValue"
      :class="[
        'w-full bg-transparent py-2.5 px-1',
        'text-sumi-700 placeholder:text-sumi-300',
        'border-0 border-b border-sumi-200',
        'focus:outline-none',
        'transition-colors duration-400',
      ]"
      :placeholder="placeholder"
      @focus="isFocused = true"
      @blur="isFocused = false"
    />
    <!-- 聚焦时下划线从中心展开 -->
    <div
      class="absolute bottom-0 left-0 h-px bg-ai transition-all duration-400 ease-out"
      :class="isFocused ? 'w-full' : 'w-0 left-1/2 -translate-x-1/2'"
    />
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'

const modelValue = defineModel<string>()
withDefaults(defineProps<{
  placeholder?: string
}>(), {
  placeholder: '',
})

const isFocused = ref(false)
</script>
```

#### 文章卡片 Hover — 墨迹浮现（Ink Ghost）

```vue
<!-- components/article/ArticleCard.vue（hover 部分） -->
<template>
  <article
    class="group relative p-6 border border-washi-200 rounded-sm transition-all duration-300"
    @mousemove="handleMouseMove"
  >
    <!-- 背景墨迹（跟随鼠标） -->
    <div
      class="pointer-events-none absolute inset-0 opacity-0 group-hover:opacity-100
             transition-opacity duration-500 rounded-sm"
      :style="{
        background: `radial-gradient(600px circle at ${mouseX}px ${mouseY}px,
                      rgba(93,140,140,0.05) 0%, transparent 50%)`,
      }"
    />

    <!-- 标题微移 -->
    <h2 class="text-h3 font-serif-jp text-sumi-800
               group-hover:translate-x-1 transition-transform duration-300 ease-out">
      {{ title }}
    </h2>

    <!-- 元信息 -->
    <div class="mt-3 flex items-center gap-3 text-sm text-sumi-400">
      <time>{{ formatDate(publishedAt) }}</time>
      <WasDivider direction="vertical" />
      <span>{{ readTime }} 分钟阅读</span>
    </div>

    <!-- 摘要 -->
    <p class="mt-4 text-body text-sumi-500 leading-relaxed line-clamp-3">
      {{ excerpt }}
    </p>
  </article>
</template>

<script setup lang="ts">
import { ref } from 'vue'

const props = defineProps<{
  title: string
  publishedAt: string
  readTime: number
  excerpt: string
}>()

const mouseX = ref(0)
const mouseY = ref(0)

const handleMouseMove = (e: MouseEvent) => {
  const rect = (e.currentTarget as HTMLElement).getBoundingClientRect()
  mouseX.value = e.clientX - rect.left
  mouseY.value = e.clientY - rect.top
}

const formatDate = (date: string) => {
  return new Date(date).toLocaleDateString('zh-CN', {
    year: 'numeric',
    month: 'long',
    day: 'numeric',
  })
}
</script>
```

### 6.5 特殊动画

#### 加载态 — 墨滴（Ink Drop）

```vue
<!-- components/ui/WasLoading.vue -->
<template>
  <div class="flex flex-col items-center justify-center gap-4 py-12">
    <div class="relative w-8 h-12">
      <!-- 墨滴下落 -->
      <div class="absolute top-0 left-1/2 -translate-x-1/2 w-2 h-2 rounded-full bg-sumi-700
                  animate-ink-drop" />
      <!-- 落点晕染 -->
      <div class="absolute bottom-0 left-1/2 -translate-x-1/2
                  w-6 h-1.5 rounded-full bg-sumi-700/20
                  animate-ink-spread" />
    </div>
    <span class="text-sm text-sumi-400 tracking-widest">加载中</span>
  </div>
</template>

<style scoped>
@keyframes ink-drop {
  0% {
    transform: translateX(-50%) translateY(-16px) scale(0.6);
    opacity: 0;
  }
  40% {
    transform: translateX(-50%) translateY(0) scale(1.2);
    opacity: 1;
  }
  60% {
    transform: translateX(-50%) translateY(0) scale(0.9);
    opacity: 1;
  }
  100% {
    transform: translateX(-50%) translateY(0) scale(1);
    opacity: 1;
  }
}

@keyframes ink-spread {
  0% {
    transform: translateX(-50%) scaleX(0);
    opacity: 0.6;
  }
  100% {
    transform: translateX(-50%) scaleX(1);
    opacity: 0;
  }
}

.animate-ink-drop {
  animation: ink-drop 1.4s ease-in-out infinite;
}

.animate-ink-spread {
  animation: ink-spread 1.4s ease-out infinite;
}
</style>
```

#### 智能导航 — 滚动显隐

```typescript
// composables/useSmartHeader.ts
import { ref, watch } from 'vue'
import { useWindowScroll } from '@vueuse/core'

export function useSmartHeader(threshold = 80) {
  const { y } = useWindowScroll()
  const isVisible = ref(true)
  let lastScrollY = 0

  watch(y, (current) => {
    // 接近顶部始终显示
    if (current < threshold) {
      isVisible.value = true
    }
    // 向上滚动显示，向下滚动隐藏
    else {
      isVisible.value = current < lastScrollY
    }
    lastScrollY = current
  })

  // 导航栏样式过渡
  const headerClasses = computed(() => ({
    'transform -translate-y-full': !isVisible.value,
    'transform translate-y-0': isVisible.value,
    'bg-washi-50/90 backdrop-blur-md border-b border-sumi-100': y.value > threshold,
    'bg-transparent border-transparent': y.value <= threshold,
  }))\n
  return { isVisible, headerClasses, scrollY: y }
}
```

---

## 七、Markdown 内容样式（阅读体验）

文章详情页的核心是**沉浸阅读**。不使用默认的 Tailwind Typography，而是自定义和纸风格：

```scss
// assets/css/prose-washi.scss
.prose-washi {
  font-family: 'LXGW WenKai', 'PingFang SC', sans-serif;
  font-size: 1.0625rem;        // 17px，略大于默认，阅读更舒适
  line-height: 1.9;              // 宽松行距
  color: #2c2c2c;

  // 段落间距
  p {
    margin: 1.8em 0;
  }

  // 标题：思源宋体
  h1, h2, h3, h4 {
    font-family: 'Noto Serif SC', serif;
    font-weight: 600;
    letter-spacing: 0.05em;
    line-height: 1.4;
    color: #1a1a1a;
    margin-top: 2.5em;
    margin-bottom: 0.8em;
  }

  h1 {
    font-size: 1.75rem;
    text-align: center;
    margin-top: 0;
    margin-bottom: 1.5em;
  }

  // h2 双边框 — 传统日式笺纸风格
  h2 {
    font-size: 1.375rem;
    padding: 0.6em 0;
    border-top: 1px solid #e0e0e0;
    border-bottom: 1px solid #e0e0e0;
    margin-top: 3em;
  }

  h3 {
    font-size: 1.125rem;
    margin-top: 2em;
    // 左侧竖线装饰
    padding-left: 0.8em;
    border-left: 3px solid #5d8c8c;
  }

  // 引用块 — 左侧墨线 + 和纸背景
  blockquote {
    margin: 2em 0;
    padding: 1.2em 1.5em;
    border-left: 3px solid #5d8c8c;
    background: #f7f5f0;
    font-style: normal;         // 中文不用斜体
    color: #616161;
    border-radius: 0 4px 4px 0;

    p:first-child { margin-top: 0; }
    p:last-child { margin-bottom: 0; }
  }

  // 代码块 — 仿宣纸背景
  pre {
    margin: 2em 0;
    padding: 1.2em 1.5em;
    background: #f7f5f0;
    border: 1px solid #e8e3d9;
    border-radius: 4px;
    overflow-x: auto;
    font-size: 0.875rem;
    line-height: 1.7;

    code {
      font-family: 'JetBrains Mono', monospace;
      background: transparent;
      padding: 0;
    }
  }

  // 行内代码
  code {
    font-family: 'JetBrains Mono', monospace;
    background: #f7f5f0;
    padding: 0.15em 0.4em;
    border-radius: 3px;
    font-size: 0.9em;
    color: #b4715f;
  }

  // 图片 — 淡入 + 和纸边框
  img {
    display: block;
    max-width: 100%;
    margin: 2.5em auto;
    border: 1px solid #e0e0e0;
    padding: 4px;
    background: #fff;
    opacity: 0;
    transition: opacity 0.8s ease;

    &[src] { opacity: 1; }
  }

  // 链接 — 下划线悬停展开
  a {
    color: #5d8c8c;
    text-decoration: none;
    border-bottom: 1px solid transparent;
    transition: border-color 0.3s ease;

    &:hover {
      border-bottom-color: #5d8c8c;
    }
  }

  // 列表
  ul, ol {
    margin: 1.5em 0;
    padding-left: 1.8em;
  }

  li {
    margin: 0.5em 0;
  }

  ul li {
    list-style-type: none;
    position: relative;

    &::before {
      content: '·';
      position: absolute;
      left: -1.2em;
      color: #9e9e9e;
    }
  }

  // 分割线 — 花结
  hr {
    border: none;
    margin: 3em 0;
    text-align: center;

    &::before {
      content: '❧';
      color: #bdbdbd;
      font-size: 1.2rem;
    }
  }

  // 表格
  table {
    width: 100%;
    margin: 2em 0;
    border-collapse: collapse;
    font-size: 0.9375rem;
  }

  th, td {
    padding: 0.75em 1em;
    border-bottom: 1px solid #e0e0e0;
    text-align: left;
  }

  th {
    font-weight: 600;
    color: #424242;
    border-bottom-width: 2px;
  }

  // 强调
  strong {
    font-weight: 600;
    color: #1a1a1a;
  }

  em {
    font-style: normal;
    color: #616161;
  }
}
```

---

## 八、状态管理设计（Pinia）

```
stores/
├── auth.ts       # 用户信息、Token、登录态、Token 刷新
├── article.ts    # 文章列表缓存、当前文章、草稿自动保存
├── category.ts   # 分类树（本地缓存，变更频率低）
├── tag.ts        # 标签列表
├── comment.ts    # 评论树（按文章 ID 隔离缓存）
├── site.ts       # 站点配置、背景图片、基础信息
├── file.ts       # 文件列表、上传队列
├── ui.ts         # 全局 UI 状态：主题、导航显隐、弹窗队列、toast 列表
└── search.ts     # 搜索历史、当前结果缓存
```

### auth 模块关键设计

```typescript
// stores/auth.ts
import { defineStore } from 'pinia'

export const useAuthStore = defineStore('auth', () => {
  const accessToken = useCookie('ley_at')   // 15 分钟，HttpOnly 由后端控制
  const refreshToken = useCookie('ley_rt')  // 7 天
  const user = ref<UserInfo | null>(null)
  const isLoggedIn = computed(() => !!accessToken.value && !!user.value)

  // 登录
  async function login(credentials: LoginRequest) {
    const res = await $fetch<LoginReply>('/api/v1/auth/login', {
      method: 'POST',
      body: credentials,
    })
    accessToken.value = res.accessToken
    refreshToken.value = res.refreshToken
    user.value = res.user
    return res
  }

  // Token 刷新
  async function refresh() {
    if (!refreshToken.value) throw new Error('无刷新令牌')
    const res = await $fetch<RefreshTokenReply>('/api/v1/auth/refresh', {
      method: 'POST',
      body: { refreshToken: refreshToken.value },
    })
    accessToken.value = res.accessToken
    refreshToken.value = res.refreshToken
    return res
  }

  // 登出
  async function logout() {
    if (accessToken.value) {
      await $fetch('/api/v1/auth/logout', {
        method: 'POST',
        body: {
          accessToken: accessToken.value,
          refreshToken: refreshToken.value,
        },
      }).catch(() => {}) // 忽略网络错误，强制清空本地状态
    }
    accessToken.value = null
    refreshToken.value = null
    user.value = null
  }

  return {
    accessToken,
    refreshToken,
    user,
    isLoggedIn,
    login,
    refresh,
    logout,
  }
})
```

### API 拦截器封装

```typescript
// composables/useApi.ts
export function useApi() {
  const auth = useAuthStore()
  const toast = useToastStore()

  const api = $fetch.create({
    baseURL: useRuntimeConfig().public.apiBase,

    async onRequest({ options }) {
      if (auth.accessToken) {
        options.headers.set('Authorization', `Bearer ${auth.accessToken}`)
      }
    },

    async onResponseError({ response, options }) {
      // 401 自动刷新并重试
      if (response.status === 401 && !options.headers.get('X-Retry')) {
        try {
          await auth.refresh()
          options.headers.set('X-Retry', '1')
          return $fetch(response.request, options)
        }
        catch {
          auth.logout()
          navigateTo('/login')
          return
        }
      }

      // 统一错误提示
      const message = response._data?.message || `请求失败: ${response.status}`
      toast.error(message)
    },
  })

  return api
}
```

---

## 九、暗黑模式（濃墨 Dense Ink）

日式暗黑不是纯黑，而是**温润的浓墨色调**：

```css
.dark {
  /* 背景 */
  --bg-primary: #1a1a1a;        /* 濡羽色 */
  --bg-secondary: #242424;      /* 浓墨 */
  --bg-tertiary: #2c2c2c;       /* 淡墨底色 */

  /* 文字 */
  --text-primary: #e0e0e0;      /* 淡墨 */
  --text-secondary: #9e9e9e;    /* 灰墨 */
  --text-tertiary: #757575;     /* 枯墨 */

  /* 边框 */
  --border-primary: #424242;
  --border-secondary: #333333;

  /* 强调色（更柔和） */
  --accent-primary: #7aaba8;    /* 淡蓝 */
  --accent-secondary: #c4908a;  /* 淡臙脂 */
  --accent-success: #9db08e;    /* 淡抹茶 */
}
```

### 切换实现

使用 Nuxt 的 `@nuxtjs/color-mode` 模块：

```typescript
// nuxt.config.ts
export default defineNuxtConfig({
  colorMode: {
    classSuffix: '',
    preference: 'system',
    fallback: 'light',
  },
})
```

切换动画使用 View Transitions API（渐进增强）：

```typescript
// composables/useThemeTransition.ts
export async function toggleTheme() {
  const colorMode = useColorMode()

  if (document.startViewTransition) {
    await document.startViewTransition(() => {
      colorMode.preference = colorMode.value === 'dark' ? 'light' : 'dark'
    }).ready
  }
  else {
    colorMode.preference = colorMode.value === 'dark' ? 'light' : 'dark'
  }
}
```

---

## 十、性能与体验优化

### 10.1 首屏优化

- **Nuxt SSR**：文章详情页、首页使用 SSR，保证 SEO 和首屏速度
- **图片懒加载**：使用 Nuxt Image 组件，MinIO 提供多尺寸缩略图
- **字体子集化**：使用 `cn-font-split` 按需加载使用的字符，首屏字体 < 100KB
- **关键 CSS 内联**：构建时提取首屏关键 CSS 内联到 HTML

### 10.2 运行时优化

- **路由预加载**：Hover 文章卡片 200ms 后预取详情数据（`prefetch`）
- **虚拟滚动**：文章管理后台长列表使用 `vue-virtual-scroller`
- **防抖节流**：搜索输入 300ms 防抖，滚动事件 16ms 节流
- **组件懒加载**：管理后台各模块使用 `defineAsyncComponent`

### 10.3 动画性能

- 所有动画仅使用 `transform`、`opacity`、`filter`
- 对频繁动画元素添加 `will-change: transform`
- 使用 `IntersectionObserver` 替代滚动监听
- GSAP 动画在组件卸载时自动 `kill()`

### 10.4 无障碍（A11y）

- 所有图片提供 `alt` 文本
- 按钮和链接有明确的焦点状态（`focus-visible:ring`）
- 支持 `prefers-reduced-motion`：复杂动画降级为简单淡入淡出
- 色彩对比度符合 WCAG AA 标准（4.5:1）
- 表单字段均有对应的 `<label>`

### 10.5 PWA（可选增强）

```typescript
// nuxt.config.ts
export default defineNuxtConfig({
  pwa: {
    manifest: {
      name: 'Ley — 简约博客',
      short_name: 'Ley',
      theme_color: '#fdfcfa',
      background_color: '#fdfcfa',
    },
    workbox: {
      navigateFallback: '/',
      globPatterns: ['**/*.{js,css,html,png,svg,ico}'],
    },
  },
})
```

---

## 十一、项目目录结构

```
web/                          # Nuxt 4 前端项目根目录
├── .nuxt/                    # Nuxt 构建产物（自动生成）
├── .output/                  # 生产构建输出（自动生成）
├── assets/
│   ├── css/
│   │   ├── main.css          # Tailwind 入口 + 基础样式
│   │   ├── fonts.css         # 字体声明
│   │   ├── prose-washi.css   # Markdown 阅读样式
│   │   └── animations.css    # 全局动画关键帧
│   └── fonts/                # 本地字体文件（如子集化后的 WOFF2）
├── components/
│   ├── animation/            # 动画容器组件
│   ├── article/              # 文章相关组件
│   ├── comment/              # 评论相关组件
│   ├── form/                 # 表单组件
│   ├── layout/               # 布局组件
│   ├── search/               # 搜索组件
│   └── ui/                   # 和纸原子组件（Washi UI）
├── composables/
│   ├── useApi.ts             # API 客户端封装
│   ├── useAuth.ts            # 认证相关组合式函数
│   ├── useSmartHeader.ts     # 智能导航
│   ├── usePageTransition.ts  # 页面切换动画
│   ├── useThemeTransition.ts # 主题切换动画
│   └── useFormat.ts          # 日期/时间格式化
├── layouts/
│   ├── default.vue
│   ├── clean.vue
│   ├── admin.vue
│   └── blank.vue
├── middleware/
│   ├── auth.ts               # 路由守卫：需登录
│   ├── admin.ts              # 路由守卫：需管理员权限
│   └── guest.ts              # 路由守卫：禁止已登录用户访问登录页
├── pages/
│   ├── index.vue
│   ├── about.vue
│   ├── articles/
│   │   ├── index.vue
│   │   ├── [slug].vue
│   │   └── [slug]/edit.vue
│   ├── write.vue
│   ├── tags/
│   │   ├── index.vue
│   │   └── [name].vue
│   ├── categories.vue
│   ├── search.vue
│   ├── login.vue
│   ├── register.vue
│   ├── profile.vue
│   └── admin/
│       ├── index.vue
│       ├── articles.vue
│       ├── comments.vue
│       ├── categories.vue
│       ├── tags.vue
│       ├── files.vue
│       └── site.vue
├── plugins/
│   ├── gsap.client.ts        # GSAP 客户端插件
│   ├── toast.ts              # 全局 Toast 插件
│   └── markdown.ts           # Markdown 渲染增强
├── public/
│   ├── favicon.ico
│   ├── robots.txt
│   └── images/               # 静态图片资源
├── server/                   # Nuxt Server（可选：代理 API）
│   └── api/
├── stores/
│   ├── auth.ts
│   ├── article.ts
│   ├── category.ts
│   ├── tag.ts
│   ├── comment.ts
│   ├── site.ts
│   ├── file.ts
│   ├── ui.ts
│   └── search.ts
├── types/
│   ├── api.ts                # 后端 API 类型（可基于 openapi.yaml 生成）
│   ├── index.ts
│   └── components.d.ts       # 全局组件类型声明
├── app.vue                   # Nuxt 根组件
├── app.config.ts             # Nuxt 应用配置
├── nuxt.config.ts            # Nuxt 配置文件
├── tailwind.config.ts        # Tailwind 配置（日式 Design Tokens）
├── tsconfig.json             # TypeScript 配置
├── package.json
└── pnpm-lock.yaml
```

---

## 十二、关键页面设计稿描述

### 12.1 首页 `/`

```
┌─────────────────────────────────────────────────────────────┐
│                                                             │
│     [簡約文字標]      文章 · 标签 · 关于      [登录/头像]   │  ← 导航栏
│                                                             │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│                                                             │
│           「你的博客名称」                                   │
│                                                             │
│           一行简短的描述，字体稍大，颜色 sumi-500             │
│                                                             │
│                 ───────────────                               │  ← WasDivider 花结
│                                                             │
│                                                             │
│  ┌─────────────────────────┐  ┌──────────────────────────┐ │
│  │                         │  │                          │ │
│  │   最新文章              │  │   分类树                  │ │
│  │   ┌──────────────┐     │  │   · 技术                 │ │
│  │   │ ArticleCard  │     │  │     · 后端               │ │
│  │   └──────────────┘     │  │     · 前端               │ │
│  │   ┌──────────────┐     │  │   · 生活                 │ │
│  │   │ ArticleCard  │     │  │   · 随笔                 │ │
│  │   └──────────────┘     │  │                          │ │
│  │   ┌──────────────┐     │  │                          │ │
│  │   │ ArticleCard  │     │  │                          │ │
│  │   └──────────────┘     │  │                          │ │
│  │                         │  │                          │ │
│  │   [加载更多 →]          │  │                          │ │
│  │                         │  │                          │ │
│  └─────────────────────────┘  └──────────────────────────┘ │
│                                                             │
│                                                             │
│                 ───────────────                               │
│                                                             │
│                                                             │
│              © 2026 · Ley · 简约而美                        │  ← 页脚
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

### 12.2 文章阅读页 `/articles/[slug]`

使用 `clean.vue` 布局，最大化阅读空间：

```
┌─────────────────────────────────────────────────────────────┐
│                                                             │
│  [← 返回列表]                                    [目 录]   │  ← 极简顶部
│                                                             │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│                                                             │
│                    文章标题                                  │
│              2026年5月30日 · 标签 · 5 分钟阅读               │
│                                                             │
│           ───────────────────────────────                    │
│                                                             │
│                                                             │
│                    正文内容...                               │
│                    （宽行距、衬线体、                         │
│                     沉浸阅读体验）                           │
│                                                             │
│                                                             │
│           ───────────────────────────────                    │
│                                                             │
│                    [♡ 42]    [↗ 分享]                        │
│                                                             │
│           ───────────────────────────────                    │
│                                                             │
│                    评论区域                                  │
│                    ├─ 评论 1                                  │
│                    │  └─ 回复 1.1                            │
│                    │  └─ 回复 1.2                            │
│                    ├─ 评论 2                                  │
│                    └─ 评论 3                                  │
│                                                             │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

### 12.3 登录页 `/login`

使用 `blank.vue` 布局，全屏居中：

```
┌─────────────────────────────────────────────────────────────┐
│                                                             │
│                                                             │
│                                                             │
│                      [簡約文字標]                             │
│                                                             │
│                                                             │
│                      欢迎回来                                 │
│                                                             │
│              ┌─────────────────────────┐                    │
│              │ 用户名 / 邮箱           │  ← WasInput        │
│              └─────────────────────────┘                    │
│                                                             │
│              ┌─────────────────────────┐                    │
│              │ 密码                    │  ← WasInput        │
│              └─────────────────────────┘                    │
│                                                             │
│                      [ 登 录 ]                                │  ← WasButton primary
│                                                             │
│              还没有账号？ 去注册 →                            │
│                                                             │
│                                                             │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

### 12.4 管理后台 `/admin/*`

使用 `admin.vue` 布局，固定侧边栏：

```
┌──────────┬──────────────────────────────────────────────────┐
│          │  Dashboard / 文章管理                             │
│  [Logo]  │  ───────────────────────────────                  │
│          │                                                   │
│  · 仪表盘 │  [+ 新建文章] [批量操作 ▼]                       │
│  · 文章   │  ┌──────────────────────────────────────────┐    │
│  · 评论   │  │ ID │ 标题       │ 分类 │ 状态 │ 操作   │    │
│  · 分类   │  ├──────────────────────────────────────────┤    │
│  · 标签   │  │ 1  │ 文章标题   │ 技术 │ 已发布│ ✎ 🗑   │    │
│  · 文件   │  │ 2  │ 另一篇     │ 生活 │ 草稿  │ ✎ 🗑   │    │
│  · 站点   │  └──────────────────────────────────────────┘    │
│          │                                                   │
│  ─────── │              ← 1 2 3 4 5 →                       │
│          │                                                   │
│  [退出]  │                                                   │
└──────────┴──────────────────────────────────────────────────┘
```

---

## 十三、字体加载策略

为避免 FOIT（Flash of Invisible Text）和 FOUT（Flash of Unstyled Text）：

```css
/* 使用 font-display: swap，先显示系统字体，加载完成后切换 */
@font-face {
  font-family: 'LXGW WenKai';
  src: url('/fonts/LXGWWenKai-Regular.subset.woff2') format('woff2');
  font-weight: 400;
  font-display: swap;
}

@font-face {
  font-family: 'Noto Serif SC';
  src: url('/fonts/NotoSerifSC-Regular.subset.woff2') format('woff2');
  font-weight: 400 600;
  font-display: swap;
}
```

使用 `cn-font-split` 工具将字体按页面使用的字符动态分包，配合 Nuxt 的 `preload`：

```typescript
// nuxt.config.ts
export default defineNuxtConfig({
  app: {
    head: {
      link: [
        {
          rel: 'preload',
          href: '/fonts/LXGWWenKai-Regular.subset.woff2',
          as: 'font',
          type: 'font/woff2',
          crossorigin: 'anonymous',
        },
      ],
    },
  },
})
```

---

## 十四、后续迭代方向

1. **编辑器增强**：集成 Milkdown 或 Milkdown ProseMirror，支持数学公式、Mermaid 图表
2. **全文搜索**：对接后端 `/api/v1/articles/search`，支持高亮和筛选
3. **国际化**：使用 `@nuxtjs/i18n`，支持简体中文/繁体中文/English/Japanese
4. **RSS/Atom**：在 `/feed.xml` 提供文章订阅
5. **阅读进度条**：文章页顶部添加细线进度指示器
6. **图片灯箱**：文章图片点击放大，支持手势滑动浏览
7. **评论通知**：集成 WebSocket 或 SSE，实时通知新回复
8. **AI 助手**：文章页集成摘要生成、阅读建议等功能

---

*本文档定义了 Ley 博客平台前端的完整设计方案。技术选型、组件架构和动画方案均已对齐后端 API 能力（基于 `openapi.yaml`）。*
