package utils

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestCache_BasicOperations(t *testing.T) {
	cache := NewCache[string, int]()
	
	// Test Set and Get
	cache.Set("key1", 42)
	value, exists := cache.Get("key1")
	if !exists {
		t.Error("expected key1 to exist")
	}
	if value != 42 {
		t.Errorf("expected value 42, got %d", value)
	}
	
	// Test non-existent key
	_, exists = cache.Get("nonexistent")
	if exists {
		t.Error("expected nonexistent key to not exist")
	}
	
	// Test Delete
	cache.Delete("key1")
	_, exists = cache.Get("key1")
	if exists {
		t.Error("expected key1 to be deleted")
	}
}

func TestCache_Clear(t *testing.T) {
	cache := NewCache[string, string]()
	
	cache.Set("key1", "value1")
	cache.Set("key2", "value2")
	
	if cache.Size() != 2 {
		t.Errorf("expected size 2, got %d", cache.Size())
	}
	
	cache.Clear()
	
	if cache.Size() != 0 {
		t.Errorf("expected size 0 after clear, got %d", cache.Size())
	}
}

func TestCache_Keys(t *testing.T) {
	cache := NewCache[string, int]()
	
	cache.Set("key1", 1)
	cache.Set("key2", 2)
	cache.Set("key3", 3)
	
	keys := cache.Keys()
	if len(keys) != 3 {
		t.Errorf("expected 3 keys, got %d", len(keys))
	}
	
	// Check all keys are present
	keyMap := make(map[string]bool)
	for _, key := range keys {
		keyMap[key] = true
	}
	
	expectedKeys := []string{"key1", "key2", "key3"}
	for _, expected := range expectedKeys {
		if !keyMap[expected] {
			t.Errorf("expected key %s to be present", expected)
		}
	}
}

func TestCache_ForEach(t *testing.T) {
	cache := NewCache[string, int]()
	
	cache.Set("key1", 1)
	cache.Set("key2", 2)
	cache.Set("key3", 3)
	
	sum := 0
	cache.ForEach(func(key string, value int) {
		sum += value
	})
	
	if sum != 6 {
		t.Errorf("expected sum 6, got %d", sum)
	}
}

func TestCache_GetStats(t *testing.T) {
	cache := NewCache[string, int]()
	
	stats := cache.GetStats()
	if stats.Size != 0 {
		t.Errorf("expected initial size 0, got %d", stats.Size)
	}
	
	cache.Set("key1", 1)
	cache.Set("key2", 2)
	
	stats = cache.GetStats()
	if stats.Size != 2 {
		t.Errorf("expected size 2, got %d", stats.Size)
	}
}

func TestCache_FileValidation(t *testing.T) {
	cache := NewCache[string, string]()
	
	// Create a temporary file
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.txt")
	
	content := "initial content"
	err := os.WriteFile(tmpFile, []byte(content), 0644)
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	
	// Cache the content with file info
	err = cache.SetWithFileInfo("test", content, tmpFile)
	if err != nil {
		t.Fatalf("failed to set cache with file info: %v", err)
	}
	
	// Get with file validation - should return cached content
	value, exists := cache.GetWithFileValidation("test", tmpFile)
	if !exists {
		t.Error("expected cached value to exist")
	}
	if value != content {
		t.Errorf("expected content %s, got %s", content, value)
	}
	
	// Modify the file
	time.Sleep(10 * time.Millisecond) // Ensure different modtime
	newContent := "modified content"
	err = os.WriteFile(tmpFile, []byte(newContent), 0644)
	if err != nil {
		t.Fatalf("failed to modify temp file: %v", err)
	}
	
	// Get with file validation - should return false due to file change
	_, exists = cache.GetWithFileValidation("test", tmpFile)
	if exists {
		t.Error("expected cached value to be invalidated after file change")
	}
	
	// Verify the item was removed from cache
	if cache.Size() != 0 {
		t.Errorf("expected cache to be empty after invalidation, got size %d", cache.Size())
	}
}

func TestCache_FileValidationNonExistentFile(t *testing.T) {
	cache := NewCache[string, string]()
	
	// Try to get with validation for non-existent file
	_, exists := cache.GetWithFileValidation("test", "/nonexistent/file.txt")
	if exists {
		t.Error("expected false for non-existent file")
	}
}

func TestCache_SetWithFileInfoNonExistentFile(t *testing.T) {
	cache := NewCache[string, string]()
	
	// Try to set with file info for non-existent file
	err := cache.SetWithFileInfo("test", "content", "/nonexistent/file.txt")
	if err == nil {
		t.Error("expected error for non-existent file")
	}
}

func TestCache_DifferentTypes(t *testing.T) {
	// Test with different key and value types
	intCache := NewCache[int, string]()
	intCache.Set(1, "one")
	intCache.Set(2, "two")
	
	value, exists := intCache.Get(1)
	if !exists || value != "one" {
		t.Errorf("expected 'one', got %s (exists: %v)", value, exists)
	}
	
	// Test with struct types
	type TestStruct struct {
		Name string
		Age  int
	}
	
	structCache := NewCache[string, TestStruct]()
	testStruct := TestStruct{Name: "John", Age: 30}
	structCache.Set("person", testStruct)
	
	retrieved, exists := structCache.Get("person")
	if !exists {
		t.Error("expected struct to exist in cache")
	}
	if retrieved.Name != "John" || retrieved.Age != 30 {
		t.Errorf("expected John/30, got %s/%d", retrieved.Name, retrieved.Age)
	}
}

func TestCache_ConcurrentAccess(t *testing.T) {
	cache := NewCache[string, int]()
	
	// Test concurrent reads and writes
	done := make(chan bool, 10)
	
	// Start multiple goroutines writing to cache
	for i := 0; i < 5; i++ {
		go func(id int) {
			for j := 0; j < 100; j++ {
				cache.Set(fmt.Sprintf("key%d_%d", id, j), id*100+j)
			}
			done <- true
		}(i)
	}
	
	// Start multiple goroutines reading from cache
	for i := 0; i < 5; i++ {
		go func(id int) {
			for j := 0; j < 100; j++ {
				cache.Get(fmt.Sprintf("key%d_%d", id, j))
			}
			done <- true
		}(i)
	}
	
	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		<-done
	}
	
	// Verify cache has expected number of items
	if cache.Size() != 500 {
		t.Errorf("expected 500 items in cache, got %d", cache.Size())
	}
}