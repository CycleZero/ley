<!--
  default.vue — 默认布局

  使用场景：
  - 首页、文章列表页、标签页、分类页、关于页等常规页面

  组成：
  - 智能导航栏（滚动显隐）
  - 移动端侧边抽屉菜单
  - 主内容区（<slot />）
  - 页脚
  - 回到顶部按钮
-->
<template>
  <div class="min-h-screen flex flex-col bg-base">
    <!-- 顶部导航 -->
    <AppHeader @toggle-mobile-menu="mobileMenuOpen = !mobileMenuOpen" />

    <!-- 移动端侧边菜单遮罩 -->
    <Transition
      enter-from-class="opacity-0"
      enter-active-class="transition-opacity duration-300"
      leave-active-class="transition-opacity duration-300"
      leave-to-class="opacity-0"
    >
      <div
        v-if="mobileMenuOpen"
        class="fixed inset-0 z-40 bg-overlay/20 backdrop-blur-sm md:hidden"
        @click="mobileMenuOpen = false"
      />
    </Transition>

    <!-- 移动端侧边菜单 -->
    <Transition
      enter-from-class="translate-x-full"
      enter-active-class="transition-transform duration-300 ease-out"
      leave-active-class="transition-transform duration-200 ease-in"
      leave-to-class="translate-x-full"
    >
      <aside
        v-if="mobileMenuOpen"
        class="fixed top-0 right-0 bottom-0 w-64 z-50 bg-base border-l border-subtle p-6 md:hidden"
      >
        <!-- 关闭按钮 -->
        <div class="flex justify-end mb-6">
          <button
            class="p-1 text-placeholder hover:text-body transition-colors"
            aria-label="关闭菜单"
            @click="mobileMenuOpen = false"
          >
            <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M6 18L18 6M6 6l12 12" />
            </svg>
          </button>
        </div>

        <!-- 移动端导航 -->
        <AppNav
          :items="mobileNavItems"
          direction="vertical"
          class="flex-col gap-4"
        />

        <!-- 移动端用户入口 -->
        <div class="mt-8 pt-6 border-t border-subtle">
          <template v-if="auth.isLoggedIn">
            <NuxtLink
              to="/profile"
              class="flex items-center gap-2 text-sm text-muted hover:text-heading transition-colors"
              @click="mobileMenuOpen = false"
            >
              <div class="w-8 h-8 rounded-full bg-accent-subtle flex items-center justify-center text-sm font-medium text-accent">
                {{ auth.user?.username?.charAt(0).toUpperCase() || '?' }}
              </div>
              <span>{{ auth.user?.username }}</span>
            </NuxtLink>
          </template>
          <template v-else>
            <NuxtLink
              to="/login"
              class="text-sm text-muted hover:text-heading transition-colors"
              @click="mobileMenuOpen = false"
            >
              登录
            </NuxtLink>
          </template>
        </div>
      </aside>
    </Transition>

    <!-- 主内容区 -->
    <main class="flex-1 pt-16">
      <slot />
    </main>

    <!-- 页脚 -->
    <AppFooter />

    <!-- 回到顶部 -->
    <AppScrollTop />
  </div>
</template>

<script setup lang="ts">
const auth = useAuthStore()

// 移动端菜单状态（布局级状态，不放入 uiStore，避免过度耦合）
const mobileMenuOpen = ref(false)

// 移动端导航项（与桌面端保持一致）
const mobileNavItems = [
  { label: '首页', to: '/' },
  { label: '文章', to: '/articles' },
  { label: '标签', to: '/tags' },
  { label: '分类', to: '/categories' },
  { label: '关于', to: '/about' },
]

// ESC 键关闭移动端菜单
function onKeydown(e: KeyboardEvent) {
  if (e.key === 'Escape' && mobileMenuOpen.value) {
    mobileMenuOpen.value = false
  }
}

onMounted(() => {
  document.addEventListener('keydown', onKeydown)
})

onUnmounted(() => {
  document.removeEventListener('keydown', onKeydown)
})
</script>
