package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/toyz/axon/internal/generator"
	"github.com/toyz/axon/internal/models"
	"github.com/toyz/axon/internal/parser"
)

// Generator coordinates the CLI generation process
type Generator struct {
	scanner        *DirectoryScanner
	moduleResolver *ModuleResolver
	parser         parser.AnnotationParser
	codeGenerator  generator.CodeGenerator
}

// NewGenerator creates a new CLI generator
func NewGenerator() *Generator {
	moduleResolver := NewModuleResolver()
	return &Generator{
		scanner:        NewDirectoryScanner(),
		moduleResolver: moduleResolver,
		parser:         parser.NewParser(),
		codeGenerator:  generator.NewGeneratorWithResolver(moduleResolver),
	}
}

// Run executes the complete generation process
func (g *Generator) Run(config Config) error {
	// Resolve module name
	moduleName, err := g.moduleResolver.ResolveModuleName(config.ModuleName)
	if err != nil {
		return fmt.Errorf("failed to resolve module name: %w", err)
	}

	// Scan directories for Go packages
	packageDirs, err := g.scanner.ScanDirectories(config.Directories)
	if err != nil {
		return fmt.Errorf("failed to scan directories: %w", err)
	}

	if len(packageDirs) == 0 {
		return fmt.Errorf("no Go packages found in specified directories")
	}

	fmt.Printf("Found %d packages to process\n", len(packageDirs))

	// Process each package
	var allModules []models.ModuleReference
	for _, packageDir := range packageDirs {
		fmt.Printf("Processing package: %s\n", packageDir)

		// Parse the package
		metadata, err := g.parser.ParseDirectory(packageDir)
		if err != nil {
			return fmt.Errorf("failed to parse package %s: %w", packageDir, err)
		}

		// Skip packages with no annotations
		if g.hasNoAnnotations(metadata) {
			fmt.Printf("  Skipping package %s (no annotations found)\n", metadata.PackageName)
			continue
		}

		// Build package import path for module references
		packageImportPath, err := g.moduleResolver.BuildPackagePath(moduleName, packageDir)
		if err != nil {
			return fmt.Errorf("failed to build package path for %s: %w", packageDir, err)
		}
		
		// Keep the original directory path for file generation
		metadata.PackagePath = packageDir

		// Generate module for this package
		generatedModule, err := g.codeGenerator.GenerateModuleWithModule(metadata, moduleName)
		if err != nil {
			return fmt.Errorf("failed to generate module for package %s: %w", metadata.PackageName, err)
		}

		// Write the generated module file
		err = g.writeModuleFile(generatedModule)
		if err != nil {
			return fmt.Errorf("failed to write module file for package %s: %w", metadata.PackageName, err)
		}

		// Add to modules list for main.go generation
		allModules = append(allModules, models.ModuleReference{
			PackageName: metadata.PackageName,
			PackagePath: packageImportPath,
			ModuleName:  "AutogenModule",
		})

		fmt.Printf("  Generated module: %s\n", generatedModule.FilePath)
	}

	// Users control their own main.go files - we just generate the modules

	return nil
}

// hasNoAnnotations checks if a package has any annotations worth generating code for
func (g *Generator) hasNoAnnotations(metadata *models.PackageMetadata) bool {
	return len(metadata.Controllers) == 0 &&
		len(metadata.CoreServices) == 0 &&
		len(metadata.Middlewares) == 0 &&
		len(metadata.Interfaces) == 0 &&
		len(metadata.Loggers) == 0
}

// writeModuleFile writes a generated module to disk
func (g *Generator) writeModuleFile(module *models.GeneratedModule) error {
	// Create directory if it doesn't exist
	dir := filepath.Dir(module.FilePath)
	if err := g.ensureDirectory(dir); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}

	// Write the file
	return g.writeFile(module.FilePath, module.Content)
}

// ensureDirectory creates a directory if it doesn't exist
func (g *Generator) ensureDirectory(dir string) error {
	// For relative paths, we don't need to create directories
	// as they should already exist (we're scanning existing directories)
	if !filepath.IsAbs(dir) {
		return nil
	}

	// For absolute paths (like main.go), create the directory
	return filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil && os.IsNotExist(err) {
			return os.MkdirAll(dir, 0755)
		}
		return err
	})
}

// writeFile writes content to a file
func (g *Generator) writeFile(path, content string) error {
	return os.WriteFile(path, []byte(content), 0644)
}