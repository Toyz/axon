package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/toyz/axon/internal/generator"
	"github.com/toyz/axon/internal/models"
	"github.com/toyz/axon/internal/parser"
	"github.com/toyz/axon/internal/utils"
	"github.com/toyz/axon/pkg/axon"
)

// Generator coordinates the CLI generation process
type Generator struct {
	scanner        *DirectoryScanner
	moduleResolver *ModuleResolver
	parser         parser.AnnotationParser
	codeGenerator  generator.CodeGenerator
	globalParsers     map[string]axon.RouteParserMetadata // Global parser registry for cross-package discovery
	globalMiddleware  map[string]models.MiddlewareMetadata  // Global middleware registry for cross-package discovery
	reporter          *DiagnosticReporter
	diagnostics       *utils.DiagnosticSystem // New diagnostic system
	customModule      string                   // Custom module name if set
	summary           GenerationSummary
}

// NewGenerator creates a new CLI generator
func NewGenerator(verbose bool) *Generator {
	moduleResolver := NewModuleResolver()
	reporter := NewDiagnosticReporter(verbose)
	return &Generator{
		scanner:        NewDirectoryScanner(),
		moduleResolver: moduleResolver,
		parser:         parser.NewParserWithReporter(reporter),
		codeGenerator:  generator.NewGeneratorWithResolver(moduleResolver),
		globalParsers:    make(map[string]axon.RouteParserMetadata),
		globalMiddleware: make(map[string]models.MiddlewareMetadata),
		reporter:         reporter,
		summary:          GenerationSummary{GeneratedFiles: make([]string, 0)},
	}
}

// NewGeneratorWithDiagnostics creates a new CLI generator with the new diagnostic system
func NewGeneratorWithDiagnostics(verbose bool, diagnostics *utils.DiagnosticSystem) *Generator {
	moduleResolver := NewModuleResolver()
	reporter := NewDiagnosticReporter(verbose)
	return &Generator{
		scanner:        NewDirectoryScanner(),
		moduleResolver: moduleResolver,
		parser:         parser.NewParserWithReporter(reporter),
		codeGenerator:  generator.NewGeneratorWithResolver(moduleResolver),
		globalParsers:    make(map[string]axon.RouteParserMetadata),
		globalMiddleware: make(map[string]models.MiddlewareMetadata),
		reporter:         reporter,
		diagnostics:      diagnostics,
		summary:          GenerationSummary{GeneratedFiles: make([]string, 0)},
	}
}

// Generate executes the generation process for the given directories
func (g *Generator) Generate(directories []string) error {
	config := Config{
		Directories: directories,
		Verbose:     g.reporter != nil && g.reporter.verbose,
		ModuleName:  g.customModule,
	}
	
	return g.Run(config)
}

// SetCustomModule sets a custom module name for import resolution
func (g *Generator) SetCustomModule(moduleName string) {
	// Store the custom module in the config for the next run
	g.customModule = moduleName
}

// GetSummary returns the generation summary
func (g *Generator) GetSummary() GenerationSummary {
	return g.summary
}

