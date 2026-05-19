package cache

import (
	"sync"
	"time"
)

type MemoryCache struct {
	mu      sync.RWMutex
	entries map[string]entry
}

type entry struct {
	value     any
	expiresAt time.Time
}

func NewMemoryCache() *MemoryCache {
	return &MemoryCache{entries: make(map[string]entry)}
}

func (c *MemoryCache) Get(key string) (any, bool) {
	c.mu.RLock()
	e, ok := c.entries[key]
	c.mu.RUnlock()
	if !ok {
		return nil, false
	}
	if !e.expiresAt.IsZero() && time.Now().After(e.expiresAt) {
		c.mu.Lock()
		delete(c.entries, key)
		c.mu.Unlock()
		return nil, false
	}
	return e.value, true
}

func (c *MemoryCache) Set(key string, value any, ttl time.Duration) {
	expiresAt := time.Time{}
	if ttl > 0 {
		expiresAt = time.Now().Add(ttl)
	}
	c.mu.Lock()
	c.entries[key] = entry{value: value, expiresAt: expiresAt}
	c.mu.Unlock()
}

func (c *MemoryCache) Clear() {
	c.mu.Lock()
	c.entries = make(map[string]entry)
	c.mu.Unlock()
}

func (c *MemoryCache) Len() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.entries)
}
