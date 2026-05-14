<script setup lang="ts">
/**
 * PublicHeader — 公开页面顶部导航栏
 *
 * 数据：GET /api/v1/site/config → site_title + site_logo（与 Footer 共享 key 去重）
 *
 * 登录态：
 *   未登录 → "登录"按钮
 *   已登录 → 用户头像 + 下拉菜单（管理后台入口 / 退出登录）
 */

const { data: siteData } = useFetch('/api/v1/site/config', {
  key: 'site-config',
  transform: (res: { config: { site_title: string; site_logo: string } }) => res.config,
})

const siteTitle = computed(() => siteData.value?.site_title || 'Ley Blog')
const siteLogo = computed(() => siteData.value?.site_logo || '')

// ---- Auth ----
const auth = useAuthStore()
const userMenuOpen = ref(false)

// ---- 移动端菜单 ----
const mobileOpen = ref(false)
const route = useRoute()
watch(() => route.fullPath, () => { mobileOpen.value = false })

// ---- 搜索 ----
const searchQuery = ref('')
const router = useRouter()
function doSearch() {
  const q = searchQuery.value.trim()
  if (q) { router.push({ path: '/search', query: { q } }); searchQuery.value = '' }
}

// ---- 登出 ----
async function handleLogout() {
  userMenuOpen.value = false
  await auth.logout()
  router.push('/')
}
</script>

