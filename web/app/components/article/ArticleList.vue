<!--
  ArticleList — 文章列表容器

  职责：
  - 接收文章数组，以网格或列表形式展示
  - 支持使用 StaggerList 动画入场
  - 支持空状态显示

  Props:
    articles        ArticleInfo[] — 文章数组
    columns         1 | 2 | 3 — 网格列数，默认 1
    showStagger     boolean — 是否使用 stagger 入场动画，默认 true
    emptyTitle      string — 空状态标题
    emptyDesc       string — 空状态描述

  Events:
    articleClick    文章点击 [slug: string]
    categoryClick   分类点击 [slug: string]
    loadMore        加载更多

  使用方式：
    <ArticleList
      :articles="articles"
      :columns="2"
      @article-click="navigateToArticle"
    />
-->
<template>
  <div>
    <!-- 空状态 -->
    <WasEmpty
      v-if="!articles?.length"
      :title="emptyTitle"
      :description="emptyDesc"
    />

    <!-- 列表 -->
    <template v-else>
      <!-- 使用 StaggerList 动画 -->
      <StaggerList v-if="showStagger">
        <div
          :class="[
            'grid gap-6',
            columnClasses[columns],
          ]"
        >
          <InkSpread
            v-for="(article, index) in articles"
            :key="article.id"
            color="#5d8c8c"
            :opacity="0.03"
            :spread-size="500"
          >
            <ArticleCard
              :article="article"
              :index="index"
              @click="$emit('articleClick', $event)"
              @category-click="$emit('categoryClick', $event)"
            />
          </InkSpread>
        </div>
      </StaggerList>

      <!-- 无动画（纯列表） -->
      <div
        v-else
        :class="[
          'grid gap-6',
          columnClasses[columns],
        ]"
      >
        <InkSpread
          v-for="article in articles"
          :key="article.id"
          color="#5d8c8c"
          :opacity="0.03"
          :spread-size="500"
        >
          <ArticleCard
            :article="article"
            @click="$emit('articleClick', $event)"
            @category-click="$emit('categoryClick', $event)"
          />
        </InkSpread>
      </div>
    </template>
  </div>
</template>

<script setup lang="ts">
withDefaults(defineProps<{
  articles: ArticleInfo[]
  columns?: 1 | 2 | 3
  showStagger?: boolean
  emptyTitle?: string
  emptyDesc?: string
}>(), {
  columns: 1,
  showStagger: true,
  emptyTitle: '暂无文章',
  emptyDesc: '',
})

defineEmits<{
  articleClick: [slug: string]
  categoryClick: [slug: string]
  loadMore: []
}>()

const columnClasses = {
  1: 'grid-cols-1',
  2: 'grid-cols-1 md:grid-cols-2',
  3: 'grid-cols-1 md:grid-cols-2 lg:grid-cols-3',
}
</script>
