package generator

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/toyz/axon/internal/models"
)

// TestGeneratedCodeCompilation tests that generated code compiles successfully
func TestGeneratedCodeCompilation(t *testing.T) {
	// Create temporary directory for test
	tempDir, err := os.MkdirTemp("", "generator_integration_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a test module directory
	moduleDir := filepath.Join(tempDir, "testmodule")
	err = os.MkdirAll(moduleDir, 0755)
	if err != nil {
		t.Fatalf("failed to create module dir: %v", err)
	}

	// Initialize go module
	cmd := exec.Command("go", "mod", "init", "testmodule")
	cmd.Dir = moduleDir
	err = cmd.Run()
	if err != nil {
		t.Fatalf("failed to initialize go module: %v", err)
	}
	
	// Add required dependencies
	dependencies := []string{
		"github.com/labstack/echo/v4@latest",
		"go.uber.org/fx@latest",
	}
	
	for _, dep := range dependencies {
		cmd = exec.Command("go", "get", dep)
		cmd.Dir = moduleDir
		err = cmd.Run()
		if err != nil {
			t.Fatalf("failed to add dependency %s: %v", dep, err)
		}
	}
	
	// Add a replace directive for the axon package
	goModPath := filepath.Join(moduleDir, "go.mod")
	goModBytes, err := os.ReadFile(goModPath)
	if err != nil {
		t.Fatalf("failed to read go.mod: %v", err)
	}
	
	goModContent := string(goModBytes) + "\nreplace github.com/toyz/axon => ./\n"
	err = os.WriteFile(goModPath, []byte(goModContent), 0644)
	if err != nil {
		t.Fatalf("failed to write go.mod: %v", err)
	}
	
	// Add the axon package as a requirement
	cmd = exec.Command("go", "get", "github.com/toyz/axon")
	cmd.Dir = moduleDir
	err = cmd.Run()
	if err != nil {
		t.Fatalf("failed to add axon dependency: %v", err)
	}



	// Generate a controller module
	generator := NewGenerator()
	
	metadata := &models.PackageMetadata{
		PackageName: "testmodule",
		PackagePath: moduleDir,
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
					{
						Method:      "POST",
						Path:        "/users",
						HandlerName: "CreateUser",
						Parameters: []models.Parameter{
							{
								Name:   "user",
								Type:   "User",
								Source: models.ParameterSourceBody,
							},
						},
						ReturnType: models.ReturnTypeInfo{
							Type: models.ReturnTypeResponseError,
						},
					},
				},
				Dependencies: []string{"UserService"},
			},
		},
	}

	// Create the controller struct file first
	controllerCode := `package testmodule

// User represents a user entity
type User struct {
	ID   int    ` + "`json:\"id\"`" + `
	Name string ` + "`json:\"name\"`" + `
}

// Response represents an HTTP response
type Response struct {
	StatusCode int
	Body       interface{}
}

// UserService represents the user service
type UserService struct{}

// UserController handles user-related HTTP requests
type UserController struct {
	UserService UserService ` + "`fx:\"in\"`" + `
}

// GetUser retrieves a user by ID
func (c *UserController) GetUser(id int) (*User, error) {
	return &User{ID: id, Name: "Test User"}, nil
}

// CreateUser creates a new user
func (c *UserController) CreateUser(user User) (*Response, error) {
	return &Response{
		StatusCode: 201,
		Body:       user,
	}, nil
}
`

	err = os.WriteFile(filepath.Join(moduleDir, "controller.go"), []byte(controllerCode), 0644)
	if err != nil {
		t.Fatalf("failed to write controller file: %v", err)
	}

	// Create a mock axon package for testing
	axonDir := filepath.Join(moduleDir, "pkg", "axon")
	err = os.MkdirAll(axonDir, 0755)
	if err != nil {
		t.Fatalf("failed to create axon package dir: %v", err)
	}
	
	mockAxonCode := `package axon

import "github.com/labstack/echo/v4"

type MiddlewareInstance struct {
	Name     string
	Handler  func(echo.HandlerFunc) echo.HandlerFunc
	Instance interface{}
}

type RouteInfo struct {
	Method              string
	Path                string
	EchoPath            string
	HandlerName         string
	ControllerName      string
	PackageName         string
	Middlewares         []string
	MiddlewareInstances []MiddlewareInstance
	ParameterTypes      map[string]string
	Handler             echo.HandlerFunc
}

type RouteRegistry interface {
	RegisterRoute(route RouteInfo)
}

type inMemoryRegistry struct{}

func (r *inMemoryRegistry) RegisterRoute(route RouteInfo) {}

var DefaultRouteRegistry RouteRegistry = &inMemoryRegistry{}
`
	
	err = os.WriteFile(filepath.Join(axonDir, "registry.go"), []byte(mockAxonCode), 0644)
	if err != nil {
		t.Fatalf("failed to write mock axon package: %v", err)
	}

	// Generate the module
	result, err := generator.GenerateModule(metadata)
	if err != nil {
		t.Fatalf("failed to generate module: %v", err)
	}

	// Write the generated module file
	err = os.WriteFile(result.FilePath, []byte(result.Content), 0644)
	if err != nil {
		t.Fatalf("failed to write generated module file: %v", err)
	}

	// Try to compile the generated code
	cmd = exec.Command("go", "build", "./...")
	cmd.Dir = moduleDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("generated code failed to compile: %v\nOutput: %s\nGenerated code:\n%s", err, output, result.Content)
	}
}

