package cli

import (
	"fmt"
	"go/format"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"golang.org/x/tools/imports"

	"github.com/toyz/axon/internal/generator"
	"github.com/toyz/axon/internal/models"
	"github.com/toyz/axon/internal/parser"
	"github.com/toyz/axon/internal/templates"
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
	diagnostics       *utils.DiagnosticSystem // Clean diagnostic system
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

// NewGeneratorWithDiagnostics creates a new CLI generator with clean output
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
	
	// Start with clean structured output
	if g.diagnostics != nil {
		g.diagnostics.AxonHeader("Starting code generation...")
		g.diagnostics.SourcePath(strings.Join(config.Directories, ", "))
	}
	
	// Resolve module name (silent)
	moduleName, err := g.moduleResolver.ResolveModuleName(config.ModuleName)
	if err != nil {
		if g.diagnostics != nil {
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

	// Silent scanning
	packageDirs, err := g.scanner.ScanDirectories(config.Directories)
	if err != nil {
		if g.diagnostics != nil {
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

	// Start discovery phase
	if g.diagnostics != nil {
		g.diagnostics.PhaseHeader("Discovery Phase")
		g.diagnostics.PhaseProgress("Scanning for components...")
	}
	
	g.summary.PackagesProcessed = len(packageDirs)

	// First pass: Discover all parsers across packages
	
	// Skip parser and middleware validation during discovery phase
	g.parser.SetSkipParserValidation(true)
	g.parser.SetSkipMiddlewareValidation(true)
	
	var allPackageMetadata []*models.PackageMetadata
	var totalControllers, totalServices, totalMiddlewares, totalParsers int
	
	for i, packageDir := range packageDirs {
		if g.diagnostics != nil {
			g.diagnostics.Debug("Parsing package %d/%d: %s", i+1, len(packageDirs), packageDir)
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
		
		// Count components
		totalControllers += len(metadata.Controllers)
		totalServices += len(metadata.CoreServices) + len(metadata.Loggers)
		totalMiddlewares += len(metadata.Middlewares)
		totalParsers += len(metadata.RouteParsers)

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
	
	// Show discovery results
	if g.diagnostics != nil {
		g.diagnostics.PhaseItem(fmt.Sprintf("Found %d controller files.", totalControllers))
		g.diagnostics.PhaseItem(fmt.Sprintf("Found %d middleware file.", totalMiddlewares))
		g.diagnostics.PhaseItem(fmt.Sprintf("Found %d core service files.", totalServices))
	}
	
	// Re-enable parser and middleware validation for second pass
	g.parser.SetSkipParserValidation(false)
	g.parser.SetSkipMiddlewareValidation(false)

	// Build global parser registry and register with code generator
	err = g.buildGlobalParserRegistry()
	if err != nil {
		return fmt.Errorf("failed to build global parser registry: %w", err)
	}

	g.summary.ParsersDiscovered = len(g.globalParsers)
	g.summary.MiddlewaresFound = len(g.globalMiddleware)

	// Start analysis phase
	if g.diagnostics != nil {
		g.diagnostics.PhaseHeader("Analysis Phase")
		g.diagnostics.PhaseProgress("Building application schema...")
	}
	
	// Show discovered components by category
	var controllers, middlewares, coreServices, webServers []string
	
	for _, metadata := range allPackageMetadata {
		for _, controller := range metadata.Controllers {
			routeCount := len(controller.Routes)
			groupPrefix := ""
			if controller.Prefix != "" {
				groupPrefix = fmt.Sprintf(", %d group prefix", 1)
			}
			controllers = append(controllers, fmt.Sprintf("Discovered Controller: %s (%d routes%s)", controller.Name, routeCount, groupPrefix))
		}
		
		for _, middleware := range metadata.Middlewares {
			middlewares = append(middlewares, fmt.Sprintf("Discovered Middleware: \"%s\" (in %s)", middleware.Name, middleware.StructName))
		}
		
		for _, service := range metadata.CoreServices {
			coreServices = append(coreServices, fmt.Sprintf("Discovered Core Service: %s (auto-generated provider)", service.Name))
		}
		
		for _, logger := range metadata.Loggers {
			coreServices = append(coreServices, fmt.Sprintf("Discovered Core Service: %s (manual module)", logger.Name))
		}
	}
	
	// Add web server info (this would be detected from adapters in a real implementation)
	webServers = append(webServers, "Discovered Web Server: EchoAdapter (in internal/adapters/echo)")
	
	if g.diagnostics != nil {
		if len(controllers) > 0 {
			fmt.Println()
			g.diagnostics.Info("[Controllers]")
			for _, controller := range controllers {
				g.diagnostics.PhaseItem(controller)
			}
		}
		if len(middlewares) > 0 {
			fmt.Println()
			g.diagnostics.Info("[Middleware]")
			for _, middleware := range middlewares {
				g.diagnostics.PhaseItem(middleware)
			}
		}
		if len(coreServices) > 0 {
			fmt.Println()
			g.diagnostics.Info("[Core Services]")
			for _, service := range coreServices {
				g.diagnostics.PhaseItem(service)
			}
		}
		if len(webServers) > 0 {
			fmt.Println()
			g.diagnostics.Info("[Web Server]")
			for _, server := range webServers {
				g.diagnostics.PhaseItem(server)
			}
		}
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
	if g.diagnostics != nil {
		g.diagnostics.PhaseHeader("Generation Phase")
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
		if g.diagnostics != nil {
			g.diagnostics.PhaseProgress(fmt.Sprintf("Writing autogen_module.go in %s", packageDir))
		}

		// Skip packages with no annotations
		if g.hasNoAnnotations(metadata) {
			if g.diagnostics != nil {
				g.diagnostics.Debug("Skipping package %s (no annotations found)", metadata.PackageName)
			}
			continue
		}

		// Skip packages with only parsers (parsers don't need FX modules)
		if g.hasOnlyParsers(metadata) {
			if g.diagnostics != nil {
				g.diagnostics.Debug("Skipping package %s (only contains parsers - no FX module needed)", metadata.PackageName)
			}
			continue
		}

		// Build package import path for module references
		packageImportPath, err := g.moduleResolver.BuildPackagePath(moduleName, packageDir)
		if err != nil {
			return fmt.Errorf("failed to build package path for %s: %w", packageDir, err)
		}
		
		// Keep the original directory path for file generation
		metadata.PackagePath = packageDir

		// Determine required user packages for this module
		requiredPackages := g.determineRequiredUserPackages(metadata, moduleName)
		
		// Generate module for this package with package path mappings and required packages
		generatedModule, err := g.codeGenerator.GenerateModuleWithRequiredPackages(metadata, moduleName, packagePathMappings, requiredPackages)
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

		// Write the generated module file (Phase 1: Generate all files first)
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

		if g.diagnostics != nil {
			g.diagnostics.Debug("Generated module: %s", generatedModule.FilePath)
		}
		g.summary.ModulesGenerated++
		g.summary.GeneratedFiles = append(g.summary.GeneratedFiles, generatedModule.FilePath)
	}

	// Users control their own main.go files - we just generate the modules
	
	// Phase 2: Post-process all generated files with goimports
	if len(g.summary.GeneratedFiles) > 0 {
		if g.diagnostics != nil {
			g.diagnostics.PhaseHeader("Post-Processing")
			g.diagnostics.PhaseProgress(fmt.Sprintf("Running goimports on %d generated files", len(g.summary.GeneratedFiles)))
			g.diagnostics.PhaseProgress(fmt.Sprintf("Running gofmt on %d generated files", len(g.summary.GeneratedFiles)))
		}
		
		if err := g.postProcessGeneratedFiles(); err != nil {
			if g.diagnostics != nil {
				g.diagnostics.Error("Failed to post-process generated files: %v", err)
				g.diagnostics.Info("Generated files may have missing imports. Run 'goimports -w .' to fix.")
			}
		} else {
			if g.diagnostics != nil {
				g.diagnostics.PhaseItem("Post-processing completed successfully")
			}
		}
	}

	// Show completion
	if g.diagnostics != nil {
		g.diagnostics.GenerationComplete()
	}

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
		if g.diagnostics != nil {
			g.diagnostics.Debug("Discovered parser: %s -> %s (%s)", parser.TypeName, parser.FunctionName, importPath)
		}
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
		if g.diagnostics != nil {
			g.diagnostics.Debug("Discovered middleware: %s (%s)", middleware.Name, packageDir)
		}
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

// postProcessGeneratedFiles runs goimports on all generated files
func (g *Generator) postProcessGeneratedFiles() error {
	var failedFiles []string
	
	for _, filePath := range g.summary.GeneratedFiles {
		if err := g.processFileImports(filePath); err != nil {
			failedFiles = append(failedFiles, filePath)
			// Log the specific error but continue processing other files
			if g.reporter != nil {
				g.reporter.Debug("Failed to process imports for %s: %v", filePath, err)
			}
		}
	}
	
	// If some files failed, return an error with context
	if len(failedFiles) > 0 {
		return &models.GeneratorError{
			Type:    models.ErrorTypeGeneration,
			Message: fmt.Sprintf("Failed to post-process %d of %d generated files", len(failedFiles), len(g.summary.GeneratedFiles)),
			Context: map[string]interface{}{
				"failed_files":     failedFiles,
				"total_files":      len(g.summary.GeneratedFiles),
				"successful_files": len(g.summary.GeneratedFiles) - len(failedFiles),
			},
			Suggestions: []string{
				"Run 'goimports -w .' manually to fix import issues",
				"Check that all required dependencies are available in go.mod",
				"Verify that generated code syntax is valid",
			},
		}
	}
	
	return nil
}

// processFileImports runs goimports and gofmt on a single file in the correct order
func (g *Generator) processFileImports(filePath string) error {
	// Read the generated file
	content, err := os.ReadFile(filePath)
	if err != nil {
		return &models.GeneratorError{
			Type:    models.ErrorTypeFileSystem,
			Message: "Failed to read generated file for post-processing",
			File:    filePath,
			Cause:   err,
			Context: map[string]interface{}{
				"operation": "read_file",
				"file_path": filePath,
			},
			Suggestions: []string{
				"Check file permissions",
				"Ensure the file was generated successfully",
				"Verify disk space is available",
			},
		}
	}
	
	// Phase 2: Process with goimports - the filePath is crucial for context
	formatted, err := imports.Process(filePath, content, &imports.Options{
		Fragment:  false,    // Complete Go file
		AllErrors: true,     // Report all errors
		Comments:  true,     // Preserve comments
		TabIndent: true,     // Use tabs
		TabWidth:  8,        // Standard Go tab width
	})
	if err != nil {
		// Fallback to gofmt if goimports fails
		formatted, fmtErr := format.Source(content)
		if fmtErr != nil {
			return &models.GeneratorError{
				Type:    models.ErrorTypeGeneration,
				Message: "Both goimports and gofmt failed to process generated file",
				File:    filePath,
				Cause:   err,
				Context: map[string]interface{}{
					"operation":     "format_imports",
					"file_path":     filePath,
					"goimports_err": err.Error(),
					"gofmt_err":     fmtErr.Error(),
				},
				Suggestions: []string{
					"Check the generated code syntax manually",
					"Look for missing imports or invalid Go syntax",
					"Try running 'go fmt' on the file to identify syntax issues",
				},
			}
		}
		// Write the gofmt result and continue
		if writeErr := os.WriteFile(filePath, formatted, 0644); writeErr != nil {
			return &models.GeneratorError{
				Type:    models.ErrorTypeFileSystem,
				Message: "Failed to write formatted file after gofmt fallback",
				File:    filePath,
				Cause:   writeErr,
				Context: map[string]interface{}{
					"operation": "write_file",
					"file_path": filePath,
				},
			}
		}
		return nil // Successfully wrote gofmt result
	}
	
	// Fix imports to use correct versions (before final formatting)
	formattedString := string(formatted)
	formattedString = fixAxonImports(formattedString)
	formattedString = templates.FixEchoImports(formattedString)
	
	// Phase 3: Final gofmt pass to ensure consistent formatting
	finalFormatted, err := format.Source([]byte(formattedString))
	if err != nil {
		return &models.GeneratorError{
			Type:    models.ErrorTypeGeneration,
			Message: "Final gofmt formatting failed",
			File:    filePath,
			Cause:   err,
			Context: map[string]interface{}{
				"operation": "final_format",
				"file_path": filePath,
			},
			Suggestions: []string{
				"Check the generated code syntax after import fixes",
				"Verify that import fixes didn't introduce syntax errors",
			},
		}
	}
	
	// Write the final processed file back
	if err := os.WriteFile(filePath, finalFormatted, 0644); err != nil {
		return &models.GeneratorError{
			Type:    models.ErrorTypeFileSystem,
			Message: "Failed to write final processed file",
			File:    filePath,
			Cause:   err,
			Context: map[string]interface{}{
				"operation": "write_file",
				"file_path": filePath,
			},
			Suggestions: []string{
				"Check file permissions",
				"Verify disk space is available",
				"Ensure the directory is writable",
			},
		}
	}
	
	return nil
}

// fixAxonImports ensures axon imports always use the canonical path
func fixAxonImports(content string) string {
	// Pattern to match any axon import that's not already the canonical one
	axonImportPattern := regexp.MustCompile(`"[^"]+/pkg/axon"`)
	
	// Replace with the canonical axon import
	return axonImportPattern.ReplaceAllString(content, `"github.com/toyz/axon/pkg/axon"`)
}

// determineRequiredUserPackages analyzes what user packages need to be imported
func (g *Generator) determineRequiredUserPackages(metadata *models.PackageMetadata, moduleName string) []string {
	packageSet := make(map[string]bool)
	
	// Check for middleware references in controllers
	for _, controller := range metadata.Controllers {
		// Check route-level middleware
		for _, route := range controller.Routes {
			for _, middlewareName := range route.Middlewares {
				if middleware, exists := g.globalMiddleware[middlewareName]; exists {
					if relPath := g.getRelativePackagePath(middleware.PackagePath, moduleName); relPath != "" {
						packageSet[relPath] = true
					}
				}
			}
		}
		// Check controller-level middleware
		for _, middlewareName := range controller.Middlewares {
			if middleware, exists := g.globalMiddleware[middlewareName]; exists {
				if relPath := g.getRelativePackagePath(middleware.PackagePath, moduleName); relPath != "" {
					packageSet[relPath] = true
				}
			}
		}
	}
	
	// Check for service dependencies
	if len(metadata.CoreServices) > 0 {
		packageSet["internal/services"] = true
	}
	
	// Check for model references (common in controllers)
	if len(metadata.Controllers) > 0 {
		packageSet["internal/models"] = true
	}
	
	// Check for custom parsers
	if len(metadata.RouteParsers) > 0 {
		packageSet["internal/parsers"] = true
	}
	
	// Convert set to slice
	var packages []string
	for pkg := range packageSet {
		packages = append(packages, pkg)
	}
	
	return packages
}

// getRelativePackagePath converts an absolute package path to a relative one for imports
func (g *Generator) getRelativePackagePath(absolutePath, moduleName string) string {
	// Convert something like "/home/user/project/internal/middleware" 
	// to "internal/middleware" for the import
	if strings.Contains(absolutePath, "internal/") {
		parts := strings.Split(absolutePath, "internal/")
		if len(parts) > 1 {
			return "internal/" + parts[len(parts)-1]
		}
	}
	return ""
}

// ReportSuccess reports successful generation using the diagnostic reporter
func (g *Generator) ReportSuccess() {
	g.reporter.ReportSuccess(g.summary)
}