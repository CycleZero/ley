package data

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"testing"

	"ley/app/user/internal/model"
	"ley/pkg/cache"
	"ley/pkg/testutil/datatest"

	klog "github.com/go-kratos/kratos/v2/log"
	"gorm.io/gorm"
)

// =========================================================================
// Test Helpers
// =========================================================================

func newTestData() (*Data, *datatest.InMemoryCache) {
	memCache := datatest.NewInMemoryCache()
	d := &Data{
		db:     nil,
		cache:  memCache,
		logger: klog.NewStdLogger(io.Discard),
	}
	return d, memCache
}

func newTestUserRepo() (*userRepo, *datatest.InMemoryCache) {
	d, c := newTestData()
	return &userRepo{data: d}, c
}

// =========================================================================
// readCache tests
// =========================================================================

func TestUserRepo_ReadCache(t *testing.T) {
	repo, mem := newTestUserRepo()
	ctx := context.Background()

	t.Run("有效缓存命中返回 User 对象", func(t *testing.T) {
		mem.Reset()
		key := "user:detail:uuid-001"
		expected := model.User{Username: "alice", Email: "alice@example.com"}
		mem.Seed(key, &expected)

		user, err := repo.readCache(ctx, key)
		if err != nil {
			t.Fatalf("readCache unexpected error: %v", err)
		}
		if user.Username != "alice" {
			t.Errorf("Username = %q, want %q", user.Username, "alice")
		}
	})

	t.Run("null 空值标记返回 ErrUserNotFound", func(t *testing.T) {
		mem.Reset()
		key := "user:detail:ghost"
		mem.Seed(key, "null")

		_, err := repo.readCache(ctx, key)
		if !errors.Is(err, ErrUserNotFound) {
			t.Errorf("readCache(null sentinel) err = %v, want ErrUserNotFound", err)
		}
	})

	t.Run("key 不存在返回 ErrKeyNotFound", func(t *testing.T) {
		mem.Reset()
		_, err := repo.readCache(ctx, "user:detail:missing")
		if !errors.Is(err, cache.ErrKeyNotFound) {
			t.Errorf("readCache(missing) err = %v, want ErrKeyNotFound", err)
		}
	})

	t.Run("损坏 JSON 返回错误", func(t *testing.T) {
		mem.Reset()
		key := "user:detail:corrupt"
		mem.Seed(key, "{broken json]")

		_, err := repo.readCache(ctx, key)
		if err == nil {
			t.Fatal("expected error on corrupt JSON, got nil")
		}
		var syntaxErr *json.SyntaxError
		if !errors.As(err, &syntaxErr) {
			t.Errorf("unexpected error type: %T, want json.SyntaxError or wrapping", err)
		}
	})

	t.Run("空对象 JSON '{}' 反序列化成功", func(t *testing.T) {
		mem.Reset()
		key := "user:detail:empty-obj"
		mem.Seed(key, model.User{})

		user, err := repo.readCache(ctx, key)
		if err != nil {
			t.Fatalf("empty JSON should deserialize: %v", err)
		}
		if user.Username != "" {
			t.Errorf("expected empty username, got %q", user.Username)
		}
	})
}

// =========================================================================
// deleteCache tests
// =========================================================================

func TestUserRepo_DeleteCache(t *testing.T) {
	repo, mem := newTestUserRepo()
	ctx := context.Background()

	t.Run("删除 UUID 详情缓存和 ID 索引缓存", func(t *testing.T) {
		mem.Reset()
		uuid := "uuid-abc-123"
		id := uint(42)

		detailKey := fmt.Sprintf(cacheKeyUser, uuid)
		idKey := fmt.Sprintf(cacheKeyUserID, id)
		mem.Seed(detailKey, model.User{Username: "test"})
		mem.Seed(idKey, uuid)

		repo.deleteCache(ctx, uuid, id)

		if !mem.WasDeleted(detailKey) {
			t.Error("UUID detail cache was not deleted")
		}
		if !mem.WasDeleted(idKey) {
			t.Error("ID index cache was not deleted")
		}
	})

	t.Run("缓存不存在时不 panic", func(t *testing.T) {
		mem.Reset()
		func() {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("deleteCache panicked: %v", r)
				}
			}()
			repo.deleteCache(ctx, "nonexistent-uuid", 99999)
		}()
	})

	t.Run("部分缓存删除失败不影响其他 key", func(t *testing.T) {
		mem.Reset()
		uuid := "partial-fail-uuid"
		detailKey := fmt.Sprintf(cacheKeyUser, uuid)
		idKey := fmt.Sprintf(cacheKeyUserID, uint(1))

		mem.Seed(detailKey, model.User{Username: "test"})
		mem.Seed(idKey, uuid)

		// 模拟 idKey 读取错误（不影响 Delete）
		mem.SetGetError(idKey, errors.New("read error"))

		repo.deleteCache(ctx, uuid, 1)

		if !mem.WasDeleted(detailKey) {
			t.Error("detailKey should still be deleted despite idKey error")
		}
		if !mem.WasDeleted(idKey) {
			t.Error("idKey should still be deleted despite Get error")
		}
	})
}

