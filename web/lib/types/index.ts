/**
 * 类型统一导出入口（index.ts）
 *
 * 本文件将所有分散在各模块中的类型定义集中导出，
 * 使其他模块可以通过单个导入语句获取所有类型。
 *
 * 使用方式：
 * import type { UserInfo, ArticleInfo, SiteConfig, ... } from '~~/lib/types'
 *
 * 导出来源模块：
 * - common.ts:   通用基础类型（TokenPair、UserInfo、AuthorInfo）
 * - auth.ts:     认证相关类型（LoginReply、RegisterReply 等）
 * - article.ts:  文章相关类型（ArticleInfo、ListArticlesReply 等）
 * - tag.ts:      标签相关类型（TagInfo、ListTagsReply 等）
 * - category.ts: 分类相关类型（CategoryInfo 等）
 * - file.ts:     文件管理类型（FileInfo、UploadFileReply 等）
 * - site.ts:     站点配置类型（SiteConfig、MusicPlaylist 等）
 *
 * 使用 export type * 语法：仅导出类型，不导出运行时的值。
 * 这确保类型定义不会被打包到最终的 JS bundle 中（tree-shaking 友好）。
 */

// =============================================================================
// 统一类型导出
// =============================================================================

export type * from './common'
export type * from './auth'
export type * from './article'
export type * from './tag'
export type * from './category'
export type * from './file'
export type * from './site'
