package data

import (
	"context"
	"io"
	"testing"

	"github.com/CycleZero/ley/app/blog/internal/biz"
	"github.com/CycleZero/ley/pkg/oss"
)

// mockOSS implements oss.OSS for unit tests.
type mockOSS struct{}

func (m *mockOSS) PutObject(ctx context.Context, key string, reader io.Reader, size int64, contentType string) error {
	return nil
}
func (m *mockOSS) GetObject(ctx context.Context, key string) (io.ReadCloser, *oss.ObjectInfo, error) {
	return nil, nil, nil
}
func (m *mockOSS) StatObject(ctx context.Context, key string) (*oss.ObjectInfo, error) {
	return nil, nil
}
func (m *mockOSS) DeleteObject(ctx context.Context, key string) error {
	return nil
}
func (m *mockOSS) GetPresignedURL(ctx context.Context, key string, expirySeconds int64) (string, error) {
	return "", nil
}
func (m *mockOSS) CopyObject(ctx context.Context, sourceKey, destKey string) error {
	return nil
}
func (m *mockOSS) ListObjects(ctx context.Context, prefix string) ([]oss.ObjectInfo, error) {
	return nil, nil
}
func (m *mockOSS) GetPresignedPutURL(ctx context.Context, key, contentType string, expirySeconds int64) (string, error) {
	return "https://presigned.example.com/" + key, nil
}

func setupSiteRepo(t *testing.T) (biz.SiteRepo, *Data) {
	t.Helper()
	d := newTestData(t)
	d.db.AutoMigrate(&SiteSettingPO{}, &SiteBackgroundPO{})
	return &siteRepo{data: d, oss: &mockOSS{}}, d
}

// =============================================================================
// GetConfig / SaveConfig
// =============================================================================

func TestSiteRepo_GetConfig(t *testing.T) {
	repo, _ := setupSiteRepo(t)

	t.Run("empty config returns default", func(t *testing.T) {
		cfg, err := repo.GetConfig(context.Background())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		// Should return empty config, not nil
		if cfg == nil {
			t.Fatal("expected non-nil config")
		}
	})
}

func TestSiteRepo_SaveConfig(t *testing.T) {
	repo, _ := setupSiteRepo(t)

	t.Run("save and retrieve", func(t *testing.T) {
		cfg := &biz.SiteSetting{SiteTitle: "Test Blog"}
		err := repo.SaveConfig(context.Background(), cfg)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		got, err := repo.GetConfig(context.Background())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got.SiteTitle != "Test Blog" {
			t.Errorf("expected Test Blog, got %s", got.SiteTitle)
		}
	})

	t.Run("update existing", func(t *testing.T) {
		cfg := &biz.SiteSetting{SiteTitle: "Updated"}
		err := repo.SaveConfig(context.Background(), cfg)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		got, _ := repo.GetConfig(context.Background())
		if got.SiteTitle != "Updated" {
			t.Errorf("expected Updated, got %s", got.SiteTitle)
		}
	})
}

// =============================================================================
// CreateBackground / ListBackgrounds / DeleteBackground / SetActiveBackground
// =============================================================================

func TestSiteRepo_Backgrounds(t *testing.T) {
	repo, _ := setupSiteRepo(t)

	t.Run("list empty", func(t *testing.T) {
		bgs, err := repo.ListBackgrounds(context.Background())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(bgs) != 0 {
			t.Errorf("expected 0 backgrounds, got %d", len(bgs))
		}
	})

	t.Run("create", func(t *testing.T) {
		bg := &biz.SiteBackground{Filename: "sunset.jpg", SortOrder: 1}
		err := repo.CreateBackground(context.Background(), bg, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if bg.ID == 0 {
			t.Error("expected ID to be populated")
		}
		if bg.CreatedAt.IsZero() {
			t.Error("expected CreatedAt to be populated")
		}
	})

	t.Run("list after create", func(t *testing.T) {
		bgs, err := repo.ListBackgrounds(context.Background())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(bgs) != 1 {
			t.Errorf("expected 1 background, got %d", len(bgs))
		}
	})

	t.Run("set active", func(t *testing.T) {
		bgs, _ := repo.ListBackgrounds(context.Background())
		err := repo.SetActiveBackground(context.Background(), bgs[0].ID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		bgs, _ = repo.ListBackgrounds(context.Background())
		if !bgs[0].IsActive {
			t.Error("expected is_active=true")
		}
	})

	t.Run("set active non-existent", func(t *testing.T) {
		err := repo.SetActiveBackground(context.Background(), 99999)
		if err == nil {
			t.Error("expected error for non-existent background")
		}
	})

	t.Run("delete", func(t *testing.T) {
		bgs, _ := repo.ListBackgrounds(context.Background())
		err := repo.DeleteBackground(context.Background(), bgs[0].ID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		bgs, _ = repo.ListBackgrounds(context.Background())
		if len(bgs) != 0 {
			t.Errorf("expected 0 after delete, got %d", len(bgs))
		}
	})
}

func TestSiteRepo_SaveConfig_Deduplication(t *testing.T) {
	repo, _ := setupSiteRepo(t)

	// Verify SaveConfig uses INSERT ... ON DUPLICATE KEY UPDATE behavior
	for i := 0; i < 5; i++ {
		cfg := &biz.SiteSetting{SiteTitle: "Save " + string(rune('A'+i))}
		err := repo.SaveConfig(context.Background(), cfg)
		if err != nil {
			t.Fatalf("save %d: %v", i, err)
		}
	}
	// Should only have one row (id=1) with latest value
	got, _ := repo.GetConfig(context.Background())
	if got.SiteTitle != "Save E" {
		t.Errorf("expected 'Save E', got %s", got.SiteTitle)
	}
}
