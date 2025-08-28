package templates

import (
	"bytes"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
	"text/template"

	"github.com/toyz/axon/internal/models"
)

// ParserRegistryInterface defines the interface for parser registry operations
type ParserRegistryInterface interface {
	RegisterParser(parser models.RouteParserMetadata) error
	GetParser(typeName string) (models.RouteParserMetadata, bool)
	ListParsers() []string
	HasParser(typeName string) bool
	Clear()
	GetAllParsers() map[string]models.RouteParserMetadata
}

// This package contains Go templates for code generation
// Route wrapper generation is handled in response.go

// Template constants are now defined in template_defs.go for better organization

// generateRouteRegistryCall generates the call to register the route with the global registry
func generateRouteRegistryCall(route models.RouteMetadata, controllerName, handlerVar, echoPath string, paramTypes map[string]string) string {
	var registryCall strings.Builder

	registryCall.WriteString("axon.DefaultRouteRegistry.RegisterRoute(axon.RouteInfo{\n")
	registryCall.WriteString(fmt.Sprintf("\t\tMethod:         \"%s\",\n", route.Method))
	registryCall.WriteString(fmt.Sprintf("\t\tPath:           \"%s\",\n", route.Path))
	registryCall.WriteString(fmt.Sprintf("\t\tEchoPath:       \"%s\",\n", echoPath))
	registryCall.WriteString(fmt.Sprintf("\t\tHandlerName:    \"%s\",\n", route.HandlerName))
	registryCall.WriteString(fmt.Sprintf("\t\tControllerName: \"%s\",\n", controllerName))
	registryCall.WriteString("\t\tPackageName:    \"PACKAGE_NAME\",\n") // Will be replaced by generator

	// Generate middlewares array
	if len(route.Middlewares) > 0 {
		registryCall.WriteString("\t\tMiddlewares:    []string{")
		for i, middleware := range route.Middlewares {
			if i > 0 {
				registryCall.WriteString(", ")
			}
			registryCall.WriteString(fmt.Sprintf("\"%s\"", middleware))
		}
		registryCall.WriteString("},\n")
	} else {
		registryCall.WriteString("\t\tMiddlewares:    []string{},\n")
	}

	// Generate parameter types map
	if len(paramTypes) > 0 {
		registryCall.WriteString("\t\tParameterTypes: map[string]string{")
		first := true
		for name, typ := range paramTypes {
			if !first {
				registryCall.WriteString(", ")
			}
			registryCall.WriteString(fmt.Sprintf("\"%s\": \"%s\"", name, typ))
			first = false
		}
		registryCall.WriteString("},\n")
	} else {
		registryCall.WriteString("\t\tParameterTypes: map[string]string{},\n")
	}

	registryCall.WriteString(fmt.Sprintf("\t\tHandler:        %s,\n", handlerVar))
	registryCall.WriteString("\t})")

	return registryCall.String()
}

// extractParameterTypes extracts parameter names and types from Axon route syntax
func extractParameterTypes(axonPath string) map[string]string {
	paramTypes := make(map[string]string)

	// Regex to match Axon parameter syntax: {param:type}
	re := regexp.MustCompile(`\{([^:}]+):([^}]+)\}`)
	matches := re.FindAllStringSubmatch(axonPath, -1)

	for _, match := range matches {
		if len(match) == 3 {
			paramName := match[1]
			paramType := match[2]
			paramTypes[paramName] = paramType
		}
	}

	return paramTypes
}

// Note: ParameterBindingData and GenerateParameterBinding were removed as they were unused.
// Parameter binding is now handled directly by GenerateParameterBindingCode.

