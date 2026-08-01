package data

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/CycleZero/ley/app/auth/internal/biz"
	kerrors "github.com/go-kratos/kratos/v2/errors"
)

// 唯一用户名/邮箱，避免与历史数据冲突（沿用项目 uniSlug 惯例）
func uniq(prefix string) string {
	return fmt.Sprintf("%s_%d", prefix, time.Now().UnixNano())
}

func cleanupUser(t *testing.T, repo biz.UserRepo, id uint) {
	t.Helper()
	if id != 0 {
		_ = repo.Delete(context.Background(), id)
	}
}

func newTestRepo(t *testing.T) (biz.UserRepo, *Data) {
	t.Helper()
	d := newTestData(t)
	return NewUserRepo(d), d
}

func makeUser(username, email string) *biz.User {
	return &biz.User{
		Username: username,
		Email:    email,
		Password: "hashed-password",
		Avatar:   "",
		Bio:      "",
		Status:   biz.UserStatusActive,
		Role:     biz.RoleReader,
	}
}

func TestUserRepo_CreateAndFindByID(t *testing.T) {
	repo, _ := newTestRepo(t)
	ctx := context.Background()

	u := makeUser(uniq("create"), uniq("create")+"@example.com")
	if err := repo.Create(ctx, u); err != nil {
		t.Fatalf("Create 失败: %v", err)
	}
	defer cleanupUser(t, repo, u.ID)

	if u.ID == 0 {
		t.Fatal("Create 后应回填 ID")
	}
	got, err := repo.FindByID(ctx, u.ID)
	if err != nil {
		t.Fatalf("FindByID 失败: %v", err)
	}
	if got.Username != u.Username || got.Email != u.Email {
		t.Errorf("用户不匹配: %+v", got)
	}
	if got.Role != biz.RoleReader || got.Status != biz.UserStatusActive {
		t.Errorf("默认角色/状态错误: %+v", got)
	}
}

func TestUserRepo_CreateDuplicate(t *testing.T) {
	repo, _ := newTestRepo(t)
	ctx := context.Background()

	username := uniq("dup")
	u1 := makeUser(username, uniq("dup")+"@example.com")
	if err := repo.Create(ctx, u1); err != nil {
		t.Fatalf("第一次 Create 失败: %v", err)
	}
	defer cleanupUser(t, repo, u1.ID)

	// 相同用户名 → 唯一约束冲突（映射为 409 ErrUserDuplicate）
	u2 := makeUser(username, uniq("dup")+"@example.com")
	err := repo.Create(ctx, u2)
	if err == nil {
		t.Fatal("重复用户名应报错")
	}
	if !kerrors.IsConflict(err) {
		t.Errorf("应返回 409 冲突: %v", err)
	}
	if kerrors.FromError(err).Reason != "USER_DUPLICATE" {
		t.Errorf("应返回 USER_DUPLICATE: %v", err)
	}
}

func TestUserRepo_FindByUsernameAndEmail(t *testing.T) {
	repo, _ := newTestRepo(t)
	ctx := context.Background()

	username := uniq("find")
	email := uniq("find") + "@example.com"
	u := makeUser(username, email)
	if err := repo.Create(ctx, u); err != nil {
		t.Fatalf("Create 失败: %v", err)
	}
	defer cleanupUser(t, repo, u.ID)

	byName, err := repo.FindByUsername(ctx, username)
	if err != nil {
		t.Fatalf("FindByUsername 失败: %v", err)
	}
	if byName.ID != u.ID {
		t.Errorf("按用户名查询 ID 不匹配: %d != %d", byName.ID, u.ID)
	}

	byEmail, err := repo.FindByEmail(ctx, email)
	if err != nil {
		t.Fatalf("FindByEmail 失败: %v", err)
	}
	if byEmail.ID != u.ID {
		t.Errorf("按邮箱查询 ID 不匹配: %d != %d", byEmail.ID, u.ID)
	}

	byAccount, err := repo.FindByAccount(ctx, username)
	if err != nil {
		t.Fatalf("FindByAccount(用户名) 失败: %v", err)
	}
	if byAccount.ID != u.ID {
		t.Errorf("FindByAccount 用户名 ID 不匹配")
	}
	byAccount2, err := repo.FindByAccount(ctx, email)
	if err != nil {
		t.Fatalf("FindByAccount(邮箱) 失败: %v", err)
	}
	if byAccount2.ID != u.ID {
		t.Errorf("FindByAccount 邮箱 ID 不匹配")
	}
}

