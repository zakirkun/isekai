package cache

import (
	"context"
	"sync"
	"time"

	"github.com/zakirkun/isekai/pkg/config"
	"github.com/zakirkun/isekai/pkg/logger"
)

// Item represents a cached item
type Item struct {
	Value      interface{}
	Expiration int64
}

// Cache represents an in-memory cache
type Cache struct {
	mu              sync.RWMutex
	items           map[string]*Item
	cleanupInterval time.Duration
	defaultTTL      time.Duration
	maxSize         int64
	log             *logger.Logger
	stopCleanup     chan bool
}

// New creates a new cache instance
func New(cfg *config.CacheConfig, log *logger.Logger) *Cache {
	c := &Cache{
		items:           make(map[string]*Item),
		cleanupInterval: cfg.CleanupInterval,
		defaultTTL:      cfg.TTL,
		maxSize:         cfg.MaxSize,
		log:             log,
		stopCleanup:     make(chan bool),
	}

	if cfg.Enabled {
		go c.startCleanup()
		log.Info("Cache initialized and cleanup started")
	}

	return c
}

// Set adds an item to the cache with default TTL
func (c *Cache) Set(key string, value interface{}) {
	c.SetWithTTL(key, value, c.defaultTTL)
}

// SetWithTTL adds an item to the cache with custom TTL
func (c *Cache) SetWithTTL(key string, value interface{}, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Check if we need to evict items
	if int64(len(c.items)) >= c.maxSize {
		c.evictOldest()
	}

	expiration := time.Now().Add(ttl).UnixNano()
	c.items[key] = &Item{
		Value:      value,
		Expiration: expiration,
	}
}

// Get retrieves an item from the cache
func (c *Cache) Get(key string) (interface{}, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	item, exists := c.items[key]
	if !exists {
		return nil, false
	}

	// Check if item has expired
	if time.Now().UnixNano() > item.Expiration {
		return nil, false
	}

	return item.Value, true
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
	c.items = make(map[string]*Item)
	c.log.Info("Cache cleared")
}

// Size returns the number of items in the cache
func (c *Cache) Size() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.items)
}

// startCleanup starts the cleanup goroutine
func (c *Cache) startCleanup() {
	ticker := time.NewTicker(c.cleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			c.deleteExpired()
		case <-c.stopCleanup:
			return
		}
	}
}

// deleteExpired removes expired items from the cache
func (c *Cache) deleteExpired() {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now().UnixNano()
	count := 0

	for key, item := range c.items {
		if now > item.Expiration {
			delete(c.items, key)
			count++
		}
	}

	if count > 0 {
		c.log.Debugf("Cleaned up %d expired cache items", count)
	}
}

// evictOldest removes the oldest item from the cache
func (c *Cache) evictOldest() {
	var oldestKey string
	var oldestTime int64 = 1<<63 - 1

	for key, item := range c.items {
		if item.Expiration < oldestTime {
			oldestTime = item.Expiration
			oldestKey = key
		}
	}

	if oldestKey != "" {
		delete(c.items, oldestKey)
		c.log.Debugf("Evicted oldest cache item: %s", oldestKey)
	}
}

// Stop stops the cleanup goroutine
func (c *Cache) Stop() {
	close(c.stopCleanup)
	c.log.Info("Cache cleanup stopped")
}

// Health checks cache health
func (c *Cache) Health(ctx context.Context) error {
	// Simple health check - just verify we can access the cache
	c.mu.RLock()
	defer c.mu.RUnlock()
	return nil
}