// GenerateParameterBindingCode generates the complete parameter binding code for a list of parameters
func GenerateParameterBindingCode(parameters []models.Parameter, parserRegistry ParserRegistryInterface) (string, error) {
	var bindingCode strings.Builder

	for _, param := range parameters {
		switch param.Source {
		case models.ParameterSourcePath:
			// Check if this is a wildcard parameter (marked with :* suffix)
			isWildcard := strings.HasSuffix(param.Name, ":*")
			var actualParamName, paramSource string

			if isWildcard {
				// Remove the :* suffix to get the actual parameter name
				actualParamName = strings.TrimSuffix(param.Name, ":*")
				paramSource = "*" // Get value from wildcard route param
			} else {
				actualParamName = param.Name
				paramSource = param.Name // Get value from named route param
			}

			// Generate parameter binding code for path parameters
			var functionCall string

			// Use ParserFunc from parameter if available, otherwise look in registry
			if param.ParserFunc != "" {
				functionCall = param.ParserFunc
			} else {
				// Extract just the type name without package prefix for registry lookup
				typeName := param.Type
				if strings.Contains(typeName, ".") {
					parts := strings.Split(typeName, ".")
					typeName = parts[len(parts)-1] // Get the last part (type name)
				}

				parser, exists := parserRegistry.GetParser(typeName)
				if !exists {
					return "", fmt.Errorf("unsupported parameter type: %s", param.Type)
				}

				// Generate parser function call
				if parser.PackagePath == "builtin" {
					// Built-in parsers use axon package prefix
					functionCall = fmt.Sprintf("axon.%s", parser.FunctionName)
				} else if parser.PackagePath != "" {
					// Custom parsers use package.FunctionName format
					packageName := filepath.Base(parser.PackagePath)
					functionCall = fmt.Sprintf("%s.%s", packageName, parser.FunctionName)
				} else {
					// For parsers in the same package, use direct function name
					functionCall = parser.FunctionName
				}
			}

			bindingCode.WriteString(fmt.Sprintf(`		%s, err := %s(c, c.Param("%s"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Invalid %s: %%v", err))
		}
`, actualParamName, functionCall, paramSource, actualParamName))
		case models.ParameterSourceContext:
			// Context parameters don't need binding code - they're passed directly
			// The context is already available as 'c' in the wrapper function
			continue
		}
	}

	return bindingCode.String(), nil
}

// getParameterSourceString converts ParameterSource enum to string
func getParameterSourceString(source models.ParameterSource) string {
	switch source {
	case models.ParameterSourcePath:
		return "path"
	case models.ParameterSourceBody:
		return "body"
	case models.ParameterSourceContext:
		return "context"
	default:
		return "unknown"
	}
}

// CoreServiceProviderData represents data needed for core service provider generation
type CoreServiceProviderData struct {
	StructName   string
	Dependencies []DependencyData // All dependencies (for struct initialization)
	InjectedDeps []DependencyData // Only injected dependencies (for function parameters)
	HasStart     bool
	HasStop      bool
	StartMode    string // lifecycle start mode: "Same" (default) or "Background"
}

// DependencyData represents a dependency for template generation
type DependencyData struct {
	Name      string // parameter name (camelCase)
	FieldName string // struct field name (original case)
	Type      string // type of the dependency
	IsInit    bool   // whether this should be initialized (not injected)
}

// generateInitCode generates initialization code for a given type
func generateInitCode(fieldType string) string {
	// Remove pointer prefix for analysis
	baseType := strings.TrimPrefix(fieldType, "*")

	// Handle different types
	if strings.HasPrefix(baseType, "map[") {
		return fmt.Sprintf("make(%s)", fieldType)
	} else if strings.HasPrefix(baseType, "[]") {
		return fmt.Sprintf("make(%s, 0)", fieldType)
	} else if strings.Contains(baseType, "chan ") {
		return fmt.Sprintf("make(%s)", fieldType)
	} else if strings.HasPrefix(fieldType, "*") {
		// For pointer types, use nil (they should be initialized in lifecycle methods)
		return "nil"
	} else {
		// For value types, use zero value constructor
		return fmt.Sprintf("%s{}", fieldType)
	}
}

// ImportAnalysis holds the results of analyzing what imports are needed
type ImportAnalysis struct {
	Dependencies map[string]bool // package names that need to be imported
	NeedsContext bool            // whether context package is needed
	NeedsLogger  bool            // whether logger packages are needed
	StandardLib  []string        // standard library imports needed
	ThirdParty   []string        // third-party imports needed
}

// analyzeRequiredImports analyzes metadata to determine what imports are needed
func analyzeRequiredImports(metadata *models.PackageMetadata) ImportAnalysis {
	analysis := ImportAnalysis{
		Dependencies: make(map[string]bool),
		StandardLib:  []string{},
		ThirdParty:   []string{},
	}

	// Analyze core services
	for _, service := range metadata.CoreServices {
		if service.HasLifecycle {
			analysis.NeedsContext = true
		}

		// Analyze dependencies for imports
		for _, dep := range service.Dependencies {
			if packagePath := extractPackageFromType(dep.Type); packagePath != "" {
				analysis.Dependencies[packagePath] = true
			}
		}
	}

	// Analyze loggers
	for _, logger := range metadata.Loggers {
		if logger.HasLifecycle {
			analysis.NeedsContext = true
		}
		analysis.NeedsLogger = true

		// Analyze dependencies for imports
		for _, dep := range logger.Dependencies {
			if packagePath := extractPackageFromType(dep.Type); packagePath != "" {
				analysis.Dependencies[packagePath] = true
			}
		}
	}

	// Analyze interfaces
	for _, iface := range metadata.Interfaces {
		for _, method := range iface.Methods {
			// Analyze parameters
			for _, param := range method.Parameters {
				if packagePath := extractPackageFromType(param.Type); packagePath != "" {
					analysis.Dependencies[packagePath] = true
				}
			}
			// Analyze return types
			for _, ret := range method.Returns {
				if packagePath := extractPackageFromType(ret); packagePath != "" {
					analysis.Dependencies[packagePath] = true
				}
			}
		}
	}

	return analysis
}