<template>
  <header class="sticky top-0 z-50 bg-white/95 backdrop-blur border-b border-gray-100 shadow-sm">
    <div class="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
      <div class="flex items-center h-16 gap-4">

        <!-- Logo + 站名 -->
        <NuxtLink to="/" class="flex items-center gap-2.5 shrink-0">
          <img
            v-if="siteLogo"
            :src="siteLogo" alt="Logo"
            class="h-8 w-8 rounded-lg object-cover"
          />
          <span v-else class="h-8 w-8 rounded-lg bg-blue-600 flex items-center justify-center text-white font-bold text-sm">
            {{ siteTitle.charAt(0) }}
          </span>
          <span class="hidden sm:block text-lg font-bold text-gray-900 tracking-tight">{{ siteTitle }}</span>
        </NuxtLink>

        <!-- 导航 Tab（桌面端） -->
        <nav class="hidden md:flex items-center gap-0.5">
          <NuxtLink
            v-for="tab in [
              { to: '/', label: '首页', exact: true },
              { to: '/articles', label: '文章' },
              { to: '/tags', label: '标签' },
              { to: '/categories', label: '分类' },
            ]"
            :key="tab.to" :to="tab.to"
            class="px-3.5 py-2 text-sm rounded-md font-medium transition-colors"
            :class="(tab.exact ? route.path === tab.to : route.path.startsWith(tab.to))
              ? 'text-blue-600 bg-blue-50' : 'text-gray-600 hover:text-gray-900 hover:bg-gray-100'"
          >
            {{ tab.label }}
          </NuxtLink>
        </nav>

        <div class="flex-1 hidden md:block" />

        <!-- 搜索 + 用户（桌面端） -->
        <div class="hidden md:flex items-center gap-3">
          <form class="relative" @submit.prevent="doSearch">
            <svg class="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-gray-400 pointer-events-none" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
              <path stroke-linecap="round" stroke-linejoin="round" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" />
            </svg>
            <input v-model="searchQuery" type="text" placeholder="搜索..."
              class="w-44 pl-9 pr-3 py-1.5 text-sm border border-gray-200 rounded-full
                     bg-gray-50 focus:bg-white focus:outline-none focus:ring-2 focus:ring-blue-200
                     focus:border-blue-400 transition-all" />
          </form>

          <!-- 未登录 -->
          <NuxtLink v-if="!auth.isLoggedIn" to="/auth/login"
            class="px-4 py-1.5 text-sm font-medium text-white bg-blue-600 rounded-full
                   hover:bg-blue-700 transition-colors shadow-sm">
            登录
          </NuxtLink>

          <!-- 已登录 → 头像 + 下拉 -->
          <div v-else class="relative">
            <button class="flex items-center gap-2 p-1 rounded-full hover:bg-gray-100 transition-colors"
              @click="userMenuOpen = !userMenuOpen">
              <img v-if="auth.user?.avatar" :src="auth.user.avatar" alt=""
                class="h-8 w-8 rounded-full object-cover ring-2 ring-gray-200" />
              <span v-else class="h-8 w-8 rounded-full bg-blue-100 text-blue-600 flex items-center justify-center text-sm font-semibold ring-2 ring-gray-200">
                {{ auth.user?.username?.charAt(0)?.toUpperCase() || 'U' }}
              </span>
              <span class="hidden lg:block text-sm text-gray-700 max-w-[80px] truncate">{{ auth.user?.username }}</span>
            </button>

            <Transition name="fade">
              <div v-if="userMenuOpen"
                class="absolute right-0 mt-2 w-48 bg-white border border-gray-200 rounded-lg shadow-lg py-1 z-50"
                @click="userMenuOpen = false">
                <div class="px-4 py-2 border-b border-gray-100">
                  <p class="text-sm font-medium text-gray-900 truncate">{{ auth.user?.username }}</p>
                  <p class="text-xs text-gray-500 truncate">{{ auth.user?.email }}</p>
                </div>
                <NuxtLink to="/admin" class="flex items-center gap-2 px-4 py-2 text-sm text-gray-700 hover:bg-gray-50">
                  ⚙ 管理后台
                </NuxtLink>
                <button class="flex items-center gap-2 w-full px-4 py-2 text-sm text-red-600 hover:bg-red-50" @click="handleLogout">
                  ↩ 退出登录
                </button>
              </div>
            </Transition>
          </div>
        </div>

        <!-- 移动端汉堡 -->
        <button class="md:hidden ml-auto p-2 rounded-md text-gray-500 hover:bg-gray-100"
          @click="mobileOpen = !mobileOpen" aria-label="菜单">
          <svg v-if="!mobileOpen" class="w-6 h-6" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
            <path stroke-linecap="round" stroke-linejoin="round" d="M4 6h16M4 12h16M4 18h16" />
          </svg>
          <svg v-else class="w-6 h-6" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
            <path stroke-linecap="round" stroke-linejoin="round" d="M6 18L18 6M6 6l12 12" />
          </svg>
        </button>
      </div>

      <!-- 移动端折叠菜单 -->
      <Transition name="slide">
        <div v-if="mobileOpen" class="md:hidden border-t border-gray-100 pt-3 pb-4 space-y-2">
          <NuxtLink to="/" class="block px-3 py-2 text-sm rounded-md" :class="route.path === '/' ? 'text-blue-600 bg-blue-50' : 'text-gray-700'">首页</NuxtLink>
          <NuxtLink to="/articles" class="block px-3 py-2 text-sm rounded-md" :class="route.path.startsWith('/articles') ? 'text-blue-600 bg-blue-50' : 'text-gray-700'">文章</NuxtLink>
          <NuxtLink to="/tags" class="block px-3 py-2 text-sm rounded-md" :class="route.path.startsWith('/tags') ? 'text-blue-600 bg-blue-50' : 'text-gray-700'">标签</NuxtLink>
          <NuxtLink to="/categories" class="block px-3 py-2 text-sm rounded-md" :class="route.path.startsWith('/categories') ? 'text-blue-600 bg-blue-50' : 'text-gray-700'">分类</NuxtLink>
          <div class="border-t border-gray-100 mt-2 pt-2">
            <form class="px-3 mb-2" @submit.prevent="doSearch">
              <input v-model="searchQuery" type="text" placeholder="搜索文章..." class="w-full px-3 py-2 text-sm border rounded-lg" />
            </form>
            <NuxtLink v-if="!auth.isLoggedIn" to="/auth/login" class="block mx-3 px-3 py-2 text-center text-sm font-medium text-white bg-blue-600 rounded-lg">登录</NuxtLink>
            <div v-else class="px-3 space-y-1">
              <p class="text-sm font-medium text-gray-900">{{ auth.user?.username }}</p>
              <NuxtLink to="/admin" class="block text-sm text-gray-600">管理后台</NuxtLink>
              <button class="text-sm text-red-500" @click="handleLogout">退出登录</button>
            </div>
          </div>
        </div>
      </Transition>
    </div>
  </header>
</template>

<style scoped>
.fade-enter-active, .fade-leave-active { transition: opacity 0.15s ease, transform 0.15s ease; }
.fade-enter-from, .fade-leave-to { opacity: 0; transform: translateY(-4px); }
.slide-enter-active, .slide-leave-active { transition: all 0.2s ease; }
.slide-enter-from, .slide-leave-to { opacity: 0; transform: translateY(-8px); }
</style>
