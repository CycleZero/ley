package ratelimit

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	config "github.com/CycleZero/ley/api/gateway/config/v1"
	v1 "github.com/CycleZero/ley/api/gateway/middleware/ratelimit/v1"
	"github.com/CycleZero/ley/app/gateway/middleware"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/redis/go-redis/v9"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
)

var logger = log.NewHelper(log.With(log.GetLogger(), "source", "middleware/ratelimit"))

func init() {
	middleware.Register("ratelimit", Middleware)
}

var (
	once    sync.Once
	client  *redis.Client
	options *v1.RateLimit
)

// Middleware creates a Redis-based fixed-window rate limit middleware.
//
// Algorithm: fixed-window counter via INCR + EXPIRE.
// Key format: ratelimit:{ruleIdx}:{keyBy}:{value}:{windowUnix}
// If INCR returns > max_requests → 429 Too Many Requests.
func Middleware(c *config.Middleware) (middleware.Middleware, error) {
	opts := &v1.RateLimit{
		LimitStatusCode: 429,
		LimitMessage:    "请求过于频繁，请稍后再试",
	}
	if c.Options != nil {
		if err := anypb.UnmarshalTo(c.Options, opts, proto.UnmarshalOptions{Merge: true}); err != nil {
			return nil, err
		}
	}

	if len(opts.Rules) == 0 {
		return func(next http.RoundTripper) http.RoundTripper { return next }, nil
	}

	once.Do(func() {
		options = opts
		redisDB := 0
		if opts.RedisDb > 0 {
			redisDB = int(opts.RedisDb)
		}
		client = redis.NewClient(&redis.Options{
			Addr:     opts.RedisAddr,
			Password: opts.RedisPassword,
			DB:       redisDB,
		})
		logger.Infof("rate limit middleware initialized, rules=%d", len(opts.Rules))
	})

	return func(next http.RoundTripper) http.RoundTripper {
		return middleware.RoundTripperFunc(func(req *http.Request) (*http.Response, error) {
			rule := matchRule(req.URL.Path, options.Rules)
			if rule == nil {
				return next.RoundTrip(req)
			}

			key := buildKey(rule, req)
			if key == "" {
				return next.RoundTrip(req)
			}

			count, err := client.Incr(req.Context(), key).Result()
			if err != nil {
				logger.Warnf("rate limit redis incr failed: %v", err)
				return next.RoundTrip(req) // Redis 不可用则放行，避免阻塞
			}

			if count == 1 {
				client.Expire(req.Context(), key, rule.Window.AsDuration())
			}

			if count > rule.MaxRequests {
				return &http.Response{
					StatusCode: int(options.LimitStatusCode),
					Status:     http.StatusText(int(options.LimitStatusCode)),
					Header: http.Header{
						"Content-Type":           {"application/json"},
						"X-RateLimit-Limit":      {strconv.FormatInt(rule.MaxRequests, 10)},
						"X-RateLimit-Remaining":  {"0"},
						"Retry-After":            {strconv.FormatInt(int64(rule.Window.AsDuration().Seconds()), 10)},
					},
					Body: io.NopCloser(bytes.NewBufferString(
						`{"code":429,"message":"` + options.LimitMessage + `"}`)),
				}, nil
			}

			resp, err := next.RoundTrip(req)
			if resp != nil && resp.Header != nil {
				resp.Header.Set("X-RateLimit-Limit", strconv.FormatInt(rule.MaxRequests, 10))
				resp.Header.Set("X-RateLimit-Remaining", strconv.FormatInt(rule.MaxRequests-count, 10))
			}
			return resp, err
		})
	}, nil
}

// matchRule finds the first rule whose pattern matches the request path.
// Rules are checked in order; the first match wins.
func matchRule(path string, rules []*v1.RateLimit_Rule) *v1.RateLimit_Rule {
	for _, r := range rules {
		if r.Pattern == "*" {
			return r
		}
		if strings.HasPrefix(path, strings.TrimSuffix(r.Pattern, "*")) {
			return r
		}
		if r.Pattern == path {
			return r
		}
	}
	return nil
}

// buildKey constructs a Redis key for the current request based on the rule's keyBy strategy.
// Format: ratelimit:{ruleIdx}:{keyBy}:{value}:{windowUnix}
func buildKey(rule *v1.RateLimit_Rule, req *http.Request) string {
	identities := strings.Split(rule.KeyBy, ",")
	parts := make([]string, 0, len(identities))
	for _, id := range identities {
		switch strings.TrimSpace(id) {
		case "ip":
			parts = append(parts, extractIP(req))
		case "user":
			uid := extractUserID(req)
			if uid == "" {
				return "" // 用户未认证则不限流
			}
			parts = append(parts, uid)
		}
	}
	if len(parts) == 0 {
		return ""
	}

	windowSec := int64(rule.Window.AsDuration().Seconds())
	if windowSec < 1 {
		windowSec = 1
	}
	windowUnix := time.Now().Unix() / windowSec

	return "ratelimit:" + strings.ReplaceAll(rule.Pattern, "/", ".") + ":" +
		strings.Join(parts, ":") + ":" + strconv.FormatInt(windowUnix, 10)
}

func extractIP(req *http.Request) string {
	if xff := req.Header.Get("X-Forwarded-For"); xff != "" {
		ips := strings.Split(xff, ",")
		return strings.TrimSpace(ips[0])
	}
	if xri := req.Header.Get("X-Real-Ip"); xri != "" {
		return xri
	}
	host := req.RemoteAddr
	if idx := strings.LastIndex(host, ":"); idx > 0 {
		return host[:idx]
	}
	return host
}

func extractUserID(req *http.Request) string {
	val := req.Header.Get("X-User-Id")
	if val != "" {
		return val
	}
	// Try to extract from JWT claims injected into context
	type claimsGetter interface {
		GetUserId() uint64
	}
	if cg, ok := req.Context().Value("jwt_claims").(claimsGetter); ok {
		return strconv.FormatUint(cg.GetUserId(), 10)
	}
	return ""
}

var _ = json.Marshal
