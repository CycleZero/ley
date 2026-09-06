// remote.go —— entry 服务 etcd 远程业务配置（键：ley/configs/entry/config.yaml）。
//
// 与微服务两级配置体系对齐（AGENTS.md「Entry 入口服务特别说明·基础设施对齐」）：
//   - 本地引导配置（Bootstrap）由 conf.go LoadBootstrap 负责；
//   - 本文件负责 etcd 远程业务配置的加载（GET）、监听（watch 热更）与持有（Holder）。
//
// 不引入 kratos etcd config source：业务配置是普通 YAML 结构（非 proto），
// 且 entry 需要「原子替换整份配置 + 显式通道投递」语义，直接基于 clientv3 更可控。
package conf

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/CycleZero/ley/pkg/log"
	"go.etcd.io/etcd/api/v3/mvccpb"
	clientv3 "go.etcd.io/etcd/client/v3"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
)

// ErrRemoteConfigNotFound 远程业务配置键不存在（etcd 返回空 Kvs）。
//
// 属于可恢复状态：配置上传/变更后 watch 会自动投递，调用方可降级为默认值继续运行。
var ErrRemoteConfigNotFound = errors.New("远程业务配置不存在")

// =====================  业务配置结构（与 etcd 键内容一一对应） =====================

// JWTConfig JWT 鉴权配置（后续 wave 供 JWT 中间件消费）。
type JWTConfig struct {
	Secret     string   `yaml:"secret"`      // 签名密钥（生产必须为 256 位随机值）
	Issuer     string   `yaml:"issuer"`      // 令牌签发者标识
	AccessTTL  Duration `yaml:"access_ttl"`  // AccessToken 有效期
	RefreshTTL Duration `yaml:"refresh_ttl"` // RefreshToken 有效期
}

// RedisConfig Redis 配置。entry 直连 Redis 仅用于 JWT 黑名单（见 infra/redis.go 注释说明）。
type RedisConfig struct {
	Host     string `yaml:"host"`     // 主机地址
	Port     int    `yaml:"port"`     // 端口
	Password string `yaml:"password"` // 密码（可空）
	DB       int    `yaml:"db"`       // 逻辑库编号
}

// CORSConfig 跨域配置（后续 wave 供 CORS 中间件消费）。
type CORSConfig struct {
	AllowOrigins []string `yaml:"allow_origins"` // 允许的来源（如 http://localhost:3000）
	AllowMethods []string `yaml:"allow_methods"` // 允许的 HTTP 方法
	AllowHeaders []string `yaml:"allow_headers"` // 允许的请求头
	MaxAge       int64    `yaml:"max_age"`       // 预检结果缓存时长（秒）
}

// RateLimitConfig 限流配置（后续 wave 供限流中间件消费）。
type RateLimitConfig struct {
	RPS   int `yaml:"rps"`   // 每秒允许请求数
	Burst int `yaml:"burst"` // 令牌桶突发容量
}

// RemoteConfig 远程业务配置整体（etcd 键 ley/configs/entry/config.yaml 的 YAML 结构）。
type RemoteConfig struct {
	JWT       JWTConfig       `yaml:"jwt"`       // JWT 鉴权
	Redis     RedisConfig     `yaml:"redis"`     // Redis（黑名单）
	CORS      CORSConfig      `yaml:"cors"`      // 跨域
	RateLimit RateLimitConfig `yaml:"ratelimit"` // 限流
}

// Duration 兼容两种 yaml 写法的时长类型：
//   - 字符串时长 "900s"/"15m"（与 auth 业务配置 access_ttl: 900s 的运维写法一致）
//   - 整数秒 900
//
// 用法：time.Duration(cfg.JWT.AccessTTL) 转换后消费。
type Duration time.Duration

// UnmarshalYAML 实现 yaml.Node 自定义解码（yaml.v3 不原生支持 time.Duration）。
func (d *Duration) UnmarshalYAML(node *yaml.Node) error {
	if node.Tag == "!!int" {
		// 整数写法：按秒解析
		sec, err := strconv.ParseInt(node.Value, 10, 64)
		if err != nil {
			return fmt.Errorf("时长整数秒解析失败：%w", err)
		}
		*d = Duration(time.Duration(sec) * time.Second)
		return nil
	}
	dur, err := time.ParseDuration(node.Value)
	if err != nil {
		return fmt.Errorf("时长字符串解析失败（支持 \"900s\"/\"15m\" 或整数秒）：%w", err)
	}
	*d = Duration(dur)
	return nil
}

