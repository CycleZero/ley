<!--
  WasDropdown — 和纸下拉菜单

  职责：
  - 触发器点击后展开下拉选项
  - 点击外部或 ESC 键自动关闭

  Props:
    trigger       'click' | 'hover' — 触发方式
    placement     'bottom-left' | 'bottom-right' | 'top-left' | 'top-right' — 位置

  Slots:
    trigger       触发器内容（必须）
    default       下拉菜单内容
-->
<template>
  <div ref="dropdownRef" class="relative inline-block">
    <!-- 触发器 -->
    <div
      @click="toggle"
      @mouseenter="trigger === 'hover' && (isOpen = true)"
      @mouseleave="trigger === 'hover' && startClose()"
    >
      <slot name="trigger" />
    </div>

    <!-- 下拉菜单 -->
    <Transition
      enter-from-class="opacity-0 scale-95"
      enter-active-class="transition-all duration-200 ease-out"
      leave-active-class="transition-all duration-150 ease-in"
      leave-to-class="opacity-0 scale-95"
    >
      <div
        v-if="isOpen"
        class="absolute z-40 min-w-[8rem] bg-base border border-subtle rounded-sm shadow-subtle py-1"
        :class="placementClasses[placement]"
        @mouseenter="trigger === 'hover' && cancelClose()"
        @mouseleave="trigger === 'hover' && startClose()"
      >
        <slot />
      </div>
    </Transition>
  </div>
</template>

<script setup lang="ts">
withDefaults(defineProps<{
  trigger?: 'click' | 'hover'
  placement?: 'bottom-left' | 'bottom-right' | 'top-left' | 'top-right'
}>(), {
  trigger: 'click',
  placement: 'bottom-left',
})

const isOpen = ref(false)
const dropdownRef = ref<HTMLElement | null>(null)
let closeTimer: ReturnType<typeof setTimeout> | null = null

function toggle() {
  isOpen.value = !isOpen.value
}

function close() {
  isOpen.value = false
}

function startClose() {
  closeTimer = setTimeout(() => {
    isOpen.value = false
  }, 150)
}

function cancelClose() {
  if (closeTimer) {
    clearTimeout(closeTimer)
    closeTimer = null
  }
}

// 点击外部关闭
function onClickOutside(event: MouseEvent) {
  if (dropdownRef.value && !dropdownRef.value.contains(event.target as Node)) {
    close()
  }
}

// ESC 关闭
function onKeydown(event: KeyboardEvent) {
  if (event.key === 'Escape') {
    close()
  }
}

onMounted(() => {
  document.addEventListener('click', onClickOutside)
  document.addEventListener('keydown', onKeydown)
})

onUnmounted(() => {
  document.removeEventListener('click', onClickOutside)
  document.removeEventListener('keydown', onKeydown)
  if (closeTimer) clearTimeout(closeTimer)
})

const placementClasses = {
  'bottom-left': 'top-full left-0 mt-1',
  'bottom-right': 'top-full right-0 mt-1',
  'top-left': 'bottom-full left-0 mb-1',
  'top-right': 'bottom-full right-0 mb-1',
}
</script>
