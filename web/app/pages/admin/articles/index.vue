<!--
  文章管理页 /admin/articles

  文章列表 + 批量操作入口。
-->
<template>
  <div>
    <!-- 页面标题栏 -->
    <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-4 mb-6">
      <div>
        <h1 class="text-2xl font-serif-jp font-semibold text-heading tracking-wider">
          文章管理
        </h1>
        <p class="mt-1 text-sm text-placeholder">
          共 {{ articleStore.total }} 篇文章
        </p>
      </div>
      <WasButton variant="primary" @click="navigateTo('/admin/articles/edit/new')">
        <svg class="w-4 h-4 mr-1.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M12 4v16m8-8H4" />
        </svg>
        写文章
      </WasButton>
    </div>

    <!-- 筛选栏 -->
    <div class="flex flex-wrap items-center gap-3 mb-4 p-4 surface-card">
      <select
        v-model="statusFilter"
        class="bg-base border border-default text-body text-sm rounded-sm px-3 py-2 focus:outline-none focus:border-accent"
      >
        <option value="">全部状态</option>
        <option value="published">已发布</option>
        <option value="draft">草稿</option>
        <option value="archived">已归档</option>
      </select>

      <div class="flex-1" />

      <WasButton
        variant="outline"
        size="sm"
        :loading="loading"
        @click="refresh"
      >
        刷新
      </WasButton>
    </div>

    <!-- 加载中 -->
    <div v-if="loading" class="py-20 flex justify-center">
      <WasLoading />
    </div>

    <!-- 文章列表 -->
    <div v-else class="surface-card overflow-x-auto">
      <table class="w-full text-sm">
        <thead>
          <tr class="border-b border-subtle text-left text-placeholder">
            <th class="px-4 py-3 font-normal">标题</th>
            <th class="px-4 py-3 font-normal w-28 hidden md:table-cell">分类</th>
            <th class="px-4 py-3 font-normal w-24">状态</th>
            <th class="px-4 py-3 font-normal w-20 hidden sm:table-cell">浏览</th>
            <th class="px-4 py-3 font-normal w-20 hidden lg:table-cell">点赞</th>
            <th class="px-4 py-3 font-normal w-28 hidden md:table-cell">更新日期</th>
            <th class="px-4 py-3 font-normal w-32 text-right">操作</th>
          </tr>
        </thead>
        <tbody class="divide-y divide-subtle">
          <tr
            v-for="article in articles"
            :key="article.id"
            class="hover:bg-surface-hover transition-colors"
          >
            <td class="px-4 py-3">
              <div class="flex items-center gap-2">
                <span v-if="article.isTop" class="text-xs text-warning">置顶</span>
                <NuxtLink
                  :to="`/admin/articles/edit/${article.id}`"
                  class="text-body hover:text-accent transition-colors line-clamp-1"
                >
                  {{ article.title }}
                </NuxtLink>
              </div>
            </td>
            <td class="px-4 py-3 text-placeholder hidden md:table-cell">
              {{ article.category?.name || '-' }}
            </td>
            <td class="px-4 py-3">
              <span
                class="inline-flex items-center px-2 py-0.5 text-xs rounded-sm border"
                :class="statusClass(article.status)"
              >
                {{ statusText(article.status) }}
              </span>
            </td>
            <td class="px-4 py-3 text-placeholder hidden sm:table-cell">
              {{ article.viewCount }}
            </td>
            <td class="px-4 py-3 text-placeholder hidden lg:table-cell">
              {{ article.likeCount }}
            </td>
            <td class="px-4 py-3 text-placeholder hidden md:table-cell">
              {{ formatDate(article.updatedAt) }}
            </td>
            <td class="px-4 py-3 text-right">
              <div class="flex items-center justify-end gap-2">
                <!-- 发布/归档 -->
                <button
                  v-if="article.status === 'draft'"
                  class="text-xs text-success hover:text-success transition-colors"
                  title="发布"
                  @click="publish(article.id)"
                >
                  发布
                </button>
                <button
                  v-else-if="article.status === 'published'"
                  class="text-xs text-placeholder hover:text-muted transition-colors"
                  title="归档"
                  @click="archive(article.id)"
                >
                  归档
                </button>

                <span class="text-subtle">|</span>

                <!-- 查看 -->
                <NuxtLink
                  :to="`/articles/${article.slug}`"
                  target="_blank"
                  class="text-xs text-placeholder hover:text-accent transition-colors"
                >
                  查看
                </NuxtLink>

                <span class="text-subtle">|</span>

                <!-- 删除 -->
                <button
                  class="text-xs text-placeholder hover:text-error transition-colors"
                  @click="confirmDelete(article)"
                >
                  删除
                </button>
              </div>
            </td>
          </tr>
        </tbody>
      </table>

      <!-- 空状态 -->
      <div v-if="!articles.length" class="py-16 text-center text-placeholder text-sm">
        暂无文章
      </div>

      <!-- 分页 -->
      <div v-if="totalPages > 1" class="flex justify-center p-4 border-t border-subtle">
        <WasPagination
          :current-page="page"
          :total-pages="totalPages"
          @change="page = $event"
        />
      </div>
    </div>

    <!-- 删除确认弹窗 -->
    <WasModal
      v-model="deleteModalOpen"
      title="确认删除"
      confirm-text="删除"
      confirm-variant="danger"
      @confirm="doDelete"
    >
      <p class="text-sm text-muted">
        确定要删除文章「{{ articleToDelete?.title }}」吗？此操作不可撤销。
      </p>
    </WasModal>
  </div>
</template>

<script setup lang="ts">
definePageMeta({
  layout: 'admin',
  middleware: 'admin',
  pageTransition: false,
})

const articleStore = useArticleStore()
const ui = useUiStore()

const page = ref(1)
const pageSize = ref(20)
const statusFilter = ref('')
const loading = ref(false)

// 加载文章列表
async function load() {
  loading.value = true
  try {
    await articleStore.fetchArticles({
      page: page.value,
      pageSize: pageSize.value,
      status: statusFilter.value || undefined,
    })
  }
  finally {
    loading.value = false
  }
}

await load()

const articles = computed(() => articleStore.articles)
const totalPages = computed(() => Math.ceil(articleStore.total / pageSize.value) || 1)

watch([page, statusFilter], () => {
  load()
})

// ---------- 操作 ----------

async function publish(id: string) {
  try {
    await articleStore.publishArticle(id)
    ui.toast('文章已发布', 'success')
  }
  catch (e: any) {
    ui.toast(e?.message || '发布失败', 'error')
  }
}

async function archive(id: string) {
  try {
    await articleStore.archiveArticle(id)
    ui.toast('文章已归档', 'success')
  }
  catch (e: any) {
    ui.toast(e?.message || '归档失败', 'error')
  }
}

const deleteModalOpen = ref(false)
const articleToDelete = ref<ArticleInfo | null>(null)

function confirmDelete(article: ArticleInfo) {
  articleToDelete.value = article
  deleteModalOpen.value = true
}

async function doDelete() {
  if (!articleToDelete.value) return
  try {
    await articleStore.deleteArticle(articleToDelete.value.id)
    ui.toast('文章已删除', 'success')
    await load()
  }
  catch (e: any) {
    ui.toast(e?.message || '删除失败', 'error')
  }
  finally {
    deleteModalOpen.value = false
    articleToDelete.value = null
  }
}

function refresh() {
  load()
}

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
  return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}-${String(d.getDate()).padStart(2, '0')}`
}

useHead({
  title: '文章管理 - 管理后台',
})
</script>
