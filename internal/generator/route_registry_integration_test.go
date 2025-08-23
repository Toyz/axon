package generator

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/toyz/axon/internal/models"
)

// TestRouteRegistryIntegration tests that generated code properly integrates with the route registry
func TestRouteRegistryIntegration(t *testing.T) {
	// Create temporary directory for test
	tempDir, err := os.MkdirTemp("", "route_registry_integration_test")
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

	// Generate a controller module with middleware
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
						Middlewares: []string{"Auth", "Logging"},
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
						Middlewares: []string{"Auth", "Validation"},
					},
				},
				Dependencies: []string{"UserService"},
			},
		},
	}

	// Create the controller struct file first
	controllerCode := `package testmodule

import "github.com/labstack/echo/v4"

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

// Auth middleware
type Auth struct{}

func (a *Auth) Handle(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		// Auth logic here
		return next(c)
	}
}

// Logging middleware
type Logging struct{}

func (l *Logging) Handle(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		// Logging logic here
		return next(c)
	}
}

// Validation middleware
type Validation struct{}

func (v *Validation) Handle(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		// Validation logic here
		return next(c)
	}
}

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

	// Create the axon package for testing
	axonDir := filepath.Join(moduleDir, "pkg", "axon")
	err = os.MkdirAll(axonDir, 0755)
	if err != nil {
		t.Fatalf("failed to create axon package dir: %v", err)
	}
	
	// Copy the actual axon registry code for testing
	axonRegistryCode := `package axon

import "github.com/labstack/echo/v4"

// MiddlewareInstance represents a middleware with its name and handler
type MiddlewareInstance struct {
	Name     string
	Handler  func(echo.HandlerFunc) echo.HandlerFunc
	Instance interface{}
}

// RouteInfo contains metadata about a registered route
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

// RouteRegistry provides access to all registered routes in the application
type RouteRegistry interface {
	GetAllRoutes() []RouteInfo
	RegisterRoute(route RouteInfo)
}

// InMemoryRouteRegistry implements RouteRegistry using an in-memory slice
type InMemoryRouteRegistry struct {
	routes []RouteInfo
}

// NewInMemoryRouteRegistry creates a new in-memory route registry
func NewInMemoryRouteRegistry() *InMemoryRouteRegistry {
	return &InMemoryRouteRegistry{
		routes: make([]RouteInfo, 0),
	}
}

// GetAllRoutes returns all registered routes
func (r *InMemoryRouteRegistry) GetAllRoutes() []RouteInfo {
	return append([]RouteInfo(nil), r.routes...) // Return a copy
}

// RegisterRoute adds a route to the registry
func (r *InMemoryRouteRegistry) RegisterRoute(route RouteInfo) {
	r.routes = append(r.routes, route)
}

// DefaultRouteRegistry is the global route registry instance
var DefaultRouteRegistry RouteRegistry = NewInMemoryRouteRegistry()

// GetRoutes returns all registered routes (convenience function)
func GetRoutes() []RouteInfo {
	return DefaultRouteRegistry.GetAllRoutes()
}
`
	
	err = os.WriteFile(filepath.Join(axonDir, "registry.go"), []byte(axonRegistryCode), 0644)
	if err != nil {
		t.Fatalf("failed to write axon registry: %v", err)
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

	// Verify the generated code contains route registry integration
	generatedContent := result.Content
	
	// Check that route registration calls are present
	expectedRegistrationElements := []string{
		"axon.DefaultRouteRegistry.RegisterRoute",
		"MiddlewareInstances: []axon.MiddlewareInstance{",
		`Method:              "GET"`,
		`Path:                "/users/{id:int}"`,
		`EchoPath:            "/users/:id"`,
		`HandlerName:         "GetUser"`,
		`ControllerName:      "UserController"`,
		`PackageName:         "testmodule"`,
		`Middlewares:         []string{"Auth", "Logging"}`,
		`Name:     "Auth"`,
		`Handler:  auth.Handle`,
		`Instance: auth`,
	}

	for _, expected := range expectedRegistrationElements {
		if !strings.Contains(generatedContent, expected) {
			t.Errorf("generated code missing expected element: %s\n\nGenerated code:\n%s", expected, generatedContent)
		}
	}

	// Try to compile the generated code
	cmd = exec.Command("go", "build", "./...")
	cmd.Dir = moduleDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("generated code failed to compile: %v\nOutput: %s\nGenerated code:\n%s", err, output, result.Content)
	}

	// Create a test file to verify runtime behavior
	testCode := `package testmodule

import (
	"testing"
	"github.com/labstack/echo/v4"
	"github.com/toyz/axon/pkg/axon"
)

