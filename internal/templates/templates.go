package templates

import (
	"bytes"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
	"text/template"

	"github.com/toyz/axon/internal/errors"
	"github.com/toyz/axon/internal/models"
	"github.com/toyz/axon/pkg/axon"
)

// This package contains Go templates for code generation
// Route wrapper generation is handled in response.go

// Template constants are now defined in template_defs.go for better organization

// Route registration template data structures
type RouteRegistrationData struct {
	Controllers    []ControllerTemplateData
	MiddlewareDeps []MiddlewareDependency
}

type ControllerTemplateData struct {
	StructName string
	VarName    string
	Prefix     string
	EchoPrefix string
	Routes     []RouteTemplateData
}

type RouteTemplateData struct {
	HandlerVar               string
	WrapperFunc              string
	ControllerVar            string
	GroupVar                 string
	Method                   string
	Path                     string
	EchoPath                 string
	HandlerName              string
	ControllerName           string
	PackageName              string
	HasMiddleware            bool
	MiddlewareList           string
	MiddlewaresArray         string
	MiddlewareInstancesArray string
	ParameterInstancesArray  string
}

type MiddlewareDependency struct {
	Name        string
	VarName     string
	PackageName string
}

type MiddlewareInstanceData struct {
	Name    string
	VarName string
}

