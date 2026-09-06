package conf

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/CycleZero/ley/pkg/log"
	"go.etcd.io/etcd/api/v3/mvccpb"
	clientv3 "go.etcd.io/etcd/client/v3"
	"go.uber.org/zap"
)

// TestMain 初始化全局日志器为丢弃日志（对齐生产「先 SetGlobalLogger 后使用」的顺序；
// pkg/log.GetLogger 在未初始化时直接 panic）
func TestMain(m *testing.M) {
	log.SetGlobalLogger(&log.Logger{Logger: zap.NewNop()})
	os.Exit(m.Run())
}

// =====================  yaml 解析正确性 =====================

// 与 etcd 键 ley/configs/entry/config.yaml 对齐的完整样例（含 auth 风格的时长字符串写法）
const fullSampleYAML = `
jwt:
  secret: "dev-secret-256bit-random-0123456789abcdef"
  issuer: "ley-entry"
  access_ttl: 900s
  refresh_ttl: 604800s
redis:
  host: 127.0.0.1
  port: 6379
  password: ""
  db: 0
cors:
  allow_origins:
    - http://localhost:3000
  allow_methods:
    - GET
    - POST
  allow_headers:
    - Content-Type
    - Authorization
  max_age: 86400
ratelimit:
  rps: 100
  burst: 200
`

func TestDecodeRemoteConfigFull(t *testing.T) {
	// Given 一份覆盖全部字段的远程业务配置 YAML
	cfg, err := decodeRemoteConfig([]byte(fullSampleYAML))
	if err != nil {
		t.Fatalf("解析完整 YAML 失败：%v", err)
	}

	// Then 各段配置均按预期解析（JWT 时长字符串转换为 time.Duration）
	if cfg.JWT.Secret != "dev-secret-256bit-random-0123456789abcdef" {
		t.Errorf("JWT.Secret 解析错误：%q", cfg.JWT.Secret)
	}
	if cfg.JWT.Issuer != "ley-entry" {
		t.Errorf("JWT.Issuer 解析错误：%q", cfg.JWT.Issuer)
	}
	if got := time.Duration(cfg.JWT.AccessTTL); got != 900*time.Second {
		t.Errorf("JWT.AccessTTL 解析错误：%v", got)
	}
	if got := time.Duration(cfg.JWT.RefreshTTL); got != 604800*time.Second {
		t.Errorf("JWT.RefreshTTL 解析错误：%v", got)
	}
	if cfg.Redis.Host != "127.0.0.1" || cfg.Redis.Port != 6379 || cfg.Redis.DB != 0 {
		t.Errorf("Redis 配置解析错误：%+v", cfg.Redis)
	}
	if len(cfg.CORS.AllowOrigins) != 1 || cfg.CORS.AllowOrigins[0] != "http://localhost:3000" {
		t.Errorf("CORS.AllowOrigins 解析错误：%v", cfg.CORS.AllowOrigins)
	}
	if len(cfg.CORS.AllowMethods) != 2 || cfg.CORS.MaxAge != 86400 {
		t.Errorf("CORS 其余字段解析错误：%+v", cfg.CORS)
	}
	if cfg.RateLimit.RPS != 100 || cfg.RateLimit.Burst != 200 {
		t.Errorf("RateLimit 配置解析错误：%+v", cfg.RateLimit)
	}
}

func TestDecodeRemoteConfigPartial(t *testing.T) {
	// Given 只含部分字段的 YAML（缺失字段应保持零值，不报错）
	cfg, err := decodeRemoteConfig([]byte("jwt:\n  secret: only-secret\n"))
	if err != nil {
		t.Fatalf("解析部分 YAML 失败：%v", err)
	}

	// Then 缺失字段为零值，已填字段生效
	if cfg.JWT.Secret != "only-secret" || cfg.JWT.Issuer != "" {
		t.Errorf("JWT 解析错误：%+v", cfg.JWT)
	}
	if cfg.Redis.Host != "" || cfg.Redis.Port != 0 {
		t.Errorf("缺失的 Redis 段应保持零值：%+v", cfg.Redis)
	}
}

