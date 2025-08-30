package utils

import (
	"os"
	"sync"
	"time"
)

// CacheItem represents a cached item with metadata for invalidation
type CacheItem[T any] struct {
	Value   T
	ModTime time.Time
	Size    int64
}

// Cache provides a generic caching utility with file-based invalidation
type Cache[K comparable, V any] struct {
	items map[K]*CacheItem[V]
	mutex sync.RWMutex
}

// NewCache creates a new generic cache
func NewCache[K comparable, V any]() *Cache[K, V] {
	return &Cache[K, V]{
		items: make(map[K]*CacheItem[V]),
	}
}

// Get retrieves an item from the cache
func (c *Cache[K, V]) Get(key K) (V, bool) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	if item, exists := c.items[key]; exists {
		return item.Value, true
	}

	var zero V
	return zero, false
}

// GetWithFileValidation retrieves an item from the cache with file-based validation
// If the file has been modified since caching, the item is removed and false is returned
func (c *Cache[K, V]) GetWithFileValidation(key K, filePath string) (V, bool) {
	c.mutex.RLock()
	item, exists := c.items[key]
	c.mutex.RUnlock()

	if !exists {
		var zero V
		return zero, false
	}

	// Check if file has been modified
	if stat, err := os.Stat(filePath); err == nil {
		if stat.ModTime().Equal(item.ModTime) && stat.Size() == item.Size {
			return item.Value, true
		}
	}

	// File changed or error, remove from cache
	c.mutex.Lock()
	delete(c.items, key)
	c.mutex.Unlock()

	var zero V
	return zero, false
}

// Set stores an item in the cache
func (c *Cache[K, V]) Set(key K, value V) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.items[key] = &CacheItem[V]{
		Value: value,
	}
}

// SetWithFileInfo stores an item in the cache with file metadata for validation
func (c *Cache[K, V]) SetWithFileInfo(key K, value V, filePath string) error {
	stat, err := os.Stat(filePath)
	if err != nil {
		return err
	}

	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.items[key] = &CacheItem[V]{
		Value:   value,
		ModTime: stat.ModTime(),
		Size:    stat.Size(),
	}

	return nil
}

// Delete removes an item from the cache
func (c *Cache[K, V]) Delete(key K) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	delete(c.items, key)
}

// Clear removes all items from the cache
func (c *Cache[K, V]) Clear() {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.items = make(map[K]*CacheItem[V])
}

// Size returns the number of items in the cache
func (c *Cache[K, V]) Size() int {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	return len(c.items)
}

// Keys returns all keys in the cache
func (c *Cache[K, V]) Keys() []K {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	keys := make([]K, 0, len(c.items))
	for key := range c.items {
		keys = append(keys, key)
	}

	return keys
}

// ForEach iterates over all items in the cache
func (c *Cache[K, V]) ForEach(fn func(key K, value V)) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	for key, item := range c.items {
		fn(key, item.Value)
	}
}

// GetStats returns cache statistics
func (c *Cache[K, V]) GetStats() CacheStats {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	return CacheStats{
		Size: len(c.items),
	}
}

// CacheStats provides cache statistics
type CacheStats struct {
	Size int
}
