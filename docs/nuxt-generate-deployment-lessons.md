# Nuxt 4 静态部署踩坑记：从文章消失到登录态丢失，一次说清

> 本文记录 Ley 博客平台从开发环境迁移到 `nuxt generate` 纯静态部署过程中遇到的一系列连锁问题，以及背后的根本原因与修复方案。如果你也在用 Nuxt 4 做静态部署，这篇文章可能帮你少走很多弯路。

---

## 背景

Ley 是一个采用 Go/Kratos 后端 + Nuxt 4 前端构建的个人博客平台。开发阶段一切正常，但当我们切换到 `nuxt generate` 纯静态部署后，页面开始出现各种"诡异"现象：

- 点击文章标题，页面显示"文章不存在"，刷新一下又好了
- 回到首页，文章列表空白，刷新一下又好了
- 登录后刷新页面，登录态丢失，头像变回"登录"按钮

这些问题看似独立，实则指向同一个底层机制：**Nuxt 静态部署下的客户端 hydration 与 Pinia store 副作用的冲突**。本文将按时间线逐一拆解。

---

## 问题一：点击文章显示"文章不存在"

### 现象

从文章列表页点击一篇文章，URL 正确跳转到 `/articles/b-gorm-d`，但页面只显示一个"文章不存在"的提示。按 F5 刷新后，文章内容正常展示。

### 排查过程

首先怀疑是 API 问题。检查后端日志，发现客户端导航时根本没发请求——这说明数据应该来自预渲染的产物。进一步查看 `nuxt generate` 生成的 `_payload.json`，文章数据确实存在，但页面就是显示"不存在"。

问题的核心在 `[slug].vue` 的这段代码：

```vue
<script setup>
const articleStore = useArticleStore()
const slug = computed(() => route.params.slug)

const { pending } = await useAsyncData(
  `article-${slug.value}`,
  () => articleStore.fetchArticle(slug.value), // 副作用：设置 store.currentArticle
  { server: true },
)

const article = computed(() => articleStore.currentArticle) // 从 store 读取
</script>
```

在 SSR/SSG 阶段，`useAsyncData` 执行了 `fetchArticle`，副作用把 `currentArticle` 设为了文章对象。Nuxt 把 `useAsyncData` 的**返回值**序列化到 `_payload.json`。

但当用户从其他页面**客户端导航**进入这篇文章时：
1. Nuxt 从 `_payload.json` 恢复 `useAsyncData` 的数据（文章对象就在 payload 里）
2. **但不会重新执行 `fetchArticle`**——所以副作用（设置 `store.currentArticle`）**不会触发**
3. Pinia store 中 `currentArticle` 仍然是 `null`
4. `computed(() => articleStore.currentArticle)` 返回 `null`
5. 页面渲染"文章不存在"

### 修复方案

双保险策略：

1. **优先使用 `useAsyncData` 直接返回的数据**：不从 store 读取，而是从 `useAsyncData` 的 `data` 中提取
2. **`onMounted` 兜底**：如果数据仍为空，在客户端手动触发一次获取

```vue
<script setup>
const { pending, data: articleData } = await useAsyncData(
  `article-${slug.value}`,
  () => articleStore.fetchArticle(slug.value),
  { server: true },
)

// 优先使用 useAsyncData 返回的数据（payload 恢复时可用）
// fallback 到 Pinia store（直接访问时 hydration 恢复）
const article = computed(() => articleData.value?.article || articleStore.currentArticle)

// 同步到 store + 兜底获取
onMounted(() => {
  if (articleData.value?.article) {
    articleStore.currentArticle = articleData.value.article
  }
  if (!article.value) {
    articleStore.fetchArticle(slug.value)
  }
})
</script>
```

---

## 问题二：首页、标签页、分类页等"回到就空白"

### 现象

从文章详情页点击"回到首页"，发现首页文章列表、分类、标签全部空白。但直接刷新首页，一切正常。

### 根本原因

与问题一完全相同：所有使用 `useAsyncData` 并在内部通过副作用更新 Pinia store 的页面，在客户端导航后都会出现数据丢失。

```vue
<!-- index.vue -->
await useAsyncData('home-data', async () => {
  // 这些副作用在 payload 恢复时不会执行
  await siteStore.fetchConfig()
  await articleStore.fetchArticles()
  await categoryStore.fetchCategories()
  await tagStore.fetchTags()
})

const articles = computed(() => articleStore.articles) // 客户端导航后为空
```

### 修复方案

为所有受影响页面（`index.vue`、`articles/index.vue`、`tags.vue`、`categories.vue`、`about.vue`）添加 `onMounted` 兜底：

```js
onMounted(() => {
  if (!siteStore.config) siteStore.fetchConfig()
  if (!articleStore.articles.length) articleStore.fetchArticles({ page: 1, pageSize: 6 })
  if (!categoryStore.categories.length) categoryStore.fetchCategories()
  if (!tagStore.tags.length) tagStore.fetchTags()
})
```

### 副作用

这种修复会带来一个可见的 UX 问题：客户端导航时，页面会先显示空白/加载状态，然后数据才填充。这是因为静态部署下，我们无法在服务端获取用户特定状态（如 cookie），只能在客户端重新请求。

**如果追求无缝体验，静态部署不是最佳选择**——SSR（服务端渲染）或 ISR（增量静态再生）才是更好的方案。但在当前资源受限（1.6GB 内存服务器）的情况下，这是可接受的权衡。

---

## 问题三：登录后刷新页面，登录态丢失

### 现象