func TestDecodeRemoteConfigInvalidYAML(t *testing.T) {
	// Given 非法 YAML
	// When 解码
	if _, err := decodeRemoteConfig([]byte("jwt: [unclosed")); err == nil {
		t.Fatal("非法 YAML 应返回错误")
	}
}

// Duration 支持整数秒写法（与字符串时长等价）
func TestDecodeRemoteConfigDurationIntSeconds(t *testing.T) {
	// Given 整数秒写法的 access_ttl
	cfg, err := decodeRemoteConfig([]byte("jwt:\n  access_ttl: 60\n"))
	if err != nil {
		t.Fatalf("解析失败：%v", err)
	}

	// Then 按 60 秒解析
	if got := time.Duration(cfg.JWT.AccessTTL); got != 60*time.Second {
		t.Errorf("整数秒应解析为 60s，实际：%v", got)
	}
}

func TestDecodeRemoteConfigInvalidDuration(t *testing.T) {
	// Given 非法时长字符串
	// When 解码
	if _, err := decodeRemoteConfig([]byte("jwt:\n  access_ttl: 不是时长\n")); err == nil {
		t.Fatal("非法时长字符串应返回错误")
	}
}

// =====================  加载（etcd GET → 解码） =====================

// fakeEtcdReader 内存假 etcd：返回预置的 GET 响应与 watch 事件流，零外部依赖。
//
// Watch 语义对齐 clientv3：首次调用返回预置事件通道，后续重订阅返回随 ctx 取消而关闭的空通道
// （模拟真实 clientv3 watch 在 ctx 取消时自动关闭）。
type fakeEtcdReader struct {
	getResp   *clientv3.GetResponse // GET 返回值
	getErr    error                 // GET 错误
	watchResp clientv3.WatchChan    // 预置的 watch 事件流（首轮）
}

func (f *fakeEtcdReader) Get(_ context.Context, _ string, _ ...clientv3.OpOption) (*clientv3.GetResponse, error) {
	return f.getResp, f.getErr
}

func (f *fakeEtcdReader) Watch(ctx context.Context, _ string, _ ...clientv3.OpOption) clientv3.WatchChan {
	if f.watchResp != nil {
		ch := f.watchResp
		f.watchResp = nil // 只喂一轮事件，后续轮次返回 ctx 门控的空通道
		return ch
	}
	ch := make(chan clientv3.WatchResponse)
	go func() {
		<-ctx.Done()
		close(ch)
	}()
	return ch
}

// kvOf 构造 etcd KeyValue（测试辅助）
func kvOf(key, value string) *mvccpb.KeyValue {
	return &mvccpb.KeyValue{Key: []byte(key), Value: []byte(value)}
}

func TestLoadRemoteConfigOK(t *testing.T) {
	// Given etcd 中存在配置键
	fake := &fakeEtcdReader{
		getResp: &clientv3.GetResponse{Kvs: []*mvccpb.KeyValue{kvOf(RemoteConfigPath, fullSampleYAML)}},
	}

	// When 加载
	cfg, err := loadRemoteConfig(context.Background(), fake, RemoteConfigPath)

	// Then 解析成功且内容正确
	if err != nil {
		t.Fatalf("加载失败：%v", err)
	}
	if cfg.JWT.Issuer != "ley-entry" {
		t.Errorf("加载结果错误：%+v", cfg.JWT)
	}
}

func TestLoadRemoteConfigNotFound(t *testing.T) {
	// Given etcd 返回空 Kvs（键不存在）
	fake := &fakeEtcdReader{getResp: &clientv3.GetResponse{}}

	// When 加载
	_, err := loadRemoteConfig(context.Background(), fake, RemoteConfigPath)

	// Then 返回哨兵错误 ErrRemoteConfigNotFound
	if !errors.Is(err, ErrRemoteConfigNotFound) {
		t.Fatalf("应返回 ErrRemoteConfigNotFound，实际：%v", err)
	}
}

func TestLoadRemoteConfigEtcdError(t *testing.T) {
	// Given etcd GET 本身失败
	fake := &fakeEtcdReader{getErr: errors.New("etcd 不可达")}

	// When 加载
	// Then 错误被包装返回（可被 errors.Is 无法匹配、%w 链保留原始原因）
	_, err := loadRemoteConfig(context.Background(), fake, RemoteConfigPath)
	if err == nil {
		t.Fatal("etcd 故障应返回错误")
	}
}

