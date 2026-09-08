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
			if e.Topic == TopicArticleDeleted {
				found = true
			}
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
			if e.Topic == TopicArticlePublished {
				found = true
			}
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

func TestArticleUseCase_SearchArticles(t *testing.T) {
	uc, _, _, _, _ := setupArticleUseCase()
	ctx := ctxWithUser(1)

	// 创建一篇已发布文章（含搜索关键词）
	created, err := uc.CreateArticle(ctx, "Kratos 微服务入门", "本文介绍 Kratos 框架的使用", "", "", nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, err := uc.PublishArticle(ctx, created.ID); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	t.Run("匹配标题", func(t *testing.T) {
		articles, total, err := uc.SearchArticles(ctx, "Kratos", 1, 10)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if total != 1 || len(articles) != 1 {
			t.Errorf("expected 1 result, got total=%d len=%d", total, len(articles))
		}
	})

	t.Run("匹配正文", func(t *testing.T) {
		articles, total, err := uc.SearchArticles(ctx, "框架", 1, 10)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if total != 1 || len(articles) != 1 {
			t.Errorf("expected 1 result, got total=%d len=%d", total, len(articles))
		}
	})

	t.Run("无结果", func(t *testing.T) {
		articles, total, err := uc.SearchArticles(ctx, "不存在的关键词", 1, 10)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if total != 0 || len(articles) != 0 {
			t.Errorf("expected 0 results, got total=%d len=%d", total, len(articles))
		}
	})

	t.Run("空关键词返回空", func(t *testing.T) {
		articles, total, err := uc.SearchArticles(ctx, "   ", 1, 10)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if total != 0 || len(articles) != 0 {
			t.Errorf("expected 0 results, got total=%d len=%d", total, len(articles))
		}
	})

	t.Run("草稿不可搜到", func(t *testing.T) {
		draft, err := uc.CreateArticle(ctx, "草稿中的机密内容", "secret", "", "", nil, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		_ = draft // 保持草稿状态
		articles, total, err := uc.SearchArticles(ctx, "机密内容", 1, 10)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if total != 0 || len(articles) != 0 {
			t.Errorf("expected 0 results for draft, got total=%d len=%d", total, len(articles))
		}
	})
}

func TestArticleUseCase_LikeArticle(t *testing.T) {
	uc, ar, _, _, eb := setupArticleUseCase()
	ctx := ctxWithUser(1)

	created, _ := uc.CreateArticle(ctx, "LikeMe", "content", "", "", nil, nil)
	if _, err := uc.PublishArticle(ctx, created.ID); err != nil {
		t.Fatalf("PublishArticle 失败: %v", err)
	}

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

// B-105: 草稿/归档仅作者或管理员可见（详情 + 列表两层校验）
func TestArticleUseCase_GetArticleVisibility(t *testing.T) {
	uc, _, _, _, _ := setupArticleUseCase()
	ctxAuthor := ctxWithUser(1)
	ctxOther := ctxWithRole(2, "reader")
	ctxAdmin := ctxWithRole(9, "admin")

	draft, err := uc.CreateArticle(ctxAuthor, "DraftOnly", "content", "", "", nil, nil)
	if err != nil {
		t.Fatalf("CreateArticle 失败: %v", err)
	}

	t.Run("anonymous cannot read draft", func(t *testing.T) {
		if _, err := uc.GetArticle(context.Background(), "1"); err != ErrArticleNotFound {
			t.Errorf("匿名读草稿应 404: got %v", err)
		}
	})
	t.Run("other reader cannot read draft", func(t *testing.T) {
		if _, err := uc.GetArticle(ctxOther, "1"); err != ErrArticleNotFound {
			t.Errorf("非作者读草稿应 404: got %v", err)
		}
	})
	t.Run("author can read own draft", func(t *testing.T) {
		a, err := uc.GetArticle(ctxAuthor, "1")
		if err != nil || a.ID != draft.ID {
			t.Errorf("作者读自己草稿应成功: err=%v id=%d", err, draft.ID)
		}
	})
	t.Run("admin can read others draft", func(t *testing.T) {
		a, err := uc.GetArticle(ctxAdmin, "1")
		if err != nil || a.ID != draft.ID {
			t.Errorf("admin 读他人草稿应成功: err=%v", err)
		}
	})

	pub, err := uc.CreateArticle(ctxAuthor, "PublishedOne", "content", "", "", nil, nil)
	if err != nil {
		t.Fatalf("CreateArticle 失败: %v", err)
	}
	if _, err := uc.PublishArticle(ctxAuthor, pub.ID); err != nil {
		t.Fatalf("PublishArticle 失败: %v", err)
	}

	t.Run("anonymous can read published", func(t *testing.T) {
		if _, err := uc.GetArticle(context.Background(), "2"); err != nil {
			t.Errorf("匿名读已发布文章应成功: %v", err)
		}
	})
	t.Run("anonymous cannot read archived", func(t *testing.T) {
		if _, err := uc.ArchiveArticle(ctxAuthor, pub.ID); err != nil {
			t.Fatalf("ArchiveArticle 失败: %v", err)
		}
		if _, err := uc.GetArticle(context.Background(), "2"); err != ErrArticleNotFound {
			t.Errorf("匿名读已归档文章应 404: got %v", err)
		}
	})
}

func TestArticleUseCase_ListArticlesVisibility(t *testing.T) {
	uc, _, _, _, _ := setupArticleUseCase()
	ctxAuthor := ctxWithUser(1)
	ctxAdmin := ctxWithRole(9, "admin")

	if _, err := uc.CreateArticle(ctxAuthor, "D1", "content", "", "", nil, nil); err != nil {
		t.Fatalf("CreateArticle 失败: %v", err)
	}

	t.Run("anonymous draft list forbidden", func(t *testing.T) {
		_, _, err := uc.ListArticles(context.Background(), ArticleListQuery{Status: "draft"})
		if err != ErrPermissionDenied {
			t.Errorf("匿名列草稿应拒绝: got %v", err)
		}
	})
	t.Run("reader draft list forbidden", func(t *testing.T) {
		_, _, err := uc.ListArticles(ctxWithRole(2, "reader"), ArticleListQuery{Status: "draft"})
		if err != ErrPermissionDenied {
			t.Errorf("reader 列草稿应拒绝: got %v", err)
		}
	})
	t.Run("admin can list drafts", func(t *testing.T) {
		if _, _, err := uc.ListArticles(ctxAdmin, ArticleListQuery{Status: "draft"}); err != nil {
			t.Errorf("admin 列草稿应放行: %v", err)
		}
	})
	t.Run("author self drafts ok", func(t *testing.T) {
		aid := uint(1)
		if _, _, err := uc.ListArticles(ctxAuthor, ArticleListQuery{Status: "draft", AuthorID: &aid}); err != nil {
			t.Errorf("作者列自己草稿应放行: %v", err)
		}
	})
	t.Run("anonymous published list ok", func(t *testing.T) {
		if _, _, err := uc.ListArticles(context.Background(), ArticleListQuery{Status: "published"}); err != nil {
			t.Errorf("匿名列已发布应放行: %v", err)
		}
	})
}

// B-207: 点赞语义——非发布文章/不存在文章不可点赞；开关关闭时拒绝
func TestArticleUseCase_LikeRequiresPublishedAndGate(t *testing.T) {
	uc, _, _, _, _ := setupArticleUseCase()
	ctxAuthor := ctxWithUser(1)

	draft, err := uc.CreateArticle(ctxAuthor, "LikeDraft", "c", "", "", nil, nil)
	if err != nil {
		t.Fatalf("CreateArticle 失败: %v", err)
	}

	t.Run("draft not likeable", func(t *testing.T) {
		if err := uc.LikeArticle(ctxAuthor, draft.ID); err != ErrArticleNotFound {
			t.Errorf("草稿不可点赞: got %v", err)
		}
	})
	t.Run("nonexistent not likeable", func(t *testing.T) {
		if err := uc.LikeArticle(ctxAuthor, 99999); err != ErrArticleNotFound {
			t.Errorf("不存在文章不可点赞: got %v", err)
		}
		if err := uc.UnlikeArticle(ctxAuthor, 99999); err != ErrArticleNotFound {
			t.Errorf("不存在文章不可取消赞: got %v", err)
		}
	})
}

// B-207: 全站点赞开关关闭时拒绝点赞
func TestArticleUseCase_LikesDisabledByGate(t *testing.T) {
	uc, _, _, _, _ := setupArticleUseCaseWithGate(&stubGate{enabled: false})
	ctx := ctxWithUser(1)

	created, err := uc.CreateArticle(ctx, "Gated", "c", "", "", nil, nil)
	if err != nil {
		t.Fatalf("CreateArticle 失败: %v", err)
	}
	if _, err := uc.PublishArticle(ctx, created.ID); err != nil {
		t.Fatalf("PublishArticle 失败: %v", err)
	}
	if err := uc.LikeArticle(ctx, created.ID); err != ErrLikesDisabled {
		t.Errorf("开关关闭应拒绝点赞: got %v", err)
	}
}
