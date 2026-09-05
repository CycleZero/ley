package jwt

import (
	"context"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	config "github.com/CycleZero/ley/api/gateway/config/v1"
	"github.com/CycleZero/ley/api/gateway/middleware/jwt/v1"
	"github.com/CycleZero/ley/pkg/cache"
	jwtpkg "github.com/CycleZero/ley/pkg/jwt"
	"github.com/CycleZero/ley/pkg/meta"
	clientv3 "go.etcd.io/etcd/client/v3"

	"github.com/CycleZero/ley/app/gateway/middleware"
	"github.com/go-kratos/kratos/v2/log"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
)

var logger = log.NewHelper(log.With(log.GetLogger(), "source", "middleware/jwt"))

func init() {
	middleware.Register("jwt", Middleware)
}

// jwtHolder 原子操作的 JWT 实例容器，支持运行时热更新密钥
type jwtHolder struct {
	instance atomic.Value // *jwtInstance
}

type jwtInstance struct {
	jwt       jwtpkg.JWT
	config    *v1.JWT
	blacklist jwtpkg.BlackListCache
}

func (h *jwtHolder) get() *jwtInstance {
	v := h.instance.Load()
	if v == nil {
		return nil
	}
	return v.(*jwtInstance)
}

func (h *jwtHolder) set(inst *jwtInstance) {
	h.instance.Store(inst)
}

// Middleware returns a JWT auth middleware.
func Middleware(c *config.Middleware) (middleware.Middleware, error) {
	options := &v1.JWT{}
	if c.Options != nil {
		if err := anypb.UnmarshalTo(c.Options, options, proto.UnmarshalOptions{Merge: true}); err != nil {
			return nil, err
		}
	}

	if !options.Enabled {
		return func(next http.RoundTripper) http.RoundTripper {
			return next
		}, nil
	}

	blacklist, err := newBlacklist(options)
	if err != nil {
		return nil, err
	}

	holder := &jwtHolder{}

	if options.EtcdEndpoints != "" && options.EtcdKeyPath != "" {
		// etcd 动态密钥源
		if _, err := loadEtcdConfig(options, holder, blacklist); err != nil {
			return nil, err
		}
		logger.Infow("msg", "JWT middleware initialized with etcd",
			"endpoints", options.EtcdEndpoints, "key_path", options.EtcdKeyPath, "skip_paths", options.SkipPaths,
		)
	} else {
		// 静态配置
		if options.SigningKey == "" {
			return nil, errUnauthorized("jwt middleware: signing_key is required")
		}
		expires := time.Hour * 24
		if options.ExpiredTime != nil {
			expires = options.ExpiredTime.AsDuration()
		}
		holder.set(&jwtInstance{
			jwt: jwtpkg.NewJWT(&jwtpkg.Config{
				SigningKey:  options.SigningKey,
				Issuer:      options.Issuer,
				ExpiredTime: expires,
			}),
			config:    options,
			blacklist: blacklist,
		})
		logger.Infow("msg", "JWT middleware initialized with static config", "skip_paths", options.SkipPaths)
	}

	return buildHandler(options, holder), nil
}

func loadEtcdConfig(options *v1.JWT, holder *jwtHolder, blacklist jwtpkg.BlackListCache) (*clientv3.Client, error) {
	dialTimeout := 5 * time.Second
	if options.EtcdDialTimeout != nil {
		dialTimeout = options.EtcdDialTimeout.AsDuration()
	}

	etcdClient, err := clientv3.New(clientv3.Config{
		Endpoints:   strings.Split(options.EtcdEndpoints, ","),
		DialTimeout: dialTimeout,
	})
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	ecfg, err := jwtpkg.LoadEtcdJWTConfig(ctx, etcdClient, options.EtcdKeyPath)
	if err != nil {
		etcdClient.Close()
		return nil, err
	}

	holder.set(&jwtInstance{
		jwt:       jwtpkg.NewJWT(ecfg.ToJWTConfig()),
		config:    options,
		blacklist: blacklist,
	})

	// 启动 watch 热更新
	go func() {
		ch := jwtpkg.WatchEtcdJWTConfig(context.Background(), etcdClient, options.EtcdKeyPath)
		for ecfg := range ch {
			holder.set(&jwtInstance{
				jwt:       jwtpkg.NewJWT(ecfg.ToJWTConfig()),
				config:    options,
				blacklist: blacklist,
			})
			logger.Infow("msg", "JWT config hot-reloaded from etcd")
		}
	}()

	return etcdClient, nil
}

