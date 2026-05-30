/**
 * 全局 UI 状态管理
 *
 * 管理主题、导航显隐、Toast 通知、侧边栏状态等。
 */
export const useUiStore = defineStore('ui', () => {
  // ---------- State ----------

  /** Toast 队列 */
  const toasts = ref<Array<{ id: string; message: string; type: 'error' | 'success' | 'info' }>>([])

  /** 导航栏是否可见 */
  const headerVisible = ref(true)

  /** 移动端侧边栏是否展开 */
  const mobileSidebarOpen = ref(false)

  /** 当前页面标题 */
  const pageTitle = ref('')

  /** 全局加载遮罩计数（用于嵌套请求） */
  const loadingCount = ref(0)
  const isGlobalLoading = computed(() => loadingCount.value > 0)

  // ---------- Actions ----------

  /**
   * 显示 Toast 提示
   * @param message 提示内容
   * @param type 类型
   * @param duration 显示时长（毫秒）
   */
  function toast(
    message: string,
    type: 'error' | 'success' | 'info' = 'info',
    duration = 3000,
  ) {
    const id = Math.random().toString(36).slice(2, 9)
    toasts.value.push({ id, message, type })

    if (duration > 0) {
      setTimeout(() => {
        removeToast(id)
      }, duration)
    }

    return id
  }

  /**
   * 移除指定 Toast
   */
  function removeToast(id: string) {
    toasts.value = toasts.value.filter(t => t.id !== id)
  }

  /**
   * 显示全局加载
   */
  function showLoading() {
    loadingCount.value++
  }

  /**
   * 隐藏全局加载
   */
  function hideLoading() {
    loadingCount.value = Math.max(0, loadingCount.value - 1)
  }

  /**
   * 切换导航栏显隐
   */
  function setHeaderVisible(visible: boolean) {
    headerVisible.value = visible
  }

  /**
   * 切换移动端侧边栏
   */
  function toggleMobileSidebar() {
    mobileSidebarOpen.value = !mobileSidebarOpen.value
  }

  /**
   * 设置页面标题
   */
  function setPageTitle(title: string) {
    pageTitle.value = title
    if (import.meta.client) {
      document.title = title ? `${title} · Ley` : 'Ley'
    }
  }

  return {
    // State
    toasts,
    headerVisible,
    mobileSidebarOpen,
    pageTitle,
    loadingCount,
    isGlobalLoading,
    // Actions
    toast,
    removeToast,
    showLoading,
    hideLoading,
    setHeaderVisible,
    toggleMobileSidebar,
    setPageTitle,
  }
})