// TestGeneratedCoreServiceModuleCompilation tests core service module compilation
func TestGeneratedCoreServiceModuleCompilation(t *testing.T) {
	// Create temporary directory for test
	tempDir, err := os.MkdirTemp("", "generator_core_integration_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a test module directory
	moduleDir := filepath.Join(tempDir, "testservices")
	err = os.MkdirAll(moduleDir, 0755)
	if err != nil {
		t.Fatalf("failed to create module dir: %v", err)
	}

	// Initialize go module
	cmd := exec.Command("go", "mod", "init", "testservices")
	cmd.Dir = moduleDir
	err = cmd.Run()
	if err != nil {
		t.Fatalf("failed to initialize go module: %v", err)
	}

	// Add required dependencies
	cmd = exec.Command("go", "get", "go.uber.org/fx@latest")
	cmd.Dir = moduleDir
	err = cmd.Run()
	if err != nil {
		t.Fatalf("failed to add fx dependency: %v", err)
	}

	// Generate a core services module
	generator := NewGenerator()
	
	metadata := &models.PackageMetadata{
		PackageName: "testservices",
		PackagePath: moduleDir,
		CoreServices: []models.CoreServiceMetadata{
			{
				Name:         "UserService",
				StructName:   "UserService",
				HasLifecycle: false,
				IsManual:     false,
				Dependencies: []string{"UserRepository"},
			},
			{
				Name:         "DatabaseService",
				StructName:   "DatabaseService",
				HasLifecycle: true,
				HasStart:     true,
				HasStop:      true,
				IsManual:     false,
				Dependencies: []string{"*Config"},
			},
		},
		Interfaces: []models.InterfaceMetadata{
			{
				Name:       "UserServiceInterface",
				StructName: "UserService",
				Methods: []models.Method{
					{
						Name: "GetUser",
						Parameters: []models.Parameter{
							{Name: "id", Type: "int"},
						},
						Returns: []string{"*User", "error"},
					},
				},
			},
		},
	}

	// Create the service struct files first
	serviceCode := `package testservices

import "context"

// User represents a user entity
type User struct {
	ID   int
	Name string
}

// Config represents application configuration
type Config struct {
	DatabaseURL string
}

// UserRepository represents the user repository
type UserRepository struct{}

// UserService handles user business logic
type UserService struct {
	UserRepository UserRepository ` + "`fx:\"in\"`" + `
}

// GetUser retrieves a user by ID
func (s *UserService) GetUser(id int) (*User, error) {
	return &User{ID: id, Name: "Test User"}, nil
}

// DatabaseService handles database connections
type DatabaseService struct {
	Config *Config ` + "`fx:\"in\"`" + `
}

// Start initializes the database connection
func (s *DatabaseService) Start(ctx context.Context) error {
	return nil
}

// Stop closes the database connection
func (s *DatabaseService) Stop(ctx context.Context) error {
	return nil
}
`

	err = os.WriteFile(filepath.Join(moduleDir, "services.go"), []byte(serviceCode), 0644)
	if err != nil {
		t.Fatalf("failed to write services file: %v", err)
	}

	// Generate the module
	result, err := generator.GenerateModule(metadata)
	if err != nil {
		t.Fatalf("failed to generate module: %v", err)
	}

	// Write the generated module file
	err = os.WriteFile(result.FilePath, []byte(result.Content), 0644)
	if err != nil {
		t.Fatalf("failed to write generated module file: %v", err)
	}

	// Try to compile the generated code
	cmd = exec.Command("go", "build", "./...")
	cmd.Dir = moduleDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("generated code failed to compile: %v\nOutput: %s\nGenerated code:\n%s", err, output, result.Content)
	}
}

// TestGeneratedMainFileCompilation tests main.go compilation
func TestGeneratedMainFileCompilation(t *testing.T) {
	// Create temporary directory for test
	tempDir, err := os.MkdirTemp("", "generator_main_integration_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Initialize go module
	cmd := exec.Command("go", "mod", "init", "testapp")
	cmd.Dir = tempDir
	err = cmd.Run()
	if err != nil {
		t.Fatalf("failed to initialize go module: %v", err)
	}

	// Add required dependencies
	dependencies := []string{
		"github.com/labstack/echo/v4@latest",
		"go.uber.org/fx@latest",
	}
	
	for _, dep := range dependencies {
		cmd = exec.Command("go", "get", dep)
		cmd.Dir = tempDir
		err = cmd.Run()
		if err != nil {
			t.Fatalf("failed to add dependency %s: %v", dep, err)
		}
	}

	// Create mock module files
	controllersDir := filepath.Join(tempDir, "controllers")
	err = os.MkdirAll(controllersDir, 0755)
	if err != nil {
		t.Fatalf("failed to create controllers dir: %v", err)
	}

	controllerModule := `package controllers

import "go.uber.org/fx"

// AutogenModule provides all controllers in this package
var AutogenModule = fx.Module("controllers")
`

	err = os.WriteFile(filepath.Join(controllersDir, "autogen_module.go"), []byte(controllerModule), 0644)
	if err != nil {
		t.Fatalf("failed to write controller module: %v", err)
	}

	servicesDir := filepath.Join(tempDir, "services")
	err = os.MkdirAll(servicesDir, 0755)
	if err != nil {
		t.Fatalf("failed to create services dir: %v", err)
	}

	serviceModule := `package services

import "go.uber.org/fx"

// AutogenModule provides all services in this package
var AutogenModule = fx.Module("services")
`

	err = os.WriteFile(filepath.Join(servicesDir, "autogen_module.go"), []byte(serviceModule), 0644)
	if err != nil {
		t.Fatalf("failed to write service module: %v", err)
	}

	// Try to compile the generated modules
	cmd = exec.Command("go", "build", "./...")
	cmd.Dir = tempDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("generated modules failed to compile: %v\nOutput: %s", err, output)
	}
}