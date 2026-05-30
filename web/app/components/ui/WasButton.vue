<!--
  WasButton — 和纸按钮

  职责：
  - 提供多种视觉变体的按钮
  - 按压时产生和纸凹陷效果

  Props:
    variant   'primary' | 'secondary' | 'ghost' | 'danger' — 视觉变体
    size      'sm' | 'md' | 'lg' — 尺寸
    disabled  boolean — 是否禁用
    loading   boolean — 是否加载中

  Events:
    click     点击事件
-->
<template>
  <button
    :type="type"
    :disabled="disabled || loading"
    class="inline-flex items-center justify-center select-none transition-all duration-150 ease-out"
    :class="[
      // 尺寸
      sizeClasses[size],
      // 变体
      variantClasses[variant],
      // 禁用态
      (disabled || loading) && 'opacity-50 cursor-not-allowed',
      // 非禁用态的 hover/active 效果
      !(disabled || loading) && 'hover:-translate-y-px active:translate-y-0',
      !(disabled || loading) && 'active:shadow-none',
    ]"
    @click="$emit('click', $event)"
  >
    <!-- 加载指示器 -->
    <span v-if="loading" class="mr-2">
      <svg class="animate-spin w-3.5 h-3.5" fill="none" viewBox="0 0 24 24">
        <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4" />
        <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z" />
      </svg>
    </span>
    <slot />
  </button>
</template>

<script setup lang="ts">
withDefaults(defineProps<{
  variant?: 'primary' | 'secondary' | 'ghost' | 'danger'
  size?: 'sm' | 'md' | 'lg'
  disabled?: boolean
  loading?: boolean
  type?: 'button' | 'submit' | 'reset'
}>(), {
  variant: 'primary',
  size: 'md',
  disabled: false,
  loading: false,
  type: 'button',
})

defineEmits<{
  click: [event: MouseEvent]
}>()

const sizeClasses = {
  sm: 'px-3 py-1.5 text-xs rounded-sm',
  md: 'px-5 py-2.5 text-sm rounded-sm',
  lg: 'px-6 py-3 text-base rounded-md',
}

const variantClasses = {
  primary: [
    'bg-inverted text-inverted',
    'hover:bg-heading',
    'shadow-[inset_0_-2px_0_0_rgba(0,0,0,0.12)]',
    'active:shadow-[inset_0_1px_3px_rgba(0,0,0,0.10)]',
  ],
  secondary: [
    'bg-surface text-body border border-default',
    'hover:bg-surface-hover hover:border-default',
    'shadow-[inset_0_-1px_0_0_rgba(0,0,0,0.06)]',
    'active:shadow-none',
  ],
  ghost: [
    'bg-transparent text-muted',
    'hover:bg-surface-hover hover:text-heading',
  ],
  danger: [
    'bg-error-subtle text-error border border-error-subtle',
    'hover:bg-enji/20',
  ],
}
</script>
