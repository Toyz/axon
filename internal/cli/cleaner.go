package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Cleaner handles cleaning up generated files
type Cleaner struct {
	scanner *DirectoryScanner
}

// NewCleaner creates a new cleaner
func NewCleaner() *Cleaner {
	return &Cleaner{
		scanner: NewDirectoryScanner(),
	}
}

// CleanGeneratedFiles removes all autogen_module.go files from the specified directories
func (c *Cleaner) CleanGeneratedFiles(directories []string) error {
	var removedFiles []string

	for _, dir := range directories {
		err := c.cleanDirectory(dir, &removedFiles)
		if err != nil {
			return fmt.Errorf("failed to clean directory %s: %w", dir, err)
		}
	}

	return nil
}

// cleanDirectory recursively cleans a single directory
func (c *Cleaner) cleanDirectory(dir string, removedFiles *[]string) error {
	// Handle Go-style patterns like ./...
	if strings.HasSuffix(dir, "/...") {
		baseDir := strings.TrimSuffix(dir, "/...")
		if baseDir == "." {
			baseDir = ""
		}
		return c.cleanRecursively(baseDir, removedFiles)
	}

	// Clean specific directory
	return c.cleanSingleDirectory(dir, removedFiles)
}

// cleanRecursively cleans directories recursively
func (c *Cleaner) cleanRecursively(baseDir string, removedFiles *[]string) error {
	startDir := "."
	if baseDir != "" {
		startDir = baseDir
	}

	return filepath.Walk(startDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			// Skip directories that don't exist or can't be accessed
			return nil
		}

		if info.IsDir() {
			err := c.cleanSingleDirectory(path, removedFiles)
			if err != nil {
				// Log error but continue with other directories
				return nil
			}
		}

		return nil
	})
}

// cleanSingleDirectory cleans a single directory
func (c *Cleaner) cleanSingleDirectory(dir string, removedFiles *[]string) error {
	// Skip if directory doesn't exist
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return nil
	}

	autogenFile := filepath.Join(dir, "autogen_module.go")

	// Check if the file exists
	if _, err := os.Stat(autogenFile); err != nil {
		if os.IsNotExist(err) {
			return nil // File doesn't exist, nothing to clean
		}
		return fmt.Errorf("failed to check file %s: %w", autogenFile, err)
	}

	// Remove the file
	err := os.Remove(autogenFile)
	if err != nil {
		return fmt.Errorf("failed to remove file %s: %w", autogenFile, err)
	}

	*removedFiles = append(*removedFiles, autogenFile)
	return nil
}
