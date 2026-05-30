/**
 * 分类 API 封装
 */
export function useCategoryApi() {
  const client = useApiClient()

  return {
    /** 获取分类树 */
    list: async () => {
      const res = await client<GatewayResponse<ListCategoriesReply>>('/api/v1/categories')
      return unwrap(res)
    },

    /** 创建分类 */
    create: async (data: CreateCategoryRequest) => {
      const res = await client<GatewayResponse<CreateCategoryReply>>('/api/v1/categories', {
        method: 'POST',
        body: data,
      })
      return unwrap(res)
    },

    /** 更新分类 */
    update: async (id: string, data: UpdateCategoryRequest) => {
      const res = await client<GatewayResponse<UpdateCategoryReply>>(`/api/v1/categories/${id}`, {
        method: 'PUT',
        body: data,
      })
      return unwrap(res)
    },

    /** 删除分类 */
    delete: async (id: string) => {
      const res = await client<GatewayResponse<DeleteCategoryReply>>(`/api/v1/categories/${id}`, {
        method: 'DELETE',
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
