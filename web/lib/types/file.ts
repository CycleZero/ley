/**
 * 文件管理相关类型定义（file.ts）
 *
 * 与后端 proto 文件 api/blog/v1/blog.proto 中的 FileInfo 及
 * 相关请求/响应消息严格对齐。JSON 字段名使用 snake_case。
 *
 * 文件管理模块用于文章附件、图片等静态资源的上传和管理。
 * 支持两种上传方式：
 * 1. 直接上传（UploadFileRequest）：前端将文件内容 base64 编码后通过 JSON 发送
 * 2. 预签名上传（GetPresignedPutURLRequest）：后端返回预签名 URL，
 *    前端直接 PUT 文件到对象存储（如 S3/MinIO），更高效
 */

// =============================================================================
// 与 api/blog/v1/blog.proto FileInfo 严格对齐
// JSON key: snake_case（与 Go json tag 一致）
// =============================================================================

/**
 * FileInfo: 文件信息
 *
 * 对应 proto 消息：api.blog.v1.FileInfo
 *
 * 字段说明：
 * - id:        文件唯一标识（uint64）
 * - filename:  原始文件名（含扩展名），如 "screenshot.png"
 * - mime_type: MIME 类型，如 "image/png"、"application/pdf"
 * - size:      文件大小（字节，int64）
 * - url:       文件访问 URL（可公开访问的完整地址）
 * - created_at: 上传时间（ISO 8601 格式）
 */
export interface FileInfo {
  /** 文件 ID，uint64 */
  id: number
  filename: string
  mime_type: string
  /** 文件大小（字节），int64 */
  size: number
  url: string
  /** 上传时间，ISO 8601 */
  created_at: string
}

// =============================================================================
// File Request / Reply 消息
// =============================================================================

/**
 * UploadFileRequest: 直接上传文件请求
 *
 * 对应 proto 消息：api.blog.v1.UploadFileRequest
 *
 * 注意：标称 multipart/form-data，但实际可能通过 JSON 传输
 * （content 字段为 base64 编码后的字节数组）。
 *
 * 字段：
 * - filename: 文件名（供服务端记录和生成 URL）
 * - mime_type: MIME 类型（服务端用于设置响应 Content-Type）
 * - content:  文件二进制内容（Uint8Array）
 *
 * 限制：单文件大小通常有限制（如 10MB），配置在后端。
 */
export interface UploadFileRequest {
  filename: string
  mime_type: string
  /** 文件字节内容，bytes */
  content: Uint8Array
}

/**
 * UploadFileReply: 上传文件响应
 *
 * 对应 proto 消息：api.blog.v1.UploadFileReply
 */
export interface UploadFileReply {
  file: FileInfo
}

/**
 * GetFileRequest: 获取文件信息请求
 *
 * 对应 proto 消息：api.blog.v1.GetFileRequest
 */
export interface GetFileRequest {
  /** 文件 ID，uint64 */
  id: number
}

/**
 * GetFileReply: 获取文件信息响应
 *
 * 对应 proto 消息：api.blog.v1.GetFileReply
 */
export interface GetFileReply {
  file: FileInfo
}

/**
 * DeleteFileRequest: 删除文件请求
 *
 * 对应 proto 消息：api.blog.v1.DeleteFileRequest
 *
 * 注意：删除文件会同时删除存储中的实际文件和相关数据库记录，
 * 操作不可逆，前端应展示确认提示。
 */
export interface DeleteFileRequest {
  /** 文件 ID，uint64 */
  id: number
}

/**
 * ListFilesRequest: 文件列表查询请求
 *
 * 对应 proto 消息：api.blog.v1.ListFilesRequest
 *
 * 字段：
 * - page:      页码（int32，从 1 开始）
 * - page_size: 每页数量（int32）
 */
export interface ListFilesRequest {
  /** 页码，int32 */
  page?: number
  /** 每页数量，int32 */
  page_size?: number
}

/**
 * ListFilesReply: 文件列表查询响应
 *
 * 对应 proto 消息：api.blog.v1.ListFilesReply
 *
 * 字段：
 * - files: 文件数组
 * - total: 文件总数（int64）
 */
export interface ListFilesReply {
  files: FileInfo[]
  /** 文件总数，int64 */
  total: number
}

/**
 * GetPresignedPutURLRequest: 获取预签名上传 URL 请求
 *
 * 对应 proto 消息：api.blog.v1.GetPresignedPutURLRequest
 *
 * 预签名上传流程：
 * 1. 前端调用此接口，传入文件名和 MIME 类型
 * 2. 后端生成一个有时效的预签名 PUT URL（通常 5-15 分钟有效）
 * 3. 前端使用该 URL 直接 PUT 文件到对象存储（绕过应用服务器）
 * 4. 上传完成后通知后端创建文件记录
 *
 * 优点：大文件上传不占用应用服务器带宽，速度更快。
 *
 * 字段：
 * - filename: 文件名
 * - mime_type: MIME 类型
 */
export interface GetPresignedPutURLRequest {
  filename: string
  mime_type: string
}

/**
 * GetPresignedPutURLReply: 获取预签名上传 URL 响应
 *
 * 对应 proto 消息：api.blog.v1.GetPresignedPutURLReply
 *
 * 字段：
 * - url:        预签名的 PUT URL（有时效性）
 * - object_key: 对象存储中的 key，用于后续关联文件记录
 */
export interface GetPresignedPutURLReply {
  url: string
  object_key: string
}
