package parser

import (
	"fmt"
	"go/ast"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/toyz/axon/internal/annotations"
	"github.com/toyz/axon/internal/models"
	"github.com/toyz/axon/internal/registry"
	"github.com/toyz/axon/internal/utils"
	"github.com/toyz/axon/pkg/axon"
)

// Parser implements the AnnotationParser interface using the new annotations package
type Parser struct {
	fileReader              *utils.FileReader
	middlewareRegistry      registry.MiddlewareRegistry
	skipParserValidation    bool               // Skip custom parser validation during discovery phase
	skipMiddlewareValidation bool              // Skip middleware validation during discovery phase
	reporter                DiagnosticReporter // For debug logging and error reporting
	annotationParser        *annotations.ParticipleParser
	annotationRegistry      annotations.AnnotationRegistry
	goModParser             *utils.GoModParser
}

// DiagnosticReporter interface for debug logging (wrapper for new diagnostic system)
type DiagnosticReporter interface {
	Debug(format string, args ...interface{})
	DebugSection(section string)
}

// NewParser creates a new annotation parser using the new annotations package
func NewParser() *Parser {
	// Create a new annotations registry and register built-in schemas
	annotationRegistry := annotations.NewRegistry()
	if err := annotations.RegisterBuiltinSchemas(annotationRegistry); err != nil {
		panic(fmt.Sprintf("failed to register builtin annotation schemas: %v", err))
	}

	fileReader := utils.NewFileReader()
	return &Parser{
		fileReader:         fileReader,
		middlewareRegistry: registry.NewMiddlewareRegistry(),
		reporter:           &noOpReporter{}, // Default no-op reporter for backward compatibility
		annotationParser:   annotations.NewParticipleParser(annotationRegistry),
		annotationRegistry: annotationRegistry,
		goModParser:        utils.NewGoModParser(fileReader),
	}
}

// NewParserWithReporter creates a new annotation parser with a diagnostic reporter
func NewParserWithReporter(reporter DiagnosticReporter) *Parser {
	// Create a new annotations registry and register built-in schemas
	annotationRegistry := annotations.NewRegistry()
	if err := annotations.RegisterBuiltinSchemas(annotationRegistry); err != nil {
		panic(fmt.Sprintf("failed to register builtin annotation schemas: %v", err))
	}

	fileReader := utils.NewFileReader()
	return &Parser{
		fileReader:         fileReader,
		middlewareRegistry: registry.NewMiddlewareRegistry(),
		reporter:           reporter,
		annotationParser:   annotations.NewParticipleParser(annotationRegistry),
		annotationRegistry: annotationRegistry,
		goModParser:        utils.NewGoModParser(fileReader),
	}
}

// noOpReporter is a no-op implementation of DiagnosticReporter for backward compatibility
type noOpReporter struct{}

func (n *noOpReporter) Debug(format string, args ...interface{}) {}
func (n *noOpReporter) DebugSection(section string)              {}

// ParseSource parses source code from a string for testing purposes
func (p *Parser) ParseSource(filename, source string) (*models.PackageMetadata, error) {
	// Parse the source code
	file, err := p.fileReader.ParseGoSource(filename, source)
	if err != nil {
		return nil, err
	}

	// Create package metadata
	metadata := &models.PackageMetadata{
		PackageName:   file.Name.Name,
		PackagePath:   "./",
		SourceImports: make(map[string][]models.Import),
	}

	// Extract imports from the file
	imports := p.ExtractImports(file)
	metadata.SourceImports[filename] = imports

	// Extract annotations
	annotations, err := p.ExtractAnnotations(file, filename)
	if err != nil {
		return nil, fmt.Errorf("failed to extract annotations: %w", err)
	}

	// Create file map
	fileMap := map[string]*ast.File{
		filename: file,
	}

	// Process annotations
	err = p.processAnnotations(annotations, metadata, fileMap)
	if err != nil {
		return nil, fmt.Errorf("failed to process annotations: %w", err)
	}

	return metadata, nil
}

// ParseDirectory recursively scans the specified directory for .go files and extracts annotations
func (p *Parser) ParseDirectory(path string) (*models.PackageMetadata, error) {
	// Validate and sanitize the input path to prevent path traversal attacks
	if !isSecureDirectoryPath(path) {
		return nil, fmt.Errorf("invalid directory path: %s", path)
	}

	// Clean and normalize the path
	cleanPath := filepath.Clean(path)

	// Ensure the clean path doesn't escape the current working directory
	if strings.Contains(cleanPath, "..") {
		return nil, fmt.Errorf("path traversal not allowed: %s", path)
	}

	// Parse all Go files in the directory using cached FileReader
	files, packageName, err := p.parseDirectoryFiles(cleanPath)
	if err != nil {
		return nil, fmt.Errorf("failed to parse directory %s: %w", cleanPath, err)
	}

	// Create package metadata
	metadata := &models.PackageMetadata{
		PackageName:   packageName,
		PackagePath:   cleanPath,
		SourceImports: make(map[string][]models.Import),
	}

	// Detect module information
	if err := p.detectModuleInfo(metadata); err != nil {
		// Log warning but don't fail - module info is optional for some use cases
		p.reporter.Debug("Warning: failed to detect module info: %v", err)
	}

	// First pass: Extract all annotations and imports from all files
	allAnnotations := []models.Annotation{}
	fileMap := files // Use the cached files directly

	for fileName, file := range files {
		// Extract imports from this file
		imports := p.ExtractImports(file)
		metadata.SourceImports[fileName] = imports

		// Extract annotations from this file
		annotations, err := p.ExtractAnnotations(file, fileName)
		if err != nil {
			return nil, fmt.Errorf("failed to extract annotations from file %s: %w", fileName, err)
		}
		allAnnotations = append(allAnnotations, annotations...)
	}

	// Second pass: Process all annotations to build metadata structures
	err = p.processAnnotations(allAnnotations, metadata, fileMap)
	if err != nil {
		return nil, fmt.Errorf("failed to process annotations: %w", err)
	}

	// Third pass: Validate middleware Handle methods in their respective files
	for fileName, file := range fileMap {
		// Extract annotations for this specific file
		fileAnnotations, err := p.ExtractAnnotations(file, fileName)
		if err != nil {
			return nil, fmt.Errorf("failed to extract annotations from file %s: %w", fileName, err)
		}

		// Validate middleware Handle methods only for middlewares defined in this file
		for _, annotation := range fileAnnotations {
			if annotation.Type == models.AnnotationTypeMiddleware {
				err = p.ValidateMiddlewareHandleMethod(file, annotation.Target)
				if err != nil {
					return nil, fmt.Errorf("middleware validation failed in file %s: %w", fileName, err)
				}
			}
		}
	}

	return metadata, nil
}

