package utils

import (
	"fmt"
	"path/filepath"
	"strings"

	"golang.org/x/mod/modfile"
)

// GoModParser provides utilities for parsing go.mod files
type GoModParser struct {
	fileReader *FileReader
}

// NewGoModParser creates a new go.mod parser with caching
func NewGoModParser(fileReader *FileReader) *GoModParser {
	return &GoModParser{
		fileReader: fileReader,
	}
}

// ParseModuleName extracts the module name from a go.mod file
func (p *GoModParser) ParseModuleName(goModPath string) (string, error) {
	// Validate path
	cleanPath := filepath.Clean(goModPath)
	if !strings.HasSuffix(cleanPath, "go.mod") {
		return "", fmt.Errorf("file is not a go.mod file: %s", goModPath)
	}

	// Use cached file reading
	content, err := p.fileReader.ReadFile(cleanPath)
	if err != nil {
		return "", fmt.Errorf("failed to read go.mod file: %w", err)
	}

	// Parse using official modfile parser
	modFile, err := modfile.Parse(cleanPath, []byte(content), nil)
	if err != nil {
		return "", fmt.Errorf("failed to parse go.mod file: %w", err)
	}

	if modFile.Module == nil {
		return "", fmt.Errorf("no module declaration found in go.mod")
	}

	return modFile.Module.Mod.Path, nil
}

// FindGoModFile searches for go.mod file starting from the given directory and walking up
func (p *GoModParser) FindGoModFile(startDir string) (string, error) {
	currentDir := filepath.Clean(startDir)
	
	for {
		goModPath := filepath.Join(currentDir, "go.mod")
		
		// Check if go.mod exists in current directory
		if content, err := p.fileReader.ReadFile(goModPath); err == nil && content != "" {
			return goModPath, nil
		}
		
		// Move to parent directory
		parentDir := filepath.Dir(currentDir)
		if parentDir == currentDir {
			// Reached root directory
			break
		}
		currentDir = parentDir
	}
	
	return "", fmt.Errorf("go.mod file not found")
}