// GenerateRouteRegistrationFunction generates the RegisterRoutes function using templates
func GenerateRouteRegistrationFunction(data RouteRegistrationData) (string, error) {
	// Parse the main template
	tmpl, err := template.New("routeRegistration").Parse(DefaultTemplateRegistry.MustGet("route-registration-function"))
	if err != nil {
		return "", errors.WrapParseError("route registration template", err)
	}

	// Add the route registration sub-template
	_, err = tmpl.New("RouteRegistration").Parse(DefaultTemplateRegistry.MustGet("route-registration"))
	if err != nil {
		return "", errors.WrapParseError("route registration sub-template", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", errors.WrapWithOperation("execute", "route registration template", err)
	}

	return buf.String(), nil
}

// Helper functions for building template data
func BuildMiddlewareInstancesArray(middlewares []string) string {
	return DefaultTemplateUtils.BuildMiddlewareInstancesArray(middlewares)
}

func BuildMiddlewaresArray(middlewares []string) string {
	return DefaultTemplateUtils.BuildMiddlewaresArray(middlewares)
}

func BuildParameterInstancesArray(paramTypes map[string]string) string {
	return DefaultTemplateUtils.BuildParameterInstancesArray(paramTypes)
}

func BuildMiddlewareList(middlewares []string) string {
	return DefaultTemplateUtils.BuildMiddlewareList(middlewares)
}

// ExtractParameterTypes extracts parameter types from a route path
func ExtractParameterTypes(path string) map[string]string {
	paramTypes := make(map[string]string)

	// Find all parameters in the format {name:type}
	re := regexp.MustCompile(`\{([^:}]+):([^}]+)\}`)
	matches := re.FindAllStringSubmatch(path, -1)

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
func GenerateParameterBindingCode(parameters []models.Parameter, parserRegistry axon.ParserRegistryInterface) (string, error) {
	var bindingCode strings.Builder

	for _, param := range parameters {
		// Special handling for QueryMap type
		if param.Type == "axon.QueryMap" {
			bindingCode.WriteString(fmt.Sprintf("\t%s := axon.NewQueryMap(c)\n", param.Name))
			continue
		}

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

// BaseProviderData represents common data needed for provider generation
type BaseProviderData struct {
	StructName   string
	Dependencies []DependencyData // All dependencies (for struct initialization)
	InjectedDeps []DependencyData // Only injected dependencies (for function parameters)
	HasStart     bool
	HasStop      bool
}

// CoreServiceProviderData represents data needed for core service provider generation
type CoreServiceProviderData struct {
	BaseProviderData
	StartMode string // lifecycle start mode: "Same" (default) or "Background"
}

// DependencyData represents a dependency for template generation
type DependencyData struct {
	Name      string // parameter name (camelCase)
	FieldName string // struct field name (original case)
	Type      string // type of the dependency
	IsInit    bool   // whether this should be initialized (not injected)
}

// toCamelCase converts PascalCase to camelCase
func toCamelCase(s string) string {
	if s == "" {
		return s
	}
	return strings.ToLower(s[:1]) + s[1:]
}


// generateImports creates an import block with the specified packages
func generateImports(packages ...string) string {
	if len(packages) == 0 {
		return ""
	}
	
	var builder strings.Builder
	builder.WriteString("import (\n")
	
	for _, pkg := range packages {
		builder.WriteString(fmt.Sprintf("\t\"%s\"\n", pkg))
	}
	
	builder.WriteString(")\n\n")
	return builder.String()
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

// Note: Removed resolveImportPath and buildModuleImportPath functions
// These were making assumptions about project structure.
// Templates now receive actual package paths from the generator.



// extractPackageFromType is now available as utils.ExtractPackageFromType

// GenerateCoreServiceProvider generates FX provider code for a core service
func GenerateCoreServiceProvider(service models.CoreServiceMetadata) (string, error) {
	if service.IsManual {
		// Manual services don't need generated providers
		return "", nil
	}

	// If a custom constructor is provided, use it instead of generating one
	if service.Constructor != "" {
		// For custom constructors, we just need to provide the existing function
		// The user has already defined the constructor function
		return "", nil // No generated provider needed - use the custom constructor directly
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
		BaseProviderData: BaseProviderData{
			StructName:   service.StructName,
			Dependencies: dependencies,
			InjectedDeps: injectedDeps,
			HasStart:     service.HasStart,
			HasStop:      service.HasStop,
		},
	}

	// Handle different lifecycle modes
	if service.Mode == "Transient" {
		// For transient services, generate a factory function
		return executeRegistryTemplate("transient-provider", data)
	} else {
		// Default Singleton mode
		if service.HasLifecycle && service.StartMode != "" {
			// For services with -Init flag (StartMode is set), generate simple provider only
			// The invoke function will be generated separately
			return executeRegistryTemplate("provider", data)
		} else if service.HasLifecycle {
			// For old-style lifecycle services (no -Init flag), generate embedded lifecycle hooks
			return executeRegistryTemplate("lifecycle-provider", data)
		} else if len(service.Dependencies) > 0 {
			// Use regular provider template for services with dependencies
			return executeRegistryTemplate("provider", data)
		} else {
			// Use FX provider template for services with no dependencies
			return executeRegistryTemplate("simple-provider", data)
		}
	}
}


// GenerateInitInvokeFunction generates an invoke function for lifecycle management
func GenerateInitInvokeFunction(service models.CoreServiceMetadata) (string, error) {
	if !service.HasLifecycle {
		return "", nil
	}

	data := CoreServiceProviderData{
		BaseProviderData: BaseProviderData{
			StructName: service.StructName,
			HasStart:   service.HasStart,
			HasStop:    service.HasStop,
		},
		StartMode: service.StartMode,
	}

	return executeRegistryTemplate("init-invoke", data)
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

	// Generate minimal imports - goimports will handle the rest
	moduleBuilder.WriteString(generateImports("context", "go.uber.org/fx"))
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
			return "", errors.WrapGenerateError("interface", iface.Name, err)
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
			return "", errors.WrapGenerateError("provider", "service "+service.Name, err)
		}

		if provider != "" {
			contentBuilder.WriteString(provider)
			contentBuilder.WriteString("\n\n")
		}

		// Generate invoke function for services with -Init flag
		if service.HasLifecycle {
			invokeFunc, err := GenerateInitInvokeFunction(service)
			if err != nil {
				return "", errors.WrapGenerateError("invoke function", "service "+service.Name, err)
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
			return "", errors.WrapGenerateError("provider", fmt.Sprintf("logger %s", logger.Name), err)
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
			return "", errors.WrapGenerateError("interface provider", iface.Name, err)
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
			if service.Constructor != "" {
				// Use custom constructor
				contentBuilder.WriteString(fmt.Sprintf("\tfx.Provide(%s),\n", service.Constructor))
			} else {
				// Use generated constructor
				contentBuilder.WriteString(fmt.Sprintf("\tfx.Provide(New%s),\n", service.StructName))
			}

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

	// Simple approach - just append the content, goimports will handle imports
	moduleBuilder.WriteString(generatedContent)

	return moduleBuilder.String(), nil
}

// extractDependencyName extracts a variable name from a dependency type
// extractDependencyName is now available as utils.ExtractDependencyName

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
	// Simple interface generation - goimports will handle imports
	var builder strings.Builder

	builder.WriteString("// Code generated by Axon framework. DO NOT EDIT.\n")
	builder.WriteString("// This file was automatically generated and should not be modified manually.\n\n")
	// Extract package name from package path
	packageName := filepath.Base(iface.PackagePath)
	builder.WriteString(fmt.Sprintf("package %s\n\n", packageName))

	// Generate minimal imports
	builder.WriteString(generateImports("go.uber.org/fx"))

	return generateInterfaceContent(iface, &builder)
}

// generateInterfaceContent generates the interface content
func generateInterfaceContent(iface models.InterfaceMetadata, builder *strings.Builder) (string, error) {
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

	interfaceCode, err := executeRegistryTemplate("interface", data)
	if err != nil {
		return "", err
	}

	return interfaceCode, nil
}

// LoggerProviderData represents data needed for logger provider generation
type LoggerProviderData struct {
	BaseProviderData
	ConfigParam string // Name of the config parameter
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
			if DefaultTemplateUtils.IsConfigLikeType(dep.Type) {
				configParam = depData.Name
			}
		}
	}

	// Check if this is a logger that needs immediate initialization
	hasLoggerField := false
	for _, dep := range dependencies {
		if dep.IsInit && DefaultTemplateUtils.IsLoggerType(dep.Type) {
			hasLoggerField = true
			break
		}
	}

	if hasLoggerField && configParam != "" {
		// Use logger template for loggers with config and slog field
		if logger.HasLifecycle {
			// Use lifecycle logger template
			data := LoggerProviderData{
				BaseProviderData: BaseProviderData{
					StructName:   logger.StructName,
					Dependencies: dependencies,
					InjectedDeps: injectedDeps,
					HasStart:     logger.HasStart,
					HasStop:      logger.HasStop,
				},
				ConfigParam: configParam,
			}
			return executeRegistryTemplate("logger-provider", data)
		} else {
			// Use simple logger template without lifecycle
			data := LoggerProviderData{
				BaseProviderData: BaseProviderData{
					StructName:   logger.StructName,
					Dependencies: dependencies,
					InjectedDeps: injectedDeps,
					HasStart:     logger.HasStart,
					HasStop:      logger.HasStop,
				},
				ConfigParam: configParam,
			}
			return executeRegistryTemplate("simple-logger-provider", data)
		}
	} else if logger.HasLifecycle {
		// Use FX lifecycle template for other loggers with lifecycle
		data := CoreServiceProviderData{
			BaseProviderData: BaseProviderData{
				StructName:   logger.StructName,
				Dependencies: dependencies,
				InjectedDeps: injectedDeps,
				HasStart:     logger.HasStart,
				HasStop:      logger.HasStop,
			},
		}
		return executeRegistryTemplate("lifecycle-provider", data)
	} else if len(logger.Dependencies) > 0 {
		// Use regular provider template for loggers with dependencies
		data := CoreServiceProviderData{
			BaseProviderData: BaseProviderData{
				StructName:   logger.StructName,
				Dependencies: dependencies,
				InjectedDeps: injectedDeps,
				HasStart:     logger.HasStart,
				HasStop:      logger.HasStop,
			},
		}
		return executeRegistryTemplate("provider", data)
	} else {
		// Use FX provider template for loggers with no dependencies
		data := CoreServiceProviderData{
			BaseProviderData: BaseProviderData{
				StructName:   logger.StructName,
				Dependencies: dependencies,
				InjectedDeps: injectedDeps,
				HasStart:     logger.HasStart,
				HasStop:      logger.HasStop,
			},
		}
		return executeRegistryTemplate("simple-provider", data)
	}
}

// GenerateInterfaceProvider generates FX provider code for interface casting
func GenerateInterfaceProvider(iface models.InterfaceMetadata) (string, error) {
	data := InterfaceData{
		Name:       iface.Name,
		StructName: iface.StructName,
	}

	return executeRegistryTemplate("interface-provider", data)
}

// executeTemplate executes a Go template with the given data
func executeTemplate(name, templateStr string, data interface{}) (string, error) {
	// Create template with custom functions
	funcMap := template.FuncMap{
		"generateInitCode": generateInitCode,
		"toCamelCase":      toCamelCase,
	}

	tmpl, err := template.New(name).Funcs(funcMap).Parse(templateStr)
	if err != nil {
		return "", errors.WrapParseError("template "+name, err)
	}

	var buf bytes.Buffer
	err = tmpl.Execute(&buf, data)
	if err != nil {
		return "", errors.WrapWithOperation("execute", fmt.Sprintf("template %s", name), err)
	}

	return buf.String(), nil
}

// ExecuteTemplate executes a Go template with the given data (exported version)
func ExecuteTemplate(name, templateStr string, data interface{}) (string, error) {
	return executeTemplate(name, templateStr, data)
}

// executeRegistryTemplate executes a template from the registry
func executeRegistryTemplate(name string, data interface{}) (string, error) {
	templateStr := DefaultTemplateRegistry.MustGet(name)
	return executeTemplate(name, templateStr, data)
}

// GenerateMiddlewareProvider generates FX provider function for middleware
func GenerateMiddlewareProvider(middleware models.MiddlewareMetadata) (string, error) {
	data := struct {
		StructName   string
		Dependencies []models.Dependency
		InjectedDeps []models.Dependency
	}{
		StructName:   middleware.StructName,
		Dependencies: middleware.Dependencies,
		InjectedDeps: filterInjectedDependencies(middleware.Dependencies),
	}

	return executeRegistryTemplate("middleware-provider", data)
}

// filterInjectedDependencies filters out dependencies that are initialized (IsInit=true)
func filterInjectedDependencies(dependencies []models.Dependency) []models.Dependency {
	var injected []models.Dependency
	for _, dep := range dependencies {
		if !dep.IsInit {
			injected = append(injected, dep)
		}
	}
	return injected
}

// GenerateGlobalMiddlewareRegistration generates function to register global middleware with Echo
func GenerateGlobalMiddlewareRegistration(middlewares []models.MiddlewareMetadata) (string, error) {
	// Filter only global middleware and sort by priority
	var globalMiddlewares []models.MiddlewareMetadata
	for _, mw := range middlewares {
		if mw.IsGlobal {
			globalMiddlewares = append(globalMiddlewares, mw)
		}
	}

	// Sort by priority (lower number = higher priority)
	for i := 0; i < len(globalMiddlewares)-1; i++ {
		for j := i + 1; j < len(globalMiddlewares); j++ {
			if globalMiddlewares[i].Priority > globalMiddlewares[j].Priority {
				globalMiddlewares[i], globalMiddlewares[j] = globalMiddlewares[j], globalMiddlewares[i]
			}
		}
	}

	data := struct {
		GlobalMiddlewares []models.MiddlewareMetadata
	}{
		GlobalMiddlewares: globalMiddlewares,
	}

	return executeRegistryTemplate("global-middleware-registration", data)
}

// GenerateMiddlewareRegistry generates function to register all middleware with axon registry
func GenerateMiddlewareRegistry(middlewares []models.MiddlewareMetadata) (string, error) {
	data := struct {
		Middlewares []models.MiddlewareMetadata
	}{
		Middlewares: middlewares,
	}

	return executeRegistryTemplate("middleware-registry", data)
}

// GenerateMiddlewareModule generates a complete middleware module using ImportManager
func GenerateMiddlewareModule(metadata *models.PackageMetadata) (string, error) {
	var contentBuilder strings.Builder

	// Generate all content first (without imports)

	// Generate middleware providers
	for _, middleware := range metadata.Middlewares {
		providerCode, err := GenerateMiddlewareProvider(middleware)
		if err != nil {
			return "", errors.WrapGenerateError("provider", fmt.Sprintf("middleware %s", middleware.Name), err)
		}
		contentBuilder.WriteString(providerCode)
		contentBuilder.WriteString("\n")
	}

	// Generate middleware registration function
	registrationCode, err := GenerateMiddlewareRegistry(metadata.Middlewares)
	if err != nil {
		return "", errors.WrapGenerateError("middleware", "registration", err)
	}
	contentBuilder.WriteString(registrationCode)
	contentBuilder.WriteString("\n")

	// Generate global middleware registration if there are global middlewares
	hasGlobalMiddleware := false
	for _, mw := range metadata.Middlewares {
		if mw.IsGlobal {
			hasGlobalMiddleware = true
			break
		}
	}

	if hasGlobalMiddleware {
		globalRegistrationCode, err := GenerateGlobalMiddlewareRegistration(metadata.Middlewares)
		if err != nil {
			return "", errors.WrapGenerateError("global middleware", "registration", err)
		}
		contentBuilder.WriteString(globalRegistrationCode)
		contentBuilder.WriteString("\n")
	}

	// Generate module variable
	contentBuilder.WriteString("// AutogenModule provides all middleware in this package\n")
	contentBuilder.WriteString(fmt.Sprintf("var AutogenModule = fx.Module(\"%s\",\n", metadata.PackageName))

	// Add middleware providers
	for _, middleware := range metadata.Middlewares {
		contentBuilder.WriteString(fmt.Sprintf("\tfx.Provide(New%s),\n", middleware.StructName))
	}

	// Add middleware registration as an invoke
	contentBuilder.WriteString("\tfx.Invoke(RegisterMiddlewares),\n")

	// Add global middleware registration if there are global middlewares
	if hasGlobalMiddleware {
		contentBuilder.WriteString("\tfx.Invoke(RegisterGlobalMiddleware),\n")
	}

	contentBuilder.WriteString(")\n")

	// Get the generated content
	generatedContent := contentBuilder.String()

	// Combine everything
	var moduleBuilder strings.Builder
	moduleBuilder.WriteString("// Code generated by Axon framework. DO NOT EDIT.\n")
	moduleBuilder.WriteString("// This file was automatically generated and should not be modified manually.\n\n")
	moduleBuilder.WriteString(fmt.Sprintf("package %s\n\n", metadata.PackageName))

	// Generate minimal imports - goimports will handle the rest
	moduleBuilder.WriteString(generateImports("go.uber.org/fx"))

	moduleBuilder.WriteString(generatedContent)

	return moduleBuilder.String(), nil
}

// resolveDependencyImportPath resolves the import path for a dependency package
func resolveDependencyImportPath(packageName string, metadata *models.PackageMetadata) string {
	// Use the module path from metadata to build the correct import path
	if metadata.ModulePath != "" {
		// For internal packages, use the module path + internal/packageName pattern
		return metadata.ModulePath + "/internal/" + packageName
	}

	// Fallback: try to extract from package path
	// packagePath is like "/home/user/project/internal/middleware"
	// We want "github.com/user/project/internal/packageName"
	parts := strings.Split(metadata.PackagePath, "/")
	for i, part := range parts {
		if part == "internal" && i > 0 {
			// Find the module root - look for github.com pattern
			for j := i - 1; j >= 0; j-- {
				if strings.Contains(parts[j], ".") { // Likely a domain
					moduleRoot := strings.Join(parts[j:i], "/")
					return moduleRoot + "/internal/" + packageName
				}
			}
		}
	}

	// Last resort: assume it's a sibling package
	return strings.Replace(metadata.PackagePath, "/middleware", "/"+packageName, 1)
}