// ExtractImports extracts import statements from a Go file
func (p *Parser) ExtractImports(file *ast.File) []models.Import {
	var imports []models.Import

	for _, importSpec := range file.Imports {
		imp := models.Import{
			Path: strings.Trim(importSpec.Path.Value, `"`), // Remove quotes
		}

		// Check for import alias
		if importSpec.Name != nil {
			imp.Alias = importSpec.Name.Name
		}

		imports = append(imports, imp)
	}

	return imports
}

// ExtractAnnotations traverses the AST and extracts axon:: annotations from comments using the new parser
func (p *Parser) ExtractAnnotations(file *ast.File, fileName string) ([]models.Annotation, error) {
	var annotations []models.Annotation

	// Walk the AST to find annotated declarations
	ast.Inspect(file, func(n ast.Node) bool {
		switch node := n.(type) {
		case *ast.GenDecl:
			// Handle struct declarations with annotations
			if node.Tok == token.TYPE {
				for _, spec := range node.Specs {
					if typeSpec, ok := spec.(*ast.TypeSpec); ok {
						if structType, ok := typeSpec.Type.(*ast.StructType); ok {
							// Extract annotations from comments
							if node.Doc != nil {
								for _, comment := range node.Doc.List {
									if annotation, err := p.parseAnnotationCommentWithFile(comment.Text, typeSpec.Name.Name, comment.Pos(), fileName); err == nil {
										// Extract dependencies for controller, middleware, and core service annotations
										if annotation.Type == models.AnnotationTypeController ||
											annotation.Type == models.AnnotationTypeMiddleware ||
											annotation.Type == models.AnnotationTypeCore ||
											annotation.Type == models.AnnotationTypeLogger {
											deps := p.extractDependencies(structType)
											annotation.Dependencies = deps
										}
										annotation.FileName = fileName
										annotations = append(annotations, annotation)
									}
								}
							}
						}
					}
				}
			}
		case *ast.FuncDecl:
			// Handle method declarations with annotations
			if node.Doc != nil {
				targetName := node.Name.Name
				// If it's a method, include receiver type
				if node.Recv != nil && len(node.Recv.List) > 0 {
					if starExpr, ok := node.Recv.List[0].Type.(*ast.StarExpr); ok {
						if ident, ok := starExpr.X.(*ast.Ident); ok {
							targetName = ident.Name + "." + node.Name.Name
						}
					}
				}

				for _, comment := range node.Doc.List {
					if annotation, err := p.parseAnnotationCommentWithFile(comment.Text, targetName, comment.Pos(), fileName); err == nil {
						annotation.FileName = fileName
						annotations = append(annotations, annotation)
					}
				}
			}
		}
		return true
	})

	return annotations, nil
}

// convertNewToOldAnnotation converts a new ParsedAnnotation to the old models.Annotation format
// createAnnotation creates a models.Annotation from a new ParsedAnnotation
func (p *Parser) createAnnotation(newAnnotation *annotations.ParsedAnnotation, target string) models.Annotation {
	return models.Annotation{
		ParsedAnnotation: newAnnotation,
		Dependencies:     []models.Dependency{}, // Will be populated later
		FileName:         newAnnotation.Location.File,
		Line:             newAnnotation.Location.Line,
	}
}

// SetSkipParserValidation controls whether custom parser validation is skipped
func (p *Parser) SetSkipParserValidation(skip bool) {
	p.skipParserValidation = skip
}

// SetSkipMiddlewareValidation controls whether middleware validation is skipped
func (p *Parser) SetSkipMiddlewareValidation(skip bool) {
	p.skipMiddlewareValidation = skip
}

// ValidateCustomParsersWithRegistry implements the AnnotationParser interface
func (p *Parser) ValidateCustomParsersWithRegistry(metadata *models.PackageMetadata, parserRegistry map[string]axon.RouteParserMetadata) error {
	// This method validates custom parsers - for now, return nil as the new parser handles validation internally
	return nil
}

