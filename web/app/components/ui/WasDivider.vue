<!--
  WasDivider — 和纸分割线

  职责：
  - 三种风格的分割线：实线、虚线、花结装饰

  Props:
    variant   'solid' | 'dashed' | 'flower' — 分割线风格
    label     string — 花结分割线中间的文字
    spacing   'sm' | 'md' | 'lg' — 上下间距
-->
<template>
  <div class="flex items-center" :class="spacingClasses[spacing]">
    <!-- 实线/虚线 -->
    <template v-if="variant !== 'flower'">
      <div
        class="w-full"
        :class="[
          variant === 'solid' ? 'h-px bg-surface-hover' : '',
          variant === 'dashed' ? 'border-t border-dashed border-default' : '',
        ]"
      />
    </template>

    <!-- 花结分割线 -->
    <template v-else>
      <div class="flex-1 h-px bg-surface-hover" />
      <span
        v-if="label"
        class="px-3 text-xs text-placeholder tracking-wider whitespace-nowrap"
      >
        {{ label }}
      </span>
      <span
        v-else
        class="px-3 text-placeholder select-none"
      >
        ❧
      </span>
      <div class="flex-1 h-px bg-surface-hover" />
    </template>
  </div>
</template>

<script setup lang="ts">
withDefaults(defineProps<{
  variant?: 'solid' | 'dashed' | 'flower'
  label?: string
  spacing?: 'sm' | 'md' | 'lg'
}>(), {
  variant: 'solid',
  label: '',
  spacing: 'md',
})

const spacingClasses = {
  sm: 'my-3',
  md: 'my-6',
  lg: 'my-10',
}
</script>
