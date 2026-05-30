<!--
  关于页 /about

  展示站点信息、作者介绍、社交链接。
-->
<template>
  <div class="max-w-2xl mx-auto px-6 py-16">
    <!-- 页面标题 -->
    <FadeIn direction="up" :delay="0">
      <div class="text-center mb-12">
        <h1 class="text-3xl font-serif-jp font-semibold text-heading tracking-widest">
          关于
        </h1>
        <div class="mt-4 flex items-center justify-center gap-3">
          <div class="w-12 h-px bg-subtle" />
          <div class="w-1.5 h-1.5 rounded-full bg-accent" />
          <div class="w-12 h-px bg-subtle" />
        </div>
      </div>
    </FadeIn>

    <!-- 站点介绍 -->
    <FadeIn direction="up" :delay="100">
      <div class="surface-card p-8 mb-10">
        <h2 class="text-lg font-serif-jp font-semibold text-heading tracking-wide mb-4">
          {{ siteStore.siteTitle }}
        </h2>
        <p class="text-sm text-muted leading-washi">
          {{ siteStore.config?.siteDescription || '在这里，用文字丈量时间的厚度，以静默回应世界的喧嚣。' }}
        </p>
      </div>
    </FadeIn>

    <!-- 统计 -->
    <FadeIn direction="up" :delay="200">
      <div class="grid grid-cols-3 gap-4 mb-10">
        <div class="surface-card p-6 text-center">
          <div class="text-2xl font-serif-jp font-semibold text-heading">
            {{ articleStore.total }}
          </div>
          <div class="mt-1 text-xs text-placeholder">文章</div>
        </div>
        <div class="surface-card p-6 text-center">
          <div class="text-2xl font-serif-jp font-semibold text-heading">
            {{ categoryStore.categories.length }}
          </div>
          <div class="mt-1 text-xs text-placeholder">分类</div>
        </div>
        <div class="surface-card p-6 text-center">
          <div class="text-2xl font-serif-jp font-semibold text-heading">
            {{ tagStore.tags.length }}
          </div>
          <div class="mt-1 text-xs text-placeholder">标签</div>
        </div>
      </div>
    </FadeIn>

    <!-- 社交链接 -->
    <FadeIn v-if="hasSocialLinks" direction="up" :delay="300">
      <div class="surface-card p-8">
        <h3 class="text-sm font-serif-jp font-semibold text-heading tracking-wider mb-4">
          联系方式
        </h3>
        <div class="space-y-3">
          <a
            v-if="siteStore.config?.socialEmail"
            :href="`mailto:${siteStore.config.socialEmail}`"
            class="flex items-center gap-3 text-sm text-muted hover:text-accent transition-colors duration-300"
          >
            <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M3 8l7.89 5.26a2 2 0 002.22 0L21 8M5 19h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v10a2 2 0 002 2z" />
            </svg>
            {{ siteStore.config.socialEmail }}
          </a>
          <a
            v-if="siteStore.config?.socialGithub"
            :href="siteStore.config.socialGithub"
            target="_blank"
            rel="noopener"
            class="flex items-center gap-3 text-sm text-muted hover:text-accent transition-colors duration-300"
          >
            <svg class="w-4 h-4" fill="currentColor" viewBox="0 0 24 24">
              <path d="M12 0c-6.626 0-12 5.373-12 12 0 5.302 3.438 9.8 8.207 11.387.599.111.793-.261.793-.577v-2.234c-3.338.726-4.033-1.416-4.033-1.416-.546-1.387-1.333-1.756-1.333-1.756-1.089-.745.083-.729.083-.729 1.205.084 1.839 1.237 1.839 1.237 1.07 1.834 2.807 1.304 3.492.997.107-.775.418-1.305.762-1.604-2.665-.305-5.467-1.334-5.467-5.931 0-1.311.469-2.381 1.236-3.221-.124-.303-.535-1.524.117-3.176 0 0 1.008-.322 3.301 1.23.957-.266 1.983-.399 3.003-.404 1.02.005 2.047.138 3.006.404 2.291-1.552 3.297-1.23 3.297-1.23.653 1.653.242 2.874.118 3.176.77.84 1.235 1.911 1.235 3.221 0 4.609-2.807 5.624-5.479 5.921.43.372.823 1.102.823 2.222v3.293c0 .319.192.694.801.576 4.765-1.589 8.199-6.086 8.199-11.386 0-6.627-5.373-12-12-12z" />
            </svg>
            GitHub
          </a>
          <a
            v-if="siteStore.config?.socialTwitter"
            :href="siteStore.config.socialTwitter"
            target="_blank"
            rel="noopener"
            class="flex items-center gap-3 text-sm text-muted hover:text-accent transition-colors duration-300"
          >
            <svg class="w-4 h-4" fill="currentColor" viewBox="0 0 24 24">
              <path d="M18.244 2.25h3.308l-7.227 8.26 8.502 11.24H16.17l-5.214-6.817L4.99 21.75H1.68l7.73-8.835L1.254 2.25H8.08l4.713 6.231zm-1.161 17.52h1.833L7.084 4.126H5.117z" />
            </svg>
            X / Twitter
          </a>
        </div>
      </div>
    </FadeIn>

    <!-- 页脚信息 -->
    <FadeIn direction="up" :delay="400">
      <div class="mt-10 text-center">
        <p v-if="siteStore.config?.footerText" class="text-xs text-placeholder">
          {{ siteStore.config.footerText }}
        </p>
        <p v-if="siteStore.config?.icpNumber" class="mt-1 text-xs text-placeholder">
          {{ siteStore.config.icpNumber }}
        </p>
      </div>
    </FadeIn>
  </div>
</template>

<script setup lang="ts">
const siteStore = useSiteStore()
const articleStore = useArticleStore()
const categoryStore = useCategoryStore()
const tagStore = useTagStore()

// SSR 并行获取数据
await useAsyncData('about-page', async () => {
  await Promise.all([
    siteStore.fetchConfig(),
    articleStore.fetchArticles({ page: 1, pageSize: 1 }),
    categoryStore.fetchCategories(),
    tagStore.fetchTags(),
  ])
  return true
}, {
  server: true,
  lazy: false,
})

// 客户端兜底：nuxt generate 下 payload 恢复不执行副作用
onMounted(() => {
  if (!siteStore.config) siteStore.fetchConfig()
  if (!articleStore.articles.length) articleStore.fetchArticles({ page: 1, pageSize: 1 })
  if (!categoryStore.categories.length) categoryStore.fetchCategories()
  if (!tagStore.tags.length) tagStore.fetchTags()
})

// 计算是否有社交链接
const hasSocialLinks = computed(() => {
  const c = siteStore.config
  return !!(c?.socialEmail || c?.socialGithub || c?.socialTwitter)
})

// 页面标题
useHead({
  title: '关于',
})
</script>
