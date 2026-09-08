package biz

import (
	"context"
	"sync"
	"time"

	"github.com/CycleZero/ley/pkg/eventbus"
	"github.com/CycleZero/ley/pkg/jwt"
	mqPkg "github.com/CycleZero/ley/pkg/mq"
	"github.com/CycleZero/ley/pkg/testutil/datatest"
	"github.com/go-kratos/kratos/v2/log"
)

var _ UserRepo = (*mockUserRepo)(nil)
var _ eventbus.EventBus = (*mockEventBus)(nil)

// =============================================================================
// Mock UserRepo — 内存实现
// =============================================================================

type mockUserRepo struct {
	mu         sync.Mutex
	users      map[uint]*User
	usernameIX map[string]uint
	emailIX    map[string]uint
	nextID     uint
	stale      *User // 测试用：非 nil 时 FindByID 返回该陈旧快照（模拟缓存未失效）
}

func newMockUserRepo() *mockUserRepo {
	return &mockUserRepo{
		users:      make(map[uint]*User),
		usernameIX: make(map[string]uint),
		emailIX:    make(map[string]uint),
		nextID:     1,
	}
}

// Seed 直接插入用户（跳过哈希，测试用）
func (m *mockUserRepo) Seed(u *User) *User {
	m.mu.Lock()
	defer m.mu.Unlock()
	u.ID = m.nextID
	m.nextID++
	now := time.Now()
	u.CreatedAt, u.UpdatedAt = now, now
	cp := *u
	m.users[u.ID] = &cp
	m.usernameIX[u.Username] = u.ID
	m.emailIX[u.Email] = u.ID
	return &cp
}

func (m *mockUserRepo) Create(ctx context.Context, u *User) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.usernameIX[u.Username]; ok {
		return ErrUsernameTaken
	}
	if _, ok := m.emailIX[u.Email]; ok {
		return ErrEmailTaken
	}
	u.ID = m.nextID
	m.nextID++
	now := time.Now()
	u.CreatedAt, u.UpdatedAt = now, now
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
		return ErrUserNotFound
	}
	u.Avatar = avatar
	u.Bio = bio
	u.UpdatedAt = time.Now()
	return nil
}

func (m *mockUserRepo) Delete(ctx context.Context, id uint) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	u, ok := m.users[id]
	if !ok {
		return ErrUserNotFound
	}
	delete(m.users, id)
	delete(m.usernameIX, u.Username)
	delete(m.emailIX, u.Email)
	return nil
}

func (m *mockUserRepo) FindByID(ctx context.Context, id uint) (*User, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.stale != nil {
		cp := *m.stale
		return &cp, nil
	}
	u, ok := m.users[id]
	if !ok {
		return nil, ErrUserNotFound
	}
	cp := *u
	return &cp, nil
}

func (m *mockUserRepo) FindByUsername(ctx context.Context, username string) (*User, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	id, ok := m.usernameIX[username]
	if !ok {
		return nil, ErrUserNotFound
	}
	cp := *m.users[id]
	return &cp, nil
}

func (m *mockUserRepo) FindByEmail(ctx context.Context, email string) (*User, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	id, ok := m.emailIX[email]
	if !ok {
		return nil, ErrUserNotFound
	}
	cp := *m.users[id]
	return &cp, nil
}

func (m *mockUserRepo) FindByAccount(ctx context.Context, account string) (*User, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if id, ok := m.usernameIX[account]; ok {
		cp := *m.users[id]
		return &cp, nil
	}
	if id, ok := m.emailIX[account]; ok {
		cp := *m.users[id]
		return &cp, nil
	}
	return nil, ErrUserNotFound
}

func (m *mockUserRepo) List(ctx context.Context, page, pageSize int) ([]*User, int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]*User, 0, len(m.users))
	for _, u := range m.users {
		cp := *u
		out = append(out, &cp)
	}
	return out, int64(len(out)), nil
}

func (m *mockUserRepo) UpdateStatus(ctx context.Context, id uint, status UserStatus) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	u, ok := m.users[id]
	if !ok {
		return ErrUserNotFound
	}
	u.Status = status
	return nil
}

// =============================================================================
// Mock EventBus
// =============================================================================

type mockEventBus struct {
	mu     sync.Mutex
	topics []string
}

func (m *mockEventBus) PublishAsync(ctx context.Context, topic string, event interface{}) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.topics = append(m.topics, topic)
	return nil
}

func (m *mockEventBus) PublishSync(ctx context.Context, topic string, event interface{}, timeout time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.topics = append(m.topics, topic)
	return nil
}

func (m *mockEventBus) Subscribe(ctx context.Context, topic string, handler eventbus.EventHandler, _ ...mqPkg.ConsumerOption) (eventbus.Subscription, error) {
	return nil, nil
}

func (m *mockEventBus) Close() error { return nil }

func (m *mockEventBus) PublishedTopics() []string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]string{}, m.topics...)
}

// =============================================================================
// Setup 辅助
// =============================================================================

func newTestJWT() jwt.JWT {
	return jwt.NewJWT(&jwt.Config{
		SigningKey:  "test-signing-key-0123456789-256bit-random",
		ExpiredTime: time.Hour,
		Issuer:      "test-issuer",
	})
}

// setupAuthUseCase 返回 (useCase, repo, eventBus, blacklist)
func setupAuthUseCase() (*AuthUseCase, *mockUserRepo, *mockEventBus, jwt.BlackListCache) {
	repo := newMockUserRepo()
	eb := &mockEventBus{}
	bl := jwt.NewBlackList(datatest.NewInMemoryCache())
	uc := NewAuthUseCase(repo, newTestJWT(), bl, eb, log.DefaultLogger)
	return uc, repo, eb, bl
}

func setupUserUseCase() (*UserUseCase, *mockUserRepo) {
	repo := newMockUserRepo()
	uc := NewUserUseCase(repo, log.DefaultLogger)
	return uc, repo
}