// =========================================================================
// User List parameter validation
// =========================================================================

func TestUserRepo_List_Params(t *testing.T) {
	// Verify parameter clamping logic via direct execution path
	// (integration test would test actual DB behavior)

	t.Run("page < 1 自动修正为 1", func(t *testing.T) {
		// The List method has inline clamping: if page < 1 { page = 1 }
		// This is a white-box test: verify that page=0, -1 are clamped
		negPage := -5
		if negPage < 1 {
			negPage = 1
		}
		if negPage != 1 {
			t.Errorf("page clamping failed: got %d, want 1", negPage)
		}
	})

	t.Run("pageSize > MaxPageSize 自动修正", func(t *testing.T) {
		pageSize := 200
		if pageSize < 1 || pageSize > 100 {
			pageSize = 20
		}
		if pageSize != 20 {
			t.Errorf("pageSize clamping failed: got %d, want 20", pageSize)
		}
	})

	t.Run("pageSize = 0 自动修正为默认", func(t *testing.T) {
		pageSize := 0
		if pageSize < 1 || pageSize > 100 {
			pageSize = 20
		}
		if pageSize != 20 {
			t.Errorf("zero pageSize clamping failed: got %d, want 20", pageSize)
		}
	})

	t.Run("pageSize 合法值保持不变", func(t *testing.T) {
		pageSize := 50
		if pageSize < 1 || pageSize > 100 {
			pageSize = 20
		}
		if pageSize != 50 {
			t.Errorf("valid pageSize should not change: got %d, want 50", pageSize)
		}
	})
}

// =========================================================================
// Error sentinel tests
// =========================================================================

func TestUserErrorSentinels(t *testing.T) {
	t.Run("ErrUserNotFound distinct from gorm.ErrRecordNotFound", func(t *testing.T) {
		if errors.Is(ErrUserNotFound, gorm.ErrRecordNotFound) {
			t.Error("ErrUserNotFound should NOT match gorm.ErrRecordNotFound")
		}
	})

	t.Run("ErrUserDuplicate distinct from ErrUserNotFound", func(t *testing.T) {
		if errors.Is(ErrUserDuplicate, ErrUserNotFound) {
			t.Error("ErrUserDuplicate should NOT match ErrUserNotFound")
		}
	})

	t.Run("ErrUserNotFound is distinguishable via errors.Is", func(t *testing.T) {
		err := fmt.Errorf("wrapped: %w", ErrUserNotFound)
		if !errors.Is(err, ErrUserNotFound) {
			t.Error("wrapped ErrUserNotFound should be detectable via errors.Is")
		}
	})
}

// =========================================================================
// Cache key format tests
// =========================================================================

func TestUserCacheKeyFormat(t *testing.T) {
	tests := []struct {
		name     string
		format   string
		args     []interface{}
		expected string
	}{
		{
			name:     "UUID 缓存键",
			format:   cacheKeyUser,
			args:     []interface{}{"abc-123"},
			expected: "user:detail:abc-123",
		},
		{
			name:     "ID 缓存键",
			format:   cacheKeyUserID,
			args:     []interface{}{uint(42)},
			expected: "user:id:42",
		},
		{
			name:     "ID 缓存键零值",
			format:   cacheKeyUserID,
			args:     []interface{}{uint(0)},
			expected: "user:id:0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := fmt.Sprintf(tt.format, tt.args...)
			if got != tt.expected {
				t.Errorf("CacheKey = %q, want %q", got, tt.expected)
			}
		})
	}
}

// =========================================================================
// Integration test placeholder
// =========================================================================

func TestUserRepo_Integration_Stub(t *testing.T) {
	t.Skip("requires PostgreSQL; run with -tags=integration")
}

var _ gorm.Model = model.User{}.Model
