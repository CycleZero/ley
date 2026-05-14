<script setup lang="ts">
/**
 * AppButton — 通用按钮组件
 *
 * Props:
 *   variant - 样式变体: primary(蓝底白字) / outline(蓝边蓝字) / danger(红底白字) / ghost(透明)
 *   size    - 尺寸: sm / md / lg
 *   disabled - 禁用状态（置灰 + 禁止点击）
 *   loading  - 加载中（显示旋转动画 + 禁用点击）
 *   type     - HTML button type 属性
 *
 * 事件:
 *   @click  - 点击事件，disabled/loading 时不触发
 */
withDefaults(defineProps<{
  variant?: 'primary' | 'outline' | 'danger' | 'ghost'
  size?: 'sm' | 'md' | 'lg'
  disabled?: boolean
  loading?: boolean
  type?: 'button' | 'submit' | 'reset'
}>(), {
  variant: 'primary',
  size: 'md',
  type: 'button',
})

const emit = defineEmits<{
  click: [e: MouseEvent]
}>()

function handleClick(e: MouseEvent) {
  // disabled/loading 状态下的点击直接忽略
  emit('click', e)
}
</script>

<template>
  <button
    :type="type"
    :disabled="disabled || loading"
    :class="[
      // ---- 基础样式 ----
      'inline-flex items-center justify-center font-medium rounded-lg transition-colors duration-150',
      'focus:outline-none focus:ring-2 focus:ring-offset-2',
      disabled || loading ? 'opacity-50 cursor-not-allowed' : 'cursor-pointer',

      // ---- Variant ----
      {
        'bg-blue-600 text-white hover:bg-blue-700 focus:ring-blue-500': variant === 'primary' && !disabled && !loading,
        'border border-blue-600 text-blue-600 hover:bg-blue-50 focus:ring-blue-500': variant === 'outline' && !disabled && !loading,
        'bg-red-600 text-white hover:bg-red-700 focus:ring-red-500': variant === 'danger' && !disabled && !loading,
        'text-gray-600 hover:text-gray-900 hover:bg-gray-100 focus:ring-gray-400': variant === 'ghost' && !disabled && !loading,
        'bg-blue-400 text-white border-transparent': variant === 'primary' && (disabled || loading),
        'border border-gray-300 text-gray-400': variant === 'outline' && (disabled || loading),
        'bg-red-400 text-white': variant === 'danger' && (disabled || loading),
        'text-gray-400': variant === 'ghost' && (disabled || loading),
      },

      // ---- Size ----
      { 'px-3 py-1.5 text-sm gap-1.5': size === 'sm' },
      { 'px-4 py-2 text-sm gap-2': size === 'md' },
      { 'px-6 py-3 text-base gap-2': size === 'lg' },
    ]"
    @click="handleClick"
  >
    <!-- 加载态：旋转 SVG 动画 -->
    <svg
      v-if="loading"
      class="animate-spin h-4 w-4"
      xmlns="http://www.w3.org/2000/svg"
      fill="none"
      viewBox="0 0 24 24"
    >
      <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4" />
      <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z" />
    </svg>
    <!-- 插槽：按钮文字或图标等 -->
    <slot />
  </button>
</template>