// decodeRemoteConfig 将 etcd 中的 YAML 字节解析为 RemoteConfig。
//
// 纯函数：加载（LoadRemoteConfig）与监听（watch 事件）共用同一解码路径，保证语义一致，也便于单测。
func decodeRemoteConfig(data []byte) (*RemoteConfig, error) {
	var cfg RemoteConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("解析远程业务配置 YAML 失败：%w", err)
	}
	return &cfg, nil
}

// =====================  etcd 最小读取接口（单测注入假实现，不依赖真实 etcd） =====================

// etcdReader 读取远程配置键所需的最小 etcd 能力。
// *clientv3.Client 天然满足；测试通过假实现注入预置的 GET 响应与 watch 事件流。
type etcdReader interface {
	Get(ctx context.Context, key string, opts ...clientv3.OpOption) (*clientv3.GetResponse, error)
	Watch(ctx context.Context, key string, opts ...clientv3.OpOption) clientv3.WatchChan
}

// 编译期断言：*clientv3.Client 实现了 etcdReader
var _ etcdReader = (*clientv3.Client)(nil)

// loadRemoteConfig 读取并解码远程业务配置（核心实现，见 LoadRemoteConfig 导出包装）。
func loadRemoteConfig(ctx context.Context, c etcdReader, key string) (*RemoteConfig, error) {
	if c == nil {
		return nil, errors.New("etcd 客户端为空")
	}
	resp, err := c.Get(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("读取远程业务配置失败：%w", err)
	}
	if len(resp.Kvs) == 0 {
		return nil, ErrRemoteConfigNotFound
	}
	return decodeRemoteConfig(resp.Kvs[0].Value)
}

// LoadRemoteConfig 一次性加载远程业务配置（etcd GET + YAML 解析）。
//
// 键不存在返回 ErrRemoteConfigNotFound（可降级默认值）；其余错误为 etcd/解析故障。
func LoadRemoteConfig(ctx context.Context, etcdClient *clientv3.Client, key string) (*RemoteConfig, error) {
	if etcdClient == nil {
		return nil, errors.New("etcd 客户端为空")
	}
	return loadRemoteConfig(ctx, etcdClient, key)
}

// =====================  watch 监听 =====================

// WatchRemoteConfig 监听远程业务配置键：
//  1. 初次立即读取当前值并投递（键不存在则跳过，等待配置上传后由 watch 事件补投）；
//  2. 后续 etcd watch 每次 PUT 变更解码后原子投递整份新配置（删除事件保留当前值）。
//
// 返回的通道在 ctx 取消或监听退出时由内部关闭，调用方 range 即自然退出；
// 通道缓冲为 1 且始终保留最新配置，消费缓慢时旧值会被丢弃（见 deliver）。
func WatchRemoteConfig(ctx context.Context, etcdClient *clientv3.Client, key string) (<-chan *RemoteConfig, error) {
	if etcdClient == nil {
		return nil, errors.New("etcd 客户端为空")
	}
	return watchRemoteConfig(ctx, etcdClient, key)
}

// watchRemoteConfig WatchRemoteConfig 的核心实现（参数收敛为 etcdReader 便于单测）。
func watchRemoteConfig(ctx context.Context, c etcdReader, key string) (<-chan *RemoteConfig, error) {
	ch := make(chan *RemoteConfig, 1)

	// 初次立即投递当前值；初始读取加 3s 超时，避免 etcd 长时间不可达时启动阻塞。
	getCtx, getCancel := context.WithTimeout(ctx, 3*time.Second)
	cfg, err := loadRemoteConfig(getCtx, c, key)
	getCancel()
	switch {
	case err == nil:
		ch <- cfg
	case errors.Is(err, ErrRemoteConfigNotFound):
		// 配置尚未上传：不视为错误，等待 watch 事件补投
	default:
		return nil, fmt.Errorf("初始化读取远程业务配置失败：%w", err)
	}

	go watchLoop(ctx, c, key, ch)
	return ch, nil
}

// watchLoop 持续监听 etcd 键变更：PUT 事件解码后投递，删除/异常保留当前值并告警。
//
// clientv3 断线自动重连；watch 通道异常关闭且 ctx 未取消时重新订阅，避免监听永久失效。
func watchLoop(ctx context.Context, c etcdReader, key string, ch chan *RemoteConfig) {
	// 保证退出时关闭通道，消费方 range 循环随之结束（防止 goroutine 泄漏）
	defer close(ch)

	for {
		watchCh := c.Watch(ctx, key)
		for resp := range watchCh {
			for _, ev := range resp.Events {
				if ev.Type != mvccpb.PUT {
					// 键被删除（DELETE/EXPIRE）：保留当前生效配置，仅记录告警
					entryLogger().Warn("远程业务配置键被删除，保留当前配置", zap.String("key", key))
					continue
				}
				cfg, err := decodeRemoteConfig(ev.Kv.Value)
				if err != nil {
					entryLogger().Warn("解码远程业务配置变更失败，忽略本次变更", zap.Error(err))
					continue
				}
				deliver(ch, cfg)
			}
			if resp.Err() != nil && ctx.Err() == nil {
				entryLogger().Warn("远程业务配置 watch 响应异常", zap.String("key", key), zap.Error(resp.Err()))
			}
		}
		if ctx.Err() != nil {
			return
		}
		entryLogger().Warn("远程业务配置 watch 中断，重新订阅", zap.String("key", key))
	}
}

