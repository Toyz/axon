package templates

import (
	"bytes"
	"fmt"
	"regexp"
	"strings"
	"text/template"

	"github.com/toyz/axon/internal/models"
)

// This package contains Go templates for code generation
// Route wrapper generation is handled in response.go

const (
	// ProviderTemplate is the template for generating FX provider functions
	ProviderTemplate = `func New{{.StructName}}({{range $i, $dep := .InjectedDeps}}{{if $i}}, {{end}}{{$dep.Name}} {{$dep.Type}}{{end}}) *{{.StructName}} {
	return &{{.StructName}}{
{{range .Dependencies}}{{if .IsInit}}		{{.FieldName}}: {{generateInitCode .Type}},
{{else}}		{{.FieldName}}: {{.Name}},
{{end}}{{end}}{{if not .Dependencies}}
{{end}}	}
}`

	// FXProviderTemplate is the template for generating FX provider functions with fx.In
	FXProviderTemplate = `func New{{.StructName}}() *{{.StructName}} {
	return &{{.StructName}}{}
}`

	// FXLifecycleProviderTemplate is the template for generating FX provider functions with fx.In and lifecycle
	FXLifecycleProviderTemplate = `func New{{.StructName}}(lc fx.Lifecycle{{range .Dependencies}}{{if not .IsInit}}, {{.Name}} {{.Type}}{{end}}{{end}}) *{{.StructName}} {
	service := &{{.StructName}}{
{{range .Dependencies}}{{if .IsInit}}		{{.FieldName}}: {{generateInitCode .Type}},
{{else}}		{{.FieldName}}: {{.Name}},
{{end}}{{end}}	}
	
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			return service.Start(ctx)
		},{{if .HasStop}}
		OnStop: func(ctx context.Context) error {
			return service.Stop(ctx)
		},{{end}}
	})
	
	return service
}`

	// LoggerProviderTemplate is the template for generating FX provider functions for loggers with immediate initialization
	LoggerProviderTemplate = `func New{{.StructName}}(lc fx.Lifecycle{{range .Dependencies}}{{if not .IsInit}}, {{.Name}} {{.Type}}{{end}}{{end}}) *{{.StructName}} {
	// Initialize logger immediately for fx.WithLogger to work
	var handler slog.Handler
	if {{.ConfigParam}}.LogLevel == "debug" {
		handler = slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelDebug,
		})
	} else {
		handler = slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelInfo,
		})
	}
	
	service := &{{.StructName}}{
{{range .Dependencies}}{{if .IsInit}}		{{.FieldName}}: slog.New(handler),
{{else}}		{{.FieldName}}: {{.Name}},
{{end}}{{end}}	}
	
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			return service.Start(ctx)
		},{{if .HasStop}}
		OnStop: func(ctx context.Context) error {
			return service.Stop(ctx)
		},{{end}}
	})
	
	return service
}`

	// SimpleLoggerProviderTemplate is for loggers without lifecycle hooks
	SimpleLoggerProviderTemplate = `func New{{.StructName}}({{range $i, $dep := .InjectedDeps}}{{if $i}}, {{end}}{{$dep.Name}} {{$dep.Type}}{{end}}) *{{.StructName}} {
	// Initialize logger immediately
	var handler slog.Handler
	if {{.ConfigParam}}.LogLevel == "debug" {
		handler = slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelDebug,
		})
	} else {
		handler = slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelInfo,
		})
	}
	
	return &{{.StructName}}{
{{range .Dependencies}}{{if .IsInit}}		{{.FieldName}}: slog.New(handler),
{{else}}		{{.FieldName}}: {{.Name}},
{{end}}{{end}}	}
}`

	// LifecycleProviderTemplate is the template for generating FX provider functions with lifecycle management
	LifecycleProviderTemplate = `func New{{.StructName}}(lc fx.Lifecycle{{range .Dependencies}}{{if not .IsInit}}, {{.Name}} {{.Type}}{{end}}{{end}}) *{{.StructName}} {
	service := &{{.StructName}}{
{{range .Dependencies}}{{if .IsInit}}		{{.FieldName}}: {{generateInitCode .Type}},
{{else}}		{{.FieldName}}: {{.Name}},
{{end}}{{end}}	}
	
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			return service.Start(ctx)
		},{{if .HasStop}}
		OnStop: func(ctx context.Context) error {
			return service.Stop(ctx)
		},{{end}}
	})
	
	return service
}`

	// InterfaceTemplate is the template for generating interfaces from structs
	InterfaceTemplate = `// {{.Name}} is the interface for {{.StructName}}
type {{.Name}} interface {
{{range .Methods}}	{{.Name}}({{range $i, $param := .Parameters}}{{if $i}}, {{end}}{{if $param.Name}}{{$param.Name}} {{end}}{{$param.Type}}{{end}}){{if .Returns}} ({{range $i, $ret := .Returns}}{{if $i}}, {{end}}{{$ret}}{{end}}){{end}}
{{end}}}`

	// InterfaceProviderTemplate is the template for generating FX provider that casts struct to interface
	InterfaceProviderTemplate = `func New{{.Name}}(impl *{{.StructName}}) {{.Name}} {
	return impl
}`
)



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



