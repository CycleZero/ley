package biz

import (
	"context"
	"testing"
)

func TestArticleUseCase_CreateArticle(t *testing.T) {
	uc, _, _, _, _ := setupArticleUseCase()
	ctx := ctxWithUser(1)

	t.Run("happy path", func(t *testing.T) {
		a, err := uc.CreateArticle(ctx, "Hello World", "content here", "", "", nil, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if a.ID == 0 || a.Status != ArticleStatusDraft {
			t.Errorf("expected draft article, got id=%d status=%d", a.ID, a.Status)
		}
		if a.Slug == "" {
			t.Error("expected slug to be generated")
		}
	})

	t.Run("unauthenticated", func(t *testing.T) {
		_, err := uc.CreateArticle(context.Background(), "Hi", "content", "", "", nil, nil)
		if err != ErrUserNotAuthenticated {
			t.Errorf("expected ErrUserNotAuthenticated, got %v", err)
		}
	})

	t.Run("title too short", func(t *testing.T) {
		_, err := uc.CreateArticle(ctx, "X", "content", "", "", nil, nil)
		if err != ErrArticleTitleInvalid {
			t.Errorf("expected ErrArticleTitleInvalid, got %v", err)
		}
	})

	t.Run("content empty", func(t *testing.T) {
		_, err := uc.CreateArticle(ctx, "Valid Title", "", "", "", nil, nil)
		if err != ErrArticleContentEmpty {
			t.Errorf("expected ErrArticleContentEmpty, got %v", err)
		}
	})

	t.Run("with tags", func(t *testing.T) {
		a, err := uc.CreateArticle(ctx, "Tagged Post", "content", "", "", nil, []string{"Go", "Kratos"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if a.ID == 0 {
			t.Error("expected article created")
		}
	})
}

func TestArticleUseCase_UpdateArticle(t *testing.T) {
	uc, ar, _, _, _ := setupArticleUseCase()
	ctx := ctxWithUser(1)

	created, _ := uc.CreateArticle(ctx, "Original", "test content", "", "", nil, nil)

	t.Run("update title", func(t *testing.T) {
		newTitle := "New Title"
		a, err := uc.UpdateArticle(ctx, created.ID, &newTitle, nil, nil, nil, nil, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if a.Title != newTitle {
			t.Errorf("expected %q, got %q", newTitle, a.Title)
		}
	})

	t.Run("not owner", func(t *testing.T) {
		ctx2 := ctxWithUser(999)
		newTitle := "Hacked"
		_, err := uc.UpdateArticle(ctx2, created.ID, &newTitle, nil, nil, nil, nil, nil)
		if err != ErrNotArticleOwner {
			t.Errorf("expected ErrNotArticleOwner, got %v", err)
		}
	})

	t.Run("not found", func(t *testing.T) {
		newTitle := "Ghost"
		_, err := uc.UpdateArticle(ctx, 99999, &newTitle, nil, nil, nil, nil, nil)
		if err != ErrArticleNotFound {
			t.Errorf("expected ErrArticleNotFound, got %v", err)
		}
	})

	_ = ar
}

func TestArticleUseCase_DeleteArticle(t *testing.T) {
	uc, _, _, _, eb := setupArticleUseCase()
	ctx := ctxWithUser(1)

	created, _ := uc.CreateArticle(ctx, "ToDelete", "content", "", "", nil, nil)

	t.Run("delete own article", func(t *testing.T) {
		err := uc.DeleteArticle(ctx, created.ID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		events := eb.Events()
		found := false
		for _, e := range events {
			if e.Topic == TopicArticleDeleted { found = true }
		}
		if !found {
			t.Error("expected delete event published")
		}
	})

	t.Run("not owner", func(t *testing.T) {
		a2, _ := uc.CreateArticle(ctxWithUser(2), "User2", "content", "", "", nil, nil)
		err := uc.DeleteArticle(ctx, a2.ID)
		if err != ErrNotArticleOwner {
			t.Errorf("expected ErrNotArticleOwner, got %v", err)
		}
	})
}

func TestArticleUseCase_PublishArticle(t *testing.T) {
	uc, _, _, _, eb := setupArticleUseCase()
	ctx := ctxWithUser(1)

	created, _ := uc.CreateArticle(ctx, "Draft", "content", "", "", nil, nil)

	t.Run("publish draft", func(t *testing.T) {
		a, err := uc.PublishArticle(ctx, created.ID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if a.Status != ArticleStatusPublished {
			t.Errorf("expected published, got %d", a.Status)
		}
		events := eb.Events()
		found := false
		for _, e := range events {
			if e.Topic == TopicArticlePublished { found = true }
		}
		if !found {
			t.Error("expected publish event")
		}
	})

	t.Run("already published", func(t *testing.T) {
		_, err := uc.PublishArticle(ctx, created.ID)
		if err != ErrArticleAlreadyPublished {
			t.Errorf("expected ErrArticleAlreadyPublished, got %v", err)
		}
	})
}

func TestArticleUseCase_ArchiveArticle(t *testing.T) {
	uc, _, _, _, _ := setupArticleUseCase()
	ctx := ctxWithUser(1)

	created, _ := uc.CreateArticle(ctx, "ToArchive", "content", "", "", nil, nil)
	// Publish first
	uc.PublishArticle(ctx, created.ID)

	a, err := uc.ArchiveArticle(ctx, created.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if a.Status != ArticleStatusArchived {
		t.Errorf("expected archived, got %d", a.Status)
	}
}

func TestArticleUseCase_GetArticle(t *testing.T) {
	uc, _, _, _, _ := setupArticleUseCase()
	ctx := ctxWithUser(1)

	created, _ := uc.CreateArticle(ctx, "FindMe", "content", "", "", nil, nil)

	t.Run("by ID", func(t *testing.T) {
		a, err := uc.GetArticle(ctx, "1")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if a.Title != "FindMe" {
			t.Errorf("expected FindMe, got %s", a.Title)
		}
	})

	t.Run("by slug", func(t *testing.T) {
		a, err := uc.GetArticle(ctx, created.Slug)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if a.ID != created.ID {
			t.Errorf("expected ID %d, got %d", created.ID, a.ID)
		}
	})
}

func TestArticleUseCase_ListArticles(t *testing.T) {
	uc, _, _, _, _ := setupArticleUseCase()
	ctx := ctxWithUser(1)

	uc.CreateArticle(ctx, "Article A", "content", "", "", nil, nil)
	uc.CreateArticle(ctx, "Article B", "content", "", "", nil, nil)

	articles, total, err := uc.ListArticles(ctx, ArticleListQuery{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if total < 2 {
		t.Errorf("expected at least 2 articles, got %d", total)
	}
	_ = articles
}

func TestArticleUseCase_LikeArticle(t *testing.T) {
	uc, ar, _, _, eb := setupArticleUseCase()
	ctx := ctxWithUser(1)

	created, _ := uc.CreateArticle(ctx, "LikeMe", "content", "", "", nil, nil)

	t.Run("like", func(t *testing.T) {
		err := uc.LikeArticle(ctx, created.ID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		liked, _ := ar.IsLiked(context.Background(), created.ID, 1)
		if !liked {
			t.Error("expected liked=true")
		}
	})

	t.Run("unlike", func(t *testing.T) {
		err := uc.UnlikeArticle(ctx, created.ID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		liked, _ := ar.IsLiked(context.Background(), created.ID, 1)
		if liked {
			t.Error("expected liked=false")
		}
	})

	_ = eb
}
