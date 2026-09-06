package infra

import (
	"context"
	"time"

	"github.com/CycleZero/ley/app/entry/conf"
	"github.com/CycleZero/ley/pkg/cache"
	jwtpkg "github.com/CycleZero/ley/pkg/jwt"
	"github.com/CycleZero/ley/pkg/log"
	"go.uber.org/zap"
)

// blacklistProbeKey 启动连通性探测键（仅 Exists 查询，不写入任何数据）。
const blacklistProbeKey = "entry:jwt-blacklist:probe"

// logger 返回全局日志器；日志器尚未初始化（单测/启动早期）时降级为丢弃日志，
// 避免 nil *zap.Logger 方法调用 panic。生产路径 main.go 在 Wire 之前已初始化全局日志器。
func logger() *log.Logger {
	l := log.GetLogger()
	if l != nil && l.Logger != nil {
		return l
	}
	return &log.Logger{Logger: zap.NewNop()}
}

// NewBlacklistCache 构建 JWT 黑名单检查器（jwt.BlackListCache）。
//
// ⚠️ 关于「entry 直连 Redis」的边界说明（AGENTS.md 契约内例外）：
// 「entry 不直接访问存储」约束针对 gorm/minio/oss 等业务存储；JWT 吊销黑名单属认证
// 基础设施，旧 gateway 中间件即有直连 Redis 先例，故此处复用 pkg/cache.NewRedisCache
// + pkg/jwt.NewBlackList。除此之外 entry 不建立任何其他 Redis 使用面。
//
// 降级策略（以 pkg/jwt.NewBlackList 实际行为为准）：
//   - 未配置 Redis 地址（cfg.Host 为空）：直接返回 nil 缓存的黑名单，IsEnabled=false；
//   - 配置了地址但启动探测不可达（dev 无 redis）：同样降级为 IsEnabled=false
//     （黑名单检查被跳过），并输出 Warn——避免每次请求都打一次连接错误。
//
// 消费方（后续 wave 的 JWT 中间件）以 IsEnabled() 判定是否执行黑名单查询即可。
func NewBlacklistCache(cfg conf.RedisConfig) jwtpkg.BlackListCache {
	if cfg.Host == "" {
		logger().Warn("未配置远程 Redis，JWT 黑名单未启用（IsEnabled=false）")
		return jwtpkg.NewBlackList(nil)
	}

	c := cache.NewRedisCache(cfg.Host, cfg.Port, cfg.Password, cfg.DB)

	// 启动连通性探测：NewRedisCache 内部 Ping 失败仅打印不返回错误，此处显式探测一次。
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	_, err := c.Exists(ctx, blacklistProbeKey)
	cancel()
	if err != nil {
		logger().Warn("Redis 不可达，JWT 黑名单降级为未启用（IsEnabled=false）",
			zap.String("addr", cfg.Host), zap.Error(err))
		return jwtpkg.NewBlackList(nil)
	}
	logger().Info("JWT 黑名单已启用", zap.String("addr", cfg.Host))
	return jwtpkg.NewBlackList(c)
}
