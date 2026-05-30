<!--
  AppScrollTop — 回到顶部按钮

  职责：
  - 监听页面滚动，超过阈值时显示按钮
  - 点击后平滑滚动到页面顶部

  约束：
  - 自包含，不依赖任何 Store
  - 滚动监听使用 passive 模式提升性能
-->
<template>
  <Transition
    enter-from-class="opacity-0 translate-y-2"
    enter-active-class="transition-all duration-300 ease-out"
    leave-active-class="transition-all duration-200 ease-in"
    leave-to-class="opacity-0 translate-y-2"
  >
    <button
      v-show="isVisible"
      class="fixed bottom-6 right-6 z-40 w-10 h-10 rounded-full bg-base border border-default
             text-muted hover:text-heading hover:border-default
             flex items-center justify-center shadow-subtle
             transition-all duration-200 active:scale-95"
      aria-label="回到顶部"
      @click="scrollToTop"
    >
      <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M5 10l7-7m0 0l7 7m-7-7v18" />
      </svg>
    </button>
  </Transition>
</template>

<script setup lang="ts">
const isVisible = ref(false)

function onScroll() {
  isVisible.value = window.scrollY > 300
}

function scrollToTop() {
  window.scrollTo({ top: 0, behavior: 'smooth' })
}

onMounted(() => {
  window.addEventListener('scroll', onScroll, { passive: true })
  onScroll() // 初始化检查
})

onUnmounted(() => {
  window.removeEventListener('scroll', onScroll)
})
</script>
