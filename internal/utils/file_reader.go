package utils

import (
	"go/ast"
	"go/token"

	"github.com/toyz/axon/internal/utils/fileops"
)

// FileReader provides common file reading functionality with caching
type FileReader struct {
	fileOps *fileops.FileOps
}

// NewFileReader creates a new FileReader instance with caching
func NewFileReader() *FileReader {
	return &FileReader{
		fileOps: fileops.NewFileOps(),
	}
}

// NewFileReaderWithCache creates a new FileReader instance with shared caching
func NewFileReaderWithCache(cacheManager *fileops.CacheManager) *FileReader {
	return &FileReader{
		fileOps: fileops.NewFileOpsWithCache(cacheManager),
	}
}

// ParseGoFile parses a Go source file and returns the AST with caching
func (fr *FileReader) ParseGoFile(filePath string) (*ast.File, error) {
	return fr.fileOps.ParseGoFile(filePath)
}

// ParseGoSource parses Go source code from a string
func (fr *FileReader) ParseGoSource(filename, source string) (*ast.File, error) {
	return fr.fileOps.ParseGoSource(filename, source)
}

// ReadFile reads a file and returns its contents as a string with caching
func (fr *FileReader) ReadFile(filePath string) (string, error) {
	return fr.fileOps.ReadFile(filePath)
}

// GetFileSet returns the token.FileSet used by this reader
func (fr *FileReader) GetFileSet() *token.FileSet {
	return fr.fileOps.CacheManager().GetFileSet()
}

// ClearCache clears all cached files
func (fr *FileReader) ClearCache() {
	fr.fileOps.CacheManager().ClearAll()
}

// InvalidateFile removes a specific file from the cache
func (fr *FileReader) InvalidateFile(filePath string) {
	cleanPath, err := fr.fileOps.PathValidator().ValidateAndClean(filePath)
	if err != nil {
		return
	}

	fr.fileOps.CacheManager().InvalidateFile(cleanPath)
}

// GetCacheStats returns statistics about the cache
func (fr *FileReader) GetCacheStats() (astFiles, contentFiles int) {
	return fr.fileOps.CacheManager().GetCacheStats()
}
