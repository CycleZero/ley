package service

import (
	"context"
	"sync"
	"testing"
	"time"

	authv1 "github.com/CycleZero/ley/api/auth/v1"
	"github.com/CycleZero/ley/app/auth/internal/biz"
	"github.com/CycleZero/ley/pkg/eventbus"
	"github.com/CycleZero/ley/pkg/jwt"
	"github.com/CycleZero/ley/pkg/meta"
	mqPkg "github.com/CycleZero/ley/pkg/mq"
	"github.com/CycleZero/ley/pkg/security"
	"github.com/CycleZero/ley/pkg/testutil/datatest"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/transport"
)

// =============================================================================
// Mock（service 包无法复用 biz 包内部 mock，这里按接口实现最小版本）
// =============================================================================

type mockUserRepo struct {
	mu         sync.Mutex
	users      map[uint]*biz.User
	usernameIX map[string]uint
	emailIX    map[string]uint
	nextID     uint
}

func newMockUserRepo() *mockUserRepo {
	return &mockUserRepo{
		users:      make(map[uint]*biz.User),
		usernameIX: make(map[string]uint),
		emailIX:    make(map[string]uint),
		nextID:     1,
	}
}

func (m *mockUserRepo) seed(u *biz.User) *biz.User {
	m.mu.Lock()
	defer m.mu.Unlock()
	u.ID = m.nextID
	m.nextID++
	u.CreatedAt, u.UpdatedAt = time.Now(), time.Now()
	cp := *u
	m.users[u.ID] = &cp
	m.usernameIX[u.Username] = u.ID
	m.emailIX[u.Email] = u.ID
	return &cp
}

func (m *mockUserRepo) Create(ctx context.Context, u *biz.User) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.usernameIX[u.Username]; ok {
		return biz.ErrUsernameTaken
	}
	u.ID = m.nextID
	m.nextID++
	u.CreatedAt, u.UpdatedAt = time.Now(), time.Now()
	cp := *u
	m.users[u.ID] = &cp
	m.usernameIX[u.Username] = u.ID
	m.emailIX[u.Email] = u.ID
	return nil
}

func (m *mockUserRepo) UpdateProfile(ctx context.Context, id uint, avatar, bio string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	u, ok := m.users[id]
	if !ok {
		return biz.ErrUserNotFound
	}
	u.Avatar = avatar
	u.Bio = bio
	u.UpdatedAt = time.Now()
	return nil
}

func (m *mockUserRepo) Delete(ctx context.Context, id uint) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.users, id)
	return nil
}

func (m *mockUserRepo) FindByID(ctx context.Context, id uint) (*biz.User, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	u, ok := m.users[id]
	if !ok {
		return nil, biz.ErrUserNotFound
	}
	cp := *u
	return &cp, nil
}

func (m *mockUserRepo) findByKey(ctx context.Context, key string) (*biz.User, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if id, ok := m.usernameIX[key]; ok {
		cp := *m.users[id]
		return &cp, nil
	}
	if id, ok := m.emailIX[key]; ok {
		cp := *m.users[id]
		return &cp, nil
	}
	return nil, biz.ErrUserNotFound
}

func (m *mockUserRepo) FindByUsername(ctx context.Context, username string) (*biz.User, error) {
	return m.findByKey(ctx, username)
}

func (m *mockUserRepo) FindByEmail(ctx context.Context, email string) (*biz.User, error) {
	return m.findByKey(ctx, email)
}

func (m *mockUserRepo) FindByAccount(ctx context.Context, account string) (*biz.User, error) {
	return m.findByKey(ctx, account)
}

func (m *mockUserRepo) List(ctx context.Context, page, pageSize int) ([]*biz.User, int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]*biz.User, 0, len(m.users))
	for _, u := range m.users {
		cp := *u
		out = append(out, &cp)
	}
	return out, int64(len(out)), nil
}

func (m *mockUserRepo) UpdateStatus(ctx context.Context, id uint, status biz.UserStatus) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if u, ok := m.users[id]; ok {
		u.Status = status
	}
	return nil
}

type mockEventBus struct{}

func (m *mockEventBus) PublishAsync(ctx context.Context, topic string, event interface{}) error { return nil }
func (m *mockEventBus) PublishSync(ctx context.Context, topic string, event interface{}, timeout time.Duration) error {
	return nil
}
func (m *mockEventBus) Subscribe(ctx context.Context, topic string, handler eventbus.EventHandler, _ ...mqPkg.ConsumerOption) (eventbus.Subscription, error) {
	return nil, nil
}
func (m *mockEventBus) Close() error { return nil }

// =============================================================================
// Setup
// =============================================================================