// processAnnotations builds metadata structures from parsed annotations
func (p *Parser) processAnnotations(annotations []models.Annotation, metadata *models.PackageMetadata, fileMap map[string]*ast.File) error {
	// First pass: collect all controllers, interface annotations, and register middlewares
	controllerNames := make(map[string]bool)
	interfaceTargets := make(map[string]bool)

	for _, annotation := range annotations {
		if annotation.Type == models.AnnotationTypeController {
			controllerNames[annotation.Target] = true
		}
		if annotation.Type == models.AnnotationTypeInterface {
			interfaceTargets[annotation.Target] = true
		}
		if annotation.Type == models.AnnotationTypeMiddleware {
			// Register middleware in the registry for validation
			middleware := &models.MiddlewareMetadata{
				Name:         annotation.GetString("Name"),
				PackagePath:  metadata.PackagePath,
				StructName:   annotation.Target,
				Dependencies: annotation.Dependencies,
				Parameters:   annotation.Parameters,
				IsGlobal:     annotation.HasParameter("Global"),
				Priority:     annotation.GetInt("Priority", 100), // Default priority 100
			}

			err := p.middlewareRegistry.Register(middleware.Name, middleware)
			if err != nil {
				return fmt.Errorf("failed to register middleware '%s': %w", middleware.Name, err)
			}
		}
	}

	// Second pass: process all annotations and validate routes
	for _, annotation := range annotations {
		switch annotation.Type {
		case models.AnnotationTypeController:
			controller := models.ControllerMetadata{
				Name:         annotation.Target,
				StructName:   annotation.Target,
				Prefix:       annotation.GetString("Prefix", ""),
				Middlewares:  annotation.GetStringSlice("Middleware"),
				Dependencies: annotation.Dependencies,
			}
			metadata.Controllers = append(metadata.Controllers, controller)

			// If this controller also has an interface annotation, generate interface
			if interfaceTargets[annotation.Target] {
				// Extract methods from the struct
				methods, err := p.extractPublicMethods(fileMap[annotation.FileName], annotation.Target)
				if err != nil {
					return fmt.Errorf("failed to extract methods for interface %s: %w", annotation.Target, err)
				}

				iface := models.InterfaceMetadata{
					Name:         annotation.Target + "Interface",
					StructName:   annotation.Target,
					PackagePath:  metadata.PackagePath,
					Methods:      methods,
					Dependencies: annotation.Dependencies,
				}
				metadata.Interfaces = append(metadata.Interfaces, iface)
			}

		case models.AnnotationTypeRoute:
			// Validate that route is on a controller-annotated struct
			parts := strings.Split(annotation.Target, ".")
			if len(parts) != 2 {
				return fmt.Errorf("invalid route target format: %s (expected ControllerName.MethodName)", annotation.Target)
			}

			controllerName := parts[0]
			methodName := parts[1]
			if !controllerNames[controllerName] {
				return fmt.Errorf("route %s is defined on struct %s which is not annotated with //axon::controller", annotation.Target, controllerName)
			}

			// Routes will be associated with controllers in a later processing step
			// For now, we'll store them temporarily
			route := models.RouteMetadata{
				Method:      annotation.GetString("method"),
				Path:        annotation.GetString("path"),
				HandlerName: annotation.Target, // Keep full target for now, will be processed later
			}

			// Parse path parameters from the route path
			pathParams, err := p.parsePathParameters(route.Path)
			if err != nil {
				return fmt.Errorf("failed to parse path parameters for route %s: %w", annotation.Target, err)
			}

			// Analyze handler method signature to detect all parameters
			var allParams []models.Parameter
			if file := fileMap[annotation.FileName]; file != nil {
				signatureParams, err := p.analyzeHandlerSignature(file, controllerName, methodName)
				if err != nil {
					return fmt.Errorf("failed to analyze handler signature for %s: %w", annotation.Target, err)
				}

				// Merge path parameters with signature parameters
				allParams = p.mergeParameters(pathParams, signatureParams)
			} else {
				// If no file available, just use path parameters
				allParams = pathParams
			}

			route.Parameters = allParams

			// Analyze return type
			if file := fileMap[annotation.FileName]; file != nil {
				returnType, err := p.analyzeReturnType(file, controllerName, methodName)
				if err != nil {
					return fmt.Errorf("failed to analyze return type for %s: %w", annotation.Target, err)
				}

				// Map string return type to enum
				var returnTypeEnum models.ReturnType
				switch returnType {
				case "error":
					returnTypeEnum = models.ReturnTypeError
				default:
					returnTypeEnum = models.ReturnTypeDataError // Default assumption
				}

				route.ReturnType = models.ReturnTypeInfo{Type: returnTypeEnum}
			}

			// Parse middleware and validate
			middlewareNames := annotation.GetStringSlice("Middleware")
			if len(middlewareNames) > 0 {
				// Validate that all middleware names exist in the registry (skip during discovery phase)
				if !p.skipMiddlewareValidation {
					err := p.middlewareRegistry.Validate(middlewareNames)
					if err != nil {
						return fmt.Errorf("route %s has invalid middleware reference: %w", annotation.Target, err)
					}
				}

				route.Middlewares = middlewareNames
			}

			// Flags are now handled through parameters

			// Find the controller this route belongs to and add it
			p.addRouteToController(route, metadata)

		case models.AnnotationTypeMiddleware:
			middleware := models.MiddlewareMetadata{
				Name:         annotation.GetString("Name"),
				PackagePath:  metadata.PackagePath,
				StructName:   annotation.Target,
				Dependencies: annotation.Dependencies,
				Parameters:   annotation.Parameters,
				IsGlobal:     annotation.HasParameter("Global"),
				Priority:     annotation.GetInt("Priority", 100), // Default priority 100
			}
			metadata.Middlewares = append(metadata.Middlewares, middleware)

		case models.AnnotationTypeCore:
			service := models.CoreServiceMetadata{
				Name:         annotation.Target,
				StructName:   annotation.Target,
				Dependencies: annotation.Dependencies,
			}

			// Check for Init parameter (now stored in Parameters map)
			hasInitParam := annotation.HasParameter("Init")

			if hasInitParam {
				// Enable lifecycle when Init parameter is present
				service.HasLifecycle = true
				initMode := annotation.GetString("Init", "Same")
				service.StartMode = initMode

				// Detect Start and Stop methods when lifecycle is enabled
				file := fileMap[annotation.FileName]
				if file != nil {
					hasStart, hasStop := p.extractLifecycleMethods(file, annotation.Target)
					service.HasStart = hasStart
					service.HasStop = hasStop

					// Validate that Start method exists when Init is used
					if !hasStart {
						return fmt.Errorf("service %s has Init parameter but missing Start(context.Context) error method", annotation.Target)
					}
				} else {
					// If file is not available (e.g., in unit tests), skip method detection
					service.HasStart = true // Assume valid for unit tests
					service.HasStop = false // Default to no Stop method
				}
			} else {
				// Default Init mode - no lifecycle required
				service.HasLifecycle = false
				service.StartMode = "Same"
			}

			// Check for Manual parameter
			manualModule := annotation.GetString("Manual", "")
			if manualModule != "" {
				service.IsManual = true
				service.ModuleName = manualModule
			}

			// Check for Mode parameter (default to Singleton)
			mode := annotation.GetString("Mode", LifecycleModeSingleton)
			if mode == LifecycleModeTransient || mode == LifecycleModeSingleton {
				service.Mode = mode
			} else {
				return fmt.Errorf("service %s has invalid mode '%s': must be 'Singleton' or 'Transient'", annotation.Target, mode)
			}

			metadata.CoreServices = append(metadata.CoreServices, service)

			// If this core service also has an interface annotation, generate interface
			if interfaceTargets[annotation.Target] {
				// Extract methods from the struct
				methods, err := p.extractPublicMethods(fileMap[annotation.FileName], annotation.Target)
				if err != nil {
					return fmt.Errorf("failed to extract methods for interface %s: %w", annotation.Target, err)
				}

				iface := models.InterfaceMetadata{
					Name:         annotation.Target + "Interface",
					StructName:   annotation.Target,
					PackagePath:  metadata.PackagePath,
					Methods:      methods,
					Dependencies: annotation.Dependencies,
				}
				metadata.Interfaces = append(metadata.Interfaces, iface)
			}

		case models.AnnotationTypeLogger:
			logger := models.LoggerMetadata{
				Name:         annotation.Target,
				StructName:   annotation.Target,
				Dependencies: annotation.Dependencies,
			}

			// Check for Init parameter (now stored in Parameters map)
			hasInitParam := annotation.HasParameter("Init")

			if hasInitParam {
				// Enable lifecycle when Init parameter is present
				logger.HasLifecycle = true
				// Detect Start and Stop methods when lifecycle is enabled
				file := fileMap[annotation.FileName]
				if file != nil {
					hasStart, hasStop := p.extractLifecycleMethods(file, annotation.Target)
					logger.HasStart = hasStart
					logger.HasStop = hasStop

					// Validate that Start method exists when Init is used
					if !hasStart {
						return fmt.Errorf("logger %s has Init parameter but missing Start(context.Context) error method", annotation.Target)
					}
				} else {
					// If file is not available (e.g., in unit tests), skip method detection
					logger.HasStart = true // Assume valid for unit tests
					logger.HasStop = false // Default to no Stop method
				}
			} else {
				// Default Init mode - no lifecycle required
				logger.HasLifecycle = false
			}

			// Check for Manual parameter
			manualModule := annotation.GetString("Manual", "")
			if manualModule != "" {
				logger.IsManual = true
				logger.ModuleName = manualModule
			}

			metadata.Loggers = append(metadata.Loggers, logger)

		case models.AnnotationTypeRouteParser:
			// Route parser annotations should be on function declarations
			typeName := annotation.GetString("name")

			// Validate that this is actually on a function and has correct signature
			file := fileMap[annotation.FileName]
			if file != nil {
				err := p.ValidateParserFunctionSignature(file, annotation.Target, typeName)
				if err != nil {
					return fmt.Errorf("parser function validation failed for %s: %w", annotation.Target, err)
				}
			}

			parser := axon.RouteParserMetadata{
				TypeName:     typeName,
				FunctionName: annotation.Target,
				PackagePath:  metadata.PackagePath,
				FileName:     annotation.FileName,
				Line:         annotation.Line,
			}

			// Extract function signature information for validation
			if file != nil {
				paramTypes, returnTypes, err := p.extractParserSignature(file, annotation.Target)
				if err != nil {
					return fmt.Errorf("failed to extract parser signature for %s: %w", annotation.Target, err)
				}
				parser.ParameterTypes = paramTypes
				parser.ReturnTypes = returnTypes
			}

			metadata.RouteParsers = append(metadata.RouteParsers, parser)
		}
	}

	return nil
}

