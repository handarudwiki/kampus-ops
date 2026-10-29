// Package redis provides the Redis client plus a small KV abstraction used for
// idempotency keys (EC-02) and caching, with an in-memory fallback for tests.
package redis

import (
	"context"
	"sync"
	"time"

	goredis "github.com/redis/go-redis/v9"
)

// KV is the minimal store contract used by the idempotency middleware and caches.
type KV interface {
	// SetNX stores value if key is absent; returns true when stored.
	SetNX(ctx context.Context, key string, value []byte, ttl time.Duration) (bool, error)
	Set(ctx context.Context, key string, value []byte, ttl time.Duration) error
	Get(ctx context.Context, key string) ([]byte, bool, error)
	Del(ctx context.Context, key string) error
	Ping(ctx context.Context) error
}

func Connect(ctx context.Context, url string) (*goredis.Client, error) {
	opt, err := goredis.ParseURL(url)
	if err != nil {
		return nil, err
	}
	c := goredis.NewClient(opt)
	if err := c.Ping(ctx).Err(); err != nil {
		return c, err
	}
	return c, nil
}

type redisKV struct{ c *goredis.Client }

func NewKV(c *goredis.Client) KV { return &redisKV{c: c} }

func (r *redisKV) SetNX(ctx context.Context, k string, v []byte, ttl time.Duration) (bool, error) {
	return r.c.SetNX(ctx, k, v, ttl).Result()
}
func (r *redisKV) Set(ctx context.Context, k string, v []byte, ttl time.Duration) error {
	return r.c.Set(ctx, k, v, ttl).Err()
}
func (r *redisKV) Get(ctx context.Context, k string) ([]byte, bool, error) {
	b, err := r.c.Get(ctx, k).Bytes()
	if err == goredis.Nil {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	return b, true, nil
}
func (r *redisKV) Del(ctx context.Context, k string) error { return r.c.Del(ctx, k).Err() }
func (r *redisKV) Ping(ctx context.Context) error          { return r.c.Ping(ctx).Err() }

// MemoryKV is a process-local KV used in tests and when Redis is unavailable.
type MemoryKV struct {
	mu sync.Mutex
	m  map[string]memItem
}

type memItem struct {
	v   []byte
	exp time.Time
}

func NewMemoryKV() *MemoryKV { return &MemoryKV{m: map[string]memItem{}} }

func (m *MemoryKV) get(k string) (memItem, bool) {
	it, ok := m.m[k]
	if ok && time.Now().After(it.exp) {
		delete(m.m, k)
		return memItem{}, false
	}
	return it, ok
}

func (m *MemoryKV) SetNX(_ context.Context, k string, v []byte, ttl time.Duration) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.get(k); ok {
		return false, nil
	}
	m.m[k] = memItem{v: v, exp: time.Now().Add(ttl)}
	return true, nil
}
func (m *MemoryKV) Set(_ context.Context, k string, v []byte, ttl time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.m[k] = memItem{v: v, exp: time.Now().Add(ttl)}
	return nil
}
func (m *MemoryKV) Get(_ context.Context, k string) ([]byte, bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	it, ok := m.get(k)
	return it.v, ok, nil
}
func (m *MemoryKV) Del(_ context.Context, k string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.m, k)
	return nil
}
func (m *MemoryKV) Ping(context.Context) error { return nil }
