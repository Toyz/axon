package cli

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/toyz/axon/internal/errors"
	"github.com/toyz/axon/internal/utils"
)

// DirectoryScanner handles recursive directory scanning for Go files
type DirectoryScanner struct{
	fileProcessor *utils.FileProcessor
}

// NewDirectoryScanner creates a new directory scanner
func NewDirectoryScanner() *DirectoryScanner {
	return &DirectoryScanner{
		fileProcessor: utils.NewFileProcessor(),
	}
}

// ScanDirectories recursively scans the provided directories for Go packages
// Returns a list of directories that contain Go files
// Supports Go-style patterns like "./..." for recursive scanning
func (s *DirectoryScanner) ScanDirectories(rootDirs []string) ([]string, error) {
	var cleanDirs []string

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
				return nil, errors.WrapWithOperation("process", fmt.Sprintf("path resolution %s", baseDir), err)
			}

			cleanDirs = append(cleanDirs, cleanPath)
		} else {
			// Clean and resolve the path
			cleanPath, err := filepath.Abs(rootDir)
			if err != nil {
				return nil, errors.WrapWithOperation("process", fmt.Sprintf("path resolution %s", rootDir), err)
			}

			cleanDirs = append(cleanDirs, cleanPath)
		}
	}

	return s.fileProcessor.ScanDirectoriesWithGoFiles(cleanDirs)
}
