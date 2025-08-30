package utils

import (
	"fmt"
	"go/ast"
	"os"
	"path/filepath"
	"strings"
)

// FileProcessor provides utilities for common file processing operations
type FileProcessor struct {
	fileReader *FileReader
}

// NewFileProcessor creates a new file processor
func NewFileProcessor() *FileProcessor {
	return &FileProcessor{
		fileReader: NewFileReader(),
	}
}

// NewFileProcessorWithReader creates a file processor with an existing FileReader
func NewFileProcessorWithReader(reader *FileReader) *FileProcessor {
	return &FileProcessor{
		fileReader: reader,
	}
}

// FileFilter defines a function that determines whether a file should be processed
type FileFilter func(path string, info os.DirEntry) bool

// DirectoryFilter defines a function that determines whether a directory should be processed
type DirectoryFilter func(path string, info os.DirEntry) bool

// FileWalkOptions configures file walking behavior
type FileWalkOptions struct {
	FileFilter      FileFilter
	DirectoryFilter DirectoryFilter
	SkipErrors      bool
}

// DefaultGoFileFilter filters for .go files, excluding tests and autogen files
func DefaultGoFileFilter() FileFilter {
	return func(path string, info os.DirEntry) bool {
		if info.IsDir() {
			return false
		}
		
		name := info.Name()
		return strings.HasSuffix(name, ".go") &&
			!strings.HasSuffix(name, "_test.go") &&
			!strings.HasPrefix(name, "autogen_")
	}
}

// TestGoFileFilter filters for Go test files
func TestGoFileFilter() FileFilter {
	return func(path string, info os.DirEntry) bool {
		if info.IsDir() {
			return false
		}
		
		return strings.HasSuffix(info.Name(), "_test.go")
	}
}

// AutogenFileFilter filters for autogen files
func AutogenFileFilter() FileFilter {
	return func(path string, info os.DirEntry) bool {
		if info.IsDir() {
			return false
		}
		
		return strings.HasPrefix(info.Name(), "autogen_")
	}
}

// DefaultDirectoryFilter skips common directories that shouldn't contain source code
func DefaultDirectoryFilter() DirectoryFilter {
	skipDirs := map[string]bool{
		"vendor":       true,
		"node_modules": true,
		".git":         true,
		".svn":         true,
		".hg":          true,
		"testdata":     true,
		"build":        true,
		"dist":         true,
		"target":       true,
	}
	
	return func(path string, info os.DirEntry) bool {
		if !info.IsDir() {
			return true
		}
		
		name := info.Name()
		
		// Skip hidden directories
		if strings.HasPrefix(name, ".") && name != "." && name != ".." {
			return false
		}
		
		// Skip known directories
		return !skipDirs[name]
	}
}

// fileInfoToDirEntry converts os.FileInfo to a simple DirEntry implementation
type fileInfoDirEntry struct {
	info os.FileInfo
}

func (f fileInfoDirEntry) Name() string               { return f.info.Name() }
func (f fileInfoDirEntry) IsDir() bool                { return f.info.IsDir() }
func (f fileInfoDirEntry) Type() os.FileMode          { return f.info.Mode().Type() }
func (f fileInfoDirEntry) Info() (os.FileInfo, error) { return f.info, nil }

// WalkFiles walks through files in a directory tree with filtering
func (fp *FileProcessor) WalkFiles(rootDir string, options FileWalkOptions) ([]string, error) {
	var matchedFiles []string
	
	err := filepath.Walk(rootDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			if options.SkipErrors {
				return nil
			}
			return err
		}
		
		// Convert FileInfo to DirEntry
		dirEntry := fileInfoDirEntry{info: info}
		
		// Apply directory filter
		if info.IsDir() && options.DirectoryFilter != nil {
			if !options.DirectoryFilter(path, dirEntry) {
				return filepath.SkipDir
			}
			return nil
		}
		
		// Apply file filter
		if !info.IsDir() && options.FileFilter != nil {
			if options.FileFilter(path, dirEntry) {
				matchedFiles = append(matchedFiles, path)
			}
		}
		
		return nil
	})
	
	return matchedFiles, err
}

// ScanDirectoriesWithGoFiles scans directories and returns those containing Go files
func (fp *FileProcessor) ScanDirectoriesWithGoFiles(rootDirs []string) ([]string, error) {
	var packageDirs []string
	visited := make(map[string]bool)
	
	for _, rootDir := range rootDirs {
		dirs, err := fp.scanDirectoryRecursive(rootDir, visited)
		if err != nil {
			return nil, err
		}
		packageDirs = append(packageDirs, dirs...)
	}
	
	return packageDirs, nil
}

