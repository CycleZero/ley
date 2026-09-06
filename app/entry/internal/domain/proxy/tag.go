package proxy

// ===================== 标签代理 handler（tag.go） =====================
//
// 承接 /api/v1/tags 的 3 个 REST 接口（Create/List/Delete），全部经 callProto
// 骨架转发到 blog 微服务 TagService（设计约定见 handle.go 顶部注释）。标签管理
// 属后台写操作，鉴权（JWT + 管理员角色）由路由层挂中间件完成，本文件只做
// 「绑定 → 转发 → 信封」。
//
// 路由表（权威，路由注册见 router wave）：
//
//	POST   /api/v1/tags        → CreateTag
//	GET    /api/v1/tags        → ListTags
//	DELETE /api/v1/tags/:id    → DeleteTag

import (
	"context"

	blogv1 "github.com/CycleZero/ley/api/blog/v1"
	"github.com/gin-gonic/gin"
	"google.golang.org/protobuf/proto"
)

// TagHandler 标签代理 handler：持有 ServiceHub，逐方法转发 blog TagService。
type TagHandler struct {
	hub *ServiceHub
}

// NewTagHandler 构造标签代理 handler。
func NewTagHandler(hub *ServiceHub) *TagHandler {
	return &TagHandler{hub: hub}
}

// CreateTag 创建标签。
//
//	@Summary 创建标签
//	@Description 创建博客标签（名称唯一、slug 生成由下游 blog 服务校验与兜底）
//	@Tags 标签
//	@Accept json
//	@Produce json
//	@Param body body blogv1.CreateTagRequest true "标签信息"
//	@Success 200 {object} common.Response{data=blogv1.CreateTagReply}
//	@Failure 400 {object} common.Response
//	@Failure 409 {object} common.Response
//	@Router /api/v1/tags [post]
func (h *TagHandler) CreateTag(c *gin.Context) {
	req := &blogv1.CreateTagRequest{}
	callProto(c, req, func(ctx context.Context, m proto.Message) (proto.Message, error) {
		return h.hub.Tag.CreateTag(ctx, m.(*blogv1.CreateTagRequest))
	})
}

// ListTags 标签列表。
//
//	@Summary 标签列表
//	@Description 返回全部标签（含文章计数；公开接口）
//	@Tags 标签
//	@Produce json
//	@Success 200 {object} common.Response{data=blogv1.ListTagsReply}
//	@Failure 500 {object} common.Response
//	@Router /api/v1/tags [get]
func (h *TagHandler) ListTags(c *gin.Context) {
	req := &blogv1.ListTagsRequest{}
	callProto(c, req, func(ctx context.Context, m proto.Message) (proto.Message, error) {
		return h.hub.Tag.ListTags(ctx, m.(*blogv1.ListTagsRequest))
	})
}

// DeleteTag 删除标签。
//
//	@Summary 删除标签
//	@Description 按 ID 删除标签（被文章引用时下游拒绝并清理关联）
//	@Tags 标签
//	@Produce json
//	@Param id path uint64 true "标签 ID"
//	@Success 200 {object} common.Response{data=blogv1.DeleteTagReply}
//	@Failure 400 {object} common.Response
//	@Failure 404 {object} common.Response
//	@Router /api/v1/tags/{id} [delete]
func (h *TagHandler) DeleteTag(c *gin.Context) {
	req := &blogv1.DeleteTagRequest{}
	// 路径参数合并须在 callProto 之前完成（见 handle.go 顶部约定）
	if !BindPathUint64(c, "id", &req.Id) {
		return
	}
	callProto(c, req, func(ctx context.Context, m proto.Message) (proto.Message, error) {
		return h.hub.Tag.DeleteTag(ctx, m.(*blogv1.DeleteTagRequest))
	})
}
