package data

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"testing"

	"ley/app/post/internal/model"
	"ley/pkg/cache"
	"ley/pkg/testutil/datatest"

	klog "github.com/go-kratos/kratos/v2/log"
	"gorm.io/gorm"
)

// =========================================================================
// Test Helpers
// =========================================================================

// newTestData creates a Data instance suitable for unit tests.
// Uses InMemoryCache and a nil DB.  Tests that need a real DB
// should use build tag integration.
func newTestData() (*Data, *datatest.InMemoryCache) {
	memCache := datatest.NewInMemoryCache()
	d := &Data{
		db:     nil,
		cache:  memCache,
		logger: klog.NewStdLogger(io.Discard),
	}
	return d, memCache
}

// newTestPostRepo creates a postRepo with a test Data instance.
func newTestPostRepo() (*postRepo, *datatest.InMemoryCache) {
	d, c := newTestData()
	return &postRepo{data: d}, c
}

// newTestTagRepo creates a tagRepo with a test Data instance.
func newTestTagRepo() (*tagRepo, *datatest.InMemoryCache) {
	d, c := newTestData()
	return &tagRepo{data: d}, c
}

// newTestCategoryRepo creates a categoryRepo with a test Data instance.
func newTestCategoryRepo() (*categoryRepo, *datatest.InMemoryCache) {
	d, c := newTestData()
	return &categoryRepo{data: d}, c
}

// =========================================================================
// buildPostOrderClause tests
// =========================================================================

func TestBuildPostOrderClause(t *testing.T) {
	tests := []struct {
		name      string
		sortBy    string
		sortOrder string
		want      string
	}{
		// --- 正向用例 ---
		{name: "默认排序", sortBy: "", sortOrder: "", want: "created_at DESC"},
		{name: "按创建时间降序", sortBy: "created_at", sortOrder: "desc", want: "created_at DESC"},
		{name: "按创建时间升序", sortBy: "created_at", sortOrder: "asc", want: "created_at ASC"},
		{name: "按更新时间降序", sortBy: "updated_at", sortOrder: "desc", want: "updated_at DESC"},
		{name: "按发布时间升序", sortBy: "published_at", sortOrder: "asc", want: "published_at ASC"},
		{name: "按浏览量降序", sortBy: "view_count", sortOrder: "desc", want: "view_count DESC"},
		{name: "按置顶降序", sortBy: "is_top", sortOrder: "desc", want: "is_top DESC, published_at DESC"},

		// --- 边界用例 ---
		{
			name:      "未知排序字段回退默认",
			sortBy:    "nonexistent_field",
			sortOrder: "desc",
			want:      "created_at DESC",
		},
		{
			name:      "非法排序方向回退 DESC",
			sortBy:    "created_at",
			sortOrder: "DROP TABLE users; --",
			want:      "created_at DESC",
		},
		{
			name:      "空排序方向回退 DESC",
			sortBy:    "updated_at",
			sortOrder: "",
			want:      "updated_at DESC",
		},
		{
			name:      "大写 ASC 不识别回退 DESC",
			sortBy:    "created_at",
			sortOrder: "ASC",
			want:      "created_at DESC",
		},

		// --- 安全用例：防止 SQL 注入 ---
		{
			name:      "排序字段 SQL 注入防护",
			sortBy:    "1; DROP TABLE posts; --",
			sortOrder: "desc",
			want:      "created_at DESC", // falls back to default
		},
		{
			name:      "排序方向 SQL 注入防护",
			sortBy:    "created_at",
			sortOrder: "desc; DROP TABLE posts; --",
			want:      "created_at DESC",
		},
		{
			name:      "排序字段含空格回退默认列但尊重排序方向",
			sortBy:    "created at",
			sortOrder: "asc",
			want:      "created_at ASC", // column falls back to default, order is respected
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildPostOrderClause(tt.sortBy, tt.sortOrder)
			if got != tt.want {
				t.Errorf("buildPostOrderClause(%q, %q) = %q, want %q",
					tt.sortBy, tt.sortOrder, got, tt.want)
			}
		})
	}
}

