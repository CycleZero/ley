/**
 * 文件 API 封装
 */
export function useFileApi() {
  const client = useApiClient()

  return {
    /** 获取文件列表 */
    list: async (params?: { page?: number; pageSize?: number }) => {
      const res = await client<GatewayResponse<ListFilesReply>>('/api/v1/files', {
        query: params,
      })
      return unwrap(res)
    },

    /** 获取预签名上传 URL（客户端直传 MinIO） */
    getPresignedPutURL: async (params: { filename: string; mimeType: string }) => {
      const res = await client<GatewayResponse<GetPresignedPutURLReply>>('/api/v1/files/presigned-upload', {
        query: params,
      })
      return unwrap(res)
    },

    /** 上传文件（服务端上传） */
    upload: async (data: UploadFileRequest) => {
      const res = await client<GatewayResponse<UploadFileReply>>('/api/v1/files/upload', {
        method: 'POST',
        body: data,
      })
      return unwrap(res)
    },

    /** 获取文件信息 */
    get: async (id: string) => {
      const res = await client<GatewayResponse<GetFileReply>>(`/api/v1/files/${id}`)
      return unwrap(res)
    },

    /** 删除文件 */
    delete: async (id: string) => {
      const res = await client<GatewayResponse<DeleteFileReply>>(`/api/v1/files/${id}`, {
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
