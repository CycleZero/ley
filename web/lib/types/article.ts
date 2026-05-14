/**
 * 文章相关类型定义（article.ts）
 *
 * 与后端 proto 文件 api/blog/v1/blog.proto 中的 ArticleInfo 及
 * 相关请求/响应消息严格对齐。JSON 字段名使用 snake_case。
 *
 * 定义了博客文章（Article）的完整类型体系，包括：
 * - ArticleInfo: 文章的核心数据结构
 * - 各类 Request/Reply: 对应 CRUD 操作的请求体和响应体
 */

// =============================================================================
// 与 api/blog/v1/blog.proto ArticleInfo 及相关消息严格对齐
// JSON key: snake_case（与 Go json tag 一致）
// =============================================================================

import type { AuthorInfo } from './common'
import type { TagInfo } from './tag'
import type { CategoryInfo } from './category'

// =============================================================================
// ArticleInfo — 文章信息
// =============================================================================

/**
 * ArticleInfo: 文章的完整信息
 *
 * 对应 proto 消息：api.blog.v1.ArticleInfo
 *
 * 字段说明：
 * - id:            文章唯一标识（uint64）
 * - title:         文章标题
 * - slug:          文章 URL 友好标识（如 "hello-world"），用于 SEO 友好的 URL
 * - content:       Markdown 格式的原文内容
 * - excerpt:       文章摘要/节选，用于列表展示
 * - cover_image:   封面图片 URL
 * - status:        文章状态："draft"（草稿）| "published"（已发布）| "archived"（已归档）
 * - author_id:     作者用户 ID
 * - author:        作者信息（嵌套 AuthorInfo 对象）
 * - category_id:   所属分类 ID
 * - category:      所属分类信息（嵌套 CategoryInfo 对象，含递归 children）
 * - tags:          关联标签数组（TagInfo[]）
 * - view_count:    浏览次数（int64）
 * - like_count:    点赞次数（int64）
 * - comment_count: 评论次数（int64）
 * - is_top:        是否置顶
 * - is_liked:      当前用户是否已点赞（需登录状态，未登录始终为 false）
 * - published_at:  首次发布时间（ISO 8601，草稿状态下为空字符串）
 * - created_at:    创建时间（ISO 8601）
 * - updated_at:    最后更新时间（ISO 8601）
 */
export interface ArticleInfo {
  /** 文章 ID，uint64 */
  id: number
  title: string
  slug: string
  /** Markdown 原文 */
  content: string
  excerpt: string
  cover_image: string
  /** "draft" | "published" | "archived" */
  status: string
  /** 作者用户 ID，uint64 */
  author_id: number
  author: AuthorInfo
  /** 分类 ID，uint64 */
  category_id: number
  category: CategoryInfo
  tags: TagInfo[]
  /** 浏览次数，int64 */
  view_count: number
  /** 点赞次数，int64 */
  like_count: number
  /** 评论次数，int64 */
  comment_count: number
  is_top: boolean
  is_liked: boolean
  /** 首次发布时间，ISO 8601 */
  published_at: string
  /** 创建时间，ISO 8601 */
  created_at: string
  /** 最后更新时间，ISO 8601 */
  updated_at: string
}

// =============================================================================
// Article Request / Reply 消息
// =============================================================================

/**
 * CreateArticleRequest: 创建文章请求
 *
 * 对应 proto 消息：api.blog.v1.CreateArticleRequest
 *
 * 必填字段：
 * - title:   文章标题
 * - content: Markdown 正文
 *
 * 可选字段：
 * - excerpt:     摘要（不填则后端自动从 content 截取）
 * - cover_image: 封面图 URL
 * - category_id: 所属分类 ID
 * - tag_names:    标签名数组（传入标签的 name 字符串，非 ID。
 *                 后端自动创建不存在的标签，已存在的标签直接关联）
 */
export interface CreateArticleRequest {
  title: string
  content: string
  excerpt?: string
  cover_image?: string
  /** 分类 ID，uint64 */
  category_id?: number
  tag_names?: string[]
}

/**
 * CreateArticleReply: 创建文章响应
 *
 * 对应 proto 消息：api.blog.v1.CreateArticleReply
 */
export interface CreateArticleReply {
  article: ArticleInfo
}

/**
 * UpdateArticleRequest: 更新文章请求
 *
 * 对应 proto 消息：api.blog.v1.UpdateArticleRequest
 *
 * 所有字段（id 除外）均为可选：
 * - 传入的字段将被更新
 * - 未传入的字段（undefined）保持原值不变
 *
 * 注意：
 * - tag_names 传入后全量替换标签（非增量添加），
 *   如需保留部分旧标签，应传入完整列表
 */
export interface UpdateArticleRequest {
  /** 文章 ID，uint64，必填 */
  id: number
  title?: string
  content?: string
  excerpt?: string
  cover_image?: string
  /** 分类 ID，uint64 */
  category_id?: number
  /** 传入后全量替换标签 */
  tag_names?: string[]
}

/**
 * UpdateArticleReply: 更新文章响应
 *
 * 对应 proto 消息：api.blog.v1.UpdateArticleReply
 */
