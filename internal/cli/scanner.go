package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// DirectoryScanner handles recursive directory scanning for Go files
type DirectoryScanner struct{}

// NewDirectoryScanner creates a new directory scanner
func NewDirectoryScanner() *DirectoryScanner {
	return &DirectoryScanner{}
}

// ScanDirectories recursively scans the provided directories for Go packages
// Returns a list of directories that contain Go files
// Supports Go-style patterns like "./..." for recursive scanning
func (s *DirectoryScanner) ScanDirectories(rootDirs []string) ([]string, error) {
	var packageDirs []string
	visited := make(map[string]bool)

	for _, rootDir := range rootDirs {
		// Handle Go-style recursive patterns like "./..."
		if strings.HasSuffix(rootDir, "/...") {
			baseDir := strings.TrimSuffix(rootDir, "/...")
			if baseDir == "" {
				baseDir = "."
			}

			// Clean and resolve the base path
			cleanPath, err := filepath.Abs(baseDir)
			if err != nil {
				return nil, fmt.Errorf("failed to resolve path %s: %w", baseDir, err)
			}

			// Recursively scan this directory
			dirs, err := s.scanDirectory(cleanPath, visited)
			if err != nil {
				return nil, fmt.Errorf("failed to scan directory %s: %w", baseDir, err)
			}

			packageDirs = append(packageDirs, dirs...)
		} else {
			// Clean and resolve the path
			cleanPath, err := filepath.Abs(rootDir)
			if err != nil {
				return nil, fmt.Errorf("failed to resolve path %s: %w", rootDir, err)
			}

			// For specific directories (not using ./...), scan recursively
			dirs, err := s.scanDirectory(cleanPath, visited)
			if err != nil {
				return nil, fmt.Errorf("failed to scan directory %s: %w", rootDir, err)
			}

			packageDirs = append(packageDirs, dirs...)
		}
	}

	return packageDirs, nil
}

// scanDirectory recursively scans a single directory for Go packages
func (s *DirectoryScanner) scanDirectory(dir string, visited map[string]bool) ([]string, error) {
	// Avoid scanning the same directory twice
	if visited[dir] {
		return nil, nil
	}
	visited[dir] = true

	var packageDirs []string

	// Check if this directory contains Go files
	hasGoFiles, err := s.hasGoFiles(dir)
	if err != nil {
		return nil, fmt.Errorf("failed to check for Go files in %s: %w", dir, err)
	}

	// If this directory has Go files, include it
	if hasGoFiles {
		packageDirs = append(packageDirs, dir)
	}

	// Recursively scan subdirectories
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory %s: %w", dir, err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			// Skip common directories that shouldn't contain source code
			if s.shouldSkipDirectory(entry.Name()) {
				continue
			}

			subDir := filepath.Join(dir, entry.Name())
			subDirs, err := s.scanDirectory(subDir, visited)
			if err != nil {
				return nil, err
			}
			packageDirs = append(packageDirs, subDirs...)
		}
	}

	return packageDirs, nil
}

// hasGoFiles checks if a directory contains any .go files (excluding test files)
func (s *DirectoryScanner) hasGoFiles(dir string) (bool, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return false, err
	}

	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".go") {
			// Skip test files and generated files
			if !strings.HasSuffix(entry.Name(), "_test.go") &&
				!strings.HasPrefix(entry.Name(), "autogen_") {
				return true, nil
			}
		}
	}

	return false, nil
}

// shouldSkipDirectory determines if a directory should be skipped during scanning
func (s *DirectoryScanner) shouldSkipDirectory(name string) bool {
	skipDirs := []string{
		"vendor",
		"node_modules",
		".git",
		".svn",
		".hg",
		"testdata",
		"tmp",
		"temp",
		"build",
		"dist",
		"bin",
	}

	// Skip hidden directories (starting with .)
	if strings.HasPrefix(name, ".") {
		return true
	}

	// Skip common build/dependency directories
	for _, skipDir := range skipDirs {
		if name == skipDir {
			return true
		}
	}

	return false
}
