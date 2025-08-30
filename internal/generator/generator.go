package generator

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/toyz/axon/internal/models"
	"github.com/toyz/axon/internal/registry"
	"github.com/toyz/axon/internal/templates"
	"github.com/toyz/axon/internal/utils"
	"github.com/toyz/axon/pkg/axon"
)

// ParserRegistryInterface defines the interface for parser registry operations
type ParserRegistryInterface interface {
	RegisterParser(parser axon.RouteParserMetadata) error
	GetParser(typeName string) (axon.RouteParserMetadata, bool)
	ListParsers() []string
	HasParser(typeName string) bool
	Clear()
	ClearCustomParsers()
	GetAllParsers() map[string]axon.RouteParserMetadata
}

// Generator implements the CodeGenerator interface
type Generator struct {
	moduleResolver ModuleResolver
	parserRegistry ParserRegistryInterface
}

// ModuleResolver interface for resolving module paths
type ModuleResolver interface {
	ResolveModuleName(customName string) (string, error)
	BuildPackagePath(moduleName, packageDir string) (string, error)
}

// NewGenerator creates a new code generator instance
func NewGenerator() *Generator {
	return &Generator{
		parserRegistry: registry.NewParserRegistry(),
	}
}

// NewGeneratorWithResolver creates a new code generator instance with a module resolver
func NewGeneratorWithResolver(resolver ModuleResolver) *Generator {
	return &Generator{
		moduleResolver: resolver,
		parserRegistry: registry.NewParserRegistry(),
	}
}

// GenerateModule generates a complete FX module file for a package with annotations
func (g *Generator) GenerateModule(metadata *models.PackageMetadata) (*models.GeneratedModule, error) {
	return g.GenerateModuleWithModule(metadata, "")
}

// GenerateModuleWithModule generates a complete FX module file for a package with annotations and module name
func (g *Generator) GenerateModuleWithModule(metadata *models.PackageMetadata, moduleName string) (*models.GeneratedModule, error) {
	return g.GenerateModuleWithPackagePaths(metadata, moduleName, nil)
}

// GenerateModuleWithPackagePaths generates a complete FX module file with package path mappings
func (g *Generator) GenerateModuleWithPackagePaths(metadata *models.PackageMetadata, moduleName string, packagePaths map[string]string) (*models.GeneratedModule, error) {
	return g.GenerateModuleWithRequiredPackages(metadata, moduleName, packagePaths, nil)
}

// GenerateModuleWithRequiredPackages generates a complete FX module file with package path mappings and required user packages
func (g *Generator) GenerateModuleWithRequiredPackages(metadata *models.PackageMetadata, moduleName string, packagePaths map[string]string, requiredPackages []string) (*models.GeneratedModule, error) {
	if metadata == nil {
		return nil, fmt.Errorf("metadata cannot be nil")
	}

	// Sort controllers by priority (lower priority numbers first, higher numbers last)
	sort.Slice(metadata.Controllers, func(i, j int) bool {
		return metadata.Controllers[i].Priority < metadata.Controllers[j].Priority
	})

	// Sort routes within each controller by priority (lower priority numbers first, higher numbers last)
	for i := range metadata.Controllers {
		sort.Slice(metadata.Controllers[i].Routes, func(j, k int) bool {
			return metadata.Controllers[i].Routes[j].Priority < metadata.Controllers[i].Routes[k].Priority
		})
	}

	// Determine the output file path
	filePath := filepath.Join(metadata.PackagePath, "autogen_module.go")

	// Generate the module content based on what annotations are present
	var content string
	var err error

	if len(metadata.Controllers) > 0 {
		// Generate controller module
		content, err = g.generateControllerModuleWithModule(metadata, moduleName, requiredPackages)
	} else if len(metadata.Middlewares) > 0 {
		// Generate middleware module
		content, err = g.generateMiddlewareModule(metadata)
	} else if len(metadata.CoreServices) > 0 || len(metadata.Interfaces) > 0 || len(metadata.Loggers) > 0 {
		// Generate core services module (includes loggers)
		content, err = templates.GenerateCoreServiceModuleWithResolver(metadata, moduleName, packagePaths)
	} else {
		// Empty module
		content = g.generateEmptyModule(metadata)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to generate module content: %w", err)
	}

	// Extract providers from the metadata
	providers := g.extractProviders(metadata)

	return &models.GeneratedModule{
		PackageName: metadata.PackageName,
		FilePath:    filePath,
		Content:     content,
		Providers:   providers,
	}, nil
}