// Run executes the complete generation process
func (g *Generator) Run(config Config) error {
	startTime := time.Now()
	
	// Initialize summary
	g.summary = GenerationSummary{GeneratedFiles: make([]string, 0)}
	
	// Use new diagnostic system if available, otherwise fall back to old output
	if g.diagnostics != nil {
		g.diagnostics.Verbose("Starting code generation at %s", startTime.Format("15:04:05"))
		g.diagnostics.Debug("Scanning directories: %v", config.Directories)
		if config.ModuleName != "" {
			g.diagnostics.Debug("Using custom module name: %s", config.ModuleName)
		}
	} else if config.Verbose {
		fmt.Printf("Starting code generation at %s\n", startTime.Format("15:04:05"))
		fmt.Printf("Verbose mode enabled\n")
		fmt.Printf("Scanning directories: %v\n", config.Directories)
		if config.ModuleName != "" {
			fmt.Printf("Using custom module name: %s\n", config.ModuleName)
		}
		fmt.Printf("\n")
	}
	
	// Resolve module name
	if g.diagnostics != nil {
		g.diagnostics.StartProgress("Resolving module name")
	} else if config.Verbose {
		fmt.Printf("Resolving module name...\n")
	}
	moduleName, err := g.moduleResolver.ResolveModuleName(config.ModuleName)
	if err != nil {
		if g.diagnostics != nil {
			g.diagnostics.EndProgress(false, "")
			g.diagnostics.Error("Failed to resolve module name: %v", err)
		}
		return &models.GeneratorError{
			Type:    models.ErrorTypeValidation,
			Message: fmt.Sprintf("Failed to resolve module name: %v", err),
			Suggestions: []string{
				"Check your go.mod file exists and is valid",
				"Ensure you're running from the correct directory",
				"Try specifying --module flag explicitly",
			},
			Context: map[string]interface{}{
				"provided_module": config.ModuleName,
				"directories":     config.Directories,
			},
		}
	}

	if g.diagnostics != nil {
		g.diagnostics.EndProgress(true, "")
		g.diagnostics.Debug("Resolved module name: %s", moduleName)
		g.diagnostics.StartProgress("Scanning directories for Go packages")
	} else if config.Verbose {
		fmt.Printf("Resolved module name: %s\n", moduleName)
		fmt.Printf("Scanning directories for Go packages...\n")
	}
	packageDirs, err := g.scanner.ScanDirectories(config.Directories)
	if err != nil {
		if g.diagnostics != nil {
			g.diagnostics.EndProgress(false, "")
			g.diagnostics.Error("Failed to scan directories: %v", err)
		}
		return &models.GeneratorError{
			Type:    models.ErrorTypeFileSystem,
			Message: fmt.Sprintf("Failed to scan directories: %v", err),
			Suggestions: []string{
				"Check that the specified directories exist",
				"Ensure you have read permissions for the directories",
				"Verify the directory paths are correct",
			},
			Context: map[string]interface{}{
				"directories": config.Directories,
			},
		}
	}

	if len(packageDirs) == 0 {
		if g.diagnostics != nil {
			g.diagnostics.EndProgress(false, "")
			g.diagnostics.Warn("No Go packages found in specified directories")
		}
		return &models.GeneratorError{
			Type:    models.ErrorTypeValidation,
			Message: "No Go packages found in specified directories",
			Suggestions: []string{
				"Ensure the directories contain Go files",
				"Check that Go files have valid package declarations",
				"Try scanning parent directories or use './...' pattern",
			},
			Context: map[string]interface{}{
				"directories": config.Directories,
			},
		}
	}

	if g.diagnostics != nil {
		g.diagnostics.EndProgress(true, "")
		g.diagnostics.Info("Found %d packages to process", len(packageDirs))
		g.diagnostics.Indent()
		for _, dir := range packageDirs {
			g.diagnostics.List("%s", dir)
		}
		g.diagnostics.Unindent()
	} else {
		fmt.Printf("Found %d packages to process\n", len(packageDirs))
		if config.Verbose {
			fmt.Printf("Package directories:\n")
			for i, dir := range packageDirs {
				fmt.Printf("  %d. %s\n", i+1, dir)
			}
			fmt.Printf("\n")
		}
	}
	
	g.summary.PackagesProcessed = len(packageDirs)

	// First pass: Discover all parsers across packages
	if g.diagnostics != nil {
		g.diagnostics.Subsection("Parser Discovery Phase")
		g.diagnostics.StartProgress("Discovering parsers across all packages")
	} else {
		fmt.Printf("Discovering parsers across all packages...\n")
		if config.Verbose {
			fmt.Printf("Phase 1: Parser discovery (validation disabled)\n")
		}
	}
	
	// Skip parser and middleware validation during discovery phase
	g.parser.SetSkipParserValidation(true)
	g.parser.SetSkipMiddlewareValidation(true)
	
	var allPackageMetadata []*models.PackageMetadata
	for i, packageDir := range packageDirs {
		if config.Verbose {
			fmt.Printf("  Parsing package %d/%d: %s\n", i+1, len(packageDirs), packageDir)
		}
		
		// Parse the package
		metadata, err := g.parser.ParseDirectory(packageDir)
		if err != nil {
			// Enhance error with context
			if genErr, ok := err.(*models.GeneratorError); ok {
				genErr.Context = map[string]interface{}{
					"package_directory": packageDir,
					"module_name":      moduleName,
				}
				return genErr
			}
			return &models.GeneratorError{
				Type:    models.ErrorTypeValidation,
				Message: fmt.Sprintf("Failed to parse package %s: %v", packageDir, err),
				Suggestions: []string{
					"Check for syntax errors in Go files",
					"Ensure all files have valid package declarations",
					"Verify annotation syntax is correct",
				},
				Context: map[string]interface{}{
					"package_directory": packageDir,
					"module_name":      moduleName,
				},
			}
		}

		// Store metadata for second pass
		allPackageMetadata = append(allPackageMetadata, metadata)
		
		// Collect summary information
		g.collectSummaryInfo(metadata)
		
		if config.Verbose {
			fmt.Printf("    Found: %d controllers, %d services, %d middlewares, %d parsers\n", 
				len(metadata.Controllers), 
				len(metadata.CoreServices)+len(metadata.Loggers), 
				len(metadata.Middlewares),
				len(metadata.RouteParsers))
		}

		// Collect parsers from this package
		err = g.collectParsersFromPackage(metadata, moduleName, packageDir)
		if err != nil {
			return err // This already returns a GeneratorError
		}
		
		// Collect middleware from this package
		err = g.collectMiddlewareFromPackage(metadata, moduleName, packageDir)
		if err != nil {
			return err // This already returns a GeneratorError
		}
	}
	
	// Re-enable parser and middleware validation for second pass
	g.parser.SetSkipParserValidation(false)
	g.parser.SetSkipMiddlewareValidation(false)

	// Build global parser registry and register with code generator
	err = g.buildGlobalParserRegistry()
	if err != nil {
		return fmt.Errorf("failed to build global parser registry: %w", err)
	}

	fmt.Printf("Discovered %d parsers across all packages\n", len(g.globalParsers))
	g.summary.ParsersDiscovered = len(g.globalParsers)
	
	fmt.Printf("Discovered %d middleware across all packages\n", len(g.globalMiddleware))
	g.summary.MiddlewaresFound = len(g.globalMiddleware)

	// Validate custom parsers across all packages using global registry
	fmt.Printf("Validating custom parsers across all packages...\n")
	
	if config.Verbose {
		fmt.Printf("Phase 2: Parser validation (validation enabled)\n")
	}
	for _, metadata := range allPackageMetadata {
		err = g.parser.ValidateCustomParsersWithRegistry(metadata, g.globalParsers)
		if err != nil {
			// Enhance error with context
			if genErr, ok := err.(*models.GeneratorError); ok {
				if genErr.Context == nil {
					genErr.Context = make(map[string]interface{})
				}
				genErr.Context["package_name"] = metadata.PackageName
				genErr.Context["package_path"] = metadata.PackagePath
				return genErr
			}
			return &models.GeneratorError{
				Type:    models.ErrorTypeParserValidation,
				Message: fmt.Sprintf("Parser validation failed for package %s: %v", metadata.PackageName, err),
				Context: map[string]interface{}{
					"package_name": metadata.PackageName,
					"package_path": metadata.PackagePath,
				},
			}
		}
	}

	// Validate middleware references across all packages using global registry
	fmt.Printf("Validating middleware references across all packages...\n")
	
	if config.Verbose {
		fmt.Printf("Phase 2.5: Middleware validation\n")
	}
	for _, metadata := range allPackageMetadata {
		err = g.validateMiddlewareReferences(metadata)
		if err != nil {
			return err // This already returns a GeneratorError
		}
	}

	// Second pass: Generate code with global parser registry
	if config.Verbose {
		fmt.Printf("Phase 3: Code generation\n")
	}
	
	// Build package path mappings for all discovered packages
	packagePathMappings := make(map[string]string)
	for i, metadata := range allPackageMetadata {
		packageDir := packageDirs[i]
		if moduleName != "" {
			packageImportPath, err := g.moduleResolver.BuildPackagePath(moduleName, packageDir)
			if err == nil {
				packagePathMappings[metadata.PackageName] = packageImportPath
			}
		}
	}

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

		// Generate module for this package with package path mappings
		generatedModule, err := g.codeGenerator.GenerateModuleWithPackagePaths(metadata, moduleName, packagePathMappings)
		if err != nil {
			return &models.GeneratorError{
				Type:    models.ErrorTypeGeneration,
				Message: fmt.Sprintf("Failed to generate module for package %s: %v", metadata.PackageName, err),
				Suggestions: []string{
					"Check that all annotations are valid",
					"Ensure all dependencies are properly defined",
					"Verify that all referenced types exist",
				},
				Context: map[string]interface{}{
					"package_name": metadata.PackageName,
					"package_path": packageDir,
					"module_name":  moduleName,
				},
			}
		}

		// Write the generated module file
		err = g.writeModuleFile(generatedModule)
		if err != nil {
			return &models.GeneratorError{
				Type:    models.ErrorTypeFileSystem,
				Message: fmt.Sprintf("Failed to write module file for package %s: %v", metadata.PackageName, err),
				Suggestions: []string{
					"Check write permissions for the target directory",
					"Ensure the target directory exists",
					"Verify there's enough disk space",
				},
				Context: map[string]interface{}{
					"package_name": metadata.PackageName,
					"file_path":    generatedModule.FilePath,
				},
			}
		}

		// Add to modules list for main.go generation
		allModules = append(allModules, models.ModuleReference{
			PackageName: metadata.PackageName,
			PackagePath: packageImportPath,
			ModuleName:  "AutogenModule",
		})

		fmt.Printf("  Generated module: %s\n", generatedModule.FilePath)
		g.summary.ModulesGenerated++
		g.summary.GeneratedFiles = append(g.summary.GeneratedFiles, generatedModule.FilePath)
	}

	// Users control their own main.go files - we just generate the modules
	
	if config.Verbose {
		duration := time.Since(startTime)
		fmt.Printf("\nGeneration completed in %v\n", duration)
		fmt.Printf("Total files processed: %d\n", len(packageDirs))
		fmt.Printf("Total modules generated: %d\n", g.summary.ModulesGenerated)
	}

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
			conflicts := []models.ParserConflict{
				{
					FileName:     existing.FileName,
					Line:         existing.Line,
					FunctionName: existing.FunctionName,
					PackagePath:  existing.PackagePath,
				},
				{
					FileName:     parser.FileName,
					Line:         parser.Line,
					FunctionName: parser.FunctionName,
					PackagePath:  packageDir,
				},
			}
			return models.NewParserConflictError(parser.TypeName, conflicts)
		}

		// Add to global registry
		g.globalParsers[parser.TypeName] = enhancedParser
		fmt.Printf("  Discovered parser: %s -> %s (%s)\n", parser.TypeName, parser.FunctionName, importPath)
	}

	return nil
}