// detectModuleInfo detects module path and root from go.mod file
func (p *Parser) detectModuleInfo(metadata *models.PackageMetadata) error {
	// Find go.mod file starting from package path
	moduleRoot, modulePath, err := p.findModuleInfo(metadata.PackagePath)
	if err != nil {
		return err
	}

	metadata.ModuleRoot = moduleRoot
	metadata.ModulePath = modulePath

	// Calculate package import path
	packageImportPath, err := p.calculatePackageImportPath(moduleRoot, modulePath, metadata.PackagePath)
	if err != nil {
		return err
	}

	metadata.PackageImportPath = packageImportPath
	return nil
}

// findModuleInfo searches for go.mod file and extracts module information
func (p *Parser) findModuleInfo(startPath string) (moduleRoot, modulePath string, err error) {
	// Validate and sanitize the input path
	if !isSecureDirectoryPath(startPath) {
		return "", "", fmt.Errorf("invalid start path: %s", startPath)
	}

	// Clean and normalize the path
	currentDir := filepath.Clean(startPath)

	// Ensure the clean path doesn't contain path traversal attempts
	if strings.Contains(currentDir, "..") {
		return "", "", fmt.Errorf("path traversal not allowed in start path: %s", startPath)
	}

	if !filepath.IsAbs(currentDir) {
		currentDir, err = filepath.Abs(currentDir)
		if err != nil {
			return "", "", fmt.Errorf("failed to get absolute path: %w", err)
		}
	}

	for {
		// Safely construct the go.mod path
		goModPath := filepath.Join(currentDir, "go.mod")

		// Additional validation to ensure we're not accessing unexpected files
		if !strings.HasSuffix(goModPath, "go.mod") {
			return "", "", fmt.Errorf("invalid go.mod path construction")
		}

		if _, err := os.Stat(goModPath); err == nil {
			// Found go.mod file - validate it's actually a go.mod file
			modulePath, err := p.parseGoModFile(goModPath)
			if err != nil {
				return "", "", fmt.Errorf("failed to parse go.mod: %w", err)
			}
			return currentDir, modulePath, nil
		}

		// Move to parent directory
		parentDir := filepath.Dir(currentDir)
		if parentDir == currentDir {
			// Reached root directory
			break
		}
		currentDir = parentDir
	}

	return "", "", fmt.Errorf("go.mod file not found")
}

// parseGoModFile parses the module name from a go.mod file
func (p *Parser) parseGoModFile(path string) (string, error) {
	// Validate that this is actually a go.mod file
	if !isValidGoModPath(path) {
		return "", fmt.Errorf("invalid go.mod file path: %s", path)
	}

	// Clean the path to prevent path traversal
	cleanPath := filepath.Clean(path)

	// Ensure the clean path doesn't contain path traversal attempts
	if strings.Contains(cleanPath, "..") {
		return "", fmt.Errorf("path traversal not allowed in go.mod path: %s", path)
	}

	// Use the shared go.mod parser
	return p.goModParser.ParseModuleName(cleanPath)
}

// calculatePackageImportPath calculates the full import path for a package
func (p *Parser) calculatePackageImportPath(moduleRoot, modulePath, packagePath string) (string, error) {
	// Validate input paths
	if !isSecureDirectoryPath(packagePath) {
		return "", fmt.Errorf("invalid package path: %s", packagePath)
	}

	if !isSecureDirectoryPath(moduleRoot) {
		return "", fmt.Errorf("invalid module root: %s", moduleRoot)
	}

	// Clean and normalize paths
	cleanPackagePath := filepath.Clean(packagePath)
	cleanModuleRoot := filepath.Clean(moduleRoot)

	// Ensure paths don't contain traversal attempts
	if strings.Contains(cleanPackagePath, "..") {
		return "", fmt.Errorf("path traversal not allowed in package path: %s", packagePath)
	}

	if strings.Contains(cleanModuleRoot, "..") {
		return "", fmt.Errorf("path traversal not allowed in module root: %s", moduleRoot)
	}

	// Convert package path to absolute path
	absPackagePath, err := filepath.Abs(cleanPackagePath)
	if err != nil {
		return "", fmt.Errorf("failed to resolve package path: %w", err)
	}

	// Convert module root to absolute path for comparison
	absModuleRoot, err := filepath.Abs(cleanModuleRoot)
	if err != nil {
		return "", fmt.Errorf("failed to resolve module root: %w", err)
	}

	// Calculate relative path from module root
	relPath, err := filepath.Rel(absModuleRoot, absPackagePath)
	if err != nil {
		return "", fmt.Errorf("failed to calculate relative path: %w", err)
	}

	// Ensure the relative path doesn't escape the module root
	if strings.HasPrefix(relPath, "..") {
		return "", fmt.Errorf("package path is outside module root")
	}

	// Convert file path separators to forward slashes for import paths
	importPath := filepath.ToSlash(relPath)

	// Build full import path
	if importPath == "." {
		return modulePath, nil
	}

	return fmt.Sprintf("%s/%s", modulePath, importPath), nil
}