// generateControllerModule generates a module file for packages with controllers
func (g *Generator) generateControllerModule(metadata *models.PackageMetadata) (string, error) {
	return g.generateControllerModuleWithModule(metadata, "", nil)
}

// generateControllerModuleWithModule generates a module file for packages with controllers with module name
func (g *Generator) generateControllerModuleWithModule(metadata *models.PackageMetadata, moduleName string, requiredPackages []string) (string, error) {
	var moduleBuilder strings.Builder

	// Generate package declaration with DO NOT EDIT header
	moduleBuilder.WriteString("// Code generated by Axon framework. DO NOT EDIT.\n")
	moduleBuilder.WriteString("// This file was automatically generated and should not be modified manually.\n\n")
	moduleBuilder.WriteString(fmt.Sprintf("package %s\n\n", metadata.PackageName))

	// Get module name for imports
	var resolvedModuleName string
	if g.moduleResolver != nil {
		var err error
		resolvedModuleName, err = g.moduleResolver.ResolveModuleName("")
		if err != nil {
			return "", fmt.Errorf("failed to resolve module name: %w", err)
		}
	} else {
		resolvedModuleName = moduleName // Use the provided module name
	}
	
	// Use provided required packages or detect them if not provided
	userPackages := requiredPackages
	if userPackages == nil {
		userPackages = g.detectRequiredUserPackages(metadata, resolvedModuleName)
	}
	
	// Generate minimal imports with user packages
	importsSection := templates.GenerateMinimalImportsWithPackages(resolvedModuleName, userPackages)
	moduleBuilder.WriteString(importsSection)

	// Generate shared helper functions for response handling
	if len(metadata.Controllers) > 0 {
		helperFunctions := g.generateResponseHelperFunctions()
		moduleBuilder.WriteString(helperFunctions)
		moduleBuilder.WriteString("\n\n")
	}

	// Generate controller providers
	for _, controller := range metadata.Controllers {
		providerCode, err := g.generateControllerProvider(controller)
		if err != nil {
			return "", fmt.Errorf("failed to generate provider for controller %s: %w", controller.Name, err)
		}
		moduleBuilder.WriteString(providerCode)
		moduleBuilder.WriteString("\n\n")
	}

	// Generate route wrapper functions
	for _, controller := range metadata.Controllers {
		for _, route := range controller.Routes {
			wrapperCode, err := templates.GenerateRouteWrapper(route, controller.StructName, g.parserRegistry)
			if err != nil {
				return "", fmt.Errorf("failed to generate wrapper for route %s.%s: %w", controller.Name, route.HandlerName, err)
			}
			moduleBuilder.WriteString(wrapperCode)
			moduleBuilder.WriteString("\n\n")
		}
	}

	// Generate route registration function
	registrationCode, err := g.generateRouteRegistrationFunction(metadata)
	if err != nil {
		return "", fmt.Errorf("failed to generate route registration function: %w", err)
	}
	moduleBuilder.WriteString(registrationCode)
	moduleBuilder.WriteString("\n\n")

	// Generate module variable
	moduleCode := g.generateControllerModuleVariable(metadata)
	moduleBuilder.WriteString(moduleCode)

	// Return raw generated code - formatting happens in post-processing phase
	return moduleBuilder.String(), nil
}