// Note: Removed resolveImportPath and buildModuleImportPath functions
// These were making assumptions about project structure.
// Templates now receive actual package paths from the generator.

// isConfigLikeType checks if a type represents a configuration dependency
func isConfigLikeType(typeName string) bool {
	// Remove pointer prefix for analysis
	baseType := strings.TrimPrefix(typeName, "*")

	// Check for common config patterns (more flexible than before)
	lowerType := strings.ToLower(baseType)
	configPatterns := []string{"config", "configuration", "settings", "options"}

	for _, pattern := range configPatterns {
		if strings.Contains(lowerType, pattern) {
			return true
		}
	}

	return false
}

// isLoggerType checks if a type represents a logger dependency
func isLoggerType(typeName string) bool {
	// Remove pointer prefix for analysis
	baseType := strings.TrimPrefix(typeName, "*")

	// Check for common logger patterns
	loggerPatterns := []string{
		"slog.Logger",
		"log.Logger",
		"Logger",
		"log.Entry",
		"logrus.Logger",
		"zap.Logger",
	}

	for _, pattern := range loggerPatterns {
		if strings.Contains(baseType, pattern) {
			return true
		}
	}

	return false
}

// isStandardLibraryPackage checks if a package name is from the Go standard library
func isStandardLibraryPackage(packageName string) bool {
	// If it contains a dot, it's definitely not standard library
	if strings.Contains(packageName, ".") {
		return false
	}

	// Common standard library packages (not exhaustive, but covers common cases)
	commonStdLib := map[string]bool{
		"fmt": true, "os": true, "io": true, "net": true, "http": true,
		"context": true, "time": true, "strings": true, "strconv": true,
		"errors": true, "log": true, "slog": true, "json": true, "sql": true,
		"sync": true, "crypto": true, "encoding": true, "path": true,
		"sort": true, "math": true, "bytes": true, "bufio": true, "regexp": true,
	}

	// Check if it's a known standard library package
	if commonStdLib[packageName] {
		return true
	}

	// For unknown simple names, assume they are local packages that need import resolution
	return false
}

// extractPackageFromType extracts the package name from a type string like "*config.Config"
func extractPackageFromType(typeStr string) string {
	// Remove pointer prefix
	typeStr = strings.TrimPrefix(typeStr, "*")

	// Handle complex types like maps, slices, channels
	if strings.HasPrefix(typeStr, "map[") {
		// For maps, extract package from the value type
		// Find the closing bracket of the key type
		bracketCount := 0
		valueStart := -1
		for i, char := range typeStr {
			if char == '[' {
				bracketCount++
			} else if char == ']' {
				bracketCount--
				if bracketCount == 0 {
					valueStart = i + 1
					break
				}
			}
		}
		if valueStart > 0 && valueStart < len(typeStr) {
			valueType := typeStr[valueStart:]
			return extractPackageFromType(valueType) // Recursive call for value type
		}
	} else if strings.HasPrefix(typeStr, "[]") {
		// For slices, extract package from the element type
		elementType := typeStr[2:]
		return extractPackageFromType(elementType) // Recursive call for element type
	} else if strings.HasPrefix(typeStr, "chan ") {
		// For channels, extract package from the element type
		elementType := typeStr[5:]
		return extractPackageFromType(elementType) // Recursive call for element type
	}

	// For simple types, check if it contains a package qualifier
	if dotIndex := strings.Index(typeStr, "."); dotIndex != -1 {
		return typeStr[:dotIndex]
	}

	return ""
}

// extractParameterName extracts a parameter name from a type string
func extractParameterName(typeStr string) string {
	// Remove pointer prefix
	typeStr = strings.TrimPrefix(typeStr, "*")

	// If it contains a package qualifier, extract the type name
	if dotIndex := strings.LastIndex(typeStr, "."); dotIndex != -1 {
		typeName := typeStr[dotIndex+1:]
		// Convert to camelCase for parameter name
		return strings.ToLower(typeName[:1]) + typeName[1:]
	}

	// If no package qualifier, use the type name directly
	return strings.ToLower(typeStr[:1]) + typeStr[1:]
}

