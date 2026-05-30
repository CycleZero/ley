<!--
  WasPagination — 和纸分页

  职责：
  - 极简分页控件：数字 + 箭头
  - 支持上一页/下一页、页码跳转

  Props:
    currentPage   number — 当前页码
    totalPages    number — 总页数
    maxVisible    number — 最多显示多少页码按钮

  Events:
    change        页码变化 [page: number]
-->
<template>
  <nav v-if="totalPages > 1" class="flex items-center justify-center gap-1" aria-label="分页">
    <!-- 上一页 -->
    <button
      class="p-2 text-placeholder hover:text-body transition-colors disabled:opacity-30 disabled:cursor-not-allowed"
      :disabled="currentPage <= 1"
      aria-label="上一页"
      @click="goTo(currentPage - 1)"
    >
      <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M15 19l-7-7 7-7" />
      </svg>
    </button>

    <!-- 页码 -->
    <template v-for="page in visiblePages" :key="page">
      <span
        v-if="page === -1"
        class="px-2 text-placeholder select-none"
      >
        …
      </span>
      <button
        v-else
        class="min-w-[2rem] h-8 px-2 text-sm rounded-sm transition-colors"
        :class="[
          page === currentPage
            ? 'bg-inverted text-inverted font-medium'
            : 'text-muted hover:bg-surface hover:text-body',
        ]"
        :aria-current="page === currentPage ? 'page' : undefined"
        @click="goTo(page)"
      >
        {{ page }}
      </button>
    </template>

    <!-- 下一页 -->
    <button
      class="p-2 text-placeholder hover:text-body transition-colors disabled:opacity-30 disabled:cursor-not-allowed"
      :disabled="currentPage >= totalPages"
      aria-label="下一页"
      @click="goTo(currentPage + 1)"
    >
      <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M9 5l7 7-7 7" />
      </svg>
    </button>
  </nav>
</template>

<script setup lang="ts">
import { computed } from 'vue'

const props = withDefaults(defineProps<{
  currentPage: number
  totalPages: number
  maxVisible?: number
}>(), {
  maxVisible: 5,
})

const emit = defineEmits<{
  change: [page: number]
}>()

function goTo(page: number) {
  if (page >= 1 && page <= props.totalPages && page !== props.currentPage) {
    emit('change', page)
  }
}

const visiblePages = computed(() => {
  const pages: number[] = []
  const { currentPage, totalPages, maxVisible } = props

  if (totalPages <= maxVisible) {
    for (let i = 1; i <= totalPages; i++) pages.push(i)
    return pages
  }

  // 计算可见页码范围
  const half = Math.floor(maxVisible / 2)
  let start = currentPage - half
  let end = currentPage + half

  if (start < 1) {
    start = 1
    end = maxVisible
  }
  if (end > totalPages) {
    end = totalPages
    start = totalPages - maxVisible + 1
  }

  // 始终显示第一页
  if (start > 1) {
    pages.push(1)
    if (start > 2) pages.push(-1) // 省略号
  }

  for (let i = start; i <= end; i++) {
    pages.push(i)
  }

  // 始终显示最后一页
  if (end < totalPages) {
    if (end < totalPages - 1) pages.push(-1)
    pages.push(totalPages)
  }

  return pages
})
</script>
