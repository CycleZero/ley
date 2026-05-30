<!--
  管理后台仪表盘 /admin
-->
<template>
  <div>
    <!-- 页面标题 -->
    <div class="mb-8">
      <h1 class="text-2xl font-serif-jp font-semibold text-heading tracking-wider">
        仪表盘
      </h1>
      <p class="mt-1 text-sm text-placeholder">
        欢迎回来，{{ auth.user?.username }}
      </p>
    </div>

    <!-- 统计卡片 -->
    <div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4 mb-8">
      <div class="surface-card p-5">
        <div class="flex items-center justify-between">
          <div>
            <p class="text-xs text-placeholder uppercase tracking-wider">文章</p>
            <p class="mt-1 text-2xl font-serif-jp font-semibold text-heading">
              {{ articleStore.total }}
            </p>
          </div>
          <div class="w-10 h-10 rounded-full bg-accent-subtle flex items-center justify-center">
            <svg class="w-5 h-5 text-accent" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M19 20H5a2 2 0 01-2-2V6a2 2 0 012-2h10a2 2 0 012 2v1m2 13a2 2 0 01-2-2V7m2 13a2 2 0 002-2V9a2 2 0 00-2-2h-2m-4-3H9M7 16h6M7 8h6v4H7V8z" />
            </svg>
          </div>
        </div>
      </div>

      <div class="surface-card p-5">
        <div class="flex items-center justify-between">
          <div>
            <p class="text-xs text-placeholder uppercase tracking-wider">分类</p>
            <p class="mt-1 text-2xl font-serif-jp font-semibold text-heading">
              {{ categoryStore.categories.length }}
            </p>
          </div>
          <div class="w-10 h-10 rounded-full bg-accent-subtle flex items-center justify-center">
            <svg class="w-5 h-5 text-accent" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M3 7v10a2 2 0 002 2h14a2 2 0 002-2V9a2 2 0 00-2-2h-6l-2-2H5a2 2 0 00-2 2z" />
            </svg>
          </div>
        </div>
      </div>

      <div class="surface-card p-5">
        <div class="flex items-center justify-between">
          <div>
            <p class="text-xs text-placeholder uppercase tracking-wider">标签</p>
            <p class="mt-1 text-2xl font-serif-jp font-semibold text-heading">
              {{ tagStore.tags.length }}
            </p>
          </div>
          <div class="w-10 h-10 rounded-full bg-accent-subtle flex items-center justify-center">
            <svg class="w-5 h-5 text-accent" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M7 7h.01M7 3h5c.512 0 1.024.195 1.414.586l7 7a2 2 0 010 2.828l-7 7a2 2 0 01-2.828 0l-7-7A1.994 1.994 0 013 12V7a4 4 0 014-4z" />
            </svg>
          </div>
        </div>
      </div>

      <div class="surface-card p-5">
        <div class="flex items-center justify-between">
          <div>
            <p class="text-xs text-placeholder uppercase tracking-wider">用户</p>
            <p class="mt-1 text-2xl font-serif-jp font-semibold text-heading">
              1
            </p>
          </div>
          <div class="w-10 h-10 rounded-full bg-accent-subtle flex items-center justify-center">
            <svg class="w-5 h-5 text-accent" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M12 4.354a4 4 0 110 5.292M15 21H3v-1a6 6 0 0112 0v1zm0 0h6v-1a6 6 0 00-9-5.197M13 7a4 4 0 11-8 0 4 4 0 018 0z" />
            </svg>
          </div>
        </div>
      </div>
    </div>

    <!-- 快捷操作 -->
    <div class="surface-card p-6 mb-8">
      <h2 class="text-sm font-serif-jp font-semibold text-heading tracking-wider mb-4">
        快捷操作
      </h2>
      <div class="flex flex-wrap gap-3">
        <WasButton variant="primary" @click="navigateTo('/admin/articles/edit/new')">
          <svg class="w-4 h-4 mr-1.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M12 4v16m8-8H4" />
          </svg>
          写文章
        </WasButton>
        <WasButton variant="secondary" @click="navigateTo('/admin/categories')">
          管理分类
        </WasButton>
        <WasButton variant="secondary" @click="navigateTo('/admin/tags')">
          管理标签
        </WasButton>
        <WasButton variant="secondary" @click="navigateTo('/admin/site')">
          站点配置
        </WasButton>
      </div>
    </div>

    <!-- 最近文章 -->
    <div class="surface-card p-6">
      <div class="flex items-center justify-between mb-4">
        <h2 class="text-sm font-serif-jp font-semibold text-heading tracking-wider">
          最近文章
        </h2>
        <NuxtLink
          to="/admin/articles"
          class="text-sm text-accent hover:underline"
        >
          查看全部
        </NuxtLink>
      </div>

      <div v-if="articleStore.loading" class="py-8 flex justify-center">
        <WasLoading />
      </div>

      <div v-else-if="!recentArticles.length" class="py-8 text-center text-placeholder text-sm">
        暂无文章
      </div>

      <div v-else class="overflow-x-auto">
        <table class="w-full text-sm">
          <thead>
            <tr class="border-b border-subtle text-left text-placeholder">
              <th class="pb-3 font-normal">标题</th>
              <th class="pb-3 font-normal w-24">状态</th>
              <th class="pb-3 font-normal w-24 hidden sm:table-cell">浏览</th>
              <th class="pb-3 font-normal w-28 hidden md:table-cell">日期</th>
              <th class="pb-3 font-normal w-20 text-right">操作</th>
            </tr>
          </thead>
          <tbody class="divide-y divide-subtle">
            <tr
              v-for="article in recentArticles"
              :key="article.id"
              class="hover:bg-surface-hover transition-colors"
            >
              <td class="py-3">
                <NuxtLink
                  :to="`/admin/articles/edit/${article.id}`"
                  class="text-body hover:text-accent transition-colors line-clamp-1"
                >
                  {{ article.title }}
                </NuxtLink>
              </td>
              <td class="py-3">
                <span
                  class="inline-flex items-center px-2 py-0.5 text-xs rounded-sm border"
                  :class="statusClass(article.status)"
                >
                  {{ statusText(article.status) }}
                </span>
              </td>
              <td class="py-3 text-placeholder hidden sm:table-cell">
                {{ article.viewCount }}
              </td>
              <td class="py-3 text-placeholder hidden md:table-cell">
                {{ formatDate(article.updatedAt) }}
              </td>
              <td class="py-3 text-right">
                <NuxtLink
                  :to="`/articles/${article.slug}`"
                  target="_blank"
                  class="text-placeholder hover:text-accent transition-colors"
                >
                  <svg class="w-4 h-4 inline" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M2.458 12C3.732 7.943 7.523 5 12 5c4.478 0 8.268 2.943 9.542 7-1.274 4.057-5.064 7-9.542 7-4.477 0-8.268-2.943-9.542-7z" />
                  </svg>
                </NuxtLink>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