// extractDependencies extracts dependency information from a struct type by looking for //axon::inject annotations
func (p *Parser) extractDependencies(structType *ast.StructType) []models.Dependency {
	var dependencies []models.Dependency

	for _, field := range structType.Fields.List {
		// Skip fields without names (embedded fields)
		if len(field.Names) == 0 {
			continue
		}

		// Check if field has //axon::inject or //axon::init annotation
		hasInjectAnnotation := false
		hasInitAnnotation := false
		if field.Doc != nil {
			for _, comment := range field.Doc.List {
				if strings.Contains(comment.Text, "axon::inject") {
					hasInjectAnnotation = true
				}
				if strings.Contains(comment.Text, "axon::init") {
					hasInitAnnotation = true
				}
			}
		}

		// Only include fields with //axon::inject or //axon::init annotation
		if !hasInjectAnnotation && !hasInitAnnotation {
			continue
		}

		// Get the type string
		typeStr := p.getTypeString(field.Type)

		// Create dependency
		dep := models.Dependency{
			Name:   field.Names[0].Name,
			Type:   typeStr,
			IsInit: hasInitAnnotation,
		}

		dependencies = append(dependencies, dep)
	}

	return dependencies
}

// getTypeString converts an AST type expression to a string
func (p *Parser) getTypeString(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.StarExpr:
		return "*" + p.getTypeString(t.X)
	case *ast.SelectorExpr:
		return p.getTypeString(t.X) + "." + t.Sel.Name
	case *ast.ArrayType:
		return "[]" + p.getTypeString(t.Elt)
	case *ast.MapType:
		return "map[" + p.getTypeString(t.Key) + "]" + p.getTypeString(t.Value)
	case *ast.InterfaceType:
		return "interface{}"
	case *ast.ChanType:
		return "chan " + p.getTypeString(t.Value)
	case *ast.FuncType:
		return p.getFuncTypeString(t)
	default:
		return "unknown"
	}
}

// getFuncTypeString converts a function type to its string representation
func (p *Parser) getFuncTypeString(funcType *ast.FuncType) string {
	result := "func("

	// Add parameters
	if funcType.Params != nil {
		var params []string
		for _, param := range funcType.Params.List {
			paramType := p.getTypeString(param.Type)
			if len(param.Names) > 0 {
				// Named parameters
				for _, name := range param.Names {
					params = append(params, name.Name+" "+paramType)
				}
			} else {
				// Unnamed parameters
				params = append(params, paramType)
			}
		}
		result += strings.Join(params, ", ")
	}

	result += ")"

	// Add return types
	if funcType.Results != nil && len(funcType.Results.List) > 0 {
		result += " "
		var returns []string
		for _, ret := range funcType.Results.List {
			returns = append(returns, p.getTypeString(ret.Type))
		}
		if len(returns) == 1 {
			result += returns[0]
		} else {
			result += "(" + strings.Join(returns, ", ") + ")"
		}
	}

	return result
}

// parsePathParameters extracts path parameters from a route path
func (p *Parser) parsePathParameters(path string) ([]models.Parameter, error) {
	var parameters []models.Parameter

	// Find all path parameters in both formats: {name:type} and {name}
	// First, find typed parameters {name:type}
	typedParamRegex := `\{([^:}]+):([^}]+)\}`
	typedMatches := regexp.MustCompile(typedParamRegex).FindAllStringSubmatch(path, -1)

	for _, match := range typedMatches {
		if len(match) != 3 {
			continue
		}

		paramName := match[1]
		paramType := match[2]

		// Validate parameter type
		_, err := p.validateParameterType(paramType)
		if err != nil {
			return nil, fmt.Errorf("invalid parameter type '%s' for parameter '%s': %w", paramType, paramName, err)
		}

		// Determine parser function and actual type
		actualType := paramType
		parserFunc := "axon.ParseString" // Default
		isCustomType := false

		// Check if it's a built-in type (including aliases)
		if actualType == "UUID" {
			actualType = "uuid.UUID" // Resolve alias
			parserFunc = "axon.ParseUUID"
		} else {
			switch paramType {
			case "int":
				parserFunc = "axon.ParseInt"
			case "string":
				parserFunc = "axon.ParseString"
			case "bool":
				parserFunc = "axon.ParseBool"
			case "float", "float64":
				parserFunc = "axon.ParseFloat64"
			case "float32":
				parserFunc = "axon.ParseFloat32"
			case "uuid.UUID":
				parserFunc = "axon.ParseUUID"
			default:
				// For custom types, leave ParserFunc empty so template can look up in registry
				parserFunc = ""
				isCustomType = true
			}
		}

		param := models.Parameter{
			Name:         paramName,
			Type:         actualType, // Use resolved type
			Source:       models.ParameterSourcePath,
			Required:     true, // Path parameters are always required
			IsCustomType: isCustomType,
			ParserFunc:   parserFunc,
		}

		parameters = append(parameters, param)
	}

	// Find untyped parameters {name} (but exclude ones already found as typed)
	untypedParamRegex := `\{([^:}]+)\}`
	untypedMatches := regexp.MustCompile(untypedParamRegex).FindAllStringSubmatch(path, -1)

	// Keep track of already processed parameter names to avoid duplicates
	processedParams := make(map[string]bool)
	for _, param := range parameters {
		processedParams[param.Name] = true
	}

	for _, match := range untypedMatches {
		if len(match) != 2 {
			continue
		}

		paramName := match[1]

		// Skip if this parameter was already processed as a typed parameter
		if processedParams[paramName] {
			continue
		}

		// Default to string type for untyped parameters
		paramType := "string"

		// Validate parameter type (even though it's just string)
		_, err := p.validateParameterType(paramType)
		if err != nil {
			return nil, fmt.Errorf("invalid parameter type '%s' for parameter '%s': %w", paramType, paramName, err)
		}

		// Use string parser for untyped parameters
		param := models.Parameter{
			Name:         paramName,
			Type:         paramType,
			Source:       models.ParameterSourcePath,
			Required:     true, // Path parameters are always required
			IsCustomType: false,
			ParserFunc:   "axon.ParseString",
		}

		parameters = append(parameters, param)
		processedParams[paramName] = true
	}

	return parameters, nil
}

