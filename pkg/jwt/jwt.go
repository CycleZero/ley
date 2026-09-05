package jwt

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/CycleZero/ley/pkg/cache"
	"github.com/CycleZero/ley/pkg/meta"
	"github.com/go-kratos/kratos/v2/middleware"
	"github.com/go-kratos/kratos/v2/transport"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

const (
	TokenTypeAccess  = "access"
	TokenTypeRefresh = "refresh"

	// 黑名单缓存键前缀
	blacklistKeyPrefix = "jwt:blacklist:"
)

type BlackListCache interface {
	// Add 将 token 加入黑名单并保持永久键（TTL=0）。
	Add(token string) error
	// AddWithTTL 将 token 加入黑名单，并按 token 剩余寿命设置 TTL，
	// 使黑名单键随 token 自然过期，避免 Redis 无限增长。
	AddWithTTL(token string, ttl time.Duration) error
	IsTokenBlackListed(token string) bool
	IsEnabled() bool
}

type blackList struct {
	cache   cache.Cache
	enabled bool
}

func (b *blackList) Add(token string) error {
	return b.AddWithTTL(token, 0)
}

func (b *blackList) AddWithTTL(token string, ttl time.Duration) error {
	if b.cache == nil {
		return errors.New("cache is not initialized")
	}
	ctx := context.Background()
	key := blacklistKeyPrefix + token
	return b.cache.Set(ctx, key, "1", ttl)
}

func (b *blackList) IsTokenBlackListed(token string) bool {
	if b.cache == nil {
		return false
	}
	ctx := context.Background()
	key := blacklistKeyPrefix + token
	exists, err := b.cache.Exists(ctx, key)
	if err != nil {
		return false
	}
	return exists
}

func (b *blackList) IsEnabled() bool {
	return b.enabled
}

func NewBlackList(cache cache.Cache) BlackListCache {
	b := &blackList{
		cache: cache,
	}
	if cache != nil {
		b.enabled = true
	} else {
		b.enabled = false
	}
	return b
}

// jwtPaser 配置
type Config struct {
	SigningKey  string        // 签名密钥
	ExpiredTime time.Duration // Token 过期时间
	Issuer      string        // 签发者
	//Cache       cache.Cache   // 缓存
}

type TokenPair struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

// Payload jwtPaser 载荷定义
type Payload struct {
	UserId   uint64 `json:"user_id"`   // 用户ID
	UserName string `json:"user_name"` // 用户名
	Role     string `json:"role"`      // 用户角色（reader/author/admin）
}

// Claims jwtPaser Claims 定义
type Claims struct {
	TokenType string
	Payload
	jwt.RegisteredClaims
}

// jwtPaser 组件
type jwtPaser struct {
	config *Config
}

// NewJWT 创建 jwtPaser 实例
func NewJWT(config *Config) JWT {
	//b := NewBlackList(config.Cache)
	return &jwtPaser{
		config: config,
		//blackList: b,
	}
}

// GenerateToken 生成 AccessToken
func (j *jwtPaser) GenerateToken(payload Payload) (string, error) {
	nowTime := time.Now()
	expireTime := nowTime.Add(j.config.ExpiredTime)

	claims := Claims{
		TokenType: TokenTypeAccess,
		Payload:   payload,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expireTime),
			IssuedAt:  jwt.NewNumericDate(nowTime),
			Issuer:    j.config.Issuer,
			// jti 保证同一秒内签发的 token 也互不相同（轮换机制依赖此唯一性）
			ID: uuid.NewString(),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(j.config.SigningKey))
}

// GenerateTokenPair 生成 Token 对（AccessToken 和 RefreshToken）
// AccessToken: 短期有效的访问令牌，使用配置的过期时间
// RefreshToken: 长期有效的刷新令牌，过期时间是 AccessToken 的 7 倍
func (j *jwtPaser) GenerateTokenPair(payload Payload) (*TokenPair, error) {
	// 生成 AccessToken
	accessToken, err := j.GenerateToken(payload)
	if err != nil {
		return nil, err
	}

	// 生成 RefreshToken（过期时间是 AccessToken 的 7 倍）
	nowTime := time.Now()
	refreshExpireTime := nowTime.Add(j.config.ExpiredTime * 7)

	refreshClaims := Claims{
		TokenType: TokenTypeRefresh,
		Payload:   payload,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(refreshExpireTime),
			IssuedAt:  jwt.NewNumericDate(nowTime),
			Issuer:    j.config.Issuer,
			ID:        uuid.NewString(),
		},
	}

	refreshToken := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims)
	refreshTokenString, err := refreshToken.SignedString([]byte(j.config.SigningKey))
	if err != nil {
		return nil, err
	}

	return &TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshTokenString,
	}, nil
}

