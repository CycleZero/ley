<!--
  WasEmpty — 空状态

  职责：
  - 数据为空时的占位提示
  - 枯山水 SVG 插画 + 提示文字 + 可选操作按钮

  Props:
    title       string — 空状态标题
    description string — 描述文字
    icon        string — 自定义图标名称（Lucide）

  Slots:
    action      底部操作区域
-->
<template>
  <div class="flex flex-col items-center justify-center py-ma-6 text-center">
    <!-- 枯山水 SVG 插画 -->
    <svg
      class="w-24 h-24 text-placeholder mb-4"
      viewBox="0 0 96 96"
      fill="none"
      xmlns="http://www.w3.org/2000/svg"
    >
      <!-- 远山 -->
      <path d="M16 64c8-16 16-24 24-24s16 8 24 24" stroke="currentColor" stroke-width="1" opacity="0.4" />
      <path d="M48 52c6-12 12-18 18-18s12 6 18 18" stroke="currentColor" stroke-width="1" opacity="0.3" />
      <!-- 沙纹 -->
      <path d="M8 72h80" stroke="currentColor" stroke-width="0.5" opacity="0.2" />
      <path d="M12 76h72" stroke="currentColor" stroke-width="0.5" opacity="0.15" />
      <path d="M16 80h64" stroke="currentColor" stroke-width="0.5" opacity="0.1" />
      <!-- 石组 -->
      <circle cx="32" cy="68" r="4" fill="currentColor" opacity="0.25" />
      <circle cx="40" cy="66" r="3" fill="currentColor" opacity="0.2" />
      <circle cx="72" cy="70" r="5" fill="currentColor" opacity="0.3" />
    </svg>

    <!-- 标题 -->
    <h3 v-if="title" class="text-h3 font-serif-jp text-muted mb-2">
      {{ title }}
    </h3>

    <!-- 描述 -->
    <p v-if="description" class="text-sm text-placeholder max-w-xs">
      {{ description }}
    </p>

    <!-- 操作按钮 -->
    <div v-if="$slots.action" class="mt-6">
      <slot name="action" />
    </div>
  </div>
</template>

<script setup lang="ts">
withDefaults(defineProps<{
  title?: string
  description?: string
}>(), {
  title: '暂无内容',
  description: '',
})
</script>
