<script setup lang="ts">
/**
 * ArticleCard — 文章列表卡片组件
 *
 * 用于首页文章列表、标签/分类下文章列表、搜索结果页。
 *
 * Props:
 *   article - ArticleInfo，来自 API 的单篇文章数据结构
 *
 * 展示内容：
 *   封面图（左侧 120×80）｜ 标题 / 摘要（2 行截断）/ 日期·标签 / 阅读/点赞/评论数
 *   无封面图时用灰色占位
 */
import type { ArticleInfo } from '~~/lib/types'
import { formatDate, formatCount } from '~~/lib/utils/format'

defineProps<{ article: ArticleInfo }>()
</script>

<template>
  <NuxtLink
    :to="`/articles/${article.slug}`"
    class="group flex gap-4 p-4 rounded-lg border border-gray-200 hover:border-blue-300 hover:shadow-sm transition-all duration-200"
  >
    <!-- 封面图 -->
    <div class="w-28 h-20 shrink-0 rounded-md overflow-hidden bg-gray-100">
      <img
        v-if="article.cover_image"
        :src="article.cover_image"
        :alt="article.title"
        class="w-full h-full object-cover group-hover:scale-105 transition-transform duration-300"
        loading="lazy"
      />
      <!-- 无封面图占位 -->
      <div v-else class="w-full h-full flex items-center justify-center text-gray-300">
        <svg class="w-8 h-8" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5">
          <path stroke-linecap="round" stroke-linejoin="round" d="M19.5 14.25v-2.625a3.375 3.375 0 00-3.375-3.375h-1.5A1.125 1.125 0 0113.5 7.125v-1.5a3.375 3.375 0 00-3.375-3.375H8.25m2.25 0H5.625c-.621 0-1.125.504-1.125 1.125v17.25c0 .621.504 1.125 1.125 1.125h12.75c.621 0 1.125-.504 1.125-1.125V11.25a9 9 0 00-9-9z" />
        </svg>
      </div>
    </div>

    <!-- 右侧信息 -->
    <div class="flex-1 min-w-0 flex flex-col justify-between">
      <!-- 标题 -->
      <h3 class="text-base font-semibold text-gray-900 group-hover:text-blue-600 transition-colors line-clamp-1">
        {{ article.title }}
      </h3>

      <!-- 摘要（最多 2 行） -->
      <p v-if="article.excerpt" class="text-sm text-gray-500 line-clamp-2 mt-1">
        {{ article.excerpt }}
      </p>

      <!-- 底部元信息行：日期 · 标签 · 统计 -->
      <div class="flex items-center gap-2 mt-2 text-xs text-gray-400 flex-wrap">
        <!-- 发表日期 -->
        <time v-if="article.published_at">{{ formatDate(article.published_at) }}</time>
        <time v-else>{{ formatDate(article.created_at) }}</time>

        <!-- 分隔点 -->
        <span v-if="article.tags?.length" class="text-gray-300">·</span>

        <!-- 标签（最多显示 3 个） -->
        <span v-for="tag in article.tags?.slice(0, 3)" :key="tag.id" class="text-blue-500">
          #{{ tag.name }}
        </span>

        <!-- 右侧统计（用 ml-auto 推到最右侧） -->
        <span class="ml-auto flex items-center gap-3">
          <span v-if="article.view_count > 0" class="flex items-center gap-0.5">
            <svg class="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
              <path stroke-linecap="round" stroke-linejoin="round" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
              <path stroke-linecap="round" stroke-linejoin="round" d="M2.458 12C3.732 7.943 7.523 5 12 5c4.478 0 8.268 2.943 9.542 7-1.274 4.057-5.064 7-9.542 7-4.477 0-8.268-2.943-9.542-7z" />
            </svg>
            {{ formatCount(article.view_count) }}
          </span>
          <span class="flex items-center gap-0.5">
            <svg class="w-3.5 h-3.5" :class="{ 'text-red-500': article.is_liked }" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
              <path stroke-linecap="round" stroke-linejoin="round" d="M4.318 6.318a4.5 4.5 0 000 6.364L12 20.364l7.682-7.682a4.5 4.5 0 00-6.364-6.364L12 7.636l-1.318-1.318a4.5 4.5 0 00-6.364 0z" />
            </svg>
            {{ formatCount(article.like_count) }}
          </span>
          <span v-if="article.comment_count > 0" class="flex items-center gap-0.5">
            <svg class="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
              <path stroke-linecap="round" stroke-linejoin="round" d="M8 12h.01M12 12h.01M16 12h.01M21 12c0 4.418-4.03 8-9 8a9.863 9.863 0 01-4.255-.949L3 20l1.395-3.72C3.512 15.042 3 13.574 3 12c0-4.418 4.03-8 9-8s9 3.582 9 8z" />
            </svg>
            {{ formatCount(article.comment_count) }}
          </span>
        </span>
      </div>
    </div>
  </NuxtLink>
</template>