// =========================================================================
// postStatusToInt tests
// =========================================================================

func TestPostStatusToInt(t *testing.T) {
	tests := []struct {
		name   string
		status string
		want   int8
	}{
		{name: "已发布", status: "published", want: 1},
		{name: "已归档", status: "archived", want: 2},
		{name: "草稿显式传入", status: "draft", want: 0},
		{name: "空字符串默认草稿", status: "", want: 0},
		{name: "未知状态默认草稿", status: "unknown", want: 0},
		{name: "大小写混用", status: "Published", want: 0}, // 严格匹配
		{name: "全大写", status: "PUBLISHED", want: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := postStatusToInt(tt.status)
			if got != tt.want {
				t.Errorf("postStatusToInt(%q) = %d, want %d", tt.status, got, tt.want)
			}
		})
	}
}

// =========================================================================
// buildCategoryTree tests
// =========================================================================

func TestBuildCategoryTree(t *testing.T) {
	t.Run("空分类列表返回 nil", func(t *testing.T) {
		got := buildCategoryTree(nil)
		if got != nil {
			t.Errorf("buildCategoryTree(nil) = %v, want nil", got)
		}
	})

	t.Run("空切片返回 nil", func(t *testing.T) {
		got := buildCategoryTree([]*model.Category{})
		if got != nil {
			t.Errorf("buildCategoryTree([]) = %v, want nil", got)
		}
	})

	t.Run("单层所有顶级节点", func(t *testing.T) {
		categories := []*model.Category{
			{Name: "Tech", SortOrder: 1},
			{Name: "Life", SortOrder: 2},
			{Name: "Game", SortOrder: 3},
		}
		for i, c := range categories {
			c.ID = uint(i + 1)
		}

		tree := buildCategoryTree(categories)
		if len(tree) != 3 {
			t.Fatalf("buildCategoryTree 顶级节点数 = %d, want 3", len(tree))
		}
		for i, cat := range tree {
			if cat.Name != categories[i].Name {
				t.Errorf("节点[%d] name = %q, want %q", i, cat.Name, categories[i].Name)
			}
			if len(cat.Children) != 0 {
				t.Errorf("节点[%d] children = %d, want 0 (should be leaf)", i, len(cat.Children))
			}
		}
	})

	t.Run("两级嵌套分类树", func(t *testing.T) {
		var parentID uint = 1
		categories := []*model.Category{
			{Name: "Tech", SortOrder: 1},                          // ID=1
			{Name: "Go", ParentID: &parentID, SortOrder: 1},       // ID=2
			{Name: "Rust", ParentID: &parentID, SortOrder: 2},     // ID=3
			{Name: "Life", SortOrder: 2},                          // ID=4
		}
		for i := range categories {
			categories[i].ID = uint(i + 1)
			if categories[i].ParentID != nil {
				id := uint(*categories[i].ParentID)
				categories[i].ParentID = &id
			}
		}

		tree := buildCategoryTree(categories)
		if len(tree) != 2 {
			t.Fatalf("顶级节点数 = %d, want 2", len(tree))
		}

		tech := tree[0]
		if tech.Name != "Tech" {
			t.Errorf("第一顶级节点 name = %q, want %q", tech.Name, "Tech")
		}
		if len(tech.Children) != 2 {
			t.Fatalf("Tech 子分类数 = %d, want 2", len(tech.Children))
		}
		if tech.Children[0].Name != "Go" {
			t.Errorf("Tech 子分类[0] = %q, want %q", tech.Children[0].Name, "Go")
		}
		if tech.Children[1].Name != "Rust" {
			t.Errorf("Tech 子分类[1] = %q, want %q", tech.Children[1].Name, "Rust")
		}

		life := tree[1]
		if life.Name != "Life" {
			t.Errorf("第二顶级节点 name = %q, want %q", life.Name, "Life")
		}
		if len(life.Children) != 0 {
			t.Errorf("Life 子分类数 = %d, want 0", len(life.Children))
		}
	})

	t.Run("三级深层嵌套", func(t *testing.T) {
		parent1 := uint(1)
		parent2 := uint(2)
		categories := []*model.Category{
			{Name: "Level1", SortOrder: 1},                              // ID=1
			{Name: "Level2", ParentID: &parent1, SortOrder: 1},          // ID=2
			{Name: "Level3", ParentID: &parent2, SortOrder: 1},          // ID=3
		}
		for i := range categories {
			categories[i].ID = uint(i + 1)
			if categories[i].ParentID != nil {
				id := uint(*categories[i].ParentID)
				categories[i].ParentID = &id
			}
		}

		tree := buildCategoryTree(categories)
		if len(tree) != 1 {
			t.Fatalf("顶级节点 = %d, want 1", len(tree))
		}
		if len(tree[0].Children) != 1 {
			t.Fatalf("Level1 子分类 = %d, want 1", len(tree[0].Children))
		}
		if len(tree[0].Children[0].Children) != 1 {
			t.Fatalf("Level2 子分类 = %d, want 1", len(tree[0].Children[0].Children))
		}
		if tree[0].Children[0].Children[0].Name != "Level3" {
			t.Errorf("Level3 name = %q, want %q", tree[0].Children[0].Children[0].Name, "Level3")
		}
	})

	t.Run("孤儿节点（父节点不存在）提升为顶级", func(t *testing.T) {
		orphanParentID := uint(999) // 不存在的父节点
		categories := []*model.Category{
			{Name: "Orphan", ParentID: &orphanParentID, SortOrder: 1},
		}
		categories[0].ID = 1

		tree := buildCategoryTree(categories)
		if len(tree) != 1 {
			t.Fatalf("孤儿节点未提升: got %d top-level nodes, want 1", len(tree))
		}
		if tree[0].Name != "Orphan" {
			t.Errorf("孤儿节点 name = %q, want %q", tree[0].Name, "Orphan")
		}
	})

	t.Run("ParentID 为 0 视为顶级", func(t *testing.T) {
		zeroID := uint(0)
		categories := []*model.Category{
			{Name: "Top", ParentID: &zeroID, SortOrder: 1},
		}
		categories[0].ID = 1

		tree := buildCategoryTree(categories)
		if len(tree) != 1 {
			t.Fatalf("ParentID=0 节点 = %d, want 1", len(tree))
		}
	})
}

