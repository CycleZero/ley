/**
 * 文章相关 API 模块（articles.ts）
 *
 * 封装所有与博客文章相关的后端 API 调用，提供完整的文章 CRUD 及附加功能：
 * - 列表查询（list）：分页、筛选、排序
 * - 详情获取（get）：支持 ID 或 Slug 定位
 * - 创建（create）：新建文章草稿
 * - 更新（update）：修改文章内容
 * - 删除（delete）：删除文章
 * - 发布（publish）：将草稿发布为正式文章
 * - 归档（archive）：将已发布文章归档
 * - 搜索（search）：关键词搜索
 * - 点赞/取消点赞（like / unlike）
 *
 * 类型定义来源：lib/types/article.ts，与后端 proto 定义对齐
 */

import type { ArticleInfo, ListArticlesRequest, ListArticlesReply, CreateArticleRequest, CreateArticleReply, UpdateArticleRequest, UpdateArticleReply, GetArticleReply, SearchArticlesReply } from '../types'
import { api } from './client'

/**
 * articlesApi: 文章 API 命名空间对象
 *
 * 使用方式：
 * import { articlesApi } from '~~/lib/api/articles'
 * const { articles, total } = await articlesApi.list({ page: 1, status: 'published' })
 */
export const articlesApi = {
  /**
   * list: 获取文章列表
   *
   * 参数：
   * @param params - 筛选条件，所有字段均为可选（Partial<ListArticlesRequest>）：
   *   - status:      文章状态（"draft" | "published" | "archived"）
   *   - category_id: 分类 ID 筛选
   *   - tags:        标签名数组筛选
   *   - author_id:   作者 ID 筛选
   *   - sort_by:     排序字段（"created_at" | "view_count" | "published_at"）
   *   - sort_order:  排序方向（"asc" | "desc"）
   *   - page:        页码（从 1 开始）
   *   - page_size:   每页数量
   *
   * 返回值：Promise<ListArticlesReply>
   * 包含 articles 数组、total 总数、page 当前页码、page_size 每页大小
   */
  list(params?: Partial<ListArticlesRequest>) {
    return api.get<ListArticlesReply>('/api/v1/articles', params as Record<string, unknown>)
  },

  /**
   * get: 获取单个文章详情
   *
   * 参数：
   * @param identifier - 文章标识符，可以是数字 ID（如 "123"）或 Slug（如 "hello-world"）
   *                     后端自动判断是 ID 还是 Slug
   *
   * 返回值：Promise<GetArticleReply>，包含完整的 ArticleInfo
   */
  get(identifier: string) {
    return api.get<GetArticleReply>(`/api/v1/articles/${identifier}`)
  },

  /**
   * create: 创建新文章（默认为草稿状态）
   *
   * 参数：
   * @param body.title        - 文章标题（必填）
   * @param body.content      - Markdown 正文（必填）
   * @param body.excerpt      - 文章摘要（可选）
   * @param body.cover_image  - 封面图片 URL（可选）
   * @param body.category_id  - 分类 ID（可选）
   * @param body.tag_names    - 标签名数组（可选，传入标签的 name 而非 ID）
   *
   * 返回值：Promise<CreateArticleReply>，包含创建的 ArticleInfo
   */
  create(body: CreateArticleRequest) {
    return api.post<CreateArticleReply>('/api/v1/articles', body)
  },

  /**
   * update: 更新文章
   *
   * 参数：
   * @param id   - 文章 ID（必填，用于定位文章）
   * @param body - 要更新的字段（Omit<UpdateArticleRequest, 'id'>，不含 id）：
   *   所有字段均为可选，未传入的字段保持原值不变
   *   注意：tag_names 传入后会全量替换标签（不是增量添加）
   *
   * 返回值：Promise<UpdateArticleReply>，包含更新后的 ArticleInfo
   */
  update(id: number, body: Omit<UpdateArticleRequest, 'id'>) {
    return api.put<UpdateArticleReply>(`/api/v1/articles/${id}`, body)
  },

  /**
   * delete: 删除文章
   *
   * 参数：
   * @param id - 文章 ID
   *
   * 返回值：Promise<void> —— 空响应体
   *
   * 注意：删除操作通常不可逆，前端应展示二次确认弹窗
   */
  delete(id: number) {
    return api.delete<void>(`/api/v1/articles/${id}`)
  },

  /**
   * publish: 发布文章（将草稿变为已发布状态）
   *
   * 参数：
   * @param id - 文章 ID
   *
   * 返回值：Promise<{ article: ArticleInfo }>
   *
   * 状态转换：draft → published
   */
  publish(id: number) {
    return api.post<{ article: ArticleInfo }>(`/api/v1/articles/${id}/publish`)
  },

  /**
   * archive: 归档文章（将已发布文章变为归档状态）
   *
   * 参数：
   * @param id - 文章 ID
   *
   * 返回值：Promise<{ article: ArticleInfo }>
   *
   * 状态转换：published → archived
   * 归档后的文章在前台默认不可见，管理后台可查看
   */
  archive(id: number) {
    return api.post<{ article: ArticleInfo }>(`/api/v1/articles/${id}/archive`)
  },

  /**
   * search: 全文搜索文章
   *
   * 参数：
   * @param keyword  - 搜索关键词（在标题和内容中匹配）
   * @param page     - 页码（默认 1）
   * @param pageSize - 每页数量（默认 10）
   *
   * 返回值：Promise<SearchArticlesReply>
   * 包含 articles 匹配结果数组和 total 总数
   *
   * 注意：搜索接口不支持排序参数，默认按相关度排序
   */
  search(keyword: string, page = 1, pageSize = 10) {
    return api.get<SearchArticlesReply>('/api/v1/articles/search', { keyword, page, page_size: pageSize })
  },

  /**
   * like: 点赞文章
   *
   * 参数：
   * @param id - 文章 ID
   *
   * 返回值：Promise<void>
   *
   * 认证要求：需要有效的 accessToken（仅登录用户可点赞）
   * 注意：同一用户对同一文章多次点赞，后端应做幂等处理（不重复计数）
   */
  like(id: number) {
    return api.post<void>(`/api/v1/articles/${id}/like`)
  },

  /**
   * unlike: 取消点赞文章
   *
   * 参数：
   * @param id - 文章 ID
   *
   * 返回值：Promise<void>
   *
   * 认证要求：需要有效的 accessToken
   * 注意：对未点赞的文章取消点赞，后端应返回成功或静默忽略
   */
  unlike(id: number) {
    return api.delete<void>(`/api/v1/articles/${id}/like`)
  },
}