// GenerateCoreServiceProvider generates FX provider code for a core service
func GenerateCoreServiceProvider(service models.CoreServiceMetadata) (string, error) {
	if service.IsManual {
		// Manual services don't need generated providers
		return "", nil
	}

	// Convert Dependency models to DependencyData for templates
	var dependencies []DependencyData
	var injectedDeps []DependencyData

	for _, dep := range service.Dependencies {
		depData := DependencyData{
			Name:      dep.Name, // Keep original field name for parameter
			FieldName: dep.Name, // Original field name
			Type:      dep.Type,
			IsInit:    dep.IsInit,
		}
		dependencies = append(dependencies, depData)

		// Only add to injected deps if it's not an init dependency
		if !dep.IsInit {
			injectedDeps = append(injectedDeps, depData)
		}
	}

	data := CoreServiceProviderData{
		StructName:   service.StructName,
		Dependencies: dependencies,
		InjectedDeps: injectedDeps,
		HasStart:     service.HasStart,
		HasStop:      service.HasStop,
	}

	// Handle different lifecycle modes
	if service.Mode == "Transient" {
		// For transient services, generate a factory function
		return generateTransientServiceProvider(data)
	} else {
		// Default Singleton mode
		if service.HasLifecycle && service.StartMode != "" {
			// For services with -Init flag (StartMode is set), generate simple provider only
			// The invoke function will be generated separately
			return executeTemplate("init-provider", InitProviderTemplate, data)
		} else if service.HasLifecycle {
			// For old-style lifecycle services (no -Init flag), generate embedded lifecycle hooks
			return executeTemplate("lifecycle-provider", LifecycleProviderTemplate, data)
		} else if len(service.Dependencies) > 0 {
			// Use regular provider template for services with dependencies
			return executeTemplate("provider", ProviderTemplate, data)
		} else {
			// Use FX provider template for services with no dependencies
			return executeTemplate("fx-provider", FXProviderTemplate, data)
		}
	}
}

// generateTransientServiceProvider generates a factory function for transient services
func generateTransientServiceProvider(data CoreServiceProviderData) (string, error) {
	return executeTemplate("transient-provider", TransientProviderTemplate, data)
}

// GenerateInitInvokeFunction generates an invoke function for lifecycle management
func GenerateInitInvokeFunction(service models.CoreServiceMetadata) (string, error) {
	if !service.HasLifecycle {
		return "", nil
	}

	data := CoreServiceProviderData{
		StructName: service.StructName,
		HasStart:   service.HasStart,
		HasStop:    service.HasStop,
		StartMode:  service.StartMode,
	}

	return executeTemplate("init-invoke", InitInvokeTemplate, data)
}

// GenerateCoreServiceModule generates the complete FX module for core services in a package
func GenerateCoreServiceModule(metadata *models.PackageMetadata) (string, error) {
	return GenerateCoreServiceModuleWithModule(metadata, "")
}

// GenerateCoreServiceModuleWithModule generates the complete FX module for core services in a package with module name
func GenerateCoreServiceModuleWithModule(metadata *models.PackageMetadata, moduleName string) (string, error) {
	return GenerateCoreServiceModuleWithResolver(metadata, moduleName, nil)
}

// PackagePathMap maps package names to their actual import paths
type PackagePathMap map[string]string

