package cache

import (
	"sync"
	"time"
)

// Cache is a tiny in-memory TTL cache for lightweight JSON responses.
type Cache[T any] struct {
	mu   sync.RWMutex
	data map[string]entry[T]
}

type entry[T any] struct {
	value T
	exp   time.Time
}

// New returns an empty cache instance.
func New[T any]() *Cache[T] {
	return &Cache[T]{data: make(map[string]entry[T])}
}

// Get returns the cached value or false if absent/expired.
func (c *Cache[T]) Get(key string) (T, bool) {
	var zero T

	c.mu.RLock()
	item, ok := c.data[key]
	c.mu.RUnlock()
	if !ok || time.Now().After(item.exp) {
		return zero, false
	}
	return item.value, true
}

// Set stores a value with the provided TTL.
func (c *Cache[T]) Set(key string, value T, ttl time.Duration) {
	c.mu.Lock()
	c.data[key] = entry[T]{value: value, exp: time.Now().Add(ttl)}
	c.mu.Unlock()
}
