<!--
  AppHeader — 智能顶部导航栏

  职责：
  - 显示站点 Logo 和主导航
  - 根据滚动方向智能显隐
  - 显示登录/用户头像状态（只读 authStore）
  - 移动端汉堡菜单开关（emit 事件给父布局）

  约束：
  - 读取但不修改 authStore（只读 user / isLoggedIn）
  - 通过 emit 触发移动端菜单，不直接操作 uiStore
  - 滚动逻辑封装在 useScrollDirection composable 中
-->
<template>
  <header
    class="fixed top-0 left-0 right-0 z-30 transition-all duration-500 ease-ink"
    :class="[
      isVisible ? 'translate-y-0' : '-translate-y-full',
      scrollY > threshold ? 'bg-base/90 backdrop-blur-md border-b border-subtle' : 'bg-transparent border-transparent',
    ]"
  >
    <div class="max-w-6xl mx-auto px-6 h-16 flex items-center justify-between">
      <!-- Logo -->
      <NuxtLink
        to="/"
        class="font-serif-jp text-lg font-semibold text-heading tracking-wider hover:text-accent transition-colors"
      >
        Ley
      </NuxtLink>

      <!-- 桌面端导航 -->
      <div class="hidden md:flex items-center gap-8">
        <AppNav
          :items="navItems"
          direction="horizontal"
        />

        <!-- 用户区域 -->
        <div class="flex items-center gap-4">
          <template v-if="auth.isLoggedIn">
            <!-- 头像下拉（简化版，点击跳转个人中心） -->
            <NuxtLink
              to="/profile"
              class="flex items-center gap-2 text-sm text-muted hover:text-heading transition-colors"
            >
              <div class="w-7 h-7 rounded-full bg-accent-subtle flex items-center justify-center text-xs font-medium text-accent">
                {{ auth.user?.username?.charAt(0).toUpperCase() || '?' }}
              </div>
              <span class="hidden lg:inline">{{ auth.user?.username }}</span>
            </NuxtLink>
          </template>

          <template v-else>
            <NuxtLink
              to="/login"
              class="text-sm text-muted hover:text-heading transition-colors"
            >
              登录
            </NuxtLink>
          </template>
        </div>
      </div>

      <!-- 移动端汉堡按钮 -->
      <button
        class="md:hidden p-2 text-muted hover:text-heading transition-colors"
        aria-label="打开菜单"
        @click="emit('toggleMobileMenu')"
      >
        <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M4 6h16M4 12h16M4 18h16" />
        </svg>
      </button>
    </div>
  </header>
</template>

<script setup lang="ts">
// 滚动方向检测（封装在 composable 中，保持组件简洁）
const { isVisible, scrollY } = useScrollDirection()
const auth = useAuthStore()

// 导航项配置（纯数据，无状态）
const navItems = [
  { label: '首页', to: '/' },
  { label: '文章', to: '/articles' },
  { label: '标签', to: '/tags' },
  { label: '分类', to: '/categories' },
  { label: '关于', to: '/about' },
]

// 滚动阈值（px），低于此值导航栏背景透明
const threshold = 80

// 事件：通知父布局切换移动端菜单
const emit = defineEmits<{
  toggleMobileMenu: []
}>()
</script>