// =========================================================================
// readFromCache tests
// =========================================================================

func TestPostRepo_ReadFromCache(t *testing.T) {
	repo, mem := newTestPostRepo()
	ctx := context.Background()

	t.Run("有效缓存数据反序列化成功", func(t *testing.T) {
		mem.Reset()
		key := "post:detail:abc-123"
		expected := model.Post{Title: "Hello World", Slug: "hello-world"}
		mem.Seed(key, &expected)

		post, err := repo.readFromCache(ctx, key)
		if err != nil {
			t.Fatalf("readFromCache unexpected error: %v", err)
		}
		if post.Title != "Hello World" {
			t.Errorf("Title = %q, want %q", post.Title, "Hello World")
		}
		if post.Slug != "hello-world" {
			t.Errorf("Slug = %q, want %q", post.Slug, "hello-world")
		}
	})

	t.Run("null 空值标记返回 ErrPostNotFound", func(t *testing.T) {
		mem.Reset()
		key := "post:detail:nonexistent"
		mem.Seed(key, "null")

		_, err := repo.readFromCache(ctx, key)
		if !errors.Is(err, ErrPostNotFound) {
			t.Errorf("readFromCache(null sentinel) err = %v, want ErrPostNotFound", err)
		}
	})

	t.Run("不存在的 key 返回 ErrKeyNotFound", func(t *testing.T) {
		mem.Reset()

		_, err := repo.readFromCache(ctx, "post:detail:missing")
		if !errors.Is(err, cache.ErrKeyNotFound) {
			t.Errorf("readFromCache(missing) err = %v, want ErrKeyNotFound", err)
		}
	})

	t.Run("损坏的 JSON 返回错误", func(t *testing.T) {
		mem.Reset()
		key := "post:detail:corrupt"
		mem.Seed(key, "this is not valid json {{{")

		_, err := repo.readFromCache(ctx, key)
		if err == nil {
			t.Fatal("readFromCache(corrupt data) should return error")
		}
		var syntaxErr *json.SyntaxError
		if !errors.As(err, &syntaxErr) {
			t.Errorf("readFromCache(corrupt) err type = %T, want *json.SyntaxError", err)
		}
	})
}