用户在登录页成功登录，跳转到首页后头像正常显示。但只要按 F5 刷新，顶栏立刻变回"登录"按钮，仿佛从未登录过。更诡异的是，DevTools 里明明看到 cookie 中的 `ley_at` 和 `ley_rt` 存在，但刷新后**cookie 消失了**。

### 根本原因：Pinia 序列化覆盖 cookie

这是今天最难排查、最隐蔽的问题。

我们的 `auth.ts` 最初是这样设计的：

```js
export const useAuthStore = defineStore('auth', () => {
  const accessToken = useCookie('ley_at', { default: () => null })
  const refreshToken = useCookie('ley_rt', { default: () => null })
  const user = ref(null)
  
  return { accessToken, refreshToken, user }
})
```

**`useCookie` 返回的是一个双向绑定的响应式 ref**：读取时从 `document.cookie` 获取，赋值时写回 `document.cookie`。

问题出在 `nuxt generate` 的**预渲染阶段**：
1. Nuxt 在构建时执行 `setup()`，`useCookie('ley_at')` 在 Node.js 环境中读取不到 cookie，返回 `null`
2. Pinia 把整个 store state 序列化到生成的 HTML 中（用于客户端 hydration）
3. 序列化的 JSON 中 `accessToken = null`
4. 浏览器加载页面时，Pinia 从 HTML 中恢复 state，把 `accessToken` 设为 `null`
5. **`useCookie` 的 ref 被设为 `null`，立即触发写回 `document.cookie`**
6. 原本有效的 `ley_at` cookie 被覆盖为 `null`，浏览器将其删除

这就解释了为什么：
- 登录后 cookie 存在（此时还没刷新，Pinia 在内存中正确维护状态）
- 一刷新，cookie 没了（Pinia hydration 把序列化的 `null` 写回 cookie）

### 修复方案：彻底断开 Pinia 与 cookie 的双向绑定

重构 `auth.ts`，核心思路：

1. **不在 Pinia state 中直接暴露 `useCookie` 的 ref**
2. **用普通 `ref` 维护内存状态，用 getter 函数读取 cookie**
3. **只有 `login`/`logout`/`refresh` action 才手动读写 cookie**

```js
export const useAuthStore = defineStore('auth', () => {
  // Cookie 读写工具（隔离在 Pinia 外部）
  const _atCookie = useCookie('ley_at', { default: () => null })
  const _rtCookie = useCookie('ley_rt', { default: () => null })
  
  function readTokenFromCookie() {
    return _atCookie.value // 直接读 cookie，绕过 Pinia state
  }
  
  function writeTokens(at, rt) {
    _atCookie.value = at
    _rtCookie.value = rt
  }
  
  // State：只用普通 ref，不绑定 cookie
  const user = ref(null)
  
  // Getter：从 cookie 实时读取
  const isLoggedIn = computed(() => !!readTokenFromCookie() && !!user.value)
  
  async function login(data) {
    const res = await $fetch('/api/v1/auth/login', { ... })
    writeTokens(res.tokenPair.accessToken, res.tokenPair.refreshToken)
    user.value = res.user
  }
  
  async function init() {
    if (user.value) return
    const at = readTokenFromCookie()
    if (at) {
      try {
        await fetchProfile()
      } catch (e) {
        // 401 时尝试 refresh
        const rt = readRefreshFromCookie()
        if (rt) await refresh()
      }
    }
  }
  
  return {
    user,
    isLoggedIn,
    // 外部只读，通过 getter 实时从 cookie 读取
    get accessToken() { return readTokenFromCookie() },
    get refreshToken() { return readRefreshFromCookie() },
    login,
    logout,
    refresh,
    init,
  }
})
```

这样 Pinia 序列化的 state 中只有 `user: null`，不会影响 cookie。`init()` 在 `onMounted` 中执行时，从 cookie 读取到真实 token，调用 API 恢复 user，登录态得以保持。

---

## 总结与反思

### 三个问题的关联

表面看是三个独立问题，实则共享一个底层主题：**`nuxt generate` 静态部署下，构建时执行环境与运行时客户端环境的差异**。

| 问题 | 构建时行为 | 客户端行为 | 冲突点 |
|------|-----------|-----------|--------|
| 文章/首页空白 | `useAsyncData` 执行，副作用更新 store | 从 payload 恢复，副作用不执行 | store 状态丢失 |
| 登录态丢失 | `useCookie` 读不到 cookie，返回 null | Pinia hydration 把 null 写回 cookie | cookie 被覆盖 |

### 经验教训

1. **`nuxt generate` 不是万能方案**。它适合内容不依赖用户状态、可完全预渲染的页面（如文档站、营销页）。对于需要登录态、用户特定数据的 SPA 行为，要么接受客户端重新获取的延迟，要么改用 SSR。

2. **`useCookie` + Pinia 是危险组合**。如果必须同时使用，确保 Pinia 不直接持有 `useCookie` 的 ref，或者用 `onMounted` 在 hydration 完成后手动修复。

3. **onMounted 兜底是静态部署的标配**。任何依赖 `useAsyncData` 副作用更新全局状态的页面，都应该在 `onMounted` 中检查状态完整性，必要时重新获取。

### 技术栈

- **后端**：Go 1.26 + Kratos 微服务框架 + GORM + PostgreSQL
- **前端**：Nuxt 4 + Vue 3 + TypeScript + Pinia + TailwindCSS
- **部署**：`nuxt generate` 纯静态文件 → Nginx 托管（阿里云 ECS，1.6GB 内存）

---

*2026 年 5 月 31 日，排查于 Ley 博客平台部署期间。作者：破元 (poyuan)*
