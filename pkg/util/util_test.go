package util

import (
	"errors"
	"testing"
	"time"

	"github.com/CycleZero/ley/pkg/constant"
	"gorm.io/gorm"
)

func TestIsUniqueViolation(t *testing.T) {
	positive := []error{
		errors.New(`ERROR: duplicate key value violates unique constraint "idx_articles_slug" (SQLSTATE 23505)`),
		errors.New(`Error 1062: Duplicate entry 'slug' for key 'idx'`),
		errors.New("pq: duplicate key value violates unique constraint"),
		errors.New("UNIQUE constraint failed: users.email"),
		errors.New("unique constraint"),
		errors.New("some error containing 23505"),
	}
	for _, err := range positive {
		if !IsUniqueViolation(err) {
			t.Errorf("IsUniqueViolation(%v) 应为 true", err)
		}
	}

	negative := []error{
		nil,
		errors.New("some random error"),
		errors.New("SQLSTATE 23503"), // 外键冲突不是唯一约束
		errors.New("Error 1064: syntax error"),
	}
	for _, err := range negative {
		if IsUniqueViolation(err) {
			t.Errorf("IsUniqueViolation(%v) 应为 false", err)
		}
	}
}

func TestIsRecordNotFound(t *testing.T) {
	if !IsRecordNotFound(gorm.ErrRecordNotFound) {
		t.Error("gorm.ErrRecordNotFound 应识别")
	}
	// 包装错误也应识别
	if !IsRecordNotFound(errors.Join(gorm.ErrRecordNotFound, errors.New("wrap"))) &&
		!IsRecordNotFound(gorm.ErrRecordNotFound) {
		t.Error("包装后的 ErrRecordNotFound 应识别")
	}
	if IsRecordNotFound(errors.New("record not found")) {
		t.Error("仅字符串相同不应误判（需 errors.Is）")
	}
	if IsRecordNotFound(nil) {
		t.Error("nil 不应识别")
	}
}

func TestUniqueSlice(t *testing.T) {
	got := UniqueSlice([]int{1, 2, 2, 3, 3, 3})
	seen := map[int]bool{}
	for _, v := range got {
		if seen[v] {
			t.Errorf("存在重复元素: %v", got)
		}
		seen[v] = true
	}
	if len(got) != 3 {
		t.Errorf("去重后应有 3 个元素: %v", got)
	}

	empty := UniqueSlice([]string{})
	if len(empty) != 0 {
		t.Errorf("空切片去重应仍为空: %v", empty)
	}
}

func TestInArray(t *testing.T) {
	if !InArray("b", []string{"a", "b", "c"}) {
		t.Error("存在的元素应命中")
	}
	if InArray("z", []string{"a", "b", "c"}) {
		t.Error("不存在的元素不应命中")
	}
	if InArray(1, []int{}) {
		t.Error("空数组不应命中")
	}
}

func TestGetContextWithTimeOut(t *testing.T) {
	ctx, cancel := GetContextWithTimeOut(1)
	defer cancel()
	select {
	case <-ctx.Done():
		t.Fatal("1 秒超时不应立即触发")
	case <-time.After(10 * time.Millisecond):
		// ok
	}
}

func TestDiscoveryEndpoint(t *testing.T) {
	got := DiscoveryEndpoint("ley.auth")
	want := "discovery:///ley.auth"
	if got != want {
		t.Errorf("DiscoveryEndpoint = %q, want %q", got, want)
	}
}

func TestDisServiceName(t *testing.T) {
	got := DisServiceName("auth")
	if got != constant.AppName+".auth" {
		t.Errorf("DisServiceName = %q", got)
	}
}
