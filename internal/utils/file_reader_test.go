package utils

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestFileReaderCaching(t *testing.T) {
	// Create a temporary test file
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test.go")
	testContent := `package main

import "fmt"

func main() {
	fmt.Println("Hello, World!")
}
`

	err := os.WriteFile(testFile, []byte(testContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	reader := NewFileReader()

	// First read - should parse and cache
	file1, err := reader.ParseGoFile(testFile)
	if err != nil {
		t.Fatalf("First parse failed: %v", err)
	}

	// Second read - should use cache
	file2, err := reader.ParseGoFile(testFile)
	if err != nil {
		t.Fatalf("Second parse failed: %v", err)
	}

	// Should be the same AST object (from cache)
	if file1 != file2 {
		t.Error("Expected cached AST to be returned")
	}

	// Check cache stats
	astFiles, contentFiles := reader.GetCacheStats()
	if astFiles != 1 {
		t.Errorf("Expected 1 cached AST file, got %d", astFiles)
	}
	if contentFiles != 0 {
		t.Errorf("Expected 0 cached content files, got %d", contentFiles)
	}

	// Modify the file
	time.Sleep(10 * time.Millisecond) // Ensure modification time changes
	newContent := `package main

import "fmt"

func main() {
	fmt.Println("Hello, Updated World!")
}
`
	err = os.WriteFile(testFile, []byte(newContent), 0644)
	if err != nil {
		t.Fatalf("Failed to update test file: %v", err)
	}

	// Third read - should invalidate cache and re-parse
	file3, err := reader.ParseGoFile(testFile)
	if err != nil {
		t.Fatalf("Third parse failed: %v", err)
	}

	// Should be a different AST object (cache invalidated)
	if file1 == file3 {
		t.Error("Expected new AST after file modification")
	}

	// Test content caching
	content1, err := reader.ReadFile(testFile)
	if err != nil {
		t.Fatalf("First read failed: %v", err)
	}

	content2, err := reader.ReadFile(testFile)
	if err != nil {
		t.Fatalf("Second read failed: %v", err)
	}

	if content1 != content2 {
		t.Error("Expected cached content to match")
	}

	// Check cache stats again
	astFiles, contentFiles = reader.GetCacheStats()
	if astFiles != 1 {
		t.Errorf("Expected 1 cached AST file, got %d", astFiles)
	}
	if contentFiles != 1 {
		t.Errorf("Expected 1 cached content file, got %d", contentFiles)
	}

	// Test cache invalidation
	reader.InvalidateFile(testFile)
	astFiles, contentFiles = reader.GetCacheStats()
	if astFiles != 0 {
		t.Errorf("Expected 0 cached AST files after invalidation, got %d", astFiles)
	}
	if contentFiles != 0 {
		t.Errorf("Expected 0 cached content files after invalidation, got %d", contentFiles)
	}

	// Test cache clearing
	reader.ParseGoFile(testFile) // Add back to cache
	reader.ReadFile(testFile)    // Add back to cache
	reader.ClearCache()

	astFiles, contentFiles = reader.GetCacheStats()
	if astFiles != 0 || contentFiles != 0 {
		t.Errorf("Expected empty cache after ClearCache, got %d AST files and %d content files", astFiles, contentFiles)
	}
}
