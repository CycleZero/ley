package cache

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisCache 基于 Redis 的缓存实现
type RedisCache struct {
	client *redis.Client
}

// NewRedisCache 创建 RedisCache 实例
func NewRedisCache(host string, port int, password string, db int) Cache {
	return &RedisCache{
		client: redis.NewClient(&redis.Options{
			Addr:     host + ":" + strconv.Itoa(port),
			Password: password,
			DB:       db,
		}),
	}
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
