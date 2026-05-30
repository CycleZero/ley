<!--
  ArticleContent — 文章正文渲染

  职责：
  - 使用 @nuxtjs/mdc 渲染 Markdown 内容
  - 应用和纸风格 prose 样式（宽行距、衬线体、墨线引用等）
  - 图片懒加载 + 淡入效果

  Props:
    content     string — Markdown 内容
    proseClass  string — 额外的 prose 样式类

  使用方式：
    <ArticleContent :content="article.content" />
-->
<template>
  <div class="article-prose" :class="proseClass">
    <MDC :value="content" />
  </div>
</template>

<script setup lang="ts">
withDefaults(defineProps<{
  content: string
  proseClass?: string
}>(), {
  proseClass: '',
})
</script>

<style scoped>
/* 和纸风格 Markdown 渲染样式 */
.article-prose :deep(h1) {
  @apply text-h1 font-serif-jp font-semibold text-heading tracking-wide mb-8 text-center;
  line-height: 1.4;
}

.article-prose :deep(h2) {
  @apply text-h2 font-serif-jp font-semibold text-heading tracking-wide mt-12 mb-4;
  line-height: 1.4;
  padding: 0.6em 0;
  border-top: 1px solid #e0e0e0;
  border-bottom: 1px solid #e0e0e0;
}

.article-prose :deep(h3) {
  @apply text-h3 font-serif-jp font-semibold text-heading mt-8 mb-3;
  padding-left: 0.8em;
  border-left: 3px solid #5d8c8c;
  line-height: 1.5;
}

.article-prose :deep(p) {
  @apply text-body text-body leading-washi mb-7;
}

.article-prose :deep(blockquote) {
  @apply my-8 px-6 py-5 bg-surface rounded-sm;
  border-left: 3px solid #5d8c8c;
  color: #616161;
  font-style: normal;
}

.article-prose :deep(blockquote p:first-child) {
  margin-top: 0;
}

.article-prose :deep(blockquote p:last-child) {
  margin-bottom: 0;
}

.article-prose :deep(pre) {
  @apply my-8 p-5 bg-surface border border-subtle rounded-sm overflow-x-auto;
  font-size: 0.875rem;
  line-height: 1.7;
}

.article-prose :deep(pre code) {
  @apply font-mono bg-transparent p-0;
  color: inherit;
}

.article-prose :deep(:not(pre) > code) {
  @apply font-mono bg-surface px-1.5 py-0.5 rounded-sm;
  font-size: 0.9em;
  color: #b4715f;
}

.article-prose :deep(img) {
  @apply my-10 mx-auto border border-subtle p-1 bg-white rounded-sm;
  max-width: 100%;
  display: block;
  opacity: 0;
  transition: opacity 0.8s ease;
}

.article-prose :deep(img[src]) {
  opacity: 1;
}

.article-prose :deep(a) {
  @apply text-accent no-underline;
  border-bottom: 1px solid transparent;
  transition: border-color 0.3s ease;
}

.article-prose :deep(a:hover) {
  border-bottom-color: #5d8c8c;
}

.article-prose :deep(ul),
.article-prose :deep(ol) {
  @apply my-6 pl-5;
}

.article-prose :deep(li) {
  @apply my-2 text-body leading-washi;
}

.article-prose :deep(ul li) {
  list-style-type: none;
  position: relative;
}

.article-prose :deep(ul li::before) {
  content: '·';
  position: absolute;
  left: -1.2em;
  color: #9e9e9e;
}

.article-prose :deep(hr) {
  @apply my-12 border-0 text-center;
}

.article-prose :deep(hr::before) {
  content: '❧';
  color: #bdbdbd;
  font-size: 1.2rem;
}

.article-prose :deep(table) {
  @apply w-full my-8 text-sm;
  border-collapse: collapse;
}

.article-prose :deep(th),
.article-prose :deep(td) {
  @apply px-4 py-3 text-left;
  border-bottom: 1px solid #e0e0e0;
}

.article-prose :deep(th) {
  @apply font-semibold text-body;
  border-bottom-width: 2px;
}

.article-prose :deep(strong) {
  @apply font-semibold text-heading;
}

.article-prose :deep(em) {
  @apply text-muted;
  font-style: normal;
}
</style>
