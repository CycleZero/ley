<script setup lang="ts">
/**
 * AppInput — 通用输入框组件
 *
 * 与 VeeValidate 配合：通过 v-bind="$attrs" 透传 useField 绑定的属性
 * （name / modelValue / onBlur / onChange 等），使组件可用于 VeeValidate
 * useForm({ validationSchema }) 校验流程。
 *
 * Props:
 *   label       - 标签文本（空字符串则不显示）
 *   error       - 错误消息（显示在输入框下方红色文本）
 *   placeholder - 占位文本
 *   type        - HTML input type（text / email / password / url / number）
 *   disabled    - 禁用输入
 */
withDefaults(defineProps<{
  label?: string
  error?: string
  type?: string
  disabled?: boolean
}>(), {
  type: 'text',
})

// v-model 双向绑定
const model = defineModel<string>({ default: '' })
</script>

<template>
  <div>
    <!-- 标签行 -->
    <label v-if="label" class="block text-sm font-medium text-gray-700 mb-1">
      {{ label }}
    </label>

    <!-- 输入框 -->
    <input
      v-model="model"
      :type="type"
      :disabled="disabled"
      v-bind="$attrs"
      :class="[
        'block w-full rounded-lg border px-3 py-2 text-sm transition-colors duration-150',
        'focus:outline-none focus:ring-2 focus:ring-offset-0',
        error
          ? 'border-red-300 focus:border-red-500 focus:ring-red-200'
          : 'border-gray-300 focus:border-blue-500 focus:ring-blue-200',
        disabled ? 'bg-gray-50 text-gray-400 cursor-not-allowed' : 'bg-white text-gray-900',
      ]"
    />

    <!-- 错误消息 -->
    <p v-if="error" class="mt-1 text-sm text-red-600">{{ error }}</p>
  </div>
</template>
