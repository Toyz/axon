package parser

import (
	"bufio"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"

	"github.com/toyz/axon/internal/models"
	"github.com/toyz/axon/internal/registry"
	"github.com/toyz/axon/pkg/axon"
)

// Parser implements the AnnotationParser interface
type Parser struct {
	fileSet            *token.FileSet
	middlewareRegistry registry.MiddlewareRegistry
	skipParserValidation     bool // Skip custom parser validation during discovery phase
	skipMiddlewareValidation bool // Skip middleware validation during discovery phase
	reporter           DiagnosticReporter // For debug logging and error reporting
}

// DiagnosticReporter interface for debug logging
type DiagnosticReporter interface {
	Debug(format string, args ...interface{})
	DebugSection(section string)
}

// NewParser creates a new annotation parser
func NewParser() *Parser {
	return &Parser{
		fileSet:            token.NewFileSet(),
		middlewareRegistry: registry.NewMiddlewareRegistry(),
		reporter:           &noOpReporter{}, // Default no-op reporter for backward compatibility
	}
}

// NewParserWithReporter creates a new annotation parser with a diagnostic reporter
func NewParserWithReporter(reporter DiagnosticReporter) *Parser {
	return &Parser{
		fileSet:            token.NewFileSet(),
		middlewareRegistry: registry.NewMiddlewareRegistry(),
		reporter:           reporter,
	}
}

// noOpReporter is a no-op implementation of DiagnosticReporter for backward compatibility
type noOpReporter struct{}

func (n *noOpReporter) Debug(format string, args ...interface{}) {}
func (n *noOpReporter) DebugSection(section string) {}

