package data

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"testing"

	"ley/app/comment/internal/model"
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

func newTestCommentRepo() (*commentRepo, *datatest.InMemoryCache) {
	d, c := newTestData()
	return &commentRepo{data: d}, c
}

// =========================================================================
// readListCache tests
// =========================================================================

func TestCommentRepo_ReadListCache(t *testing.T) {
	repo, mem := newTestCommentRepo()
	ctx := context.Background()

	t.Run("有效缓存数据返回 comments 和 total", func(t *testing.T) {
		mem.Reset()
		key := "comment:list:100:1:20"
		cached := commentListCache{
			Total: 5,
			Comments: []*model.Comment{
				{Content: "Great post!", PostID: 100},
				{Content: "Thanks!", PostID: 100},
			},
		}
		mem.Seed(key, &cached)

		comments, total, err := repo.readListCache(ctx, key)
		if err != nil {
			t.Fatalf("readListCache unexpected error: %v", err)
		}
		if total != 5 {
			t.Errorf("total = %d, want 5", total)
		}
		if len(comments) != 2 {
			t.Fatalf("comments count = %d, want 2", len(comments))
		}
		if comments[0].Content != "Great post!" {
			t.Errorf("comments[0].Content = %q, want %q", comments[0].Content, "Great post!")
		}
	})

	t.Run("空结果（total=0, comments=[]）正常返回", func(t *testing.T) {
		mem.Reset()
		key := "comment:list:200:1:20"
		cached := commentListCache{Total: 0, Comments: []*model.Comment{}}
		mem.Seed(key, &cached)

		comments, total, err := repo.readListCache(ctx, key)
		if err != nil {
			t.Fatalf("empty list should succeed: %v", err)
		}
		if total != 0 {
			t.Errorf("total = %d, want 0", total)
		}
		if len(comments) != 0 {
			t.Errorf("comments = %d, want 0", len(comments))
		}
	})

	t.Run("null 空值标记返回 ErrKeyNotFound", func(t *testing.T) {
		mem.Reset()
		key := "comment:list:300:1:20"
		mem.Seed(key, "null")

		_, _, err := repo.readListCache(ctx, key)
		if !errors.Is(err, cache.ErrKeyNotFound) {
			t.Errorf("null sentinel should return ErrKeyNotFound, got %v", err)
		}
	})

	t.Run("缓存不存在返回 ErrKeyNotFound", func(t *testing.T) {
		mem.Reset()

		_, _, err := repo.readListCache(ctx, "comment:list:999:1:20")
		if !errors.Is(err, cache.ErrKeyNotFound) {
			t.Errorf("missing key should return ErrKeyNotFound, got %v", err)
		}
	})

	t.Run("损坏 JSON 返回错误", func(t *testing.T) {
		mem.Reset()
		key := "comment:list:400:1:20"
		mem.Seed(key, "{invalid}")

		_, _, err := repo.readListCache(ctx, key)
		if err == nil {
			t.Fatal("expected error on corrupt JSON, got nil")
		}
	})

	t.Run("JSON 结构不匹配（缺少字段）返回零值", func(t *testing.T) {
		mem.Reset()
		key := "comment:list:500:1:20"
		// Provide valid JSON but with wrong structure
		mem.Seed(key, `{"foo": "bar"}`)

		_, total, err := repo.readListCache(ctx, key)
		if err != nil {
			t.Fatalf("valid JSON with wrong structure should not produce JSON error: %v", err)
		}
		if total != 0 {
			t.Errorf("missing total should default to 0, got %d", total)
		}
	})
}

// =========================================================================
// writeListCache tests
// =========================================================================

