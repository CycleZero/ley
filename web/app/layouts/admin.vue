<!--
  admin.vue — 管理后台布局

  使用场景：
  - 文章管理、分类管理、标签管理、站点配置等后台页面

  组成：
  - 固定左侧边栏导航
  - 顶部面包屑（预留）
  - 主内容区
  - 移动端侧边栏抽屉
-->
<template>
  <div class="min-h-screen flex bg-base">
    <!-- 桌面端侧边栏 -->
    <aside class="hidden lg:flex flex-col w-56 border-r border-subtle bg-base fixed inset-y-0 left-0">
      <!-- Logo -->
      <div class="h-16 flex items-center px-6 border-b border-subtle">
        <NuxtLink to="/" class="font-serif-jp text-lg font-semibold text-heading tracking-wider">
          Ley
        </NuxtLink>
        <span class="ml-2 text-xs text-placeholder">管理后台</span>
      </div>

      <!-- 导航 -->
      <nav class="flex-1 p-4 space-y-1 overflow-y-auto">
        <NuxtLink
          v-for="item in adminNavItems"
          :key="item.to"
          :to="item.to"
          class="flex items-center gap-3 px-3 py-2.5 text-sm rounded-sm transition-colors"
          :class="[
            isActiveNav(item.to)
              ? 'bg-surface-hover text-heading font-medium'
              : 'text-muted hover:bg-surface hover:text-body',
          ]"
        >
          <!-- 图标 -->
          <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" :d="item.icon" />
          </svg>
          {{ item.label }}
        </NuxtLink>
      </nav>

      <!-- 底部：返回前台 -->
      <div class="p-4 border-t border-subtle">
        <NuxtLink
          to="/"
          class="flex items-center gap-2 text-sm text-placeholder hover:text-body transition-colors"
        >
          <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M10 19l-7-7m0 0l7-7m-7 7h18" />
          </svg>
          返回前台
        </NuxtLink>
      </div>
    </aside>

    <!-- 移动端顶部栏 + 抽屉 -->
    <div class="lg:hidden fixed top-0 left-0 right-0 z-30 h-14 bg-base border-b border-subtle flex items-center px-4">
      <button
        class="p-2 text-muted hover:text-heading transition-colors"
        aria-label="打开菜单"
        @click="mobileMenuOpen = true"
      >
        <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M4 6h16M4 12h16M4 18h16" />
        </svg>
      </button>
      <span class="ml-3 font-serif-jp text-base font-semibold text-heading">Ley 管理后台</span>
    </div>

    <!-- 移动端抽屉 -->
    <Transition
      enter-from-class="opacity-0"
      enter-active-class="transition-opacity duration-300"
      leave-active-class="transition-opacity duration-300"
      leave-to-class="opacity-0"
    >
      <div
        v-if="mobileMenuOpen"
        class="fixed inset-0 z-40 bg-overlay/20 backdrop-blur-sm lg:hidden"
        @click="mobileMenuOpen = false"
      />
    </Transition>

    <Transition
      enter-from-class="-translate-x-full"
      enter-active-class="transition-transform duration-300 ease-out"
      leave-active-class="transition-transform duration-200 ease-in"
      leave-to-class="-translate-x-full"
    >
      <aside
        v-if="mobileMenuOpen"
        class="fixed top-0 left-0 bottom-0 w-56 z-50 bg-base border-r border-subtle flex flex-col lg:hidden"
      >
        <div class="h-14 flex items-center justify-between px-4 border-b border-subtle">
          <span class="font-serif-jp text-base font-semibold text-heading">管理后台</span>
          <button class="p-1 text-placeholder hover:text-body" @click="mobileMenuOpen = false">
            <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M6 18L18 6M6 6l12 12" />
            </svg>
          </button>
        </div>

        <nav class="flex-1 p-4 space-y-1 overflow-y-auto">
          <NuxtLink
            v-for="item in adminNavItems"
            :key="item.to"
            :to="item.to"
            class="flex items-center gap-3 px-3 py-2.5 text-sm rounded-sm transition-colors"
            :class="[
              isActiveNav(item.to)
                ? 'bg-surface-hover text-heading font-medium'
                : 'text-muted hover:bg-surface hover:text-body',
            ]"
            @click="mobileMenuOpen = false"
          >
            <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" :d="item.icon" />
            </svg>
            {{ item.label }}
          </NuxtLink>
        </nav>

        <div class="p-4 border-t border-subtle">
          <NuxtLink to="/" class="flex items-center gap-2 text-sm text-placeholder hover:text-body" @click="mobileMenuOpen = false">
            <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M10 19l-7-7m0 0l7-7m-7 7h18" />
            </svg>
            返回前台
          </NuxtLink>
        </div>
      </aside>
    </Transition>

    <!-- 主内容区 -->
    <main class="flex-1 lg:ml-56 min-h-screen">
      <!-- 移动端顶部占位 -->
      <div class="h-14 lg:hidden" />
      <div class="p-6 lg:p-8 max-w-6xl">
        <slot />
      </div>
    </main>
  </div>
