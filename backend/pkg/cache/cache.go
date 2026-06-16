package cache

import (
	"sync"
	"time"
)

// Cache represents a simple in-memory cache
type Cache struct {
	mu       sync.RWMutex
	items    map[string]cacheItem
	stopCh   chan struct{}
}

type cacheItem struct {
	value     interface{}
	expiresAt time.Time
}

// NewCache creates a new cache instance
func NewCache() *Cache {
	return &Cache{
		items:  make(map[string]cacheItem),
		stopCh: make(chan struct{}),
	}
}

// Set adds an item to the cache
func (c *Cache) Set(key string, value interface{}, duration time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.items[key] = cacheItem{
		value:     value,
		expiresAt: time.Now().Add(duration),
	}
}

// Get retrieves an item from the cache
func (c *Cache) Get(key string) (interface{}, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	item, exists := c.items[key]
	if !exists || time.Now().After(item.expiresAt) {
		return nil, false
	}

	return item.value, true
}

// Delete removes an item from the cache
func (c *Cache) Delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.items, key)
}

// Clear removes all items from the cache
func (c *Cache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.items = make(map[string]cacheItem)
}

// StartCleanup starts periodic cleanup of expired items
func (c *Cache) StartCleanup(interval time.Duration) {
	ticker := time.NewTicker(interval)
	go func() {
		for {
			select {
			case <-ticker.C:
				c.cleanup()
			case <-c.stopCh:
				ticker.Stop()
				return
			}
		}
	}()
}

// Stop stops the cleanup goroutine
func (c *Cache) Stop() {
	close(c.stopCh)
}

func (c *Cache) cleanup() {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	for key, item := range c.items {
		if now.After(item.expiresAt) {
			delete(c.items, key)
		}
	}
}