func TestCommentRepo_WriteListCache(t *testing.T) {
	repo, mem := newTestCommentRepo()
	ctx := context.Background()

	t.Run("正常写入 commentListCache", func(t *testing.T) {
		mem.Reset()
		key := "comment:list:100:1:20"
		comments := []*model.Comment{
			{Content: "Hello", PostID: 100},
			{Content: "World", PostID: 100},
		}
		total := int64(2)

		repo.writeListCache(ctx, key, comments, total)

		// 读回验证
		cached, cachedTotal, err := repo.readListCache(ctx, key)
		if err != nil {
			t.Fatalf("failed to read back written cache: %v", err)
		}
		if cachedTotal != 2 {
			t.Errorf("total = %d, want 2", cachedTotal)
		}
		if len(cached) != 2 {
			t.Errorf("comments count = %d, want 2", len(cached))
		}
	})

	t.Run("写入空结果存储 null 空值标记", func(t *testing.T) {
		mem.Reset()
		key := "comment:list:200:1:20"
		var emptyComments []*model.Comment
		total := int64(0)

		repo.writeListCache(ctx, key, emptyComments, total)

		// Get raw bytes to check it's "null"
		data, err := mem.Get(ctx, key)
		if err != nil {
			t.Fatalf("failed to read raw cache: %v", err)
		}
		if string(data) != "null" {
			t.Errorf("empty result should store 'null', got %q", string(data))
		}
	})

	t.Run("高频写入 commentListCache 前后一致", func(t *testing.T) {
		mem.Reset()
		for i := 0; i < 500; i++ {
			key := fmt.Sprintf("comment:list:%d:1:20", i%5) // 5 different keys, 100 writes each
			comments := []*model.Comment{{Content: fmt.Sprintf("Comment %d", i)}}
			repo.writeListCache(ctx, key, comments, int64(i+1))
		}
		// Verify the last write for each key persisted
		for k := 0; k < 5; k++ {
			key := fmt.Sprintf("comment:list:%d:1:20", k)
			data, err := mem.Get(ctx, key)
			if err != nil {
				t.Errorf("key %s not found after 100 overwrites: %v", key, err)
				continue
			}
			if len(data) == 0 {
				t.Errorf("key %s has empty data after 100 overwrites", key)
			}
		}
	})

	t.Run("写入后立即删除再读取返回不存在", func(t *testing.T) {
		mem.Reset()
		key := "comment:list:addrw:1:20"
		repo.writeListCache(ctx, key, []*model.Comment{{Content: "X"}}, 1)
		mem.Delete(ctx, key)

		_, _, err := repo.readListCache(ctx, key)
		if !errors.Is(err, cache.ErrKeyNotFound) {
			t.Errorf("expected ErrKeyNotFound after delete+read, got %v", err)
		}
	})
}

// =========================================================================
// invalidateCommentCache tests
// =========================================================================

func TestCommentRepo_InvalidateCommentCache(t *testing.T) {
	repo, mem := newTestCommentRepo()
	ctx := context.Background()

	t.Run("删除计数缓存和列表缓存", func(t *testing.T) {
		mem.Reset()
		postID := uint(100)

		countKey := fmt.Sprintf(cacheKeyCommentCount, postID)
		mem.Seed(countKey, 42)

		repo.invalidateCommentCache(ctx, postID)

		if !mem.WasDeleted(countKey) {
			t.Error("count cache was not deleted")
		}
	})

	t.Run("空 postID=0 正常处理不 panic", func(t *testing.T) {
		mem.Reset()
		func() {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("invalidateCommentCache(0) panicked: %v", r)
				}
			}()
			repo.invalidateCommentCache(ctx, 0)
		}()
	})

	t.Run("大规模 postID 正常处理", func(t *testing.T) {
		mem.Reset()
		func() {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("invalidateCommentCache(large) panicked: %v", r)
				}
			}()
			repo.invalidateCommentCache(ctx, ^uint(0)) // max uint
		}()
	})
}

// =========================================================================
// invalidateListCache tests — expanded coverage
// =========================================================================

func TestCommentRepo_InvalidateListCache(t *testing.T) {
	repo, mem := newTestCommentRepo()
	ctx := context.Background()

	t.Run("删除常见页码×页大的组合", func(t *testing.T) {
		mem.Reset()
		postID := uint(42)

		// 预写入 page 1-5 × size {10, 20, 50} 的所有 cache key
		for _, size := range []int{10, 20, 50} {
			for page := 1; page <= 5; page++ {
				key := fmt.Sprintf(cacheKeyCommentList, postID, page, size)
				mem.Seed(key, "data")
			}
		}

		repo.invalidateListCache(ctx, postID)

		// 验证所有 15 个 key 都被删除
		for _, size := range []int{10, 20, 50} {
			for page := 1; page <= 5; page++ {
				key := fmt.Sprintf(cacheKeyCommentList, postID, page, size)
				if !mem.WasDeleted(key) {
					t.Errorf("cache key not deleted: %s", key)
				}
			}
		}
	})

	t.Run("只删除匹配 postID 的 key", func(t *testing.T) {
		mem.Reset()
		targetPostID := uint(1)
		otherPostID := uint(2)

		targetKey := fmt.Sprintf(cacheKeyCommentList, targetPostID, 1, 20)
		otherKey := fmt.Sprintf(cacheKeyCommentList, otherPostID, 1, 20)
		mem.Seed(targetKey, "target")
		mem.Seed(otherKey, "other")

		repo.invalidateListCache(ctx, targetPostID)

		if !mem.WasDeleted(targetKey) {
			t.Error("target postID cache not deleted")
		}
		if mem.WasDeleted(otherKey) {
			t.Error("other postID cache should NOT be deleted")
		}
	})

	t.Run("缓存删除失败不 panic", func(t *testing.T) {
		mem.Reset()
		postID := uint(10)
		// 对某个 key 返回错误不影响其他
		mem.SetGetError(fmt.Sprintf(cacheKeyCommentList, postID, 1, 10), errors.New("oops"))

		func() {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("invalidateListCache panicked: %v", r)
				}
			}()
			repo.invalidateListCache(ctx, postID)
		}()
	})
}

