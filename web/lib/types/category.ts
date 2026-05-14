/**
 * 分类相关类型定义（category.ts）
 *
 * 与后端 proto 文件 api/blog/v1/blog.proto 中的 CategoryInfo 及
 * 相关请求/响应消息严格对齐。JSON 字段名使用 snake_case。
 *
 * 分类（Category）是文章的主要组织方式，与标签（Tag）的区别：
 * - 分类是树形层级结构（通过 parent_id 和 children 实现递归）
 * - 一篇文章只能属于一个分类（一对一关系）
 * - 标签是扁平的（无层级），一篇文章可有多个标签（多对多关系）
 */

// =============================================================================
// 与 api/blog/v1/blog.proto CategoryInfo 严格对齐
// JSON key: snake_case（与 Go json tag 一致）
// =============================================================================

/**
 * CategoryInfo: 分类信息
 *
 * 对应 proto 消息：api.blog.v1.CategoryInfo
 *
 * 支持递归 children 字段，形成树形结构。
 * 例如：
 *   { name: "前端", children: [
 *     { name: "Vue", children: [] },
 *     { name: "React", children: [] }
 *   ]}
 *
 * 字段说明：
 * - id:            分类唯一标识（uint64）
 * - name:          分类名称（如 "前端"、"后端"）
 * - slug:          分类 URL 友好标识（如 "frontend"、"backend"），
 *                  用于路由 /categories/{slug}
 * - description:   分类描述
 * - parent_id:     父分类 ID（uint64），0 表示顶级分类
 * - sort_order:    排序权重（int32），数值越小越靠前，用于自定义排序
 * - article_count: 该分类（含子分类）下的文章总数（int64）
 * - children:      子分类数组（CategoryInfo[]），递归结构
 *
 * 注意：article_count 是计算字段，后端汇总了该分类及其所有子分类的文章数量。
 * children 字段可能导致响应体积较大（递归加载整棵树），
 * 对于大型分类体系可考虑前端按需懒加载子节点。
 */
export interface CategoryInfo {
  /** 分类 ID，uint64 */
  id: number
  name: string
  slug: string
  description: string
  /** 父分类 ID，uint64 */
  parent_id: number
  /** 排序权重，int32 */
  sort_order: number
  /** 文章总数（含子分类），int64 */
  article_count: number
  /** 递归子分类 */
  children: CategoryInfo[]
}

// =============================================================================
// Category Request / Reply 消息
// =============================================================================

/**
 * CreateCategoryRequest: 创建分类请求
 *
 * 对应 proto 消息：api.blog.v1.CreateCategoryRequest
 *
 * 必填字段：name（分类名称）
 * 可选字段：
 * - slug:        URL 标识（未填则后端根据 name 自动生成）
 * - description: 分类描述
 * - parent_id:   父分类 ID（用于创建子分类，不填则为顶级分类）
 * - sort_order:  排序权重
 */
export interface CreateCategoryRequest {
  name: string
  slug?: string
  description?: string
  /** 父分类 ID，uint64 */
  parent_id?: number
  /** 排序权重，int32 */
  sort_order?: number
}

/**
 * CreateCategoryReply: 创建分类响应
 *
 * 对应 proto 消息：api.blog.v1.CreateCategoryReply
 */
export interface CreateCategoryReply {
  category: CategoryInfo
}

/**
 * ListCategoriesReply: 分类列表响应
 *
 * 对应 proto 消息：api.blog.v1.ListCategoriesReply
 *
 * 返回顶级分类数组，每个顶级分类的 children 包含其子分类树。
 * 前端可用此数据构建导航菜单或分类选择器。
 */
export interface ListCategoriesReply {
  categories: CategoryInfo[]
}

/**
 * UpdateCategoryRequest: 更新分类请求
 *
 * 对应 proto 消息：api.blog.v1.UpdateCategoryRequest
 *
 * id 为必填字段，其余字段可选：
 * - 传入的字段将被更新
 * - 未传入的字段（undefined）保持原值不变
 */
export interface UpdateCategoryRequest {
  /** 分类 ID，uint64，必填 */
  id: number
  name?: string
  slug?: string
  description?: string
  /** 父分类 ID，uint64 */
  parent_id?: number
  /** 排序权重，int32 */
  sort_order?: number
}

/**
 * UpdateCategoryReply: 更新分类响应
 *
 * 对应 proto 消息：api.blog.v1.UpdateCategoryReply
 */
export interface UpdateCategoryReply {
  category: CategoryInfo
}

/**
 * DeleteCategoryRequest: 删除分类请求
 *
 * 对应 proto 消息：api.blog.v1.DeleteCategoryRequest
 *
 * 注意：
 * - 删除分类不会删除其下的文章，文章将变为"未分类"状态
 * - 如果分类有子分类，通常需要先处理子分类（移动或删除），
 *   具体行为由后端业务逻辑决定
 */
export interface DeleteCategoryRequest {
  /** 分类 ID，uint64 */
  id: number
}
