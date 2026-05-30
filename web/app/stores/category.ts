/**
 * 分类状态管理
 *
 * 分类数据变更频率低，适合本地缓存。
 */
export const useCategoryStore = defineStore('category', () => {
  const api = useCategoryApi()

  // ---------- State ----------

  /** 分类树 */
  const categories = ref<CategoryInfo[]>([])
  /** 是否加载中 */
  const loading = ref(false)
  /** 当前选中分类 */
  const currentCategory = ref<CategoryInfo | null>(null)

  // ---------- Getters ----------

  /** 扁平化的分类列表（含层级路径） */
  const flatCategories = computed(() => {
    const result: Array<{ category: CategoryInfo; depth: number; path: string }> = []
    function walk(list: CategoryInfo[], depth: number, prefix: string) {
      for (const cat of list) {
        const name = prefix ? `${prefix} / ${cat.name}` : cat.name
        result.push({ category: cat, depth, path: name })
        if (cat.children?.length) {
          walk(cat.children, depth + 1, name)
        }
      }
    }
    walk(categories.value, 0, '')
    return result
  })

  /** 按 ID 查找分类 */
  const findById = computed(() => (id: string) => {
    function walk(list: CategoryInfo[]): CategoryInfo | null {
      for (const cat of list) {
        if (cat.id === id) return cat
        if (cat.children?.length) {
          const found = walk(cat.children)
          if (found) return found
        }
      }
      return null
    }
    return walk(categories.value)
  })

  // ---------- Actions ----------

  /**
   * 获取分类树
   */
  async function fetchCategories() {
    if (categories.value.length) return // 已缓存则跳过
    loading.value = true
    try {
      const res = await api.list()
      categories.value = res.categories
      return res
    }
    finally {
      loading.value = false
    }
  }

  /**
   * 创建分类
   */
  async function createCategory(data: CreateCategoryRequest) {
    const res = await api.create(data)
    // 重新加载分类树以保持数据一致性
    await fetchCategories()
    return res
  }

  /**
   * 更新分类
   */
  async function updateCategory(id: string, data: UpdateCategoryRequest) {
    const res = await api.update(id, data)
    await fetchCategories()
    return res
  }

  /**
   * 删除分类
   */
  async function deleteCategory(id: string) {
    await api.delete(id)
    await fetchCategories()
  }

  /**
   * 设置当前选中分类
   */
  function selectCategory(category: CategoryInfo | null) {
    currentCategory.value = category
  }

  return {
    // State
    categories,
    loading,
    currentCategory,
    // Getters
    flatCategories,
    findById,
    // Actions
    fetchCategories,
    createCategory,
    updateCategory,
    deleteCategory,
    selectCategory,
  }
})
