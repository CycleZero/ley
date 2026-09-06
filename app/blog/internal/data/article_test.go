package data

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/CycleZero/ley/app/blog/internal/biz"
)

func setupArticleRepo(t *testing.T) (biz.ArticleRepo, *Data) {
	t.Helper()
	d := newTestData(t)
	repo := NewArticleRepo(d)
	return repo, d
}

// uniSlug generates a unique slug for tests to avoid clashes with previous runs.
func uniSlug(t *testing.T, base string) string {
	t.Helper()
	return fmt.Sprintf("%s-%d", base, time.Now().UnixNano())
}

func ptrUint(v uint) *uint { return &v }

func createTestArticle(t *testing.T, repo biz.ArticleRepo, title, slug string, authorID uint) *biz.Article {
	t.Helper()
	a := &biz.Article{
		Title:    title,
		Slug:     slug,
		Content:  "test content for " + title,
		Excerpt:  "excerpt",
		AuthorID: authorID,
		Status:   biz.ArticleStatusDraft,
	}
	if err := repo.Create(context.Background(), a); err != nil {
		t.Fatalf("create test article: %v", err)
	}
	return a
}

func TestArticleRepo_Create(t *testing.T) {
	repo, _ := setupArticleRepo(t)

	t.Run("happy path", func(t *testing.T) {
		a := &biz.Article{Title: "Hello", Slug: uniSlug(t, "hello"), Content: "world", AuthorID: 1}
		err := repo.Create(context.Background(), a)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if a.ID == 0 {
			t.Error("expected ID to be populated")
		}
		if a.CreatedAt.IsZero() {
			t.Error("expected CreatedAt to be populated")
		}
	})

	t.Run("duplicate slug", func(t *testing.T) {
		slug := uniSlug(t, "dup")
		createTestArticle(t, repo, "First", slug, 1)
		a2 := &biz.Article{Title: "Second", Slug: slug, Content: "x", AuthorID: 1}
		err := repo.Create(context.Background(), a2)
		if err != biz.ErrSlugAlreadyExists {
			t.Errorf("expected ErrSlugAlreadyExists, got %v", err)
		}
	})
}

