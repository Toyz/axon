package generator

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/toyz/axon/internal/models"
)

func TestNewGenerator(t *testing.T) {
	generator := NewGenerator()
	if generator == nil {
		t.Fatal("NewGenerator() returned nil")
	}
}

func TestGenerateModule_NilMetadata(t *testing.T) {
	generator := NewGenerator()
	
	_, err := generator.GenerateModule(nil)
	if err == nil {
		t.Fatal("expected error for nil metadata")
	}
	
	if !strings.Contains(err.Error(), "metadata cannot be nil") {
		t.Errorf("expected error message about nil metadata, got: %v", err)
	}
}

func TestGenerateModule_EmptyPackage(t *testing.T) {
	generator := NewGenerator()
	
	metadata := &models.PackageMetadata{
		PackageName: "empty",
		PackagePath: "./empty",
	}
	
	result, err := generator.GenerateModule(metadata)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	
	if result.PackageName != "empty" {
		t.Errorf("expected package name 'empty', got %s", result.PackageName)
	}
	
	expectedPath := filepath.Join("./empty", "autogen_module.go")
	if result.FilePath != expectedPath {
		t.Errorf("expected file path %s, got %s", expectedPath, result.FilePath)
	}
	
	// Check that it generates an empty module
	if !strings.Contains(result.Content, "package empty") {
		t.Errorf("expected package declaration, got: %s", result.Content)
	}
	
	if !strings.Contains(result.Content, "var AutogenModule = fx.Module(\"empty\")") {
		t.Errorf("expected empty module declaration, got: %s", result.Content)
	}
}

func TestGenerateModule_CoreServices(t *testing.T) {
	generator := NewGenerator()
	
	metadata := &models.PackageMetadata{
		PackageName: "services",
		PackagePath: "./services",
		CoreServices: []models.CoreServiceMetadata{
			{
				Name:         "UserService",
				StructName:   "UserService",
				HasLifecycle: false,
				IsManual:     false,
				Dependencies: []models.Dependency{{Name: "UserRepository", Type: "UserRepository"}},
			},
		},
	}
	
	result, err := generator.GenerateModule(metadata)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	
	// Check basic structure
	if !strings.Contains(result.Content, "package services") {
		t.Errorf("expected package declaration")
	}
	
	if !strings.Contains(result.Content, "func NewUserService(") {
		t.Errorf("expected provider function")
	}
	
	if !strings.Contains(result.Content, "fx.Provide(NewUserService)") {
		t.Errorf("expected provider in module")
	}
	
	// Check providers
	if len(result.Providers) != 1 {
		t.Errorf("expected 1 provider, got %d", len(result.Providers))
	}
	
	provider := result.Providers[0]
	if provider.Name != "NewUserService" {
		t.Errorf("expected provider name 'NewUserService', got %s", provider.Name)
	}
}

