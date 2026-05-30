<!--
  WasLoading — 墨滴加载动画

  职责：
  - 以日式水墨风格呈现加载状态
  - 墨滴下落 + 落点晕染效果

  Props:
    size      'sm' | 'md' | 'lg' — 尺寸
    text      string — 加载提示文字
-->
<template>
  <div class="flex flex-col items-center justify-center">
    <div class="relative" :class="sizeClasses[size].container">
      <!-- 墨滴下落 -->
      <div
        class="absolute top-0 left-1/2 -translate-x-1/2 rounded-full bg-heading animate-ink-drop"
        :class="sizeClasses[size].drop"
      />
      <!-- 落点晕染 -->
      <div
        class="absolute left-1/2 -translate-x-1/2 rounded-full bg-heading/20 animate-ink-spread"
        :class="sizeClasses[size].spread"
      />
    </div>
    <span v-if="text" class="text-placeholder tracking-widest" :class="sizeClasses[size].text">
      {{ text }}
    </span>
  </div>
</template>

<script setup lang="ts">
withDefaults(defineProps<{
  size?: 'sm' | 'md' | 'lg'
  text?: string
}>(), {
  size: 'md',
  text: '加载中',
})

const sizeClasses = {
  sm: {
    container: 'w-5 h-8',
    drop: 'w-1.5 h-1.5',
    spread: 'bottom-0 w-3 h-1',
    text: 'text-xs mt-2',
  },
  md: {
    container: 'w-8 h-12',
    drop: 'w-2 h-2',
    spread: 'bottom-0 w-6 h-1.5',
    text: 'text-sm mt-3',
  },
  lg: {
    container: 'w-12 h-16',
    drop: 'w-3 h-3',
    spread: 'bottom-0 w-9 h-2',
    text: 'text-base mt-4',
  },
}
</script>