func buildHandler(options *v1.JWT, holder *jwtHolder) middleware.Middleware {
	skipPaths := make(map[string]struct{})
	for _, path := range options.SkipPaths {
		skipPaths[path] = struct{}{}
	}

	required := true
	if options.Required != nil && !*options.Required {
		required = false
	}

	return func(next http.RoundTripper) http.RoundTripper {
		return middleware.RoundTripperFunc(func(req *http.Request) (*http.Response, error) {
			_, shouldSkip := skipPaths[req.URL.Path]

			authHeader := req.Header.Get("Authorization")
			if authHeader == "" {
				if !shouldSkip && required {
					return newUnauthorizedResponse(req, "missing authorization header"), nil
				}
				return next.RoundTrip(req)
			}

			tokenString := extractToken(authHeader)
			if tokenString == "" {
				if !shouldSkip && required {
					return newUnauthorizedResponse(req, "invalid authorization header format"), nil
				}
				return next.RoundTrip(req)
			}

			inst := holder.get()
			if inst == nil {
				return newUnauthorizedResponse(req, "jwt auth service not ready"), nil
			}

			claims, err := inst.jwt.ParseAccessToken(tokenString)
			if err != nil {
				if !shouldSkip && required {
					return newUnauthorizedResponse(req, "invalid or expired token"), nil
				}
				return next.RoundTrip(req)
			}

			// B-101: access token 已被吊销（登出/轮换黑名单）时，无论 required/skip 一律拒绝，
			// 保证「登出后令牌不可再使用」的契约。
			if isRevoked(inst.blacklist, tokenString) {
				return newUnauthorizedResponse(req, "token has been revoked"), nil
			}

			req.Header.Set(meta.AuthUserIDKey, strconv.FormatUint(claims.UserId, 10))
			req.Header.Set(meta.AuthUserNameKey, claims.UserName)
			if claims.Role != "" {
				req.Header.Set(meta.AuthUserRoleKey, claims.Role)
			}

			reqMeta := &meta.RequestMetaData{
				Auth: meta.Auth{
					UserID:   claims.UserId,
					UserName: claims.UserName,
					Role:     claims.Role,
				},
			}
			ctx := meta.NewClientCtx(req.Context(), reqMeta)
			return next.RoundTrip(req.WithContext(ctx))
		})
	}
}

// newBlacklist 依据 options 构建吊销黑名单检查器。
// 未配置 redis_addr 时返回 nil（不启用黑名单校验，向后兼容）。
func newBlacklist(options *v1.JWT) (jwtpkg.BlackListCache, error) {
	if options.RedisAddr == "" {
		return nil, nil
	}
	host, portStr, err := net.SplitHostPort(options.RedisAddr)
	if err != nil {
		return nil, errUnauthorized("jwt middleware: invalid redis_addr: " + err.Error())
	}
	port, err := strconv.Atoi(portStr)
	if err != nil {
		return nil, errUnauthorized("jwt middleware: invalid redis_addr port: " + err.Error())
	}
	c := cache.NewRedisCache(host, port, options.RedisPassword, int(options.RedisDb))
	return jwtpkg.NewBlackList(c), nil
}

// isRevoked 判断 token 是否已被吊销
func isRevoked(bl jwtpkg.BlackListCache, token string) bool {
	return bl != nil && bl.IsEnabled() && bl.IsTokenBlackListed(token)
}

func extractToken(authHeader string) string {
	const prefix = "Bearer "
	if strings.HasPrefix(authHeader, prefix) {
		return strings.TrimPrefix(authHeader, prefix)
	}
	const prefixLower = "bearer "
	if strings.HasPrefix(authHeader, prefixLower) {
		return strings.TrimPrefix(authHeader, prefixLower)
	}
	return ""
}

func newUnauthorizedResponse(req *http.Request, reason string) *http.Response {
	return &http.Response{
		StatusCode: http.StatusUnauthorized,
		Status:     "401 Unauthorized",
		Proto:      "HTTP/1.1",
		ProtoMajor: 1,
		ProtoMinor: 1,
		Header: http.Header{
			"Content-Type":           []string{"application/json"},
			"X-Content-Type-Options": []string{"nosniff"},
		},
		Body:          http.NoBody,
		ContentLength: -1,
		Request:       req,
	}
}

func errUnauthorized(msg string) error {
	return &http.ProtocolError{ErrorString: msg}
}