func newTestService(t *testing.T) (*AuthService, *mockUserRepo) {
	t.Helper()
	repo := newMockUserRepo()
	j := jwt.NewJWT(&jwt.Config{
		SigningKey:  "test-signing-key-0123456789-256bit-random",
		ExpiredTime: time.Hour,
		Issuer:      "test",
	})
	bl := jwt.NewBlackList(datatest.NewInMemoryCache())
	authUC := biz.NewAuthUseCase(repo, j, bl, &mockEventBus{}, log.DefaultLogger)
	userUC := biz.NewUserUseCase(repo, log.DefaultLogger)
	return NewAuthService(authUC, userUC, log.DefaultLogger), repo
}

// ctxWithUser 构造带用户身份的上下文（模拟网关透传）
func ctxWithUser(userID uint) context.Context {
	return meta.NewClientCtx(context.Background(), &meta.RequestMetaData{
		Auth: meta.Auth{UserID: uint64(userID), UserName: "tester"},
	})
}

// =============================================================================
// Register handler
// =============================================================================

func TestServiceRegister(t *testing.T) {
	svc, _ := newTestService(t)
	resp, err := svc.Register(context.Background(), &authv1.RegisterRequest{
		Username: "tester",
		Email:    "t@example.com",
		Password: "Password123",
	})
	if err != nil {
		t.Fatalf("Register 失败: %v", err)
	}
	if resp.User.Username != "tester" {
		t.Errorf("UserInfo 不匹配: %+v", resp.User)
	}
	if resp.TokenPair.AccessToken == "" || resp.TokenPair.RefreshToken == "" {
		t.Error("应返回 token pair")
	}
	if resp.TokenPair.ExpiresIn != 3600 {
		t.Errorf("ExpiresIn 应与 access TTL(1h) 一致: got %d want 3600", resp.TokenPair.ExpiresIn)
	}
}

func TestServiceRegisterValidationError(t *testing.T) {
	svc, _ := newTestService(t)
	_, err := svc.Register(context.Background(), &authv1.RegisterRequest{
		Username: "ab", // 太短
		Email:    "t@example.com",
		Password: "Password123",
	})
	if err == nil {
		t.Fatal("非法用户名应报错")
	}
}

// =============================================================================
// Login handler
// =============================================================================

func TestServiceLogin(t *testing.T) {
	svc, repo := newTestService(t)
	hash, _ := security.HashPassword("Password123")
	repo.seed(&biz.User{Username: "tester", Email: "t@example.com", Password: hash, Role: biz.RoleReader, Status: biz.UserStatusActive})

	resp, err := svc.Login(context.Background(), &authv1.LoginRequest{Account: "tester", Password: "Password123"})
	if err != nil {
		t.Fatalf("Login 失败: %v", err)
	}
	if resp.User.Id != 1 {
		t.Errorf("用户 ID 不匹配: %d", resp.User.Id)
	}
	if resp.User.Role != "reader" {
		t.Errorf("角色不匹配: %s", resp.User.Role)
	}
}

func TestServiceLoginBadCredentials(t *testing.T) {
	svc, repo := newTestService(t)
	hash, _ := security.HashPassword("Password123")
	repo.seed(&biz.User{Username: "tester", Email: "t@example.com", Password: hash, Role: biz.RoleReader, Status: biz.UserStatusActive})

	_, err := svc.Login(context.Background(), &authv1.LoginRequest{Account: "tester", Password: "Wrong1"})
	if err == nil {
		t.Fatal("错误密码应报错")
	}
}

// =============================================================================
// RefreshToken handler
// =============================================================================

func TestServiceRefreshToken(t *testing.T) {
	svc, repo := newTestService(t)
	hash, _ := security.HashPassword("Password123")
	repo.seed(&biz.User{Username: "tester", Email: "t@example.com", Password: hash, Role: biz.RoleReader, Status: biz.UserStatusActive})

	loginResp, err := svc.Login(context.Background(), &authv1.LoginRequest{Account: "tester", Password: "Password123"})
	if err != nil {
		t.Fatalf("Login 失败: %v", err)
	}

	resp, err := svc.RefreshToken(context.Background(), &authv1.RefreshTokenRequest{
		RefreshToken: loginResp.TokenPair.RefreshToken,
	})
	if err != nil {
		t.Fatalf("RefreshToken 失败: %v", err)
	}
	if resp.TokenPair.AccessToken == loginResp.TokenPair.AccessToken {
		t.Error("access token 应轮换")
	}
}

// =============================================================================
// GetProfile / UpdateProfile handler（依赖 meta 注入用户 ID）
// =============================================================================

func TestServiceGetProfile(t *testing.T) {
	svc, repo := newTestService(t)
	repo.seed(&biz.User{Username: "tester", Email: "t@example.com", Password: "h", Role: biz.RoleReader, Status: biz.UserStatusActive})

	resp, err := svc.GetProfile(ctxWithUser(1), &authv1.GetProfileRequest{})
	if err != nil {
		t.Fatalf("GetProfile 失败: %v", err)
	}
	if resp.User.Username != "tester" || resp.User.Id != 1 {
		t.Errorf("Profile 不匹配: %+v", resp.User)
	}
}