func TestUserRepo_FindNotFound(t *testing.T) {
	repo, _ := newTestRepo(t)
	ctx := context.Background()

	if _, err := repo.FindByID(ctx, 99999999); err == nil {
		t.Error("不存在的 ID 应报错")
	}
	if _, err := repo.FindByUsername(ctx, uniq("ghost")); err == nil {
		t.Error("不存在的用户名应报错")
	}
}

func TestUserRepo_Update(t *testing.T) {
	repo, _ := newTestRepo(t)
	ctx := context.Background()

	u := makeUser(uniq("upd"), uniq("upd")+"@example.com")
	if err := repo.Create(ctx, u); err != nil {
		t.Fatalf("Create 失败: %v", err)
	}
	defer cleanupUser(t, repo, u.ID)

	u.Avatar = "https://avatar.example.com/a.png"
	u.Bio = "新的简介"
	if err := repo.Update(ctx, u); err != nil {
		t.Fatalf("Update 失败: %v", err)
	}

	got, _ := repo.FindByID(ctx, u.ID)
	if got.Avatar != u.Avatar || got.Bio != u.Bio {
		t.Errorf("资料未更新: %+v", got)
	}
}

func TestUserRepo_Delete(t *testing.T) {
	repo, _ := newTestRepo(t)
	ctx := context.Background()

	u := makeUser(uniq("del"), uniq("del")+"@example.com")
	if err := repo.Create(ctx, u); err != nil {
		t.Fatalf("Create 失败: %v", err)
	}
	if err := repo.Delete(ctx, u.ID); err != nil {
		t.Fatalf("Delete 失败: %v", err)
	}
	if _, err := repo.FindByID(ctx, u.ID); err == nil {
		t.Error("删除后不应能查到")
	}
}

func TestUserRepo_List(t *testing.T) {
	repo, _ := newTestRepo(t)
	ctx := context.Background()

	u1 := makeUser(uniq("list1"), uniq("list1")+"@example.com")
	u2 := makeUser(uniq("list2"), uniq("list2")+"@example.com")
	if err := repo.Create(ctx, u1); err != nil {
		t.Fatalf("Create u1 失败: %v", err)
	}
	defer cleanupUser(t, repo, u1.ID)
	if err := repo.Create(ctx, u2); err != nil {
		t.Fatalf("Create u2 失败: %v", err)
	}
	defer cleanupUser(t, repo, u2.ID)

	users, total, err := repo.List(ctx, 1, 10)
	if err != nil {
		t.Fatalf("List 失败: %v", err)
	}
	if total < 2 {
		t.Errorf("total 应 >= 2: %d", total)
	}
	if len(users) == 0 {
		t.Error("列表不应为空")
	}
}

func TestUserRepo_UpdateStatus(t *testing.T) {
	repo, _ := newTestRepo(t)
	ctx := context.Background()

	u := makeUser(uniq("status"), uniq("status")+"@example.com")
	if err := repo.Create(ctx, u); err != nil {
		t.Fatalf("Create 失败: %v", err)
	}
	defer cleanupUser(t, repo, u.ID)

	if err := repo.UpdateStatus(ctx, u.ID, biz.UserStatusDisabled); err != nil {
		t.Fatalf("UpdateStatus 失败: %v", err)
	}
	got, _ := repo.FindByID(ctx, u.ID)
	if got.Status != biz.UserStatusDisabled {
		t.Errorf("状态未更新: %v", got.Status)
	}
}

// 缓存行为：FindByID 二次命中缓存（同一实例下 cache 有值）
func TestUserRepo_CacheRoundTrip(t *testing.T) {
	repo, _ := newTestRepo(t)
	ctx := context.Background()

	u := makeUser(uniq("cache"), uniq("cache")+"@example.com")
	if err := repo.Create(ctx, u); err != nil {
		t.Fatalf("Create 失败: %v", err)
	}
	defer cleanupUser(t, repo, u.ID)

	// 第一次查询回写缓存
	if _, err := repo.FindByID(ctx, u.ID); err != nil {
		t.Fatalf("首次查询失败: %v", err)
	}
	// 第二次应命中缓存
	if _, err := repo.FindByID(ctx, u.ID); err != nil {
		t.Fatalf("二次查询失败: %v", err)
	}
}
