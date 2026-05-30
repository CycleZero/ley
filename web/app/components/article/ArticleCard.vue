<!--
  ArticleCard — 文章卡片

  职责：
  - 以和纸卡片形式展示文章摘要
  - 悬停时产生上浮 + 墨迹扩散效果
  - 点击跳转到文章详情

  Props:
    article     ArticleInfo — 文章数据
    index       number — 可选，用于 stagger 动画索引

  Events:
    click       点击事件 [slug: string]

  使用方式：
    <InkSpread>
      <ArticleCard :article="article" @click="goToArticle" />
    </InkSpread>
-->
<template>
  <article
    class="group relative p-5 border border-subtle rounded-sm bg-base
           hover:-translate-y-1 hover:shadow-subtle transition-all duration-300 ease-out cursor-pointer"
    :data-index="index"
    @click="handleClick"
  >
    <!-- 封面图（如果有） -->
    <div
      v-if="article.coverImage"
      class="mb-4 aspect-video rounded-sm overflow-hidden bg-surface"
    >
      <NuxtImg
        :src="article.coverImage"
        :alt="article.title"
        class="w-full h-full object-cover group-hover:scale-105 transition-transform duration-500"
        loading="lazy"
      />
    </div>

    <!-- 分类标签 -->
    <div v-if="article.category?.name" class="mb-2">
      <WasTag
        :label="article.category.name"
        variant="ai"
        @click.stop="$emit('categoryClick', article.category.slug)"
      />
    </div>

    <!-- 标题 -->
    <h3
      class="text-h3 font-serif-jp text-heading mb-2 leading-snug
             group-hover:translate-x-0.5 transition-transform duration-300"
    >
      {{ article.title }}
    </h3>

    <!-- 摘要 -->
    <p class="text-sm text-muted leading-relaxed line-clamp-2 mb-4">
      {{ article.excerpt || '' }}
    </p>

    <!-- 元信息 -->
    <ArticleMeta
      :published-at="article.publishedAt"
      :view-count="article.viewCount"
      :like-count="article.likeCount"
      :tags="article.tags"
      size="sm"
    />
  </article>
</template>

<script setup lang="ts">
const props = defineProps<{
  article: ArticleInfo
  index?: number
}>()

const emit = defineEmits<{
  click: [slug: string]
  categoryClick: [slug: string]
}>()

function handleClick() {
  emit('click', props.article.slug)
}
</script>