func TestRouteRegistryIntegration(t *testing.T) {
	// Create Echo instance and register routes
	e := echo.New()
	
	// Create controller and middleware instances
	userService := UserService{}
	controller := &UserController{UserService: userService}
	auth := &Auth{}
	logging := &Logging{}
	validation := &Validation{}
	
	// Register routes
	RegisterRoutes(e, controller, auth, logging, validation)
	
	// Verify routes were registered in the registry
	routes := axon.GetRoutes()
	if len(routes) != 2 {
		t.Errorf("expected 2 routes, got %d", len(routes))
	}
	
	// Check first route (GET /users/{id:int})
	var getUserRoute *axon.RouteInfo
	for _, route := range routes {
		if route.Method == "GET" && route.Path == "/users/{id:int}" {
			getUserRoute = &route
			break
		}
	}
	
	if getUserRoute == nil {
		t.Fatal("GET /users/{id:int} route not found in registry")
	}
	
	// Verify route metadata
	if getUserRoute.HandlerName != "GetUser" {
		t.Errorf("expected HandlerName 'GetUser', got '%s'", getUserRoute.HandlerName)
	}
	
	if getUserRoute.ControllerName != "UserController" {
		t.Errorf("expected ControllerName 'UserController', got '%s'", getUserRoute.ControllerName)
	}
	
	if getUserRoute.PackageName != "testmodule" {
		t.Errorf("expected PackageName 'testmodule', got '%s'", getUserRoute.PackageName)
	}
	
	if getUserRoute.EchoPath != "/users/:id" {
		t.Errorf("expected EchoPath '/users/:id', got '%s'", getUserRoute.EchoPath)
	}
	
	// Verify middlewares
	expectedMiddlewares := []string{"Auth", "Logging"}
	if len(getUserRoute.Middlewares) != len(expectedMiddlewares) {
		t.Errorf("expected %d middlewares, got %d", len(expectedMiddlewares), len(getUserRoute.Middlewares))
	}
	
	for i, expected := range expectedMiddlewares {
		if i >= len(getUserRoute.Middlewares) || getUserRoute.Middlewares[i] != expected {
			t.Errorf("expected middleware %d to be '%s', got '%s'", i, expected, getUserRoute.Middlewares[i])
		}
	}
	
	// Verify middleware instances
	if len(getUserRoute.MiddlewareInstances) != 2 {
		t.Errorf("expected 2 middleware instances, got %d", len(getUserRoute.MiddlewareInstances))
	}
	
	if getUserRoute.MiddlewareInstances[0].Name != "Auth" {
		t.Errorf("expected first middleware instance name 'Auth', got '%s'", getUserRoute.MiddlewareInstances[0].Name)
	}
	
	if getUserRoute.MiddlewareInstances[1].Name != "Logging" {
		t.Errorf("expected second middleware instance name 'Logging', got '%s'", getUserRoute.MiddlewareInstances[1].Name)
	}
	
	// Verify parameter types
	expectedParamTypes := map[string]string{"id": "int"}
	if len(getUserRoute.ParameterTypes) != len(expectedParamTypes) {
		t.Errorf("expected %d parameter types, got %d", len(expectedParamTypes), len(getUserRoute.ParameterTypes))
	}
	
	for name, expectedType := range expectedParamTypes {
		if actualType, exists := getUserRoute.ParameterTypes[name]; !exists {
			t.Errorf("parameter type '%s' not found", name)
		} else if actualType != expectedType {
			t.Errorf("expected parameter type '%s' to be '%s', got '%s'", name, expectedType, actualType)
		}
	}
	
	// Verify handler is not nil
	if getUserRoute.Handler == nil {
		t.Error("handler should not be nil")
	}
}
`

	err = os.WriteFile(filepath.Join(moduleDir, "registry_test.go"), []byte(testCode), 0644)
	if err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	// Run the runtime test
	cmd = exec.Command("go", "test", "-v", "./...")
	cmd.Dir = moduleDir
	output, err = cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("runtime test failed: %v\nOutput: %s", err, output)
	}

	// Verify the test output contains success
	if !strings.Contains(string(output), "PASS") {
		t.Errorf("runtime test did not pass. Output: %s", output)
	}
}

// TestRouteRegistryWithComplexMiddleware tests route registry with complex middleware scenarios
func TestRouteRegistryWithComplexMiddleware(t *testing.T) {
	// Create temporary directory for test
	tempDir, err := os.MkdirTemp("", "complex_middleware_registry_test")
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

	// Generate a controller module with complex middleware chains
	generator := NewGenerator()
	
	metadata := &models.PackageMetadata{
		PackageName: "testmodule",
		PackagePath: moduleDir,
		Controllers: []models.ControllerMetadata{
			{
				Name:       "APIController",
				StructName: "APIController",
				Routes: []models.RouteMetadata{
					{
						Method:      "GET",
						Path:        "/api/v1/users/{id:int}",
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
						Middlewares: []string{"CORS", "Auth", "RateLimit", "Logging", "Metrics"},
					},
					{
						Method:      "POST",
						Path:        "/api/v1/admin/users",
						HandlerName: "CreateAdminUser",
						Parameters: []models.Parameter{
							{
								Name:   "user",
								Type:   "AdminUser",
								Source: models.ParameterSourceBody,
							},
						},
						ReturnType: models.ReturnTypeInfo{
							Type: models.ReturnTypeResponseError,
						},
						Middlewares: []string{"CORS", "Auth", "AdminOnly", "Validation", "AuditLog"},
					},
				},
				Dependencies: []string{"UserService"},
			},
		},
	}

	// Create the controller and middleware files
	controllerCode := `package testmodule

