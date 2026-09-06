package proxy

// ===================== 文件域代理 handler（file.go） =====================
//
// 覆盖 FileService 全部 7 个 RPC。路由与 api/blog/v1/blog.proto 的
// google.api.http 注解一一对应（权威路由表，路径/方法以 proto 为准）：
//
//	UploadFile               POST   /api/v1/files/upload
//	GetFile                  GET    /api/v1/files/{id}
//	DeleteFile               DELETE /api/v1/files/{id}
//	ListFiles                GET    /api/v1/files
//	GetPresignedPutURL       GET    /api/v1/files/presigned-upload
//	CreatePresignedUpload    POST   /api/v1/files/presigned-uploads
//	CompletePresignedUpload  POST   /api/v1/files/presigned-uploads/complete
//
// handler 自身只做：分配请求消息 → 合并路径/query 参数 → callProto 转发
// （骨架细节见 handle.go 头部设计约定）。本文件不承载任何文件业务规则
// （安全校验链/对象键签发/文件登记全部留在 blog 服务）。
//
// 字节传输说明（base64）：blog FileService 的请求消息含 bytes 字段
// （UploadFileRequest.content），JSON 传输时按 protobuf JSON 映射标准以
// base64 字符串承载。callProto 的 protojson 绑定在反序列化时自动完成
// base64 → []byte 解码（标准/URL-safe 字母表、有无 padding 均接受），
// handler 无需任何手工 base64 处理——客户端契约只是把文件字节 base64 后
// 放入 JSON 的 content 字段即可，本文件不引入额外编码逻辑。

import (
	"context"

	blogv1 "github.com/CycleZero/ley/api/blog/v1"
	"github.com/gin-gonic/gin"
	"google.golang.org/protobuf/proto"
)

// FileHandler 文件域代理 handler 聚合：持有下游 FileService gRPC client
// 所在 ServiceHub（仅使用 hub.File），各方法签名与 gin.HandlerFunc 一致，
// 可直接挂路由（路由注册由 router wave 统一完成）。
type FileHandler struct {
	hub *ServiceHub // 下游服务客户端聚合（文件域只用 hub.File）
}

// NewFileHandler 构造文件域代理 handler。
func NewFileHandler(hub *ServiceHub) *FileHandler {
	return &FileHandler{hub: hub}
}

// UploadFile 上传文件（服务端代理上传：body 经 protojson 绑定，其中 content
// 为文件字节的 base64 字符串，自动解码后经 gRPC 转发 blog，见本文件头部
// 「字节传输说明」）。
//
//	@Tags        文件
//	@Accept      json
//	@Produce     json
//	@Param       request body blogv1.UploadFileRequest true "上传请求（filename 文件名、mime_type MIME 类型、content 为 base64 文件字节）"
//	@Success     200 {object} common.Response{data=blogv1.UploadFileReply}
//	@Failure     400 {object} common.Response
//	@Router      /api/v1/files/upload [post]
func (h *FileHandler) UploadFile(c *gin.Context) {
	req := &blogv1.UploadFileRequest{}
	callProto(c, req, func(ctx context.Context, m proto.Message) (proto.Message, error) {
		return h.hub.File.UploadFile(ctx, m.(*blogv1.UploadFileRequest))
	})
}

// GetFile 获取文件元数据（认证用户只能访问自己的文件，鉴权在 blog 服务）。
//
//	@Tags        文件
//	@Accept      json
//	@Produce     json
//	@Param       id   path integer true "文件 ID"
//	@Success     200  {object} common.Response{data=blogv1.GetFileReply}
//	@Failure     400  {object} common.Response
//	@Failure     404  {object} common.Response
//	@Router      /api/v1/files/{id} [get]
func (h *FileHandler) GetFile(c *gin.Context) {
	req := &blogv1.GetFileRequest{}
	// 路径参数合并（显式逐字段）：路由 :id → proto Id 字段
	if !BindPathUint64(c, "id", &req.Id) {
		return
	}
	callProto(c, req, func(ctx context.Context, m proto.Message) (proto.Message, error) {
		return h.hub.File.GetFile(ctx, m.(*blogv1.GetFileRequest))
	})
}

// DeleteFile 删除文件（需为上传者本人，鉴权在 blog 服务；删除类接口无业务数据）。
//
//	@Tags        文件
//	@Accept      json
//	@Produce     json
//	@Param       id   path integer true "文件 ID"
//	@Success     200  {object} common.Response{data=blogv1.DeleteFileReply}
//	@Failure     400  {object} common.Response
//	@Failure     403  {object} common.Response
//	@Router      /api/v1/files/{id} [delete]
func (h *FileHandler) DeleteFile(c *gin.Context) {
	req := &blogv1.DeleteFileRequest{}
	// 路径参数合并（显式逐字段）：路由 :id → proto Id 字段
	if !BindPathUint64(c, "id", &req.Id) {
		return
	}
	callProto(c, req, func(ctx context.Context, m proto.Message) (proto.Message, error) {
		return h.hub.File.DeleteFile(ctx, m.(*blogv1.DeleteFileRequest))
	})
}

