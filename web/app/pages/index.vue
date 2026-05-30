<!--
  首页 — 日式简约风格

  结构：
  - Hero：站点标题、副标题、描述（大面积留白，居中对齐）
  - 内容区：左侧文章列表 + 右侧侧边栏（标签云、分类、关于）

  数据：
  - siteStore 提供站点信息
  - articleStore 提供最新文章
  - tagStore 提供热门标签
  - categoryStore 提供分类树
-->
<template>
  <div>
    <!-- ========== Hero 区 ========== -->
    <section class="relative pt-24 pb-20 md:pt-32 md:pb-28">
      <div class="max-w-3xl mx-auto px-6 text-center">
        <!-- 站点标题 -->
        <FadeIn direction="up" :delay="0">
          <h1 class="text-4xl md:text-5xl font-serif-jp font-semibold tracking-widest text-heading leading-tight">
            {{ siteStore.siteTitle }}
          </h1>
        </FadeIn>

        <!-- 副标题 -->
        <FadeIn direction="up" :delay="150">
          <p class="mt-4 text-lg md:text-xl text-muted tracking-wide">
            {{ siteStore.siteSubtitle || '记录与思考的空间' }}
          </p>
        </FadeIn>

        <!-- 描述 -->
        <FadeIn direction="up" :delay="300">
          <p class="mt-6 text-sm text-placeholder leading-washi max-w-xl mx-auto">
            {{ siteStore.config?.siteDescription || '在这里，用文字丈量时间的厚度，以静默回应世界的喧嚣。' }}
          </p>
        </FadeIn>

        <!-- 装饰分割线 -->
        <FadeIn direction="up" :delay="450">
          <div class="mt-10 flex items-center justify-center gap-3">
            <div class="w-12 h-px bg-subtle" />
            <div class="w-1.5 h-1.5 rounded-full bg-accent" />
            <div class="w-12 h-px bg-subtle" />
          </div>
        </FadeIn>

        <!-- CTA -->
        <FadeIn direction="up" :delay="600">
          <div class="mt-8 flex items-center justify-center gap-4">
            <NuxtLink
              to="/articles"
              class="inline-flex items-center gap-2 px-6 py-2.5 bg-heading text-inverted text-sm tracking-wider
                     hover:bg-body transition-colors duration-300"
            >
              浏览文章
              <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M17 8l4 4m0 0l-4 4m4-4H3" />
              </svg>
            </NuxtLink>
          </div>
        </FadeIn>
      </div>
    </section>

    <!-- ========== 主体内容区 ========== -->
    <section class="max-w-6xl mx-auto px-6 pb-20">
      <div class="grid grid-cols-1 lg:grid-cols-12 gap-10 lg:gap-12">
        <!-- 左侧：文章列表 -->
        <div class="lg:col-span-8">
          <!-- 区块标题 -->
          <div class="flex items-center justify-between mb-8">
            <h2 class="text-xl font-serif-jp font-semibold text-heading tracking-wide">
              最新文章
            </h2>
            <NuxtLink
              to="/articles"
              class="text-sm text-muted hover:text-accent transition-colors duration-300 flex items-center gap-1"
            >
              全部文章
              <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M9 5l7 7-7 7" />
              </svg>
            </NuxtLink>
          </div>

          <!-- 加载中 -->
          <div v-if="pending" class="py-16 flex flex-col items-center">
            <WasLoading size="md" />
            <span class="mt-4 text-sm text-placeholder tracking-widest">加载中...</span>
          </div>

          <!-- 错误 -->
          <div v-else-if="error" class="py-16 text-center">
            <p class="text-error text-sm">{{ error }}</p>
            <button
              class="mt-3 text-sm text-muted hover:text-accent transition-colors"
              @click="refresh"
            >
              重试
            </button>
          </div>

          <!-- 文章列表 -->
          <StaggerList v-else-if="articles.length" :stagger="80">
            <article
              v-for="article in articles"
              :key="article.id"
              class="group mb-10 last:mb-0"
            >
              <NuxtLink :to="`/articles/${article.slug}`" class="block">
                <!-- 日期 -->
                <time class="text-xs text-placeholder tracking-wider">
                  {{ formatDate(article.publishedAt) }}
                </time>

                <!-- 标题 -->
                <h3 class="mt-2 text-xl font-serif-jp font-semibold text-heading group-hover:text-accent transition-colors duration-300 leading-snug">
                  {{ article.title }}
                </h3>

                <!-- 摘要 -->
                <p class="mt-3 text-sm text-muted leading-washi line-clamp-2">
                  {{ article.excerpt || '暂无摘要' }}
                </p>

                <!-- 元信息 -->
                <div class="mt-4 flex items-center gap-4 text-xs text-placeholder">
                  <span v-if="article.category" class="flex items-center gap-1">
                    <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M3 7v10a2 2 0 002 2h14a2 2 0 002-2V9a2 2 0 00-2-2h-6l-2-2H5a2 2 0 00-2 2z" />
                    </svg>
                    {{ article.category.name }}
                  </span>
                  <span class="flex items-center gap-1">
                    <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
                      <path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M2.458 12C3.732 7.943 7.523 5 12 5c4.478 0 8.268 2.943 9.542 7-1.274 4.057-5.064 7-9.542 7-4.477 0-8.268-2.943-9.542-7z" />
                    </svg>
                    {{ article.viewCount }}
                  </span>
                  <span class="flex items-center gap-1">
                    <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M4.318 6.318a4.5 4.5 0 000 6.364L12 20.364l7.682-7.682a4.5 4.5 0 00-6.364-6.364L12 7.636l-1.318-1.318a4.5 4.5 0 00-6.364 0z" />
                    </svg>
                    {{ article.likeCount }}
                  </span>
                </div>
              </NuxtLink>

              <!-- 分割线 -->
              <WasDivider class="mt-10" />
            </article>
          </StaggerList>

          <!-- 空状态 -->
          <WasEmpty v-else description="暂无文章" />
        </div>

        <!-- 右侧：侧边栏 -->
        <aside class="lg:col-span-4 space-y-10">
          <!-- 关于站点 -->
          <FadeIn direction="up" :delay="100">
            <div class="surface-card p-6">
              <h3 class="text-sm font-serif-jp font-semibold text-heading tracking-wider mb-4">
                关于
              </h3>
              <p class="text-sm text-muted leading-washi">
                {{ siteStore.config?.siteDescription || '在这里，用文字丈量时间的厚度，以静默回应世界的喧嚣。' }}
              </p>
            </div>
          </FadeIn>

          <!-- 分类 -->
          <FadeIn direction="up" :delay="200">
            <div class="surface-card p-6">
              <h3 class="text-sm font-serif-jp font-semibold text-heading tracking-wider mb-4">
                分类
              </h3>
              <div v-if="categoryStore.categories.length" class="space-y-2">
                <NuxtLink
                  v-for="cat in categoryStore.flatCategories.slice(0, 8)"
                  :key="cat.category.id"
                  :to="`/articles?category=${cat.category.id}`"
                  class="flex items-center justify-between text-sm text-muted hover:text-accent transition-colors duration-300 py-1.5 border-b border-subtle last:border-0"
                >
                  <span class="flex items-center gap-2">
                    <span class="text-placeholder">{{ '　'.repeat(cat.depth) }}</span>
                    {{ cat.category.name }}
                  </span>
                  <span class="text-xs text-placeholder">{{ cat.category.articleCount }}</span>
                </NuxtLink>
              </div>
              <p v-else class="text-sm text-placeholder">暂无分类</p>
            </div>
          </FadeIn>

          <!-- 标签云 -->
          <FadeIn direction="up" :delay="300">
            <div class="surface-card p-6">
              <h3 class="text-sm font-serif-jp font-semibold text-heading tracking-wider mb-4">
                标签
              </h3>
              <div v-if="tagStore.hotTags.length" class="flex flex-wrap gap-2">
                <WasTag
                  v-for="tag in tagStore.hotTags.slice(0, 16)"
                  :key="tag.id"
                  :label="tag.name"
                  variant="default"
                  @click="navigateTo(`/articles?tag=${tag.id}`)"
                />
              </div>
              <p v-else class="text-sm text-placeholder">暂无标签</p>
            </div>
          </FadeIn>
        </aside>
      </div>
    </section>
  </div>
</template>

<script setup lang="ts">
const siteStore = useSiteStore()
const articleStore = useArticleStore()
const categoryStore = useCategoryStore()
const tagStore = useTagStore()

// ---------- SSR 数据获取 ----------

const { error, pending, refresh } = await useAsyncData('home-data', async () => {
  // 并行获取所有数据
  await Promise.all([
    siteStore.fetchConfig(),
    articleStore.fetchArticles({ page: 1, pageSize: 6 }),
    categoryStore.fetchCategories(),
    tagStore.fetchTags(),
  ])
  return true
}, {
  server: true,
  lazy: false,
})

// 从 store 取数据（SSR 已填充）
const articles = computed(() => articleStore.articles)

// ---------- 页面标题 ----------

useHead(() => ({
  title: siteStore.siteTitle,
}))

// ---------- 工具函数 ----------

function formatDate(date: string) {
  const d = new Date(date)
  return `${d.getFullYear()}年${d.getMonth() + 1}月${d.getDate()}日`
}
</script>