// validateParameterType validates that a parameter type is supported
func (p *Parser) validateParameterType(typeStr string) (string, error) {
	validTypes := map[string]string{
		"int":         "int",
		"string":      "string",
		"bool":        "bool",
		"float":       "float64",
		"uuid":        "string", // UUID is represented as string
		"UUID":        "string", // UUID (uppercase) is also supported
		"ProductCode": "string", // Custom types are treated as strings
		"DateRange":   "string", // Custom types are treated as strings
	}

	if validType, exists := validTypes[typeStr]; exists {
		return validType, nil
	}

	// Allow any custom type that looks like a valid Go identifier
	if isValidGoIdentifier(typeStr) {
		return "string", nil // Custom types are treated as strings for parameter parsing
	}

	return "", fmt.Errorf("unsupported parameter type: %s", typeStr)
}

// isValidGoIdentifier checks if a string is a valid Go identifier
func isValidGoIdentifier(s string) bool {
	if s == "" {
		return false
	}

	// First character must be a letter or underscore
	if !((s[0] >= 'a' && s[0] <= 'z') || (s[0] >= 'A' && s[0] <= 'Z') || s[0] == '_') {
		return false
	}

	// Remaining characters must be letters, digits, or underscores
	for i := 1; i < len(s); i++ {
		c := s[i]
		if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_' || c == '.') {
			return false
		}
	}

	return true
}

// analyzeHandlerSignature analyzes a handler method signature to extract parameters
func (p *Parser) analyzeHandlerSignature(file *ast.File, controllerName, methodName string) ([]models.Parameter, error) {
	var parameters []models.Parameter

	// Find the method in the AST
	ast.Inspect(file, func(n ast.Node) bool {
		if funcDecl, ok := n.(*ast.FuncDecl); ok {
			// Check if this is the method we're looking for
			if funcDecl.Name.Name == methodName && funcDecl.Recv != nil {
				// Check receiver type
				if len(funcDecl.Recv.List) > 0 {
					if starExpr, ok := funcDecl.Recv.List[0].Type.(*ast.StarExpr); ok {
						if ident, ok := starExpr.X.(*ast.Ident); ok && ident.Name == controllerName {
							// This is our method - extract parameters
							if funcDecl.Type.Params != nil {
								for i, param := range funcDecl.Type.Params.List {
									if len(param.Names) > 0 {
										paramType := p.getTypeString(param.Type)
										for _, name := range param.Names {
											// Determine parameter source based on type and position
											source := models.ParameterSourceBody // Default

											// Check if this is an echo.Context parameter
											if paramType == "echo.Context" {
												source = models.ParameterSourceContext
											} else if paramType == "axon.QueryMap" {
												source = models.ParameterSourceQuery
											}

											p := models.Parameter{
												Name:     name.Name,
												Type:     paramType,
												Source:   source,
												Position: i, // Track position for context parameters
											}
											parameters = append(parameters, p)
										}
									}
								}
							}
							return false // Stop searching
						}
					}
				}
			}
		}
		return true
	})

	return parameters, nil
}

// analyzeReturnType analyzes a handler method's return type
func (p *Parser) analyzeReturnType(file *ast.File, controllerName, methodName string) (string, error) {
	var returnType string

	// Find the method in the AST
	ast.Inspect(file, func(n ast.Node) bool {
		if funcDecl, ok := n.(*ast.FuncDecl); ok {
			// Check if this is the method we're looking for
			if funcDecl.Name.Name == methodName && funcDecl.Recv != nil {
				// Check receiver type
				if len(funcDecl.Recv.List) > 0 {
					if starExpr, ok := funcDecl.Recv.List[0].Type.(*ast.StarExpr); ok {
						if ident, ok := starExpr.X.(*ast.Ident); ok && ident.Name == controllerName {
							// This is our method - extract return type
							if funcDecl.Type.Results != nil && len(funcDecl.Type.Results.List) > 0 {
								returnType = p.getTypeString(funcDecl.Type.Results.List[0].Type)
							}
							return false // Stop searching
						}
					}
				}
			}
		}
		return true
	})

	if returnType == "" {
		return "void", nil
	}

	return returnType, nil
}

// mergeParameters merges path parameters with signature parameters
func (p *Parser) mergeParameters(pathParams, signatureParams []models.Parameter) []models.Parameter {
	// Create maps for quick lookup
	pathParamMap := make(map[string]models.Parameter)
	for _, param := range pathParams {
		pathParamMap[param.Name] = param
	}

	signatureParamMap := make(map[string]models.Parameter)
	for _, param := range signatureParams {
		signatureParamMap[param.Name] = param
	}

	var merged []models.Parameter

	// Add context parameters from signature (always include these)
	for _, sigParam := range signatureParams {
		if sigParam.Source == models.ParameterSourceContext {
			merged = append(merged, sigParam)
		}
	}

	// Add path parameters ONLY if they exist in the method signature
	for _, pathParam := range pathParams {
		if pathParam.Name == "*" {
			// For wildcard parameters, find the first string parameter in signature that's not context
			for _, sigParam := range signatureParams {
				if sigParam.Source != models.ParameterSourceContext &&
					sigParam.Type == "string" {

					// Check if this param is already matched to another path param
					_, alreadyMatched := pathParamMap[sigParam.Name]
					if !alreadyMatched {
						// Match wildcard to this string parameter - mark it as coming from wildcard
						mergedParam := sigParam
						mergedParam.Source = models.ParameterSourcePath
						mergedParam.Required = true
						mergedParam.IsCustomType = pathParam.IsCustomType
						mergedParam.ParserFunc = pathParam.ParserFunc
						// Mark this parameter as sourcing from wildcard by setting a special name pattern
						mergedParam.Name = sigParam.Name + ":*"
						merged = append(merged, mergedParam)
						break
					}
				}
			}
		} else if sigParam, exists := signatureParamMap[pathParam.Name]; exists {
			// Regular path parameter exists in method signature - use the signature parameter
			// but with path parameter metadata (Required, Source, etc.)
			mergedParam := sigParam
			mergedParam.Source = models.ParameterSourcePath
			mergedParam.Required = true
			mergedParam.IsCustomType = pathParam.IsCustomType
			mergedParam.ParserFunc = pathParam.ParserFunc
			merged = append(merged, mergedParam)
		}
		// If path parameter doesn't exist in signature, don't include it
		// The method will extract it from context manually
	} // Add other signature parameters that are not path parameters and not context
	for _, sigParam := range signatureParams {
		if sigParam.Source != models.ParameterSourceContext {
			// Check if this parameter was matched to a path parameter
			_, matchedToPath := pathParamMap[sigParam.Name]

			// For wildcard routes, check if any path parameter was matched to this signature param
			isWildcardMatched := false
			for _, pathParam := range pathParams {
				if pathParam.Name == "*" {
					// Check if this signature param was matched to wildcard
					for _, merged := range merged {
						if strings.HasSuffix(merged.Name, ":*") &&
							strings.TrimSuffix(merged.Name, ":*") == sigParam.Name {
							isWildcardMatched = true
							break
						}
					}
					break
				}
			}

			// Only add as body parameter if not matched to any path parameter
			if !matchedToPath && !isWildcardMatched {
				merged = append(merged, sigParam)
			}
		}
	}

	return merged
}

