<!--
  WasTag — 和纸标签

  职责：
  - 显示标签文本，支持可删除模式
  - 悬停时填充背景，如和纸上的印章

  Props:
    label       string — 标签文本
    removable   boolean — 是否显示删除按钮
    variant     'default' | 'ai' | 'enji' | 'matcha' | 'kiiro' — 颜色变体

  Events:
    click       点击事件
    remove      删除事件
-->
<template>
  <span
    class="inline-flex items-center gap-1.5 px-2.5 py-1 text-xs rounded-sm
           border transition-all duration-200 cursor-pointer select-none"
    :class="[
      variantClasses[variant],
      removable && 'pr-1.5',
    ]"
    @click="$emit('click', $event)"
  >
    {{ label }}

    <!-- 删除按钮 -->
    <button
      v-if="removable"
      class="ml-0.5 p-0.5 rounded-sm hover:bg-surface-hover/30 transition-colors"
      aria-label="移除标签"
      @click.stop="$emit('remove', $event)"
    >
      <svg class="w-3 h-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
      </svg>
    </button>
  </span>
</template>

<script setup lang="ts">
withDefaults(defineProps<{
  label: string
  removable?: boolean
  variant?: 'default' | 'ai' | 'enji' | 'matcha' | 'kiiro'
}>(), {
  removable: false,
  variant: 'default',
})

defineEmits<{
  click: [event: MouseEvent]
  remove: [event: MouseEvent]
}>()

const variantClasses = {
  default: 'border-default text-muted bg-transparent hover:bg-surface-hover',
  ai:      'border-accent-subtle text-accent bg-accent-subtle hover:bg-accent/10',
  enji:    'border-error-subtle text-error bg-error-subtle hover:bg-error/10',
  matcha:  'border-success-subtle text-success bg-success-subtle hover:bg-success/10',
  kiiro:   'border-warning/30 text-warning bg-warning/5 hover:bg-warning/10',
}
</script>