func TestGenerateModule_Controllers(t *testing.T) {
	generator := NewGenerator()
	
	metadata := &models.PackageMetadata{
		PackageName: "controllers",
		PackagePath: "./controllers",
		Controllers: []models.ControllerMetadata{
			{
				Name:       "UserController",
				StructName: "UserController",
				Routes: []models.RouteMetadata{
					{
						Method:      "GET",
						Path:        "/users/{id:int}",
						HandlerName: "GetUser",
						Parameters: []models.Parameter{
							{
								Name:   "id",
								Type:   "int",
								Source: models.ParameterSourcePath,
							},
						},
						ReturnType: models.ReturnTypeInfo{
							Type: models.ReturnTypeDataError,
						},
					},
				},
				Dependencies: []models.Dependency{{Name: "UserService", Type: "UserService"}},
			},
		},
	}
	
	result, err := generator.GenerateModule(metadata)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	
	// Check basic structure
	if !strings.Contains(result.Content, "package controllers") {
		t.Errorf("expected package declaration")
	}
	
	// Check imports
	expectedImports := []string{
		"\"net/http\"",
		"\"fmt\"",
		"\"github.com/labstack/echo/v4\"",
		"\"go.uber.org/fx\"",
		"\"github.com/toyz/axon/pkg/axon\"",
	}
	
	for _, imp := range expectedImports {
		if !strings.Contains(result.Content, imp) {
			t.Errorf("expected import %s", imp)
		}
	}
	
	// Check controller provider
	if !strings.Contains(result.Content, "func NewUserController(") {
		t.Errorf("expected controller provider function")
	}
	
	// Check route wrapper
	if !strings.Contains(result.Content, "func wrapUserControllerGetUser(") {
		t.Errorf("expected route wrapper function")
	}
	
	// Check route registration function
	if !strings.Contains(result.Content, "func RegisterRoutes(") {
		t.Errorf("expected route registration function")
	}
	
	// Check module variable
	if !strings.Contains(result.Content, "var AutogenModule = fx.Module(") {
		t.Errorf("expected module variable")
	}
	
	if !strings.Contains(result.Content, "fx.Provide(NewUserController)") {
		t.Errorf("expected controller provider in module")
	}
	
	if !strings.Contains(result.Content, "fx.Invoke(RegisterRoutes)") {
		t.Errorf("expected route registration in module")
	}
}

func TestGenerateControllerProvider(t *testing.T) {
	generator := NewGenerator()
	
	controller := models.ControllerMetadata{
		Name:       "UserController",
		StructName: "UserController",
		Dependencies: []models.Dependency{
			{Name: "UserService", Type: "UserService"},
			{Name: "Config", Type: "*Config"},
		},
	}
	
	result, err := generator.generateControllerProvider(controller)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	
	// Check function signature
	if !strings.Contains(result, "func NewUserController(") {
		t.Errorf("expected function name")
	}
	
	// Check dependencies
	if !strings.Contains(result, "userService UserService") {
		t.Errorf("expected UserService parameter")
	}
	
	if !strings.Contains(result, "config *Config") {
		t.Errorf("expected Config parameter")
	}
	
	// Check return statement
	if !strings.Contains(result, "return &UserController{") {
		t.Errorf("expected return statement")
	}
}

func TestCollectMiddlewareDependencies(t *testing.T) {
	generator := NewGenerator()
	
	metadata := &models.PackageMetadata{
		Controllers: []models.ControllerMetadata{
			{
				Routes: []models.RouteMetadata{
					{
						Middlewares: []string{"Auth", "Logging"},
					},
					{
						Middlewares: []string{"Auth", "RateLimit"},
					},
				},
			},
		},
	}
	
	result := generator.collectMiddlewareDependencies(metadata)
	
	// Should have unique middlewares
	expectedCount := 3 // Auth, Logging, RateLimit
	if len(result) != expectedCount {
		t.Errorf("expected %d unique middlewares, got %d: %v", expectedCount, len(result), result)
	}
	
	// Check that all expected middlewares are present
	middlewareSet := make(map[string]bool)
	for _, middleware := range result {
		middlewareSet[middleware] = true
	}
	
	expected := []string{"Auth", "Logging", "RateLimit"}
	for _, middleware := range expected {
		if !middlewareSet[middleware] {
			t.Errorf("expected middleware %s not found in result", middleware)
		}
	}
}

