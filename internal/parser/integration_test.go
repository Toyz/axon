package parser

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/toyz/axon/internal/models"
)

func TestParser_ParseDirectory_Integration(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "axon_parser_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create test Go files
	testFiles := map[string]string{
		"controllers.go": `package testpkg

import "go.uber.org/fx"

//axon::controller
type UserController struct {
	fx.In
	UserService UserServiceInterface
}

//axon::route GET /users/{id:int}
func (c *UserController) GetUser(id int) (*User, error) {
	return c.UserService.GetUser(id)
}

//axon::route POST /users -Middleware=Auth,Logging
func (c *UserController) CreateUser(user User) (*User, error) {
	return c.UserService.CreateUser(user)
}`,

		"middleware.go": `package testpkg

import "github.com/labstack/echo/v4"

//axon::middleware Auth
type AuthMiddleware struct {
	fx.In
	TokenService TokenServiceInterface
}

func (m *AuthMiddleware) Handle(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		// Auth logic here
		return next(c)
	}
}

//axon::middleware Logging
type LoggingMiddleware struct {
	fx.In
	Logger LoggerInterface
}

func (m *LoggingMiddleware) Handle(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		// Logging logic here
		return next(c)
	}
}`,

		"services.go": `package testpkg

import "context"

//axon::core -Init
type DatabaseService struct {
	fx.In
	Config *Config
}

func (s *DatabaseService) Start(ctx context.Context) error {
	// Start database connection
	return nil
}

func (s *DatabaseService) Stop(ctx context.Context) error {
	// Close database connection
	return nil
}

//axon::core
//axon::interface
type UserService struct {
	fx.In
	Repository UserRepository
}

func (s *UserService) GetUser(id int) (*User, error) {
	return s.Repository.FindByID(id)
}

func (s *UserService) CreateUser(user User) (*User, error) {
	return s.Repository.Create(user)
}`,
	}

	// Write test files
	for filename, content := range testFiles {
		filePath := filepath.Join(tempDir, filename)
		err := os.WriteFile(filePath, []byte(content), 0644)
		if err != nil {
			t.Fatalf("failed to write test file %s: %v", filename, err)
		}
	}

	// Parse the directory
	parser := NewParser()
	metadata, err := parser.ParseDirectory(tempDir)
	if err != nil {
		t.Fatalf("failed to parse directory: %v", err)
	}

	// Verify package metadata
	if metadata.PackageName != "testpkg" {
		t.Errorf("expected package name 'testpkg', got '%s'", metadata.PackageName)
	}

	if metadata.PackagePath != tempDir {
		t.Errorf("expected package path '%s', got '%s'", tempDir, metadata.PackagePath)
	}

	// Verify controllers
	if len(metadata.Controllers) != 1 {
		t.Errorf("expected 1 controller, got %d", len(metadata.Controllers))
	} else {
		controller := metadata.Controllers[0]
		if controller.Name != "UserController" {
			t.Errorf("expected controller name 'UserController', got '%s'", controller.Name)
		}

		if len(controller.Routes) != 2 {
			t.Errorf("expected 2 routes, got %d", len(controller.Routes))
		} else {
			// Check first route
			route1 := controller.Routes[0]
			if route1.Method != "GET" {
				t.Errorf("expected first route method 'GET', got '%s'", route1.Method)
			}
			if route1.Path != "/users/{id:int}" {
				t.Errorf("expected first route path '/users/{id:int}', got '%s'", route1.Path)
			}
			
			// Check that path parameters were parsed correctly
			if len(route1.Parameters) != 1 {
				t.Errorf("expected 1 parameter for first route, got %d", len(route1.Parameters))
			} else {
				param := route1.Parameters[0]
				if param.Name != "id" {
					t.Errorf("expected parameter name 'id', got '%s'", param.Name)
				}
				if param.Type != "int" {
					t.Errorf("expected parameter type 'int', got '%s'", param.Type)
				}
				if param.Source != models.ParameterSourcePath {
					t.Errorf("expected parameter source 'path', got %v", param.Source)
				}
				if !param.Required {
					t.Errorf("expected parameter to be required")
				}
			}

			// Check second route
			route2 := controller.Routes[1]
			if route2.Method != "POST" {
				t.Errorf("expected second route method 'POST', got '%s'", route2.Method)
			}
			if route2.Path != "/users" {
				t.Errorf("expected second route path '/users', got '%s'", route2.Path)
			}
			
			// Check that no path parameters were parsed for this route
			var pathParams []models.Parameter
			for _, param := range route2.Parameters {
				if param.Source == models.ParameterSourcePath {
					pathParams = append(pathParams, param)
				}
			}
			if len(pathParams) != 0 {
				t.Errorf("expected 0 path parameters for second route, got %d", len(pathParams))
			}
			
			if len(route2.Middlewares) != 2 {
				t.Errorf("expected 2 middlewares, got %d", len(route2.Middlewares))
			} else {
				if route2.Middlewares[0] != "Auth" || route2.Middlewares[1] != "Logging" {
					t.Errorf("expected middlewares [Auth, Logging], got %v", route2.Middlewares)
				}
			}
		}
	}

	// Verify middlewares
	if len(metadata.Middlewares) != 2 {
		t.Errorf("expected 2 middlewares, got %d", len(metadata.Middlewares))
	} else {
		// Find Auth and Logging middlewares
		var authMiddleware, loggingMiddleware *models.MiddlewareMetadata
		for i := range metadata.Middlewares {
			if metadata.Middlewares[i].Name == "Auth" {
				authMiddleware = &metadata.Middlewares[i]
			} else if metadata.Middlewares[i].Name == "Logging" {
				loggingMiddleware = &metadata.Middlewares[i]
			}
		}
		
		if authMiddleware == nil {
			t.Errorf("expected to find Auth middleware")
		} else {
			if authMiddleware.StructName != "AuthMiddleware" {
				t.Errorf("expected Auth middleware struct name 'AuthMiddleware', got '%s'", authMiddleware.StructName)
			}
		}
		
		if loggingMiddleware == nil {
			t.Errorf("expected to find Logging middleware")
		} else {
			if loggingMiddleware.StructName != "LoggingMiddleware" {
				t.Errorf("expected Logging middleware struct name 'LoggingMiddleware', got '%s'", loggingMiddleware.StructName)
			}
		}
	}

	// Verify core services
	if len(metadata.CoreServices) != 2 {
		t.Errorf("expected 2 core services, got %d", len(metadata.CoreServices))
	} else {
		// Find DatabaseService
		var dbService *models.CoreServiceMetadata
		var userService *models.CoreServiceMetadata
		for i := range metadata.CoreServices {
			if metadata.CoreServices[i].Name == "DatabaseService" {
				dbService = &metadata.CoreServices[i]
			} else if metadata.CoreServices[i].Name == "UserService" {
				userService = &metadata.CoreServices[i]
			}
		}
		
		if dbService == nil {
			t.Errorf("expected to find DatabaseService")
		} else {
			if !dbService.HasLifecycle {
				t.Errorf("expected DatabaseService to have lifecycle")
			}
		}
		
		if userService == nil {
			t.Errorf("expected to find UserService")
		}
	}

	// Verify interfaces
	if len(metadata.Interfaces) != 1 {
		t.Errorf("expected 1 interface, got %d", len(metadata.Interfaces))
	} else {
		iface := metadata.Interfaces[0]
		if iface.Name != "UserServiceInterface" {
			t.Errorf("expected interface name 'UserServiceInterface', got '%s'", iface.Name)
		}
		if iface.StructName != "UserService" {
			t.Errorf("expected struct name 'UserService', got '%s'", iface.StructName)
		}
	}
}

