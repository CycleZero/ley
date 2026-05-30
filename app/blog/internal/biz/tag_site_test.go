package biz

import (
	"context"
	"testing"
)

func TestTagUseCase_CreateTag(t *testing.T) {
	uc, _ := setupTagUseCase()

	t.Run("happy path", func(t *testing.T) {
		tag, err := uc.CreateTag(context.Background(), "TypeScript")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if tag.Name != "TypeScript" || tag.Slug != "typescript" {
			t.Errorf("unexpected tag: %+v", tag)
		}
	})

	t.Run("empty name", func(t *testing.T) {
		_, err := uc.CreateTag(context.Background(), "   ")
		if err != ErrTagNameEmpty {
			t.Errorf("expected ErrTagNameEmpty, got %v", err)
		}
	})

	t.Run("duplicate", func(t *testing.T) {
		uc.CreateTag(context.Background(), "Go")
		_, err := uc.CreateTag(context.Background(), "Go")
		if err != ErrTagNameExists {
			t.Errorf("expected ErrTagNameExists, got %v", err)
		}
	})
}

func TestTagUseCase_ListTags(t *testing.T) {
	uc, _ := setupTagUseCase()
	uc.CreateTag(context.Background(), "Vue")
	uc.CreateTag(context.Background(), "React")

	tags, err := uc.ListTags(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tags) != 2 {
		t.Errorf("expected 2 tags, got %d", len(tags))
	}
}

