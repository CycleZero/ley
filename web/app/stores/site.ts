/**
 * 站点配置状态管理
 */
export const useSiteStore = defineStore('site', () => {
  const api = useSiteApi()

  // ---------- State ----------

  /** 站点配置 */
  const config = ref<SiteConfig | null>(null)
  /** 背景图片列表 */
  const backgrounds = ref<SiteBackground[]>([])
  /** 当前激活背景 */
  const activeBackground = computed(() =>
    backgrounds.value.find(b => b.isActive) || null,
  )
  /** 歌单 */
  const playlist = ref<MusicPlaylist | null>(null)
  /** 是否加载中 */
  const loading = ref(false)

  // ---------- Getters ----------

  /** 站点标题 */
  const siteTitle = computed(() => config.value?.siteTitle || 'Ley')
  /** 站点副标题 */
  const siteSubtitle = computed(() => config.value?.siteSubtitle || '')
  /** 是否开启点赞 */
  const enableLikes = computed(() => config.value?.enableLikes ?? true)

  // ---------- Actions ----------

  /**
   * 获取站点配置
   */
  async function fetchConfig() {
    if (config.value) return
    loading.value = true
    try {
      const res = await api.getConfig()
      config.value = res.config
      return res
    }
    finally {
      loading.value = false
    }
  }

  /**
   * 更新站点配置
   */
  async function updateConfig(data: SiteConfig) {
    const res = await api.updateConfig({ config: data })
    config.value = res.config
    return res
  }

  /**
   * 获取背景图片列表
   */
  async function fetchBackgrounds() {
    loading.value = true
    try {
      const res = await api.listBackgrounds()
      backgrounds.value = res.backgrounds
      return res
    }
    finally {
      loading.value = false
    }
  }

  /**
   * 上传背景图片
   */
  async function uploadBackground(data: UploadBackgroundRequest) {
    const res = await api.uploadBackground(data)
    backgrounds.value.push(res.background)
    return res
  }

  /**
   * 删除背景图片
   */
  async function deleteBackground(id: string) {
    await api.deleteBackground(id)
    backgrounds.value = backgrounds.value.filter(b => b.id !== id)
  }

  /**
   * 设为当前背景
   */
  async function setActiveBackground(id: string) {
    await api.setActiveBackground(id)
    backgrounds.value.forEach(b => {
      b.isActive = b.id === id
    })
  }

  /**
   * 获取歌单
   */
  async function fetchPlaylist() {
    loading.value = true
    try {
      const res = await api.getPlaylist()
      playlist.value = res.playlist
      return res
    }
    finally {
      loading.value = false
    }
  }

  /**
   * 更新歌单
   */
  async function updatePlaylist(data: MusicPlaylist) {
    const res = await api.updatePlaylist({ playlist: data })
    playlist.value = res.playlist
    return res
  }

  return {
    // State
    config,
    backgrounds,
    playlist,
    loading,
    // Getters
    activeBackground,
    siteTitle,
    siteSubtitle,
    enableLikes,
    // Actions
    fetchConfig,
    updateConfig,
    fetchBackgrounds,
    uploadBackground,
    deleteBackground,
    setActiveBackground,
    fetchPlaylist,
    updatePlaylist,
  }
})
