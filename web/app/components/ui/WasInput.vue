<!--
  WasInput — 和纸输入框

  职责：
  - 单行文本输入，下划线聚焦展开动画
  - 支持 label、placeholder、错误提示

  Props:
    modelValue     string — 绑定值
    label          string — 标签文本
    placeholder    string — 占位提示
    type           string — 输入类型（text, password, email 等）
    autocomplete   string — 自动完成属性（current-password, new-password, email 等）
    error          string — 错误提示文本
    disabled       boolean — 是否禁用

  Events:
    update:modelValue   值变化
    blur                失去焦点
    focus               获得焦点
-->
<template>
  <div class="w-full">
    <!-- Label -->
    <label v-if="label" class="block text-sm text-muted mb-1.5">
      {{ label }}
    </label>

    <!-- 输入框容器 -->
    <div class="relative">
      <input
        :type="type"
        :value="modelValue"
        :placeholder="placeholder"
        :autocomplete="autocomplete"
        :disabled="disabled"
        class="w-full bg-transparent py-2.5 text-body placeholder:text-placeholder
               border-0 border-b border-default focus:outline-none
               transition-colors duration-400 disabled:opacity-40"
        :class="error ? 'border-error-subtle' : ''"
        @input="$emit('update:modelValue', ($event.target as HTMLInputElement).value)"
        @blur="$emit('blur', $event)"
        @focus="$emit('focus', $event)"
      />

      <!-- 聚焦时下划线从中心展开 -->
      <div
        class="absolute bottom-0 left-1/2 h-px bg-accent transition-all duration-400 ease-out pointer-events-none"
        :class="isFocused ? 'w-full left-0' : 'w-0'"
      />
    </div>

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
  type?: string
  autocomplete?: string
  error?: string
  disabled?: boolean
}>(), {
  modelValue: '',
  label: '',
  placeholder: '',
  type: 'text',
  autocomplete: 'off',
  error: '',
  disabled: false,
})

defineEmits<{
  'update:modelValue': [value: string]
  blur: [event: FocusEvent]
  focus: [event: FocusEvent]
}>()

const isFocused = ref(false)
</script>