// scanDirectoryRecursive recursively scans a directory for Go files
func (fp *FileProcessor) scanDirectoryRecursive(dir string, visited map[string]bool) ([]string, error) {
	// Resolve absolute path to handle symlinks and avoid cycles
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return nil, WrapProcessError(fmt.Sprintf("path resolution %s", dir), err)
	}
	
	if visited[absDir] {
		return nil, nil
	}
	visited[absDir] = true
	
	var packageDirs []string
	
	// Check if this directory has Go files
	hasGoFiles, err := fp.HasGoFiles(dir)
	if err != nil {
		return nil, WrapProcessError(fmt.Sprintf("Go file check in %s", dir), err)
	}
	
	if hasGoFiles {
		packageDirs = append(packageDirs, dir)
	}
	
	// Recursively scan subdirectories
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, WrapProcessError(fmt.Sprintf("directory read %s", dir), err)
	}
	
	directoryFilter := DefaultDirectoryFilter()
	
	for _, entry := range entries {
		if entry.IsDir() {
			entryPath := filepath.Join(dir, entry.Name())
			
			// Apply directory filter
			if !directoryFilter(entryPath, entry) {
				continue
			}
			
			subDirs, err := fp.scanDirectoryRecursive(entryPath, visited)
			if err != nil {
				return nil, err
			}
			packageDirs = append(packageDirs, subDirs...)
		}
	}
	
	return packageDirs, nil
}

// HasGoFiles checks if a directory contains any .go files (excluding test files and autogen files)
func (fp *FileProcessor) HasGoFiles(dir string) (bool, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return false, err
	}
	
	fileFilter := DefaultGoFileFilter()
	
	for _, entry := range entries {
		if fileFilter(filepath.Join(dir, entry.Name()), entry) {
			return true, nil
		}
	}
	
	return false, nil
}

// ParseDirectoryFiles parses all Go files in a directory
func (fp *FileProcessor) ParseDirectoryFiles(dirPath string) (map[string]*ast.File, string, error) {
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return nil, "", WrapProcessError(fmt.Sprintf("directory read %s", dirPath), err)
	}
	
	files := make(map[string]*ast.File)
	var packageName string
	fileFilter := DefaultGoFileFilter()
	
	for _, entry := range entries {
		if !fileFilter(filepath.Join(dirPath, entry.Name()), entry) {
			continue
		}
		
		filePath := filepath.Join(dirPath, entry.Name())
		
		file, err := fp.fileReader.ParseGoFile(filePath)
		if err != nil {
			return nil, "", WrapProcessError(fmt.Sprintf("file parse %s", entry.Name()), err)
		}
		
		// Verify all files belong to the same package
		if packageName == "" {
			packageName = file.Name.Name
		} else if file.Name.Name != packageName {
			return nil, "", fmt.Errorf("multiple packages found in directory: %s and %s", packageName, file.Name.Name)
		}
		
		files[filePath] = file
	}
	
	if len(files) == 0 {
		return nil, "", fmt.Errorf("no Go files found in directory")
	}
	
	return files, packageName, nil
}

// CleanDirectories removes autogen files from directories
func (fp *FileProcessor) CleanDirectories(baseDirs []string) ([]string, error) {
	var removedFiles []string
	
	for _, baseDir := range baseDirs {
		err := fp.cleanDirectory(baseDir, &removedFiles)
		if err != nil {
			return removedFiles, WrapProcessError(fmt.Sprintf("directory clean %s", baseDir), err)
		}
	}
	
	return removedFiles, nil
}

// cleanDirectory cleans a single directory tree
func (fp *FileProcessor) cleanDirectory(baseDir string, removedFiles *[]string) error {
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
			err := fp.cleanSingleDirectory(path, removedFiles)
			if err != nil {
				// Log error but continue with other directories
				return nil
			}
		}
		
		return nil
	})
}

// cleanSingleDirectory cleans autogen files from a single directory
func (fp *FileProcessor) cleanSingleDirectory(dir string, removedFiles *[]string) error {
	// Skip if directory doesn't exist
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return nil
	}
	
	autogenFile := filepath.Join(dir, "autogen_module.go")
	
	// Check if autogen file exists
	if _, err := os.Stat(autogenFile); os.IsNotExist(err) {
		return nil // File doesn't exist, nothing to clean
	} else if err != nil {
		return WrapProcessError(fmt.Sprintf("file check %s", autogenFile), err)
	}
	
	// Remove the autogen file
	err := os.Remove(autogenFile)
	if err != nil {
		return WrapProcessError(fmt.Sprintf("file removal %s", autogenFile), err)
	}
	
	*removedFiles = append(*removedFiles, autogenFile)
	return nil
}

// GetFileReader returns the underlying FileReader for advanced operations
func (fp *FileProcessor) GetFileReader() *FileReader {
	return fp.fileReader
}

// GetDirectoryFilter returns the default directory filter
func (fp *FileProcessor) GetDirectoryFilter() DirectoryFilter {
	return DefaultDirectoryFilter()
}