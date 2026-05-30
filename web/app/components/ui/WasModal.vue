<!--
  WasModal — 和纸模态框

  职责：
  - 以日式淡墨风格呈现的模态对话框
  - 支持点击遮罩关闭、ESC 键关闭

  Props:
    modelValue    boolean — 是否显示
    title         string — 标题
    showClose     boolean — 是否显示关闭按钮
    persistent    boolean — 点击遮罩是否不关闭

  Events:
    update:modelValue   显隐变化
    close               关闭事件
-->
<template>
  <Teleport to="body">
    <Transition
      enter-from-class="opacity-0"
      enter-active-class="transition-opacity duration-300"
      leave-active-class="transition-opacity duration-200"
      leave-to-class="opacity-0"
    >
      <div
        v-if="modelValue"
        class="fixed inset-0 z-50 flex items-center justify-center p-4"
        @click="onBackdropClick"
      >
        <!-- 遮罩 -->
        <div class="absolute inset-0 bg-overlay/30 backdrop-blur-sm" />

        <!-- 对话框 -->
        <Transition
          enter-from-class="opacity-0 scale-95"
          enter-active-class="transition-all duration-300 ease-out"
          leave-active-class="transition-all duration-200 ease-in"
          leave-to-class="opacity-0 scale-95"
        >
          <div
            v-if="modelValue"
            class="relative w-full max-w-md bg-base rounded-sm shadow-subtle border border-subtle"
          >
            <!-- 头部 -->
            <div v-if="title || showClose" class="flex items-center justify-between px-6 py-4 border-b border-subtle">
              <h3 v-if="title" class="text-h3 font-serif-jp text-heading">
                {{ title }}
              </h3>
              <button
                v-if="showClose"
                class="p-1 text-placeholder hover:text-body transition-colors ml-auto"
                aria-label="关闭"
                @click="close"
              >
                <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M6 18L18 6M6 6l12 12" />
                </svg>
              </button>
            </div>

            <!-- 内容 -->
            <div class="px-6 py-5">
              <slot />
            </div>

            <!-- 底部操作 -->
            <div v-if="$slots.footer" class="px-6 py-4 border-t border-subtle flex justify-end gap-3">
              <slot name="footer" />
            </div>
          </div>
        </Transition>
      </div>
    </Transition>
  </Teleport>
</template>

<script setup lang="ts">
const props = withDefaults(defineProps<{
  modelValue: boolean
  title?: string
  showClose?: boolean
  persistent?: boolean
}>(), {
  title: '',
  showClose: true,
  persistent: false,
})

const emit = defineEmits<{
  'update:modelValue': [value: boolean]
  close: []
}>()

function close() {
  emit('update:modelValue', false)
  emit('close')
}

function onBackdropClick() {
  if (!props.persistent) {
    close()
  }
}

// ESC 关闭
function onKeydown(e: KeyboardEvent) {
  if (e.key === 'Escape' && props.modelValue) {
    close()
  }
}

onMounted(() => {
  document.addEventListener('keydown', onKeydown)
})

onUnmounted(() => {
  document.removeEventListener('keydown', onKeydown)
})

// 打开时锁定 body 滚动
watch(() => props.modelValue, (val) => {
  if (import.meta.client) {
    document.body.style.overflow = val ? 'hidden' : ''
  }
})
</script>
