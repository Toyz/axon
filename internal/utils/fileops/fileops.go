package fileops

import (
	"go/ast"
	"go/parser"
	"os"
)

// FileOps provides a unified interface for common file operations
// combining path validation, error handling, and caching
type FileOps struct {
	pathValidator *PathValidator
	errorWrapper  *ErrorWrapper
	cacheManager  *CacheManager
}

// NewFileOps creates a new FileOps instance with all components
func NewFileOps() *FileOps {
	return &FileOps{
		pathValidator: NewPathValidator(),
		errorWrapper:  NewErrorWrapper(),
		cacheManager:  NewCacheManager(),
	}
}

// NewFileOpsWithCache creates a FileOps instance with a shared cache manager
func NewFileOpsWithCache(cacheManager *CacheManager) *FileOps {
	return &FileOps{
		pathValidator: NewPathValidator(),
		errorWrapper:  NewErrorWrapper(),
		cacheManager:  cacheManager,
	}
}

// PathValidator returns the path validator instance
func (fo *FileOps) PathValidator() *PathValidator {
	return fo.pathValidator
}

// ErrorWrapper returns the error wrapper instance
func (fo *FileOps) ErrorWrapper() *ErrorWrapper {
	return fo.errorWrapper
}

// CacheManager returns the cache manager instance
func (fo *FileOps) CacheManager() *CacheManager {
	return fo.cacheManager
}

// ParseGoFile parses a Go source file with path validation, error handling, and caching
func (fo *FileOps) ParseGoFile(filePath string) (*ast.File, error) {
	// Validate and clean the path
	cleanPath, err := fo.pathValidator.ValidateAndClean(filePath)
	if err != nil {
		return nil, err
	}

	// Check cache first
	if cached, exists := fo.cacheManager.GetAST(cleanPath); exists {
		return cached, nil
	}

	// Parse the file
	file, err := parser.ParseFile(fo.cacheManager.GetFileSet(), cleanPath, nil, parser.ParseComments)
	if err != nil {
		return nil, fo.errorWrapper.WrapParseError(cleanPath, err)
	}

	// Cache the result
	fo.cacheManager.SetAST(cleanPath, file)

	return file, nil
}

// ParseGoSource parses Go source code from a string
func (fo *FileOps) ParseGoSource(filename, source string) (*ast.File, error) {
	file, err := parser.ParseFile(fo.cacheManager.GetFileSet(), filename, source, parser.ParseComments)
	if err != nil {
		return nil, fo.errorWrapper.WrapParseError(filename, err)
	}
	return file, nil
}

// ReadFile reads a file and returns its contents as a string with caching
func (fo *FileOps) ReadFile(filePath string) (string, error) {
	cleanPath, err := fo.pathValidator.ValidateAndClean(filePath)
	if err != nil {
		return "", err
	}

	// Check cache first
	if cached, exists := fo.cacheManager.GetContent(cleanPath); exists {
		return cached, nil
	}

	content, err := os.ReadFile(cleanPath)
	if err != nil {
		return "", fo.errorWrapper.WrapFileReadError(cleanPath, err)
	}

	contentStr := string(content)

	// Cache the result
	fo.cacheManager.SetContent(cleanPath, contentStr)

	return contentStr, nil
}

// WriteFile writes content to a file with path validation and error handling
func (fo *FileOps) WriteFile(filePath string, content []byte, perm os.FileMode) error {
	cleanPath, err := fo.pathValidator.ValidateAndCleanOptional(filePath)
	if err != nil {
		return err
	}

	err = os.WriteFile(cleanPath, content, perm)
	if err != nil {
		return fo.errorWrapper.WrapFileWriteError(cleanPath, err)
	}

	// Invalidate cache for this file since we modified it
	fo.cacheManager.InvalidateFile(cleanPath)

	return nil
}

// RemoveFile removes a file with path validation and error handling
func (fo *FileOps) RemoveFile(filePath string) error {
	cleanPath, err := fo.pathValidator.ValidateAndClean(filePath)
	if err != nil {
		return err
	}

	err = os.Remove(cleanPath)
	if err != nil {
		return fo.errorWrapper.WrapFileRemovalError(cleanPath, err)
	}

	// Invalidate cache for this file since we removed it
	fo.cacheManager.InvalidateFile(cleanPath)

	return nil
}

// ReadDir reads a directory with path validation and error handling
func (fo *FileOps) ReadDir(dirPath string) ([]os.DirEntry, error) {
	cleanPath, err := fo.pathValidator.ValidateAndClean(dirPath)
	if err != nil {
		return nil, err
	}

	entries, err := os.ReadDir(cleanPath)
	if err != nil {
		return nil, fo.errorWrapper.WrapDirectoryReadError(cleanPath, err)
	}

	return entries, nil
}

// Exists checks if a path exists using the path validator
func (fo *FileOps) Exists(path string) bool {
	return fo.pathValidator.Exists(path)
}

// IsDir checks if a path is a directory using the path validator
func (fo *FileOps) IsDir(path string) bool {
	return fo.pathValidator.IsDir(path)
}

// IsFile checks if a path is a regular file using the path validator
func (fo *FileOps) IsFile(path string) bool {
	return fo.pathValidator.IsFile(path)
}