// ParseToken 解析 Token（不校验类型）
func (j *jwtPaser) ParseToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(j.config.SigningKey), nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}

	return nil, errors.New("invalid token")
}

// ParseAccessToken 解析并校验 AccessToken
func (j *jwtPaser) ParseAccessToken(tokenString string) (*Claims, error) {
	claims, err := j.ParseToken(tokenString)
	if err != nil {
		return nil, err
	}

	if claims.TokenType != TokenTypeAccess {
		return nil, errors.New("invalid token type: expected access token")
	}

	return claims, nil
}

// ParseRefreshToken 解析并校验 RefreshToken
func (j *jwtPaser) ParseRefreshToken(tokenString string) (*Claims, error) {
	claims, err := j.ParseToken(tokenString)
	if err != nil {
		return nil, err
	}

	if claims.TokenType != TokenTypeRefresh {
		return nil, errors.New("invalid token type: expected refresh token")
	}

	return claims, nil
}

// Server 服务端中间件 - 用于各微服务验证 Token
// 只接受 AccessToken，拒绝 RefreshToken
func (j *jwtPaser) Server() middleware.Middleware {
	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req interface{}) (interface{}, error) {
			// 从 context 中获取 transport header
			if header, ok := transport.FromServerContext(ctx); ok {
				authHeader := header.RequestHeader().Get("Authorization")
				if authHeader != "" {
					// 解析 Bearer token
					tokenString := extractToken(authHeader)
					if tokenString != "" {
						// 使用 ParseAccessToken 校验 Token 类型
						claims, err := j.ParseAccessToken(tokenString)
						if err == nil {
							// 注入用户信息到 Kratos metadata（x-md-global- 全局透传）
							// biz 层通过 meta.GetRequestMetaData(ctx) 提取
							reqMeta := &meta.RequestMetaData{
								Auth: meta.Auth{
									UserID:   claims.UserId,
									UserName: claims.UserName,
									Role:     claims.Role,
								},
							}
							ctx = meta.NewClientCtx(ctx, reqMeta)
						}
					}
				}
			}
			return handler(ctx, req)
		}
	}
}

// Client 客户端中间件 - 用于服务间调用传递用户认证信息
//
// 从当前上下文提取用户身份（由 Server 中间件注入到 pkg/meta），
// 通过 Kratos metadata（x-md-global- 前缀）全链路透传到下游服务。
// 标准化方案替代旧的 X-User-Id / X-User-Name 自定义头。
func (j *jwtPaser) Client() middleware.Middleware {
	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req interface{}) (interface{}, error) {
			// 从 meta 包提取当前请求的用户身份（优先服务端上下文，回退客户端上下文）
			reqMeta := meta.GetRequestMetaData(ctx)
			if reqMeta != nil && reqMeta.Auth.UserID > 0 {
				// 将用户身份注入到出站请求的 Kratos metadata 中（x-md-global- 前缀自动全链路透传）
				ctx = meta.NewClientCtx(ctx, reqMeta)
			}
			return handler(ctx, req)
		}
	}
}

// extractToken 从 Authorization header 中提取 token
func extractToken(authHeader string) string {
	const prefix = "Bearer "
	if strings.HasPrefix(authHeader, prefix) {
		return strings.TrimPrefix(authHeader, prefix)
	}
	return authHeader
}

type JWT interface {
	GenerateToken(payload Payload) (string, error)
	GenerateTokenPair(payload Payload) (*TokenPair, error)
	ParseToken(tokenString string) (*Claims, error)
	ParseAccessToken(tokenString string) (*Claims, error)
	ParseRefreshToken(tokenString string) (*Claims, error)
	Server() middleware.Middleware
	Client() middleware.Middleware
}
