<!--
  文章详情页 /articles/[slug]

  路由：/articles/:slug
  布局：clean.vue

  职责：
  - 展示单篇文章的完整内容
  - 沉浸阅读体验：宽行距、衬线体、侧边目录
  - 点赞/分享操作
-->
<template>
  <div class="max-w-4xl mx-auto px-6 py-ma-6">
    <!-- 加载中 -->
    <div v-if="pending" class="flex justify-center py-20">
      <WasLoading />
    </div>

    <!-- 文章不存在 -->
    <WasEmpty
      v-else-if="!article"
      title="文章不存在"
      description="该文章可能已被删除或移动"
    >
      <template #action>
        <WasButton variant="secondary" @click="navigateTo('/articles')">
          返回文章列表
        </WasButton>
      </template>
    </WasEmpty>

    <!-- 文章内容 -->
    <template v-else>
      <!-- 文章头部 -->
      <FadeIn direction="up" :delay="0">
        <header class="text-center mb-ma-7">
          <!-- 分类 -->
          <div v-if="article.category?.name" class="mb-4">
            <WasTag
              :label="article.category.name"
              variant="ai"
              @click="navigateTo(`/articles?category=${article.category.id}`)"
            />
          </div>

          <!-- 标题 -->
          <h1 class="text-display font-serif-jp text-heading mb-4 leading-tight">
            {{ article.title }}
          </h1>

          <!-- 元信息 -->
          <ArticleMeta
            :published-at="article.publishedAt"
            :view-count="article.viewCount"
            :like-count="article.likeCount"
            :tags="article.tags"
            :reading-time="readingTime"
            size="md"
            class="justify-center"
            @tag-click="goToTag"
          />
        </header>
      </FadeIn>

      <!-- 花结分割线 -->
      <FadeIn direction="none" :delay="0.1">
        <WasDivider variant="flower" spacing="lg" />
      </FadeIn>

      <!-- 正文 + 目录 -->
      <div class="flex gap-12">
        <!-- 正文 -->
        <main class="flex-1 min-w-0">
          <FadeIn direction="up" :delay="0.2">
            <ArticleContent :content="article.content" />
          </FadeIn>

          <!-- 底部操作栏 -->
          <FadeIn direction="up" :delay="0.3">
            <div class="flex items-center justify-center gap-6 mt-ma-7 pt-8 border-t border-subtle">
              <!-- 点赞 -->
              <button
                class="flex items-center gap-2 px-5 py-2.5 rounded-sm border transition-all duration-200"
                :class="article.isLiked
                  ? 'border-error-subtle bg-error-subtle text-error'
                  : 'border-default text-muted hover:border-default hover:text-body'"
                @click="toggleLike"
              >
                <svg class="w-5 h-5" :fill="article.isLiked ? 'currentColor' : 'none'" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M4.318 6.318a4.5 4.5 0 000 6.364L12 20.364l7.682-7.682a4.5 4.5 0 00-6.364-6.364L12 7.636l-1.318-1.318a4.5 4.5 0 00-6.364 0z" />
                </svg>
                <span class="text-sm">{{ article.likeCount }}</span>
              </button>

              <!-- 分享 -->
              <button
                class="flex items-center gap-2 px-5 py-2.5 rounded-sm border border-default text-muted hover:border-default hover:text-body transition-all duration-200"
                @click="shareArticle"
              >
                <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M8.684 13.342C8.886 12.938 9 12.482 9 12c0-.482-.114-.938-.316-1.342m0 2.684a3 3 0 110-2.684m0 2.684l6.632 3.316m-6.632-6l6.632-3.316m0 0a3 3 0 105.367-2.684 3 3 0 00-5.367 2.684zm0 9.316a3 3 0 105.368 2.684 3 3 0 00-5.368-2.684z" />
                </svg>
                <span class="text-sm">分享</span>
              </button>
            </div>
          </FadeIn>

          <!-- 预留评论区域 -->
          <FadeIn direction="up" :delay="0.4">
            <div class="mt-ma-7 pt-8 border-t border-subtle">
              <div class="text-center py-12">
                <p class="text-sm text-placeholder">评论功能开发中</p>
              </div>
            </div>
          </FadeIn>
        </main>

        <!-- 侧边目录（桌面端） -->
        <aside class="hidden lg:block w-48 flex-shrink-0">
          <ArticleToc :content="article.content" :offset="96" />
        </aside>
      </div>
    </template>
  </div>
</template>

<script setup lang="ts">
const route = useRoute()
const articleStore = useArticleStore()
const ui = useUiStore()

// 从路由获取 slug
const slug = computed(() => route.params.slug as string)

// 加载文章（useAsyncData 数据用于显示；payload 恢复时副作用不执行，需手动同步到 store）
const { pending, data: articleData } = await useAsyncData(
  `article-${slug.value}`,
  () => articleStore.fetchArticle(slug.value),
  { server: true },
)

// 优先使用 useAsyncData 返回的数据（payload 恢复时可用）
// fallback 到 Pinia store（直接访问时 hydration 恢复）
const article = computed(() => articleData.value?.article || articleStore.currentArticle)

// 同步到 store，确保 toggleLike 等操作使用 store 中的 currentArticle
// 兜底：nuxt generate 下客户端导航时 payload 恢复可能失败，需手动获取
onMounted(() => {
  if (articleData.value?.article) {
    articleStore.currentArticle = articleData.value.article
  }
  if (!article.value) {
    articleStore.fetchArticle(slug.value)
  }
  // 记录浏览量（客户端渲染完成后触发，避免 SSR 期间误计）
  if (article.value?.id) {
    articleStore.recordView(article.value.id)
  }
})

// 阅读时长估算（中文字数 / 500）
const readingTime = computed(() => {
  if (!article.value?.content) return 0
  const textLength = article.value.content.replace(/[#*`\s]/g, '').length
  return Math.max(1, Math.ceil(textLength / 500))
})

// 页面标题
useHead(() => ({
  title: article.value?.title || '文章详情',
}))

// 点赞
async function toggleLike() {
  if (!article.value) return
  await articleStore.toggleLike(article.value.id)
}

// 分享
function shareArticle() {
  if (navigator.share) {
    navigator.share({
      title: article.value?.title || '',
      url: window.location.href,
    }).catch(() => {})
  }
  else {
    navigator.clipboard.writeText(window.location.href)
    ui.toast('链接已复制到剪贴板', 'success')
  }
}

function goToTag(slug: string) {
  navigateTo(`/articles?tag=${slug}`)
}
</script>