// GenerateCoreServiceModuleWithResolver generates the complete FX module with actual package paths
func GenerateCoreServiceModuleWithResolver(metadata *models.PackageMetadata, moduleName string, packagePaths PackagePathMap) (string, error) {
	if packagePaths == nil {
		packagePaths = make(PackagePathMap)
	}
	var moduleBuilder strings.Builder

	// Generate package declaration with DO NOT EDIT header
	moduleBuilder.WriteString("// Code generated by Axon framework. DO NOT EDIT.\n")
	moduleBuilder.WriteString("// This file was automatically generated and should not be modified manually.\n\n")
	moduleBuilder.WriteString(fmt.Sprintf("package %s\n\n", metadata.PackageName))

	// Create ImportManager for proper import detection
	importManager := NewImportManager()

	// Set up package resolver if module name is provided
	if moduleName != "" {
		resolver := &PackageResolver{
			ModulePath: moduleName,
			PackageMap: make(map[string]string),
		}
		importManager.packageResolver = resolver
	}
	// Generate all code content first, then analyze imports
	var contentBuilder strings.Builder

	// Generate fxLogger adapter if there are loggers
	if len(metadata.Loggers) > 0 {
		firstLogger := metadata.Loggers[0]
		contentBuilder.WriteString(fmt.Sprintf("// fxLogger adapts %s to fxevent.Logger\n", firstLogger.StructName))
		contentBuilder.WriteString(fmt.Sprintf("type fxLogger struct {\n\tlogger *%s\n}\n\n", firstLogger.StructName))

		// Implement fxevent.Logger interface
		contentBuilder.WriteString("func (l *fxLogger) LogEvent(event fxevent.Event) {\n")
		contentBuilder.WriteString("\tswitch e := event.(type) {\n")
		contentBuilder.WriteString("\tcase *fxevent.OnStartExecuting:\n")
		contentBuilder.WriteString("\t\tl.logger.Info(\"OnStart hook executing\", \"callee\", e.FunctionName, \"caller\", e.CallerName)\n")
		contentBuilder.WriteString("\tcase *fxevent.OnStartExecuted:\n")
		contentBuilder.WriteString("\t\tif e.Err != nil {\n")
		contentBuilder.WriteString("\t\t\tl.logger.Error(\"OnStart hook failed\", \"callee\", e.FunctionName, \"caller\", e.CallerName, \"error\", e.Err)\n")
		contentBuilder.WriteString("\t\t} else {\n")
		contentBuilder.WriteString("\t\t\tl.logger.Info(\"OnStart hook executed\", \"callee\", e.FunctionName, \"caller\", e.CallerName, \"runtime\", e.Runtime)\n")
		contentBuilder.WriteString("\t\t}\n")
		contentBuilder.WriteString("\tcase *fxevent.OnStopExecuting:\n")
		contentBuilder.WriteString("\t\tl.logger.Info(\"OnStop hook executing\", \"callee\", e.FunctionName, \"caller\", e.CallerName)\n")
		contentBuilder.WriteString("\tcase *fxevent.OnStopExecuted:\n")
		contentBuilder.WriteString("\t\tif e.Err != nil {\n")
		contentBuilder.WriteString("\t\t\tl.logger.Error(\"OnStop hook failed\", \"callee\", e.FunctionName, \"caller\", e.CallerName, \"error\", e.Err)\n")
		contentBuilder.WriteString("\t\t} else {\n")
		contentBuilder.WriteString("\t\t\tl.logger.Info(\"OnStop hook executed\", \"callee\", e.FunctionName, \"caller\", e.CallerName, \"runtime\", e.Runtime)\n")
		contentBuilder.WriteString("\t\t}\n")
		contentBuilder.WriteString("\tcase *fxevent.Supplied:\n")
		contentBuilder.WriteString("\t\tl.logger.Debug(\"supplied\", \"type\", e.TypeName, \"module\", e.ModuleName)\n")
		contentBuilder.WriteString("\tcase *fxevent.Provided:\n")
		contentBuilder.WriteString("\t\tl.logger.Debug(\"provided\", \"constructor\", e.ConstructorName, \"module\", e.ModuleName)\n")
		contentBuilder.WriteString("\tcase *fxevent.Invoking:\n")
		contentBuilder.WriteString("\t\tl.logger.Debug(\"invoking\", \"function\", e.FunctionName, \"module\", e.ModuleName)\n")
		contentBuilder.WriteString("\tcase *fxevent.Invoked:\n")
		contentBuilder.WriteString("\t\tif e.Err != nil {\n")
		contentBuilder.WriteString("\t\t\tl.logger.Error(\"invoke failed\", \"error\", e.Err, \"stack\", e.Trace, \"function\", e.FunctionName, \"module\", e.ModuleName)\n")
		contentBuilder.WriteString("\t\t} else {\n")
		contentBuilder.WriteString("\t\t\tl.logger.Debug(\"invoked\", \"function\", e.FunctionName, \"module\", e.ModuleName)\n")
		contentBuilder.WriteString("\t\t}\n")
		contentBuilder.WriteString("\tcase *fxevent.Stopping:\n")
		contentBuilder.WriteString("\t\tl.logger.Info(\"received signal\", \"signal\", e.Signal)\n")
		contentBuilder.WriteString("\tcase *fxevent.Stopped:\n")
		contentBuilder.WriteString("\t\tif e.Err != nil {\n")
		contentBuilder.WriteString("\t\t\tl.logger.Error(\"stop failed\", \"error\", e.Err)\n")
		contentBuilder.WriteString("\t\t} else {\n")
		contentBuilder.WriteString("\t\t\tl.logger.Info(\"stopped\")\n")
		contentBuilder.WriteString("\t\t}\n")
		contentBuilder.WriteString("\tcase *fxevent.RollingBack:\n")
		contentBuilder.WriteString("\t\tl.logger.Error(\"start failed, rolling back\", \"error\", e.StartErr)\n")
		contentBuilder.WriteString("\tcase *fxevent.RolledBack:\n")
		contentBuilder.WriteString("\t\tif e.Err != nil {\n")
		contentBuilder.WriteString("\t\t\tl.logger.Error(\"rollback failed\", \"error\", e.Err)\n")
		contentBuilder.WriteString("\t\t} else {\n")
		contentBuilder.WriteString("\t\t\tl.logger.Info(\"rolled back\")\n")
		contentBuilder.WriteString("\t\t}\n")
		contentBuilder.WriteString("\tcase *fxevent.Started:\n")
		contentBuilder.WriteString("\t\tif e.Err != nil {\n")
		contentBuilder.WriteString("\t\t\tl.logger.Error(\"start failed\", \"error\", e.Err)\n")
		contentBuilder.WriteString("\t\t} else {\n")
		contentBuilder.WriteString("\t\t\tl.logger.Info(\"started\")\n")
		contentBuilder.WriteString("\t\t}\n")
		contentBuilder.WriteString("\tcase *fxevent.LoggerInitialized:\n")
		contentBuilder.WriteString("\t\tif e.Err != nil {\n")
		contentBuilder.WriteString("\t\t\tl.logger.Error(\"custom logger initialization failed\", \"error\", e.Err)\n")
		contentBuilder.WriteString("\t\t} else {\n")
		contentBuilder.WriteString("\t\t\tl.logger.Debug(\"initialized custom fxevent.Logger\", \"function\", e.ConstructorName)\n")
		contentBuilder.WriteString("\t\t}\n")
		contentBuilder.WriteString("\t}\n")
		contentBuilder.WriteString("}\n\n")
	}

	// Generate interfaces
	for _, iface := range metadata.Interfaces {
		interfaceCode, err := GenerateInterface(iface)
		if err != nil {
			return "", fmt.Errorf("failed to generate interface %s: %w", iface.Name, err)
		}

		contentBuilder.WriteString(interfaceCode)
		contentBuilder.WriteString("\n\n")
	}

	// Generate provider functions for each core service
	for _, service := range metadata.CoreServices {
		if service.IsManual {
			continue // Skip manual services
		}

		provider, err := GenerateCoreServiceProvider(service)
		if err != nil {
			return "", fmt.Errorf("failed to generate provider for service %s: %w", service.Name, err)
		}

		if provider != "" {
			contentBuilder.WriteString(provider)
			contentBuilder.WriteString("\n\n")
		}

		// Generate invoke function for services with -Init flag
		if service.HasLifecycle {
			invokeFunc, err := GenerateInitInvokeFunction(service)
			if err != nil {
				return "", fmt.Errorf("failed to generate invoke function for service %s: %w", service.Name, err)
			}

			if invokeFunc != "" {
				contentBuilder.WriteString(invokeFunc)
				contentBuilder.WriteString("\n\n")
			}
		}
	}

	// Generate provider functions for each logger
	for _, logger := range metadata.Loggers {
		if logger.IsManual {
			continue // Skip manual loggers
		}

		provider, err := GenerateLoggerProvider(logger)
		if err != nil {
			return "", fmt.Errorf("failed to generate provider for logger %s: %w", logger.Name, err)
		}

		if provider != "" {
			contentBuilder.WriteString(provider)
			contentBuilder.WriteString("\n\n")
		}
	}

	// Generate interface providers
	for _, iface := range metadata.Interfaces {
		providerCode, err := GenerateInterfaceProvider(iface)
		if err != nil {
			return "", fmt.Errorf("failed to generate interface provider %s: %w", iface.Name, err)
		}

		contentBuilder.WriteString(providerCode)
		contentBuilder.WriteString("\n\n")
	}

	// Generate module variable
	contentBuilder.WriteString("// AutogenModule provides all core services in this package\n")
	contentBuilder.WriteString(fmt.Sprintf("var AutogenModule = fx.Module(\"%s\",\n", metadata.PackageName))

	// Add fx.WithLogger if there are loggers
	if len(metadata.Loggers) > 0 {
		// Use the first logger as the FX logger
		firstLogger := metadata.Loggers[0]
		contentBuilder.WriteString(fmt.Sprintf("\tfx.WithLogger(func(logger *%s) fxevent.Logger {\n", firstLogger.StructName))
		contentBuilder.WriteString("\t\treturn &fxLogger{logger: logger}\n")
		contentBuilder.WriteString("\t}),\n")
	}

	for _, service := range metadata.CoreServices {
		if service.IsManual {
			// Reference manual module
			if service.ModuleName != "" {
				contentBuilder.WriteString(fmt.Sprintf("\t%s,\n", service.ModuleName))
			}
		} else if service.Mode == "Transient" {
			// Transient services provide a factory function
			contentBuilder.WriteString(fmt.Sprintf("\tfx.Provide(New%sFactory),\n", service.StructName))
		} else {
			// Singleton services (default) use fx.Provide to make them available for dependency injection
			contentBuilder.WriteString(fmt.Sprintf("\tfx.Provide(New%s),\n", service.StructName))

			// Add fx.Invoke for services with -Init flag
			if service.HasLifecycle {
				contentBuilder.WriteString(fmt.Sprintf("\tfx.Invoke(init%sLifecycle),\n", service.StructName))
			}
		}
	}

	// Add logger providers to the module
	for _, logger := range metadata.Loggers {
		if logger.IsManual {
			// Reference manual module
			if logger.ModuleName != "" {
				contentBuilder.WriteString(fmt.Sprintf("\t%s,\n", logger.ModuleName))
			}
		} else {
			// All loggers use fx.Provide to make them available for dependency injection
			contentBuilder.WriteString(fmt.Sprintf("\tfx.Provide(New%s),\n", logger.StructName))
		}
	}

	// Add interface providers to the module
	for _, iface := range metadata.Interfaces {
		contentBuilder.WriteString(fmt.Sprintf("\tfx.Provide(New%s),\n", iface.Name))
	}

	contentBuilder.WriteString(")\n")

	// Get the generated content
	generatedContent := contentBuilder.String()

	// Use ImportManager to detect required imports
	requiredImports := importManager.GetRequiredImports(generatedContent)

	// Add local package imports from packagePaths
	for packageName, packagePath := range packagePaths {
		// Only add if the package is actually used in the generated content
		if strings.Contains(generatedContent, packageName+".") {
			requiredImports = append(requiredImports, Import{Path: packagePath})
		}
	}

	// Filter out unused imports
	usedImports := importManager.FilterUnusedImports(requiredImports, generatedContent)

	// Generate import block
	importBlock := importManager.GenerateImportBlock(usedImports)

	// Combine everything
	if importBlock != "" {
		moduleBuilder.WriteString(importBlock)
		moduleBuilder.WriteString("\n\n")
	}
	moduleBuilder.WriteString(generatedContent)

	return moduleBuilder.String(), nil
}

