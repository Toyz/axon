package fileops

import (
	"go/ast"
	"go/token"
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

// GetWithFileValidation retrieves an item from the cache with file-based validation
func (c *Cache[K, V]) GetWithFileValidation(key K, filePath string) (V, bool) {
	c.mutex.RLock()
	item, exists := c.items[key]
	c.mutex.RUnlock()

	if !exists {
		var zero V
		return zero, false
	}

	// Check if file has been modified
	stat, err := os.Stat(filePath)
	if err != nil {
		// File doesn't exist or can't be accessed, remove from cache
		c.Delete(key)
		var zero V
		return zero, false
	}

	if stat.ModTime().After(item.ModTime) || stat.Size() != item.Size {
		// File has been modified, remove from cache
		c.Delete(key)
		var zero V
		return zero, false
	}

	return item.Value, true
}

// SetWithFileInfo sets an item in the cache with file metadata
func (c *Cache[K, V]) SetWithFileInfo(key K, value V, filePath string) {
	stat, err := os.Stat(filePath)
	if err != nil {
		return // Can't get file info, don't cache
	}

	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.items[key] = &CacheItem[V]{
		Value:   value,
		ModTime: stat.ModTime(),
		Size:    stat.Size(),
	}
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

// CacheManager provides centralized caching functionality for file operations
type CacheManager struct {
	astCache     *Cache[string, *ast.File]
	contentCache *Cache[string, string]
	fileSet      *token.FileSet
}

// NewCacheManager creates a new CacheManager instance
func NewCacheManager() *CacheManager {
	return &CacheManager{
		astCache:     NewCache[string, *ast.File](),
		contentCache: NewCache[string, string](),
		fileSet:      token.NewFileSet(),
	}
}

// GetAST retrieves a cached AST or returns false if not found
func (cm *CacheManager) GetAST(filePath string) (*ast.File, bool) {
	return cm.astCache.GetWithFileValidation(filePath, filePath)
}

// SetAST caches an AST with file validation
func (cm *CacheManager) SetAST(filePath string, file *ast.File) {
	cm.astCache.SetWithFileInfo(filePath, file, filePath)
}

// GetContent retrieves cached file content or returns false if not found
func (cm *CacheManager) GetContent(filePath string) (string, bool) {
	return cm.contentCache.GetWithFileValidation(filePath, filePath)
}

// SetContent caches file content with file validation
func (cm *CacheManager) SetContent(filePath string, content string) {
	cm.contentCache.SetWithFileInfo(filePath, content, filePath)
}

// GetFileSet returns the token.FileSet used for parsing
func (cm *CacheManager) GetFileSet() *token.FileSet {
	return cm.fileSet
}

// ClearAll clears all cached data
func (cm *CacheManager) ClearAll() {
	cm.astCache.Clear()
	cm.contentCache.Clear()
}

// InvalidateFile removes a specific file from all caches
func (cm *CacheManager) InvalidateFile(filePath string) {
	cm.astCache.Delete(filePath)
	cm.contentCache.Delete(filePath)
}

// GetCacheStats returns statistics about cached items
func (cm *CacheManager) GetCacheStats() (astFiles, contentFiles int) {
	return cm.astCache.Size(), cm.contentCache.Size()
}

// HasAST checks if an AST is cached for the given file
func (cm *CacheManager) HasAST(filePath string) bool {
	_, exists := cm.astCache.GetWithFileValidation(filePath, filePath)
	return exists
}

// HasContent checks if content is cached for the given file
func (cm *CacheManager) HasContent(filePath string) bool {
	_, exists := cm.contentCache.GetWithFileValidation(filePath, filePath)
	return exists
}