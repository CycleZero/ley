/**
 * 文章 API 封装
 *
 * 后端响应格式: { code, msg, data }
 * 本模块自动解包 data。
 */
export function useArticleApi() {
  const client = useApiClient()

  return {
    /** 获取文章列表 */
    list: async (params?: {
      status?: string
      categoryId?: string
      tags?: string[]
      authorId?: string
      sortBy?: string
      sortOrder?: string
      page?: number
      pageSize?: number
    }) => {
      const res = await client<GatewayResponse<ListArticlesReply>>('/api/v1/articles', {
        method: 'GET',
        query: params,
      })
      return unwrap(res)
    },

    /** 创建文章 */
    create: async (data: CreateArticleRequest) => {
      const res = await client<GatewayResponse<CreateArticleReply>>('/api/v1/articles', {
        method: 'POST',
        body: data,
      })
      return unwrap(res)
    },

    /** 全文搜索 */
    search: async (params: { keyword?: string; page?: number; pageSize?: number }) => {
      const res = await client<GatewayResponse<SearchArticlesReply>>('/api/v1/articles/search', {
        method: 'GET',
        query: params,
      })
      return unwrap(res)
    },

    /** 获取文章详情（支持 ID 或 slug） */
    get: async (identifier: string) => {
      const res = await client<GatewayResponse<GetArticleReply>>(`/api/v1/articles/${identifier}`)
      return unwrap(res)
    },

    /** 更新文章 */
    update: async (id: string, data: UpdateArticleRequest) => {
      const res = await client<GatewayResponse<UpdateArticleReply>>(`/api/v1/articles/${id}`, {
        method: 'PUT',
        body: data,
      })
      return unwrap(res)
    },

    /** 删除文章 */
    delete: async (id: string) => {
      const res = await client<GatewayResponse<DeleteArticleReply>>(`/api/v1/articles/${id}`, {
        method: 'DELETE',
      })
      return unwrap(res)
    },

    /** 归档文章 */
    archive: async (id: string) => {
      const res = await client<GatewayResponse<ArchiveArticleReply>>(`/api/v1/articles/${id}/archive`, {
        method: 'POST',
        body: { id },
      })
      return unwrap(res)
    },

    /** 发布文章 */
    publish: async (id: string) => {
      const res = await client<GatewayResponse<PublishArticleReply>>(`/api/v1/articles/${id}/publish`, {
        method: 'POST',
        body: { id },
      })
      return unwrap(res)
    },

    /** 点赞文章 */
    like: async (id: string) => {
      const res = await client<GatewayResponse<LikeArticleReply>>(`/api/v1/articles/${id}/like`, {
        method: 'POST',
        body: { id },
      })
      return unwrap(res)
    },

    /** 取消点赞 */
    unlike: async (id: string) => {
      const res = await client<GatewayResponse<UnlikeArticleReply>>(`/api/v1/articles/${id}/like`, {
        method: 'DELETE',
      })
      return unwrap(res)
    },
  }
}

/** 网关标准响应包装 */
interface GatewayResponse<T> {
  code: number
  msg: string
  data: T
}

/** 解包网关响应，检查业务错误 */
function unwrap<T>(res: GatewayResponse<T>): T {
  if (res.code !== 0) {
    throw new Error(res.msg || `请求失败: code=${res.code}`)
  }
  return res.data
}