// collectMiddlewareFromPackage collects all middleware from a package and adds them to the global registry
func (g *Generator) collectMiddlewareFromPackage(metadata *models.PackageMetadata, moduleName, packageDir string) error {
	for _, middleware := range metadata.Middlewares {
		// Check for conflicts
		if existing, exists := g.globalMiddleware[middleware.Name]; exists {
			return &models.GeneratorError{
				Type:    models.ErrorTypeValidation,
				Message: fmt.Sprintf("Middleware name conflict: '%s' is defined in multiple packages", middleware.Name),
				Suggestions: []string{
					"Rename one of the conflicting middleware classes",
					"Ensure middleware names are unique across packages",
				},
				Context: map[string]interface{}{
					"middleware_name":     middleware.Name,
					"existing_package":    existing.PackagePath,
					"conflicting_package": packageDir,
				},
			}
		}

		// Create enhanced middleware metadata with package path
		enhancedMiddleware := middleware
		enhancedMiddleware.PackagePath = packageDir

		// Add to global registry
		g.globalMiddleware[middleware.Name] = enhancedMiddleware
		fmt.Printf("  Discovered middleware: %s (%s)\n", middleware.Name, packageDir)
	}

	return nil
}

// validateMiddlewareReferences validates that all middleware references in routes exist in the global registry
func (g *Generator) validateMiddlewareReferences(metadata *models.PackageMetadata) error {
	for _, controller := range metadata.Controllers {
		for _, route := range controller.Routes {
			for _, middlewareName := range route.Middlewares {
				if _, exists := g.globalMiddleware[middlewareName]; !exists {
					return &models.GeneratorError{
						Type:    models.ErrorTypeValidation,
						Message: fmt.Sprintf("Route %s.%s references unknown middleware: %s", controller.Name, route.HandlerName, middlewareName),
						Suggestions: []string{
							fmt.Sprintf("Check that middleware '%s' is defined with //axon::middleware annotation", middlewareName),
							"Ensure the middleware package is included in the scan directories",
							"Verify the middleware name matches exactly (case-sensitive)",
						},
						Context: map[string]interface{}{
							"route":           fmt.Sprintf("%s.%s", controller.Name, route.HandlerName),
							"middleware_name": middlewareName,
							"available_middleware": g.getAvailableMiddlewareNames(),
						},
					}
				}
			}
		}
	}
	return nil
}

// getAvailableMiddlewareNames returns a list of available middleware names for error reporting
func (g *Generator) getAvailableMiddlewareNames() []string {
	var names []string
	for name := range g.globalMiddleware {
		names = append(names, name)
	}
	return names
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

// collectSummaryInfo collects summary information from package metadata
func (g *Generator) collectSummaryInfo(metadata *models.PackageMetadata) {
	g.summary.ControllersFound += len(metadata.Controllers)
	g.summary.ServicesFound += len(metadata.CoreServices) + len(metadata.Loggers)
	g.summary.MiddlewaresFound += len(metadata.Middlewares)
}

// ReportSuccess reports successful generation using the diagnostic reporter
func (g *Generator) ReportSuccess() {
	g.reporter.ReportSuccess(g.summary)
}