func TestLoadRemoteConfigNilClient(t *testing.T) {
	// Given 空 etcd 客户端
	// When 加载
	// Then 返回明确错误而非 panic
	if _, err := LoadRemoteConfig(context.Background(), nil, RemoteConfigPath); err == nil {
		t.Fatal("nil etcd 客户端应返回错误")
	}
}

// =====================  watch 监听（初次投递 + 事件解码投递） =====================

func TestWatchRemoteConfigDeliver(t *testing.T) {
	// Given etcd 中已有配置 v1（初次投递），且随后来一条 PUT 变更事件（v2，含删除事件干扰）
	const v1YAML = "jwt:\n  secret: v1-secret\n"
	const v2Full = "jwt:\n  secret: v2-secret\n  issuer: v2-issuer\nredis:\n  host: 10.0.0.2\n"

	watchCh := make(chan clientv3.WatchResponse, 1)
	watchCh <- clientv3.WatchResponse{
		Events: []*clientv3.Event{
			{Type: mvccpb.DELETE, Kv: kvOf(RemoteConfigPath, "")}, // 删除事件：应被忽略，不影响后续 PUT
			{Type: mvccpb.PUT, Kv: kvOf(RemoteConfigPath, v2Full)},
		},
	}
	close(watchCh)
	fake := &fakeEtcdReader{
		getResp:   &clientv3.GetResponse{Kvs: []*mvccpb.KeyValue{kvOf(RemoteConfigPath, v1YAML)}},
		watchResp: watchCh,
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// When 启动监听
	ch, err := watchRemoteConfig(ctx, fake, RemoteConfigPath)
	if err != nil {
		t.Fatalf("启动监听失败：%v", err)
	}

	// Then 在超时保护内收集投递的配置，最终必须收到 v2（删除事件被跳过）
	seen := map[string]bool{}
	deadline := time.After(2 * time.Second)
	for !seen["v2-secret"] {
		select {
		case cfg, ok := <-ch:
			if !ok {
				t.Fatal("配置通道提前关闭")
			}
			seen[cfg.JWT.Secret] = true
		case <-deadline:
			t.Fatalf("等待 v2 配置超时，已收到：%v", seen)
		}
	}
	// v1（初次值）可能因 deliver 丢弃旧值语义未送达，不做强制断言
}

func TestWatchRemoteConfigInitReadError(t *testing.T) {
	// Given 初次读取遇到非「键不存在」的 etcd 故障
	fake := &fakeEtcdReader{getErr: errors.New("etcd 不可达")}

	// When 启动监听
	// Then 直接返回错误且通道为 nil（调用方降级，不启动后台循环）
	ch, err := watchRemoteConfig(context.Background(), fake, RemoteConfigPath)
	if err == nil {
		t.Fatal("初次读取故障应返回错误")
	}
	if ch != nil {
		t.Fatal("出错时不应返回通道")
	}
}

func TestWatchRemoteConfigInitNotFound(t *testing.T) {
	// Given 键尚未上传（GET 空 Kvs）
	fake := &fakeEtcdReader{getResp: &clientv3.GetResponse{}}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// When 启动监听（配置上传前）
	ch, err := watchRemoteConfig(ctx, fake, RemoteConfigPath)

	// Then 不报错（等待后续 watch 事件补投），通道保持开启
	if err != nil {
		t.Fatalf("键不存在不应视为错误：%v", err)
	}
	cancel()
	if _, ok := <-ch; ok {
		// 取消后通道被关闭（close 语义），此处无需进一步断言
		t.Fatal("取消后通道应关闭")
	}
}

// =====================  持有者（无 etcd 降级路径） =====================

func TestNewRemoteConfigHolderWithoutEtcd(t *testing.T) {
	// Given 无 etcd 客户端（开发/单测环境）
	h, stop := NewRemoteConfigHolder(nil)
	defer stop()

	// Then 持有者可安全使用，Get 返回非 nil 零值配置（不 panic、不启动监听）
	cfg := h.Get()
	if cfg == nil {
		t.Fatal("Get 不应返回 nil")
	}
	stop() // stop 幂等（context.CancelFunc 多次调用安全）
}
