<script setup lang="ts">
/**
 * AppLoading — 加载骨架屏组件
 *
 * Props:
 *   lines   - 显示的行数（用于文章列表等场景，"矩形块"模拟内容行）
 *   variant - card(文章卡片占位) / list(纯行列表) / text(文字段落占位)
 */
withDefaults(defineProps<{
  lines?: number
  variant?: 'card' | 'list' | 'text'
}>(), {
  lines: 3,
  variant: 'list',
})
</script>

<template>
  <!-- 卡片骨架（文章列表场景） -->
  <div v-if="variant === 'card'" class="animate-pulse space-y-4">
    <div v-for="i in lines" :key="i" class="flex gap-4 p-4 border rounded-lg">
      <!-- 封面图占位 -->
      <div class="w-32 h-20 bg-gray-200 rounded" />
      <div class="flex-1 space-y-2">
        <div class="h-5 bg-gray-200 rounded w-3/4" />
        <div class="h-4 bg-gray-200 rounded w-full" />
        <div class="h-4 bg-gray-200 rounded w-1/2" />
      </div>
    </div>
  </div>

  <!-- 纯行骨架 -->
  <div v-else-if="variant === 'list'" class="animate-pulse space-y-3">
    <div v-for="i in lines" :key="i" class="h-4 bg-gray-200 rounded" :class="i === lines ? 'w-3/4' : 'w-full'" />
  </div>

  <!-- 文字段落骨架 -->
  <div v-else class="animate-pulse space-y-3">
    <div class="h-4 bg-gray-200 rounded w-full" />
    <div class="h-4 bg-gray-200 rounded w-5/6" />
    <div class="h-4 bg-gray-200 rounded w-4/6" />
    <div class="h-4 bg-gray-200 rounded w-full" />
    <div class="h-4 bg-gray-200 rounded w-3/4" />
  </div>
</template>