definePageMeta({
  layout: 'admin',
  middleware: 'admin',
  pageTransition: false,
})

const auth = useAuthStore()
const articleStore = useArticleStore()
const categoryStore = useCategoryStore()
const tagStore = useTagStore()

// 并行加载数据
await useAsyncData('admin-dashboard', async () => {
  await Promise.all([
    articleStore.fetchArticles({ page: 1, pageSize: 5 }),
    categoryStore.fetchCategories(),
    tagStore.fetchTags(),
  ])
  return true
}, { server: false })

const recentArticles = computed(() => articleStore.articles.slice(0, 5))

// ---------- 工具函数 ----------

function statusClass(status: string) {
  switch (status) {
    case 'published': return 'border-success-subtle text-success bg-success-subtle'
    case 'draft': return 'border-default text-muted bg-surface'
    case 'archived': return 'border-default text-placeholder'
    default: return 'border-default text-muted'
  }
}

function statusText(status: string) {
  switch (status) {
    case 'published': return '已发布'
    case 'draft': return '草稿'
    case 'archived': return '已归档'
    default: return status
  }
}

function formatDate(date: string) {
  const d = new Date(date)
  return `${d.getMonth() + 1}/${d.getDate()}`
}

useHead({
  title: '仪表盘 - 管理后台',
})
</script>
