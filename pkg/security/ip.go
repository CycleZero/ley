package security

import (
	"context"
	"net"
	"strings"

	"ley/pkg/meta"

	"github.com/go-kratos/kratos/v2/transport"
)

// GetRealIp 从 context 中获取客户端真实 IP
// 优先从 Kratos 元数据获取，如果没有则从 HTTP headers 中提取
func GetRealIp(ctx context.Context) string {
	// 优先级1：从 Kratos 元数据中获取（全局透传的真实 IP）
	if requestMeta := meta.GetRequestMetaData(ctx); requestMeta != nil && requestMeta.RealClientIp != "" {
		return requestMeta.RealClientIp
	}

	// 优先级2：从 transport header 中获取
	if header, ok := transport.FromServerContext(ctx); ok {
		// 优先级顺序的头信息列表
		headers := []string{
			"EO-Client-Ip",     // EO CDN 客户端真实 IP（最高优先级）
			"Cf-Connecting-Ip", // Cloudflare
			"True-Client-Ip",   // Cloudflare Enterprise
			"X-Real-Ip",        // Nginx 代理
			"X-Forwarded-For",  // 标准代理头
			"X-Forwarded",      // 代理变体
			"Forwarded-For",    // 代理变体
			"Forwarded",        // RFC 7239
			"Client-Ip",        // 代理变体
		}

		for _, headerName := range headers {
			ip := header.RequestHeader().Get(headerName)
			if ip == "" {
				continue
			}

			// 处理逗号分隔的 IP 列表（X-Forwarded-For 可能包含多个 IP）
			if strings.Contains(ip, ",") {
				ips := strings.Split(ip, ",")
				// 取第一个 IP（通常是原始客户端 IP）
				ip = strings.TrimSpace(ips[0])
			}

			// 验证 IP 地址是否有效
			if IsValidIp(ip) && !IsPrivateIp(ip) {
				return ip
			}
		}

		// 如果所有方法都失败，回退到 X-Forwarded-For 中的第一个 IP（即使为内网 IP）
		if xff := header.RequestHeader().Get("X-Forwarded-For"); xff != "" {
			if strings.Contains(xff, ",") {
				ips := strings.Split(xff, ",")
				ip := strings.TrimSpace(ips[0])
				if IsValidIp(ip) {
					return ip
				}
			} else if IsValidIp(xff) {
				return xff
			}
		}
	}

	// 默认返回
	return "0.0.0.0"
}

// IsValidIp 验证 IP 地址是否有效
func IsValidIp(ip string) bool {
	parsedIp := net.ParseIP(ip)
	if parsedIp == nil {
		return false
	}

	// 检查是否为 IPv4 映射的 IPv6 地址
	if parsedIp.To4() != nil {
		return true
	}

	// IPv6 地址也认为是有效的
	return true
}

// IsPrivateIp 检查是否为内网或保留 IP
func IsPrivateIp(ip string) bool {
	parsedIp := net.ParseIP(ip)
	if parsedIp == nil {
		return false
	}

	// 检查是否为私有 IP 范围
	// IPv4 私有范围：10.0.0.0/8, 172.16.0.0/12, 192.168.0.0/16
	// IPv6 私有范围：fc00::/7
	if parsedIp.IsPrivate() {
		return true
	}

	// 检查是否为环回地址
	if parsedIp.IsLoopback() {
		return true
	}

	// 检查是否为链路本地地址
	if parsedIp.IsLinkLocalUnicast() || parsedIp.IsLinkLocalMulticast() {
		return true
	}

	// 特殊 IPv4 范围检查
	if parsedIp.To4() != nil {
		// 0.0.0.0/8
		if parsedIp[0] == 0 {
			return true
		}
		// 169.254.0.0/16 (链路本地)
		if parsedIp[0] == 169 && parsedIp[1] == 254 {
			return true
		}
		// 224.0.0.0/4 (组播)
		if parsedIp[0] >= 224 && parsedIp[0] <= 239 {
			return true
		}
		// 240.0.0.0/4 (保留)
		if parsedIp[0] >= 240 {
			return true
		}
	}

	return false
}
