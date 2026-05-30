<!--
  ArticleMeta — 文章元信息

  职责：
  - 显示文章的时间、阅读量、点赞数、标签等元信息
  - 支持两种尺寸：sm（卡片用）和 md（详情页用）

  Props:
    publishedAt    string — 发布时间
    viewCount      string | number — 阅读次数
    likeCount      string | number — 点赞数
    tags           TagInfo[] — 标签列表
    size           'sm' | 'md' — 尺寸
    showAuthor     boolean — 是否显示作者
    author         AuthorInfo — 作者信息
    readingTime    number — 阅读时长（分钟）

  Events:
    tagClick       标签点击 [slug: string]
-->
<template>
  <div class="flex flex-wrap items-center gap-x-4 gap-y-1" :class="sizeClasses[size]">
    <!-- 发布时间 -->
    <time class="text-placeholder" :class="textSize">
      {{ formatDate(publishedAt) }}
    </time>

    <!-- 阅读时长 -->
    <span v-if="readingTime" class="text-placeholder" :class="textSize">
      {{ readingTime }} 分钟阅读
    </span>

    <!-- 阅读量 -->
    <span class="flex items-center gap-1 text-placeholder" :class="textSize">
      <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M2.458 12C3.732 7.943 7.523 5 12 5c4.478 0 8.268 2.943 9.542 7-1.274 4.057-5.064 7-9.542 7-4.477 0-8.268-2.943-9.542-7z" />
      </svg>
      {{ viewCount }}
    </span>

    <!-- 点赞 -->
    <span class="flex items-center gap-1 text-placeholder" :class="textSize">
      <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M4.318 6.318a4.5 4.5 0 000 6.364L12 20.364l7.682-7.682a4.5 4.5 0 00-6.364-6.364L12 7.636l-1.318-1.318a4.5 4.5 0 00-6.364 0z" />
      </svg>
      {{ likeCount }}
    </span>

    <!-- 标签（小尺寸不显示，中尺寸显示） -->
    <div v-if="size === 'md' && tags?.length" class="flex items-center gap-2 mt-1 w-full">
      <WasTag
        v-for="tag in tags.slice(0, 5)"
        :key="tag.id"
        :label="tag.name"
        variant="default"
        @click="$emit('tagClick', tag.slug)"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
const props = withDefaults(defineProps<{
  publishedAt: string
  viewCount: string | number
  likeCount: string | number
  tags?: TagInfo[]
  size?: 'sm' | 'md'
  readingTime?: number
}>(), {
  tags: () => [],
  size: 'sm',
  readingTime: undefined,
})

defineEmits<{
  tagClick: [slug: string]
}>()

const sizeClasses = {
  sm: '',
  md: 'mt-4',
}

const textSize = computed(() => {
  return props.size === 'sm' ? 'text-xs' : 'text-sm'
})

function formatDate(date: string) {
  return new Date(date).toLocaleDateString('zh-CN', {
    year: 'numeric',
    month: 'long',
    day: 'numeric',
  })
}
</script>
