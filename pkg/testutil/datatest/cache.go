// Package datatest provides test utilities for data layer unit tests.
// Contains an in-memory Cache implementation with call tracking,
// enabling tests to verify cache interactions without a real Redis.
package datatest

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"time"

	"ley/pkg/cache"
)

// CacheOp records a cache operation for assertion
type CacheOp int

const (
	OpGet    CacheOp = iota
	OpSet
	OpDelete
	OpExists
	OpFlush
)

func (o CacheOp) String() string {
	switch o {
	case OpGet:
		return "Get"
	case OpSet:
		return "Set"
	case OpDelete:
		return "Delete"
	case OpExists:
		return "Exists"
	case OpFlush:
		return "Flush"
	default:
		return "Unknown"
	}
}

// InMemoryCache is a thread-safe in-memory implementation of cache.Cache.
// Records every Get/Set/Delete call for test assertions.
// Preconfigured data and errors enable deterministic test scenarios.
type InMemoryCache struct {
	mu    sync.RWMutex
	store map[string][]byte

	// Preconfigured errors per key for Get operations
	getErrors map[string]error

	// Call tracking
	GetCalls    map[string]int
	SetCalls    map[string]int
	DeleteCalls map[string]int
}

// NewInMemoryCache creates a cache with pre-seeded data.
// data: initial key-value pairs (JSON-serializable values).
func NewInMemoryCache() *InMemoryCache {
	return &InMemoryCache{
		store:      make(map[string][]byte),
		getErrors:  make(map[string]error),
		GetCalls:   make(map[string]int),
		SetCalls:   make(map[string]int),
		DeleteCalls: make(map[string]int),
	}
}

// Seed stores raw bytes for a key, used for pre-seeding cache content.
func (c *InMemoryCache) Seed(key string, value interface{}) {
	c.mu.Lock()
	defer c.mu.Unlock()

	switch v := value.(type) {
	case []byte:
		c.store[key] = v
	case string:
		c.store[key] = []byte(v)
	default:
		data, err := json.Marshal(v)
		if err != nil {
			panic("datatest: failed to marshal seed value: " + err.Error())
		}
		c.store[key] = data
	}
}

// SetGetError configures a specific error to return for Get(key).
// Use cache.ErrKeyNotFound or custom errors to simulate failure scenarios.
func (c *InMemoryCache) SetGetError(key string, err error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.getErrors[key] = err
}

// Get retrieves the cached value, respecting preconfigured errors.
func (c *InMemoryCache) Get(ctx context.Context, key string) ([]byte, error) {
	c.mu.Lock()
	c.GetCalls[key]++
	c.mu.Unlock()

	c.mu.RLock()
	defer c.mu.RUnlock()

	if err, ok := c.getErrors[key]; ok {
		return nil, err
	}
	data, ok := c.store[key]
	if !ok {
		return nil, cache.ErrKeyNotFound
	}
	return data, nil
}

// GetObject deserializes the cached value into the given pointer.
func (c *InMemoryCache) GetObject(ctx context.Context, key string, value interface{}) error {
	data, err := c.Get(ctx, key)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, value)
}

// Set stores a value with optional expiration (expiration is ignored in-memory).
func (c *InMemoryCache) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	c.mu.Lock()
	c.SetCalls[key]++
	c.mu.Unlock()

	c.mu.Lock()
	defer c.mu.Unlock()

	switch v := value.(type) {
	case []byte:
		c.store[key] = v
	case string:
		c.store[key] = []byte(v)
	default:
		data, err := json.Marshal(v)
		if err != nil {
			return err
		}
		c.store[key] = data
	}
	return nil
}

// Delete removes a key from the cache.
func (c *InMemoryCache) Delete(ctx context.Context, key string) error {
	c.mu.Lock()
	c.DeleteCalls[key]++
	c.mu.Unlock()

	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.store, key)
	return nil
}

// Exists checks if a key exists in the cache store.
func (c *InMemoryCache) Exists(ctx context.Context, key string) (bool, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if err, ok := c.getErrors[key]; ok && !errors.Is(err, cache.ErrKeyNotFound) {
		return false, err
	}
	_, ok := c.store[key]
	return ok, nil
}

// TTL returns the remaining time (always -1 = permanent in-memory).
func (c *InMemoryCache) TTL(ctx context.Context, key string) (time.Duration, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if _, ok := c.getErrors[key]; ok {
		return -2, nil
	}
	if _, ok := c.store[key]; ok {
		return -1, nil
	}
	return -2, nil
}

// Flush removes all keys from the cache.
func (c *InMemoryCache) Flush(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.store = make(map[string][]byte)
	return nil
}

// Close is a no-op.
func (c *InMemoryCache) Close() error {
	return nil
}

// Reset clears all data, errors, and call counters.
func (c *InMemoryCache) Reset() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.store = make(map[string][]byte)
	c.getErrors = make(map[string]error)
	c.GetCalls = make(map[string]int)
	c.SetCalls = make(map[string]int)
	c.DeleteCalls = make(map[string]int)
}

// WasDeleted returns true if the key was deleted at least once.
func (c *InMemoryCache) WasDeleted(key string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.DeleteCalls[key] > 0
}

// DeleteCount returns the number of times a key was deleted.
func (c *InMemoryCache) DeleteCount(key string) int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.DeleteCalls[key]
}

// GetCount returns the number of times a key was read.
func (c *InMemoryCache) GetCount(key string) int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.GetCalls[key]
}