// ParameterBindingData represents data needed for parameter binding template
type ParameterBindingData struct {
	Name           string
	Type           string
	Source         string
	ConversionFunc string
}

// GenerateParameterBinding generates parameter binding code for a route parameter
func GenerateParameterBinding(param models.Parameter) (ParameterBindingData, error) {
	data := ParameterBindingData{
		Name:   param.Name,
		Type:   param.Type,
		Source: getParameterSourceString(param.Source),
	}

	// Generate appropriate conversion function based on parameter type
	switch param.Type {
	case "int":
		data.ConversionFunc = "strconv.Atoi"
	case "string":
		data.ConversionFunc = "func(s string) (string, error) { return s, nil }"
	default:
		return ParameterBindingData{}, fmt.Errorf("unsupported parameter type: %s", param.Type)
	}

	return data, nil
}

// GenerateParameterBindingCode generates the complete parameter binding code for a list of parameters
func GenerateParameterBindingCode(parameters []models.Parameter) (string, error) {
	var bindingCode strings.Builder

	for _, param := range parameters {
		switch param.Source {
		case models.ParameterSourcePath:
			// Generate parameter binding code for path parameters
			switch param.Type {
			case "int":
				bindingCode.WriteString(fmt.Sprintf(`		%s, err := strconv.Atoi(c.Param("%s"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Invalid %s: must be an integer")
		}
`, param.Name, param.Name, param.Name))
			case "string":
				bindingCode.WriteString(fmt.Sprintf(`		%s := c.Param("%s")
`, param.Name, param.Name))
			default:
				return "", fmt.Errorf("unsupported parameter type: %s", param.Type)
			}
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
	StructName    string
	Dependencies  []DependencyData // All dependencies (for struct initialization)
	InjectedDeps  []DependencyData // Only injected dependencies (for function parameters)
	HasStart      bool
	HasStop       bool
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

// isStandardLibraryPackage checks if a package name is from the Go standard library
func isStandardLibraryPackage(packageName string) bool {
	// List of common standard library packages that might appear in type annotations
	standardLibPackages := map[string]bool{
		"slog":    true, // log/slog
		"log":     true, // log
		"fmt":     true, // fmt
		"time":    true, // time
		"context": true, // context
		"http":    true, // net/http
		"url":     true, // net/url
		"json":    true, // encoding/json
		"sql":     true, // database/sql
		"os":      true, // os
		"io":      true, // io
		"strings": true, // strings
		"strconv": true, // strconv
		"sync":    true, // sync
		"errors":  true, // errors
	}
	
	return standardLibPackages[packageName]
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
			Name:      strings.ToLower(dep.Name[:1]) + dep.Name[1:], // Convert to camelCase for parameter
			FieldName: dep.Name,                                     // Original field name
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

	if service.HasLifecycle {
		// Use FX lifecycle template for services with lifecycle
		return executeTemplate("fx-lifecycle-provider", FXLifecycleProviderTemplate, data)
	} else if len(service.Dependencies) > 0 {
		// Use regular provider template for services with dependencies
		return executeTemplate("provider", ProviderTemplate, data)
	} else {
		// Use FX provider template for services with no dependencies
		return executeTemplate("fx-provider", FXProviderTemplate, data)
	}
}

// GenerateCoreServiceModule generates the complete FX module for core services in a package
func GenerateCoreServiceModule(metadata *models.PackageMetadata) (string, error) {
	return GenerateCoreServiceModuleWithModule(metadata, "")
}

// GenerateCoreServiceModuleWithModule generates the complete FX module for core services in a package with module name
func GenerateCoreServiceModuleWithModule(metadata *models.PackageMetadata, moduleName string) (string, error) {
	var moduleBuilder strings.Builder
	
	// Generate package declaration with DO NOT EDIT header
	moduleBuilder.WriteString("// Code generated by Axon framework. DO NOT EDIT.\n")
	moduleBuilder.WriteString("// This file was automatically generated and should not be modified manually.\n\n")
	moduleBuilder.WriteString(fmt.Sprintf("package %s\n\n", metadata.PackageName))
	
	// Analyze what imports are needed
	needsContext := false
	needsModels := false
	dependencyImports := make(map[string]bool)
	
	// Check if any core services have lifecycle (need context)
	for _, service := range metadata.CoreServices {
		if service.HasLifecycle {
			needsContext = true
		}
		
		// Analyze dependencies for imports
		for _, dep := range service.Dependencies {
			if packagePath := extractPackageFromType(dep.Type); packagePath != "" {
				dependencyImports[packagePath] = true
			}
		}
	}
	
	// Check if any loggers have lifecycle (need context)
	for _, logger := range metadata.Loggers {
		if logger.HasLifecycle {
			needsContext = true
		}
		
		// Analyze dependencies for imports
		for _, dep := range logger.Dependencies {
			if packagePath := extractPackageFromType(dep.Type); packagePath != "" {
				dependencyImports[packagePath] = true
			}
		}
	}
	
	// Check if any interfaces reference model types
	for _, iface := range metadata.Interfaces {
		for _, method := range iface.Methods {
			for _, param := range method.Parameters {
				if strings.Contains(param.Type, "models.") {
					needsModels = true
					break
				}
			}
			if needsModels {
				break
			}
			for _, ret := range method.Returns {
				if strings.Contains(ret, "models.") {
					needsModels = true
					break
				}
			}
			if needsModels {
				break
			}
		}
		if needsModels {
			break
		}
	}
	
	// Generate imports
	moduleBuilder.WriteString("import (\n")
	if needsContext {
		moduleBuilder.WriteString("\t\"context\"\n")
	}
	moduleBuilder.WriteString("\t\"go.uber.org/fx\"\n")
	
	// Add fxevent import if there are loggers
	if len(metadata.Loggers) > 0 {
		moduleBuilder.WriteString("\t\"go.uber.org/fx/fxevent\"\n")
		moduleBuilder.WriteString("\t\"log/slog\"\n")
		moduleBuilder.WriteString("\t\"os\"\n")
	}
	
	// Add dependency imports
	for packageName := range dependencyImports {
		// Handle standard library packages
		if isStandardLibraryPackage(packageName) {
			// Skip standard library packages - they don't need explicit imports in this context
			// since they should be imported in the original source files
			continue
		}
		
		if moduleName != "" {
			moduleBuilder.WriteString(fmt.Sprintf("\t\"%s/internal/%s\"\n", moduleName, packageName))
		} else {
			moduleBuilder.WriteString(fmt.Sprintf("\t\"../%s\"\n", packageName))
		}
	}
	
	if needsModels {
		if moduleName != "" {
			moduleBuilder.WriteString(fmt.Sprintf("\t\"%s/internal/models\"\n", moduleName))
		} else {
			moduleBuilder.WriteString("\t\"../models\"\n")
		}
	}
	moduleBuilder.WriteString(")\n\n")
	
	// Generate fxLogger adapter if there are loggers
	if len(metadata.Loggers) > 0 {
		firstLogger := metadata.Loggers[0]
		moduleBuilder.WriteString(fmt.Sprintf("// fxLogger adapts %s to fxevent.Logger\n", firstLogger.StructName))
		moduleBuilder.WriteString(fmt.Sprintf("type fxLogger struct {\n\tlogger *%s\n}\n\n", firstLogger.StructName))
		
		// Implement fxevent.Logger interface
		moduleBuilder.WriteString("func (l *fxLogger) LogEvent(event fxevent.Event) {\n")
		moduleBuilder.WriteString("\tswitch e := event.(type) {\n")
		moduleBuilder.WriteString("\tcase *fxevent.OnStartExecuting:\n")
		moduleBuilder.WriteString("\t\tl.logger.Info(\"OnStart hook executing\", \"callee\", e.FunctionName, \"caller\", e.CallerName)\n")
		moduleBuilder.WriteString("\tcase *fxevent.OnStartExecuted:\n")
		moduleBuilder.WriteString("\t\tif e.Err != nil {\n")
		moduleBuilder.WriteString("\t\t\tl.logger.Error(\"OnStart hook failed\", \"callee\", e.FunctionName, \"caller\", e.CallerName, \"error\", e.Err)\n")
		moduleBuilder.WriteString("\t\t} else {\n")
		moduleBuilder.WriteString("\t\t\tl.logger.Info(\"OnStart hook executed\", \"callee\", e.FunctionName, \"caller\", e.CallerName, \"runtime\", e.Runtime)\n")
		moduleBuilder.WriteString("\t\t}\n")
		moduleBuilder.WriteString("\tcase *fxevent.OnStopExecuting:\n")
		moduleBuilder.WriteString("\t\tl.logger.Info(\"OnStop hook executing\", \"callee\", e.FunctionName, \"caller\", e.CallerName)\n")
		moduleBuilder.WriteString("\tcase *fxevent.OnStopExecuted:\n")
		moduleBuilder.WriteString("\t\tif e.Err != nil {\n")
		moduleBuilder.WriteString("\t\t\tl.logger.Error(\"OnStop hook failed\", \"callee\", e.FunctionName, \"caller\", e.CallerName, \"error\", e.Err)\n")
		moduleBuilder.WriteString("\t\t} else {\n")
		moduleBuilder.WriteString("\t\t\tl.logger.Info(\"OnStop hook executed\", \"callee\", e.FunctionName, \"caller\", e.CallerName, \"runtime\", e.Runtime)\n")
		moduleBuilder.WriteString("\t\t}\n")
		moduleBuilder.WriteString("\tcase *fxevent.Supplied:\n")
		moduleBuilder.WriteString("\t\tl.logger.Debug(\"supplied\", \"type\", e.TypeName, \"module\", e.ModuleName)\n")
		moduleBuilder.WriteString("\tcase *fxevent.Provided:\n")
		moduleBuilder.WriteString("\t\tl.logger.Debug(\"provided\", \"constructor\", e.ConstructorName, \"module\", e.ModuleName)\n")
		moduleBuilder.WriteString("\tcase *fxevent.Invoking:\n")
		moduleBuilder.WriteString("\t\tl.logger.Debug(\"invoking\", \"function\", e.FunctionName, \"module\", e.ModuleName)\n")
		moduleBuilder.WriteString("\tcase *fxevent.Invoked:\n")
		moduleBuilder.WriteString("\t\tif e.Err != nil {\n")
		moduleBuilder.WriteString("\t\t\tl.logger.Error(\"invoke failed\", \"error\", e.Err, \"stack\", e.Trace, \"function\", e.FunctionName, \"module\", e.ModuleName)\n")
		moduleBuilder.WriteString("\t\t} else {\n")
		moduleBuilder.WriteString("\t\t\tl.logger.Debug(\"invoked\", \"function\", e.FunctionName, \"module\", e.ModuleName)\n")
		moduleBuilder.WriteString("\t\t}\n")
		moduleBuilder.WriteString("\tcase *fxevent.Stopping:\n")
		moduleBuilder.WriteString("\t\tl.logger.Info(\"received signal\", \"signal\", e.Signal)\n")
		moduleBuilder.WriteString("\tcase *fxevent.Stopped:\n")
		moduleBuilder.WriteString("\t\tif e.Err != nil {\n")
		moduleBuilder.WriteString("\t\t\tl.logger.Error(\"stop failed\", \"error\", e.Err)\n")
		moduleBuilder.WriteString("\t\t} else {\n")
		moduleBuilder.WriteString("\t\t\tl.logger.Info(\"stopped\")\n")
		moduleBuilder.WriteString("\t\t}\n")
		moduleBuilder.WriteString("\tcase *fxevent.RollingBack:\n")
		moduleBuilder.WriteString("\t\tl.logger.Error(\"start failed, rolling back\", \"error\", e.StartErr)\n")
		moduleBuilder.WriteString("\tcase *fxevent.RolledBack:\n")
		moduleBuilder.WriteString("\t\tif e.Err != nil {\n")
		moduleBuilder.WriteString("\t\t\tl.logger.Error(\"rollback failed\", \"error\", e.Err)\n")
		moduleBuilder.WriteString("\t\t} else {\n")
		moduleBuilder.WriteString("\t\t\tl.logger.Info(\"rolled back\")\n")
		moduleBuilder.WriteString("\t\t}\n")
		moduleBuilder.WriteString("\tcase *fxevent.Started:\n")
		moduleBuilder.WriteString("\t\tif e.Err != nil {\n")
		moduleBuilder.WriteString("\t\t\tl.logger.Error(\"start failed\", \"error\", e.Err)\n")
		moduleBuilder.WriteString("\t\t} else {\n")
		moduleBuilder.WriteString("\t\t\tl.logger.Info(\"started\")\n")
		moduleBuilder.WriteString("\t\t}\n")
		moduleBuilder.WriteString("\tcase *fxevent.LoggerInitialized:\n")
		moduleBuilder.WriteString("\t\tif e.Err != nil {\n")
		moduleBuilder.WriteString("\t\t\tl.logger.Error(\"custom logger initialization failed\", \"error\", e.Err)\n")
		moduleBuilder.WriteString("\t\t} else {\n")
		moduleBuilder.WriteString("\t\t\tl.logger.Debug(\"initialized custom fxevent.Logger\", \"function\", e.ConstructorName)\n")
		moduleBuilder.WriteString("\t\t}\n")
		moduleBuilder.WriteString("\t}\n")
		moduleBuilder.WriteString("}\n\n")
	}
	
	// Generate interfaces
	for _, iface := range metadata.Interfaces {
		interfaceCode, err := GenerateInterface(iface)
		if err != nil {
			return "", fmt.Errorf("failed to generate interface %s: %w", iface.Name, err)
		}
		
		moduleBuilder.WriteString(interfaceCode)
		moduleBuilder.WriteString("\n\n")
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
			moduleBuilder.WriteString(provider)
			moduleBuilder.WriteString("\n\n")
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
			moduleBuilder.WriteString(provider)
			moduleBuilder.WriteString("\n\n")
		}
	}
	
	// Generate interface providers
	for _, iface := range metadata.Interfaces {
		providerCode, err := GenerateInterfaceProvider(iface)
		if err != nil {
			return "", fmt.Errorf("failed to generate interface provider %s: %w", iface.Name, err)
		}
		
		moduleBuilder.WriteString(providerCode)
		moduleBuilder.WriteString("\n\n")
	}
	
	// Generate module variable
	moduleBuilder.WriteString("// AutogenModule provides all core services in this package\n")
	moduleBuilder.WriteString(fmt.Sprintf("var AutogenModule = fx.Module(\"%s\",\n", metadata.PackageName))
	
	// Add fx.WithLogger if there are loggers
	if len(metadata.Loggers) > 0 {
		// Use the first logger as the FX logger
		firstLogger := metadata.Loggers[0]
		moduleBuilder.WriteString(fmt.Sprintf("\tfx.WithLogger(func(logger *%s) fxevent.Logger {\n", firstLogger.StructName))
		moduleBuilder.WriteString("\t\treturn &fxLogger{logger: logger}\n")
		moduleBuilder.WriteString("\t}),\n")
	}
	
	for _, service := range metadata.CoreServices {
		if service.IsManual {
			// Reference manual module
			if service.ModuleName != "" {
				moduleBuilder.WriteString(fmt.Sprintf("\t%s,\n", service.ModuleName))
			}
		} else {
			// All services use fx.Provide to make them available for dependency injection
			moduleBuilder.WriteString(fmt.Sprintf("\tfx.Provide(New%s),\n", service.StructName))
		}
	}
	
	// Add logger providers to the module
	for _, logger := range metadata.Loggers {
		if logger.IsManual {
			// Reference manual module
			if logger.ModuleName != "" {
				moduleBuilder.WriteString(fmt.Sprintf("\t%s,\n", logger.ModuleName))
			}
		} else {
			// All loggers use fx.Provide to make them available for dependency injection
			moduleBuilder.WriteString(fmt.Sprintf("\tfx.Provide(New%s),\n", logger.StructName))
		}
	}
	
	// Add interface providers to the module
	for _, iface := range metadata.Interfaces {
		moduleBuilder.WriteString(fmt.Sprintf("\tfx.Provide(New%s),\n", iface.Name))
	}
	
	moduleBuilder.WriteString(")\n")
	
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
	
	return executeTemplate("interface", InterfaceTemplate, data)
}

// LoggerProviderData represents data needed for logger provider generation
type LoggerProviderData struct {
	StructName    string
	Dependencies  []DependencyData
	InjectedDeps  []DependencyData
	HasStart      bool
	HasStop       bool
	ConfigParam   string // Name of the config parameter
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
			// Check if this is a config dependency
			if strings.Contains(dep.Type, "Config") {
				configParam = depData.Name
			}
		}
	}

	// Check if this is a logger that needs immediate initialization
	hasLoggerField := false
	for _, dep := range dependencies {
		if dep.IsInit && (strings.Contains(dep.Type, "slog.Logger") || strings.Contains(dep.Type, "*slog.Logger")) {
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