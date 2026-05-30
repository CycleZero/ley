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
