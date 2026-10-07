package sync

import (
	"sync"
	"time"
)

type Cache struct {
	mu      sync.Mutex
	entries map[string]time.Time
	limit   int
	ttl     time.Duration
}

func NewCache(limit int, ttl time.Duration) *Cache {
	if limit < 1 || ttl <= 0 {
		panic("invalid cache bounds")
	}
	return &Cache{entries: make(map[string]time.Time), limit: limit, ttl: ttl}
}
func (c *Cache) Seen(id string, now time.Time) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	t, ok := c.entries[id]
	return ok && now.Before(t.Add(c.ttl))
}
func (c *Cache) Add(id string, now time.Time) {
	c.mu.Lock()
	defer c.mu.Unlock()
	for k, t := range c.entries {
		if !now.Before(t.Add(c.ttl)) {
			delete(c.entries, k)
		}
	}
	if len(c.entries) >= c.limit {
		var oldest string
		var at time.Time
		for k, t := range c.entries {
			if oldest == "" || t.Before(at) {
				oldest = k
				at = t
			}
		}
		delete(c.entries, oldest)
	}
	c.entries[id] = now
}