// addRouteToController adds a route to its corresponding controller
func (p *Parser) addRouteToController(route models.RouteMetadata, metadata *models.PackageMetadata) {
	// Extract controller name from handler name
	parts := strings.Split(route.HandlerName, ".")
	if len(parts) != 2 {
		return
	}

	controllerName := parts[0]
	methodName := parts[1]

	// Update the route to use just the method name for HandlerName
	route.HandlerName = methodName

	// Find the controller and add the route
	for i, controller := range metadata.Controllers {
		if controller.Name == controllerName {
			// Merge controller prefix with route if prefix exists
			if controller.Prefix != "" {
				mergedRoute, err := p.mergeControllerPrefixWithRoute(controller, route)
				if err != nil {
					// Log error but don't fail - use original route
					p.reporter.Debug("Failed to merge controller prefix: %v", err)
					metadata.Controllers[i].Routes = append(metadata.Controllers[i].Routes, route)
				} else {
					metadata.Controllers[i].Routes = append(metadata.Controllers[i].Routes, mergedRoute)
				}
			} else {
				metadata.Controllers[i].Routes = append(metadata.Controllers[i].Routes, route)
			}
			return
		}
	}
}

// extractLifecycleMethods checks if a struct has Start and Stop methods
func (p *Parser) extractLifecycleMethods(file *ast.File, structName string) (hasStart, hasStop bool) {
	ast.Inspect(file, func(n ast.Node) bool {
		if funcDecl, ok := n.(*ast.FuncDecl); ok {
			// Check if this is a method with a receiver
			if funcDecl.Recv != nil && len(funcDecl.Recv.List) > 0 {
				// Check receiver type
				if starExpr, ok := funcDecl.Recv.List[0].Type.(*ast.StarExpr); ok {
					if ident, ok := starExpr.X.(*ast.Ident); ok && ident.Name == structName {
						// This is a method of our struct
						switch funcDecl.Name.Name {
						case "Start":
							hasStart = true
						case "Stop":
							hasStop = true
						}
					}
				}
			}
		}
		return true
	})

	return hasStart, hasStop
}

// extractPublicMethods extracts public methods from a struct
func (p *Parser) extractPublicMethods(file *ast.File, structName string) ([]models.Method, error) {
	var methods []models.Method

	if file == nil {
		return methods, nil
	}

	ast.Inspect(file, func(n ast.Node) bool {
		if funcDecl, ok := n.(*ast.FuncDecl); ok {
			// Check if this is a public method with a receiver
			if funcDecl.Name.IsExported() && funcDecl.Recv != nil && len(funcDecl.Recv.List) > 0 {
				// Check receiver type
				if starExpr, ok := funcDecl.Recv.List[0].Type.(*ast.StarExpr); ok {
					if ident, ok := starExpr.X.(*ast.Ident); ok && ident.Name == structName {
						// This is a public method of our struct
						method := models.Method{
							Name: funcDecl.Name.Name,
						}

						// Extract parameters
						if funcDecl.Type.Params != nil {
							for _, param := range funcDecl.Type.Params.List {
								paramType := p.getTypeString(param.Type)
								if len(param.Names) > 0 {
									for _, name := range param.Names {
										method.Parameters = append(method.Parameters, models.Parameter{
											Name: name.Name,
											Type: paramType,
										})
									}
								} else {
									// Anonymous parameter
									method.Parameters = append(method.Parameters, models.Parameter{
										Type: paramType,
									})
								}
							}
						}

						// Extract return types
						if funcDecl.Type.Results != nil {
							for _, result := range funcDecl.Type.Results.List {
								returnType := p.getTypeString(result.Type)
								method.Returns = append(method.Returns, returnType)
							}
						}

						methods = append(methods, method)
					}
				}
			}
		}
		return true
	})

	return methods, nil
}

// ValidateMiddlewareHandleMethod validates that a middleware has a proper Handle method
func (p *Parser) ValidateMiddlewareHandleMethod(file *ast.File, middlewareName string) error {
	hasHandleMethod := false

	ast.Inspect(file, func(n ast.Node) bool {
		if funcDecl, ok := n.(*ast.FuncDecl); ok {
			// Check if this is a Handle method with a receiver
			if funcDecl.Name.Name == "Handle" && funcDecl.Recv != nil && len(funcDecl.Recv.List) > 0 {
				// Check receiver type
				if starExpr, ok := funcDecl.Recv.List[0].Type.(*ast.StarExpr); ok {
					if ident, ok := starExpr.X.(*ast.Ident); ok && ident.Name == middlewareName {
						hasHandleMethod = true
						return false // Stop searching
					}
				}
			}
		}
		return true
	})

	if !hasHandleMethod {
		return fmt.Errorf("middleware %s is missing Handle method", middlewareName)
	}

	return nil
}

// ValidateParserFunctionSignature validates that a parser function has the correct signature
func (p *Parser) ValidateParserFunctionSignature(file *ast.File, functionName, typeName string) error {
	// For now, just check that the function exists
	functionExists := false

	ast.Inspect(file, func(n ast.Node) bool {
		if funcDecl, ok := n.(*ast.FuncDecl); ok {
			if funcDecl.Name.Name == functionName {
				functionExists = true
				return false // Stop searching
			}
		}
		return true
	})

	if !functionExists {
		return fmt.Errorf("parser function %s not found", functionName)
	}

	return nil
}

// extractParserSignature extracts parameter and return types from a parser function
func (p *Parser) extractParserSignature(file *ast.File, functionName string) ([]string, []string, error) {
	var paramTypes, returnTypes []string

	ast.Inspect(file, func(n ast.Node) bool {
		if funcDecl, ok := n.(*ast.FuncDecl); ok {
			if funcDecl.Name.Name == functionName {
				// Extract parameter types
				if funcDecl.Type.Params != nil {
					for _, param := range funcDecl.Type.Params.List {
						paramType := p.getTypeString(param.Type)
						paramTypes = append(paramTypes, paramType)
					}
				}

				// Extract return types
				if funcDecl.Type.Results != nil {
					for _, result := range funcDecl.Type.Results.List {
						returnType := p.getTypeString(result.Type)
						returnTypes = append(returnTypes, returnType)
					}
				}

				return false // Stop searching
			}
		}
		return true
	})

	return paramTypes, returnTypes, nil
}

