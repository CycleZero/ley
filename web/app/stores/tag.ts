/**
 * 标签状态管理
 */
export const useTagStore = defineStore('tag', () => {
  const api = useTagApi()

  // ---------- State ----------

  /** 标签列表 */
  const tags = ref<TagInfo[]>([])
  /** 是否加载中 */
  const loading = ref(false)
  /** 当前选中标签 */
  const currentTag = ref<TagInfo | null>(null)

  // ---------- Getters ----------

  /** 按文章数降序排列的标签（后端已排序，此处可做二次排序） */
  const sortedTags = computed(() => {
    return [...tags.value].sort((a, b) => Number(b.articleCount) - Number(a.articleCount))
  })

  /** 热门标签（文章数 > 0） */
  const hotTags = computed(() => {
    return tags.value.filter(t => Number(t.articleCount) > 0)
  })

  // ---------- Actions ----------

  /**
   * 获取标签列表
   */
  async function fetchTags() {
    if (tags.value.length) return // 已缓存则跳过
    loading.value = true
    try {
      const res = await api.list()
      tags.value = res.tags
      return res
    }
    finally {
      loading.value = false
    }
  }

  /**
   * 创建标签
   */
  async function createTag(data: CreateTagRequest) {
    const res = await api.create(data)
    tags.value.push(res.tag)
    return res
  }

  /**
   * 删除标签
   */
  async function deleteTag(id: string) {
    await api.delete(id)
    tags.value = tags.value.filter(t => t.id !== id)
  }

  /**
   * 设置当前选中标签
   */
  function selectTag(tag: TagInfo | null) {
    currentTag.value = tag
  }

  return {
    // State
    tags,
    loading,
    currentTag,
    // Getters
    sortedTags,
    hotTags,
    // Actions
    fetchTags,
    createTag,
    deleteTag,
    selectTag,
  }
})