func TestParser_ParseDirectory_ErrorCases(t *testing.T) {
	parser := NewParser()

	// Test non-existent directory
	_, err := parser.ParseDirectory("/non/existent/directory")
	if err == nil {
		t.Errorf("expected error for non-existent directory")
	}

	// Test directory with no Go files
	tempDir, err := os.MkdirTemp("", "axon_parser_empty_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a non-Go file
	filePath := filepath.Join(tempDir, "readme.txt")
	err = os.WriteFile(filePath, []byte("This is not a Go file"), 0644)
	if err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	_, err = parser.ParseDirectory(tempDir)
	if err == nil {
		t.Errorf("expected error for directory with no Go packages")
	}
}

func TestParser_PathParameterParsing_Integration(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "axon_parser_params_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a test Go file with complex path parameters
	testFile := `package testpkg

import "go.uber.org/fx"

//axon::controller
type APIController struct {
	fx.In
	Service APIService
}

//axon::route GET /users/{id:int}
func (c *APIController) GetUser(id int) (*User, error) {
	return c.Service.GetUser(id)
}

//axon::route GET /users/{id:int}/posts/{slug:string}
func (c *APIController) GetUserPost(id int, slug string) (*Post, error) {
	return c.Service.GetUserPost(id, slug)
}

//axon::route POST /categories/{name:string}/items
func (c *APIController) CreateItem(name string, item Item) (*Item, error) {
	return c.Service.CreateItem(name, item)
}

//axon::route GET /health
func (c *APIController) HealthCheck() (string, error) {
	return "OK", nil
}`

	err = os.WriteFile(filepath.Join(tempDir, "test.go"), []byte(testFile), 0644)
	if err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	// Parse the directory
	parser := NewParser()
	metadata, err := parser.ParseDirectory(tempDir)
	if err != nil {
		t.Fatalf("failed to parse directory: %v", err)
	}

	// Verify the results
	if len(metadata.Controllers) != 1 {
		t.Fatalf("expected 1 controller, got %d", len(metadata.Controllers))
	}

	controller := metadata.Controllers[0]
	if len(controller.Routes) != 4 {
		t.Fatalf("expected 4 routes, got %d", len(controller.Routes))
	}

	// Test cases for each route
	testCases := []struct {
		path               string
		expectedParamCount int
		expectedParams     []struct {
			name     string
			paramType string
		}
	}{
		{
			path:               "/users/{id:int}",
			expectedParamCount: 1,
			expectedParams: []struct {
				name     string
				paramType string
			}{
				{name: "id", paramType: "int"},
			},
		},
		{
			path:               "/users/{id:int}/posts/{slug:string}",
			expectedParamCount: 2,
			expectedParams: []struct {
				name     string
				paramType string
			}{
				{name: "id", paramType: "int"},
				{name: "slug", paramType: "string"},
			},
		},
		{
			path:               "/categories/{name:string}/items",
			expectedParamCount: 1,
			expectedParams: []struct {
				name     string
				paramType string
			}{
				{name: "name", paramType: "string"},
			},
		},
		{
			path:               "/health",
			expectedParamCount: 0,
			expectedParams:     []struct {
				name     string
				paramType string
			}{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.path, func(t *testing.T) {
			var route *models.RouteMetadata
			for i := range controller.Routes {
				if controller.Routes[i].Path == tc.path {
					route = &controller.Routes[i]
					break
				}
			}

			if route == nil {
				t.Fatalf("could not find route with path %s", tc.path)
			}

			// Filter for only path parameters since this test is specifically for path parameter parsing
			var pathParams []models.Parameter
			for _, param := range route.Parameters {
				if param.Source == models.ParameterSourcePath {
					pathParams = append(pathParams, param)
				}
			}

			if len(pathParams) != tc.expectedParamCount {
				t.Errorf("expected %d path parameters, got %d", tc.expectedParamCount, len(pathParams))
				return
			}

			for i, expectedParam := range tc.expectedParams {
				if i >= len(pathParams) {
					t.Errorf("missing path parameter at index %d", i)
					continue
				}

				actualParam := pathParams[i]

				if actualParam.Name != expectedParam.name {
					t.Errorf("parameter %d: expected name %s, got %s", i, expectedParam.name, actualParam.Name)
				}

				if actualParam.Type != expectedParam.paramType {
					t.Errorf("parameter %d: expected type %s, got %s", i, expectedParam.paramType, actualParam.Type)
				}

				if actualParam.Source != models.ParameterSourcePath {
					t.Errorf("parameter %d: expected source 'path', got %v", i, actualParam.Source)
				}

				if !actualParam.Required {
					t.Errorf("parameter %d: expected parameter to be required", i)
				}
			}
		})
	}
}