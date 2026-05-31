package cache

import (
	"context"
	"errors"
	"time"
)

// 通用缓存错误
var (
	ErrKeyNotFound = errors.New("cache: key not found") // 键不存在
	ErrKeyExpired  = errors.New("cache: key expired")   // 键已过期
)

// Cache 通用缓存接口
// 所有缓存实现（Redis/内存/Memcached）都必须实现此接口
type Cache interface {
	// Get 获取缓存
	Get(ctx context.Context, key string) ([]byte, error)
	// GetObject 获取缓存并反序列化为对象,value 必须为指针
	GetObject(ctx context.Context, key string, value any) error
	// MGet 批量获取缓存，返回 key → 原始字节映射
	MGet(ctx context.Context, keys []string) (map[string][]byte, error)
	// Set 设置缓存，expiration=0 表示永不过期
	Set(ctx context.Context, key string, value any, expiration time.Duration) error
	// SetNX 仅当键不存在时设置缓存，返回 true 表示设置成功。
	// 用于分布式锁、防重、限频等需要原子性"检查并设置"的场景。
	SetNX(ctx context.Context, key string, value any, expiration time.Duration) (bool, error)
	// Delete 删除缓存
	Delete(ctx context.Context, key string) error
	// Exists 判断键是否存在
	Exists(ctx context.Context, key string) (bool, error)
	// TTL 获取键剩余过期时间，返回-2表示键不存在，-1表示永不过期
	TTL(ctx context.Context, key string) (time.Duration, error)
	// Expire 设置键的过期时间，不修改值。用于延长已有 key 的 TTL。
	Expire(ctx context.Context, key string, expiration time.Duration) error
	// Incr 原子递增 1，返回递增后的值。键不存在时从 0 开始递增。
	Incr(ctx context.Context, key string) (int64, error)
	// Decr 原子递减 1，返回递减后的值。键不存在时从 0 开始递减。
	Decr(ctx context.Context, key string) (int64, error)
	// GetOrSet 读取缓存，未命中时调用 loader 获取值并写入缓存，返回最终值。
	// 用于缓存穿透保护（cache-aside 模式的一站式封装）。
	GetOrSet(ctx context.Context, key string, loader func() (any, error), expiration time.Duration) ([]byte, error)
	// GetDel 原子读取并删除缓存（Redis GETDEL 命令），返回被删除的原始字节值。
	// key 不存在时返回 ErrKeyNotFound。用于验证码一次性令牌等"读取后即销毁"的场景。
	GetDel(ctx context.Context, key string) ([]byte, error)
	// GetObjectDel 原子读取、反序列化并删除缓存，value 必须为指针。
	// key 不存在时返回 ErrKeyNotFound。
	GetObjectDel(ctx context.Context, key string, value any) error
	// ScanAll 扫描并返回所有匹配指定模式的键（内部使用 SCAN 迭代，避免阻塞 Redis）。
	// 用于批量处理特定前缀的缓存键（如浏览量 flush、清理过期会话等）。
	ScanAll(ctx context.Context, pattern string) ([]string, error)
	// Flush 清空所有缓存
	Flush(ctx context.Context) error
	// Close 关闭缓存客户端（释放资源）
	Close() error
}
