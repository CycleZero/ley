package proxy

// ===================== 站点配置域代理 handler（site.go） =====================
//
// 覆盖 blog SiteService 全部 8 个 RPC 的 HTTP→gRPC 转发。handler 只做「构造
// 请求消息 + 路径参数合并」，绑定/元数据注入/调用/信封响应全部复用 handle.go
// 的 callProto 骨架（设计约定见 handle.go 头部注释），本站点域不再重复。
//
// 请求体嵌套结构说明（UpdateSiteConfig / UpdateMusicPlaylist / UploadBackground）：
// blog.proto 中上述 RPC 的 HTTP 注解均为 body:"*"——整个 proto 请求消息即请求体，
// 因此 JSON 形态天然带一层包装键：更新站点配置为 {"config":{...}}、更新歌单为
// {"playlist":{...}}，UploadBackground 为 {"filename":...,"content":"<base64>"}
// （content 为 proto bytes 字段，JSON 传输按 base64 字符串编码）。handler 直接把
// body 用 protojson 解到请求消息即可——包装键由 protojson 自动剥开/填回，无需
// 手工拆包，响应序列化（UseProtoNames）后同样保持该包装形态。
//
// 鉴权不在 handler 内做：各 RPC 的公开/管理员属性由路由 wave 按 T4 认证中间件
// 分组挂载决定（proto 内已标注公开/管理员语义，见 api/blog/v1/blog.proto）。

import (
	"context"

	blogv1 "github.com/CycleZero/ley/api/blog/v1"
	"github.com/gin-gonic/gin"
	"google.golang.org/protobuf/proto"
)

// SiteHandler 站点配置域 handler 聚合：持有 ServiceHub，8 个方法一一对应
// SiteService 的 8 个 RPC（每 API 一个 handler，见 handle.go 设计约定 1）。
type SiteHandler struct {
	hub *ServiceHub
}

// NewSiteHandler 构造站点配置域 handler（供路由 wave 装配）。
func NewSiteHandler(hub *ServiceHub) *SiteHandler {
	return &SiteHandler{hub: hub}
}

// GetSiteConfig 获取站点配置（公开接口，无请求体）。
//
// @Summary 获取站点配置
// @Description 获取站点全局配置（公开）
// @Tags 站点
// @Success 200 {object} common.Response{data=blogv1.GetSiteConfigReply} "成功信封，data 为 {config:{...}}"
// @Router /api/v1/site/config [get]
func (h *SiteHandler) GetSiteConfig(c *gin.Context) {
	req := &blogv1.GetSiteConfigRequest{}
	callProto(c, req, func(ctx context.Context, m proto.Message) (proto.Message, error) {
		return h.hub.Site.GetSiteConfig(ctx, m.(*blogv1.GetSiteConfigRequest))
	})
}

// UpdateSiteConfig 更新站点配置（管理员）。
//
// 请求体 = UpdateSiteConfigRequest 全消息（body:"*"），JSON 形态为
// {"config":{...}}：config 字段包装在顶层 config 键下，protojson 解 body 时
// 自动把 config 子对象填入 req.Config，handler 无需手工拆包。
//
// @Summary 更新站点配置
// @Description 更新站点全局配置（管理员）
// @Tags 站点
// @Accept json
// @Param request body blogv1.UpdateSiteConfigRequest true "更新站点配置请求（config 嵌套包装）"
// @Success 200 {object} common.Response{data=blogv1.UpdateSiteConfigReply} "成功信封，data 为更新后的 {config:{...}}"
// @Router /api/v1/site/config [put]
func (h *SiteHandler) UpdateSiteConfig(c *gin.Context) {
	req := &blogv1.UpdateSiteConfigRequest{}
	callProto(c, req, func(ctx context.Context, m proto.Message) (proto.Message, error) {
		return h.hub.Site.UpdateSiteConfig(ctx, m.(*blogv1.UpdateSiteConfigRequest))
	})
}

// ListBackgrounds 背景图片列表（公开接口，无请求体）。
//
// @Summary 背景图片列表
// @Description 获取所有背景图片（公开）
// @Tags 站点
// @Success 200 {object} common.Response{data=blogv1.ListBackgroundsReply} "成功信封，data 为 {backgrounds:[...]}"
// @Router /api/v1/site/backgrounds [get]
func (h *SiteHandler) ListBackgrounds(c *gin.Context) {
	req := &blogv1.ListBackgroundsRequest{}
	callProto(c, req, func(ctx context.Context, m proto.Message) (proto.Message, error) {
		return h.hub.Site.ListBackgrounds(ctx, m.(*blogv1.ListBackgroundsRequest))
	})
}

