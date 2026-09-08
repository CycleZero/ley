package common

import (
	"strings"

	"github.com/gin-gonic/gin"

	pkgmeta "github.com/CycleZero/ley/pkg/meta"
)

// ===================== 请求元数据（gin 上下文传递） =====================
//
// 类型设计：gin handler 之间传递请求元数据使用本文件的 RequestMetaData（内部形态，
// 字段镜像 pkg/meta.RequestMetaData，避免直接依赖 pkg/meta 的命名造成混淆）。
// 需要注入下游 gRPC 调用时，统一经 BuildRequestMeta 转为 *pkg/meta.RequestMetaData，
// 再交给 pkg/meta.NewClientCtx 写入 gRPC metadata —— 转换只发生在这一个出口。

// requestMetaKey gin.Context 中存取请求元数据的私有 key。
const requestMetaKey = "ley.request-meta"

// RequestMetaData gin 上下文内传递的请求元数据（认证中间件种入）。
// 字段镜像 pkg/meta.RequestMetaData：Auth 为 JWT 解析结果，RealClientIp 在
// BuildRequestMeta 时按客户端 IP 优先级解析填充。
type RequestMetaData struct {
	Auth         Auth   // 认证信息；未登录时为零值
	RealClientIp string // 真实客户端 IP（X-Forwarded-For 首段优先）
	AccessToken  string // 原始 access token（透传下游，供登出吊销）
}

// Auth 认证信息（镜像 pkg/meta.Auth 字段）。
type Auth struct {
	UserID   uint64 // 用户ID
	UserName string // 用户名
	Role     string // 角色：reader/author/admin
}

// SetRequestMeta 将请求元数据写入 gin 上下文。
// 认证中间件在 JWT 校验通过后调用，供后续 handler / BuildRequestMeta 读取。
func SetRequestMeta(c *gin.Context, m *RequestMetaData) {
	c.Set(requestMetaKey, m)
}

// GetRequestMeta 从 gin 上下文读取请求元数据。
// 中间件未种入（匿名请求或中间件未挂载）时返回 nil，调用方需自行兜底。
func GetRequestMeta(c *gin.Context) *RequestMetaData {
	v, ok := c.Get(requestMetaKey)
	if !ok {
		return nil
	}
	m, _ := v.(*RequestMetaData)
	return m
}

// BuildRequestMeta 组装下游 gRPC 调用所需的完整请求元数据：
//   - 认证信息：取自 gin 上下文（认证中间件经 SetRequestMeta 种入），未认证时为空；
//   - RealClientIp：按 X-Forwarded-For 首段 → X-Real-IP → c.ClientIP() 优先级解析。
//
// 返回 *pkg/meta.RequestMetaData，可直接经 pkg/meta.NewClientCtx 注入 gRPC metadata 透传下游。
func BuildRequestMeta(c *gin.Context) *pkgmeta.RequestMetaData {
	out := &pkgmeta.RequestMetaData{RealClientIp: realClientIP(c)}
	if m := GetRequestMeta(c); m != nil {
		out.Auth = pkgmeta.Auth{
			UserID:   m.Auth.UserID,
			UserName: m.Auth.UserName,
			Role:     m.Auth.Role,
		}
		out.AccessToken = m.AccessToken
	}
	return out
}

// realClientIP 解析真实客户端 IP：优先取 X-Forwarded-For 首段（逗号分隔、去除空白，
// 防止伪造方追加段与空白干扰），其次取 X-Real-IP，最后回落到 gin 直连地址
// （c.ClientIP() 解析 RemoteAddr）。
func realClientIP(c *gin.Context) string {
	// X-Forwarded-For 形如 "1.2.3.4, 10.0.0.1"，取第一段即可
	if xff := c.GetHeader("X-Forwarded-For"); xff != "" {
		first, _, _ := strings.Cut(xff, ",")
		if first = strings.TrimSpace(first); first != "" {
			return first
		}
	}
	if xrip := strings.TrimSpace(c.GetHeader("X-Real-IP")); xrip != "" {
		return xrip
	}
	return c.ClientIP()
}