// deliver 向通道投递最新配置：通道已满（消费方处理不及）时丢弃旧值后写入新值，
// 保证不阻塞 watch 循环且消费者始终能拿到最新配置。要求双向通道以支持丢弃旧值。
func deliver(ch chan *RemoteConfig, cfg *RemoteConfig) {
	select {
	case ch <- cfg:
	default:
		select {
		case <-ch:
		default:
		}
		ch <- cfg
	}
}

// =====================  配置持有者（启动加载 + 后台热更） =====================

// RemoteConfigHolder 远程业务配置持有者：
//   - 启动即 Load 当前值（失败仅 Warn，以默认零值启动，等 watch 补投）；
//   - 后台 watch goroutine 原子更新 atomic.Pointer，业务侧 Get() 随时取到最新配置。
//
// 供 Wire 注入到各中间件/基础设施 Provider，避免每处各自维护监听逻辑。
type RemoteConfigHolder struct {
	cfg  atomic.Pointer[RemoteConfig] // 当前生效配置（只读，勿修改返回值）
	stop context.CancelFunc           // 停止后台监听
}

// NewRemoteConfigHolder 创建并启动配置持有者（监听 conf.RemoteConfigPath 键）。
//
// etcdClient 为空（无 etcd 的开发/单测环境）时降级为纯默认值持有者，不启动监听；
// 返回的 stop 用于停止后台 watch goroutine（接入 wire cleanup，随应用退出调用）。
func NewRemoteConfigHolder(etcdClient *clientv3.Client) (*RemoteConfigHolder, func()) {
	h := &RemoteConfigHolder{}
	ctx, cancel := context.WithCancel(context.Background())
	h.stop = cancel

	stop := func() { cancel() }

	if etcdClient == nil {
		entryLogger().Warn("etcd 客户端为空，远程业务配置使用默认值且不启动监听")
		return h, stop
	}

	// 1. 启动即 Load：失败仅 Warn 并保持默认零值（配置上传后 watch 会补投最新值）
	loadCtx, loadCancel := context.WithTimeout(ctx, 3*time.Second)
	cfg, err := LoadRemoteConfig(loadCtx, etcdClient, RemoteConfigPath)
	loadCancel()
	switch {
	case err == nil:
		h.cfg.Store(cfg)
		entryLogger().Info("远程业务配置加载成功", zap.String("path", RemoteConfigPath))
	default:
		entryLogger().Warn("加载远程业务配置失败，使用默认配置（等待 etcd watch 更新）",
			zap.String("path", RemoteConfigPath), zap.Error(err))
	}

	// 2. 后台 watch 热更：每次变更整体替换当前配置指针
	ch, err := watchRemoteConfig(ctx, etcdClient, RemoteConfigPath)
	if err != nil {
		entryLogger().Warn("启动远程业务配置监听失败，仅使用启动时配置", zap.Error(err))
		return h, stop
	}
	go func() {
		for cfg := range ch {
			h.cfg.Store(cfg)
			entryLogger().Info("远程业务配置已热更新", zap.String("path", RemoteConfigPath))
		}
	}()
	return h, stop
}

// Get 返回当前生效的远程业务配置（只读，调用方不得修改返回值）。
// 尚未加载成功时返回零值配置，保证调用方永远拿到非 nil 配置。
func (h *RemoteConfigHolder) Get() *RemoteConfig {
	if cfg := h.cfg.Load(); cfg != nil {
		return cfg
	}
	return &RemoteConfig{}
}

// entryLogger 返回全局日志器；日志器尚未初始化（启动早期/单测）时降级为丢弃日志，
// 避免 nil *zap.Logger 方法调用 panic。生产路径 main.go 在 Wire 之前已初始化全局日志器。
func entryLogger() *log.Logger {
	l := log.GetLogger()
	if l != nil && l.Logger != nil {
		return l
	}
	return &log.Logger{Logger: zap.NewNop()}
}
