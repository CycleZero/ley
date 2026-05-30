package data

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/CycleZero/ley/app/blog/internal/biz"
)

func setupTagRepo(t *testing.T) (biz.TagRepo, *Data) {
	t.Helper()
	d := newTestData(t)
	return NewTagRepo(d), d
}

func uniq(t *testing.T, prefix string) string {
	t.Helper()
	return fmt.Sprintf("%s-%d", prefix, time.Now().UnixNano())
}

func TestTagRepo_Create(t *testing.T) {
	repo, _ := setupTagRepo(t)

	t.Run("happy path", func(t *testing.T) {
		name := uniq(t, "TypeScript")
		tag, err := repo.FindOrCreate(context.Background(), name, uniq(t, "ts"))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if tag.ID == 0 || tag.Name != name {
			t.Errorf("expected %s tag, got %+v", name, tag)
		}
	})

	t.Run("duplicate returns existing", func(t *testing.T) {
		name := uniq(t, "GoLang")
		slug := uniq(t, "go")
		t1, _ := repo.FindOrCreate(context.Background(), name, slug)
		t2, err := repo.FindOrCreate(context.Background(), name, slug)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if t1.ID != t2.ID {
			t.Errorf("expected same tag ID, got %d vs %d", t1.ID, t2.ID)
		}
	})
}

func TestTagRepo_List(t *testing.T) {
	repo, _ := setupTagRepo(t)

	t.Run("contains created tags", func(t *testing.T) {
		name := uniq(t, "MyTag")
		created, _ := repo.FindOrCreate(context.Background(), name, uniq(t, "mt"))
		tags, err := repo.List(context.Background())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		found := false
		for _, tag := range tags {
			if tag.ID == created.ID {
				found = true
				break
			}
		}
		if !found {
			t.Error("expected created tag in list")
		}
	})
}

func TestTagRepo_Delete(t *testing.T) {
	repo, _ := setupTagRepo(t)

	t.Run("happy path", func(t *testing.T) {
		created, _ := repo.FindOrCreate(context.Background(), uniq(t, "Rust"), uniq(t, "rust"))
		err := repo.Delete(context.Background(), created.ID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("not found", func(t *testing.T) {
		err := repo.Delete(context.Background(), 99999)
		if err == nil {
			t.Error("expected error for non-existent tag")
		}
	})
}

// =============================================================================
// CategoryRepo
// =============================================================================

func setupCategoryRepo(t *testing.T) (biz.CategoryRepo, *Data) {
	t.Helper()
	d := newTestData(t)
	return NewCategoryRepo(d), d
}

func TestCategoryRepo_Create(t *testing.T) {
	repo, _ := setupCategoryRepo(t)

	t.Run("top-level category", func(t *testing.T) {
		cat := &biz.Category{Name: uniq(t, "Frontend"), Slug: uniq(t, "fe"), SortOrder: 1}
		err := repo.Create(context.Background(), cat)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cat.ID == 0 {
			t.Error("expected ID to be populated")
		}
	})

	t.Run("child category", func(t *testing.T) {
		parent := &biz.Category{Name: uniq(t, "Backend"), Slug: uniq(t, "be"), SortOrder: 2}
		repo.Create(context.Background(), parent)

		child := &biz.Category{Name: uniq(t, "Go"), Slug: uniq(t, "go-cat"), ParentID: &parent.ID, SortOrder: 1}
		err := repo.Create(context.Background(), child)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if child.ParentID == nil || *child.ParentID != parent.ID {
			t.Error("expected parent ID to match")
		}
	})
}

func TestCategoryRepo_Update(t *testing.T) {
	repo, _ := setupCategoryRepo(t)

	cat := &biz.Category{Name: uniq(t, "OldCat"), Slug: uniq(t, "old"), SortOrder: 1}
	repo.Create(context.Background(), cat)

	t.Run("happy path", func(t *testing.T) {
		newName := uniq(t, "Updated")
		cat.Name = newName
		err := repo.Update(context.Background(), cat)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		found, _ := repo.FindByID(context.Background(), cat.ID)
		if found.Name != newName {
			t.Errorf("expected %s, got %s", newName, found.Name)
		}
	})
}

func TestCategoryRepo_ListTree(t *testing.T) {
	repo, _ := setupCategoryRepo(t)

	t.Run("contains created tree", func(t *testing.T) {
		root := &biz.Category{Name: uniq(t, "RootCat"), Slug: uniq(t, "root"), SortOrder: 1}
		repo.Create(context.Background(), root)
		sub := &biz.Category{Name: uniq(t, "SubCat"), Slug: uniq(t, "sub"), ParentID: &root.ID, SortOrder: 1}
		repo.Create(context.Background(), sub)

		tree, err := repo.ListTree(context.Background())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		// Find our root in the tree
		var foundRoot *biz.Category
		for _, c := range tree {
			if c.ID == root.ID {
				foundRoot = c
				break
			}
		}
		if foundRoot == nil {
			t.Fatal("created root not found in tree")
		}
		if len(foundRoot.Children) != 1 {
			t.Errorf("expected 1 child, got %d", len(foundRoot.Children))
		}
	})
}

func TestCategoryRepo_ListChildren(t *testing.T) {
	repo, _ := setupCategoryRepo(t)

	root := &biz.Category{Name: uniq(t, "DevRoot"), Slug: uniq(t, "devr"), SortOrder: 1}
	repo.Create(context.Background(), root)
	c1 := &biz.Category{Name: uniq(t, "ChildA"), Slug: uniq(t, "cha"), ParentID: &root.ID, SortOrder: 2}
	repo.Create(context.Background(), c1)
	c2 := &biz.Category{Name: uniq(t, "ChildB"), Slug: uniq(t, "chb"), ParentID: &root.ID, SortOrder: 1}
	repo.Create(context.Background(), c2)

	children, err := repo.ListChildren(context.Background(), root.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(children) != 2 {
		t.Errorf("expected 2 children, got %d", len(children))
	}
	if children[0].SortOrder > children[1].SortOrder {
		t.Error("expected children sorted by sort_order ASC")
	}
}

func TestCategoryRepo_Delete(t *testing.T) {
	repo, _ := setupCategoryRepo(t)
	cat := &biz.Category{Name: uniq(t, "TempCat"), Slug: uniq(t, "tmp"), SortOrder: 1}
	repo.Create(context.Background(), cat)
	err := repo.Delete(context.Background(), cat.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCategoryRepo_IncrementArticleCount(t *testing.T) {
	repo, _ := setupCategoryRepo(t)
	cat := &biz.Category{Name: uniq(t, "CountCat"), Slug: uniq(t, "cnt"), SortOrder: 1}
	repo.Create(context.Background(), cat)
	err := repo.IncrementArticleCount(context.Background(), cat.ID, 5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	found, _ := repo.FindByID(context.Background(), cat.ID)
	if found.ArticleCount != 5 {
		t.Errorf("expected 5, got %d", found.ArticleCount)
	}
}
