package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/toyz/axon/internal/errors"
	"github.com/toyz/axon/internal/utils"
)

// ModuleResolver handles resolving Go module information
type ModuleResolver struct {
	goModParser *utils.GoModParser
}

// NewModuleResolver creates a new module resolver
func NewModuleResolver() *ModuleResolver {
	fileReader := utils.NewFileReader()
	return &ModuleResolver{
		goModParser: utils.NewGoModParser(fileReader),
	}
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
		return "", errors.WrapWithOperation("process", "module name determination (consider using --module flag)", err)
	}

	return moduleName, nil
}

// readGoModFile reads the module name from go.mod file
func (r *ModuleResolver) readGoModFile() (string, error) {
	// Look for go.mod in current directory and parent directories
	currentDir, err := os.Getwd()
	if err != nil {
		return "", errors.WrapWithOperation("process", "current directory retrieval", err)
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

// parseGoModFile parses the module name from a go.mod file using the shared utility
func (r *ModuleResolver) parseGoModFile(path string) (string, error) {
	return r.goModParser.ParseModuleName(path)
}

// BuildPackagePath builds the full import path for a package directory
func (r *ModuleResolver) BuildPackagePath(moduleName, packageDir string) (string, error) {
	// Get the current working directory to calculate relative paths
	currentDir, err := os.Getwd()
	if err != nil {
		return "", errors.WrapWithOperation("process", "current directory retrieval", err)
	}

	// Convert package directory to absolute path
	absPackageDir, err := filepath.Abs(packageDir)
	if err != nil {
		return "", errors.WrapWithOperation("process", "package directory resolution", err)
	}

	// On macOS, /var is a symlink to /private/var, so we need to ensure both paths
	// use the same resolved form. Try to resolve symlinks for both paths.
	resolvedCurrentDir, err := filepath.EvalSymlinks(currentDir)
	if err != nil {
		resolvedCurrentDir = currentDir
	}

	resolvedPackageDir, err := filepath.EvalSymlinks(absPackageDir)
	if err != nil {
		resolvedPackageDir = absPackageDir
	}

	// Calculate relative path from current directory
	relPath, err := filepath.Rel(resolvedCurrentDir, resolvedPackageDir)
	if err != nil {
		// If symlink resolution didn't work, try with original paths
		relPath, err = filepath.Rel(currentDir, absPackageDir)
		if err != nil {
			return "", errors.WrapWithOperation("process", "relative path calculation", err)
		}
	}

	// Convert file path separators to forward slashes for import paths
	importPath := filepath.ToSlash(relPath)

	// Build full import path
	if importPath == "." {
		return moduleName, nil
	}

	return fmt.Sprintf("%s/%s", moduleName, importPath), nil
}
