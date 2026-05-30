<!--
  文章列表页 /articles

  支持按分类、标签筛选。
  路由：/articles?category=xxx&tag=xxx
  布局：default.vue
-->
<template>
  <div class="max-w-6xl mx-auto px-6 py-ma-6">
    <!-- 页面标题 -->
    <FadeIn direction="up" :delay="0">
      <h1 class="text-h1 font-serif-jp text-heading mb-2">
        {{ pageTitle }}
      </h1>
      <p class="text-placeholder mb-ma-6">{{ pageSubtitle }}</p>
    </FadeIn>

    <!-- 筛选器 -->
    <FadeIn direction="up" :delay="0.1">
      <div v-if="hasActiveFilter" class="flex flex-wrap items-center gap-3 mb-8 pb-6 border-b border-subtle">
        <WasTag
          :label="activeFilterLabel"
          variant="ai"
          removable
          @remove="clearFilter"
        />
      </div>
    </FadeIn>

    <!-- 加载中 -->
    <div v-if="pending" class="flex justify-center py-20">
      <WasLoading />
    </div>

    <!-- 文章列表 -->
    <FadeIn v-else direction="up" :delay="0.2">
      <ArticleList
        :articles="articles"
        :columns="2"
        :show-stagger="true"
        :empty-title="emptyTitle"
        :empty-desc="emptyDesc"
        @article-click="goToArticle"
        @category-click="goToCategory"
      />
    </FadeIn>

    <!-- 分页 -->
    <div v-if="totalPages > 1" class="flex justify-center mt-ma-6">
      <WasPagination
        :current-page="page"
        :total-pages="totalPages"
        @change="onPageChange"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
const route = useRoute()
const articleStore = useArticleStore()
const categoryStore = useCategoryStore()
const tagStore = useTagStore()

// ---------- Query 参数 ----------

const categoryId = computed(() => route.query.category as string | undefined)
const tagId = computed(() => route.query.tag as string | undefined)

const hasActiveFilter = computed(() => !!categoryId.value || !!tagId.value)

const activeFilterLabel = computed(() => {
  if (categoryId.value) {
    const cat = categoryStore.findById(categoryId.value)
    return `分类: ${cat?.name || categoryId.value}`
  }
  if (tagId.value) {
    const tag = tagStore.tags.find(t => t.id === tagId.value || t.slug === tagId.value)
    return `标签: ${tag?.name || tagId.value}`
  }
  return ''
})

// ---------- 数据获取 ----------

const page = ref(1)
const pageSize = ref(10)

const fetchParams = computed(() => {
  const params: any = {
    page: page.value,
    pageSize: pageSize.value,
  }
  if (categoryId.value) {
    params.categoryId = categoryId.value
  }
  if (tagId.value) {
    params.tags = [tagId.value]
  }
  return params
})

const { pending } = await useAsyncData(
  () => `articles-list-${categoryId.value || ''}-${tagId.value || ''}-${page.value}`,
  () => articleStore.fetchArticles(fetchParams.value),
  { server: true, watch: [page, categoryId, tagId] },
)

const articles = computed(() => articleStore.articles)
const totalPages = computed(() => Math.ceil(articleStore.total / pageSize.value) || 1)

// ---------- 页面标题 ----------

const pageTitle = computed(() => {
  if (categoryId.value) {
    const cat = categoryStore.findById(categoryId.value)
    return cat?.name || '文章列表'
  }
  if (tagId.value) {
    const tag = tagStore.tags.find(t => t.id === tagId.value || t.slug === tagId.value)
    return tag?.name || '文章列表'
  }
  return '文章'
})

const pageSubtitle = computed(() => {
  if (categoryId.value) return '按分类筛选的文章'
  if (tagId.value) return '按标签筛选的文章'
  return '全部文章列表'
})

const emptyTitle = computed(() => {
  if (hasActiveFilter.value) return '暂无相关文章'
  return '暂无文章'
})

const emptyDesc = computed(() => {
  if (hasActiveFilter.value) return '该筛选条件下没有文章'
  return '还没有发布任何文章'
})

useHead(() => ({
  title: pageTitle.value,
}))

// ---------- 方法 ----------

function goToArticle(slug: string) {
  navigateTo(`/articles/${slug}`)
}

function goToCategory(slug: string) {
  // 预留：文章卡片点击分类时的处理
  // 当前跳转到文章列表并带分类参数
  const cat = categoryStore.categories.find(c => c.slug === slug)
  if (cat) {
    navigateTo(`/articles?category=${cat.id}`)
  }
}

function onPageChange(newPage: number) {
  page.value = newPage
}

function clearFilter() {
  navigateTo('/articles')
}

// 客户端兜底：nuxt generate 下 payload 恢复不执行副作用
onMounted(() => {
  if (!articleStore.articles.length) {
    articleStore.fetchArticles(fetchParams.value)
  }
})

// 当筛选条件变化时，重置到第一页
watch([categoryId, tagId], () => {
  page.value = 1
})
</script>