import "github.com/labstack/echo/v4"

// AdminUser represents an admin user entity
type AdminUser struct {
	ID       int    ` + "`json:\"id\"`" + `
	Name     string ` + "`json:\"name\"`" + `
	IsAdmin  bool   ` + "`json:\"is_admin\"`" + `
}

// Response represents an HTTP response
type Response struct {
	StatusCode int
	Body       interface{}
}

// UserService represents the user service
type UserService struct{}

// Middleware definitions
type CORS struct{}
func (m *CORS) Handle(next echo.HandlerFunc) echo.HandlerFunc { return next }

type Auth struct{}
func (m *Auth) Handle(next echo.HandlerFunc) echo.HandlerFunc { return next }

type RateLimit struct{}
func (m *RateLimit) Handle(next echo.HandlerFunc) echo.HandlerFunc { return next }

type Logging struct{}
func (m *Logging) Handle(next echo.HandlerFunc) echo.HandlerFunc { return next }

type Metrics struct{}
func (m *Metrics) Handle(next echo.HandlerFunc) echo.HandlerFunc { return next }

type AdminOnly struct{}
func (m *AdminOnly) Handle(next echo.HandlerFunc) echo.HandlerFunc { return next }

type Validation struct{}
func (m *Validation) Handle(next echo.HandlerFunc) echo.HandlerFunc { return next }

type AuditLog struct{}
func (m *AuditLog) Handle(next echo.HandlerFunc) echo.HandlerFunc { return next }

// APIController handles API requests
type APIController struct {
	UserService UserService ` + "`fx:\"in\"`" + `
}

// GetUser retrieves a user by ID
func (c *APIController) GetUser(id int) (*AdminUser, error) {
	return &AdminUser{ID: id, Name: "Test User", IsAdmin: false}, nil
}

// CreateAdminUser creates a new admin user
func (c *APIController) CreateAdminUser(user AdminUser) (*Response, error) {
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

	// Create a minimal axon package
	axonDir := filepath.Join(moduleDir, "pkg", "axon")
	err = os.MkdirAll(axonDir, 0755)
	if err != nil {
		t.Fatalf("failed to create axon package dir: %v", err)
	}
	
	minimalAxonCode := `package axon

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
	
	err = os.WriteFile(filepath.Join(axonDir, "registry.go"), []byte(minimalAxonCode), 0644)
	if err != nil {
		t.Fatalf("failed to write axon registry: %v", err)
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

	// Verify the generated code contains complex middleware registration
	generatedContent := result.Content
	
	// Check that all middlewares are properly registered
	expectedMiddlewares := []string{"CORS", "Auth", "RateLimit", "Logging", "Metrics", "AdminOnly", "Validation", "AuditLog"}
	
	for _, middleware := range expectedMiddlewares {
		expectedInstance := fmt.Sprintf(`Name:     "%s"`, middleware)
		if !strings.Contains(generatedContent, expectedInstance) {
			t.Errorf("generated code missing middleware instance: %s\n\nGenerated code:\n%s", expectedInstance, generatedContent)
		}
		
		expectedHandler := fmt.Sprintf(`Handler:  %s.Handle`, strings.ToLower(middleware))
		if !strings.Contains(generatedContent, expectedHandler) {
			t.Errorf("generated code missing middleware handler: %s\n\nGenerated code:\n%s", expectedHandler, generatedContent)
		}
	}

	// Try to compile the generated code
	cmd = exec.Command("go", "build", "./...")
	cmd.Dir = moduleDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("generated code failed to compile: %v\nOutput: %s\nGenerated code:\n%s", err, output, result.Content)
	}
}