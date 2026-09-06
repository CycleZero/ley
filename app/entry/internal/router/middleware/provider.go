// provider.go —— 中间件层 Wire ProviderSet。
//
// 提供依赖远程业务配置（etcd）才能构建的中间件原料（如 JWT 解析器），
// 供路由层 RegisteredMiddleWire 及后续 wave 复用，避免各处各自拼接配置。
package middleware

import (
	"time"

	"github.com/CycleZero/ley/app/entry/conf"
	jwtpkg "github.com/CycleZero/ley/pkg/jwt"
	"github.com/google/wire"
)

// MiddleWireProviderSet 中间件层 Wire ProviderSet。
var MiddleWireProviderSet = wire.NewSet(NewJWT)

// NewJWT 依据远程业务配置构建 JWT 解析器（供认证中间件消费）。
//
// 配置缺失（etcd 尚未上传业务配置）时以空密钥构建：签发/解析必然失败并降级为
// 401——「认证全部失败」比「nil 空指针 panic」安全，配置上传后重启即恢复。
// 注意：解析器构建于启动期，JWT 密钥热更需重启 entry 生效。
func NewJWT(h *conf.RemoteConfigHolder) jwtpkg.JWT {
	j := h.Get().JWT
	return jwtpkg.NewJWT(&jwtpkg.Config{
		SigningKey:         j.Secret,
		Issuer:             j.Issuer,
		ExpiredTime:        time.Duration(j.AccessTTL),
		RefreshExpiredTime: time.Duration(j.RefreshTTL),
	})
}