func TestArticleRepo_FindByID(t *testing.T) {
	repo, _ := setupArticleRepo(t)

	t.Run("found", func(t *testing.T) {
		created := createTestArticle(t, repo, "Test", uniSlug(t, "find"), 1)
		found, err := repo.FindByID(context.Background(), created.ID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if found.Title != "Test" {
			t.Errorf("expected title Test, got %s", found.Title)
		}
	})

	t.Run("not found", func(t *testing.T) {
		_, err := repo.FindByID(context.Background(), 99999)
		if err == nil {
			t.Error("expected error for non-existent ID")
		}
	})
}

func TestArticleRepo_FindBySlug(t *testing.T) {
	repo, _ := setupArticleRepo(t)

	t.Run("find existing", func(t *testing.T) {
		slug := uniSlug(t, "tgt")
		created := createTestArticle(t, repo, "Target", slug, 1)
		found, err := repo.FindBySlug(context.Background(), slug)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if found.ID != created.ID {
			t.Errorf("expected ID %d, got %d", created.ID, found.ID)
		}
	})

	t.Run("not found", func(t *testing.T) {
		_, err := repo.FindBySlug(context.Background(), uniSlug(t, "noexist"))
		if err == nil {
			t.Error("expected error for non-existent slug")
		}
	})
}

func TestArticleRepo_Update(t *testing.T) {
	repo, _ := setupArticleRepo(t)

	t.Run("happy path", func(t *testing.T) {
		a := createTestArticle(t, repo, "Old", uniSlug(t, "upd"), 1)
		a.Title = "NewTitle"
		a.Content = "NewContent"
		err := repo.Update(context.Background(), a)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		found, err := repo.FindByID(context.Background(), a.ID)
		if err != nil {
			t.Fatalf("find after update error: %v", err)
		}
		if found.Title != "NewTitle" {
			t.Errorf("expected NewTitle, got %s", found.Title)
		}
	})

	t.Run("not found", func(t *testing.T) {
		a := &biz.Article{ID: 99999, Title: "X", Content: "Y"}
		err := repo.Update(context.Background(), a)
		if err == nil {
			t.Error("expected error for non-existent ID")
		}
	})
}

func TestArticleRepo_Delete(t *testing.T) {
	repo, _ := setupArticleRepo(t)

	t.Run("happy path", func(t *testing.T) {
		a := createTestArticle(t, repo, "Del", uniSlug(t, "del"), 1)
		err := repo.Delete(context.Background(), a.ID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}

func TestArticleRepo_List(t *testing.T) {
	repo, _ := setupArticleRepo(t)

	t.Run("empty list", func(t *testing.T) {
		articles, total, err := repo.List(context.Background(), biz.ArticleListQuery{
			AuthorID: ptrUint(99999),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if total != 0 {
			t.Errorf("expected 0 total, got %d", total)
		}
		if len(articles) != 0 {
			t.Errorf("expected 0 articles, got %d", len(articles))
		}
	})

	t.Run("pagination", func(t *testing.T) {
		authorID := uint(77777 + time.Now().UnixNano()%10000)
		for i := 0; i < 5; i++ {
			createTestArticle(t, repo, "Article "+string(rune('A'+i)), uniSlug(t, fmt.Sprintf("sl%d", i)), authorID)
		}
		articles, total, err := repo.List(context.Background(), biz.ArticleListQuery{AuthorID: &authorID, Page: 1, PageSize: 3})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if total != 5 {
			t.Errorf("expected 5 total, got %d", total)
		}
		if len(articles) != 3 {
			t.Errorf("expected 3 articles, got %d", len(articles))
		}
	})
}

func TestArticleRepo_Like(t *testing.T) {
	repo, _ := setupArticleRepo(t)
	a := createTestArticle(t, repo, "LikeMe", uniSlug(t, "like"), 1)

	t.Run("insert like", func(t *testing.T) {
		err := repo.InsertLike(context.Background(), a.ID, 100)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		liked, err := repo.IsLiked(context.Background(), a.ID, 100)
		if err != nil {
			t.Fatalf("IsLiked error: %v", err)
		}
		if !liked {
			t.Error("expected liked=true")
		}
	})

	t.Run("duplicate like idempotent", func(t *testing.T) {
		err := repo.InsertLike(context.Background(), a.ID, 100)
		if err != nil {
			t.Fatalf("duplicate insert should not error: %v", err)
		}
	})

	t.Run("delete like", func(t *testing.T) {
		err := repo.DeleteLike(context.Background(), a.ID, 100)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		liked, _ := repo.IsLiked(context.Background(), a.ID, 100)
		if liked {
			t.Error("expected liked=false after delete")
		}
	})
}

func TestArticleRepo_IncrementViewCount(t *testing.T) {
	repo, _ := setupArticleRepo(t)
	a := createTestArticle(t, repo, "Views", uniSlug(t, "view"), 1)

	err := repo.IncrementViewCount(context.Background(), a.ID, 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	found, _ := repo.FindByID(context.Background(), a.ID)
	if found.ViewCount != 10 {
		t.Errorf("expected 10 views, got %d", found.ViewCount)
	}
}

func TestArticleRepo_SyncTags(t *testing.T) {
	repo, d := setupArticleRepo(t)
	a := createTestArticle(t, repo, "Tags", uniSlug(t, "tag"), 1)

	tagRepo := NewTagRepo(d)
	ts := time.Now().UnixNano()
	tag1Name := fmt.Sprintf("Vue%d", ts)
	tag1Slug := fmt.Sprintf("vue%d", ts)
	tag2Name := fmt.Sprintf("Go%d", ts+1)
	tag2Slug := fmt.Sprintf("go%d", ts+1)
	tag3Name := fmt.Sprintf("Rust%d", ts+2)
	tag3Slug := fmt.Sprintf("rust%d", ts+2)
	tag1, _ := tagRepo.FindOrCreate(context.Background(), tag1Name, tag1Slug)
	tag2, _ := tagRepo.FindOrCreate(context.Background(), tag2Name, tag2Slug)
	tag3, _ := tagRepo.FindOrCreate(context.Background(), tag3Name, tag3Slug)

	t.Run("sync tags", func(t *testing.T) {
		err := repo.SyncTags(context.Background(), a.ID, []uint{tag1.ID, tag2.ID, tag3.ID})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		found, _ := repo.FindByID(context.Background(), a.ID)
		if len(found.Tags) != 3 {
			t.Errorf("expected 3 tags, got %d", len(found.Tags))
		}
	})
}

// B-107: data 层须填充作者显示名/头像与分类名（auth/blog 同库部署，直查 users/categories）
func TestArticleRepo_EnrichesAuthorAndCategory(t *testing.T) {
	repo, d := setupArticleRepo(t)
	ctx := context.Background()

	uname := fmt.Sprintf("enrich-%d", time.Now().UnixNano())
	user := map[string]interface{}{
		"username": uname,
		"email":    uname + "@test.dev",
		"password": "hash-placeholder",
		"avatar":   "https://cdn.test.dev/avatar.png",
		"role":     "reader",
	}
	if err := d.db.Table("users").Create(user).Error; err != nil {
		t.Fatalf("种入测试用户失败: %v", err)
	}
	var uid int64
	if err := d.db.Table("users").Select("id").Where("username = ?", uname).Scan(&uid).Error; err != nil {
		t.Fatalf("查询测试用户 ID 失败: %v", err)
	}
	authorID := uint(uid)
	t.Cleanup(func() { d.db.Table("users").Where("id = ?", authorID).Delete(nil) })

	catSlug := uniSlug(t, "cat")
	cat := &CategoryPO{Name: "测试分类" + uname, Slug: catSlug, SortOrder: 0}
	if err := d.db.Create(cat).Error; err != nil {
		t.Fatalf("种入分类失败: %v", err)
	}
	t.Cleanup(func() { d.db.Delete(cat) })

	a := createTestArticle(t, repo, "富化测试", uniSlug(t, "enrich-article"), authorID)
	catID := cat.ID
	a.CategoryID = &catID
	if err := repo.(interface {
		Update(ctx context.Context, a *biz.Article) error
	}).Update(ctx, a); err != nil {
		t.Fatalf("更新文章分类失败: %v", err)
	}

	got, err := repo.FindByID(ctx, a.ID)
	if err != nil {
		t.Fatalf("FindByID 失败: %v", err)
	}
	if got.AuthorName != uname {
		t.Errorf("AuthorName 未填充: got %q want %q", got.AuthorName, uname)
	}
	if got.AuthorAvatar != "https://cdn.test.dev/avatar.png" {
		t.Errorf("AuthorAvatar 未填充: got %q", got.AuthorAvatar)
	}
	if got.CategoryName != cat.Name {
		t.Errorf("CategoryName 未填充: got %q want %q", got.CategoryName, cat.Name)
	}

	list, total, err := repo.List(ctx, biz.ArticleListQuery{Status: "", AuthorID: &authorID})
	if err != nil {
		t.Fatalf("List 失败: %v", err)
	}
	if total < 1 {
		t.Fatalf("列表应包含文章: total=%d", total)
	}
	for _, item := range list {
		if item.ID == a.ID {
			if item.AuthorName != uname || item.CategoryName != cat.Name {
				t.Errorf("列表项未富化: author=%q cat=%q", item.AuthorName, item.CategoryName)
			}
		}
	}
}

// B-202: 标签 AND 过滤须在 MySQL 上工作（回归：原 PG schema 限定 Group 写法会语法错误）
func TestArticleRepo_ListWithTagsFilter(t *testing.T) {
	repo, d := setupArticleRepo(t)
	ctx := context.Background()

	sfx := time.Now().UnixNano()
	tag1 := &TagPO{Name: fmt.Sprintf("tag-a-%d", sfx), Slug: fmt.Sprintf("tag-a-%d", sfx)}
	tag2 := &TagPO{Name: fmt.Sprintf("tag-b-%d", sfx), Slug: fmt.Sprintf("tag-b-%d", sfx)}
	if err := d.db.Create(tag1).Error; err != nil {
		t.Fatalf("创建 tag1 失败: %v", err)
	}
	if err := d.db.Create(tag2).Error; err != nil {
		t.Fatalf("创建 tag2 失败: %v", err)
	}
	t.Cleanup(func() { d.db.Where("id IN ?", []uint{tag1.ID, tag2.ID}).Delete(&TagPO{}) })

	a := createTestArticle(t, repo, "双标签文章", uniSlug(t, "both-tags"), 1)
	b := createTestArticle(t, repo, "单标签文章", uniSlug(t, "one-tag"), 1)

	if err := repo.AssociateTags(ctx, a.ID, []uint{tag1.ID, tag2.ID}); err != nil {
		t.Fatalf("关联双标签失败: %v", err)
	}
	if err := repo.AssociateTags(ctx, b.ID, []uint{tag1.ID}); err != nil {
		t.Fatalf("关联单标签失败: %v", err)
	}
	// 发布两篇（List 默认只查 published）
	for _, art := range []*biz.Article{a, b} {
		if err := d.db.Model(&ArticlePO{}).Where("id = ?", art.ID).
			Update("status", int8(biz.ArticleStatusPublished)).Error; err != nil {
			t.Fatalf("发布文章失败: %v", err)
		}
	}

	articles, total, err := repo.List(ctx, biz.ArticleListQuery{
		Status: "published",
		Tags:   []string{tag1.Name, tag2.Name},
	})
	if err != nil {
		t.Fatalf("List 带双标签过滤失败: %v", err)
	}
	if total != 1 || len(articles) != 1 || articles[0].ID != a.ID {
		t.Errorf("AND 过滤应只返回同时含两标签的文章: total=%d ids=%v", total, articleIDs(articles))
	}

	// 过滤单个标签应返回两篇
	one, oneTotal, err := repo.List(ctx, biz.ArticleListQuery{Status: "published", Tags: []string{tag1.Name}})
	if err != nil {
		t.Fatalf("List 带单标签过滤失败: %v", err)
	}
	if oneTotal != 2 || len(one) != 2 {
		t.Errorf("单标签应返回两篇: total=%d", oneTotal)
	}
}

func articleIDs(arts []*biz.Article) []uint {
	ids := make([]uint, 0, len(arts))
	for _, x := range arts {
		ids = append(ids, x.ID)
	}
	return ids
}

// B-204: 标签计数批量调整须失效 tag:all 缓存
func TestArticleRepo_UpdateTagsCountInvalidatesTagAll(t *testing.T) {
	repo, d := setupArticleRepo(t)
	ctx := context.Background()

	tag := &TagPO{Name: uniq(t, "cnt"), Slug: uniq(t, "cnt-slug")}
	if err := d.db.Create(tag).Error; err != nil {
		t.Fatalf("创建 tag 失败: %v", err)
	}
	t.Cleanup(func() { d.db.Where("id = ?", tag.ID).Delete(&TagPO{}) })

	_ = d.cache.Set(ctx, cacheKeyTagAll, []byte(`[{"id":1}]`), 0)
	if err := repo.UpdateTagsArticleCount(ctx, []uint{tag.ID}, 1); err != nil {
		t.Fatalf("UpdateTagsArticleCount 失败: %v", err)
	}
	if _, err := d.cache.Get(ctx, cacheKeyTagAll); err == nil {
		t.Error("标签计数变化后 tag:all 缓存应被失效")
	}
}

// B-205: Update 变更 slug 后须清除旧 slug→id 映射缓存
func TestArticleRepo_UpdateSlugClearsOldSlugCache(t *testing.T) {
	repo, d := setupArticleRepo(t)
	ctx := context.Background()

	oldSlug := uniSlug(t, "old-slug")
	a := createTestArticle(t, repo, "原标题", oldSlug, 1)
	t.Cleanup(func() { d.db.Unscoped().Where("id = ?", a.ID).Delete(&ArticlePO{}) })

	if err := d.cache.Set(ctx, fmt.Sprintf(keyArticleSlug, oldSlug), []byte("1"), 0); err != nil {
		t.Fatalf("预置旧 slug 缓存失败: %v", err)
	}

	newSlug := uniSlug(t, "new-slug")
	a.Slug = newSlug
	a.Title = "新标题"
	if err := repo.Update(ctx, a); err != nil {
		t.Fatalf("Update 失败: %v", err)
	}

	if _, err := d.cache.Get(ctx, fmt.Sprintf(keyArticleSlug, oldSlug)); err == nil {
		t.Error("旧 slug 映射缓存应被清除")
	}
}