func TestTagUseCase_DeleteTag(t *testing.T) {
	uc, _ := setupTagUseCase()
	tag, _ := uc.CreateTag(context.Background(), "Rust")

	err := uc.DeleteTag(context.Background(), tag.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCategoryUseCase_CreateCategory(t *testing.T) {
	uc, _ := setupCategoryUseCase()

	t.Run("root category", func(t *testing.T) {
		cat, err := uc.CreateCategory(context.Background(), "Frontend", "", "", nil, 1)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cat.ID == 0 || cat.ParentID != nil {
			t.Error("expected root category")
		}
	})

	t.Run("empty name", func(t *testing.T) {
		_, err := uc.CreateCategory(context.Background(), "  ", "slug", "", nil, 0)
		if err == nil {
			t.Error("expected error for empty name")
		}
	})
}

func TestCategoryUseCase_DeleteCategory(t *testing.T) {
	uc, cr := setupCategoryUseCase()

	cat := &Category{Name: "DevOps", Slug: "devops"}
	cr.Create(context.Background(), cat)

	t.Run("delete empty category", func(t *testing.T) {
		err := uc.DeleteCategory(context.Background(), cat.ID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("delete with children blocks", func(t *testing.T) {
		parent := &Category{Name: "Parent", Slug: "parent"}
		cr.Create(context.Background(), parent)
		pid := parent.ID
		child := &Category{Name: "Child", Slug: "child", ParentID: &pid}
		cr.Create(context.Background(), child)

		err := uc.DeleteCategory(context.Background(), parent.ID)
		if err != ErrCategoryHasChildren {
			t.Errorf("expected ErrCategoryHasChildren, got %v", err)
		}
	})
}

func TestCategoryUseCase_UpdateCategory(t *testing.T) {
	uc, cr := setupCategoryUseCase()

	cat := &Category{Name: "Old", Slug: "old"}
	cr.Create(context.Background(), cat)

	_, err := uc.UpdateCategory(context.Background(), cat.ID, "NewName", "new-slug", "desc", nil, 2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	found, _ := cr.FindByID(context.Background(), cat.ID)
	if found.Name != "NewName" {
		t.Errorf("expected NewName, got %s", found.Name)
	}
}

func TestCategoryUseCase_ListCategories(t *testing.T) {
	uc, _ := setupCategoryUseCase()
	uc.CreateCategory(context.Background(), "Tech", "tech", "", nil, 1)

	cats, err := uc.ListCategories(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cats) == 0 {
		t.Error("expected at least 1 category")
	}
}

// =============================================================================
// SiteUseCase
// =============================================================================

func TestSiteUseCase_GetConfig(t *testing.T) {
	uc, sr := setupSiteUseCase()
	sr.config = &SiteSetting{SiteTitle: "My Blog"}

	cfg, err := uc.GetConfig(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.SiteTitle != "My Blog" {
		t.Errorf("expected My Blog, got %s", cfg.SiteTitle)
	}
}

func TestSiteUseCase_UpdateConfig(t *testing.T) {
	uc, sr := setupSiteUseCase()
	sr.config = &SiteSetting{SiteTitle: "Old", SiteSubtitle: "Keep"}

	cfg, err := uc.UpdateConfig(context.Background(), &SiteSetting{SiteTitle: "New"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.SiteTitle != "New" {
		t.Errorf("expected New, got %s", cfg.SiteTitle)
	}
	if cfg.SiteSubtitle != "Keep" {
		t.Errorf("expected Keep preserved, got %s", cfg.SiteSubtitle)
	}
}

func TestSiteUseCase_Backgrounds(t *testing.T) {
	uc, _ := setupSiteUseCase()

	validImage := makeValidImageContent()

	bg, err := uc.AddBackground(context.Background(), "photo.jpg", validImage)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if bg.ID == 0 {
		t.Error("expected ID populated")
	}

	bgs, _ := uc.ListBackgrounds(context.Background())
	if len(bgs) != 1 {
		t.Errorf("expected 1 background, got %d", len(bgs))
	}

	err = uc.SetActiveBackground(context.Background(), bg.ID)
	if err != nil {
		t.Fatalf("SetActive error: %v", err)
	}

	err = uc.DeleteBackground(context.Background(), bg.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSiteUseCase_UpdatePlaylist(t *testing.T) {
	uc, _ := setupSiteUseCase()

	pl, err := uc.UpdatePlaylist(context.Background(), &MusicPlaylist{
		Tracks: []MusicTrack{
			{Title: "Song1", Artist: "Artist1", URL: "https://music.example.com/1.mp3", CoverURL: ""},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(pl.Tracks) != 1 {
		t.Errorf("expected 1 track, got %d", len(pl.Tracks))
	}

	t.Run("invalid url", func(t *testing.T) {
		_, err := uc.UpdatePlaylist(context.Background(), &MusicPlaylist{
			Tracks: []MusicTrack{{Title: "Bad", URL: "ftp://bad.com/music.mp3"}},
		})
		if err != ErrInvalidMusicURL {
			t.Errorf("expected ErrInvalidMusicURL, got %v", err)
		}
	})
}

func TestSiteUseCase_AddBackground_InvalidImage(t *testing.T) {
	uc, _ := setupSiteUseCase()
	_, err := uc.AddBackground(context.Background(), "notimage.txt", []byte("hello world"))
	if err != ErrInvalidImageFormat {
		t.Errorf("expected ErrInvalidImageFormat, got %v", err)
	}
}

// =============================================================================
// Pure Functions
// =============================================================================

func makeValidImageContent() []byte {
	// Minimal valid PNG
	return []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A, 0, 0, 0, 0}
}

func TestValidateArticleInput(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		err := validateArticleInput("Hello World", "content")
		if err != nil {
			t.Errorf("expected nil, got %v", err)
		}
	})
	t.Run("title too short", func(t *testing.T) {
		err := validateArticleInput("X", "content")
		if err != ErrArticleTitleInvalid {
			t.Errorf("expected ErrArticleTitleInvalid, got %v", err)
		}
	})
	t.Run("content empty", func(t *testing.T) {
		err := validateArticleInput("Valid Title", "")
		if err != ErrArticleContentEmpty {
			t.Errorf("expected ErrArticleContentEmpty, got %v", err)
		}
	})
}

func TestGenerateSlug(t *testing.T) {
	slug := generateSlug("Hello World")
	if slug == "" {
		t.Error("expected non-empty slug")
	}
}

func TestTruncateContent(t *testing.T) {
	short := "hello"
	result := truncateContent(short, 500)
	if result != short {
		t.Errorf("expected %q, got %q", short, result)
	}

	long := make([]byte, 600)
	for i := range long { long[i] = 'a' }
	result = truncateContent(string(long), 500)
	if len([]rune(result)) > 503 { // 500 + "..."
		t.Errorf("expected truncated, got len=%d", len(result))
	}
}

func TestMergeConfig(t *testing.T) {
	old := &SiteSetting{SiteTitle: "Old", SiteSubtitle: "Keep"}
	new := &SiteSetting{SiteTitle: "New"}

	result := mergeConfig(old, new)
	if result.SiteTitle != "New" {
		t.Errorf("expected New title, got %s", result.SiteTitle)
	}
	if result.SiteSubtitle != "Keep" {
		t.Errorf("expected Keep subtitle, got %s", result.SiteSubtitle)
	}
}

func TestVerifyMagicNumber(t *testing.T) {
	// JPEG magic: FF D8 FF
	jpeg := []byte{0xFF, 0xD8, 0xFF, 0xE0}
	if !verifyMagicNumber(jpeg, "image/jpeg") {
		t.Error("expected JPEG magic match")
	}
	if verifyMagicNumber(jpeg, "image/png") {
		t.Error("expected JPEG NOT match PNG")
	}
}

func TestIsImageContent(t *testing.T) {
	if !isImageContent(makeValidImageContent()) {
		t.Error("PNG should be valid image")
	}
	if isImageContent([]byte("not an image")) {
		t.Error("text should not be valid image")
	}
}

func TestNormalizeCategoryID(t *testing.T) {
	zero := uint(0)
	if normalizeCategoryID(&zero) != nil {
		t.Error("expected nil for 0")
	}
	one := uint(1)
	if normalizeCategoryID(&one) == nil || *normalizeCategoryID(&one) != 1 {
		t.Error("expected 1 for 1")
	}
	if normalizeCategoryID(nil) != nil {
		t.Error("expected nil for nil")
	}
}

func TestTagIDs(t *testing.T) {
	tags := []*Tag{{ID: 1}, {ID: 2}, {ID: 3}}
	ids := tagIDs(tags)
	if len(ids) != 3 || ids[0] != 1 || ids[2] != 3 {
		t.Errorf("expected [1,2,3], got %v", ids)
	}
}

func TestTagNameToSlug(t *testing.T) {
	slug := tagNameToSlug("My Tag Name")
	if slug != "my-tag-name" {
		t.Errorf("expected my-tag-name, got %s", slug)
	}
}
