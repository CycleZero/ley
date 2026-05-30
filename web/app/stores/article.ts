/**
 * 文章状态管理
 *
 * 提供文章列表缓存、当前文章、草稿自动保存（预留）等功能。
 * 评论功能已预留扩展接口，暂不实现。
 */
export const useArticleStore = defineStore('article', () => {
  const api = useArticleApi()

  // ---------- State ----------

  /** 文章列表 */
  const articles = ref<ArticleInfo[]>([])
  /** 总条数 */
  const total = ref(0)
  /** 当前页码 */
  const page = ref(1)
  /** 每页条数 */
  const pageSize = ref(10)
  /** 是否加载中 */
  const loading = ref(false)

  /** 当前文章详情 */
  const currentArticle = ref<ArticleInfo | null>(null)

  /** 编辑中的草稿 */
  const draft = ref<Partial<CreateArticleRequest>>({
    title: '',
    content: '',
    excerpt: '',
    coverImage: '',
    categoryId: '',
    tagNames: [],
  })

  /** 搜索关键词 */
  const searchKeyword = ref('')
  /** 搜索结果 */
  const searchResults = ref<ArticleInfo[]>([])
  const searchTotal = ref(0)

  // ---------- Getters ----------

  /** 当前文章是否已点赞 */
  const isCurrentLiked = computed(() => currentArticle.value?.isLiked ?? false)

  // ---------- Actions ----------

  /**
   * 获取文章列表
   */
  async function fetchArticles(params?: {
    status?: string
    categoryId?: string
    tags?: string[]
    authorId?: string
    sortBy?: string
    sortOrder?: string
    page?: number
    pageSize?: number
  }) {
    loading.value = true
    try {
      const res = await api.list({
        page: page.value,
        pageSize: pageSize.value,
        ...params,
      })
      articles.value = res.articles
      total.value = Number(res.total)
      page.value = res.page
      pageSize.value = res.pageSize
      return res
    }
    finally {
      loading.value = false
    }
  }

  /**
   * 加载更多（追加到列表）
   */
  async function loadMore(params?: Parameters<typeof api.list>[0]) {
    loading.value = true
    try {
      const nextPage = page.value + 1
      const res = await api.list({
        ...params,
        page: nextPage,
        pageSize: pageSize.value,
      })
      articles.value.push(...res.articles)
      total.value = Number(res.total)
      page.value = res.page
      return res
    }
    finally {
      loading.value = false
    }
  }

  /**
   * 获取文章详情
   */
  async function fetchArticle(identifier: string) {
    loading.value = true
    try {
      const res = await api.get(identifier)
      currentArticle.value = res.article
      return res
    }
    finally {
      loading.value = false
    }
  }

  /**
   * 创建文章
   */
  async function createArticle(data: CreateArticleRequest) {
    const res = await api.create(data)
    articles.value.unshift(res.article)
    total.value++
    return res
  }

  /**
   * 更新文章
   */
  async function updateArticle(id: string, data: UpdateArticleRequest) {
    const res = await api.update(id, data)
    // 更新列表和当前文章
    const idx = articles.value.findIndex(a => a.id === id)
    if (idx !== -1) articles.value[idx] = res.article
    if (currentArticle.value?.id === id) currentArticle.value = res.article
    return res
  }

  /**
   * 删除文章
   */
  async function deleteArticle(id: string) {
    await api.delete(id)
    articles.value = articles.value.filter(a => a.id !== id)
    total.value--
    if (currentArticle.value?.id === id) currentArticle.value = null
  }

  /**
   * 发布文章
   */
  async function publishArticle(id: string) {
    const res = await api.publish(id)
    const idx = articles.value.findIndex(a => a.id === id)
    if (idx !== -1) articles.value[idx] = res.article
    if (currentArticle.value?.id === id) currentArticle.value = res.article
    return res
  }

  /**
   * 归档文章
   */
  async function archiveArticle(id: string) {
    const res = await api.archive(id)
    const idx = articles.value.findIndex(a => a.id === id)
    if (idx !== -1) articles.value[idx] = res.article
    if (currentArticle.value?.id === id) currentArticle.value = res.article
    return res
  }

  /**
   * 点赞 / 取消点赞
   */
  async function toggleLike(id: string) {
    if (!currentArticle.value) return
    if (currentArticle.value.isLiked) {
      await api.unlike(id)
      currentArticle.value.isLiked = false
      currentArticle.value.likeCount = String(Number(currentArticle.value.likeCount) - 1)
    }
    else {
      await api.like(id)
      currentArticle.value.isLiked = true
      currentArticle.value.likeCount = String(Number(currentArticle.value.likeCount) + 1)
    }
  }

  /**
   * 全文搜索
   */
  async function searchArticles(keyword: string, params?: { page?: number; pageSize?: number }) {
    loading.value = true
    try {
      searchKeyword.value = keyword
      const res = await api.search({ keyword, ...params })
      searchResults.value = res.articles
      searchTotal.value = Number(res.total)
      return res
    }
    finally {
      loading.value = false
    }
  }

  /**
   * 重置草稿
   */
  function resetDraft() {
    draft.value = {
      title: '',
      content: '',
      excerpt: '',
      coverImage: '',
      categoryId: '',
      tagNames: [],
    }
  }

  /**
   * 清空搜索
   */
  function clearSearch() {
    searchKeyword.value = ''
    searchResults.value = []
    searchTotal.value = 0
  }

  // ---------- 预留：评论功能 ----------
  // 后续接入评论系统时，在此处扩展：
  // const comments = ref<CommentInfo[]>([])
  // async function fetchComments(articleId: string) { ... }
  // async function createComment(data: CreateCommentRequest) { ... }

  return {
    // State
    articles,
    total,
    page,
    pageSize,
    loading,
    currentArticle,
    draft,
    searchKeyword,
    searchResults,
    searchTotal,
    // Getters
    isCurrentLiked,
    // Actions
    fetchArticles,
    loadMore,
    fetchArticle,
    createArticle,
    updateArticle,
    deleteArticle,
    publishArticle,
    archiveArticle,
    toggleLike,
    searchArticles,
    resetDraft,
    clearSearch,
  }
})