export interface UpdateArticleReply {
  article: ArticleInfo
}

/**
 * DeleteArticleRequest: 删除文章请求
 *
 * 对应 proto 消息：api.blog.v1.DeleteArticleRequest
 */
export interface DeleteArticleRequest {
  /** 文章 ID，uint64 */
  id: number
}

/**
 * GetArticleRequest: 获取单篇文章请求
 *
 * 对应 proto 消息：api.blog.v1.GetArticleRequest
 *
 * 字段：
 * - identifier: 文章标识符，可以是数字 ID 字符串或 Slug 字符串
 *               后端自动判断类型，例如 "123" 按 ID 查询，"hello-world" 按 Slug 查询
 */
export interface GetArticleRequest {
  /** ID 或 Slug */
  identifier: string
}

/**
 * GetArticleReply: 获取单篇文章响应
 *
 * 对应 proto 消息：api.blog.v1.GetArticleReply
 */
export interface GetArticleReply {
  article: ArticleInfo
}

/**
 * ListArticlesRequest: 文章列表查询请求
 *
 * 对应 proto 消息：api.blog.v1.ListArticlesRequest
 *
 * 所有字段均为可选，不传则使用默认值：
 * - status:      默认返回 "published"
 * - sort_by:     默认按 "created_at" 排序
 * - sort_order:  默认 "desc"（最新在前）
 * - page:        默认 1
 * - page_size:   默认 10
 */
export interface ListArticlesRequest {
  /** "draft" | "published" | "archived" */
  status?: string
  /** 分类 ID，uint64 */
  category_id?: number
  tags?: string[]
  /** 作者 ID，uint64 */
  author_id?: number
  /** "created_at" | "view_count" | "published_at" */
  sort_by?: string
  /** "asc" | "desc" */
  sort_order?: string
  /** 页码，int32 */
  page?: number
  /** 每页数量，int32 */
  page_size?: number
}

/**
 * ListArticlesReply: 文章列表查询响应
 *
 * 对应 proto 消息：api.blog.v1.ListArticlesReply
 *
 * 字段：
 * - articles:  当前页的文章数组
 * - total:     满足筛选条件的文章总数
 * - page:      当前页码
 * - page_size: 每页数量
 */
export interface ListArticlesReply {
  articles: ArticleInfo[]
  /** 总数，int64 */
  total: number
  /** 当前页码，int32 */
  page: number
  /** 每页数量，int32 */
  page_size: number
}

/**
 * PublishArticleRequest: 发布文章请求
 *
 * 对应 proto 消息：api.blog.v1.PublishArticleRequest
 *
 * 将草稿状态（draft）的文章发布为正式文章（published）。
 * 如果文章之前已发布，published_at 保持不变。
 */
export interface PublishArticleRequest {
  /** 文章 ID，uint64 */
  id: number
}

/**
 * PublishArticleReply: 发布文章响应
 *
 * 对应 proto 消息：api.blog.v1.PublishArticleReply
 */
export interface PublishArticleReply {
  article: ArticleInfo
}

/**
 * ArchiveArticleRequest: 归档文章请求
 *
 * 对应 proto 消息：api.blog.v1.ArchiveArticleRequest
 *
 * 将已发布（published）的文章归档（archived），归档后前台不展示。
 */
export interface ArchiveArticleRequest {
  /** 文章 ID，uint64 */
  id: number
}

/**
 * ArchiveArticleReply: 归档文章响应
 *
 * 对应 proto 消息：api.blog.v1.ArchiveArticleReply
 */
export interface ArchiveArticleReply {
  article: ArticleInfo
}

/**
 * SearchArticlesRequest: 搜索文章请求
 *
 * 对应 proto 消息：api.blog.v1.SearchArticlesRequest
 *
 * 字段：
 * - keyword:  搜索关键词（在标题和正文中全文匹配）
 * - page:     页码，默认 1
 * - page_size: 每页数量，默认 10
 */
export interface SearchArticlesRequest {
  keyword: string
  /** 页码，int32 */
  page?: number
  /** 每页数量，int32 */
  page_size?: number
}

/**
 * SearchArticlesReply: 搜索文章响应
 *
 * 对应 proto 消息：api.blog.v1.SearchArticlesReply
 *
 * 与 ListArticlesReply 的区别：不返回 page/page_size，仅返回 total。
 * 搜索结果默认按相关度排序。
 */
export interface SearchArticlesReply {
  articles: ArticleInfo[]
  /** 匹配总数，int64 */
  total: number
}

/**
 * LikeArticleRequest: 点赞文章请求
 *
 * 对应 proto 消息：api.blog.v1.LikeArticleRequest
 *
 * 需要认证（携带 accessToken），同一用户对同一文章多次点赞应做幂等处理。
 */
export interface LikeArticleRequest {
  /** 文章 ID，uint64 */
  id: number
}

/**
 * UnlikeArticleRequest: 取消点赞请求
 *
 * 对应 proto 消息：api.blog.v1.UnlikeArticleRequest
 *
 * 需要认证，对未点赞的文章取消点赞应返回成功或静默忽略。
 */
export interface UnlikeArticleRequest {
  /** 文章 ID，uint64 */
  id: number
}
