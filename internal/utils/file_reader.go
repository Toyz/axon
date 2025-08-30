package utils

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
)

// FileReader provides common file reading functionality with caching
type FileReader struct {
	fileSet      *token.FileSet
	astCache     *Cache[string, *ast.File]
	contentCache *Cache[string, string]
}

// NewFileReader creates a new FileReader instance with caching
func NewFileReader() *FileReader {
	return &FileReader{
		fileSet:      token.NewFileSet(),
		astCache:     NewCache[string, *ast.File](),
		contentCache: NewCache[string, string](),
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
	if cached, exists := fr.astCache.GetWithFileValidation(cleanPath, cleanPath); exists {
		return cached, nil
	}

	// Parse the file
	file, err := parser.ParseFile(fr.fileSet, cleanPath, nil, parser.ParseComments)
	if err != nil {
		return nil, fmt.Errorf("failed to parse Go file %s: %w", filepath.Base(cleanPath), err)
	}

	// Cache the result
	fr.astCache.SetWithFileInfo(cleanPath, file, cleanPath)

	return file, nil
}

// ParseGoSource parses Go source code from a string
func (fr *FileReader) ParseGoSource(filename, source string) (*ast.File, error) {
	file, err := parser.ParseFile(fr.fileSet, filename, source, parser.ParseComments)
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
	if cached, exists := fr.contentCache.GetWithFileValidation(cleanPath, cleanPath); exists {
		return cached, nil
	}

	content, err := os.ReadFile(cleanPath)
	if err != nil {
		return "", fmt.Errorf("failed to read file %s: %w", filepath.Base(cleanPath), err)
	}

	contentStr := string(content)

	// Cache the result
	fr.contentCache.SetWithFileInfo(cleanPath, contentStr, cleanPath)

	return contentStr, nil
}

// GetFileSet returns the token.FileSet used by this reader
func (fr *FileReader) GetFileSet() *token.FileSet {
	return fr.fileSet
}

// ClearCache clears all cached files
func (fr *FileReader) ClearCache() {
	fr.astCache.Clear()
	fr.contentCache.Clear()
}

// InvalidateFile removes a specific file from the cache
func (fr *FileReader) InvalidateFile(filePath string) {
	cleanPath, err := fr.validateAndCleanPath(filePath)
	if err != nil {
		return
	}

	fr.astCache.Delete(cleanPath)
	fr.contentCache.Delete(cleanPath)
}

// GetCacheStats returns statistics about the cache
func (fr *FileReader) GetCacheStats() (astFiles, contentFiles int) {
	return fr.astCache.Size(), fr.contentCache.Size()
}

// validateAndCleanPath validates and cleans a file path
func (fr *FileReader) validateAndCleanPath(filePath string) (string, error) {
	if err := NotEmpty("filePath")(filePath); err != nil {
		return "", fmt.Errorf("file path %w", err)
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