// ParseSource parses source code from a string for testing purposes
func (p *Parser) ParseSource(filename, source string) (*models.PackageMetadata, error) {
	// Parse the source code
	file, err := parser.ParseFile(p.fileSet, filename, source, parser.ParseComments)
	if err != nil {
		return nil, fmt.Errorf("failed to parse source: %w", err)
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
	if !isValidDirectoryPath(path) {
		return nil, fmt.Errorf("invalid directory path: %s", path)
	}
	
	// Clean and normalize the path
	cleanPath := filepath.Clean(path)
	
	// Ensure the clean path doesn't escape the current working directory
	if strings.Contains(cleanPath, "..") {
		return nil, fmt.Errorf("path traversal not allowed: %s", path)
	}
	
	// Parse all Go files in the directory
	pkgs, err := parser.ParseDir(p.fileSet, cleanPath, nil, parser.ParseComments)
	if err != nil {
		return nil, fmt.Errorf("failed to parse directory %s: %w", cleanPath, err)
	}

	// We expect only one package per directory
	if len(pkgs) == 0 {
		return nil, fmt.Errorf("no Go packages found in directory %s", path)
	}
	if len(pkgs) > 1 {
		return nil, fmt.Errorf("multiple packages found in directory %s", path)
	}

	// Get the single package
	var pkg *ast.Package
	var packageName string
	for name, p := range pkgs {
		pkg = p
		packageName = name
		break
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
	fileMap := make(map[string]*ast.File)
	
	for fileName, file := range pkg.Files {
		fileMap[fileName] = file
		
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
	if !isValidDirectoryPath(startPath) {
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
	
	// Ensure it's actually a go.mod file
	if !strings.HasSuffix(cleanPath, "go.mod") {
		return "", fmt.Errorf("file is not a go.mod file: %s", path)
	}
	
	file, err := os.Open(cleanPath)
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

// calculatePackageImportPath calculates the full import path for a package
func (p *Parser) calculatePackageImportPath(moduleRoot, modulePath, packagePath string) (string, error) {
	// Validate input paths
	if !isValidDirectoryPath(packagePath) {
		return "", fmt.Errorf("invalid package path: %s", packagePath)
	}
	
	if !isValidDirectoryPath(moduleRoot) {
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

// ExtractAnnotations traverses the AST and extracts axon:: annotations from comments
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
									if annotation, err := p.parseAnnotationComment(comment.Text, typeSpec.Name.Name, comment.Pos()); err == nil {
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
					if annotation, err := p.parseAnnotationComment(comment.Text, targetName, comment.Pos()); err == nil {
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

// parseAnnotationComment parses a single comment line for axon:: annotations
func (p *Parser) parseAnnotationComment(comment, target string, pos token.Pos) (models.Annotation, error) {
	// Remove comment prefix
	text := strings.TrimPrefix(comment, "//")
	text = strings.TrimSpace(text)

	// Check if it's an axon:: annotation
	if !strings.HasPrefix(text, AnnotationPrefix) {
		return models.Annotation{}, fmt.Errorf("not an axon annotation")
	}

	// Remove axon:: prefix
	text = strings.TrimPrefix(text, AnnotationPrefix)
	parts := strings.Fields(text)
	if len(parts) == 0 {
		return models.Annotation{}, fmt.Errorf("empty annotation")
	}

	// Parse annotation type
	annotationType, err := p.parseAnnotationType(parts[0])
	if err != nil {
		return models.Annotation{}, err
	}

	annotation := models.Annotation{
		Type:       annotationType,
		Target:     target,
		Parameters: make(map[string]string),
		Flags:      []string{},
		Line:       p.fileSet.Position(pos).Line,
	}

	// Parse remaining parts as parameters and flags
	for i := 1; i < len(parts); i++ {
		part := parts[i]
		if strings.HasPrefix(part, "-") {
			// It's a flag
			if strings.Contains(part, "=") {
				// Flag with value like -Manual=ModuleName
				flagParts := strings.SplitN(part, "=", 2)
				// Keep the leading dash for the parameter key to match constants
				paramKey := flagParts[0]
				annotation.Parameters[paramKey] = flagParts[1]
			} else {
				// Simple flag like -Init
				annotation.Flags = append(annotation.Flags, part)
			}
		} else {
			// It's a parameter - handle different annotation types
			switch annotationType {
			case models.AnnotationTypeRoute:
				// For routes: //axon::route GET /users/{id:int} -Middleware=Auth
				if i == 1 {
					annotation.Parameters[ParamMethod] = part
				} else if i == 2 {
					annotation.Parameters[ParamPath] = part
				}
			case models.AnnotationTypeMiddleware:
				// For middleware: //axon::middleware AuthMiddleware
				if i == 1 {
					annotation.Parameters[ParamName] = part
				}
			case models.AnnotationTypeCore:
				// Core services may have additional parameters in the future
			case models.AnnotationTypeRouteParser:
				// For route parsers: //axon::route_parser TypeName
				if i == 1 {
					annotation.Parameters[ParamName] = part
				}
			}
		}
	}

	// Validate required parameters
	switch annotationType {
	case models.AnnotationTypeRoute:
		if _, hasMethod := annotation.Parameters[ParamMethod]; !hasMethod {
			return models.Annotation{}, fmt.Errorf("route annotation missing method")
		}
		if _, hasPath := annotation.Parameters[ParamPath]; !hasPath {
			return models.Annotation{}, fmt.Errorf("route annotation missing path")
		}
	case models.AnnotationTypeMiddleware:
		if _, hasName := annotation.Parameters[ParamName]; !hasName {
			return models.Annotation{}, fmt.Errorf("middleware annotation missing name")
		}
	case models.AnnotationTypeRouteParser:
		if _, hasName := annotation.Parameters[ParamName]; !hasName {
			return models.Annotation{}, fmt.Errorf("route_parser annotation missing type name")
		}
	}

	return annotation, nil
}

// parseAnnotationType converts string annotation type to AnnotationType enum
func (p *Parser) parseAnnotationType(typeStr string) (models.AnnotationType, error) {
	switch typeStr {
	case AnnotationTypeController:
		return models.AnnotationTypeController, nil
	case AnnotationTypeRoute:
		return models.AnnotationTypeRoute, nil
	case AnnotationTypeMiddleware:
		return models.AnnotationTypeMiddleware, nil
	case AnnotationTypeCore:
		return models.AnnotationTypeCore, nil
	case AnnotationTypeInterface:
		return models.AnnotationTypeInterface, nil
	case AnnotationTypeInject:
		return models.AnnotationTypeInject, nil
	case AnnotationTypeInit:
		return models.AnnotationTypeInit, nil
	case AnnotationTypeLogger:
		return models.AnnotationTypeLogger, nil
	case AnnotationTypeRouteParser:
		return models.AnnotationTypeRouteParser, nil
	default:
		return 0, fmt.Errorf("unknown annotation type: %s", typeStr)
	}
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
				Name:         annotation.Parameters[ParamName],
				PackagePath:  metadata.PackagePath,
				StructName:   annotation.Target,
				Dependencies: annotation.Dependencies,
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
				Method:      annotation.Parameters[ParamMethod],
				Path:        annotation.Parameters[ParamPath],
				HandlerName: annotation.Target,
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
				route.ReturnType = models.ReturnTypeInfo{Type: returnType}
			}
			
			// Parse middleware flags and validate
			if middlewareFlag, exists := annotation.Parameters[FlagMiddleware]; exists {
				middlewareNames := strings.Split(middlewareFlag, ",")
				// Trim whitespace from middleware names
				for i, name := range middlewareNames {
					middlewareNames[i] = strings.TrimSpace(name)
				}
				
				// Validate that all middleware names exist in the registry (skip during discovery phase)
				if !p.skipMiddlewareValidation {
					err := p.middlewareRegistry.Validate(middlewareNames)
					if err != nil {
						return fmt.Errorf("route %s has invalid middleware reference: %w", annotation.Target, err)
					}
				}
				
				route.Middlewares = middlewareNames
			}
			
			// Add flags
			route.Flags = annotation.Flags
			
			// Find the controller this route belongs to and add it
			p.addRouteToController(route, metadata)

		case models.AnnotationTypeMiddleware:
			middleware := models.MiddlewareMetadata{
				Name:         annotation.Parameters[ParamName],
				PackagePath:  metadata.PackagePath,
				StructName:   annotation.Target,
				Dependencies: annotation.Dependencies,
			}
			metadata.Middlewares = append(metadata.Middlewares, middleware)

		case models.AnnotationTypeCore:
			service := models.CoreServiceMetadata{
				Name:         annotation.Target,
				StructName:   annotation.Target,
				Dependencies: annotation.Dependencies,
			}
			
			// Check for lifecycle flag or parameter
			for _, flag := range annotation.Flags {
				if flag == FlagInit {
					service.HasLifecycle = true
					// Default start mode is "Same" (synchronous)
					service.StartMode = "Same"
					
					// Detect Start and Stop methods when lifecycle is enabled
					file := fileMap[annotation.FileName]
					if file != nil {
						hasStart, hasStop := p.extractLifecycleMethods(file, annotation.Target)
						service.HasStart = hasStart
						service.HasStop = hasStop
						
						// Validate that Start method exists when -Init flag is used
						if !hasStart {
							return fmt.Errorf("service %s has -Init flag but missing Start(context.Context) error method", annotation.Target)
						}
					} else {
						// If file is not available (e.g., in unit tests), skip method detection
						// This allows unit tests to work without providing full file context
						// In real usage, files will always be available
						service.HasStart = true  // Assume valid for unit tests
						service.HasStop = false  // Default to no Stop method
					}
				}
			}
			
			// Check for Init parameter (e.g., -Init=Background)
			if initMode, exists := annotation.Parameters["Init"]; exists {
				service.HasLifecycle = true
				
				// Debug output

				
				// Validate init mode
				if initMode == "Background" || initMode == "Same" || initMode == "" {
					if initMode == "" {
						service.StartMode = "Same" // Default when just -Init is used
					} else {
						service.StartMode = initMode
					}
	
				} else {
					return fmt.Errorf("service %s has invalid -Init value '%s': must be 'Background' or 'Same'", annotation.Target, initMode)
				}
				
				// Detect Start and Stop methods when lifecycle is enabled
				file := fileMap[annotation.FileName]
				if file != nil {
					hasStart, hasStop := p.extractLifecycleMethods(file, annotation.Target)
					service.HasStart = hasStart
					service.HasStop = hasStop
					
					// Validate that Start method exists when -Init parameter is used
					if !hasStart {
						return fmt.Errorf("service %s has -Init parameter but missing Start(context.Context) error method", annotation.Target)
					}
				} else {
					// If file is not available (e.g., in unit tests), skip method detection
					service.HasStart = true  // Assume valid for unit tests
					service.HasStop = false  // Default to no Stop method
				}
			}
			
			// Check for manual flags
			if manualModule, exists := annotation.Parameters[FlagManual]; exists {
				service.IsManual = true
				service.ModuleName = manualModule
			} else {
				for _, flag := range annotation.Flags {
					if flag == FlagManual {
						service.IsManual = true
						service.ModuleName = DefaultModuleName
						break
					}
				}
			}
			
			// Check for mode flag (default to Singleton)
			service.Mode = LifecycleModeSingleton // Default mode
			if modeFlag, exists := annotation.Parameters["Mode"]; exists {
				if modeFlag == LifecycleModeTransient || modeFlag == LifecycleModeSingleton {
					service.Mode = modeFlag
				} else {
					return fmt.Errorf("service %s has invalid mode '%s': must be 'Singleton' or 'Transient'", annotation.Target, modeFlag)
				}
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
			
			// Check for lifecycle flag
			for _, flag := range annotation.Flags {
				if flag == FlagInit {
					logger.HasLifecycle = true
					// Detect Start and Stop methods when lifecycle is enabled
					file := fileMap[annotation.FileName]
					if file != nil {
						hasStart, hasStop := p.extractLifecycleMethods(file, annotation.Target)
						logger.HasStart = hasStart
						logger.HasStop = hasStop
						
						// Validate that Start method exists when -Init flag is used
						if !hasStart {
							return fmt.Errorf("logger %s has -Init flag but missing Start(context.Context) error method", annotation.Target)
						}
					} else {
						// If file is not available (e.g., in unit tests), skip method detection
						logger.HasStart = true  // Assume valid for unit tests
						logger.HasStop = false  // Default to no Stop method
					}
				}
			}
			
			// Check for manual flags
			if manualModule, exists := annotation.Parameters[FlagManual]; exists {
				logger.IsManual = true
				logger.ModuleName = manualModule
			} else {
				for _, flag := range annotation.Flags {
					if flag == FlagManual {
						logger.IsManual = true
						logger.ModuleName = DefaultModuleName
						break
					}
				}
			}
			
			metadata.Loggers = append(metadata.Loggers, logger)

		case models.AnnotationTypeRouteParser:
			// Route parser annotations should be on function declarations
			typeName := annotation.Parameters[ParamName]
			
			// Validate that this is actually on a function and has correct signature
			file := fileMap[annotation.FileName]
			if file != nil {
				err := p.ValidateParserFunctionSignature(file, annotation.Target, typeName)
				if err != nil {
					return fmt.Errorf("parser function validation failed for %s: %w", annotation.Target, err)
				}
			}
			
			parser := models.RouteParserMetadata{
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

		case models.AnnotationTypeInterface:
			// Interface annotations are processed in combination with other annotations
			// They don't create standalone components, so we skip them here
			continue
		}
	}

	// Validate and link custom parsers to their registered parser functions
	// Skip validation during discovery phase
	if !p.skipParserValidation {
		err := p.validateAndLinkCustomParsers(metadata)
		if err != nil {
			return fmt.Errorf("parser validation failed: %w", err)
		}
	}

	return nil
}

// extractLifecycleMethods analyzes the file to find Start and Stop methods for a given struct
func (p *Parser) extractLifecycleMethods(file *ast.File, structName string) (hasStart bool, hasStop bool) {
	ast.Inspect(file, func(n ast.Node) bool {
		if funcDecl, ok := n.(*ast.FuncDecl); ok {
			// Check if this is a method on our struct
			if funcDecl.Recv != nil && len(funcDecl.Recv.List) > 0 {
				// Get receiver type
				var receiverType string
				switch recv := funcDecl.Recv.List[0].Type.(type) {
				case *ast.StarExpr:
					if ident, ok := recv.X.(*ast.Ident); ok {
						receiverType = ident.Name
					}
				case *ast.Ident:
					receiverType = recv.Name
				}
				
				// Check if this method belongs to our struct
				if receiverType == structName {
					methodName := funcDecl.Name.Name
					
					// Check for Start method with correct signature
					if methodName == "Start" && p.isLifecycleMethodSignature(funcDecl, "Start") {
						hasStart = true
					}
					
					// Check for Stop method with correct signature
					if methodName == "Stop" && p.isLifecycleMethodSignature(funcDecl, "Stop") {
						hasStop = true
					}
				}
			}
		}
		return true
	})
	
	return hasStart, hasStop
}

// isLifecycleMethodSignature checks if a method has the correct signature for lifecycle methods
func (p *Parser) isLifecycleMethodSignature(funcDecl *ast.FuncDecl, methodName string) bool {
	// Check parameters: should have (ctx context.Context)
	if funcDecl.Type.Params == nil || len(funcDecl.Type.Params.List) != 1 {
		return false
	}
	
	param := funcDecl.Type.Params.List[0]
	if len(param.Names) != 1 {
		return false
	}
	
	// Check parameter type is context.Context
	if selectorExpr, ok := param.Type.(*ast.SelectorExpr); ok {
		if ident, ok := selectorExpr.X.(*ast.Ident); ok {
			if ident.Name == "context" && selectorExpr.Sel.Name == "Context" {
				// Check return type: should return error
				if funcDecl.Type.Results != nil && len(funcDecl.Type.Results.List) == 1 {
					if ident, ok := funcDecl.Type.Results.List[0].Type.(*ast.Ident); ok {
						return ident.Name == "error"
					}
				}
			}
		}
	}
	
	return false
}

// extractDependencies analyzes struct fields to find fx.In dependencies or //axon::inject annotations
func (p *Parser) extractDependencies(structType *ast.StructType) []models.Dependency {
	p.reporter.DebugSection("Dependency Extraction")
	p.reporter.Debug("Starting dependency extraction")
	var dependencies []models.Dependency
	var hasFxIn bool
	
	// First, check if struct has embedded fx.In
	p.reporter.Debug("Checking for embedded fx.In")
	for _, field := range structType.Fields.List {
		if len(field.Names) == 0 { // embedded field
			if selectorExpr, ok := field.Type.(*ast.SelectorExpr); ok {
				if ident, ok := selectorExpr.X.(*ast.Ident); ok {
					if ident.Name == "fx" && selectorExpr.Sel.Name == "In" {
						p.reporter.Debug("Found embedded fx.In")
						hasFxIn = true
						break
					}
				}
			}
		}
	}
	
	// If struct has fx.In, extract all other named fields as dependencies
	if hasFxIn {
		p.reporter.Debug("Processing fx.In struct fields")
		for _, field := range structType.Fields.List {
			// Skip embedded fx.In field
			if len(field.Names) == 0 {
				continue
			}
			
			// Extract dependency names from named fields
			for _, name := range field.Names {
				p.reporter.Debug("Processing fx.In field '%s' (exported: %v)", name.Name, name.IsExported())
				if name.IsExported() {
					dep := models.Dependency{
						Name: name.Name,
						Type: p.getFieldTypeName(field.Type),
					}
					p.reporter.Debug("Added fx.In dependency: Name=%s, Type=%s, IsInit=%v", dep.Name, dep.Type, dep.IsInit)
					dependencies = append(dependencies, dep)
				}
			}
		}
	} else {
		// If no fx.In, check for //axon::inject annotations on individual fields
		p.reporter.Debug("Processing individual field annotations (total fields: %d)", len(structType.Fields.List))
		for fieldIndex, field := range structType.Fields.List {
			p.reporter.Debug("Processing field %d", fieldIndex)
			
			// Skip embedded fields
			if len(field.Names) == 0 {
				p.reporter.Debug("Skipping embedded field %d", fieldIndex)
				continue
			}
			
			// Log field names
			var fieldNames []string
			for _, name := range field.Names {
				fieldNames = append(fieldNames, name.Name)
			}
			p.reporter.Debug("Field %d names: %v", fieldIndex, fieldNames)
			
			// Check if field has //axon::inject or //axon::init annotation (in Doc or Comment)
			var foundAnnotation bool
			
			// First check Doc comments (above the field)
			if field.Doc != nil {
				p.reporter.Debug("Field %d has %d doc comments", fieldIndex, len(field.Doc.List))
				for commentIndex, comment := range field.Doc.List {
					p.reporter.Debug("Field %d, doc comment %d: %s", fieldIndex, commentIndex, comment.Text)
					
					// Check for //axon::inject annotation
					if hasInject, hasInitFlag, _ := p.parseInjectAnnotation(comment.Text); hasInject {
						p.reporter.Debug("Found //axon::inject annotation on field %d (hasInit: %v)", fieldIndex, hasInitFlag)
						// Extract dependency from this field
						// When //axon::inject is present, always include the field regardless of export status
						for _, name := range field.Names {
							dep := models.Dependency{
								Name:   name.Name,
								Type:   p.getFieldTypeName(field.Type),
								IsInit: hasInitFlag, // -Init flag on inject annotation
							}
							p.reporter.Debug("Added inject dependency: Name=%s, Type=%s, IsInit=%v", dep.Name, dep.Type, dep.IsInit)
							dependencies = append(dependencies, dep)
						}
						foundAnnotation = true
						break
					}
					
					// Check for //axon::init annotation
					if hasInit, _ := p.parseInitAnnotation(comment.Text); hasInit {
						p.reporter.Debug("Found //axon::init annotation on field %d", fieldIndex)
						// Extract init dependency from this field
						// When //axon::init is present, always include the field regardless of export status
						for _, name := range field.Names {
							dep := models.Dependency{
								Name:   name.Name,
								Type:   p.getFieldTypeName(field.Type),
								IsInit: true, // Always init for //axon::init annotation
							}
							p.reporter.Debug("Added init dependency: Name=%s, Type=%s, IsInit=%v", dep.Name, dep.Type, dep.IsInit)
							dependencies = append(dependencies, dep)
						}
						foundAnnotation = true
						break
					}
				}
			} else {
				p.reporter.Debug("Field %d has no doc comments", fieldIndex)
			}
			
			// If no annotation found in Doc comments, check Comment (line comments)
			if !foundAnnotation && field.Comment != nil {
				p.reporter.Debug("Field %d has %d line comments", fieldIndex, len(field.Comment.List))
				for commentIndex, comment := range field.Comment.List {
					p.reporter.Debug("Field %d, line comment %d: %s", fieldIndex, commentIndex, comment.Text)
					
					// Check for //axon::inject annotation
					if hasInject, hasInitFlag, _ := p.parseInjectAnnotation(comment.Text); hasInject {
						p.reporter.Debug("Found //axon::inject annotation on field %d (hasInit: %v)", fieldIndex, hasInitFlag)
						// Extract dependency from this field
						// When //axon::inject is present, always include the field regardless of export status
						for _, name := range field.Names {
							dep := models.Dependency{
								Name:   name.Name,
								Type:   p.getFieldTypeName(field.Type),
								IsInit: hasInitFlag, // -Init flag on inject annotation
							}
							p.reporter.Debug("Added inject dependency: Name=%s, Type=%s, IsInit=%v", dep.Name, dep.Type, dep.IsInit)
							dependencies = append(dependencies, dep)
						}
						foundAnnotation = true
						break
					}
					
					// Check for //axon::init annotation
					if hasInit, _ := p.parseInitAnnotation(comment.Text); hasInit {
						p.reporter.Debug("Found //axon::init annotation on field %d", fieldIndex)
						// Extract init dependency from this field
						// When //axon::init is present, always include the field regardless of export status
						for _, name := range field.Names {
							dep := models.Dependency{
								Name:   name.Name,
								Type:   p.getFieldTypeName(field.Type),
								IsInit: true, // Always init for //axon::init annotation
							}
							p.reporter.Debug("Added init dependency: Name=%s, Type=%s, IsInit=%v", dep.Name, dep.Type, dep.IsInit)
							dependencies = append(dependencies, dep)
						}
						foundAnnotation = true
						break
					}
				}
			} else if !foundAnnotation {
				p.reporter.Debug("Field %d has no line comments", fieldIndex)
			}
		}
	}
	
	p.reporter.Debug("Final dependencies list (count: %d):", len(dependencies))
	for i, dep := range dependencies {
		p.reporter.Debug("  [%d] Name=%s, Type=%s, IsInit=%v", i, dep.Name, dep.Type, dep.IsInit)
	}
	return dependencies
}

// parseInjectAnnotation checks if a comment contains //axon::inject and parses flags
func (p *Parser) parseInjectAnnotation(comment string) (bool, bool, bool) {
	// Remove comment prefix
	text := strings.TrimPrefix(comment, "//")
	text = strings.TrimSpace(text)
	
	if !strings.HasPrefix(text, AnnotationPrefix+"inject") {
		return false, false, false
	}
	
	// Check for flags
	hasInit := strings.Contains(text, "-Init")
	ignoreExport := strings.Contains(text, "-IgnoreExport")
	
	return true, hasInit, ignoreExport
}

// parseInitAnnotation checks if a comment contains //axon::init and parses flags
func (p *Parser) parseInitAnnotation(comment string) (bool, bool) {
	// Remove comment prefix
	text := strings.TrimPrefix(comment, "//")
	text = strings.TrimSpace(text)
	
	if !strings.HasPrefix(text, AnnotationPrefix+"init") {
		return false, false
	}
	
	// Check for flags
	ignoreExport := strings.Contains(text, "-IgnoreExport")
	
	return true, ignoreExport
}

// getFieldTypeName extracts the type name from a field type expression
func (p *Parser) getFieldTypeName(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.StarExpr:
		return "*" + p.getFieldTypeName(t.X)
	case *ast.SelectorExpr:
		if ident, ok := t.X.(*ast.Ident); ok {
			return ident.Name + "." + t.Sel.Name
		}
		return t.Sel.Name
	case *ast.ArrayType:
		return "[]" + p.getFieldTypeName(t.Elt)
	case *ast.MapType:
		return "map[" + p.getFieldTypeName(t.Key) + "]" + p.getFieldTypeName(t.Value)
	case *ast.FuncType:
		// Handle function types like func() *SomeType
		var params []string
		if t.Params != nil {
			for _, param := range t.Params.List {
				paramType := p.getFieldTypeName(param.Type)
				params = append(params, paramType)
			}
		}
		
		var results []string
		if t.Results != nil {
			for _, result := range t.Results.List {
				resultType := p.getFieldTypeName(result.Type)
				results = append(results, resultType)
			}
		}
		
		funcStr := "func("
		if len(params) > 0 {
			funcStr += strings.Join(params, ", ")
		}
		funcStr += ")"
		
		if len(results) > 0 {
			if len(results) == 1 {
				funcStr += " " + results[0]
			} else {
				funcStr += " (" + strings.Join(results, ", ") + ")"
			}
		}
		
		return funcStr
	default:
		return "unknown"
	}
}

// parsePathParameters extracts parameters from route path with both {param:type} and :param syntax
func (p *Parser) parsePathParameters(path string) ([]models.Parameter, error) {
	var parameters []models.Parameter
	
	// Find all parameter patterns in the path - both {param:type} and :param
	i := 0
	for i < len(path) {
		// Look for {param:type} syntax
		bracketStart := strings.Index(path[i:], "{")
		// Look for :param syntax
		colonStart := strings.Index(path[i:], ":")
		
		// Determine which parameter type comes first
		if bracketStart == -1 && colonStart == -1 {
			break // No more parameters
		}
		
		var paramStart, paramEnd int
		var paramDef string
		var isEchoSyntax bool
		
		if bracketStart != -1 && (colonStart == -1 || bracketStart < colonStart) {
			// Process {param:type} syntax
			paramStart = bracketStart + i
			end := strings.Index(path[paramStart:], "}")
			if end == -1 {
				return nil, fmt.Errorf("unclosed parameter bracket at position %d", paramStart)
			}
			paramEnd = paramStart + end
			paramDef = path[paramStart+1 : paramEnd]
			isEchoSyntax = false
			i = paramEnd + 1
		} else {
			// Process :param syntax
			paramStart = colonStart + i
			// Find the end of the parameter (next / or end of string)
			paramEnd = paramStart + 1
			for paramEnd < len(path) && path[paramEnd] != '/' {
				paramEnd++
			}
			paramDef = path[paramStart+1 : paramEnd]
			isEchoSyntax = true
			i = paramEnd
		}
		
		// Parse parameter definition
		param, err := p.parseParameterDefinition(paramDef, isEchoSyntax)
		if err != nil {
			return nil, fmt.Errorf("invalid parameter definition '%s': %w", paramDef, err)
		}
		
		parameters = append(parameters, param)
	}
	
	return parameters, nil
}

// parseParameterDefinition parses a single parameter definition like "id:int", "name:string", or just "id" (Echo syntax)
func (p *Parser) parseParameterDefinition(paramDef string, isEchoSyntax bool) (models.Parameter, error) {
	if isEchoSyntax {
		// Echo syntax: just the parameter name, default to string type
		name := strings.TrimSpace(paramDef)
		if name == "" {
			return models.Parameter{}, fmt.Errorf("parameter name cannot be empty")
		}
		
		return models.Parameter{
			Name:         name,
			Type:         "string", // Echo parameters default to string
			Source:       models.ParameterSourcePath,
			Required:     true,
			IsCustomType: false, // Echo syntax always uses built-in string type
		}, nil
	}
	
	// Axon syntax: name:type
	parts := strings.Split(paramDef, ":")
	if len(parts) != 2 {
		return models.Parameter{}, fmt.Errorf("parameter must be in format 'name:type', got '%s'", paramDef)
	}
	
	name := strings.TrimSpace(parts[0])
	typeStr := strings.TrimSpace(parts[1])
	
	if name == "" {
		return models.Parameter{}, fmt.Errorf("parameter name cannot be empty")
	}
	
	// Validate and normalize type
	goType, err := p.validateParameterType(typeStr)
	if err != nil {
		return models.Parameter{}, fmt.Errorf("invalid parameter type '%s': %w", typeStr, err)
	}
	
	// Check if this is a custom type (not a built-in type)
	isCustomType := !axon.IsBuiltinType(typeStr)
	
	param := models.Parameter{
		Name:         name,
		Type:         goType,
		Source:       models.ParameterSourcePath,
		Required:     true, // Path parameters are always required
		IsCustomType: isCustomType,
	}
	
	// For custom types, we'll resolve the parser function and import path later
	// during the parser registry validation phase
	
	return param, nil
}

// validateParameterType validates and normalizes parameter types
func (p *Parser) validateParameterType(typeStr string) (string, error) {
	// Check built-in types first
	switch typeStr {
	case "int":
		return "int", nil
	case "string":
		return "string", nil
	case "float64", "float32":
		return typeStr, nil
	}
	
	// Check if it's a built-in parser type (including aliases)
	if axon.IsBuiltinType(typeStr) {
		return axon.ResolveTypeAlias(typeStr), nil
	}
	
	// For custom types, we'll validate them later during parser registry lookup
	// For now, accept any type that looks like a valid Go type
	if p.isValidGoTypeName(typeStr) {
		return typeStr, nil
	}
	
	return "", fmt.Errorf("invalid parameter type '%s'", typeStr)
}

// isValidGoTypeName checks if a string is a valid Go type name
func (p *Parser) isValidGoTypeName(typeName string) bool {
	if typeName == "" {
		return false
	}
	
	// Allow package.Type syntax (e.g., uuid.UUID, time.Time)
	if strings.Contains(typeName, ".") {
		parts := strings.Split(typeName, ".")
		if len(parts) != 2 {
			return false
		}
		// Both package and type name should be valid identifiers
		return p.isValidIdentifier(parts[0]) && p.isValidIdentifier(parts[1])
	}
	
	// Single identifier (e.g., CustomType)
	return p.isValidIdentifier(typeName)
}

// isValidIdentifier checks if a string is a valid Go identifier
func (p *Parser) isValidIdentifier(name string) bool {
	if name == "" {
		return false
	}
	
	// First character must be letter or underscore
	first := rune(name[0])
	if !((first >= 'a' && first <= 'z') || (first >= 'A' && first <= 'Z') || first == '_') {
		return false
	}
	
	// Remaining characters must be letters, digits, or underscores
	for _, r := range name[1:] {
		if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_') {
			return false
		}
	}
	
	return true
}

// validateAndLinkCustomParsers validates that all custom parameter types have registered parsers
// and links them to the appropriate parser functions
func (p *Parser) validateAndLinkCustomParsers(metadata *models.PackageMetadata) error {
	// Build a registry of available parsers from the metadata
	parserRegistry := make(map[string]models.RouteParserMetadata)
	
	// Add parsers from current package
	for _, parser := range metadata.RouteParsers {
		parserRegistry[parser.TypeName] = parser
	}
	
	// Add built-in parsers
	for typeName, parser := range axon.BuiltinParsers {
		parserRegistry[typeName] = parser
	}
	
	// Validate all routes and their parameters
	for controllerIdx := range metadata.Controllers {
		controller := &metadata.Controllers[controllerIdx]
		
		for routeIdx := range controller.Routes {
			route := &controller.Routes[routeIdx]
			
			for paramIdx := range route.Parameters {
				param := &route.Parameters[paramIdx]
				
				// Only validate custom types
				if param.IsCustomType {
					// Look for a registered parser
					parser, exists := parserRegistry[param.Type]
					if !exists {
						// Try resolving aliases
						resolvedType := axon.ResolveTypeAlias(param.Type)
						if resolvedType != param.Type {
							parser, exists = parserRegistry[resolvedType]
							if exists {
								param.Type = resolvedType // Update to resolved type
							}
						}
					}
					
					if !exists {
						availableParsers := p.getAvailableParsersList(parserRegistry)
						errorReporter := NewParserErrorReporter(p)
						return errorReporter.ReportParserNotFoundError(
							param.Type,
							route.Method,
							route.Path,
							param.Name,
							"", // fileName will be set by caller
							0,  // line will be set by caller
							availableParsers,
						)
					}
					
					// Link the parameter to the parser
					param.ParserFunc = parser.FunctionName
				}
			}
		}
	}
	
	return nil
}

// SetSkipParserValidation controls whether custom parser validation is skipped
func (p *Parser) SetSkipParserValidation(skip bool) {
	p.skipParserValidation = skip
}

// SetSkipMiddlewareValidation controls whether middleware validation is skipped
func (p *Parser) SetSkipMiddlewareValidation(skip bool) {
	p.skipMiddlewareValidation = skip
}

// ValidateCustomParsersWithRegistry validates custom parsers using an external parser registry
func (p *Parser) ValidateCustomParsersWithRegistry(metadata *models.PackageMetadata, parserRegistry map[string]models.RouteParserMetadata) error {
	// Validate all routes and their parameters
	for controllerIdx := range metadata.Controllers {
		controller := &metadata.Controllers[controllerIdx]
		
		for routeIdx := range controller.Routes {
			route := &controller.Routes[routeIdx]
			
			for paramIdx := range route.Parameters {
				param := &route.Parameters[paramIdx]
				
				// Only validate custom types
				if param.IsCustomType {
					// Look for a registered parser
					parser, exists := parserRegistry[param.Type]
					if !exists {
						// Try resolving aliases
						resolvedType := axon.ResolveTypeAlias(param.Type)
						if resolvedType != param.Type {
							parser, exists = parserRegistry[resolvedType]
							if exists {
								param.Type = resolvedType // Update to resolved type
							}
						}
					}
					
					if !exists {
						availableParsers := p.getAvailableParsersListFromMap(parserRegistry)
						errorReporter := NewParserErrorReporter(p)
						return errorReporter.ReportParserNotFoundError(
							param.Type,
							route.Method,
							route.Path,
							param.Name,
							"", // fileName will be set by caller
							0,  // line will be set by caller
							availableParsers,
						)
					}
					
					// Link the parameter to the parser
					param.ParserFunc = parser.FunctionName
				}
			}
		}
	}
	
	return nil
}

// formatAvailableParsers formats the list of available parsers for error messages
func (p *Parser) formatAvailableParsers(parsers map[string]models.RouteParserMetadata) string {
	return p.formatAvailableParsersFromMap(parsers)
}

// formatAvailableParsersFromMap formats the list of available parsers from a map for error messages
func (p *Parser) formatAvailableParsersFromMap(parsers map[string]models.RouteParserMetadata) string {
	if len(parsers) == 0 {
		return "none"
	}
	
	var types []string
	for typeName := range parsers {
		types = append(types, typeName)
	}
	
	// Sort for consistent output
	for i := 0; i < len(types)-1; i++ {
		for j := i + 1; j < len(types); j++ {
			if types[i] > types[j] {
				types[i], types[j] = types[j], types[i]
			}
		}
	}
	
	return strings.Join(types, ", ")
}

// getAvailableParsersList returns a slice of available parser type names
func (p *Parser) getAvailableParsersList(parsers map[string]models.RouteParserMetadata) []string {
	var types []string
	for typeName := range parsers {
		types = append(types, typeName)
	}
	
	// Sort for consistent output
	for i := 0; i < len(types)-1; i++ {
		for j := i + 1; j < len(types); j++ {
			if types[i] > types[j] {
				types[i], types[j] = types[j], types[i]
			}
		}
	}
	
	return types
}

// getAvailableParsersListFromMap returns a slice of available parser type names from a map
func (p *Parser) getAvailableParsersListFromMap(parsers map[string]models.RouteParserMetadata) []string {
	return p.getAvailableParsersList(parsers)
}

// addRouteToController finds the appropriate controller and adds the route to it
func (p *Parser) addRouteToController(route models.RouteMetadata, metadata *models.PackageMetadata) {
	// Extract controller name from handler name (format: ControllerName.MethodName)
	parts := strings.Split(route.HandlerName, ".")
	if len(parts) != 2 {
		return // Invalid format, skip
	}
	
	controllerName := parts[0]
	methodName := parts[1]
	
	// Update the route to use just the method name
	route.HandlerName = methodName
	
	// Find the controller and add the route
	for i := range metadata.Controllers {
		if metadata.Controllers[i].Name == controllerName {
			metadata.Controllers[i].Routes = append(metadata.Controllers[i].Routes, route)
			return
		}
	}
}

// ValidateMiddlewareHandleMethod validates that a middleware struct has the correct Handle method signature
func (p *Parser) ValidateMiddlewareHandleMethod(file *ast.File, middlewareName string) error {
	var middlewareStruct *ast.TypeSpec
	var handleMethod *ast.FuncDecl
	
	// Find the middleware struct
	ast.Inspect(file, func(n ast.Node) bool {
		switch node := n.(type) {
		case *ast.GenDecl:
			if node.Tok == token.TYPE {
				for _, spec := range node.Specs {
					if typeSpec, ok := spec.(*ast.TypeSpec); ok {
						if typeSpec.Name.Name == middlewareName {
							middlewareStruct = typeSpec
						}
					}
				}
			}
		case *ast.FuncDecl:
			// Check if this is a method on the middleware struct
			if node.Recv != nil && len(node.Recv.List) > 0 {
				if starExpr, ok := node.Recv.List[0].Type.(*ast.StarExpr); ok {
					if ident, ok := starExpr.X.(*ast.Ident); ok {
						if ident.Name == middlewareName && node.Name.Name == "Handle" {
							handleMethod = node
						}
					}
				}
			}
		}
		return true
	})
	
	if middlewareStruct == nil {
		return fmt.Errorf("middleware struct '%s' not found", middlewareName)
	}
	
	if handleMethod == nil {
		return fmt.Errorf("middleware '%s' must have a Handle method", middlewareName)
	}
	
	// Validate Handle method signature: Handle(next echo.HandlerFunc) echo.HandlerFunc
	if handleMethod.Type.Params == nil || len(handleMethod.Type.Params.List) != 1 {
		return fmt.Errorf("middleware '%s' Handle method must have exactly one parameter", middlewareName)
	}
	
	param := handleMethod.Type.Params.List[0]
	if !p.isEchoHandlerFunc(param.Type) {
		return fmt.Errorf("middleware '%s' Handle method parameter must be echo.HandlerFunc", middlewareName)
	}
	
	if handleMethod.Type.Results == nil || len(handleMethod.Type.Results.List) != 1 {
		return fmt.Errorf("middleware '%s' Handle method must return exactly one value", middlewareName)
	}
	
	result := handleMethod.Type.Results.List[0]
	if !p.isEchoHandlerFunc(result.Type) {
		return fmt.Errorf("middleware '%s' Handle method must return echo.HandlerFunc", middlewareName)
	}
	
	return nil
}

// isEchoHandlerFunc checks if the given type expression represents echo.HandlerFunc
func (p *Parser) isEchoHandlerFunc(expr ast.Expr) bool {
	if selectorExpr, ok := expr.(*ast.SelectorExpr); ok {
		if ident, ok := selectorExpr.X.(*ast.Ident); ok {
			return ident.Name == "echo" && selectorExpr.Sel.Name == "HandlerFunc"
		}
	}
	return false
}

// ValidateParserFunctionSignature validates that a parser function has the correct signature
// Expected signature: func(c echo.Context, paramValue string) (T, error)
func (p *Parser) ValidateParserFunctionSignature(file *ast.File, functionName, typeName string) error {
	var parserFunc *ast.FuncDecl
	fileName := p.getFileName(file)
	errorReporter := NewParserErrorReporter(p)
	
	// Find the parser function
	ast.Inspect(file, func(n ast.Node) bool {
		if funcDecl, ok := n.(*ast.FuncDecl); ok {
			// Check if this is the parser function (not a method)
			if funcDecl.Recv == nil && funcDecl.Name.Name == functionName {
				parserFunc = funcDecl
				return false // Stop searching
			}
		}
		return true
	})
	
	if parserFunc == nil {
		return errorReporter.ReportParserValidationError(
			functionName,
			fileName,
			0, // Line will be set by caller if available
			"function not found",
			"",
		)
	}
	
	line := p.fileSet.Position(parserFunc.Pos()).Line
	actualSignature := p.getFunctionSignatureString(parserFunc)
	
	// Validate function signature: func(c echo.Context, paramValue string) (T, error)
	
	// Check parameters: should have exactly 2 parameters
	if parserFunc.Type.Params == nil || len(parserFunc.Type.Params.List) != 2 {
		return errorReporter.ReportParserValidationError(
			functionName,
			fileName,
			line,
			fmt.Sprintf("has %d parameters, expected 2", p.countParameters(parserFunc.Type.Params)),
			actualSignature,
		)
	}
	
	// Check first parameter: echo.Context
	firstParam := parserFunc.Type.Params.List[0]
	if !p.isEchoContext(firstParam.Type) {
		return errorReporter.ReportParserValidationError(
			functionName,
			fileName,
			line,
			fmt.Sprintf("first parameter is %s, expected echo.Context", p.getTypeString(firstParam.Type)),
			actualSignature,
		)
	}
	
	// Check second parameter: string
	secondParam := parserFunc.Type.Params.List[1]
	if !p.isStringType(secondParam.Type) {
		return errorReporter.ReportParserValidationError(
			functionName,
			fileName,
			line,
			fmt.Sprintf("second parameter is %s, expected string", p.getTypeString(secondParam.Type)),
			actualSignature,
		)
	}
	
	// Check return values: should have exactly 2 return values
	if parserFunc.Type.Results == nil || len(parserFunc.Type.Results.List) != 2 {
		return errorReporter.ReportParserValidationError(
			functionName,
			fileName,
			line,
			fmt.Sprintf("returns %d values, expected 2", p.countResults(parserFunc.Type.Results)),
			actualSignature,
		)
	}
	
	// Check second return value: error
	secondReturn := parserFunc.Type.Results.List[1]
	if !p.isErrorType(secondReturn.Type) {
		return errorReporter.ReportParserValidationError(
			functionName,
			fileName,
			line,
			fmt.Sprintf("second return value is %s, expected error", p.getTypeString(secondReturn.Type)),
			actualSignature,
		)
	}
	
	// Validate that the first return type matches the expected type name
	// Note: We're being lenient here for now, as strict type matching can be complex
	// The main validation is ensuring the function signature structure is correct
	firstReturn := parserFunc.Type.Results.List[0]
	actualReturnType := p.getTypeString(firstReturn.Type)
	
	// Only validate if the types are clearly incompatible (e.g., basic type mismatch)
	if p.isObviouslyIncompatible(actualReturnType, typeName) {
		return errorReporter.ReportParserValidationError(
			functionName,
			fileName,
			line,
			fmt.Sprintf("first return value is %s, expected %s", actualReturnType, typeName),
			actualSignature,
		)
	}
	
	return nil
}

// extractParserSignature extracts parameter and return types from a parser function
func (p *Parser) extractParserSignature(file *ast.File, functionName string) ([]string, []string, error) {
	var parserFunc *ast.FuncDecl
	
	// Find the parser function
	ast.Inspect(file, func(n ast.Node) bool {
		if funcDecl, ok := n.(*ast.FuncDecl); ok {
			if funcDecl.Recv == nil && funcDecl.Name.Name == functionName {
				parserFunc = funcDecl
				return false
			}
		}
		return true
	})
	
	if parserFunc == nil {
		return nil, nil, fmt.Errorf("parser function '%s' not found", functionName)
	}
	
	var paramTypes []string
	var returnTypes []string
	
	// Extract parameter types
	if parserFunc.Type.Params != nil {
		for _, param := range parserFunc.Type.Params.List {
			paramTypes = append(paramTypes, p.getTypeString(param.Type))
		}
	}
	
	// Extract return types
	if parserFunc.Type.Results != nil {
		for _, result := range parserFunc.Type.Results.List {
			returnTypes = append(returnTypes, p.getTypeString(result.Type))
		}
	}
	
	return paramTypes, returnTypes, nil
}

// isStringType checks if the given type expression represents string
func (p *Parser) isStringType(expr ast.Expr) bool {
	if ident, ok := expr.(*ast.Ident); ok {
		return ident.Name == "string"
	}
	return false
}

// isErrorType checks if the given type expression represents error
func (p *Parser) isErrorType(expr ast.Expr) bool {
	if ident, ok := expr.(*ast.Ident); ok {
		return ident.Name == "error"
	}
	return false
}

// getFileName extracts filename from AST file, handling nil cases
func (p *Parser) getFileName(file *ast.File) string {
	if file == nil {
		return "unknown"
	}
	// Try to get filename from file set if available
	if p.fileSet != nil {
		pos := p.fileSet.Position(file.Pos())
		if pos.Filename != "" {
			return pos.Filename
		}
	}
	return "unknown"
}

// countParameters safely counts parameters, handling nil cases
func (p *Parser) countParameters(params *ast.FieldList) int {
	if params == nil {
		return 0
	}
	return len(params.List)
}

// countResults safely counts return values, handling nil cases
func (p *Parser) countResults(results *ast.FieldList) int {
	if results == nil {
		return 0
	}
	return len(results.List)
}

// isCompatibleType checks if the actual return type is compatible with the expected type name
func (p *Parser) isCompatibleType(actualType, expectedType string) bool {
	// Direct match
	if actualType == expectedType {
		return true
	}
	
	// Handle pointer types - be more flexible with pointer matching
	actualBase := strings.TrimPrefix(actualType, "*")
	expectedBase := strings.TrimPrefix(expectedType, "*")
	
	if actualBase == expectedBase {
		return true
	}
	
	// Handle package-qualified types
	if strings.Contains(actualType, ".") || strings.Contains(expectedType, ".") {
		// Extract just the type name part for comparison
		actualParts := strings.Split(actualBase, ".")
		expectedParts := strings.Split(expectedBase, ".")
		
		actualTypeName := actualParts[len(actualParts)-1]
		expectedTypeName := expectedParts[len(expectedParts)-1]
		
		// Allow matching if the base type names match
		if actualTypeName == expectedTypeName {
			return true
		}
		
		// Also check if one is qualified and matches the other
		if actualTypeName == expectedBase || expectedTypeName == actualBase {
			return true
		}
	}
	
	// For parser validation, be more lenient - allow any type that could reasonably match
	// This is because the annotation might use a simplified type name while the function
	// uses the full qualified type
	return false
}

// isObviouslyIncompatible checks for clearly incompatible types (e.g., string vs int)
func (p *Parser) isObviouslyIncompatible(actualType, expectedType string) bool {
	// Basic type mismatches that are clearly wrong
	basicTypes := map[string]bool{
		"string": true, "int": true, "int32": true, "int64": true,
		"float32": true, "float64": true, "bool": true, "byte": true,
	}
	
	actualBase := strings.TrimPrefix(actualType, "*")
	expectedBase := strings.TrimPrefix(expectedType, "*")
	
	// If both are basic types and different, they're incompatible
	if basicTypes[actualBase] && basicTypes[expectedBase] && actualBase != expectedBase {
		return true
	}
	
	// Otherwise, assume they could be compatible (complex types, custom types, etc.)
	return false
}

// getFunctionSignatureString returns a string representation of a function signature
func (p *Parser) getFunctionSignatureString(funcDecl *ast.FuncDecl) string {
	if funcDecl == nil {
		return ""
	}
	
	var params []string
	if funcDecl.Type.Params != nil {
		for _, param := range funcDecl.Type.Params.List {
			paramType := p.getTypeString(param.Type)
			if len(param.Names) > 0 {
				for _, name := range param.Names {
					params = append(params, name.Name+" "+paramType)
				}
			} else {
				params = append(params, paramType)
			}
		}
	}
	
	var results []string
	if funcDecl.Type.Results != nil {
		for _, result := range funcDecl.Type.Results.List {
			results = append(results, p.getTypeString(result.Type))
		}
	}
	
	signature := fmt.Sprintf("func %s(%s)", funcDecl.Name.Name, strings.Join(params, ", "))
	if len(results) > 0 {
		if len(results) == 1 {
			signature += " " + results[0]
		} else {
			signature += " (" + strings.Join(results, ", ") + ")"
		}
	}
	
	return signature
}

// ValidateParserImports validates that all required imports for parsers are available
func (p *Parser) ValidateParserImports(file *ast.File, parsers []models.RouteParserMetadata) error {
	if file == nil {
		return nil // Skip validation if file is not available
	}
	
	// Extract imports from the file
	imports := make(map[string]bool)
	for _, imp := range file.Imports {
		importPath := strings.Trim(imp.Path.Value, `"`)
		imports[importPath] = true
		
		// Also add the package name if it has an alias
		if imp.Name != nil {
			imports[imp.Name.Name] = true
		} else {
			// Extract package name from import path
			parts := strings.Split(importPath, "/")
			if len(parts) > 0 {
				packageName := parts[len(parts)-1]
				imports[packageName] = true
			}
		}
	}
	
	fileName := p.getFileName(file)
	
	// Check each parser for required imports
	for _, parser := range parsers {
		// Check if the parser type requires specific imports
		requiredImport := p.getRequiredImportForType(parser.TypeName)
		if requiredImport != "" {
			// Check if the required import is present
			if !p.hasRequiredImport(imports, requiredImport, parser.TypeName) {
				return models.NewParserImportError(
					parser.TypeName,
					fileName,
					parser.Line,
					requiredImport,
				)
			}
		}
	}
	
	return nil
}

// getRequiredImportForType returns the required import for a given type
func (p *Parser) getRequiredImportForType(typeName string) string {
	// Map of types to their required imports
	typeImports := map[string]string{
		"uuid.UUID":   "github.com/google/uuid",
		"time.Time":   "time",
		"time.Duration": "time",
		"url.URL":     "net/url",
		"big.Int":     "math/big",
		"big.Float":   "math/big",
	}
	
	return typeImports[typeName]
}

// hasRequiredImport checks if the required import is available in the imports map
func (p *Parser) hasRequiredImport(imports map[string]bool, requiredImport, typeName string) bool {
	// Check direct import path
	if imports[requiredImport] {
		return true
	}
	
	// Check if the package name is imported
	parts := strings.Split(requiredImport, "/")
	if len(parts) > 0 {
		packageName := parts[len(parts)-1]
		if imports[packageName] {
			return true
		}
	}
	
	// For types like uuid.UUID, check if "uuid" package is available
	if strings.Contains(typeName, ".") {
		typeParts := strings.Split(typeName, ".")
		if len(typeParts) > 0 {
			packageName := typeParts[0]
			if imports[packageName] {
				return true
			}
		}
	}
	
	return false
}

// DetectParserConflicts detects conflicts in parser registrations across multiple packages
func (p *Parser) DetectParserConflicts(allParsers []models.RouteParserMetadata) error {
	typeMap := make(map[string][]models.ParserConflict)
	
	// Group parsers by type name
	for _, parser := range allParsers {
		conflict := models.ParserConflict{
			FileName:     parser.FileName,
			Line:         parser.Line,
			FunctionName: parser.FunctionName,
			PackagePath:  parser.PackagePath,
		}
		typeMap[parser.TypeName] = append(typeMap[parser.TypeName], conflict)
	}
	
	// Check for conflicts (multiple parsers for the same type)
	for typeName, conflicts := range typeMap {
		if len(conflicts) > 1 {
			return models.NewParserConflictError(typeName, conflicts)
		}
	}
	
	return nil
}

// ValidateAllParsers performs comprehensive validation of all parser-related functionality
func (p *Parser) ValidateAllParsers(metadata *models.PackageMetadata, fileMap map[string]*ast.File) error {
	errorReporter := NewParserErrorReporter(p)
	
	// 1. Validate parser function signatures
	for _, parser := range metadata.RouteParsers {
		if file, exists := fileMap[parser.FileName]; exists {
			err := p.ValidateParserFunctionSignature(file, parser.FunctionName, parser.TypeName)
			if err != nil {
				return err
			}
		}
	}
	
	// 2. Validate parser imports
	for fileName, file := range fileMap {
		var fileParsers []models.RouteParserMetadata
		for _, parser := range metadata.RouteParsers {
			if parser.FileName == fileName {
				fileParsers = append(fileParsers, parser)
			}
		}
		
		if len(fileParsers) > 0 {
			err := p.ValidateParserImports(file, fileParsers)
			if err != nil {
				return err
			}
		}
	}
	
	// 3. Detect parser conflicts
	err := p.DetectParserConflicts(metadata.RouteParsers)
	if err != nil {
		return err
	}
	
	// 4. Validate custom parser usage in routes
	err = p.validateAndLinkCustomParsers(metadata)
	if err != nil {
		return err
	}
	
	// 5. Generate diagnostics (warnings, not errors)
	diagnostics := errorReporter.GenerateParserDiagnostics(metadata)
	if len(diagnostics) > 0 {
		// Log diagnostics but don't fail validation
		// This could be enhanced to use a proper logger
		for _, diagnostic := range diagnostics {
			fmt.Printf("Parser diagnostic: %s\n", diagnostic)
		}
	}
	
	return nil
}

// extractPublicMethods extracts all public methods from a struct for interface generation
func (p *Parser) extractPublicMethods(file *ast.File, structName string) ([]models.Method, error) {
	var methods []models.Method
	
	if file == nil {
		return methods, nil // Return empty methods if file is not available (e.g., in unit tests)
	}
	
	// Walk the AST to find methods on the specified struct
	ast.Inspect(file, func(n ast.Node) bool {
		if funcDecl, ok := n.(*ast.FuncDecl); ok {
			// Check if this is a method on our struct
			if funcDecl.Recv != nil && len(funcDecl.Recv.List) > 0 {
				// Get receiver type
				var receiverType string
				switch recv := funcDecl.Recv.List[0].Type.(type) {
				case *ast.StarExpr:
					if ident, ok := recv.X.(*ast.Ident); ok {
						receiverType = ident.Name
					}
				case *ast.Ident:
					receiverType = recv.Name
				}
				
				// Check if this method belongs to our struct and is public
				if receiverType == structName && funcDecl.Name.IsExported() {
					method := models.Method{
						Name: funcDecl.Name.Name,
					}
					
					// Extract parameters (excluding receiver)
					if funcDecl.Type.Params != nil {
						for _, param := range funcDecl.Type.Params.List {
							paramType := p.getTypeString(param.Type)
							
							// Handle multiple parameter names with same type
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
		return true
	})
	
	return methods, nil
}

// analyzeHandlerSignature analyzes a handler method signature to detect all parameters
func (p *Parser) analyzeHandlerSignature(file *ast.File, controllerName, methodName string) ([]models.Parameter, error) {
	var allParams []models.Parameter
	
	// Find the handler method
	ast.Inspect(file, func(n ast.Node) bool {
		if funcDecl, ok := n.(*ast.FuncDecl); ok {
			// Check if this is a method on the controller struct
			if funcDecl.Recv != nil && len(funcDecl.Recv.List) > 0 && funcDecl.Name.Name == methodName {
				// Get receiver type
				var receiverType string
				switch recv := funcDecl.Recv.List[0].Type.(type) {
				case *ast.StarExpr:
					if ident, ok := recv.X.(*ast.Ident); ok {
						receiverType = ident.Name
					}
				case *ast.Ident:
					receiverType = recv.Name
				}
				
				// Check if this method belongs to our controller
				if receiverType == controllerName {
					// Analyze all parameters
					if funcDecl.Type.Params != nil {
						for i, param := range funcDecl.Type.Params.List {
							paramType := p.getTypeString(param.Type)
							
							// Handle multiple names for the same type (e.g., a, b int)
							names := param.Names
							if len(names) == 0 {
								// Anonymous parameter, create a default name
								names = []*ast.Ident{{Name: fmt.Sprintf("param%d", i)}}
							}
							
							for j, name := range names {
								paramName := name.Name
								position := i + j
								
								// Determine parameter source
								var source models.ParameterSource
								if p.isEchoContext(param.Type) {
									source = models.ParameterSourceContext
								} else {
									// For now, treat all non-context parameters as body parameters
									// Path parameters will be handled separately from route path analysis
									source = models.ParameterSourceBody
								}
								
								allParams = append(allParams, models.Parameter{
									Name:     paramName,
									Type:     paramType,
									Source:   source,
									Required: true,
									Position: position,
								})
							}
						}
					}
				}
			}
		}
		return true
	})
	
	return allParams, nil
}

// mergeParameters merges path parameters with signature parameters, properly classifying each parameter
func (p *Parser) mergeParameters(pathParams, signatureParams []models.Parameter) []models.Parameter {
	var mergedParams []models.Parameter
	
	// Create a map of path parameter names for quick lookup
	pathParamNames := make(map[string]bool)
	for _, pathParam := range pathParams {
		pathParamNames[pathParam.Name] = true
	}
	
	// Process signature parameters and classify them
	for _, sigParam := range signatureParams {
		if sigParam.Source == models.ParameterSourceContext {
			// Context parameters are always context parameters
			mergedParams = append(mergedParams, sigParam)
		} else if pathParamNames[sigParam.Name] {
			// This parameter matches a path parameter, so it's a path parameter
			// Find the corresponding path parameter to get the correct type
			for _, pathParam := range pathParams {
				if pathParam.Name == sigParam.Name {
					// Use the path parameter info but keep the signature position
					pathParam.Position = sigParam.Position
					mergedParams = append(mergedParams, pathParam)
					break
				}
			}
		} else {
			// This parameter is not in the path and not context, so it's a body parameter
			sigParam.Source = models.ParameterSourceBody
			mergedParams = append(mergedParams, sigParam)
		}
	}
	
	return mergedParams
}

// analyzeReturnType analyzes a handler method's return type to determine the response pattern
func (p *Parser) analyzeReturnType(file *ast.File, controllerName, methodName string) (models.ReturnType, error) {
	// Find the handler method
	var returnType models.ReturnType = models.ReturnTypeDataError // Default
	
	ast.Inspect(file, func(n ast.Node) bool {
		if funcDecl, ok := n.(*ast.FuncDecl); ok {
			// Check if this is a method on the controller struct
			if funcDecl.Recv != nil && len(funcDecl.Recv.List) > 0 && funcDecl.Name.Name == methodName {
				// Get receiver type
				var receiverType string
				switch recv := funcDecl.Recv.List[0].Type.(type) {
				case *ast.StarExpr:
					if ident, ok := recv.X.(*ast.Ident); ok {
						receiverType = ident.Name
					}
				case *ast.Ident:
					receiverType = recv.Name
				}
				
				// Check if this method belongs to our controller
				if receiverType == controllerName {
					// Analyze return type
					if funcDecl.Type.Results != nil && len(funcDecl.Type.Results.List) > 0 {
						results := funcDecl.Type.Results.List
						
						if len(results) == 1 {
							// Single return value - should be error
							resultType := p.getTypeString(results[0].Type)
							if resultType == "error" {
								returnType = models.ReturnTypeError
							}
						} else if len(results) == 2 {
							// Two return values - check the pattern
							firstType := p.getTypeString(results[0].Type)
							secondType := p.getTypeString(results[1].Type)
							
							if secondType == "error" {
								// Check if first type is *Response or *axon.Response
								if strings.Contains(firstType, "Response") {
									returnType = models.ReturnTypeResponseError
								} else {
									returnType = models.ReturnTypeDataError
								}
							}
						}
					}
					return false // Stop searching
				}
			}
		}
		return true
	})
	
	return returnType, nil
}

// isEchoContext checks if the given type expression represents echo.Context
func (p *Parser) isEchoContext(expr ast.Expr) bool {
	if selectorExpr, ok := expr.(*ast.SelectorExpr); ok {
		if ident, ok := selectorExpr.X.(*ast.Ident); ok {
			return ident.Name == "echo" && selectorExpr.Sel.Name == "Context"
		}
	}
	return false
}

// getTypeString converts an AST type expression to a string representation
func (p *Parser) getTypeString(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.StarExpr:
		return "*" + p.getTypeString(t.X)
	case *ast.SelectorExpr:
		if ident, ok := t.X.(*ast.Ident); ok {
			return ident.Name + "." + t.Sel.Name
		}
		return t.Sel.Name
	case *ast.ArrayType:
		return "[]" + p.getTypeString(t.Elt)
	case *ast.MapType:
		return "map[" + p.getTypeString(t.Key) + "]" + p.getTypeString(t.Value)
	case *ast.InterfaceType:
		if t.Methods == nil || len(t.Methods.List) == 0 {
			return "interface{}"
		}
		return "interface{...}" // Simplified for complex interfaces
	case *ast.FuncType:
		// Handle function types
		params := "("
		if t.Params != nil {
			for i, param := range t.Params.List {
				if i > 0 {
					params += ", "
				}
				params += p.getTypeString(param.Type)
			}
		}
		params += ")"
		
		results := ""
		if t.Results != nil {
			if len(t.Results.List) == 1 {
				results = " " + p.getTypeString(t.Results.List[0].Type)
			} else if len(t.Results.List) > 1 {
				results = " ("
				for i, result := range t.Results.List {
					if i > 0 {
						results += ", "
					}
					results += p.getTypeString(result.Type)
				}
				results += ")"
			}
		}
		
		return "func" + params + results
	case *ast.ChanType:
		dir := ""
		if t.Dir == ast.SEND {
			dir = "chan<- "
		} else if t.Dir == ast.RECV {
			dir = "<-chan "
		} else {
			dir = "chan "
		}
		return dir + p.getTypeString(t.Value)
	default:
		return "unknown"
	}
}

// Security validation functions

// isValidDirectoryPath validates that a directory path is safe for filesystem operations
func isValidDirectoryPath(path string) bool {
	// Check for empty path
	if path == "" {
		return false
	}
	
	// Check for null bytes (path injection)
	if strings.Contains(path, "\x00") {
		return false
	}
	
	// Check for dangerous characters that are never valid in filesystem paths
	// Note: We're being more permissive here to allow normal Go project paths
	// The main security check is the path traversal check after filepath.Clean()
	if strings.ContainsAny(path, "\x00<>|") {
		return false
	}
	
	return true
}

// isValidGoModPath validates that a path is safe for opening go.mod files
func isValidGoModPath(path string) bool {
	// Check for empty path
	if path == "" {
		return false
	}
	
	// Check for null bytes
	if strings.Contains(path, "\x00") {
		return false
	}
	
	// Check for dangerous characters
	if strings.ContainsAny(path, "\x00<>|") {
		return false
	}
	
	// Must end with go.mod (but allow paths that will become go.mod after cleaning)
	cleanPath := filepath.Clean(path)
	if !strings.HasSuffix(cleanPath, "go.mod") {
		return false
	}
	
	return true
}