/**
 * 标签相关类型定义（tag.ts）
 *
 * 与后端 proto 文件 api/blog/v1/blog.proto 中的 TagInfo 及
 * 相关请求/响应消息严格对齐。JSON 字段名使用 snake_case。
 *
 * 标签是文章的分类维度之一（与 Category 区别在于标签更灵活：
 * 一篇文章可有多个标签，标签之间无层级关系）。
 */

// =============================================================================
// 与 api/blog/v1/blog.proto TagInfo 严格对齐
// JSON key: snake_case（与 Go json tag 一致）
// =============================================================================

/**
 * TagInfo: 标签信息
 *
 * 对应 proto 消息：api.blog.v1.TagInfo
 *
 * 字段说明：
 * - id:            标签唯一标识（uint64）
 * - name:          标签名称（如 "Vue3"、"TypeScript"），用于展示
 * - slug:          标签 URL 友好标识（如 "vue3"、"typescript"），
 *                  用于路由 /tags/{slug}
 * - article_count: 该标签下的文章数量（int64），用于标签云显示权重
 */
export interface TagInfo {
  /** 标签 ID，uint64 */
  id: number
  name: string
  slug: string
  /** 关联文章数，int64 */
  article_count: number
}

// =============================================================================
// Tag Request / Reply 消息
// =============================================================================

/**
 * CreateTagRequest: 创建标签请求
 *
 * 对应 proto 消息：api.blog.v1.CreateTagRequest
 *
 * 字段：
 * - name: 标签名称（slug 由后端根据 name 自动生成）
 */
export interface CreateTagRequest {
  name: string
}

/**
 * CreateTagReply: 创建标签响应
 *
 * 对应 proto 消息：api.blog.v1.CreateTagReply
 */
export interface CreateTagReply {
  tag: TagInfo
}

/**
 * ListTagsReply: 标签列表响应
 *
 * 对应 proto 消息：api.blog.v1.ListTagsReply
 *
 * 返回所有标签，通常按 article_count 降序排列，
 * 用于标签云或侧边栏展示。
 */
export interface ListTagsReply {
  tags: TagInfo[]
}

/**
 * DeleteTagRequest: 删除标签请求
 *
 * 对应 proto 消息：api.blog.v1.DeleteTagRequest
 *
 * 注意：删除标签不会删除关联的文章，仅解除关联关系。
 */
export interface DeleteTagRequest {
  /** 标签 ID，uint64 */
  id: number
}