// detectRequiredUserPackages analyzes metadata to determine which user packages need to be imported
// This is a simplified version - the CLI will pass the required package information
func (g *Generator) detectRequiredUserPackages(metadata *models.PackageMetadata, moduleName string) []string {
	packageSet := make(map[string]bool)
	
	// For now, include common packages that are likely needed
	// The CLI should pass this information properly in the future
	
	// Check for service dependencies (if controllers inject services)
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

// generateControllerProvider generates an FX provider for a controller
func (g *Generator) generateControllerProvider(controller models.ControllerMetadata) (string, error) {
	// Convert dependencies to template data
	var deps []templates.DependencyData
	var injectedDeps []templates.DependencyData

	for _, dep := range controller.Dependencies {
		// Use the actual field name, converted to camelCase for parameter
		paramName := strings.ToLower(dep.Name[:1]) + dep.Name[1:]
		depData := templates.DependencyData{
			Name:      paramName,
			FieldName: dep.Name,
			Type:      dep.Type,
			IsInit:    dep.IsInit,
		}
		deps = append(deps, depData)

		// Only add to injected deps if it's not an init dependency
		if !dep.IsInit {
			injectedDeps = append(injectedDeps, depData)
		}
	}

	data := templates.CoreServiceProviderData{
		StructName:   controller.StructName,
		Dependencies: deps,
		InjectedDeps: injectedDeps,
		HasStart:     false,
		HasStop:      false,
	}

	return templates.ExecuteTemplate("controller-provider", templates.ProviderTemplate, data)
}



// extractPackageFromType is now available as utils.ExtractPackageFromType

// isWellKnownPackage checks if a package is already imported or is a well-known package
func (g *Generator) isWellKnownPackage(packageName string) bool {
	wellKnownPackages := map[string]bool{
		"echo":    true, // github.com/labstack/echo/v4
		"http":    true, // net/http
		"context": true, // context
		"fmt":     true, // fmt
		"errors":  true, // errors
		"uuid":    true, // github.com/google/uuid (already imported in controller)
		"axon":    true, // github.com/toyz/axon/pkg/axon (already imported)
	}

	return wellKnownPackages[packageName]
}

// resolvePackageImportPath resolves the import path for a package relative to the current package
func (g *Generator) resolvePackageImportPath(moduleName, currentPackagePath, targetPackage string) string {
	if g.moduleResolver != nil && moduleName != "" {
		// Use the module resolver to build the proper package path
		// Construct the target package directory path
		// This assumes the target package is a sibling of the current package
		baseDir := filepath.Dir(currentPackagePath)
		targetPackageDir := filepath.Join(baseDir, targetPackage)

		// Use the module resolver to build the import path
		if importPath, err := g.moduleResolver.BuildPackagePath(moduleName, targetPackageDir); err == nil {
			return importPath
		}

		// If BuildPackagePath fails, try to construct the path manually
		// Get current working directory
		if currentDir, err := os.Getwd(); err == nil {
			// Calculate relative path from current directory to target package
			if relPath, err := filepath.Rel(currentDir, targetPackageDir); err == nil {
				// Convert to import path format
				importPath := filepath.ToSlash(relPath)
				if importPath != "." {
					return fmt.Sprintf("%s/%s", moduleName, importPath)
				}
				return moduleName
			}
		}
	}

	// Final fallback to standard internal structure
	if moduleName != "" {
		return fmt.Sprintf("%s/internal/%s", moduleName, targetPackage)
	}

	// If moduleName is empty, try to detect it from the current working directory
	if cwd, err := os.Getwd(); err == nil {
		if goModPath := filepath.Join(cwd, "go.mod"); fileExists(goModPath) {
			if detectedModule := extractModuleNameFromGoMod(goModPath); detectedModule != "" {
				return fmt.Sprintf("%s/internal/%s", detectedModule, targetPackage)
			}
		}
	}

	// Last resort: use a reasonable default module path instead of relative imports
	// This avoids the "relative import paths are not supported in module mode" error
	return fmt.Sprintf("testmodule/%s", targetPackage)
}

// fileExists checks if a file exists
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// extractModuleNameFromGoMod extracts the module name from a go.mod file using the shared utility
func extractModuleNameFromGoMod(goModPath string) string {
	fileReader := utils.NewFileReader()
	goModParser := utils.NewGoModParser(fileReader)
	
	moduleName, err := goModParser.ParseModuleName(goModPath)
	if err != nil {
		return ""
	}
	return moduleName
}


// generateRouteRegistrationFunction generates a function that registers all routes with Echo
func (g *Generator) generateRouteRegistrationFunction(metadata *models.PackageMetadata) (string, error) {
	// Collect all middleware dependencies
	middlewareDeps := g.collectMiddlewareDependencies(metadata)

	// Build template data
	data := templates.RouteRegistrationData{
		Controllers:    make([]templates.ControllerTemplateData, 0, len(metadata.Controllers)),
		MiddlewareDeps: make([]templates.MiddlewareDependency, 0, len(middlewareDeps)),
	}

	// Convert middleware dependencies
	for _, mw := range middlewareDeps {
		data.MiddlewareDeps = append(data.MiddlewareDeps, templates.MiddlewareDependency{
			Name:        mw.Name,
			VarName:     strings.ToLower(mw.Name),
			PackageName: mw.PackageName,
		})
	}

	// Convert controllers and routes
	for _, controller := range metadata.Controllers {
		controllerData := templates.ControllerTemplateData{
			StructName: controller.StructName,
			VarName:    strings.ToLower(controller.StructName),
			Prefix:     controller.Prefix,
			EchoPrefix: g.convertToEchoPath(controller.Prefix),
			Routes:     make([]templates.RouteTemplateData, 0, len(controller.Routes)),
		}

		// Determine group variable
		var groupVar string
		if controller.Prefix != "" {
			groupVar = fmt.Sprintf("%sGroup", controllerData.VarName)
		} else {
			groupVar = "e"
		}

		// Convert routes
		for _, route := range controller.Routes {
			routeData, err := g.buildRouteTemplateData(route, controller, groupVar, metadata.PackageName)
			if err != nil {
				return "", fmt.Errorf("failed to build route template data for %s: %w", route.HandlerName, err)
			}
			controllerData.Routes = append(controllerData.Routes, routeData)
		}

		data.Controllers = append(data.Controllers, controllerData)
	}

	// Generate using template
	return templates.GenerateRouteRegistrationFunction(data)
}

// MiddlewareDependency represents a middleware with its package information
type MiddlewareDependency struct {
	Name        string // e.g., "AuthMiddleware"
	PackageName string // e.g., "middleware"
	ImportPath  string // e.g., "module/internal/middleware"
}

// collectMiddlewareDependencies collects all unique middleware used across routes with package info
func (g *Generator) collectMiddlewareDependencies(metadata *models.PackageMetadata) []MiddlewareDependency {
	middlewareSet := make(map[string]MiddlewareDependency)

	for _, controller := range metadata.Controllers {
		for _, route := range controller.Routes {
			for _, middlewareName := range route.Middlewares {
				// For now, assume all middleware are in the "middleware" package
				// This should be enhanced to get actual package info from global registry
				middlewareSet[middlewareName] = MiddlewareDependency{
					Name:        middlewareName,
					PackageName: "middleware",
					ImportPath:  "internal/middleware", // This should come from actual package detection
				}
			}
		}
	}

	var middlewares []MiddlewareDependency
	for _, middleware := range middlewareSet {
		middlewares = append(middlewares, middleware)
	}

	return middlewares
}

// extractParameterTypes extracts parameter types from a route path
// e.g., "/users/{id:int}/posts/{slug:string}" -> {"id": "int", "slug": "string"}
func (g *Generator) extractParameterTypes(path string) map[string]string {
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

// getMiddlewareReference returns the package-qualified reference for a middleware name
func (g *Generator) getMiddlewareReference(middlewareName string, middlewareDeps []MiddlewareDependency) string {
	for _, dep := range middlewareDeps {
		if dep.Name == middlewareName {
			return fmt.Sprintf("%s.%s", strings.ToLower(dep.Name), dep.PackageName)
		}
	}
	// Fallback to just the lowercase name if not found
	return strings.ToLower(middlewareName)
}

// generateControllerModuleVariable generates the FX module variable for controllers
func (g *Generator) generateControllerModuleVariable(metadata *models.PackageMetadata) string {
	var moduleBuilder strings.Builder

	moduleBuilder.WriteString("// AutogenModule provides all controllers and route registration in this package\n")
	moduleBuilder.WriteString("var AutogenModule = fx.Module(\"")
	moduleBuilder.WriteString(metadata.PackageName)
	moduleBuilder.WriteString("\",\n")

	// Add controller providers
	for _, controller := range metadata.Controllers {
		moduleBuilder.WriteString(fmt.Sprintf("\tfx.Provide(New%s),\n", controller.StructName))
	}

	// Add route registration as an invoke
	moduleBuilder.WriteString("\tfx.Invoke(RegisterRoutes),\n")

	moduleBuilder.WriteString(")\n")

	// Return the unformatted code - goimports will run later on all files
	return moduleBuilder.String()
}

// generateMiddlewareModule generates a module file for packages with middleware
func (g *Generator) generateMiddlewareModule(metadata *models.PackageMetadata) (string, error) {
	// Use the unified template system with ImportManager
	return templates.GenerateMiddlewareModule(metadata)
}

// Removed old inline middleware generation functions - now using templates

// generateEmptyModule generates an empty module for packages with no annotations
func (g *Generator) generateEmptyModule(metadata *models.PackageMetadata) string {
	return fmt.Sprintf(`// Code generated by Axon framework. DO NOT EDIT.
// This file was automatically generated and should not be modified manually.

package %s

import "go.uber.org/fx"

// AutogenModule provides an empty module for this package
var AutogenModule = fx.Module("%s")
`, metadata.PackageName, metadata.PackageName)
}

// extractProviders extracts provider information from package metadata
func (g *Generator) extractProviders(metadata *models.PackageMetadata) []models.Provider {
	var providers []models.Provider

	// Add controller providers
	for _, controller := range metadata.Controllers {
		providers = append(providers, models.Provider{
			Name:         fmt.Sprintf("New%s", controller.StructName),
			StructName:   controller.StructName,
			Dependencies: controller.Dependencies,
			IsLifecycle:  false,
		})
	}

	// Add core service providers
	for _, service := range metadata.CoreServices {
		if !service.IsManual {
			providers = append(providers, models.Provider{
				Name:         fmt.Sprintf("New%s", service.StructName),
				StructName:   service.StructName,
				Dependencies: service.Dependencies,
				IsLifecycle:  service.HasLifecycle,
			})
		}
	}

	// Add interface providers
	for _, iface := range metadata.Interfaces {
		providers = append(providers, models.Provider{
			Name:         fmt.Sprintf("New%s", iface.Name),
			StructName:   iface.StructName,
			Dependencies: []models.Dependency{{Name: "impl", Type: fmt.Sprintf("*%s", iface.StructName)}},
			IsLifecycle:  false,
		})
	}

	return providers
}

// GenerateRootModule generates an autogen_root_module.go file that combines sub-package modules
func (g *Generator) GenerateRootModule(packageName string, subModules []models.ModuleReference, outputPath string) error {
	if len(subModules) == 0 {
		return fmt.Errorf("no sub-modules provided for root module generation")
	}

	var rootBuilder strings.Builder

	// Generate package declaration
	rootBuilder.WriteString(fmt.Sprintf("package %s\n\n", packageName))

	// Generate imports
	rootBuilder.WriteString("import (\n")
	rootBuilder.WriteString("\t\"go.uber.org/fx\"\n")

	// Add imports for all sub-module packages
	for _, module := range subModules {
		if module.PackagePath != "" {
			rootBuilder.WriteString(fmt.Sprintf("\t\"%s\"\n", module.PackagePath))
		}
	}

	rootBuilder.WriteString(")\n\n")

	// Generate root module variable
	rootBuilder.WriteString("// AutogenRootModule combines all sub-package modules\n")
	rootBuilder.WriteString(fmt.Sprintf("var AutogenRootModule = fx.Module(\"%s\",\n", packageName))

	// Add all sub-modules
	for _, module := range subModules {
		if module.PackageName != "" && module.ModuleName != "" {
			rootBuilder.WriteString(fmt.Sprintf("\t%s.%s,\n", module.PackageName, module.ModuleName))
		}
	}

	rootBuilder.WriteString(")\n")

	// Write the file
	err := os.MkdirAll(filepath.Dir(outputPath), 0755)
	if err != nil {
		return fmt.Errorf("failed to create directory for root module file: %w", err)
	}

	// Format the generated code before writing
	formattedCode, err := utils.FormatGoCodeString(rootBuilder.String())
	if err != nil {
		// If formatting fails, write the unformatted code with a warning
		formattedCode = rootBuilder.String()
	}

	err = os.WriteFile(outputPath, []byte(formattedCode), 0644)
	if err != nil {
		return fmt.Errorf("failed to write root module file: %w", err)
	}

	return nil
}


// generateResponseHelperFunctions generates shared helper functions for response handling
func (g *Generator) generateResponseHelperFunctions() string {
	return `// handleAxonResponse processes an axon.Response and applies headers, cookies, and content type
func handleAxonResponse(c echo.Context, response *axon.Response) error {
	// Set headers
	for key, value := range response.Headers {
		c.Response().Header().Set(key, value)
	}
	
	// Set cookies
	for _, cookie := range response.Cookies {
		httpCookie := &http.Cookie{
			Name:     cookie.Name,
			Value:    cookie.Value,
			Path:     cookie.Path,
			Domain:   cookie.Domain,
			MaxAge:   cookie.MaxAge,
			Secure:   cookie.Secure,
			HttpOnly: cookie.HttpOnly,
		}
		if cookie.SameSite != "" {
			switch cookie.SameSite {
			case "Strict":
				httpCookie.SameSite = http.SameSiteStrictMode
			case "Lax":
				httpCookie.SameSite = http.SameSiteLaxMode
			case "None":
				httpCookie.SameSite = http.SameSiteNoneMode
			}
		}
		c.SetCookie(httpCookie)
	}
	
	// Set content type and return response
	if response.ContentType != "" {
		return c.Blob(response.StatusCode, response.ContentType, []byte(fmt.Sprintf("%v", response.Body)))
	}
	return c.JSON(response.StatusCode, response.Body)
}

// handleHttpError processes an axon.HttpError and returns appropriate JSON response
func handleHttpError(c echo.Context, httpErr *axon.HttpError) error {
	return c.JSON(httpErr.StatusCode, httpErr)
}

// handleError processes any error and returns appropriate response (HttpError or generic error)
func handleError(c echo.Context, err error) error {
	if httpErr, ok := err.(*axon.HttpError); ok {
		return handleHttpError(c, httpErr)
	}
	return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
}`
}

// GetParserRegistry returns the parser registry for cross-package parser discovery
func (g *Generator) GetParserRegistry() ParserRegistryInterface {
	return g.parserRegistry
}
// convertToEchoPath converts Axon path format to Echo path format
func (g *Generator) convertToEchoPath(axonPath string) string {
	// Convert {param:type} to :param
	re := regexp.MustCompile(`\{([^:}]+):[^}]+\}`)
	return re.ReplaceAllString(axonPath, ":$1")
}

// buildRouteTemplateData builds template data for a single route
func (g *Generator) buildRouteTemplateData(route models.RouteMetadata, controller models.ControllerMetadata, groupVar, packageName string) (templates.RouteTemplateData, error) {
	controllerVar := strings.ToLower(controller.StructName)
	handlerVar := fmt.Sprintf("handler_%s%s", controllerVar, strings.ToLower(route.HandlerName))
	wrapperFunc := fmt.Sprintf("wrap%s%s", controller.StructName, route.HandlerName)
	
	// Combine controller middleware and route middleware
	allMiddlewares := append([]string{}, controller.Middlewares...)
	allMiddlewares = append(allMiddlewares, route.Middlewares...)
	
	// Calculate route path relative to group
	routePath := route.Path
	if controller.Prefix != "" {
		// Remove controller prefix from route path since group already has it
		if strings.HasPrefix(route.Path, controller.Prefix) {
			routePath = strings.TrimPrefix(route.Path, controller.Prefix)
			if routePath == "" {
				routePath = "/"
			}
		}
	}
	
	echoPath := g.convertToEchoPath(routePath)
	paramTypes := templates.ExtractParameterTypes(route.Path)
	
	return templates.RouteTemplateData{
		HandlerVar:               handlerVar,
		WrapperFunc:              wrapperFunc,
		ControllerVar:            controllerVar,
		GroupVar:                 groupVar,
		Method:                   route.Method,
		Path:                     route.Path,
		EchoPath:                 echoPath,
		HandlerName:              route.HandlerName,
		ControllerName:           controller.StructName,
		PackageName:              packageName,
		HasMiddleware:            len(allMiddlewares) > 0,
		MiddlewareList:           templates.BuildMiddlewareList(allMiddlewares),
		MiddlewaresArray:         templates.BuildMiddlewaresArray(allMiddlewares),
		MiddlewareInstancesArray: templates.BuildMiddlewareInstancesArray(allMiddlewares),
		ParameterInstancesArray:  templates.BuildParameterInstancesArray(paramTypes),
	}, nil
}