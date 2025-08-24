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
	globalParsers  map[string]models.RouteParserMetadata // Global parser registry for cross-package discovery
}

// NewGenerator creates a new CLI generator
func NewGenerator() *Generator {
	moduleResolver := NewModuleResolver()
	return &Generator{
		scanner:        NewDirectoryScanner(),
		moduleResolver: moduleResolver,
		parser:         parser.NewParser(),
		codeGenerator:  generator.NewGeneratorWithResolver(moduleResolver),
		globalParsers:  make(map[string]models.RouteParserMetadata),
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

	// First pass: Discover all parsers across packages
	fmt.Printf("Discovering parsers across all packages...\n")
	
	// Skip parser validation during discovery phase
	g.parser.SetSkipParserValidation(true)
	
	var allPackageMetadata []*models.PackageMetadata
	for _, packageDir := range packageDirs {
		// Parse the package
		metadata, err := g.parser.ParseDirectory(packageDir)
		if err != nil {
			return fmt.Errorf("failed to parse package %s: %w", packageDir, err)
		}

		// Store metadata for second pass
		allPackageMetadata = append(allPackageMetadata, metadata)

		// Collect parsers from this package
		err = g.collectParsersFromPackage(metadata, moduleName, packageDir)
		if err != nil {
			return fmt.Errorf("failed to collect parsers from package %s: %w", packageDir, err)
		}
	}
	
	// Re-enable parser validation for second pass
	g.parser.SetSkipParserValidation(false)

	// Build global parser registry and register with code generator
	err = g.buildGlobalParserRegistry()
	if err != nil {
		return fmt.Errorf("failed to build global parser registry: %w", err)
	}

	fmt.Printf("Discovered %d parsers across all packages\n", len(g.globalParsers))

	// Validate custom parsers across all packages using global registry
	fmt.Printf("Validating custom parsers across all packages...\n")
	for _, metadata := range allPackageMetadata {
		err = g.parser.ValidateCustomParsersWithRegistry(metadata, g.globalParsers)
		if err != nil {
			return fmt.Errorf("parser validation failed for package %s: %w", metadata.PackageName, err)
		}
	}

	// Second pass: Generate code with global parser registry
	var allModules []models.ModuleReference
	for i, metadata := range allPackageMetadata {
		packageDir := packageDirs[i]
		fmt.Printf("Processing package: %s\n", packageDir)

		// Skip packages with no annotations
		if g.hasNoAnnotations(metadata) {
			fmt.Printf("  Skipping package %s (no annotations found)\n", metadata.PackageName)
			continue
		}

		// Skip packages with only parsers (parsers don't need FX modules)
		if g.hasOnlyParsers(metadata) {
			fmt.Printf("  Skipping package %s (only contains parsers - no FX module needed)\n", metadata.PackageName)
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
// Note: RouteParsers are excluded because they are just functions and don't need FX modules
func (g *Generator) hasNoAnnotations(metadata *models.PackageMetadata) bool {
	return len(metadata.Controllers) == 0 &&
		len(metadata.CoreServices) == 0 &&
		len(metadata.Middlewares) == 0 &&
		len(metadata.Interfaces) == 0 &&
		len(metadata.Loggers) == 0 &&
		len(metadata.RouteParsers) == 0
}

// hasOnlyParsers checks if a package only contains parsers (which don't need FX modules)
func (g *Generator) hasOnlyParsers(metadata *models.PackageMetadata) bool {
	return len(metadata.Controllers) == 0 &&
		len(metadata.CoreServices) == 0 &&
		len(metadata.Middlewares) == 0 &&
		len(metadata.Interfaces) == 0 &&
		len(metadata.Loggers) == 0 &&
		len(metadata.RouteParsers) > 0
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

// collectParsersFromPackage collects all parsers from a package and adds them to the global registry
func (g *Generator) collectParsersFromPackage(metadata *models.PackageMetadata, moduleName, packageDir string) error {
	for _, parser := range metadata.RouteParsers {
		// Resolve import path for this parser
		importPath, err := g.resolveParserImportPath(moduleName, packageDir, metadata.PackageName)
		if err != nil {
			return fmt.Errorf("failed to resolve import path for parser %s: %w", parser.FunctionName, err)
		}

		// Create enhanced parser metadata with package path
		enhancedParser := parser
		enhancedParser.PackagePath = packageDir

		// Check for conflicts
		if existing, exists := g.globalParsers[parser.TypeName]; exists {
			return fmt.Errorf("parser conflict: type '%s' is already registered by parser '%s' at %s:%d, cannot register duplicate parser '%s' at %s:%d",
				parser.TypeName,
				existing.FunctionName, existing.FileName, existing.Line,
				parser.FunctionName, parser.FileName, parser.Line)
		}

		// Add to global registry
		g.globalParsers[parser.TypeName] = enhancedParser
		fmt.Printf("  Discovered parser: %s -> %s (%s)\n", parser.TypeName, parser.FunctionName, importPath)
	}

	return nil
}

// buildGlobalParserRegistry builds the global parser registry and registers it with the code generator
func (g *Generator) buildGlobalParserRegistry() error {
	// Get the parser registry from the code generator
	parserRegistry := g.codeGenerator.GetParserRegistry()
	if parserRegistry == nil {
		return fmt.Errorf("code generator does not have a parser registry")
	}

	// Clear existing custom parsers (keeps built-ins)
	if clearableRegistry, ok := parserRegistry.(interface{ ClearCustomParsers() }); ok {
		clearableRegistry.ClearCustomParsers()
	} else {
		// Fallback to full clear if ClearCustomParsers is not available
		parserRegistry.Clear()
	}

	// Register all discovered parsers
	for _, parser := range g.globalParsers {
		err := parserRegistry.RegisterParser(parser)
		if err != nil {
			return fmt.Errorf("failed to register parser %s: %w", parser.FunctionName, err)
		}
	}

	return nil
}

// resolveParserImportPath resolves the import path for a parser package
func (g *Generator) resolveParserImportPath(moduleName, packageDir, packageName string) (string, error) {
	if g.moduleResolver != nil && moduleName != "" {
		// Use the module resolver to build the proper package path
		importPath, err := g.moduleResolver.BuildPackagePath(moduleName, packageDir)
		if err != nil {
			return "", fmt.Errorf("failed to build package path: %w", err)
		}
		return importPath, nil
	}

	// Fallback to standard internal structure
	if moduleName != "" {
		return fmt.Sprintf("%s/internal/%s", moduleName, packageName), nil
	}

	// Fallback to relative import (not recommended for production)
	return fmt.Sprintf("./%s", packageName), nil
}