// extractDependencyName extracts a variable name from a dependency type
func extractDependencyName(depType string) string {
	// Remove pointer prefix
	name := strings.TrimPrefix(depType, "*")

	// Handle package-qualified types (e.g., "pkg.Type" -> "type")
	if dotIndex := strings.LastIndex(name, "."); dotIndex != -1 {
		name = name[dotIndex+1:]
	}

	// Keep the original case for field names - Go struct fields are exported (PascalCase)
	return name
}

// InterfaceData represents data needed for interface generation
type InterfaceData struct {
	Name       string
	StructName string
	Methods    []MethodData
}

// MethodData represents a method for interface generation
type MethodData struct {
	Name       string
	Parameters []ParameterData
	Returns    []string
}

// ParameterData represents a parameter for interface generation
type ParameterData struct {
	Name string
	Type string
}

// GenerateInterface generates interface code from metadata
func GenerateInterface(iface models.InterfaceMetadata) (string, error) {
	return GenerateInterfaceWithImportManager(iface, nil)
}

// GenerateInterfaceWithImportManager generates interface code with proper import detection
func GenerateInterfaceWithImportManager(iface models.InterfaceMetadata, importManager *ImportManager) (string, error) {
	// Convert methods to template data
	var methods []MethodData
	for _, method := range iface.Methods {
		var params []ParameterData
		for _, param := range method.Parameters {
			params = append(params, ParameterData{
				Name: param.Name,
				Type: param.Type,
			})
		}

		methods = append(methods, MethodData{
			Name:       method.Name,
			Parameters: params,
			Returns:    method.Returns,
		})
	}

	data := InterfaceData{
		Name:       iface.Name,
		StructName: iface.StructName,
		Methods:    methods,
	}

	interfaceCode, err := executeTemplate("interface", InterfaceTemplate, data)
	if err != nil {
		return "", err
	}

	// If ImportManager is provided, we can add import detection logic here
	// For now, just return the interface code as the imports will be handled
	// at the module level by the calling function

	return interfaceCode, nil
}

