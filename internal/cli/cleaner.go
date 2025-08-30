package cli

import (
	"os"
	"path/filepath"
	"strings"
	"github.com/toyz/axon/internal/utils"
)

// Cleaner handles cleaning up generated files
type Cleaner struct {
	scanner       *DirectoryScanner
	fileProcessor *utils.FileProcessor
}

// NewCleaner creates a new cleaner
func NewCleaner() *Cleaner {
	return &Cleaner{
		scanner:       NewDirectoryScanner(),
		fileProcessor: utils.NewFileProcessor(),
	}
}

// CleanGeneratedFiles removes all autogen_module.go files from the specified directories
func (c *Cleaner) CleanGeneratedFiles(directories []string) error {
	// Expand directory patterns (like ./...) to actual directories
	expandedDirs, err := c.expandDirectoryPatterns(directories)
	if err != nil {
		return err
	}
	
	// Clean the expanded directories
	_, err = c.fileProcessor.CleanDirectories(expandedDirs)
	return err
}

// expandDirectoryPatterns expands directory patterns like "./..." to actual directories
func (c *Cleaner) expandDirectoryPatterns(directories []string) ([]string, error) {
	var expandedDirs []string
	
	for _, dir := range directories {
		if strings.HasSuffix(dir, "/...") {
			// Handle recursive pattern
			baseDir := strings.TrimSuffix(dir, "/...")
			if baseDir == "" {
				baseDir = "."
			}
			
			// Get all directories recursively
			err := filepath.Walk(baseDir, func(path string, info os.FileInfo, err error) error {
				if err != nil {
					return nil // Skip directories that can't be accessed
				}
				
				if info.IsDir() {
					// Apply directory filter to skip unwanted directories
					dirFilter := c.fileProcessor.GetDirectoryFilter()
					if dirFilter != nil {
						// Create a DirEntry from FileInfo for the filter
						dirEntry := &fileInfoDirEntry{info}
						if !dirFilter(path, dirEntry) {
							if path != baseDir { // Don't skip the base directory itself
								return filepath.SkipDir
							}
						}
					}
					expandedDirs = append(expandedDirs, path)
				}
				
				return nil
			})
			
			if err != nil {
				return nil, err
			}
		} else {
			// Regular directory path
			expandedDirs = append(expandedDirs, dir)
		}
	}
	
	return expandedDirs, nil
}

// fileInfoDirEntry adapts os.FileInfo to os.DirEntry interface
type fileInfoDirEntry struct {
	os.FileInfo
}

func (f *fileInfoDirEntry) Type() os.FileMode {
	return f.FileInfo.Mode().Type()
}

func (f *fileInfoDirEntry) Info() (os.FileInfo, error) {
	return f.FileInfo, nil
}
