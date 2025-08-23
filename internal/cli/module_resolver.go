package cli

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ModuleResolver handles resolving Go module information
type ModuleResolver struct{}

// NewModuleResolver creates a new module resolver
func NewModuleResolver() *ModuleResolver {
	return &ModuleResolver{}
}

// ResolveModuleName resolves the module name for imports
// If customModule is provided, it uses that; otherwise reads from go.mod
func (r *ModuleResolver) ResolveModuleName(customModule string) (string, error) {
	if customModule != "" {
		return customModule, nil
	}

	// Try to find and read go.mod file
	moduleName, err := r.readGoModFile()
	if err != nil {
		return "", fmt.Errorf("failed to determine module name: %w (consider using --module flag)", err)
	}

	return moduleName, nil
}

// readGoModFile reads the module name from go.mod file
func (r *ModuleResolver) readGoModFile() (string, error) {
	// Look for go.mod in current directory and parent directories
	currentDir, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get current directory: %w", err)
	}

	for {
		goModPath := filepath.Join(currentDir, "go.mod")
		if _, err := os.Stat(goModPath); err == nil {
			return r.parseGoModFile(goModPath)
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

// parseGoModFile parses the module name from a go.mod file
func (r *ModuleResolver) parseGoModFile(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("failed to open go.mod file: %w", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "module ") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				return parts[1], nil
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("failed to read go.mod file: %w", err)
	}

	return "", fmt.Errorf("module declaration not found in go.mod")
}

// BuildPackagePath builds the full import path for a package directory
func (r *ModuleResolver) BuildPackagePath(moduleName, packageDir string) (string, error) {
	// Get the current working directory to calculate relative paths
	currentDir, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get current directory: %w", err)
	}

	// Convert package directory to absolute path
	absPackageDir, err := filepath.Abs(packageDir)
	if err != nil {
		return "", fmt.Errorf("failed to resolve package directory: %w", err)
	}

	// Calculate relative path from current directory
	relPath, err := filepath.Rel(currentDir, absPackageDir)
	if err != nil {
		return "", fmt.Errorf("failed to calculate relative path: %w", err)
	}

	// Convert file path separators to forward slashes for import paths
	importPath := filepath.ToSlash(relPath)

	// Build full import path
	if importPath == "." {
		return moduleName, nil
	}

	return fmt.Sprintf("%s/%s", moduleName, importPath), nil
}