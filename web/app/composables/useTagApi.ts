/**
 * 标签 API 封装
 */
export function useTagApi() {
  const client = useApiClient()

  return {
    /** 获取全量标签列表 */
    list: async () => {
      const res = await client<GatewayResponse<ListTagsReply>>('/api/v1/tags')
      return unwrap(res)
    },

    /** 创建标签 */
    create: async (data: CreateTagRequest) => {
      const res = await client<GatewayResponse<CreateTagReply>>('/api/v1/tags', {
        method: 'POST',
        body: data,
      })
      return unwrap(res)
    },

    /** 删除标签 */
    delete: async (id: string) => {
      const res = await client<GatewayResponse<DeleteTagReply>>(`/api/v1/tags/${id}`, {
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
