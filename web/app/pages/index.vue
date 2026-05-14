<script setup lang="ts">
/**
 * 首页 — / (ISR 60s)
 *
 * 数据：
 *   GET /api/v1/site/config       → 站点标题/副标题/描述（供 SEO 元信息）
 *   GET /api/v1/site/backgrounds  → 活跃背景图（Banner 背景）
 *   GET /api/v1/articles          → 已发布文章列表（分页，每页 10 条）
 *
 * 布局：
 *   Banner 区（标题+副标题+背景图） → 文章列表 + 分页
 *
 * 注意：useHead / useSeoMeta 必须在 await 之前调用，
 *       否则 Nuxt SSR 阶段无法注入 <head> 内容。
 */

import type { ArticleInfo, ListArticlesReply, SiteBackground } from '~~/lib/types'
import { formatCount } from '~~/lib/utils/format'

// ---- 分页状态（声明在 await 之前，避免 hydration 时序问题） ----
const page = ref(1)
const pageSize = 10

// ---- SEO 元信息（必须在 await 之前调用） ----
useHead({
  titleTemplate: (title) => title ? `${title} - Ley Blog` : 'Ley Blog',
})
useSeoMeta({
  description: '',
  ogTitle: 'Ley Blog',
  ogDescription: '',
})

// ---- 站点配置（供页面内标题/副标题/SEO 描述） ----
const { data: siteData } = await useFetch('/api/v1/site/config', {
  key: 'site-config',
  transform: (
    res: { config: { site_title: string; site_subtitle: string; site_description: string } },
  ) => res.config,
})

const siteTitle = computed(() => siteData.value?.site_title || 'Ley Blog')
const siteSubtitle = computed(() => siteData.value?.site_subtitle || '')

// 动态更新 SEO 描述（服务端/客户端两端生效）
useSeoMeta({
  title: siteTitle.value,
  description: siteData.value?.site_description || siteSubtitle.value || siteTitle.value,
  ogTitle: siteTitle.value,
  ogDescription: siteData.value?.site_description || siteSubtitle.value || siteTitle.value,
})

// ---- 活跃背景图 ----
const { data: bgData } = await useFetch('/api/v1/site/backgrounds', {
  transform: (res: { backgrounds: SiteBackground[] }) =>
    res.backgrounds?.filter((b) => b.is_active),
})

const bannerBg = computed(() => bgData.value?.[0]?.url || '')

// ---- 文章列表（分页，客户端换页时重新 fetch） ----
const { data: articlesData, pending } = await useFetch('/api/v1/articles', {
  key: computed(() => `home-articles-p${page.value}`),
  query: computed(() => ({
    status: 'published',
    page: page.value,
    page_size: pageSize,
  })),
  transform: (res: ListArticlesReply) => res,
  watch: [page],
})

const articles = computed<ArticleInfo[]>(() => articlesData.value?.articles ?? [])
const totalArticles = computed(() => articlesData.value?.total ?? 0)
const totalPages = computed(() => Math.ceil(totalArticles.value / pageSize))
</script>

<template>
  <div>
    <!-- ================================
         Banner 区
         无背景图 → 渐变色背景（蓝→靛）
         有背景图 → 图片 + 半透明遮罩
         ================================ -->
    <section
      class="relative overflow-hidden py-24 sm:py-32 lg:py-40"
      :class="bannerBg ? '' : 'bg-gradient-to-br from-blue-600 to-indigo-700'"
    >
      <!-- 背景图 -->
      <img
        v-if="bannerBg"
        :src="bannerBg"
        alt=""
        class="absolute inset-0 w-full h-full object-cover"
        loading="eager"
      />
      <!-- 遮罩层 -->
      <div v-if="bannerBg" class="absolute inset-0 bg-black/50" />

      <!-- 文字 -->
      <div class="relative z-10 max-w-4xl mx-auto px-4 text-center text-white">
        <h1 class="text-3xl sm:text-4xl lg:text-5xl font-extrabold tracking-tight">
          {{ siteTitle }}
        </h1>
        <p v-if="siteSubtitle" class="mt-4 text-base sm:text-lg opacity-90">
          {{ siteSubtitle }}
        </p>
        <p class="mt-3 text-sm opacity-60">
          共 {{ formatCount(totalArticles) }} 篇文章
        </p>
      </div>
    </section>

    <!-- ================================
         文章列表区
         ================================ -->
    <section class="max-w-3xl mx-auto px-4 sm:px-6 lg:px-8 py-10">
      <!-- 加载态（首次 SSR 后客户端瀑布流 / 换页时触发） -->
      <AppLoading v-if="pending" :lines="4" variant="card" />

      <!-- 空数据 -->
      <AppEmpty v-else-if="articles.length === 0" message="还没有文章，敬请期待" />

      <!-- 卡片列表 -->
      <template v-else>
        <div class="space-y-4">
          <ArticleCard
            v-for="article in articles"
            :key="article.id"
            :article="article"
          />
        </div>

        <!-- 分页 -->
        <div v-if="totalPages > 1" class="mt-10">
          <AppPagination
            :page="page"
            :total-pages="totalPages"
            @change="(p: number) => (page = p)"
          />
        </div>
      </template>
    </section>
  </div>
</template>
