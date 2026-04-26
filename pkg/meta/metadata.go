package meta

import (
	"context"
	"strconv"

	"github.com/go-kratos/kratos/v2/metadata"
)

// =====================  Kratos 规范元数据键（核心：全局透传前缀） =====================
// 遵循官方中间件源码规则：x-md-global- 前缀 → 全链路跨服务透传
const (
	MetaDataKeyPrefix = "x-md-global-"
	// AuthUserIDKey 认证用户ID（全局透传）
	AuthUserIDKey = MetaDataKeyPrefix + "auth-user-id"
	// AuthUserNameKey 认证用户名（全局透传）
	AuthUserNameKey = MetaDataKeyPrefix + "auth-user-name"
	// AuthRealClientIpKey 真实IP（全局透传）
	AuthRealClientIpKey = MetaDataKeyPrefix + "auth-real-ip"
)

// =====================  业务元数据结构体 =====================

// RequestMetaData 业务请求元数据（用户认证信息）
type RequestMetaData struct {
	Auth         Auth   // 认证信息
	RealClientIp string // 真实IP（全局透传）
}

// Auth 认证信息
type Auth struct {
	UserID   uint64 // 用户ID
	UserName string // 用户名
}

// =====================  核心转换方法（兼容 Kratos Metadata 结构） =====================

// IntoMetadata 将业务元数据转换为 Kratos 标准 Metadata
// 用于：注入到客户端上下文
func (m *RequestMetaData) IntoMetadata() metadata.Metadata {
	if m == nil {
		return metadata.Metadata{}
	}
	md := metadata.Metadata{}

	// 批量写入规范键值
	if m.Auth.UserID > 0 {
		md.Set(AuthUserIDKey, strconv.FormatUint(m.Auth.UserID, 10))
	}
	if m.Auth.UserName != "" {
		md.Set(AuthUserNameKey, m.Auth.UserName)
	}
	if m.RealClientIp != "" {
		md.Set(AuthRealClientIpKey, m.RealClientIp)
	}

	return md
}

// ParseMetadata 从 Kratos 标准 Metadata 解析业务元数据
// 用于：服务端/客户端读取元数据
func ParseMetadata(md metadata.Metadata) *RequestMetaData {
	meta := &RequestMetaData{}

	// 解析用户ID（带错误处理，避免 panic）
	if val := md.Get(AuthUserIDKey); val != "" {
		uid, _ := strconv.ParseUint(val, 10, 64)
		meta.Auth.UserID = uid
	}

	// 解析用户名
	meta.Auth.UserName = md.Get(AuthUserNameKey)
	meta.RealClientIp = md.Get(AuthRealClientIpKey)

	return meta
}

// =====================  上下文工具方法（贴合官方中间件源码） =====================

// NewClientCtx 客户端：创建带业务元数据的上下文
// 用法：客户端调用下游服务前注入
func NewClientCtx(ctx context.Context, meta *RequestMetaData) context.Context {
	return metadata.NewClientContext(ctx, meta.IntoMetadata())
}

// GetServerMeta 服务端：从上下文获取业务元数据（官方推荐）
// 用法：服务端业务代码直接调用
func GetServerMeta(ctx context.Context) *RequestMetaData {
	// 从 Kratos 服务端上下文解析（官方中间件自动注入）
	md, ok := metadata.FromServerContext(ctx)
	if !ok {
		return &RequestMetaData{Auth: Auth{}}
	}
	return ParseMetadata(md)
}

// GetClientMeta 客户端：从上下文获取业务元数据
// 用法：客户端本地读取元数据
func GetClientMeta(ctx context.Context) *RequestMetaData {
	md, ok := metadata.FromClientContext(ctx)
	if !ok {
		return &RequestMetaData{Auth: Auth{}}
	}
	return ParseMetadata(md)
}

// GetRequestMetaData 从上下文获取业务元数据（统一接口）
// 优先从服务端上下文获取，若失败则从客户端上下文获取
// 用法：业务代码统一调用，无需区分服务端/客户端
func GetRequestMetaData(ctx context.Context) *RequestMetaData {
	// 优先尝试从服务端上下文获取
	if md, ok := metadata.FromServerContext(ctx); ok {
		return ParseMetadata(md)
	}
	// 回退到客户端上下文
	if md, ok := metadata.FromClientContext(ctx); ok {
		return ParseMetadata(md)
	}
	return &RequestMetaData{Auth: Auth{}}
}
