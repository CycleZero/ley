// ratelimit.go —— per-IP 内存限流中间件（令牌桶）。
//
// 实现假设（与部署形态强相关，见注释）：
//   - **单实例主机网络部署**（docker-compose host 网络 / 单副本）：桶只存于本进程
//     内存，不做跨实例协调；多副本部署需换分布式实现（如 Redis 滑动窗口），
//     本 wave 按 AGENTS.md 约束不引入 Redis 限流。
//   - 桶以 c.ClientIP() 为键；若前置代理不可信，伪造 X-Forwarded-For 可绕过限流，
//     该场景需配合可信代理配置（gin TrustedProxies）后使用真实来源 IP。
package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/CycleZero/ley/app/entry/internal/common"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

const (
	// maxIPBuckets 桶数上限：达到后触发一次空闲桶清理，防止 IP 字典无限增长
	// （防内存泄漏的简单阈值清理，替代完整 LRU）。
	maxIPBuckets = 10_000
	// ipIdleTimeout 桶空闲回收时长：超过该时长无请求的 IP 桶被清理。
	ipIdleTimeout = 10 * time.Minute
)

// ipRateLimiter per-IP 令牌桶集合（内部状态，构造后由闭包持有）。
//
// 并发安全：buckets/lastSeen 读写全部经 mu 串行化；限流判定 rate.Limiter.Allow
// 本身线程安全，可在锁外执行。
type ipRateLimiter struct {
	mu       sync.Mutex
	limit    rate.Limit // 令牌补充速率（个/秒）
	burst    int        // 桶容量（突发上限）
	buckets  map[string]*rate.Limiter
	lastSeen map[string]time.Time // 各 IP 最近活跃时刻（供空闲清理）
}

// NewRateLimiter 构建 per-IP 限流中间件。
//
//	limit：令牌补充速率（golang.org/x/time/rate.Limit，如 rate.Limit(10)=10 req/s）；
//	burst：桶容量，允许的瞬时突发请求数。
//
// limit <= 0 视为「未启用」：直接放行不拦截——远程业务配置缺省（rps=0）即此形态，
// 组装侧（router.Register）在 rps>0 时才挂载本中间件，此处兜底保证构造安全。
func NewRateLimiter(limit rate.Limit, burst int) gin.HandlerFunc {
	if limit <= 0 {
		mwLogger().Warn("限流未启用：速率配置<=0，请求直接放行")
		return func(c *gin.Context) { c.Next() }
	}
	rl := &ipRateLimiter{
		limit:    limit,
		burst:    burst,
		buckets:  make(map[string]*rate.Limiter),
		lastSeen: make(map[string]time.Time),
	}
	return func(c *gin.Context) {
		if !rl.bucketFor(c.ClientIP()).Allow() {
			common.Fail(c, http.StatusTooManyRequests, http.StatusTooManyRequests, "请求过于频繁，请稍后再试")
			c.Abort()
			return
		}
		c.Next()
	}
}

// bucketFor 取（或懒创建）IP 对应令牌桶，并刷新其活跃时间。
//
// 桶数达到 maxIPBuckets 时先清理空闲桶再创建，避免 map 无限膨胀。
func (rl *ipRateLimiter) bucketFor(ip string) *rate.Limiter {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	if l, ok := rl.buckets[ip]; ok {
		rl.lastSeen[ip] = now
		return l
	}
	if len(rl.buckets) >= maxIPBuckets {
		rl.sweepIdle(now)
	}
	l := rate.NewLimiter(rl.limit, rl.burst)
	rl.buckets[ip] = l
	rl.lastSeen[ip] = now
	return l
}

// sweepIdle 清理 ipIdleTimeout 内无请求的桶（调用方须已持有 mu）。
func (rl *ipRateLimiter) sweepIdle(now time.Time) {
	for ip, seen := range rl.lastSeen {
		if now.Sub(seen) > ipIdleTimeout {
			delete(rl.buckets, ip)
			delete(rl.lastSeen, ip)
		}
	}
}
