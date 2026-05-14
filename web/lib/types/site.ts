/**
 * 站点配置相关类型定义（site.ts）
 *
 * 与后端 proto 文件 api/blog/v1/blog.proto 中的 SiteConfig、
 * SiteBackground、MusicTrack/MusicPlaylist 及相关请求/响应消息严格对齐。
 * JSON 字段名使用 snake_case。
 *
 * 包含三大数据模型：
 * 1. SiteConfig:      站点全局配置（SEO、社交链接、功能开关等）
 * 2. SiteBackground:  站点背景图片管理
 * 3. MusicPlaylist:   站点背景音乐歌单
 */

// =============================================================================
// 与 api/blog/v1/blog.proto SiteConfig / SiteBackground / MusicTrack 严格对齐
// JSON key: snake_case（与 Go json tag 一致）
// =============================================================================

// =============================================================================
// SiteConfig — 站点全局配置
// =============================================================================

/**
 * SiteConfig: 站点全局配置
 *
 * 对应 proto 消息：api.blog.v1.SiteConfig
 *
 * 共 15 个字段，覆盖站点的展示、SEO、社交、功能开关等方面。
 *
 * 字段分组：
 *
 * 【展示相关】
 * - site_title:       站点标题（浏览器标签页 <title> 显示）
 * - site_subtitle:    站点副标题（首页大标语/口号）
 * - site_logo:        站点 Logo 图片 URL
 * - site_favicon:     浏览器 Favicon 图标 URL
 *
 * 【SEO 相关】
 * - site_description:  站点简介（HTML meta description）
 * - seo_keywords:      全局 SEO 关键词（meta keywords）
 * - seo_description:   全局 SEO 描述（优先于 site_description，
 *                      如果为空则回退到 site_description）
 *
 * 【社交链接】
 * - social_github:    GitHub 主页链接
 * - social_twitter:   Twitter 主页链接
 * - social_email:     联系邮箱
 *
 * 【页脚】
 * - footer_text:      页脚文案（通常包含版权声明，如 "© 2024 Ley Blog"）
 * - icp_number:       ICP 备案号（中国网站必备）
 *
 * 【功能开关】
 * - enable_comments:          全站评论功能开关（false 时所有文章禁止评论）
 * - enable_likes:             全站点赞功能开关（false 时所有文章禁止点赞）
 * - auto_approve_comments:    评论自动审核通过开关
 *                             true  = 评论无需审核直接发布
 *                             false = 评论需管理员审核才能公开显示
 */
export interface SiteConfig {
  /** 站点标题 */
  site_title: string
  /** 站点副标题（首页标语） */
  site_subtitle: string
  /** 站点简介（SEO description） */
  site_description: string
  /** 站点 Logo URL */
  site_logo: string
  /** 浏览器 Favicon URL */
  site_favicon: string
  /** SEO 关键词（meta keywords） */
  seo_keywords: string
  /** SEO 描述（优先于 site_description） */
  seo_description: string
  /** GitHub 主页链接 */
  social_github: string
  /** Twitter 主页链接 */
  social_twitter: string
  /** 联系邮箱 */
  social_email: string
  /** 页脚文案（版权声明等） */
  footer_text: string
  /** ICP 备案号 */
  icp_number: string
  /** 全站评论开关 */
  enable_comments: boolean
  /** 全站点赞开关 */
  enable_likes: boolean
  /** 评论自动审核通过（false=需管理员审核） */
  auto_approve_comments: boolean
}

// =============================================================================
// SiteBackground — 背景图片
// =============================================================================

/**
 * SiteBackground: 站点背景图片
 *
 * 对应 proto 消息：api.blog.v1.SiteBackground
 *
 * 字段说明：
 * - id:         背景图片唯一标识（uint64）
 * - filename:   文件名
 * - url:        背景图片访问 URL
 * - is_active:  是否为当前活跃背景（同一时间仅允许一个背景为活跃状态）
 * - sort_order: 排序权重（int32），用于管理后台列表排序
 * - created_at: 上传时间（ISO 8601）
 */
export interface SiteBackground {
  /** 背景 ID，uint64 */
  id: number
  filename: string
  url: string
  is_active: boolean
  /** 排序权重，int32 */
  sort_order: number
  /** 上传时间，ISO 8601 */
  created_at: string
}

// =============================================================================
// MusicTrack / MusicPlaylist — 歌单
// =============================================================================

