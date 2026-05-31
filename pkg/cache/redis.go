package cache

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisCache 基于 Redis 的缓存实现
type RedisCache struct {
	client *redis.Client
}

// NewRedisCache 创建 RedisCache 实例并验证连接
func NewRedisCache(host string, port int, password string, db int) Cache {
	addr := host + ":" + strconv.Itoa(port)
	client := redis.NewClient(&redis.Options{
		Addr:         addr,
		Password:     password,
		DB:           db,
		PoolSize:     20, // 限制连接池大小（默认 10×CPU=80，5 个服务共 400 连接过度浪费）
		MinIdleConns: 5,  // 保持少量热连接，减少冷启动延迟
	})

	// 启动时 PING 验证连接
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := client.Ping(ctx).Err(); err != nil {
		fmt.Printf("[cache] Redis 连接失败 %s (db=%d): %v\n", addr, db, err)
	} else {
		fmt.Printf("[cache] Redis 连接成功 %s (db=%d)\n", addr, db)
	}

	return &RedisCache{client: client}
}

// NewRedisCacheWithClient 使用已有的 redis.Client 创建 RedisCache
func NewRedisCacheWithClient(client *redis.Client) *RedisCache {
	return &RedisCache{client: client}
}

// Get 获取缓存值
func (r *RedisCache) Get(ctx context.Context, key string) ([]byte, error) {
	val, err := r.client.Get(ctx, key).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, ErrKeyNotFound
		}
		return nil, err
	}
	return []byte(val), nil
}

// GetObject 获取缓存并反序列化为对象，value 必须为指针
func (r *RedisCache) GetObject(ctx context.Context, key string, value any) error {
	val, err := r.Get(ctx, key)
	if err != nil {
		return err
	}
	return json.Unmarshal(val, value)
}

// MGet 批量获取缓存，返回 key → 原始字节映射。未命中或错误的 key 不出现在结果中。
func (r *RedisCache) MGet(ctx context.Context, keys []string) (map[string][]byte, error) {
	if len(keys) == 0 {
		return make(map[string][]byte), nil
	}
	vals, err := r.client.MGet(ctx, keys...).Result()
	if err != nil {
		return nil, err
	}
	result := make(map[string][]byte, len(keys))
	for i, v := range vals {
		if v == nil {
			continue
		}
		if s, ok := v.(string); ok {
			result[keys[i]] = []byte(s)
		}
	}
	return result, nil
}

// Set 设置缓存，expiration=0 表示永不过期
func (r *RedisCache) Set(ctx context.Context, key string, value any, expiration time.Duration) error {
	var data []byte
	var err error

	switch v := value.(type) {
	case []byte:
		data = v
	case string:
		data = []byte(v)
	default:
		data, err = json.Marshal(v)
		if err != nil {
			return err
		}
	}

	if expiration == 0 {
		return r.client.Set(ctx, key, data, 0).Err()
	}
	return r.client.SetEx(ctx, key, data, expiration).Err()
}

// Delete 删除缓存
func (r *RedisCache) Delete(ctx context.Context, key string) error {
	return r.client.Del(ctx, key).Err()
}

// Exists 判断键是否存在
func (r *RedisCache) Exists(ctx context.Context, key string) (bool, error) {
	n, err := r.client.Exists(ctx, key).Result()
	if err != nil {
		return false, err
	}
	return n > 0, nil
}

// TTL 获取键剩余过期时间
// 返回 -2 表示键不存在，返回 -1 表示永不过期
func (r *RedisCache) TTL(ctx context.Context, key string) (time.Duration, error) {
	return r.client.TTL(ctx, key).Result()
}

// Flush 清空当前数据库的所有缓存
func (r *RedisCache) Flush(ctx context.Context) error {
	return r.client.FlushDB(ctx).Err()
}

// Close 关闭 Redis 客户端连接
func (r *RedisCache) Close() error {
	return r.client.Close()
}

// SetNX 仅当键不存在时设置缓存，返回 true 表示设置成功。
func (r *RedisCache) SetNX(ctx context.Context, key string, value any, expiration time.Duration) (bool, error) {
	var data []byte
	var err error

	switch v := value.(type) {
	case []byte:
		data = v
	case string:
		data = []byte(v)
	default:
		data, err = json.Marshal(v)
		if err != nil {
			return false, err
		}
	}

	return r.client.SetNX(ctx, key, data, expiration).Result()
}

// Expire 设置键的过期时间，不修改值。
func (r *RedisCache) Expire(ctx context.Context, key string, expiration time.Duration) error {
	return r.client.Expire(ctx, key, expiration).Err()
}

// Incr 原子递增 1，返回递增后的值。
func (r *RedisCache) Incr(ctx context.Context, key string) (int64, error) {
	return r.client.Incr(ctx, key).Result()
}

// Decr 原子递减 1，返回递减后的值。
func (r *RedisCache) Decr(ctx context.Context, key string) (int64, error) {
	return r.client.Decr(ctx, key).Result()
}

// GetOrSet 读取缓存，未命中时调用 loader 获取值并写入缓存，返回最终值。
func (r *RedisCache) GetOrSet(ctx context.Context, key string, loader func() (any, error), expiration time.Duration) ([]byte, error) {
	val, err := r.Get(ctx, key)
	if err == nil {
		return val, nil
	}
	if !errors.Is(err, ErrKeyNotFound) {
		return nil, err
	}

	loaded, err := loader()
	if err != nil {
		return nil, err
	}

	if setErr := r.Set(ctx, key, loaded, expiration); setErr != nil {
		return nil, setErr
	}

	// 读取刚写入的值，确保返回格式一致
	return r.Get(ctx, key)
}

// GetDel 调用 go-redis 内置 GETDEL 命令，原子读取并删除缓存（Redis 6.2.0+）
func (r *RedisCache) GetDel(ctx context.Context, key string) ([]byte, error) {
	val, err := r.client.GetDel(ctx, key).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, ErrKeyNotFound
		}
		return nil, err
	}
	return []byte(val), nil
}

// GetObjectDel 原子读取、反序列化并删除缓存，value 必须为指针
func (r *RedisCache) GetObjectDel(ctx context.Context, key string, value any) error {
	val, err := r.GetDel(ctx, key)
	if err != nil {
		return err
	}
	return json.Unmarshal(val, value)
}

// ScanAll 使用 Redis SCAN 命令迭代匹配 pattern 的所有键，避免 KEYS 命令阻塞。
// 返回完整的键列表（空列表表示无匹配）。
func (r *RedisCache) ScanAll(ctx context.Context, pattern string) ([]string, error) {
	var keys []string
	var cursor uint64
	for {
		result, nextCursor, err := r.client.Scan(ctx, cursor, pattern, 100).Result()
		if err != nil {
			return nil, err
		}
		keys = append(keys, result...)
		cursor = nextCursor
		if cursor == 0 {
			break
		}
	}
	return keys, nil
}