// LoggerProviderData represents data needed for logger provider generation
type LoggerProviderData struct {
	StructName   string
	Dependencies []DependencyData
	InjectedDeps []DependencyData
	HasStart     bool
	HasStop      bool
	ConfigParam  string // Name of the config parameter
}

// GenerateLoggerProvider generates FX provider code for a logger
func GenerateLoggerProvider(logger models.LoggerMetadata) (string, error) {
	// Convert Dependency models to DependencyData for templates
	var dependencies []DependencyData
	var injectedDeps []DependencyData
	var configParam string

	for _, dep := range logger.Dependencies {
		depData := DependencyData{
			Name:      strings.ToLower(dep.Name[:1]) + dep.Name[1:], // Convert to camelCase for parameter
			FieldName: dep.Name,                                     // Original field name
			Type:      dep.Type,
			IsInit:    dep.IsInit,
		}
		dependencies = append(dependencies, depData)

		// Only add to injected deps if it's not an init dependency
		if !dep.IsInit {
			injectedDeps = append(injectedDeps, depData)

			// Check if this dependency can be used for logger configuration
			if isConfigLikeType(dep.Type) {
				configParam = depData.Name
			}
		}
	}

	// Check if this is a logger that needs immediate initialization
	hasLoggerField := false
	for _, dep := range dependencies {
		if dep.IsInit && isLoggerType(dep.Type) {
			hasLoggerField = true
			break
		}
	}

	if hasLoggerField && configParam != "" {
		// Use logger template for loggers with config and slog field
		if logger.HasLifecycle {
			// Use lifecycle logger template
			data := LoggerProviderData{
				StructName:   logger.StructName,
				Dependencies: dependencies,
				InjectedDeps: injectedDeps,
				HasStart:     logger.HasStart,
				HasStop:      logger.HasStop,
				ConfigParam:  configParam,
			}
			return executeTemplate("logger-provider", LoggerProviderTemplate, data)
		} else {
			// Use simple logger template without lifecycle
			data := LoggerProviderData{
				StructName:   logger.StructName,
				Dependencies: dependencies,
				InjectedDeps: injectedDeps,
				HasStart:     logger.HasStart,
				HasStop:      logger.HasStop,
				ConfigParam:  configParam,
			}
			return executeTemplate("simple-logger-provider", SimpleLoggerProviderTemplate, data)
		}
	} else if logger.HasLifecycle {
		// Use FX lifecycle template for other loggers with lifecycle
		data := CoreServiceProviderData{
			StructName:   logger.StructName,
			Dependencies: dependencies,
			InjectedDeps: injectedDeps,
			HasStart:     logger.HasStart,
			HasStop:      logger.HasStop,
		}
		return executeTemplate("fx-lifecycle-provider", FXLifecycleProviderTemplate, data)
	} else if len(logger.Dependencies) > 0 {
		// Use regular provider template for loggers with dependencies
		data := CoreServiceProviderData{
			StructName:   logger.StructName,
			Dependencies: dependencies,
			InjectedDeps: injectedDeps,
			HasStart:     logger.HasStart,
			HasStop:      logger.HasStop,
		}
		return executeTemplate("provider", ProviderTemplate, data)
	} else {
		// Use FX provider template for loggers with no dependencies
		data := CoreServiceProviderData{
			StructName:   logger.StructName,
			Dependencies: dependencies,
			InjectedDeps: injectedDeps,
			HasStart:     logger.HasStart,
			HasStop:      logger.HasStop,
		}
		return executeTemplate("fx-provider", FXProviderTemplate, data)
	}
}

// GenerateInterfaceProvider generates FX provider code for interface casting
func GenerateInterfaceProvider(iface models.InterfaceMetadata) (string, error) {
	data := InterfaceData{
		Name:       iface.Name,
		StructName: iface.StructName,
	}

	return executeTemplate("interface-provider", InterfaceProviderTemplate, data)
}

// executeTemplate executes a Go template with the given data
func executeTemplate(name, templateStr string, data interface{}) (string, error) {
	// Create template with custom functions
	funcMap := template.FuncMap{
		"generateInitCode": generateInitCode,
	}

	tmpl, err := template.New(name).Funcs(funcMap).Parse(templateStr)
	if err != nil {
		return "", fmt.Errorf("failed to parse template %s: %w", name, err)
	}

	var buf bytes.Buffer
	err = tmpl.Execute(&buf, data)
	if err != nil {
		return "", fmt.Errorf("failed to execute template %s: %w", name, err)
	}

	return buf.String(), nil
}

// ExecuteTemplate executes a Go template with the given data (exported version)
func ExecuteTemplate(name, templateStr string, data interface{}) (string, error) {
	return executeTemplate(name, templateStr, data)
}