// UploadBackground 上传背景图片（管理员）。
//
// 请求体 = UploadBackgroundRequest 全消息（body:"*"）：filename 为文件名，
// content 为图片二进制（proto bytes 字段，JSON 按 base64 字符串传输，protojson
// 解 body 时自动解码为字节），无路径参数。
//
// @Summary 上传背景图片
// @Description 上传新的背景图片（管理员）
// @Tags 站点
// @Accept json
// @Param request body blogv1.UploadBackgroundRequest true "上传背景请求（content 为 base64）"
// @Success 200 {object} common.Response{data=blogv1.UploadBackgroundReply} "成功信封，data 为上传后的背景信息"
// @Router /api/v1/site/backgrounds [post]
func (h *SiteHandler) UploadBackground(c *gin.Context) {
	req := &blogv1.UploadBackgroundRequest{}
	callProto(c, req, func(ctx context.Context, m proto.Message) (proto.Message, error) {
		return h.hub.Site.UploadBackground(ctx, m.(*blogv1.UploadBackgroundRequest))
	})
}

// DeleteBackground 删除背景图片（管理员）。
//
// 路径参数 :id 经 BindPathUint64 合并进 req.Id（失败已写 400 信封并 return）；
// 请求体不得携带 id 字段（REST 契约，见 handle.go 设计约定 2）。
//
// @Summary 删除背景图片
// @Description 删除指定背景图片（管理员）
// @Tags 站点
// @Param id path integer true "背景图片 ID"
// @Success 200 {object} common.Response{data=blogv1.DeleteBackgroundReply} "成功信封，data 为 {}"
// @Router /api/v1/site/backgrounds/{id} [delete]
func (h *SiteHandler) DeleteBackground(c *gin.Context) {
	req := &blogv1.DeleteBackgroundRequest{}
	if !BindPathUint64(c, "id", &req.Id) {
		return
	}
	callProto(c, req, func(ctx context.Context, m proto.Message) (proto.Message, error) {
		return h.hub.Site.DeleteBackground(ctx, m.(*blogv1.DeleteBackgroundRequest))
	})
}

// SetActiveBackground 将指定背景设为当前激活（管理员）。
//
// 路径参数 :id 合并进 req.Id（同 DeleteBackground）；激活语义（同一时间仅一个
// 激活）由下游 blog 服务保证，entry 不做任何业务判断。
//
// @Summary 设为当前背景
// @Description 将指定背景图片设为当前激活状态（同一时间仅一个激活）
// @Tags 站点
// @Param id path integer true "背景图片 ID"
// @Success 200 {object} common.Response{data=blogv1.SetActiveBackgroundReply} "成功信封，data 为 {}"
// @Router /api/v1/site/backgrounds/{id}/active [put]
func (h *SiteHandler) SetActiveBackground(c *gin.Context) {
	req := &blogv1.SetActiveBackgroundRequest{}
	if !BindPathUint64(c, "id", &req.Id) {
		return
	}
	callProto(c, req, func(ctx context.Context, m proto.Message) (proto.Message, error) {
		return h.hub.Site.SetActiveBackground(ctx, m.(*blogv1.SetActiveBackgroundRequest))
	})
}

// GetMusicPlaylist 获取歌单（公开接口，无请求体）。
//
// @Summary 获取歌单
// @Description 获取当前歌单（公开）
// @Tags 站点
// @Success 200 {object} common.Response{data=blogv1.GetMusicPlaylistReply} "成功信封，data 为 {playlist:{tracks:[...]}}"
// @Router /api/v1/site/music/playlist [get]
func (h *SiteHandler) GetMusicPlaylist(c *gin.Context) {
	req := &blogv1.GetMusicPlaylistRequest{}
	callProto(c, req, func(ctx context.Context, m proto.Message) (proto.Message, error) {
		return h.hub.Site.GetMusicPlaylist(ctx, m.(*blogv1.GetMusicPlaylistRequest))
	})
}

// UpdateMusicPlaylist 更新歌单（管理员）。
//
// 请求体 = UpdateMusicPlaylistRequest 全消息（body:"*"），JSON 形态为
// {"playlist":{"tracks":[{title,artist,url,cover_url},...]}}：playlist 同样由
// protojson 自动剥壳填入 req.Playlist，无需手工拆包。
//
// @Summary 更新歌单
// @Description 更新歌单（管理员），音乐文件存储为外部链接
// @Tags 站点
// @Accept json
// @Param request body blogv1.UpdateMusicPlaylistRequest true "更新歌单请求（playlist 嵌套包装）"
// @Success 200 {object} common.Response{data=blogv1.UpdateMusicPlaylistReply} "成功信封，data 为更新后的 {playlist:{...}}"
// @Router /api/v1/site/music/playlist [put]
func (h *SiteHandler) UpdateMusicPlaylist(c *gin.Context) {
	req := &blogv1.UpdateMusicPlaylistRequest{}
	callProto(c, req, func(ctx context.Context, m proto.Message) (proto.Message, error) {
		return h.hub.Site.UpdateMusicPlaylist(ctx, m.(*blogv1.UpdateMusicPlaylistRequest))
	})
}