/**
 * MusicTrack: 音乐曲目
 *
 * 对应 proto 消息：api.blog.v1.MusicTrack
 *
 * 字段说明：
 * - title:    歌曲标题
 * - artist:   艺人/表演者
 * - url:      音乐文件的外部链接（http/https），
 *             通常为非本地托管的音乐资源（如 CDN 或第三方服务）
 * - cover_url: 封面图片链接
 *
 * 注意：音乐文件本身不存储在站点服务器上，
 * 仅存储外部 URL 引用。确保 URL 可跨域访问（CORS 配置正确）。
 */
export interface MusicTrack {
  title: string
  artist: string
  /** 外部音乐链接（http/https） */
  url: string
  /** 封面图链接 */
  cover_url: string
}

/**
 * MusicPlaylist: 音乐歌单
 *
 * 对应 proto 消息：api.blog.v1.MusicPlaylist
 *
 * 包含一个有序的 MusicTrack 数组，前端按数组顺序播放。
 * tracks 可为空数组（表示没有音乐）。
 */
export interface MusicPlaylist {
  tracks: MusicTrack[]
}

// =============================================================================
// Site Request / Reply 消息
// =============================================================================

/**
 * GetSiteConfigReply: 获取站点配置响应
 *
 * 对应 proto 消息：api.blog.v1.GetSiteConfigReply
 *
 * 无对应的 GetSiteConfigRequest，因为 GET 请求无请求体。
 */
export interface GetSiteConfigReply {
  config: SiteConfig
}

/**
 * UpdateSiteConfigRequest: 更新站点配置请求
 *
 * 对应 proto 消息：api.blog.v1.UpdateSiteConfigRequest
 *
 * 传入完整的 SiteConfig 对象（全量替换，非部分更新）。
 * 建议使用流程：先 getConfig 获取当前配置 → 修改 → 提交完整配置。
 */
export interface UpdateSiteConfigRequest {
  config: SiteConfig
}

/**
 * UpdateSiteConfigReply: 更新站点配置响应
 *
 * 对应 proto 消息：api.blog.v1.UpdateSiteConfigReply
 */
export interface UpdateSiteConfigReply {
  config: SiteConfig
}

/**
 * ListBackgroundsReply: 背景图片列表响应
 *
 * 对应 proto 消息：api.blog.v1.ListBackgroundsReply
 */
export interface ListBackgroundsReply {
  backgrounds: SiteBackground[]
}

/**
 * UploadBackgroundRequest: 上传背景图片请求
 *
 * 对应 proto 消息：api.blog.v1.UploadBackgroundRequest
 *
 * 字段：
 * - filename: 文件名
 * - content:  文件二进制内容（Uint8Array / bytes）
 */
export interface UploadBackgroundRequest {
  filename: string
  /** 文件字节内容，bytes */
  content: Uint8Array
}

/**
 * UploadBackgroundReply: 上传背景图片响应
 *
 * 对应 proto 消息：api.blog.v1.UploadBackgroundReply
 */
export interface UploadBackgroundReply {
  background: SiteBackground
}

/**
 * DeleteBackgroundRequest: 删除背景图片请求
 *
 * 对应 proto 消息：api.blog.v1.DeleteBackgroundRequest
 */
export interface DeleteBackgroundRequest {
  /** 背景 ID，uint64 */
  id: number
}

/**
 * SetActiveBackgroundRequest: 设置活跃背景请求
 *
 * 对应 proto 消息：api.blog.v1.SetActiveBackgroundRequest
 *
 * 将指定背景设为活跃状态，之前的活跃背景自动取消。
 */
export interface SetActiveBackgroundRequest {
  /** 背景 ID，uint64 */
  id: number
}

/**
 * GetMusicPlaylistReply: 获取歌单响应
 *
 * 对应 proto 消息：api.blog.v1.GetMusicPlaylistReply
 */
export interface GetMusicPlaylistReply {
  playlist: MusicPlaylist
}

/**
 * UpdateMusicPlaylistRequest: 更新歌单请求
 *
 * 对应 proto 消息：api.blog.v1.UpdateMusicPlaylistRequest
 *
 * 传入完整的 MusicPlaylist 对象（全量替换 tracks 数组）。
 */
export interface UpdateMusicPlaylistRequest {
  playlist: MusicPlaylist
}

/**
 * UpdateMusicPlaylistReply: 更新歌单响应
 *
 * 对应 proto 消息：api.blog.v1.UpdateMusicPlaylistReply
 */
export interface UpdateMusicPlaylistReply {
  playlist: MusicPlaylist
}
