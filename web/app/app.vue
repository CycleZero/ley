<template>
  <div class="min-h-screen bg-base text-heading">
    <!-- 全局 Toast 容器 -->
    <div
      v-if="ui.toasts.length"
      class="fixed top-4 right-4 z-50 flex flex-col gap-2"
    >
      <TransitionGroup
        enter-from-class="opacity-0 translate-x-4"
        enter-active-class="transition-all duration-300 ease-out"
        leave-active-class="transition-all duration-200 ease-in"
        leave-to-class="opacity-0 translate-x-4"
      >
        <div
          v-for="item in ui.toasts"
          :key="item.id"
          class="px-4 py-3 rounded-sm text-sm shadow-subtle border"
          :class="toastClass(item.type)"
        >
          {{ item.message }}
        </div>
      </TransitionGroup>
    </div>

    <!-- 全局加载指示器 -->
    <div
      v-if="ui.isGlobalLoading"
      class="fixed inset-0 z-40 flex items-center justify-center bg-base/60 backdrop-blur-sm"
    >
      <div class="flex flex-col items-center gap-3">
        <div class="relative w-8 h-12">
          <div class="absolute top-0 left-1/2 -translate-x-1/2 w-2 h-2 rounded-full bg-heading animate-ink-drop" />
          <div class="absolute bottom-0 left-1/2 -translate-x-1/2 w-6 h-1.5 rounded-full bg-heading/20 animate-ink-spread" />
        </div>
        <span class="text-sm text-placeholder tracking-widest">加载中</span>
      </div>
    </div>

    <!-- 路由出口 -->
    <NuxtLayout>
      <NuxtPage />
    </NuxtLayout>
  </div>
</template>

<script setup lang="ts">
const ui = useUiStore()
const auth = useAuthStore()

// 初始化：恢复登录态
onMounted(() => {
  auth.init()
})

function toastClass(type: string) {
  switch (type) {
    case 'error':
      return 'bg-error-subtle border-error-subtle text-error'
    case 'success':
      return 'bg-success-subtle border-success-subtle text-success'
    case 'info':
    default:
      return 'bg-accent-subtle border-accent-subtle text-accent'
  }
}
</script>
