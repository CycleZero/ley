<!--
  WasSkeleton — 和纸骨架屏

  职责：
  - 内容加载前的占位符
  - 使用 shimmer 动画营造和纸质感

  Props:
    variant   'text' | 'rect' | 'circle' | 'card' — 骨架形状
    lines     number — 文本行数（仅 variant='text' 有效）
    width     string — 自定义宽度（如 'w-3/4'）
    height    string — 自定义高度（如 'h-32'）
-->
<template>
  <div class="animate-pulse">
    <!-- 文本骨架 -->
    <template v-if="variant === 'text'">
      <div
        v-for="i in lines"
        :key="i"
        class="h-4 bg-surface-hover rounded-sm mb-3"
        :class="[
          i === lines && lines > 1 ? 'w-2/3' : 'w-full',
          width,
        ]"
      />
    </template>

    <!-- 矩形骨架 -->
    <div
      v-else-if="variant === 'rect'"
      class="bg-surface-hover rounded-sm"
      :class="[width, height]"
    />

    <!-- 圆形骨架 -->
    <div
      v-else-if="variant === 'circle'"
      class="bg-surface-hover rounded-full"
      :class="[width, height]"
    />

    <!-- 卡片骨架 -->
    <div v-else-if="variant === 'card'" class="space-y-4">
      <div class="h-48 bg-surface-hover rounded-sm" :class="width" />
      <div class="h-5 bg-surface-hover rounded-sm w-3/4" />
      <div class="h-4 bg-surface-hover rounded-sm w-full" />
      <div class="h-4 bg-surface-hover rounded-sm w-5/6" />
    </div>
  </div>
</template>

<script setup lang="ts">
withDefaults(defineProps<{
  variant?: 'text' | 'rect' | 'circle' | 'card'
  lines?: number
  width?: string
  height?: string
}>(), {
  variant: 'text',
  lines: 3,
  width: '',
  height: '',
})
</script>
