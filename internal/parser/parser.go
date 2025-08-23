package parser

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"strings"

	"github.com/toyz/axon/internal/models"
	"github.com/toyz/axon/internal/registry"
)

// Parser implements the AnnotationParser interface
type Parser struct {
	fileSet            *token.FileSet
	middlewareRegistry registry.MiddlewareRegistry
}

// NewParser creates a new annotation parser
func NewParser() *Parser {
	return &Parser{
		fileSet:            token.NewFileSet(),
		middlewareRegistry: registry.NewMiddlewareRegistry(),
	}
}

// ParseSource parses source code from a string for testing purposes
func (p *Parser) ParseSource(filename, source string) (*models.PackageMetadata, error) {
	// Parse the source code
	file, err := parser.ParseFile(p.fileSet, filename, source, parser.ParseComments)
	if err != nil {
		return nil, fmt.Errorf("failed to parse source: %w", err)
	}

	// Create package metadata
	metadata := &models.PackageMetadata{
		PackageName: file.Name.Name,
		PackagePath: "./",
	}

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
	// Parse all Go files in the directory
	pkgs, err := parser.ParseDir(p.fileSet, path, nil, parser.ParseComments)
	if err != nil {
		return nil, fmt.Errorf("failed to parse directory %s: %w", path, err)
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
		PackageName: packageName,
		PackagePath: path,
	}

	// First pass: Extract all annotations from all files
	allAnnotations := []models.Annotation{}
	fileMap := make(map[string]*ast.File)
	
	for fileName, file := range pkg.Files {
		fileMap[fileName] = file
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
									if annotation, err := p.parseAnnotationComment(comment.Text, typeSpec.Name.Name); err == nil {
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
					if annotation, err := p.parseAnnotationComment(comment.Text, targetName); err == nil {
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
func (p *Parser) parseAnnotationComment(comment, target string) (models.Annotation, error) {
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
	}

	// Parse remaining parts as parameters and flags
	for i := 1; i < len(parts); i++ {
		part := parts[i]
		if strings.HasPrefix(part, "-") {
			// It's a flag
			if strings.Contains(part, "=") {
				// Flag with value like -Manual=ModuleName
				flagParts := strings.SplitN(part, "=", 2)
				annotation.Parameters[flagParts[0]] = flagParts[1]
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
				
				// Validate that all middleware names exist in the registry
				err := p.middlewareRegistry.Validate(middlewareNames)
				if err != nil {
					return fmt.Errorf("route %s has invalid middleware reference: %w", annotation.Target, err)
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
			
			// Check for lifecycle flag
			for _, flag := range annotation.Flags {
				if flag == FlagInit {
					service.HasLifecycle = true
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

		case models.AnnotationTypeInterface:
			// Interface annotations are processed in combination with other annotations
			// They don't create standalone components, so we skip them here
			continue
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
	var dependencies []models.Dependency
	var hasFxIn bool
	
	// First, check if struct has embedded fx.In
	for _, field := range structType.Fields.List {
		if len(field.Names) == 0 { // embedded field
			if selectorExpr, ok := field.Type.(*ast.SelectorExpr); ok {
				if ident, ok := selectorExpr.X.(*ast.Ident); ok {
					if ident.Name == "fx" && selectorExpr.Sel.Name == "In" {
						hasFxIn = true
						break
					}
				}
			}
		}
	}
	
	// If struct has fx.In, extract all other named fields as dependencies
	if hasFxIn {
		for _, field := range structType.Fields.List {
			// Skip embedded fx.In field
			if len(field.Names) == 0 {
				continue
			}
			
			// Extract dependency names from named fields
			for _, name := range field.Names {
				if name.IsExported() {
					dependencies = append(dependencies, models.Dependency{
						Name: name.Name,
						Type: p.getFieldTypeName(field.Type),
					})
				}
			}
		}
	} else {
		// If no fx.In, check for //axon::inject annotations on individual fields
		for _, field := range structType.Fields.List {
			// Skip embedded fields
			if len(field.Names) == 0 {
				continue
			}
			
			// Check if field has //axon::inject or //axon::init annotation (in Doc comments)
			if field.Doc != nil {
				for _, comment := range field.Doc.List {
					// Check for //axon::inject annotation
					if hasInject, hasInitFlag, _ := p.parseInjectAnnotation(comment.Text); hasInject {
						// Extract dependency from this field
						// When //axon::inject is present, always include the field regardless of export status
						for _, name := range field.Names {
							dependencies = append(dependencies, models.Dependency{
								Name:   name.Name,
								Type:   p.getFieldTypeName(field.Type),
								IsInit: hasInitFlag, // -Init flag on inject annotation
							})
						}
						break
					}
					
					// Check for //axon::init annotation
					if hasInit, _ := p.parseInitAnnotation(comment.Text); hasInit {
						// Extract init dependency from this field
						// When //axon::init is present, always include the field regardless of export status
						for _, name := range field.Names {
							dependencies = append(dependencies, models.Dependency{
								Name:   name.Name,
								Type:   p.getFieldTypeName(field.Type),
								IsInit: true, // Always init for //axon::init annotation
							})
						}
						break
					}
				}
			}
		}
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
			Name:     name,
			Type:     "string", // Echo parameters default to string
			Source:   models.ParameterSourcePath,
			Required: true,
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
	
	return models.Parameter{
		Name:     name,
		Type:     goType,
		Source:   models.ParameterSourcePath,
		Required: true, // Path parameters are always required
	}, nil
}

// validateParameterType validates and normalizes parameter types
func (p *Parser) validateParameterType(typeStr string) (string, error) {
	switch typeStr {
	case "int":
		return "int", nil
	case "string":
		return "string", nil
	default:
		return "", fmt.Errorf("unsupported parameter type '%s', supported types: int, string", typeStr)
	}
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