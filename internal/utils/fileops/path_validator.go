package fileops

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// PathValidator provides centralized path validation and cleaning functionality
type PathValidator struct{}

// NewPathValidator creates a new PathValidator instance
func NewPathValidator() *PathValidator {
	return &PathValidator{}
}

// ValidateAndClean validates and cleans a file path, ensuring it's safe and exists
func (pv *PathValidator) ValidateAndClean(filePath string) (string, error) {
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

// ValidateAndCleanOptional validates and cleans a path but doesn't require it to exist
func (pv *PathValidator) ValidateAndCleanOptional(filePath string) (string, error) {
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

	return cleanPath, nil
}

// Exists checks if a path exists
func (pv *PathValidator) Exists(path string) bool {
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
}

// IsDir checks if a path exists and is a directory
func (pv *PathValidator) IsDir(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

// IsFile checks if a path exists and is a regular file
func (pv *PathValidator) IsFile(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

// GetAbsolutePath resolves a path to its absolute form
func (pv *PathValidator) GetAbsolutePath(path string) (string, error) {
	cleanPath, err := pv.ValidateAndCleanOptional(path)
	if err != nil {
		return "", err
	}

	absPath, err := filepath.Abs(cleanPath)
	if err != nil {
		return "", fmt.Errorf("failed to resolve absolute path for %s: %w", cleanPath, err)
	}

	return absPath, nil
}