// Security validation functions
func isSecureDirectoryPath(path string) bool {
	// Basic validation - reject obviously malicious paths
	if path == "" {
		return false
	}

	// Check for null bytes
	if strings.Contains(path, "\x00") {
		return false
	}

	// Check for dangerous characters
	dangerousChars := []string{"<", ">", "|", "\""}
	for _, char := range dangerousChars {
		if strings.Contains(path, char) {
			return false
		}
	}

	// Allow path traversal here - it will be checked after filepath.Clean()
	// This allows legitimate relative paths like "../malicious" to pass initial validation

	// Reject direct access to system directories
	if strings.HasPrefix(path, "/etc") || strings.HasPrefix(path, "/proc") || strings.HasPrefix(path, "/sys") {
		return false
	}

	return true
}

func isValidGoModPath(path string) bool {
	// Validate that this looks like a go.mod file path
	return strings.HasSuffix(path, "go.mod") && !strings.Contains(path, "..")
}

// parseAnnotationComment (3-argument version for backward compatibility with tests)
func (p *Parser) parseAnnotationComment(comment, target string, pos token.Pos) (models.Annotation, error) {
	return p.parseAnnotationCommentWithFile(comment, target, pos, "")
}

// parseAnnotationCommentWithFile is the full implementation
func (p *Parser) parseAnnotationCommentWithFile(comment, target string, pos token.Pos, fileName string) (models.Annotation, error) {
	// Create source location for the new parser
	location := annotations.SourceLocation{
		File:   fileName,
		Line:   p.fileReader.GetFileSet().Position(pos).Line,
		Column: p.fileReader.GetFileSet().Position(pos).Column,
	}

	// Parse with the new annotations parser
	newAnnotation, err := p.annotationParser.ParseAnnotation(comment, location)
	if err != nil {
		return models.Annotation{}, err
	}

	// Set the target field
	newAnnotation.Target = target

	// Create annotation using the new approach
	return p.createAnnotation(newAnnotation, target), nil
}

// parseParameterDefinition parses a parameter definition (for backward compatibility with tests)
func (p *Parser) parseParameterDefinition(paramDef string, isEchoSyntax bool) (models.Parameter, error) {
	// This is a simplified implementation for test compatibility
	// The new parser handles this internally
	parts := strings.Split(paramDef, ":")
	if len(parts) != 2 {
		return models.Parameter{}, fmt.Errorf("invalid parameter definition: %s", paramDef)
	}

	name := strings.TrimSpace(parts[0])
	typeStr := strings.TrimSpace(parts[1])

	// Validate type
	validType, err := p.validateParameterType(typeStr)
	if err != nil {
		return models.Parameter{}, err
	}

	return models.Parameter{
		Name:   name,
		Type:   validType,
		Source: models.ParameterSourcePath,
	}, nil
}
// parseDirectoryFiles parses all Go files in a directory using cached FileReader
func (p *Parser) parseDirectoryFiles(dirPath string) (map[string]*ast.File, string, error) {
	// Read directory contents
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return nil, "", fmt.Errorf("failed to read directory: %w", err)
	}

	files := make(map[string]*ast.File)
	var packageName string
	
	// Parse each .go file using cached FileReader
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") {
			continue
		}
		
		filePath := filepath.Join(dirPath, entry.Name())
		
		// Use cached ParseGoFile instead of parsing directly
		file, err := p.fileReader.ParseGoFile(filePath)
		if err != nil {
			return nil, "", fmt.Errorf("failed to parse file %s: %w", entry.Name(), err)
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

// extractParametersFromPath extracts parameters from a URL path (e.g., /users/{id:int})
func (p *Parser) extractParametersFromPath(path string) ([]models.Parameter, error) {
	var parameters []models.Parameter
	
	// Find all parameter patterns like {name:type}
	paramPattern := regexp.MustCompile(`\{([^}]+)\}`)
	matches := paramPattern.FindAllStringSubmatch(path, -1)
	
	for _, match := range matches {
		if len(match) < 2 {
			continue
		}
		
		paramDef := match[1]
		parts := strings.Split(paramDef, ":")
		
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid parameter format in path %s: expected {name:type}, got {%s}", path, paramDef)
		}
		
		paramName := strings.TrimSpace(parts[0])
		paramType := strings.TrimSpace(parts[1])
		
		// Convert type to Go type
		goType, err := p.validateParameterType(paramType)
		if err != nil {
			return nil, fmt.Errorf("invalid parameter type %s in path %s: %w", paramType, path, err)
		}
		
		parameter := models.Parameter{
			Name:     paramName,
			Type:     goType,
			Source:   models.ParameterSourcePath,
			Required: true,
		}
		
		parameters = append(parameters, parameter)
	}
	
	return parameters, nil
}

// mergeControllerPrefixWithRoute merges controller prefix parameters with route parameters
// and handles duplicate parameter names by renaming them appropriately
func (p *Parser) mergeControllerPrefixWithRoute(controller models.ControllerMetadata, route models.RouteMetadata) (models.RouteMetadata, error) {
	if controller.Prefix == "" {
		return route, nil
	}
	
	// Extract parameters from controller prefix
	prefixParams, err := p.extractParametersFromPath(controller.Prefix)
	if err != nil {
		return route, fmt.Errorf("failed to parse controller prefix %s: %w", controller.Prefix, err)
	}
	
	// Extract parameters from route path
	routeParams, err := p.extractParametersFromPath(route.Path)
	if err != nil {
		return route, fmt.Errorf("failed to parse route path %s: %w", route.Path, err)
	}
	
	// Check for parameter name conflicts and resolve them
	paramNameMap := make(map[string]string) // old name -> new name
	usedNames := make(map[string]bool)
	
	// First, mark all prefix parameter names as used
	for _, param := range prefixParams {
		usedNames[param.Name] = true
	}
	
	// Then, check route parameters for conflicts and rename if necessary
	for i, param := range routeParams {
		if usedNames[param.Name] {
			// Generate a new name by prefixing with controller name
			newName := fmt.Sprintf("%s%s", strings.ToLower(controller.Name), strings.Title(param.Name))
			paramNameMap[param.Name] = newName
			routeParams[i].Name = newName
			usedNames[newName] = true
		} else {
			usedNames[param.Name] = true
		}
	}
	
	// Merge parameters (prefix first, then route)
	allParams := append(prefixParams, routeParams...)
	
	// Merge existing route parameters (from function signature) with path parameters
	mergedParams := p.mergeParameters(allParams, route.Parameters)
	
	// Update route with merged parameters and combined path
	updatedRoute := route
	updatedRoute.Path = controller.Prefix + route.Path
	updatedRoute.Parameters = mergedParams
	
	return updatedRoute, nil
}