func TestServiceGetProfileWithoutUser(t *testing.T) {
	svc, _ := newTestService(t)
	_, err := svc.GetProfile(context.Background(), &authv1.GetProfileRequest{})
	if err == nil {
		t.Fatal("无用户上下文应报错")
	}
}

func TestServiceUpdateProfile(t *testing.T) {
	svc, repo := newTestService(t)
	repo.seed(&biz.User{Username: "tester", Email: "t@example.com", Password: "h", Role: biz.RoleReader, Status: biz.UserStatusActive})

	resp, err := svc.UpdateProfile(ctxWithUser(1), &authv1.UpdateProfileRequest{
		Avatar: "https://avatar.png",
		Bio:    "新的简介",
	})
	if err != nil {
		t.Fatalf("UpdateProfile 失败: %v", err)
	}
	if resp.User.Bio != "新的简介" {
		t.Errorf("Bio 未更新: %+v", resp.User)
	}
}

// =============================================================================
// Logout handler（从 transport header 提取 access token）
// =============================================================================

type fakeHeader struct {
	h map[string]string
}

func (f *fakeHeader) Get(key string) string { return f.h[key] }
func (f *fakeHeader) Set(key, value string) { f.h[key] = value }
func (f *fakeHeader) Add(key, value string) { f.h[key] = value }
func (f *fakeHeader) Values(key string) []string {
	if v, ok := f.h[key]; ok {
		return []string{v}
	}
	return nil
}
func (f *fakeHeader) Keys() []string {
	keys := make([]string, 0, len(f.h))
	for k := range f.h {
		keys = append(keys, k)
	}
	return keys
}

type fakeTransporter struct {
	kind   transport.Kind
	reqH   transport.Header
	replyH transport.Header
}

func (t *fakeTransporter) Kind() transport.Kind          { return t.kind }
func (t *fakeTransporter) Endpoint() string              { return "" }
func (t *fakeTransporter) Operation() string             { return "" }
func (t *fakeTransporter) RequestHeader() transport.Header { return t.reqH }
func (t *fakeTransporter) ReplyHeader() transport.Header   { return t.replyH }

func TestServiceLogout(t *testing.T) {
	svc, repo := newTestService(t)
	hash, _ := security.HashPassword("Password123")
	repo.seed(&biz.User{Username: "tester", Email: "t@example.com", Password: hash, Role: biz.RoleReader, Status: biz.UserStatusActive})

	loginResp, err := svc.Login(context.Background(), &authv1.LoginRequest{Account: "tester", Password: "Password123"})
	if err != nil {
		t.Fatalf("Login 失败: %v", err)
	}

	_, err = svc.Logout(context.Background(), &authv1.LogoutRequest{
		RefreshToken: loginResp.TokenPair.RefreshToken,
	})
	if err != nil {
		t.Fatalf("Logout 失败: %v", err)
	}
}

// FIX-1: entry 经 gRPC 转发时不携带原始 Authorization 头，access token 只能经
// pkg/meta 透传；Logout 必须据此吊销 access token，否则登出后 15 分钟内仍可用。
func TestServiceLogoutBlacklistsAccessTokenFromMeta(t *testing.T) {
	repo := newMockUserRepo()
	j := jwt.NewJWT(&jwt.Config{
		SigningKey:  "test-signing-key-0123456789-256bit-random",
		ExpiredTime: time.Hour,
		Issuer:      "test",
	})
	bl := jwt.NewBlackList(datatest.NewInMemoryCache())
	authUC := biz.NewAuthUseCase(repo, j, bl, &mockEventBus{}, log.DefaultLogger)
	svc := NewAuthService(authUC, biz.NewUserUseCase(repo, log.DefaultLogger), log.DefaultLogger)

	hash, _ := security.HashPassword("Password123")
	repo.seed(&biz.User{Username: "tester", Email: "t@example.com", Password: hash, Role: biz.RoleReader, Status: biz.UserStatusActive})
	loginResp, err := svc.Login(context.Background(), &authv1.LoginRequest{Account: "tester", Password: "Password123"})
	if err != nil {
		t.Fatalf("Login 失败: %v", err)
	}

	// 模拟 entry：无 transport Authorization 头，仅经 pkg/meta 透传 access token
	ctx := meta.NewClientCtx(context.Background(), &meta.RequestMetaData{
		Auth:        meta.Auth{UserID: 1, UserName: "tester"},
		AccessToken: loginResp.TokenPair.AccessToken,
	})
	if _, err := svc.Logout(ctx, &authv1.LogoutRequest{RefreshToken: loginResp.TokenPair.RefreshToken}); err != nil {
		t.Fatalf("Logout 失败: %v", err)
	}

	if !bl.IsTokenBlackListed(loginResp.TokenPair.AccessToken) {
		t.Error("登出后 access token 应进入黑名单（entry gRPC 路径）")
	}
	if !bl.IsTokenBlackListed(loginResp.TokenPair.RefreshToken) {
		t.Error("登出后 refresh token 应进入黑名单")
	}
}
