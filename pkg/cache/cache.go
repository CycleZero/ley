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
	// Set 设置缓存，expiration=0 表示永不过期
	Set(ctx context.Context, key string, value any, expiration time.Duration) error
	// Delete 删除缓存
	Delete(ctx context.Context, key string) error
	// Exists 判断键是否存在
	Exists(ctx context.Context, key string) (bool, error)
	// TTL 获取键剩余过期时间，返回-2表示键不存在，-1表示永不过期
	TTL(ctx context.Context, key string) (time.Duration, error)
	// Flush 清空所有缓存
	Flush(ctx context.Context) error
	// Close 关闭缓存客户端（释放资源）
	Close() error
}