// ListFiles 文件列表（分页）：GET 无请求体，page/page_size 走 query 参数
// （与 kratos HTTP 网关语义一致——未出现在路径/body 的字段取自 query；
// 参数缺失时保持零值，分页默认值由 blog 服务侧兜底）。
//
//	@Tags        文件
//	@Accept      json
//	@Produce     json
//	@Param       page      query integer false "页码（从 1 起，缺省由服务端兜底）"
//	@Param       page_size query integer false "每页条数（缺省由服务端兜底）"
//	@Success     200       {object} common.Response{data=blogv1.ListFilesReply}
//	@Failure     400       {object} common.Response
//	@Router      /api/v1/files [get]
func (h *FileHandler) ListFiles(c *gin.Context) {
	req := &blogv1.ListFilesRequest{}
	// query 参数合并（本文件内辅助，见文件尾部说明）
	if !bindQueryInt32(c, "page", &req.Page) || !bindQueryInt32(c, "page_size", &req.PageSize) {
		return
	}
	callProto(c, req, func(ctx context.Context, m proto.Message) (proto.Message, error) {
		return h.hub.File.ListFiles(ctx, m.(*blogv1.ListFilesRequest))
	})
}

// GetPresignedPutURL 获取 MinIO 预签名上传 URL（1 小时有效，客户端可直接上传）：
// GET 无请求体，filename/mime_type 走 query 参数（同 ListFiles 的网关语义）。
//
//	@Tags        文件
//	@Accept      json
//	@Produce     json
//	@Param       filename query string true "目标文件名"
//	@Param       mime_type query string true "文件 MIME 类型"
//	@Success     200       {object} common.Response{data=blogv1.GetPresignedPutURLReply}
//	@Failure     400       {object} common.Response
//	@Router      /api/v1/files/presigned-upload [get]
func (h *FileHandler) GetPresignedPutURL(c *gin.Context) {
	req := &blogv1.GetPresignedPutURLRequest{}
	// query 参数合并：参数缺失时保持零值交下游校验（blog 侧对 filename/mime_type 有校验链）
	bindQueryString(c, "filename", &req.Filename)
	bindQueryString(c, "mime_type", &req.MimeType)
	callProto(c, req, func(ctx context.Context, m proto.Message) (proto.Message, error) {
		return h.hub.File.GetPresignedPutURL(ctx, m.(*blogv1.GetPresignedPutURLRequest))
	})
}

// CreatePresignedUpload 创建预签名上传：校验文件名/类型/大小后由 blog 服务签发
// MinIO 预签名直传 URL（对象键由服务端生成，须登录）。
//
//	@Tags        文件
//	@Accept      json
//	@Produce     json
//	@Param       request body blogv1.CreatePresignedUploadRequest true "创建请求（filename、mime_type、size 文件字节数）"
//	@Success     200     {object} common.Response{data=blogv1.CreatePresignedUploadReply}
//	@Failure     400     {object} common.Response
//	@Router      /api/v1/files/presigned-uploads [post]
func (h *FileHandler) CreatePresignedUpload(c *gin.Context) {
	req := &blogv1.CreatePresignedUploadRequest{}
	callProto(c, req, func(ctx context.Context, m proto.Message) (proto.Message, error) {
		return h.hub.File.CreatePresignedUpload(ctx, m.(*blogv1.CreatePresignedUploadRequest))
	})
}

// CompletePresignedUpload 完成预签名上传：客户端直传 MinIO 后登记文件记录
// （须登录；blog 侧仅接受其自身签发的 uploads/ 前缀对象键）。
//
//	@Tags        文件
//	@Accept      json
//	@Produce     json
//	@Param       request body blogv1.CompletePresignedUploadRequest true "完成请求（object_key 为预签名签发时返回的对象键）"
//	@Success     200     {object} common.Response{data=blogv1.CompletePresignedUploadReply}
//	@Failure     400     {object} common.Response
//	@Router      /api/v1/files/presigned-uploads/complete [post]
func (h *FileHandler) CompletePresignedUpload(c *gin.Context) {
	req := &blogv1.CompletePresignedUploadRequest{}
	callProto(c, req, func(ctx context.Context, m proto.Message) (proto.Message, error) {
		return h.hub.File.CompletePresignedUpload(ctx, m.(*blogv1.CompletePresignedUploadRequest))
	})
}

// ===================== query 参数合并辅助 =====================
//
// 分页/数字类 query 绑定复用 article.go 的共享辅助 bindQueryInt32
// （语义与 BindPath* 对齐：参数缺失时跳过保持零值、解析失败写 400 信封
// 返回 false），本文件不再重复声明。bindQueryString 仅文件域两处字符串
// query（filename/mime_type）使用，暂留本文件——若后续有第二个使用方
// 应上移 handle.go 与 BindPathString 同置。

// bindQueryString 将 query 参数 name 原样写入字符串字段 dst（缺失时跳过）。
func bindQueryString(c *gin.Context, name string, dst *string) bool {
	if v := c.Query(name); v != "" {
		*dst = v
	}
	return true
}