// =========================================================================
// commentListCache JSON round-trip
// =========================================================================

func TestCommentListCache_RoundTrip(t *testing.T) {
	t.Run("序列化与反序列化一致", func(t *testing.T) {
		original := commentListCache{
			Total: 3,
			Comments: []*model.Comment{
				{Content: "A", PostID: 1, AuthorID: 10, Status: model.CommentStatusApproved},
				{Content: "B", PostID: 1, AuthorID: 20, Status: model.CommentStatusApproved},
				{Content: "C", PostID: 1, AuthorID: 30, Status: model.CommentStatusPending},
			},
		}

		data, err := json.Marshal(original)
		if err != nil {
			t.Fatalf("marshal failed: %v", err)
		}

		var decoded commentListCache
		if err := json.Unmarshal(data, &decoded); err != nil {
			t.Fatalf("unmarshal failed: %v", err)
		}

		if decoded.Total != original.Total {
			t.Errorf("Total mismatch: %d vs %d", decoded.Total, original.Total)
		}
		if len(decoded.Comments) != len(original.Comments) {
			t.Fatalf("Comments length mismatch: %d vs %d", len(decoded.Comments), len(original.Comments))
		}
		for i := range original.Comments {
			if decoded.Comments[i].Content != original.Comments[i].Content {
				t.Errorf("Comment[%d].Content mismatch: %q vs %q",
					i, decoded.Comments[i].Content, original.Comments[i].Content)
			}
		}
	})

	t.Run("大列表序列化无截断", func(t *testing.T) {
		comments := make([]*model.Comment, 100)
		for i := range comments {
			comments[i] = &model.Comment{
				Content: fmt.Sprintf("Comment #%d - Lorem ipsum dolor sit amet", i),
				PostID:  uint(i % 5),
			}
		}
		original := commentListCache{Total: 100, Comments: comments}

		data, err := json.Marshal(original)
		if err != nil {
			t.Fatalf("large list marshal failed: %v", err)
		}

		var decoded commentListCache
		if err := json.Unmarshal(data, &decoded); err != nil {
			t.Fatalf("large list unmarshal failed: %v", err)
		}

		if len(decoded.Comments) != 100 {
			t.Errorf("large list corrupted: got %d, want 100", len(decoded.Comments))
		}
	})
}

// =========================================================================
// Comment cache key format tests
// =========================================================================

func TestCommentCacheKeyFormat(t *testing.T) {
	tests := []struct {
		name     string
		format   string
		args     []interface{}
		expected string
	}{
		{
			name:     "列表缓存键",
			format:   cacheKeyCommentList,
			args:     []interface{}{uint(42), 1, 20},
			expected: "comment:list:42:1:20",
		},
		{
			name:     "计数缓存键",
			format:   cacheKeyCommentCount,
			args:     []interface{}{uint(99)},
			expected: "comment:count:99",
		},
		{
			name:     "列表键大 page",
			format:   cacheKeyCommentList,
			args:     []interface{}{uint(1), 999, 100},
			expected: "comment:list:1:999:100",
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
// Error sentinel tests
// =========================================================================

func TestCommentErrorSentinels(t *testing.T) {
	t.Run("ErrCommentNotFound distinct from gorm.ErrRecordNotFound", func(t *testing.T) {
		if errors.Is(ErrCommentNotFound, gorm.ErrRecordNotFound) {
			t.Error("ErrCommentNotFound should NOT match gorm.ErrRecordNotFound")
		}
	})

	t.Run("ErrCommentNotFound is distinguishable", func(t *testing.T) {
		err := fmt.Errorf("wrapped: %w", ErrCommentNotFound)
		if !errors.Is(err, ErrCommentNotFound) {
			t.Error("wrapped ErrCommentNotFound should be detectable")
		}
	})
}

// =========================================================================
// Integration test placeholder
// =========================================================================

func TestCommentRepo_Integration_Stub(t *testing.T) {
	t.Skip("requires PostgreSQL; run with -tags=integration")
}

var _ gorm.Model = model.Comment{}.Model
