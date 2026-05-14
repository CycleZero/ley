<script setup lang="ts">
/**
 * AppPagination — 通用分页器组件
 *
 * 设计：移动端显示"上一页 / 下一页"简洁按钮，
 *       桌面端显示完整数字页码（带省略号）。
 *
 * Props:
 *   page       - 当前页码（1-based）
 *   totalPages - 总页数（total / pageSize 向上取整）
 *
 * 事件:
 *   @change(newPage) - 点击页码时触发，携带新页码
 */
defineProps<{
  page: number
  totalPages: number
}>()

const emit = defineEmits<{
  change: [page: number]
}>()

/** 生成带省略号的页码数组，例如 [1, '...', 4, 5, 6, '...', 20] */
function getPages(current: number, total: number): (number | '...')[] {
  if (total <= 7) {
    return Array.from({ length: total }, (_, i) => i + 1)
  }
  const pages: (number | '...')[] = [1]

  // 前省略：当前页距开头 > 3 页时插入
  if (current > 4) pages.push('...')

  // 中间窗口：当前页 ± 1
  const start = Math.max(2, current - 1)
  const end = Math.min(total - 1, current + 1)
  for (let i = start; i <= end; i++) pages.push(i)

  // 后省略：当前页距末尾 > 3 页时插入
  if (current < total - 3) pages.push('...')

  pages.push(total)
  return pages
}
</script>

<template>
  <nav v-if="totalPages > 1" class="flex items-center justify-center gap-1" aria-label="分页导航">
    <!-- 上一页 -->
    <button
      :disabled="page <= 1"
      :class="[
        'px-3 py-1.5 text-sm rounded-md border transition-colors',
        page <= 1
          ? 'border-gray-200 text-gray-400 cursor-not-allowed'
          : 'border-gray-300 text-gray-700 hover:bg-gray-50',
      ]"
      @click="emit('change', page - 1)"
    >
      上一页
    </button>

    <!-- 页码数字（桌面端 visible） -->
    <template v-for="p in getPages(page, totalPages)" :key="p">
      <span v-if="p === '...'" class="px-2 text-gray-400">...</span>
      <button
        v-else
        :class="[
          'hidden sm:inline-flex px-3 py-1.5 text-sm rounded-md border transition-colors',
          p === page
            ? 'bg-blue-600 text-white border-blue-600'
            : 'border-gray-300 text-gray-700 hover:bg-gray-50',
        ]"
        @click="emit('change', p)"
      >
        {{ p }}
      </button>
    </template>

    <!-- 下一页 -->
    <button
      :disabled="page >= totalPages"
      :class="[
        'px-3 py-1.5 text-sm rounded-md border transition-colors',
        page >= totalPages
          ? 'border-gray-200 text-gray-400 cursor-not-allowed'
          : 'border-gray-300 text-gray-700 hover:bg-gray-50',
      ]"
      @click="emit('change', page + 1)"
    >
      下一页
    </button>
  </nav>
</template>