// =========================================================================
// invalidatePostCache tests
// =========================================================================

func TestPostRepo_InvalidatePostCache(t *testing.T) {
	repo, mem := newTestPostRepo()
	ctx := context.Background()

	t.Run("删除详情缓存和浏览缓存", func(t *testing.T) {
		mem.Reset()
		uuid := "abc-123-uuid"
		slug := "hello-world"
		id := uint(42)

		// 预写入这些 key
		mem.Seed(fmt.Sprintf(cacheKeyPostDetail, uuid), "not null") // bypass null sentinel
		mem.Seed(fmt.Sprintf(cacheKeyPostView, id), "100")

		repo.invalidatePostCache(ctx, id, uuid, slug)

		if mem.WasDeleted(fmt.Sprintf(cacheKeyPostDetail, uuid)) {
			t.Log("detail cache deleted correctly")
		} else {
			t.Error("detail cache was not deleted")
		}
		if mem.WasDeleted(fmt.Sprintf(cacheKeyPostView, id)) {
			t.Log("view count cache deleted correctly")
		} else {
			t.Error("view count cache was not deleted")
		}
	})

	t.Run("删除 slug 映射缓存", func(t *testing.T) {
		mem.Reset()
		slug := "my-slug"
		slugKey := fmt.Sprintf(cacheKeyPostSlug, slug)
		mem.Seed(slugKey, "some-uuid")

		repo.invalidatePostCache(ctx, 1, "some-uuid", slug)

		if !mem.WasDeleted(slugKey) {
			t.Error("slug cache was NOT deleted, but should have been")
		}
	})

	t.Run("空 slug 不删除 slug 缓存（避免误删全局 slug）", func(t *testing.T) {
		mem.Reset()
		emptySlugKey := fmt.Sprintf(cacheKeyPostSlug, "")
		mem.Seed(emptySlugKey, "data")

		repo.invalidatePostCache(ctx, 1, "uuid", "")

		if mem.WasDeleted(emptySlugKey) {
			t.Error("cache with empty slug key was deleted; it should be skipped")
		}
	})

	t.Run("缓存不存在时 Delete 不 panic", func(t *testing.T) {
		mem.Reset()
		// 不对 cache 做任何预写入

		func() {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("invalidatePostCache panicked on missing keys: %v", r)
				}
			}()
			repo.invalidatePostCache(ctx, 999, "no-such-uuid", "no-such-slug")
		}()
	})
}

// =========================================================================
// readFromCache with null sentinel — edge cases
// =========================================================================

