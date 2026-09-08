package infra

import (
	"context"
	"encoding/json"
	"strconv"
	"time"

	"github.com/CycleZero/ley/pkg/log"
	"github.com/CycleZero/ley/pkg/trace"

	"github.com/redis/go-redis/v9"
)

type RedisClient struct {
	*redis.Client
}

func NewRedisClient(
	host string,
	port int,
	user string,
	password string,
	db int,
) *redis.Client {

	//log.GetLogger().Info("Redis连接信息: " + host + ":" + port)
	rdb := redis.NewClient(&redis.Options{
		Addr:     host + ":" + strconv.Itoa(port),
		Password: password,
		DB:       db, // use default DB
	})
	// 挂载 OTel Hook：所有 Redis 命令自动产生 span 与耗时指标。
	rdb.AddHook(trace.NewRedisHook())
	log.GetLogger().Info("连接Redis成功")
	return rdb
}

func NewCustomRedisClient(rdb *redis.Client) *RedisClient {
	return &RedisClient{rdb}
}

// target 为指针类型
func (r *RedisClient) GetObject(ctx context.Context, key string, target any) error {

	res, err := r.Get(ctx, key).Result()
	if err != nil {
		return err
	}
	return json.Unmarshal([]byte(res), target)
}

func (r *RedisClient) PutObject(ctx context.Context, key string, target any, expiration time.Duration) error {

	str, err := json.Marshal(target)
	if err != nil {
		return err
	}
	return r.SetEx(ctx, key, string(str), expiration).Err()
}