func TestExtractProviders(t *testing.T) {
	generator := NewGenerator()
	
	metadata := &models.PackageMetadata{
		Controllers: []models.ControllerMetadata{
			{
				Name:         "UserController",
				StructName:   "UserController",
				Dependencies: []models.Dependency{{Name: "UserService", Type: "UserService"}},
			},
		},
		CoreServices: []models.CoreServiceMetadata{
			{
				Name:         "UserService",
				StructName:   "UserService",
				HasLifecycle: true,
				IsManual:     false,
				Dependencies: []models.Dependency{{Name: "UserRepository", Type: "UserRepository"}},
			},
			{
				Name:         "ConfigService",
				StructName:   "ConfigService",
				IsManual:     true,
				ModuleName:   "CustomModule",
			},
		},
		Interfaces: []models.InterfaceMetadata{
			{
				Name:       "UserServiceInterface",
				StructName: "UserService",
			},
		},
	}
	
	result := generator.extractProviders(metadata)
	
	// Should have 3 providers: controller, core service (not manual), interface
	expectedCount := 3
	if len(result) != expectedCount {
		t.Errorf("expected %d providers, got %d", expectedCount, len(result))
	}
	
	// Check controller provider
	controllerProvider := findProvider(result, "NewUserController")
	if controllerProvider == nil {
		t.Errorf("expected controller provider not found")
	} else {
		if controllerProvider.IsLifecycle {
			t.Errorf("controller provider should not be lifecycle")
		}
	}
	
	// Check core service provider
	serviceProvider := findProvider(result, "NewUserService")
	if serviceProvider == nil {
		t.Errorf("expected service provider not found")
	} else {
		if !serviceProvider.IsLifecycle {
			t.Errorf("service provider should be lifecycle")
		}
	}
	
	// Check interface provider
	interfaceProvider := findProvider(result, "NewUserServiceInterface")
	if interfaceProvider == nil {
		t.Errorf("expected interface provider not found")
	}
	
	// Manual service should not have a provider
	manualProvider := findProvider(result, "NewConfigService")
	if manualProvider != nil {
		t.Errorf("manual service should not have a provider")
	}
}

func findProvider(providers []models.Provider, name string) *models.Provider {
	for _, provider := range providers {
		if provider.Name == name {
			return &provider
		}
	}
	return nil
}



func TestGenerateRootModule(t *testing.T) {
	generator := NewGenerator()
	
	// Create temporary directory for test
	tempDir, err := os.MkdirTemp("", "generator_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)
	
	outputPath := filepath.Join(tempDir, "autogen_root_module.go")
	
	subModules := []models.ModuleReference{
		{
			PackageName: "controllers",
			PackagePath: "github.com/example/app/controllers",
			ModuleName:  "AutogenModule",
		},
		{
			PackageName: "services",
			PackagePath: "github.com/example/app/services",
			ModuleName:  "AutogenModule",
		},
	}
	
	err = generator.GenerateRootModule("app", subModules, outputPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	
	// Check that file was created
	if _, err := os.Stat(outputPath); os.IsNotExist(err) {
		t.Fatalf("root module file was not created")
	}
	
	// Read and check content
	content, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("failed to read generated file: %v", err)
	}
	
	contentStr := string(content)
	
	// Check basic structure
	expectedContents := []string{
		"package app",
		"import (",
		"\"go.uber.org/fx\"",
		"\"github.com/example/app/controllers\"",
		"\"github.com/example/app/services\"",
		"var AutogenRootModule = fx.Module(",
		"controllers.AutogenModule,",
		"services.AutogenModule,",
	}
	
	for _, expected := range expectedContents {
		if !strings.Contains(contentStr, expected) {
			t.Errorf("expected content '%s' not found in generated root module", expected)
		}
	}
}

func TestGenerateRootModule_NoSubModules(t *testing.T) {
	generator := NewGenerator()
	
	err := generator.GenerateRootModule("app", []models.ModuleReference{}, "root.go")
	if err == nil {
		t.Fatal("expected error for empty sub-modules")
	}
	
	if !strings.Contains(err.Error(), "no sub-modules provided") {
		t.Errorf("expected error message about no sub-modules, got: %v", err)
	}
}

func TestExtractDependencyName(t *testing.T) {
	tests := []struct {
		depType  string
		expected string
	}{
		{"UserRepository", "UserRepository"},
		{"*Config", "Config"},
		{"pkg.Service", "Service"},
		{"*pkg.Interface", "Interface"},
		{"HTTPClient", "HTTPClient"},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.depType, func(t *testing.T) {
			result := extractDependencyName(tt.depType)
			if result != tt.expected {
				t.Errorf("extractDependencyName(%s) = %s, expected %s", tt.depType, result, tt.expected)
			}
		})
	}
}