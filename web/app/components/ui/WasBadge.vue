<!--
  WasBadge — 徽标

  职责：
  - 显示小圆点或数字徽标
  - 用于标记新消息、状态指示

  Props:
    variant   'default' | 'ai' | 'enji' | 'matcha' — 颜色
    dot       boolean — 是否只显示圆点
    count     number | string — 数字内容
    max       number — 最大显示数字，超过显示 {max}+
-->
<template>
  <span
    class="inline-flex items-center justify-center text-xs font-medium"
    :class="[
      dot ? 'w-2 h-2 rounded-full' : 'min-w-[1.25rem] h-5 px-1 rounded-full',
      variantClasses[variant],
    ]"
  >
    <template v-if="!dot">
      {{ displayCount }}
    </template>
  </span>
</template>

<script setup lang="ts">
import { computed } from 'vue'

const props = withDefaults(defineProps<{
  variant?: 'default' | 'ai' | 'enji' | 'matcha'
  dot?: boolean
  count?: number | string
  max?: number
}>(), {
  variant: 'default',
  dot: false,
  count: 0,
  max: 99,
})

const displayCount = computed(() => {
  const num = typeof props.count === 'string' ? parseInt(props.count, 10) : props.count
  if (num > props.max) return `${props.max}+`
  return String(num)
})

const variantClasses = {
  default: 'bg-surface-hover text-body',
  ai:      'bg-accent-subtle text-accent',
  enji:    'bg-error-subtle text-error',
  matcha:  'bg-success-subtle text-success',
}
</script>
