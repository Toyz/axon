package utils

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// FileReader provides common file reading functionality with caching
type FileReader struct {
	fileSet    *token.FileSet
	astCache   map[string]*cachedAST
	contentCache map[string]*cachedContent
	mutex      sync.RWMutex
}

// cachedAST holds a cached AST file with metadata
type cachedAST struct {
	file    *ast.File
	modTime time.Time
	size    int64
}

// cachedContent holds cached file content with metadata
type cachedContent struct {
	content string
	modTime time.Time
	size    int64
}

// NewFileReader creates a new FileReader instance with caching
func NewFileReader() *FileReader {
	return &FileReader{
		fileSet:      token.NewFileSet(),
		astCache:     make(map[string]*cachedAST),
		contentCache: make(map[string]*cachedContent),
	}
}

// ParseGoFile parses a Go source file and returns the AST with caching
func (fr *FileReader) ParseGoFile(filePath string) (*ast.File, error) {
	// Validate and clean the path
	cleanPath, err := fr.validateAndCleanPath(filePath)
	if err != nil {
		return nil, err
	}

	// Check cache first
	fr.mutex.RLock()
	if cached, exists := fr.astCache[cleanPath]; exists {
		// Check if file has been modified
		if stat, err := os.Stat(cleanPath); err == nil {
			if stat.ModTime().Equal(cached.modTime) && stat.Size() == cached.size {
				fr.mutex.RUnlock()
				return cached.file, nil
			}
		}
		// File changed or error, remove from cache
		delete(fr.astCache, cleanPath)
	}
	fr.mutex.RUnlock()

	// Parse the file
	file, err := parser.ParseFile(fr.fileSet, cleanPath, nil, parser.ParseComments)
	if err != nil {
		return nil, fmt.Errorf("failed to parse Go file %s: %w", filepath.Base(cleanPath), err)
	}

	// Cache the result
	if stat, err := os.Stat(cleanPath); err == nil {
		fr.mutex.Lock()
		fr.astCache[cleanPath] = &cachedAST{
			file:    file,
			modTime: stat.ModTime(),
			size:    stat.Size(),
		}
		fr.mutex.Unlock()
	}

	return file, nil
}

// ParseGoSource parses Go source code from a string
func (fr *FileReader) ParseGoSource(filename, source string) (*ast.File, error) {
	fr.mutex.RLock()
	file, err := parser.ParseFile(fr.fileSet, filename, source, parser.ParseComments)
	fr.mutex.RUnlock()
	if err != nil {
		return nil, fmt.Errorf("failed to parse Go source: %w", err)
	}
	return file, nil
}

// ReadFile reads a file and returns its contents as a string with caching
func (fr *FileReader) ReadFile(filePath string) (string, error) {
	cleanPath, err := fr.validateAndCleanPath(filePath)
	if err != nil {
		return "", err
	}

	// Check cache first
	fr.mutex.RLock()
	if cached, exists := fr.contentCache[cleanPath]; exists {
		// Check if file has been modified
		if stat, err := os.Stat(cleanPath); err == nil {
			if stat.ModTime().Equal(cached.modTime) && stat.Size() == cached.size {
				fr.mutex.RUnlock()
				return cached.content, nil
			}
		}
		// File changed or error, remove from cache
		delete(fr.contentCache, cleanPath)
	}
	fr.mutex.RUnlock()

	content, err := os.ReadFile(cleanPath)
	if err != nil {
		return "", fmt.Errorf("failed to read file %s: %w", filepath.Base(cleanPath), err)
	}

	contentStr := string(content)

	// Cache the result
	if stat, err := os.Stat(cleanPath); err == nil {
		fr.mutex.Lock()
		fr.contentCache[cleanPath] = &cachedContent{
			content: contentStr,
			modTime: stat.ModTime(),
			size:    stat.Size(),
		}
		fr.mutex.Unlock()
	}

	return contentStr, nil
}

// GetFileSet returns the token.FileSet used by this reader
func (fr *FileReader) GetFileSet() *token.FileSet {
	return fr.fileSet
}

// ClearCache clears all cached files
func (fr *FileReader) ClearCache() {
	fr.mutex.Lock()
	defer fr.mutex.Unlock()
	fr.astCache = make(map[string]*cachedAST)
	fr.contentCache = make(map[string]*cachedContent)
}

// InvalidateFile removes a specific file from the cache
func (fr *FileReader) InvalidateFile(filePath string) {
	cleanPath, err := fr.validateAndCleanPath(filePath)
	if err != nil {
		return
	}

	fr.mutex.Lock()
	delete(fr.astCache, cleanPath)
	delete(fr.contentCache, cleanPath)
	fr.mutex.Unlock()
}

// GetCacheStats returns statistics about the cache
func (fr *FileReader) GetCacheStats() (astFiles, contentFiles int) {
	fr.mutex.RLock()
	defer fr.mutex.RUnlock()
	return len(fr.astCache), len(fr.contentCache)
}

// validateAndCleanPath validates and cleans a file path
func (fr *FileReader) validateAndCleanPath(filePath string) (string, error) {
	if filePath == "" {
		return "", fmt.Errorf("file path cannot be empty")
	}

	// Clean the path to prevent path traversal
	cleanPath := filepath.Clean(filePath)

	// Ensure the clean path doesn't contain path traversal attempts
	if strings.Contains(cleanPath, "..") {
		// Allow .. only if it's at the beginning (relative path)
		if !strings.HasPrefix(cleanPath, "..") {
			return "", fmt.Errorf("path traversal not allowed in file path: %s", filePath)
		}
	}

	// Check if file exists
	if _, err := os.Stat(cleanPath); os.IsNotExist(err) {
		return "", fmt.Errorf("file does not exist: %s", cleanPath)
	}

	return cleanPath, nil
}
