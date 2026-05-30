/**
 * 站点配置 API 封装
 */
export function useSiteApi() {
  const client = useApiClient()

  return {
    // ---------- 站点配置 ----------

    /** 获取站点配置 */
    getConfig: async () => {
      const res = await client<GatewayResponse<GetSiteConfigReply>>('/api/v1/site/config')
      return unwrap(res)
    },

    /** 更新站点配置 */
    updateConfig: async (data: SiteConfig) => {
      const res = await client<GatewayResponse<UpdateSiteConfigReply>>('/api/v1/site/config', {
        method: 'PUT',
        body: data,
      })
      return unwrap(res)
    },

    // ---------- 背景图片 ----------

    /** 获取背景图片列表 */
    listBackgrounds: async () => {
      const res = await client<GatewayResponse<ListBackgroundsReply>>('/api/v1/site/backgrounds')
      return unwrap(res)
    },

    /** 上传背景图片 */
    uploadBackground: async (data: UploadBackgroundRequest) => {
      const res = await client<GatewayResponse<UploadBackgroundReply>>('/api/v1/site/backgrounds', {
        method: 'POST',
        body: data,
      })
      return unwrap(res)
    },

    /** 删除背景图片 */
    deleteBackground: async (id: string) => {
      const res = await client<GatewayResponse<DeleteBackgroundReply>>(`/api/v1/site/backgrounds/${id}`, {
        method: 'DELETE',
      })
      return unwrap(res)
    },

    /** 设为当前背景 */
    setActiveBackground: async (id: string) => {
      const res = await client<GatewayResponse<SetActiveBackgroundReply>>(`/api/v1/site/backgrounds/${id}/active`, {
        method: 'PUT',
        body: { id },
      })
      return unwrap(res)
    },

    // ---------- 歌单 ----------

    /** 获取歌单 */
    getPlaylist: async () => {
      const res = await client<GatewayResponse<GetMusicPlaylistReply>>('/api/v1/site/music/playlist')
      return unwrap(res)
    },

    /** 更新歌单 */
    updatePlaylist: async (data: MusicPlaylist) => {
      const res = await client<GatewayResponse<UpdateMusicPlaylistReply>>('/api/v1/site/music/playlist', {
        method: 'PUT',
        body: data,
      })
      return unwrap(res)
    },
  }
}

interface GatewayResponse<T> {
  code: number
  msg: string
  data: T
}

function unwrap<T>(res: GatewayResponse<T>): T {
  if (res.code !== 0) {
    throw new Error(res.msg || `请求失败: code=${res.code}`)
  }
  return res.data
}
