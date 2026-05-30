<!--
  WasTextarea — 和纸文本域

  职责：
  - 多行文本输入，边框呼吸效果
  - 支持 label、placeholder、错误提示

  Props:
    modelValue     string — 绑定值
    label          string — 标签文本
    placeholder    string — 占位提示
    rows           number — 行数
    error          string — 错误提示文本
    disabled       boolean — 是否禁用

  Events:
    update:modelValue   值变化
-->
<template>
  <div class="w-full">
    <!-- Label -->
    <label v-if="label" class="block text-sm text-muted mb-1.5">
      {{ label }}
    </label>

    <textarea
      :value="modelValue"
      :placeholder="placeholder"
      :rows="rows"
      :disabled="disabled"
      class="w-full bg-base text-body placeholder:text-placeholder
             border border-subtle rounded-sm p-3
             focus:outline-none focus:border-accent/60 focus:bg-base/80
             transition-all duration-300
             resize-y disabled:opacity-40"
      :class="error ? 'border-error-subtle' : ''"
      @input="$emit('update:modelValue', ($event.target as HTMLTextAreaElement).value)"
    />

    <!-- 错误提示 -->
    <p v-if="error" class="mt-1.5 text-xs text-error">
      {{ error }}
    </p>
  </div>
</template>

<script setup lang="ts">
withDefaults(defineProps<{
  modelValue?: string
  label?: string
  placeholder?: string
  rows?: number
  error?: string
  disabled?: boolean
}>(), {
  modelValue: '',
  label: '',
  placeholder: '',
  rows: 4,
  error: '',
  disabled: false,
})

defineEmits<{
  'update:modelValue': [value: string]
}>()
</script>