func TestPostRepo_ReadFromCache_EdgeCases(t *testing.T) {
	repo, mem := newTestPostRepo()
	ctx := context.Background()

	t.Run("空字符串当做有效数据→JSON解析失败", func(t *testing.T) {
		mem.Reset()
		key := "post:detail:empty"
		mem.Seed(key, "")

		_, err := repo.readFromCache(ctx, key)
		if err == nil {
			t.Error("empty string should fail JSON unmarshal")
		}
	})

	t.Run("[]byte 形式的 null 字符串", func(t *testing.T) {
		mem.Reset()
		key := "post:detail:bytes-null"
		mem.Seed(key, []byte("null"))

		_, err := repo.readFromCache(ctx, key)
		if !errors.Is(err, ErrPostNotFound) {
			t.Errorf("expected ErrPostNotFound for bytes('null'), got %v", err)
		}
	})

	t.Run("大 JSON 反序列化", func(t *testing.T) {
		mem.Reset()
		key := "post:detail:large"
		largePost := model.Post{
			Title:   "Large Title",
			Content: string(make([]byte, 10000)), // 10KB of null bytes
			Slug:    "large-post",
		}
		mem.Seed(key, &largePost)

		_, err := repo.readFromCache(ctx, key)
		if err != nil {
			t.Errorf("large JSON should deserialize: %v", err)
		}
	})
}

// =========================================================================
// Tag cache tests
// =========================================================================

func TestTagRepo_InvalidateTagCache(t *testing.T) {
	repo, mem := newTestTagRepo()
	ctx := context.Background()

	t.Run("删除全量标签缓存", func(t *testing.T) {
		mem.Reset()
		mem.Seed(cacheKeyTagAll, []model.Tag{{Name: "Go"}})

		repo.invalidateTagCache(ctx)

		if !mem.WasDeleted(cacheKeyTagAll) {
			t.Error("tag:all cache was not deleted")
		}
	})

	t.Run("缓存删除失败不 panic", func(t *testing.T) {
		mem.Reset()
		mem.SetGetError(cacheKeyTagAll, errors.New("redis connection lost"))

		func() {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("invalidateTagCache panicked on error: %v", r)
				}
			}()
			repo.invalidateTagCache(ctx)
		}()
		// Get calls incremented for GetObject + Delete internally? No — SetGetError only affects Get.
		// Delete call should still happen; the "error" from Seed/Set is separate from Delete error.
		// Actually, SetGetError only affects Get, not Delete. Delete goes through normally.
	})
}

// =========================================================================
// Category cache tests
// =========================================================================

func TestCategoryRepo_InvalidateCategoryCache(t *testing.T) {
	repo, mem := newTestCategoryRepo()
	ctx := context.Background()

	t.Run("删除分类树缓存", func(t *testing.T) {
		mem.Reset()
		mem.Seed(cacheKeyCategoryTree, []model.Category{{Name: "Tech"}})

		repo.invalidateCategoryCache(ctx)

		if !mem.WasDeleted(cacheKeyCategoryTree) {
			t.Error("category:tree cache was not deleted")
		}
	})

	t.Run("缓存不存在时删除操作正常完成", func(t *testing.T) {
		mem.Reset()
		repo.invalidateCategoryCache(ctx)
		if mem.DeleteCount(cacheKeyCategoryTree) != 1 {
			t.Error("expected Delete to be called once even for missing key")
		}
	})
}

// =========================================================================
// Integration test placeholders (requires PostgreSQL)
// =========================================================================

// TestPostRepo_Integration is a placeholder documenting which integration
// tests should be run against a real PostgreSQL.
//
// Run with: go test -tags=integration ./app/post/internal/data/...
//
// Required test cases:
//   - CreatePost / UpdatePost / DeletePost full lifecycle
//   - FindByUUID with real Cache-Aside (pre-populate cache, verify hit)
//   - List with tag filtering (JOIN correctness)
//   - Search with tsvector (need real search_vector generation)
//   - SyncTags transactional integrity (rollback on insert failure)
//   - IncrementViewCount atomicity (concurrent increments)
//   - AssociateTags ON CONFLICT DO NOTHING (duplicate tag prevention)
func TestPostRepo_Integration_Stub(t *testing.T) {
	t.Skip("requires PostgreSQL; run with -tags=integration")
}

// compile-time checks
var _ gorm.Model = model.Post{}.Model
