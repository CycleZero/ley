/**
 * 站点配置相关 API 模块（site.ts）
 *
 * 封装站点级配置管理的后端 API 调用，包括三大类：
 *
 * 1. 站点全局配置（SiteConfig）：
 *    - 获取/更新站点标题、SEO 信息、社交链接、评论/点赞开关等
 *
 * 2. 站点背景图片（SiteBackground）：
 *    - 列表查询、上传、删除、设为活跃
 *
 * 3. 音乐歌单（MusicPlaylist）：
 *    - 获取/更新站点背景音乐播放列表
 *
 * 这些接口通常需要管理员权限，后端通过中间件校验角色。
 */

import type { GetSiteConfigReply, UpdateSiteConfigRequest, UpdateSiteConfigReply, ListBackgroundsReply, UploadBackgroundReply, GetMusicPlaylistReply, UpdateMusicPlaylistRequest, UpdateMusicPlaylistReply } from '../types'
import { api } from './client'

/**
 * siteApi: 站点配置 API 命名空间对象
 */
export const siteApi = {
  /**
   * getConfig: 获取站点全局配置
   *
   * 返回值：Promise<GetSiteConfigReply>
   * 包含完整的 SiteConfig 对象（15 个字段，见 lib/types/site.ts）
   *
   * 使用场景：首页加载时获取站点标题、副标题、社交链接等信息，用于渲染页面头部和底部
   */
  getConfig() {
    return api.get<GetSiteConfigReply>('/api/v1/site/config')
  },

  /**
   * updateConfig: 更新站点全局配置
   *
   * 参数：
   * @param config - 完整的 SiteConfig 对象（全量替换，非部分更新）
   *
   * 返回值：Promise<UpdateSiteConfigReply>，包含更新后的 SiteConfig
   *
   * 注意：前端需要将整个 SiteConfig 对象提交，而非仅提交修改的字段。
   * 因此更新前应先调用 getConfig 获取当前配置，修改后再提交。
   *
   * 权限要求：管理员
   */
  updateConfig(config: UpdateSiteConfigRequest) {
    return api.put<UpdateSiteConfigReply>('/api/v1/site/config', config)
  },

  /**
   * listBackgrounds: 获取所有背景图片列表
   *
   * 返回值：Promise<ListBackgroundsReply>
   * 包含 backgrounds 数组，每个元素为 SiteBackground 对象
   */
  listBackgrounds() {
    return api.get<ListBackgroundsReply>('/api/v1/site/backgrounds')
  },

  /**
   * uploadBackground: 上传新的背景图片
   *
   * 参数：
   * @param filename - 文件名（含扩展名，如 "sunset.jpg"）
   * @param content  - 文件的二进制内容（Uint8Array）
   *
   * 返回值：Promise<UploadBackgroundReply>，包含新创建的 SiteBackground 对象
   *
   * 注意：此处通过 JSON 传递文件内容（base64 编码后），
   * 而非标准的 multipart/form-data 上传方式。
   * 大文件上传可能性能较差，适合背景图这类小文件场景。
   */
  uploadBackground(filename: string, content: Uint8Array) {
    return api.post<UploadBackgroundReply>('/api/v1/site/backgrounds', { filename, content })
  },

  /**
   * deleteBackground: 删除指定背景图片
   *
   * 参数：
   * @param id - 背景图片 ID
   *
   * 返回值：Promise<void>
   *
   * 注意：如果删除的是当前活跃（is_active === true）的背景，
   * 页面将回退到默认背景色。
   */
  deleteBackground(id: number) {
    return api.delete<void>(`/api/v1/site/backgrounds/${id}`)
  },

  /**
   * setActiveBackground: 将指定背景图片设为当前活跃状态
   *
   * 参数：
   * @param id - 背景图片 ID
   *
   * 返回值：Promise<void>
   *
   * 行为：设置后，之前活跃的背景自动变为非活跃（is_active = false）。
   * 每次仅允许一个背景为活跃状态。
   */
  setActiveBackground(id: number) {
    return api.put<void>(`/api/v1/site/backgrounds/${id}/active`)
  },

  /**
   * getPlaylist: 获取当前音乐歌单
   *
   * 返回值：Promise<GetMusicPlaylistReply>
   * 包含 playlist (MusicPlaylist) 对象，其 tracks 数组包含所有音乐曲目
   */
  getPlaylist() {
    return api.get<GetMusicPlaylistReply>('/api/v1/site/music/playlist')
  },

  /**
   * updatePlaylist: 更新音乐歌单（全量替换）
   *
   * 参数：
   * @param playlist - 完整的 MusicPlaylist 对象，包含 tracks 数组
   *
   * 返回值：Promise<UpdateMusicPlaylistReply>，包含更新后的 MusicPlaylist
   *
   * 注意：更新为全量替换，非增量添加。
   * 如需添加单首歌曲，应先获取当前歌单，修改 tracks 数组后再提交。
   *
   * 权限要求：管理员
   */
  updatePlaylist(playlist: UpdateMusicPlaylistRequest) {
    return api.put<UpdateMusicPlaylistReply>('/api/v1/site/music/playlist', playlist)
  },
}
