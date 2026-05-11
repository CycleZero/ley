package jwt

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"go.etcd.io/etcd/client/v3"
)

// EtcdJWTConfig etcd 中存储的 JWT 配置格式
type EtcdJWTConfig struct {
	SigningKey   string `json:"signing_key"`
	Issuer       string `json:"issuer"`
	ExpiredHours int    `json:"expired_hours,omitempty"`
}

// LoadEtcdJWTConfig 从 etcd 读取 JWT 配置
func LoadEtcdJWTConfig(ctx context.Context, etcd *clientv3.Client, keyPath string) (*EtcdJWTConfig, error) {
	resp, err := etcd.Get(ctx, keyPath)
	if err != nil {
		return nil, fmt.Errorf("etcd get %s: %w", keyPath, err)
	}
	if len(resp.Kvs) == 0 {
		return nil, fmt.Errorf("etcd key not found: %s", keyPath)
	}

	var cfg EtcdJWTConfig
	if err := json.Unmarshal(resp.Kvs[0].Value, &cfg); err != nil {
		return nil, fmt.Errorf("parse etcd JWT config: %w", err)
	}
	if cfg.SigningKey == "" {
		return nil, fmt.Errorf("signing_key is empty in etcd config")
	}
	return &cfg, nil
}

// ToJWTConfig 转换为 JWT Config
func (c *EtcdJWTConfig) ToJWTConfig() *Config {
	expires := time.Hour * 24
	if c.ExpiredHours > 0 {
		expires = time.Duration(c.ExpiredHours) * time.Hour
	}
	return &Config{
		SigningKey:  c.SigningKey,
		Issuer:      c.Issuer,
		ExpiredTime: expires,
	}
}

// NewJWTEtcd 从 etcd 读取 JWT 配置并创建实例（一次性读取，不 watch）
func NewJWTEtcd(ctx context.Context, etcd *clientv3.Client, keyPath string) (JWT, error) {
	cfg, err := LoadEtcdJWTConfig(ctx, etcd, keyPath)
	if err != nil {
		return nil, err
	}
	return NewJWT(cfg.ToJWTConfig()), nil
}

// WatchEtcdJWTConfig 监听 etcd 键变更，返回变更通知通道
// keyPath: etcd 键路径
// 返回的 channel 在 etcd watch 关闭时关闭
func WatchEtcdJWTConfig(ctx context.Context, etcd *clientv3.Client, keyPath string) <-chan *EtcdJWTConfig {
	ch := make(chan *EtcdJWTConfig, 1)
	watchCh := etcd.Watch(ctx, keyPath)

	go func() {
		defer close(ch)
		for wr := range watchCh {
			for _, ev := range wr.Events {
				if ev.Kv == nil {
					continue
				}
				var cfg EtcdJWTConfig
				if err := json.Unmarshal(ev.Kv.Value, &cfg); err != nil {
					continue
				}
				if cfg.SigningKey == "" {
					continue
				}
				select {
				case ch <- &cfg:
				case <-ctx.Done():
					return
				}
			}
		}
	}()

	return ch
}