</template>

<script setup lang="ts">
const route = useRoute()

// 导航项激活判断（/admin 根路径只精确匹配，避免子页面也高亮）
function isActiveNav(to: string) {
  if (to === '/admin') {
    return route.path === '/admin'
  }
  return route.path === to || route.path.startsWith(to + '/')
}

// 移动端菜单状态
const mobileMenuOpen = ref(false)

// 后台导航项
const adminNavItems = [
  {
    label: '仪表盘',
    to: '/admin',
    icon: 'M4 6a2 2 0 012-2h2a2 2 0 012 2v2a2 2 0 01-2 2H6a2 2 0 01-2-2V6zM14 6a2 2 0 012-2h2a2 2 0 012 2v2a2 2 0 01-2 2h-2a2 2 0 01-2-2V6zM4 16a2 2 0 012-2h2a2 2 0 012 2v2a2 2 0 01-2 2H6a2 2 0 01-2-2v-2zM14 16a2 2 0 012-2h2a2 2 0 012 2v2a2 2 0 01-2 2h-2a2 2 0 01-2-2v-2z',
  },
  {
    label: '文章管理',
    to: '/admin/articles',
    icon: 'M19 20H5a2 2 0 01-2-2V6a2 2 0 012-2h10a2 2 0 012 2v1m2 13a2 2 0 01-2-2V7m2 13a2 2 0 002-2V9a2 2 0 00-2-2h-2m-4-3H9M7 16h6M7 8h6v4H7V8z',
  },

  {
    label: '分类管理',
    to: '/admin/categories',
    icon: 'M3 7v10a2 2 0 002 2h14a2 2 0 002-2V9a2 2 0 00-2-2h-6l-2-2H5a2 2 0 00-2 2z',
  },
  {
    label: '标签管理',
    to: '/admin/tags',
    icon: 'M7 7h.01M7 3h5c.512 0 1.024.195 1.414.586l7 7a2 2 0 010 2.828l-7 7a2 2 0 01-2.828 0l-7-7A1.994 1.994 0 013 12V7a4 4 0 014-4z',
  },
  {
    label: '文件管理',
    to: '/admin/files',
    icon: 'M4 16l4.586-4.586a2 2 0 012.828 0L16 16m-2-2l1.586-1.586a2 2 0 012.828 0L20 14m-6-6h.01M6 20h12a2 2 0 002-2V6a2 2 0 00-2-2H6a2 2 0 00-2 2v12a2 2 0 002 2z',
  },
  {
    label: '站点配置',
    to: '/admin/site',
    icon: 'M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z M15 12a3 3 0 11-6 0 3 3 0 016 0z',
  },
]

// ESC 关闭移动端菜单
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
