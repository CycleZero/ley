<script setup lang="ts">
/**
 * AppEmpty — 空状态占位组件
 *
 * 场景：列表无数据 / 搜索结果为空 / 评论为空
 *
 * Props:
 *   message - 显示文字（默认"暂无内容"）
 */
withDefaults(defineProps<{
  message?: string
}>(), {
  message: '暂无内容',
})
</script>

<template>
  <div class="flex flex-col items-center justify-center py-16 text-gray-400">
    <!-- 空盒子图标（SVG） -->
    <svg class="w-16 h-16 mb-4" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5">
      <path stroke-linecap="round" stroke-linejoin="round" d="M20 13V6a2 2 0 00-2-2H6a2 2 0 00-2 2v7m16 0v5a2 2 0 01-2 2H6a2 2 0 01-2-2v-5m16 0h-2.586a1 1 0 00-.707.293l-2.414 2.414a1 1 0 01-.707.293h-3.172a1 1 0 01-.707-.293l-2.414-2.414A1 1 0 006.586 13H4" />
    </svg>
    <p class="text-sm">{{ message }}</p>
    <!-- 插槽：可放操作按钮（如"去写一篇文章"） -->
    <slot />
  </div>
</template>
