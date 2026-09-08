package biz

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/CycleZero/ley/pkg/metrics"
	"github.com/go-kratos/kratos/v2/log"
)

// captureLogger 捕获 Kratos 日志，用于断言热路径日志级别与消息。
type captureLogger struct {
	mu      sync.Mutex
	entries []capturedLog
}

// capturedLog 单条捕获的日志记录。
type capturedLog struct {
	level log.Level
	msg   string
}

// Log 实现 Kratos log.Logger：提取 msg 键值并记录级别。
func (c *captureLogger) Log(level log.Level, keyvals ...any) error {
	msg := ""
	for i := 0; i+1 < len(keyvals); i += 2 {
		if k, _ := keyvals[i].(string); k == "msg" {
			msg, _ = keyvals[i+1].(string)
		}
	}
	c.mu.Lock()
	c.entries = append(c.entries, capturedLog{level: level, msg: msg})
	c.mu.Unlock()
	return nil
}

// hasLevel 判断是否存在指定级别且消息包含 substr 的日志。
func (c *captureLogger) hasLevel(level log.Level, substr string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, e := range c.entries {
		if e.level == level && strings.Contains(e.msg, substr) {
			return true
		}
	}
	return false
}

// 场景 S3（指标回归）：recordResult 必须按 result 标签分别计入成功/失败。
func TestRecordResultCountsByResult(t *testing.T) {
	p, err := metrics.New("blog-biz-observability-test")
	if err != nil {
		t.Fatalf("初始化 metrics 失败：%v", err)
	}
	defer func() { _ = p.Shutdown(context.Background()) }()

	counter := metrics.Counter("blog_obs_test_total", "测试计数")
	recordResult(context.Background(), counter, nil)
	recordResult(context.Background(), counter, errors.New("boom"))

	rec := httptest.NewRecorder()
	p.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/metrics", nil))
	body, _ := io.ReadAll(rec.Result().Body)
	for _, want := range []string{`result="success"`, `result="error"`} {
		if !strings.Contains(string(body), want) {
			t.Fatalf("/metrics 缺少标签 %s，实际：%s", want, body)
		}
	}
}

// 场景 S3（热路径日志）：GetArticle 未命中必须打 Warn，而非静默返回 404。
func TestGetArticleMissLogsWarn(t *testing.T) {
	capture := &captureLogger{}
	uc, _, _, _, _ := setupArticleUseCase()
	uc.log = log.NewHelper(capture)

	if _, err := uc.GetArticle(context.Background(), "no-such-slug"); err == nil {
		t.Fatal("预期文章不存在错误")
	}
	if !capture.hasLevel(log.LevelWarn, "文章不存在") {
		t.Fatalf("未命中应打 Warn 日志，实际：%+v", capture.entries)
	}
}
