package middleware

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/go-kratos/kratos/v2/middleware"
	"github.com/go-kratos/kratos/v2/transport"
)

// EdgeOneConfig EdgeOne 鉴权配置
type EdgeOneConfig struct {
	AppKey      string   // APP 密钥
	AppTokenP   string   // APP 签名参数名
	AppTsP      string   // APP 时间戳参数名
	AppTTL      int64    // APP 有效时长（秒）
	WebKey      string   // Web 密钥
	WebTokenP   string   // Web 签名参数名
	WebTsP      string   // Web 时间戳参数名
	WebTTL      int64    // Web 有效时长（秒）
	Enabled     bool     // 是否启用鉴权
	ExemptPaths []string // 豁免路径
}

// EdgeOne EdgeOne 鉴权中间件
type EdgeOne struct {
	config *EdgeOneConfig
}

// NewEdgeOne 创建 EdgeOne 鉴权实例
func NewEdgeOne(config *EdgeOneConfig) *EdgeOne {
	return &EdgeOne{
		config: config,
	}
}

// Server EdgeOne 鉴权中间件（用于源站验证，主要用于备用端点）
func (e *EdgeOne) Server() middleware.Middleware {
	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req interface{}) (interface{}, error) {
			// 如果鉴权未启用，直接通过
			if !e.config.Enabled {
				return handler(ctx, req)
			}

			// 从 context 中获取 transport header
			if header, ok := transport.FromServerContext(ctx); ok {
				path := header.Operation()

				// 检查是否是豁免路径
				if e.isExemptPath(path) {
					return handler(ctx, req)
				}

				// 根据路径前缀确定使用的密钥和参数
				var key, tokenP, tsP string
				var ttl int64

				switch {
				case strings.HasPrefix(path, "/mobapi/"):
					key = e.config.AppKey
					tokenP = e.config.AppTokenP
					tsP = e.config.AppTsP
					ttl = e.config.AppTTL
				case strings.HasPrefix(path, "/webapi/"):
					key = e.config.WebKey
					tokenP = e.config.WebTokenP
					tsP = e.config.WebTsP
					ttl = e.config.WebTTL
				case strings.HasPrefix(path, "/api/"):
					// /api/* 使用 JWT，这里只放行，JWT 验证由另一个中间件处理
					return handler(ctx, req)
				default:
					// 未知路径，直接放行（或者根据需求返回错误）
					return handler(ctx, req)
				}

				// 验证签名
				if err := e.verifySignature(ctx, path, key, tokenP, tsP, ttl); err != nil {
					return nil, fmt.Errorf("edgeone auth failed: %w", err)
				}
			}

			return handler(ctx, req)
		}
	}
}

// verifySignature 验证签名
func (e *EdgeOne) verifySignature(ctx context.Context, path, key, tokenP, tsP string, ttl int64) error {
	if header, ok := transport.FromServerContext(ctx); ok {
		// 获取查询参数（这里简化处理，实际需要从请求中获取 query params）
		// 注意：kratos 的 transport.Header 主要用于 HTTP headers
		// 查询参数需要通过其他方式获取，这里用 header 模拟演示

		token := header.RequestHeader().Get(tokenP)
		tsStr := header.RequestHeader().Get(tsP)

		if token == "" || tsStr == "" {
			return errors.New("missing signature or timestamp")
		}

		ts, err := strconv.ParseInt(tsStr, 10, 64)
		if err != nil {
			return errors.New("invalid timestamp")
		}

		// 检查时间戳是否过期
		now := time.Now().Unix()
		if ts < now-ttl || ts > now+300 { // 允许 5 分钟的时钟偏差
			return errors.New("signature expired")
		}

		// 计算期望的签名
		expectedSign := e.calculateSignature(key, path, tsStr)

		// 比较签名
		if token != expectedSign {
			return errors.New("invalid signature")
		}

		return nil
	}

	return errors.New("transport header not found")
}

// calculateSignature 计算签名
func (e *EdgeOne) calculateSignature(key, path, ts string) string {
	signStr := key + path + ts
	hash := md5.Sum([]byte(signStr))
	return hex.EncodeToString(hash[:])
}

// isExemptPath 检查是否是豁免路径
func (e *EdgeOne) isExemptPath(path string) bool {
	for _, exemptPath := range e.config.ExemptPaths {
		if strings.HasPrefix(path, exemptPath) {
			return true
		}
	}
	return false
}

// GenerateAppSignature 生成 APP 签名（用于测试）
func (e *EdgeOne) GenerateAppSignature(path string) (string, string, string, string) {
	ts := strconv.FormatInt(time.Now().Unix(), 10)
	sign := e.calculateSignature(e.config.AppKey, path, ts)
	return e.config.AppTokenP, sign, e.config.AppTsP, ts
}

// GenerateWebSignature 生成 Web 签名（用于测试）
func (e *EdgeOne) GenerateWebSignature(path string) (string, string, string, string) {
	ts := strconv.FormatInt(time.Now().Unix(), 10)
	sign := e.calculateSignature(e.config.WebKey, path, ts)
	return e.config.WebTokenP, sign, e.config.WebTsP, ts
}
