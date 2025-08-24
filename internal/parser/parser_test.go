package parser

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/toyz/axon/internal/models"
)

func TestParser_parseAnnotationComment(t *testing.T) {
	p := NewParser()

	tests := []struct {
		name        string
		comment     string
		target      string
		expected    models.Annotation
		expectError bool
	}{
		{
			name:    "controller annotation",
			comment: "//axon::controller",
			target:  "UserController",
			expected: models.Annotation{
				Type:         models.AnnotationTypeController,
				Target:       "UserController",
				Parameters:   map[string]string{},
				Flags:        []string{},
				Dependencies: []models.Dependency{},
			},
			expectError: false,
		},
		{
			name:    "route annotation with method and path",
			comment: "//axon::route GET /users/{id:int}",
			target:  "UserController.GetUser",
			expected: models.Annotation{
				Type:   models.AnnotationTypeRoute,
				Target: "UserController.GetUser",
				Parameters: map[string]string{
					"method": "GET",
					"path":   "/users/{id:int}",
				},
				Flags:        []string{},
				Dependencies: []models.Dependency{},
			},
			expectError: false,
		},
		{
			name:    "route annotation with middleware flag",
			comment: "//axon::route POST /users -Middleware=Auth,Logging",
			target:  "UserController.CreateUser",
			expected: models.Annotation{
				Type:   models.AnnotationTypeRoute,
				Target: "UserController.CreateUser",
				Parameters: map[string]string{
					"method":      "POST",
					"path":        "/users",
					"-Middleware": "Auth,Logging",
				},
				Flags:        []string{},
				Dependencies: []models.Dependency{},
			},
			expectError: false,
		},
		{
			name:    "route annotation with PassContext flag",
			comment: "//axon::route GET /health -PassContext",
			target:  "HealthController.Check",
			expected: models.Annotation{
				Type:   models.AnnotationTypeRoute,
				Target: "HealthController.Check",
				Parameters: map[string]string{
					"method": "GET",
					"path":   "/health",
				},
				Flags:        []string{"-PassContext"},
				Dependencies: []models.Dependency{},
			},
			expectError: false,
		},
		{
			name:    "middleware annotation",
			comment: "//axon::middleware AuthMiddleware",
			target:  "AuthMiddleware",
			expected: models.Annotation{
				Type:   models.AnnotationTypeMiddleware,
				Target: "AuthMiddleware",
				Parameters: map[string]string{
					"name": "AuthMiddleware",
				},
				Flags:        []string{},
				Dependencies: []models.Dependency{},
			},
			expectError: false,
		},
		{
			name:    "core service with Init flag",
			comment: "//axon::core -Init",
			target:  "DatabaseService",
			expected: models.Annotation{
				Type:         models.AnnotationTypeCore,
				Target:       "DatabaseService",
				Parameters:   map[string]string{},
				Flags:        []string{"-Init"},
				Dependencies: []models.Dependency{},
			},
			expectError: false,
		},
		{
			name:    "core service with Manual flag and module name",
			comment: "//axon::core -Manual=CustomModule",
			target:  "ConfigService",
			expected: models.Annotation{
				Type:   models.AnnotationTypeCore,
				Target: "ConfigService",
				Parameters: map[string]string{
					"-Manual": "CustomModule",
				},
				Flags:        []string{},
				Dependencies: []models.Dependency{},
			},
			expectError: false,
		},
		{
			name:    "core service with Manual flag only",
			comment: "//axon::core -Manual",
			target:  "ConfigService",
			expected: models.Annotation{
				Type:         models.AnnotationTypeCore,
				Target:       "ConfigService",
				Parameters:   map[string]string{},
				Flags:        []string{"-Manual"},
				Dependencies: []models.Dependency{},
			},
			expectError: false,
		},
		{
			name:    "interface annotation",
			comment: "//axon::interface",
			target:  "UserService",
			expected: models.Annotation{
				Type:         models.AnnotationTypeInterface,
				Target:       "UserService",
				Parameters:   map[string]string{},
				Flags:        []string{},
				Dependencies: []models.Dependency{},
			},
			expectError: false,
		},
		{
			name:        "non-axon comment",
			comment:     "// This is just a regular comment",
			target:      "SomeStruct",
			expectError: true,
		},
		{
			name:        "empty axon annotation",
			comment:     "//axon::",
			target:      "SomeStruct",
			expectError: true,
		},
		{
			name:        "unknown annotation type",
			comment:     "//axon::unknown",
			target:      "SomeStruct",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := p.parseAnnotationComment(tt.comment, tt.target, token.NoPos)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if result.Type != tt.expected.Type {
				t.Errorf("expected type %v, got %v", tt.expected.Type, result.Type)
			}

			if result.Target != tt.expected.Target {
				t.Errorf("expected target %s, got %s", tt.expected.Target, result.Target)
			}

			// Check parameters
			if len(result.Parameters) != len(tt.expected.Parameters) {
				t.Errorf("expected %d parameters, got %d", len(tt.expected.Parameters), len(result.Parameters))
			}

			for key, expectedValue := range tt.expected.Parameters {
				if actualValue, exists := result.Parameters[key]; !exists || actualValue != expectedValue {
					t.Errorf("expected parameter %s=%s, got %s=%s", key, expectedValue, key, actualValue)
				}
			}

			// Check flags
			if len(result.Flags) != len(tt.expected.Flags) {
				t.Errorf("expected %d flags, got %d", len(tt.expected.Flags), len(result.Flags))
			}

			for i, expectedFlag := range tt.expected.Flags {
				if i >= len(result.Flags) || result.Flags[i] != expectedFlag {
					t.Errorf("expected flag %s at position %d, got %s", expectedFlag, i, result.Flags[i])
				}
			}
		})
	}
}

func TestParser_parseAnnotationType(t *testing.T) {
	p := NewParser()

	tests := []struct {
		input       string
		expected    models.AnnotationType
		expectError bool
	}{
		{"controller", models.AnnotationTypeController, false},
		{"route", models.AnnotationTypeRoute, false},
		{"middleware", models.AnnotationTypeMiddleware, false},
		{"core", models.AnnotationTypeCore, false},
		{"interface", models.AnnotationTypeInterface, false},
		{"unknown", 0, true},
		{"", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result, err := p.parseAnnotationType(tt.input)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestParser_ExtractAnnotations(t *testing.T) {
	tests := []struct {
		name     string
		source   string
		expected []models.Annotation
	}{
		{
			name: "controller with routes",
			source: `package main

//axon::controller
type UserController struct {
	fx.In
	UserService UserServiceInterface
}

//axon::route GET /users/{id:int}
func (c *UserController) GetUser(id int) (*User, error) {
	return c.UserService.GetUser(id)
}

//axon::route POST /users -Middleware=Auth
func (c *UserController) CreateUser(user User) (*User, error) {
	return c.UserService.CreateUser(user)
}`,
			expected: []models.Annotation{
				{
					Type:         models.AnnotationTypeController,
					Target:       "UserController",
					Parameters:   map[string]string{},
					Flags:        []string{},
					Dependencies: []models.Dependency{{Name: "UserServiceInterface", Type: "UserServiceInterface"}},
				},
				{
					Type:   models.AnnotationTypeRoute,
					Target: "UserController.GetUser",
					Parameters: map[string]string{
						"method": "GET",
						"path":   "/users/{id:int}",
					},
					Flags:        []string{},
					Dependencies: []models.Dependency{},
				},
				{
					Type:   models.AnnotationTypeRoute,
					Target: "UserController.CreateUser",
					Parameters: map[string]string{
						"method":      "POST",
						"path":        "/users",
						"-Middleware": "Auth",
					},
					Flags:        []string{},
					Dependencies: []models.Dependency{},
				},
			},
		},
		{
			name: "middleware and core service",
			source: `package main

//axon::middleware AuthMiddleware
type AuthMiddleware struct {
	fx.In
	TokenService TokenServiceInterface
}

//axon::core -Init
type DatabaseService struct {
	fx.In
	Config *Config
}`,
			expected: []models.Annotation{
				{
					Type:   models.AnnotationTypeMiddleware,
					Target: "AuthMiddleware",
					Parameters: map[string]string{
						"name": "AuthMiddleware",
					},
					Flags:        []string{},
					Dependencies: []models.Dependency{{Name: "TokenServiceInterface", Type: "TokenServiceInterface"}},
				},
				{
					Type:         models.AnnotationTypeCore,
					Target:       "DatabaseService",
					Parameters:   map[string]string{},
					Flags:        []string{"-Init"},
					Dependencies: []models.Dependency{{Name: "*Config", Type: "*Config"}},
				},
			},
		},
		{
			name: "core service with interface annotation",
			source: `package main

//axon::core
//axon::interface
type UserService struct {
	fx.In
	Repository UserRepository
}`,
			expected: []models.Annotation{
				{
					Type:         models.AnnotationTypeCore,
					Target:       "UserService",
					Parameters:   map[string]string{},
					Flags:        []string{},
					Dependencies: []models.Dependency{{Name: "UserRepository", Type: "UserRepository"}},
				},
				{
					Type:         models.AnnotationTypeInterface,
					Target:       "UserService",
					Parameters:   map[string]string{},
					Flags:        []string{},
					Dependencies: []models.Dependency{{Name: "UserRepository", Type: "UserRepository"}},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Parse the source code
			fset := token.NewFileSet()
			file, err := parser.ParseFile(fset, "test.go", tt.source, parser.ParseComments)
			if err != nil {
				t.Fatalf("failed to parse source: %v", err)
			}

			p := NewParser()
			annotations, err := p.ExtractAnnotations(file, "test.go")
			if err != nil {
				t.Fatalf("failed to extract annotations: %v", err)
			}

			if len(annotations) != len(tt.expected) {
				t.Errorf("expected %d annotations, got %d", len(tt.expected), len(annotations))
				return
			}

			for i, expected := range tt.expected {
				if i >= len(annotations) {
					t.Errorf("missing annotation at index %d", i)
					continue
				}

				actual := annotations[i]

				if actual.Type != expected.Type {
					t.Errorf("annotation %d: expected type %v, got %v", i, expected.Type, actual.Type)
				}

				if actual.Target != expected.Target {
					t.Errorf("annotation %d: expected target %s, got %s", i, expected.Target, actual.Target)
				}

				// Check parameters
				for key, expectedValue := range expected.Parameters {
					if actualValue, exists := actual.Parameters[key]; !exists || actualValue != expectedValue {
						t.Errorf("annotation %d: expected parameter %s=%s, got %s=%s", i, key, expectedValue, key, actualValue)
					}
				}

				// Check flags
				if len(actual.Flags) != len(expected.Flags) {
					t.Errorf("annotation %d: expected %d flags, got %d", i, len(expected.Flags), len(actual.Flags))
				}

				for j, expectedFlag := range expected.Flags {
					if j >= len(actual.Flags) || actual.Flags[j] != expectedFlag {
						t.Errorf("annotation %d: expected flag %s at position %d, got %s", i, expectedFlag, j, actual.Flags[j])
					}
				}
			}
		})
	}
}

func TestParser_processAnnotations(t *testing.T) {
	p := NewParser()

	annotations := []models.Annotation{
		{
			Type:         models.AnnotationTypeController,
			Target:       "UserController",
			Parameters:   map[string]string{},
			Flags:        []string{},
			Dependencies: []models.Dependency{{Name: "UserServiceInterface", Type: "UserServiceInterface"}},
		},
		{
			Type:   models.AnnotationTypeRoute,
			Target: "UserController.GetUser",
			Parameters: map[string]string{
				"method": "GET",
				"path":   "/users/{id:int}",
			},
			Flags:        []string{},
			Dependencies: []models.Dependency{},
		},
		{
			Type:   models.AnnotationTypeMiddleware,
			Target: "AuthMiddleware",
			Parameters: map[string]string{
				"name": "AuthMiddleware",
			},
			Flags:        []string{},
			Dependencies: []models.Dependency{{Name: "TokenServiceInterface", Type: "TokenServiceInterface"}},
		},
		{
			Type:         models.AnnotationTypeCore,
			Target:       "DatabaseService",
			Parameters:   map[string]string{},
			Flags:        []string{"-Init"},
			Dependencies: []models.Dependency{{Name: "*Config", Type: "*Config"}},
		},
		{
			Type:         models.AnnotationTypeCore,
			Target:       "UserService",
			Parameters:   map[string]string{},
			Flags:        []string{},
			Dependencies: []models.Dependency{{Name: "UserRepository", Type: "UserRepository"}},
		},
		{
			Type:         models.AnnotationTypeInterface,
			Target:       "UserService",
			Parameters:   map[string]string{},
			Flags:        []string{},
			Dependencies: []models.Dependency{{Name: "UserRepository", Type: "UserRepository"}},
		},
	}

	metadata := &models.PackageMetadata{
		PackageName: "test",
		PackagePath: "/test",
	}

	err := p.processAnnotations(annotations, metadata, map[string]*ast.File{})
	if err != nil {
		t.Fatalf("failed to process annotations: %v", err)
	}

	// Check controllers
	if len(metadata.Controllers) != 1 {
		t.Errorf("expected 1 controller, got %d", len(metadata.Controllers))
	} else {
		controller := metadata.Controllers[0]
		if controller.Name != "UserController" {
			t.Errorf("expected controller name UserController, got %s", controller.Name)
		}
		if len(controller.Dependencies) != 1 || controller.Dependencies[0].Name != "UserServiceInterface" {
			t.Errorf("expected controller dependencies [UserServiceInterface], got %v", controller.Dependencies)
		}
		if len(controller.Routes) != 1 {
			t.Errorf("expected 1 route, got %d", len(controller.Routes))
		} else {
			route := controller.Routes[0]
			if route.Method != "GET" {
				t.Errorf("expected route method GET, got %s", route.Method)
			}
			if route.Path != "/users/{id:int}" {
				t.Errorf("expected route path /users/{id:int}, got %s", route.Path)
			}
		}
	}

	// Check middlewares
	if len(metadata.Middlewares) != 1 {
		t.Errorf("expected 1 middleware, got %d", len(metadata.Middlewares))
	} else {
		middleware := metadata.Middlewares[0]
		if middleware.Name != "AuthMiddleware" {
			t.Errorf("expected middleware name AuthMiddleware, got %s", middleware.Name)
		}
	}

	// Check core services
	if len(metadata.CoreServices) != 2 {
		t.Errorf("expected 2 core services, got %d", len(metadata.CoreServices))
	} else {
		// Find DatabaseService and UserService
		var dbService, userService *models.CoreServiceMetadata
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

	// Check interfaces
	if len(metadata.Interfaces) != 1 {
		t.Errorf("expected 1 interface, got %d", len(metadata.Interfaces))
	} else {
		iface := metadata.Interfaces[0]
		if iface.Name != "UserServiceInterface" {
			t.Errorf("expected interface name UserServiceInterface, got %s", iface.Name)
		}
		if iface.StructName != "UserService" {
			t.Errorf("expected struct name UserService, got %s", iface.StructName)
		}
	}
}

// Test invalid syntax scenarios
func TestParser_InvalidSyntax(t *testing.T) {
	p := NewParser()

	invalidComments := []string{
		"//axon::",                    // empty annotation
		"//axon::unknown",             // unknown type
		"//axon::route",               // route without method/path
		"//axon::middleware",          // middleware without name
	}

	for _, comment := range invalidComments {
		t.Run(comment, func(t *testing.T) {
			_, err := p.parseAnnotationComment(comment, "TestTarget", token.NoPos)
			if err == nil {
				t.Errorf("expected error for invalid comment: %s", comment)
			}
		})
	}
}

func TestParser_extractDependencies(t *testing.T) {
	tests := []struct {
		name     string
		source   string
		expected []string
	}{
		{
			name: "struct with fx.In and dependencies",
			source: `type UserController struct {
				fx.In
				UserService UserServiceInterface
				Logger      *Logger
			}`,
			expected: []string{"UserServiceInterface", "*Logger"},
		},
		{
			name: "struct with no fx.In",
			source: `type UserController struct {
				UserService UserServiceInterface
				Logger      *Logger
			}`,
			expected: []string{},
		},
		{
			name: "struct with embedded fx.In only",
			source: `type UserController struct {
				fx.In
			}`,
			expected: []string{},
		},
		{
			name: "struct with multiple dependencies",
			source: `type UserController struct {
				fx.In
				UserService    UserServiceInterface
				AuthService    AuthServiceInterface
				Config         *Config
				Database       DatabaseInterface
			}`,
			expected: []string{"UserServiceInterface", "AuthServiceInterface", "*Config", "DatabaseInterface"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Parse the source code
			fset := token.NewFileSet()
			file, err := parser.ParseFile(fset, "test.go", "package main\n"+tt.source, parser.ParseComments)
			if err != nil {
				t.Fatalf("failed to parse source: %v", err)
			}

			// Find the struct type
			var structType *ast.StructType
			ast.Inspect(file, func(n ast.Node) bool {
				if genDecl, ok := n.(*ast.GenDecl); ok && genDecl.Tok == token.TYPE {
					for _, spec := range genDecl.Specs {
						if typeSpec, ok := spec.(*ast.TypeSpec); ok {
							if st, ok := typeSpec.Type.(*ast.StructType); ok {
								structType = st
								return false
							}
						}
					}
				}
				return true
			})

			if structType == nil {
				t.Fatal("could not find struct type in source")
			}

			p := NewParser()
			dependencies := p.extractDependencies(structType)

			if len(dependencies) != len(tt.expected) {
				t.Errorf("expected %d dependencies, got %d: %v", len(tt.expected), len(dependencies), dependencies)
				return
			}

			for i, expected := range tt.expected {
				if i >= len(dependencies) || dependencies[i].Type != expected {
					t.Errorf("expected dependency %s at position %d, got %s", expected, i, dependencies[i].Type)
				}
			}
		})
	}
}

func TestParser_parsePathParameters(t *testing.T) {
	p := NewParser()

	tests := []struct {
		name        string
		path        string
		expected    []models.Parameter
		expectError bool
		errorMsg    string
	}{
		{
			name: "single int parameter - axon syntax",
			path: "/users/{id:int}",
			expected: []models.Parameter{
				{
					Name:     "id",
					Type:     "int",
					Source:   models.ParameterSourcePath,
					Required: true,
				},
			},
			expectError: false,
		},
		{
			name: "single string parameter - axon syntax",
			path: "/users/{name:string}",
			expected: []models.Parameter{
				{
					Name:     "name",
					Type:     "string",
					Source:   models.ParameterSourcePath,
					Required: true,
				},
			},
			expectError: false,
		},
		{
			name: "single parameter - echo syntax",
			path: "/users/:id",
			expected: []models.Parameter{
				{
					Name:     "id",
					Type:     "string",
					Source:   models.ParameterSourcePath,
					Required: true,
				},
			},
			expectError: false,
		},
		{
			name: "multiple parameters - mixed syntax",
			path: "/users/{id:int}/posts/:slug",
			expected: []models.Parameter{
				{
					Name:     "id",
					Type:     "int",
					Source:   models.ParameterSourcePath,
					Required: true,
				},
				{
					Name:     "slug",
					Type:     "string",
					Source:   models.ParameterSourcePath,
					Required: true,
				},
			},
			expectError: false,
		},
		{
			name: "multiple parameters - echo syntax",
			path: "/users/:id/posts/:slug",
			expected: []models.Parameter{
				{
					Name:     "id",
					Type:     "string",
					Source:   models.ParameterSourcePath,
					Required: true,
				},
				{
					Name:     "slug",
					Type:     "string",
					Source:   models.ParameterSourcePath,
					Required: true,
				},
			},
			expectError: false,
		},
		{
			name: "multiple parameters - axon syntax",
			path: "/users/{id:int}/posts/{slug:string}",
			expected: []models.Parameter{
				{
					Name:     "id",
					Type:     "int",
					Source:   models.ParameterSourcePath,
					Required: true,
				},
				{
					Name:     "slug",
					Type:     "string",
					Source:   models.ParameterSourcePath,
					Required: true,
				},
			},
			expectError: false,
		},
		{
			name:        "no parameters",
			path:        "/users",
			expected:    []models.Parameter{},
			expectError: false,
		},
		{
			name:        "unclosed parameter bracket",
			path:        "/users/{id:int",
			expectError: true,
			errorMsg:    "unclosed parameter bracket",
		},
		{
			name:        "invalid parameter format - no colon",
			path:        "/users/{id}",
			expectError: true,
			errorMsg:    "parameter must be in format 'name:type'",
		},
		{
			name:        "invalid parameter format - empty name axon",
			path:        "/users/{:int}",
			expectError: true,
			errorMsg:    "parameter name cannot be empty",
		},
		{
			name:        "invalid parameter format - empty name echo",
			path:        "/users/:",
			expectError: true,
			errorMsg:    "parameter name cannot be empty",
		},
		{
			name:        "invalid parameter type name",
			path:        "/users/{id:123invalid}",
			expectError: true,
			errorMsg:    "invalid parameter type '123invalid'",
		},
		{
			name:        "parameter with extra colons",
			path:        "/users/{id:int:extra}",
			expectError: true,
			errorMsg:    "parameter must be in format 'name:type'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := p.parsePathParameters(tt.path)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
					return
				}
				if tt.errorMsg != "" && !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("expected error message to contain '%s', got '%s'", tt.errorMsg, err.Error())
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if len(result) != len(tt.expected) {
				t.Errorf("expected %d parameters, got %d", len(tt.expected), len(result))
				return
			}

			for i, expected := range tt.expected {
				if i >= len(result) {
					t.Errorf("missing parameter at index %d", i)
					continue
				}

				actual := result[i]

				if actual.Name != expected.Name {
					t.Errorf("parameter %d: expected name %s, got %s", i, expected.Name, actual.Name)
				}

				if actual.Type != expected.Type {
					t.Errorf("parameter %d: expected type %s, got %s", i, expected.Type, actual.Type)
				}

				if actual.Source != expected.Source {
					t.Errorf("parameter %d: expected source %v, got %v", i, expected.Source, actual.Source)
				}

				if actual.Required != expected.Required {
					t.Errorf("parameter %d: expected required %v, got %v", i, expected.Required, actual.Required)
				}
			}
		})
	}
}

func TestParser_parseParameterDefinition(t *testing.T) {
	p := NewParser()

	tests := []struct {
		name         string
		paramDef     string
		isEchoSyntax bool
		expected     models.Parameter
		expectError  bool
		errorMsg     string
	}{
		{
			name:         "valid int parameter - axon syntax",
			paramDef:     "id:int",
			isEchoSyntax: false,
			expected: models.Parameter{
				Name:     "id",
				Type:     "int",
				Source:   models.ParameterSourcePath,
				Required: true,
			},
			expectError: false,
		},
		{
			name:         "valid string parameter - axon syntax",
			paramDef:     "name:string",
			isEchoSyntax: false,
			expected: models.Parameter{
				Name:     "name",
				Type:     "string",
				Source:   models.ParameterSourcePath,
				Required: true,
			},
			expectError: false,
		},
		{
			name:         "valid parameter - echo syntax",
			paramDef:     "id",
			isEchoSyntax: true,
			expected: models.Parameter{
				Name:     "id",
				Type:     "string",
				Source:   models.ParameterSourcePath,
				Required: true,
			},
			expectError: false,
		},
		{
			name:         "parameter with whitespace - axon syntax",
			paramDef:     " id : int ",
			isEchoSyntax: false,
			expected: models.Parameter{
				Name:     "id",
				Type:     "int",
				Source:   models.ParameterSourcePath,
				Required: true,
			},
			expectError: false,
		},
		{
			name:         "parameter with whitespace - echo syntax",
			paramDef:     " id ",
			isEchoSyntax: true,
			expected: models.Parameter{
				Name:     "id",
				Type:     "string",
				Source:   models.ParameterSourcePath,
				Required: true,
			},
			expectError: false,
		},
		{
			name:         "missing colon - axon syntax",
			paramDef:     "id",
			isEchoSyntax: false,
			expectError:  true,
			errorMsg:     "parameter must be in format 'name:type'",
		},
		{
			name:         "empty name - axon syntax",
			paramDef:     ":int",
			isEchoSyntax: false,
			expectError:  true,
			errorMsg:     "parameter name cannot be empty",
		},
		{
			name:         "empty name - echo syntax",
			paramDef:     "",
			isEchoSyntax: true,
			expectError:  true,
			errorMsg:     "parameter name cannot be empty",
		},
		{
			name:         "invalid type name - axon syntax",
			paramDef:     "id:123invalid",
			isEchoSyntax: false,
			expectError:  true,
			errorMsg:     "invalid parameter type '123invalid'",
		},
		{
			name:         "multiple colons - axon syntax",
			paramDef:     "id:int:extra",
			isEchoSyntax: false,
			expectError:  true,
			errorMsg:     "parameter must be in format 'name:type'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := p.parseParameterDefinition(tt.paramDef, tt.isEchoSyntax)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
					return
				}
				if tt.errorMsg != "" && !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("expected error message to contain '%s', got '%s'", tt.errorMsg, err.Error())
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if result.Name != tt.expected.Name {
				t.Errorf("expected name %s, got %s", tt.expected.Name, result.Name)
			}

			if result.Type != tt.expected.Type {
				t.Errorf("expected type %s, got %s", tt.expected.Type, result.Type)
			}

			if result.Source != tt.expected.Source {
				t.Errorf("expected source %v, got %v", tt.expected.Source, result.Source)
			}

			if result.Required != tt.expected.Required {
				t.Errorf("expected required %v, got %v", tt.expected.Required, result.Required)
			}
		})
	}
}

func TestParser_analyzeHandlerSignature(t *testing.T) {
	tests := []struct {
		name           string
		source         string
		controllerName string
		methodName     string
		expected       []models.Parameter
		expectError    bool
	}{
		{
			name: "handler with echo.Context parameter",
			source: `package test
import "github.com/labstack/echo/v4"

type UserController struct{}

func (uc *UserController) GetUser(c echo.Context, id int) (interface{}, error) {
	return nil, nil
}`,
			controllerName: "UserController",
			methodName:     "GetUser",
			expected: []models.Parameter{
				{
					Name:     "c",
					Type:     "echo.Context",
					Source:   models.ParameterSourceContext,
					Required: true,
					Position: 0,
				},
			},
			expectError: false,
		},
		{
			name: "handler with echo.Context in middle position",
			source: `package test
import "github.com/labstack/echo/v4"

type UserController struct{}

func (uc *UserController) UpdateUser(id int, ctx echo.Context, data string) error {
	return nil
}`,
			controllerName: "UserController",
			methodName:     "UpdateUser",
			expected: []models.Parameter{
				{
					Name:     "ctx",
					Type:     "echo.Context",
					Source:   models.ParameterSourceContext,
					Required: true,
					Position: 1,
				},
			},
			expectError: false,
		},
		{
			name: "handler without echo.Context",
			source: `package test

type UserController struct{}

func (uc *UserController) GetUser(id int) (interface{}, error) {
	return nil, nil
}`,
			controllerName: "UserController",
			methodName:     "GetUser",
			expected:       []models.Parameter{},
			expectError:    false,
		},
		{
			name: "handler with multiple echo.Context parameters",
			source: `package test
import "github.com/labstack/echo/v4"

type UserController struct{}

func (uc *UserController) ComplexHandler(c1 echo.Context, id int, c2 echo.Context) error {
	return nil
}`,
			controllerName: "UserController",
			methodName:     "ComplexHandler",
			expected: []models.Parameter{
				{
					Name:     "c1",
					Type:     "echo.Context",
					Source:   models.ParameterSourceContext,
					Required: true,
					Position: 0,
				},
				{
					Name:     "c2",
					Type:     "echo.Context",
					Source:   models.ParameterSourceContext,
					Required: true,
					Position: 2,
				},
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewParser()
			
			// Parse the source code
			file, err := parser.ParseFile(p.fileSet, "test.go", tt.source, parser.ParseComments)
			if err != nil {
				t.Fatalf("failed to parse source: %v", err)
			}

			result, err := p.analyzeHandlerSignature(file, tt.controllerName, tt.methodName)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if len(result) != len(tt.expected) {
				t.Errorf("expected %d context parameters, got %d", len(tt.expected), len(result))
				return
			}

			for i, expected := range tt.expected {
				if i >= len(result) {
					t.Errorf("missing parameter at index %d", i)
					continue
				}

				actual := result[i]

				if actual.Name != expected.Name {
					t.Errorf("parameter %d: expected name %s, got %s", i, expected.Name, actual.Name)
				}

				if actual.Type != expected.Type {
					t.Errorf("parameter %d: expected type %s, got %s", i, expected.Type, actual.Type)
				}

				if actual.Source != expected.Source {
					t.Errorf("parameter %d: expected source %v, got %v", i, expected.Source, actual.Source)
				}

				if actual.Position != expected.Position {
					t.Errorf("parameter %d: expected position %d, got %d", i, expected.Position, actual.Position)
				}
			}
		})
	}
}

func TestParser_validateParameterType(t *testing.T) {
	p := NewParser()

	tests := []struct {
		name        string
		typeStr     string
		expected    string
		expectError bool
	}{
		{
			name:        "valid int type",
			typeStr:     "int",
			expected:    "int",
			expectError: false,
		},
		{
			name:        "valid string type",
			typeStr:     "string",
			expected:    "string",
			expectError: false,
		},
		{
			name:        "valid custom type",
			typeStr:     "CustomType",
			expectError: false,
			expected:    "CustomType",
		},
		{
			name:        "valid qualified type",
			typeStr:     "uuid.UUID",
			expectError: false,
			expected:    "uuid.UUID",
		},
		{
			name:        "invalid type name starting with number",
			typeStr:     "123Invalid",
			expectError: true,
		},
		{
			name:        "empty type",
			typeStr:     "",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := p.validateParameterType(tt.typeStr)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if result != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestParser_RouteValidation(t *testing.T) {
	tests := []struct {
		name        string
		source      string
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid route on controller",
			source: `package main

//axon::controller
type UserController struct {
	fx.In
	UserService UserServiceInterface
}

//axon::route GET /users/{id:int}
func (c *UserController) GetUser(id int) (*User, error) {
	return c.UserService.GetUser(id)
}`,
			expectError: false,
		},
		{
			name: "route on non-controller struct",
			source: `package main

type UserService struct {
	fx.In
	Repository UserRepository
}

//axon::route GET /users/{id:int}
func (s *UserService) GetUser(id int) (*User, error) {
	return s.Repository.GetUser(id)
}`,
			expectError: true,
			errorMsg:    "route UserService.GetUser is defined on struct UserService which is not annotated with //axon::controller",
		},
		{
			name: "multiple controllers with routes",
			source: `package main

//axon::controller
type UserController struct {
	fx.In
	UserService UserServiceInterface
}

//axon::controller
type AuthController struct {
	fx.In
	AuthService AuthServiceInterface
}

//axon::route GET /users/{id:int}
func (c *UserController) GetUser(id int) (*User, error) {
	return c.UserService.GetUser(id)
}

//axon::route POST /auth/login
func (c *AuthController) Login(req LoginRequest) (*Token, error) {
	return c.AuthService.Login(req)
}`,
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Parse the source code
			fset := token.NewFileSet()
			file, err := parser.ParseFile(fset, "test.go", tt.source, parser.ParseComments)
			if err != nil {
				t.Fatalf("failed to parse source: %v", err)
			}

			p := NewParser()
			annotations, err := p.ExtractAnnotations(file, "test.go")
			if err != nil {
				t.Fatalf("failed to extract annotations: %v", err)
			}

			metadata := &models.PackageMetadata{
				PackageName: "test",
				PackagePath: "/test",
			}

			err = p.processAnnotations(annotations, metadata, map[string]*ast.File{})

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
					return
				}
				if !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("expected error message to contain '%s', got '%s'", tt.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

func TestParser_StandaloneInterfaceAnnotation(t *testing.T) {
	// Test that standalone interface annotations don't create components
	p := NewParser()

	annotations := []models.Annotation{
		{
			Type:         models.AnnotationTypeInterface,
			Target:       "UserService",
			Parameters:   map[string]string{},
			Flags:        []string{},
			Dependencies: []models.Dependency{{Name: "UserRepository", Type: "UserRepository"}},
		},
	}

	metadata := &models.PackageMetadata{
		PackageName: "test",
		PackagePath: "/test",
	}

	err := p.processAnnotations(annotations, metadata, map[string]*ast.File{})
	if err != nil {
		t.Fatalf("failed to process annotations: %v", err)
	}

	// Verify that no components were created from standalone interface annotation
	if len(metadata.Controllers) != 0 {
		t.Errorf("expected 0 controllers, got %d", len(metadata.Controllers))
	}
	if len(metadata.CoreServices) != 0 {
		t.Errorf("expected 0 core services, got %d", len(metadata.CoreServices))
	}
	if len(metadata.Interfaces) != 0 {
		t.Errorf("expected 0 interfaces, got %d", len(metadata.Interfaces))
	}
	if len(metadata.Middlewares) != 0 {
		t.Errorf("expected 0 middlewares, got %d", len(metadata.Middlewares))
	}
}

func TestParser_ControllerWithInterface(t *testing.T) {
	// Test that controller + interface annotations work together
	p := NewParser()

	annotations := []models.Annotation{
		{
			Type:         models.AnnotationTypeController,
			Target:       "UserController",
			Parameters:   map[string]string{},
			Flags:        []string{},
			Dependencies: []models.Dependency{{Name: "UserService", Type: "UserService"}},
		},
		{
			Type:         models.AnnotationTypeInterface,
			Target:       "UserController",
			Parameters:   map[string]string{},
			Flags:        []string{},
			Dependencies: []models.Dependency{{Name: "UserService", Type: "UserService"}},
		},
	}

	metadata := &models.PackageMetadata{
		PackageName: "test",
		PackagePath: "/test",
	}

	err := p.processAnnotations(annotations, metadata, map[string]*ast.File{})
	if err != nil {
		t.Fatalf("failed to process annotations: %v", err)
	}

	// Verify that both controller and interface were created
	if len(metadata.Controllers) != 1 {
		t.Errorf("expected 1 controller, got %d", len(metadata.Controllers))
	} else {
		controller := metadata.Controllers[0]
		if controller.Name != "UserController" {
			t.Errorf("expected controller name UserController, got %s", controller.Name)
		}
	}

	if len(metadata.Interfaces) != 1 {
		t.Errorf("expected 1 interface, got %d", len(metadata.Interfaces))
	} else {
		iface := metadata.Interfaces[0]
		if iface.Name != "UserControllerInterface" {
			t.Errorf("expected interface name UserControllerInterface, got %s", iface.Name)
		}
		if iface.StructName != "UserController" {
			t.Errorf("expected struct name UserController, got %s", iface.StructName)
		}
	}
}

func TestParser_extractPublicMethods(t *testing.T) {
	tests := []struct {
		name       string
		source     string
		structName string
		expected   []models.Method
	}{
		{
			name: "struct with public methods",
			source: `package main

type UserService struct {
	repository UserRepository
}

func (s *UserService) GetUser(id int) (*User, error) {
	return s.repository.FindByID(id)
}

func (s *UserService) CreateUser(user User) (*User, error) {
	return s.repository.Create(user)
}

func (s *UserService) privateMethod() string {
	return "private"
}`,
			structName: "UserService",
			expected: []models.Method{
				{
					Name: "GetUser",
					Parameters: []models.Parameter{
						{Name: "id", Type: "int"},
					},
					Returns: []string{"*User", "error"},
				},
				{
					Name: "CreateUser",
					Parameters: []models.Parameter{
						{Name: "user", Type: "User"},
					},
					Returns: []string{"*User", "error"},
				},
			},
		},
		{
			name: "struct with no public methods",
			source: `package main

type UserService struct {
	repository UserRepository
}

func (s *UserService) privateMethod() string {
	return "private"
}`,
			structName: "UserService",
			expected:   []models.Method{},
		},
		{
			name: "struct with complex method signatures",
			source: `package main

type UserService struct {
	repository UserRepository
}

func (s *UserService) ProcessUsers(ctx context.Context, users []User, callback func(User) error) (map[string]interface{}, error) {
	return nil, nil
}

func (s *UserService) GetChannel() <-chan User {
	return nil
}`,
			structName: "UserService",
			expected: []models.Method{
				{
					Name: "ProcessUsers",
					Parameters: []models.Parameter{
						{Name: "ctx", Type: "context.Context"},
						{Name: "users", Type: "[]User"},
						{Name: "callback", Type: "func(User) error"},
					},
					Returns: []string{"map[string]interface{}", "error"},
				},
				{
					Name:       "GetChannel",
					Parameters: []models.Parameter{},
					Returns:    []string{"<-chan User"},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Parse the source code
			fset := token.NewFileSet()
			file, err := parser.ParseFile(fset, "test.go", tt.source, parser.ParseComments)
			if err != nil {
				t.Fatalf("failed to parse source: %v", err)
			}

			p := NewParser()
			methods, err := p.extractPublicMethods(file, tt.structName)
			if err != nil {
				t.Fatalf("failed to extract methods: %v", err)
			}

			if len(methods) != len(tt.expected) {
				t.Errorf("expected %d methods, got %d", len(tt.expected), len(methods))
				return
			}

			for i, expected := range tt.expected {
				if i >= len(methods) {
					t.Errorf("missing method at index %d", i)
					continue
				}

				actual := methods[i]

				if actual.Name != expected.Name {
					t.Errorf("method %d: expected name %s, got %s", i, expected.Name, actual.Name)
				}

				// Check parameters
				if len(actual.Parameters) != len(expected.Parameters) {
					t.Errorf("method %d: expected %d parameters, got %d", i, len(expected.Parameters), len(actual.Parameters))
				} else {
					for j, expectedParam := range expected.Parameters {
						if j >= len(actual.Parameters) {
							t.Errorf("method %d: missing parameter at index %d", i, j)
							continue
						}

						actualParam := actual.Parameters[j]
						if actualParam.Name != expectedParam.Name {
							t.Errorf("method %d, param %d: expected name %s, got %s", i, j, expectedParam.Name, actualParam.Name)
						}
						if actualParam.Type != expectedParam.Type {
							t.Errorf("method %d, param %d: expected type %s, got %s", i, j, expectedParam.Type, actualParam.Type)
						}
					}
				}

				// Check return types
				if len(actual.Returns) != len(expected.Returns) {
					t.Errorf("method %d: expected %d return types, got %d", i, len(expected.Returns), len(actual.Returns))
				} else {
					for j, expectedReturn := range expected.Returns {
						if j >= len(actual.Returns) {
							t.Errorf("method %d: missing return type at index %d", i, j)
							continue
						}

						if actual.Returns[j] != expectedReturn {
							t.Errorf("method %d, return %d: expected %s, got %s", i, j, expectedReturn, actual.Returns[j])
						}
					}
				}
			}
		})
	}
}

func TestParser_getTypeString(t *testing.T) {
	tests := []struct {
		name     string
		source   string
		expected string
	}{
		{
			name:     "simple type",
			source:   "int",
			expected: "int",
		},
		{
			name:     "pointer type",
			source:   "*User",
			expected: "*User",
		},
		{
			name:     "qualified type",
			source:   "context.Context",
			expected: "context.Context",
		},
		{
			name:     "slice type",
			source:   "[]User",
			expected: "[]User",
		},
		{
			name:     "map type",
			source:   "map[string]interface{}",
			expected: "map[string]interface{}",
		},
		{
			name:     "channel type",
			source:   "<-chan User",
			expected: "<-chan User",
		},
		{
			name:     "function type",
			source:   "func(User) error",
			expected: "func(User) error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Parse the type expression
			expr, err := parser.ParseExpr(tt.source)
			if err != nil {
				t.Fatalf("failed to parse expression: %v", err)
			}

			p := NewParser()
			result := p.getTypeString(expr)

			if result != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestParser_InterfaceGenerationIntegration(t *testing.T) {
	// Test complete interface generation from source code
	source := `package main

//axon::core
//axon::interface
type UserService struct {
	fx.In
	Repository UserRepository
	Logger     *Logger
}

func (s *UserService) GetUser(id int) (*User, error) {
	return s.Repository.FindByID(id)
}

func (s *UserService) CreateUser(user User) (*User, error) {
	return s.Repository.Create(user)
}

func (s *UserService) ListUsers(ctx context.Context, limit int) ([]User, error) {
	return s.Repository.List(ctx, limit)
}

func (s *UserService) privateMethod() string {
	return "private"
}`

	p := NewParser()
	metadata, err := p.ParseSource("test.go", source)
	if err != nil {
		t.Fatalf("failed to parse source: %v", err)
	}

	// Verify core service was created
	if len(metadata.CoreServices) != 1 {
		t.Errorf("expected 1 core service, got %d", len(metadata.CoreServices))
	} else {
		service := metadata.CoreServices[0]
		if service.Name != "UserService" {
			t.Errorf("expected service name UserService, got %s", service.Name)
		}
	}

	// Verify interface was created
	if len(metadata.Interfaces) != 1 {
		t.Errorf("expected 1 interface, got %d", len(metadata.Interfaces))
	} else {
		iface := metadata.Interfaces[0]
		if iface.Name != "UserServiceInterface" {
			t.Errorf("expected interface name UserServiceInterface, got %s", iface.Name)
		}
		if iface.StructName != "UserService" {
			t.Errorf("expected struct name UserService, got %s", iface.StructName)
		}

		// Verify methods were extracted (should only include public methods)
		expectedMethods := []string{"GetUser", "CreateUser", "ListUsers"}
		if len(iface.Methods) != len(expectedMethods) {
			t.Errorf("expected %d methods, got %d", len(expectedMethods), len(iface.Methods))
		} else {
			for i, expectedName := range expectedMethods {
				if iface.Methods[i].Name != expectedName {
					t.Errorf("method %d: expected name %s, got %s", i, expectedName, iface.Methods[i].Name)
				}
			}
		}

		// Verify GetUser method signature
		if len(iface.Methods) > 0 {
			getUserMethod := iface.Methods[0]
			if len(getUserMethod.Parameters) != 1 {
				t.Errorf("GetUser: expected 1 parameter, got %d", len(getUserMethod.Parameters))
			} else {
				param := getUserMethod.Parameters[0]
				if param.Name != "id" || param.Type != "int" {
					t.Errorf("GetUser: expected parameter (id int), got (%s %s)", param.Name, param.Type)
				}
			}

			if len(getUserMethod.Returns) != 2 {
				t.Errorf("GetUser: expected 2 return types, got %d", len(getUserMethod.Returns))
			} else {
				if getUserMethod.Returns[0] != "*User" || getUserMethod.Returns[1] != "error" {
					t.Errorf("GetUser: expected returns (*User, error), got (%s, %s)", getUserMethod.Returns[0], getUserMethod.Returns[1])
				}
			}
		}
	}
}

func TestParser_MiddlewareValidation(t *testing.T) {
	tests := []struct {
		name        string
		annotations []models.Annotation
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid middleware reference",
			annotations: []models.Annotation{
				{
					Type:   models.AnnotationTypeMiddleware,
					Target: "AuthMiddleware",
					Parameters: map[string]string{
						"name": "Auth",
					},
					Dependencies: []models.Dependency{},
				},
				{
					Type:         models.AnnotationTypeController,
					Target:       "UserController",
					Parameters:   map[string]string{},
					Dependencies: []models.Dependency{},
				},
				{
					Type:   models.AnnotationTypeRoute,
					Target: "UserController.GetUser",
					Parameters: map[string]string{
						"method":      "GET",
						"path":        "/users/{id:int}",
						"-Middleware": "Auth",
					},
					Dependencies: []models.Dependency{},
				},
			},
			expectError: false,
		},
		{
			name: "invalid middleware reference",
			annotations: []models.Annotation{
				{
					Type:         models.AnnotationTypeController,
					Target:       "UserController",
					Parameters:   map[string]string{},
					Dependencies: []models.Dependency{},
				},
				{
					Type:   models.AnnotationTypeRoute,
					Target: "UserController.GetUser",
					Parameters: map[string]string{
						"method":      "GET",
						"path":        "/users/{id:int}",
						"-Middleware": "NonExistentMiddleware",
					},
					Dependencies: []models.Dependency{},
				},
			},
			expectError: true,
			errorMsg:    "unknown middleware(s): NonExistentMiddleware",
		},
		{
			name: "multiple valid middlewares",
			annotations: []models.Annotation{
				{
					Type:   models.AnnotationTypeMiddleware,
					Target: "AuthMiddleware",
					Parameters: map[string]string{
						"name": "Auth",
					},
					Dependencies: []models.Dependency{},
				},
				{
					Type:   models.AnnotationTypeMiddleware,
					Target: "LoggingMiddleware",
					Parameters: map[string]string{
						"name": "Logging",
					},
					Dependencies: []models.Dependency{},
				},
				{
					Type:         models.AnnotationTypeController,
					Target:       "UserController",
					Parameters:   map[string]string{},
					Dependencies: []models.Dependency{},
				},
				{
					Type:   models.AnnotationTypeRoute,
					Target: "UserController.CreateUser",
					Parameters: map[string]string{
						"method":      "POST",
						"path":        "/users",
						"-Middleware": "Auth,Logging",
					},
					Dependencies: []models.Dependency{},
				},
			},
			expectError: false,
		},
		{
			name: "mixed valid and invalid middlewares",
			annotations: []models.Annotation{
				{
					Type:   models.AnnotationTypeMiddleware,
					Target: "AuthMiddleware",
					Parameters: map[string]string{
						"name": "Auth",
					},
					Dependencies: []models.Dependency{},
				},
				{
					Type:         models.AnnotationTypeController,
					Target:       "UserController",
					Parameters:   map[string]string{},
					Dependencies: []models.Dependency{},
				},
				{
					Type:   models.AnnotationTypeRoute,
					Target: "UserController.CreateUser",
					Parameters: map[string]string{
						"method":      "POST",
						"path":        "/users",
						"-Middleware": "Auth,InvalidMiddleware",
					},
					Dependencies: []models.Dependency{},
				},
			},
			expectError: true,
			errorMsg:    "unknown middleware(s): InvalidMiddleware",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewParser()
			metadata := &models.PackageMetadata{
				PackageName: "test",
				PackagePath: "/test",
			}

			err := p.processAnnotations(tt.annotations, metadata, map[string]*ast.File{})

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
					return
				}
				if !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("expected error message to contain '%s', got '%s'", tt.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

func TestParser_MiddlewareOrdering(t *testing.T) {
	// Test that middleware ordering is preserved in route metadata
	p := NewParser()

	annotations := []models.Annotation{
		{
			Type:   models.AnnotationTypeMiddleware,
			Target: "AuthMiddleware",
			Parameters: map[string]string{
				"name": "Auth",
			},
			Dependencies: []models.Dependency{},
		},
		{
			Type:   models.AnnotationTypeMiddleware,
			Target: "LoggingMiddleware",
			Parameters: map[string]string{
				"name": "Logging",
			},
			Dependencies: []models.Dependency{},
		},
		{
			Type:   models.AnnotationTypeMiddleware,
			Target: "RateLimitMiddleware",
			Parameters: map[string]string{
				"name": "RateLimit",
			},
			Dependencies: []models.Dependency{},
		},
		{
			Type:         models.AnnotationTypeController,
			Target:       "UserController",
			Parameters:   map[string]string{},
			Dependencies: []models.Dependency{},
		},
		{
			Type:   models.AnnotationTypeRoute,
			Target: "UserController.CreateUser",
			Parameters: map[string]string{
				"method":      "POST",
				"path":        "/users",
				"-Middleware": "Auth,Logging,RateLimit",
			},
			Dependencies: []models.Dependency{},
		},
	}

	metadata := &models.PackageMetadata{
		PackageName: "test",
		PackagePath: "/test",
	}

	err := p.processAnnotations(annotations, metadata, map[string]*ast.File{})
	if err != nil {
		t.Fatalf("failed to process annotations: %v", err)
	}

	// Verify that the route has the correct middleware order
	if len(metadata.Controllers) != 1 {
		t.Fatalf("expected 1 controller, got %d", len(metadata.Controllers))
	}

	controller := metadata.Controllers[0]
	if len(controller.Routes) != 1 {
		t.Fatalf("expected 1 route, got %d", len(controller.Routes))
	}

	route := controller.Routes[0]
	expectedMiddlewares := []string{"Auth", "Logging", "RateLimit"}
	
	if len(route.Middlewares) != len(expectedMiddlewares) {
		t.Errorf("expected %d middlewares, got %d", len(expectedMiddlewares), len(route.Middlewares))
	}

	for i, expected := range expectedMiddlewares {
		if i >= len(route.Middlewares) || route.Middlewares[i] != expected {
			t.Errorf("expected middleware %s at position %d, got %s", expected, i, route.Middlewares[i])
		}
	}
}

func TestParser_MiddlewareAnnotationProcessing(t *testing.T) {
	tests := []struct {
		name        string
		source      string
		expectError bool
		errorMsg    string
		expected    []models.MiddlewareMetadata
	}{
		{
			name: "valid middleware annotation",
			source: `package main

//axon::middleware AuthMiddleware
type AuthMiddleware struct {
	fx.In
	TokenService TokenServiceInterface
}

func (m *AuthMiddleware) Handle(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		// Auth logic here
		return next(c)
	}
}`,
			expectError: false,
			expected: []models.MiddlewareMetadata{
				{
					Name:         "AuthMiddleware",
					PackagePath:  "/test",
					StructName:   "AuthMiddleware",
					Dependencies: []models.Dependency{{Name: "TokenServiceInterface", Type: "TokenServiceInterface"}},
				},
			},
		},
		{
			name: "multiple middleware annotations",
			source: `package main

//axon::middleware AuthMiddleware
type AuthMiddleware struct {
	fx.In
	TokenService TokenServiceInterface
}

func (m *AuthMiddleware) Handle(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		return next(c)
	}
}

//axon::middleware LoggingMiddleware
type LoggingMiddleware struct {
	fx.In
	Logger LoggerInterface
}

func (m *LoggingMiddleware) Handle(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		return next(c)
	}
}`,
			expectError: false,
			expected: []models.MiddlewareMetadata{
				{
					Name:         "AuthMiddleware",
					PackagePath:  "/test",
					StructName:   "AuthMiddleware",
					Dependencies: []models.Dependency{{Name: "TokenServiceInterface", Type: "TokenServiceInterface"}},
				},
				{
					Name:         "LoggingMiddleware",
					PackagePath:  "/test",
					StructName:   "LoggingMiddleware",
					Dependencies: []models.Dependency{{Name: "LoggerInterface", Type: "LoggerInterface"}},
				},
			},
		},
		{
			name: "middleware without Handle method",
			source: `package main

//axon::middleware AuthMiddleware
type AuthMiddleware struct {
	fx.In
	TokenService TokenServiceInterface
}`,
			expectError: true,
			errorMsg:    "middleware 'AuthMiddleware' must have a Handle method",
		},
		{
			name: "middleware with incorrect Handle method signature - wrong parameter",
			source: `package main

//axon::middleware AuthMiddleware
type AuthMiddleware struct {
	fx.In
	TokenService TokenServiceInterface
}

func (m *AuthMiddleware) Handle(next string) echo.HandlerFunc {
	return func(c echo.Context) error {
		return nil
	}
}`,
			expectError: true,
			errorMsg:    "middleware 'AuthMiddleware' Handle method parameter must be echo.HandlerFunc",
		},
		{
			name: "middleware with incorrect Handle method signature - wrong return type",
			source: `package main

//axon::middleware AuthMiddleware
type AuthMiddleware struct {
	fx.In
	TokenService TokenServiceInterface
}

func (m *AuthMiddleware) Handle(next echo.HandlerFunc) string {
	return "invalid"
}`,
			expectError: true,
			errorMsg:    "middleware 'AuthMiddleware' Handle method must return echo.HandlerFunc",
		},
		{
			name: "middleware with too many parameters",
			source: `package main

//axon::middleware AuthMiddleware
type AuthMiddleware struct {
	fx.In
	TokenService TokenServiceInterface
}

func (m *AuthMiddleware) Handle(next echo.HandlerFunc, extra string) echo.HandlerFunc {
	return func(c echo.Context) error {
		return nil
	}
}`,
			expectError: true,
			errorMsg:    "middleware 'AuthMiddleware' Handle method must have exactly one parameter",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Parse the source code
			fset := token.NewFileSet()
			file, err := parser.ParseFile(fset, "test.go", tt.source, parser.ParseComments)
			if err != nil {
				t.Fatalf("failed to parse source: %v", err)
			}

			p := NewParser()
			annotations, err := p.ExtractAnnotations(file, "test.go")
			if err != nil {
				t.Fatalf("failed to extract annotations: %v", err)
			}

			metadata := &models.PackageMetadata{
				PackageName: "test",
				PackagePath: "/test",
			}

			err = p.processAnnotations(annotations, metadata, map[string]*ast.File{})
			if err != nil && !tt.expectError {
				t.Fatalf("failed to process annotations: %v", err)
			}

			// Validate middleware Handle methods
			for _, annotation := range annotations {
				if annotation.Type == models.AnnotationTypeMiddleware {
					err = p.ValidateMiddlewareHandleMethod(file, annotation.Target)
					if err != nil {
						break
					}
				}
			}

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
					return
				}
				if tt.errorMsg != "" && !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("expected error message to contain '%s', got '%s'", tt.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
					return
				}

				// Verify middleware metadata
				if len(metadata.Middlewares) != len(tt.expected) {
					t.Errorf("expected %d middlewares, got %d", len(tt.expected), len(metadata.Middlewares))
					return
				}

				for i, expected := range tt.expected {
					if i >= len(metadata.Middlewares) {
						t.Errorf("missing middleware at index %d", i)
						continue
					}

					actual := metadata.Middlewares[i]

					if actual.Name != expected.Name {
						t.Errorf("middleware %d: expected name %s, got %s", i, expected.Name, actual.Name)
					}

					if actual.PackagePath != expected.PackagePath {
						t.Errorf("middleware %d: expected package path %s, got %s", i, expected.PackagePath, actual.PackagePath)
					}

					if actual.StructName != expected.StructName {
						t.Errorf("middleware %d: expected struct name %s, got %s", i, expected.StructName, actual.StructName)
					}

					if len(actual.Dependencies) != len(expected.Dependencies) {
						t.Errorf("middleware %d: expected %d dependencies, got %d", i, len(expected.Dependencies), len(actual.Dependencies))
					}

					for j, expectedDep := range expected.Dependencies {
						if j >= len(actual.Dependencies) || actual.Dependencies[j].Type != expectedDep.Type {
							t.Errorf("middleware %d: expected dependency %s at position %d, got %s", i, expectedDep.Type, j, actual.Dependencies[j].Type)
						}
					}
				}
			}
		})
	}
}

func TestParser_MiddlewareValidationInRoutes(t *testing.T) {
	tests := []struct {
		name        string
		source      string
		expectError bool
		errorMsg    string
	}{
		{
			name: "route with valid middleware reference",
			source: `package main

//axon::middleware AuthMiddleware
type AuthMiddleware struct {
	fx.In
	TokenService TokenServiceInterface
}

func (m *AuthMiddleware) Handle(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		return next(c)
	}
}

//axon::controller
type UserController struct {
	fx.In
	UserService UserServiceInterface
}

//axon::route GET /users/{id:int} -Middleware=AuthMiddleware
func (c *UserController) GetUser(id int) (*User, error) {
	return c.UserService.GetUser(id)
}`,
			expectError: false,
		},
		{
			name: "route with multiple valid middleware references",
			source: `package main

//axon::middleware AuthMiddleware
type AuthMiddleware struct {
	fx.In
	TokenService TokenServiceInterface
}

func (m *AuthMiddleware) Handle(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		return next(c)
	}
}

//axon::middleware LoggingMiddleware
type LoggingMiddleware struct {
	fx.In
	Logger LoggerInterface
}

func (m *LoggingMiddleware) Handle(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		return next(c)
	}
}

//axon::controller
type UserController struct {
	fx.In
	UserService UserServiceInterface
}

//axon::route GET /users/{id:int} -Middleware=AuthMiddleware,LoggingMiddleware
func (c *UserController) GetUser(id int) (*User, error) {
	return c.UserService.GetUser(id)
}`,
			expectError: false,
		},
		{
			name: "route with invalid middleware reference",
			source: `package main

//axon::middleware AuthMiddleware
type AuthMiddleware struct {
	fx.In
	TokenService TokenServiceInterface
}

func (m *AuthMiddleware) Handle(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		return next(c)
	}
}

//axon::controller
type UserController struct {
	fx.In
	UserService UserServiceInterface
}

//axon::route GET /users/{id:int} -Middleware=NonExistentMiddleware
func (c *UserController) GetUser(id int) (*User, error) {
	return c.UserService.GetUser(id)
}`,
			expectError: true,
			errorMsg:    "route UserController.GetUser has invalid middleware reference: unknown middleware(s): NonExistentMiddleware",
		},
		{
			name: "route with mix of valid and invalid middleware references",
			source: `package main

//axon::middleware AuthMiddleware
type AuthMiddleware struct {
	fx.In
	TokenService TokenServiceInterface
}

func (m *AuthMiddleware) Handle(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		return next(c)
	}
}

//axon::controller
type UserController struct {
	fx.In
	UserService UserServiceInterface
}

//axon::route GET /users/{id:int} -Middleware=AuthMiddleware,NonExistentMiddleware
func (c *UserController) GetUser(id int) (*User, error) {
	return c.UserService.GetUser(id)
}`,
			expectError: true,
			errorMsg:    "route UserController.GetUser has invalid middleware reference: unknown middleware(s): NonExistentMiddleware",
		},
		{
			name: "route with middleware names containing whitespace",
			source: `package main

//axon::middleware AuthMiddleware
type AuthMiddleware struct {
	fx.In
	TokenService TokenServiceInterface
}

func (m *AuthMiddleware) Handle(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		return next(c)
	}
}

//axon::middleware LoggingMiddleware
type LoggingMiddleware struct {
	fx.In
	Logger LoggerInterface
}

func (m *LoggingMiddleware) Handle(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		return next(c)
	}
}

//axon::controller
type UserController struct {
	fx.In
	UserService UserServiceInterface
}

//axon::route GET /users/{id:int} -Middleware= AuthMiddleware , LoggingMiddleware 
func (c *UserController) GetUser(id int) (*User, error) {
	return c.UserService.GetUser(id)
}`,
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Parse the source code
			fset := token.NewFileSet()
			file, err := parser.ParseFile(fset, "test.go", tt.source, parser.ParseComments)
			if err != nil {
				t.Fatalf("failed to parse source: %v", err)
			}

			p := NewParser()
			annotations, err := p.ExtractAnnotations(file, "test.go")
			if err != nil {
				t.Fatalf("failed to extract annotations: %v", err)
			}

			metadata := &models.PackageMetadata{
				PackageName: "test",
				PackagePath: "/test",
			}

			err = p.processAnnotations(annotations, metadata, map[string]*ast.File{})

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
					return
				}
				if tt.errorMsg != "" && !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("expected error message to contain '%s', got '%s'", tt.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

func TestParser_ValidateMiddlewareHandleMethod(t *testing.T) {
	tests := []struct {
		name        string
		source      string
		middleware  string
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid Handle method",
			source: `package main

type AuthMiddleware struct {
	fx.In
	TokenService TokenServiceInterface
}

func (m *AuthMiddleware) Handle(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		return next(c)
	}
}`,
			middleware:  "AuthMiddleware",
			expectError: false,
		},
		{
			name: "missing Handle method",
			source: `package main

type AuthMiddleware struct {
	fx.In
	TokenService TokenServiceInterface
}`,
			middleware:  "AuthMiddleware",
			expectError: true,
			errorMsg:    "middleware 'AuthMiddleware' must have a Handle method",
		},
		{
			name: "Handle method with wrong parameter type",
			source: `package main

type AuthMiddleware struct {
	fx.In
	TokenService TokenServiceInterface
}

func (m *AuthMiddleware) Handle(next string) echo.HandlerFunc {
	return nil
}`,
			middleware:  "AuthMiddleware",
			expectError: true,
			errorMsg:    "middleware 'AuthMiddleware' Handle method parameter must be echo.HandlerFunc",
		},
		{
			name: "Handle method with wrong return type",
			source: `package main

type AuthMiddleware struct {
	fx.In
	TokenService TokenServiceInterface
}

func (m *AuthMiddleware) Handle(next echo.HandlerFunc) string {
	return ""
}`,
			middleware:  "AuthMiddleware",
			expectError: true,
			errorMsg:    "middleware 'AuthMiddleware' Handle method must return echo.HandlerFunc",
		},
		{
			name: "Handle method with no parameters",
			source: `package main

type AuthMiddleware struct {
	fx.In
	TokenService TokenServiceInterface
}

func (m *AuthMiddleware) Handle() echo.HandlerFunc {
	return nil
}`,
			middleware:  "AuthMiddleware",
			expectError: true,
			errorMsg:    "middleware 'AuthMiddleware' Handle method must have exactly one parameter",
		},
		{
			name: "Handle method with too many parameters",
			source: `package main

type AuthMiddleware struct {
	fx.In
	TokenService TokenServiceInterface
}

func (m *AuthMiddleware) Handle(next echo.HandlerFunc, extra string) echo.HandlerFunc {
	return nil
}`,
			middleware:  "AuthMiddleware",
			expectError: true,
			errorMsg:    "middleware 'AuthMiddleware' Handle method must have exactly one parameter",
		},
		{
			name: "Handle method with no return value",
			source: `package main

type AuthMiddleware struct {
	fx.In
	TokenService TokenServiceInterface
}

func (m *AuthMiddleware) Handle(next echo.HandlerFunc) {
}`,
			middleware:  "AuthMiddleware",
			expectError: true,
			errorMsg:    "middleware 'AuthMiddleware' Handle method must return exactly one value",
		},
		{
			name: "Handle method with multiple return values",
			source: `package main

type AuthMiddleware struct {
	fx.In
	TokenService TokenServiceInterface
}

func (m *AuthMiddleware) Handle(next echo.HandlerFunc) (echo.HandlerFunc, error) {
	return nil, nil
}`,
			middleware:  "AuthMiddleware",
			expectError: true,
			errorMsg:    "middleware 'AuthMiddleware' Handle method must return exactly one value",
		},
		{
			name: "middleware struct not found",
			source: `package main

type SomeOtherStruct struct {
}`,
			middleware:  "AuthMiddleware",
			expectError: true,
			errorMsg:    "middleware struct 'AuthMiddleware' not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Parse the source code
			fset := token.NewFileSet()
			file, err := parser.ParseFile(fset, "test.go", tt.source, parser.ParseComments)
			if err != nil {
				t.Fatalf("failed to parse source: %v", err)
			}

			p := NewParser()
			err = p.ValidateMiddlewareHandleMethod(file, tt.middleware)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
					return
				}
				if tt.errorMsg != "" && !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("expected error message to contain '%s', got '%s'", tt.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

func TestParser_RouteWithPathParameters(t *testing.T) {
	tests := []struct {
		name     string
		source   string
		expected []models.Parameter
	}{
		{
			name: "route with single int parameter",
			source: `package main

//axon::controller
type UserController struct {
	fx.In
	UserService UserServiceInterface
}

//axon::route GET /users/{id:int}
func (c *UserController) GetUser(id int) (*User, error) {
	return c.UserService.GetUser(id)
}`,
			expected: []models.Parameter{
				{
					Name:     "id",
					Type:     "int",
					Source:   models.ParameterSourcePath,
					Required: true,
				},
			},
		},
		{
			name: "route with multiple parameters",
			source: `package main

//axon::controller
type UserController struct {
	fx.In
	UserService UserServiceInterface
}

//axon::route GET /users/{id:int}/posts/{slug:string}
func (c *UserController) GetUserPost(id int, slug string) (*Post, error) {
	return c.UserService.GetUserPost(id, slug)
}`,
			expected: []models.Parameter{
				{
					Name:     "id",
					Type:     "int",
					Source:   models.ParameterSourcePath,
					Required: true,
				},
				{
					Name:     "slug",
					Type:     "string",
					Source:   models.ParameterSourcePath,
					Required: true,
				},
			},
		},
		{
			name: "route with no parameters",
			source: `package main

//axon::controller
type UserController struct {
	fx.In
	UserService UserServiceInterface
}

//axon::route GET /users
func (c *UserController) GetUsers() ([]User, error) {
	return c.UserService.GetUsers()
}`,
			expected: []models.Parameter{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Parse the source code
			fset := token.NewFileSet()
			file, err := parser.ParseFile(fset, "test.go", tt.source, parser.ParseComments)
			if err != nil {
				t.Fatalf("failed to parse source: %v", err)
			}

			p := NewParser()
			annotations, err := p.ExtractAnnotations(file, "test.go")
			if err != nil {
				t.Fatalf("failed to extract annotations: %v", err)
			}

			metadata := &models.PackageMetadata{
				PackageName: "test",
				PackagePath: "/test",
			}

			err = p.processAnnotations(annotations, metadata, map[string]*ast.File{})
			if err != nil {
				t.Fatalf("failed to process annotations: %v", err)
			}

			// Check that the route has the expected parameters
			if len(metadata.Controllers) != 1 {
				t.Fatalf("expected 1 controller, got %d", len(metadata.Controllers))
			}

			controller := metadata.Controllers[0]
			if len(controller.Routes) != 1 {
				t.Fatalf("expected 1 route, got %d", len(controller.Routes))
			}

			route := controller.Routes[0]
			if len(route.Parameters) != len(tt.expected) {
				t.Errorf("expected %d parameters, got %d", len(tt.expected), len(route.Parameters))
				return
			}

			for i, expected := range tt.expected {
				if i >= len(route.Parameters) {
					t.Errorf("missing parameter at index %d", i)
					continue
				}

				actual := route.Parameters[i]

				if actual.Name != expected.Name {
					t.Errorf("parameter %d: expected name %s, got %s", i, expected.Name, actual.Name)
				}

				if actual.Type != expected.Type {
					t.Errorf("parameter %d: expected type %s, got %s", i, expected.Type, actual.Type)
				}

				if actual.Source != expected.Source {
					t.Errorf("parameter %d: expected source %v, got %v", i, expected.Source, actual.Source)
				}

				if actual.Required != expected.Required {
					t.Errorf("parameter %d: expected required %v, got %v", i, expected.Required, actual.Required)
				}
			}
		})
	}
}

// TestParser_CoreServiceAnnotationProcessing tests the complete processing of core service annotations
func TestParser_CoreServiceAnnotationProcessing(t *testing.T) {
	tests := []struct {
		name     string
		source   string
		expected []models.CoreServiceMetadata
	}{
		{
			name: "core service with Init flag",
			source: `package testpkg

import (
	"context"
	"go.uber.org/fx"
)

//axon::core -Init
type DatabaseService struct {
	fx.In
	Config *Config
}

func (s *DatabaseService) Start(ctx context.Context) error {
	return nil
}

func (s *DatabaseService) Stop(ctx context.Context) error {
	return nil
}`,
			expected: []models.CoreServiceMetadata{
				{
					Name:         "DatabaseService",
					StructName:   "DatabaseService",
					HasLifecycle: true,
					IsManual:     false,
					ModuleName:   "",
					Dependencies: []models.Dependency{{Name: "*Config", Type: "*Config"}},
				},
			},
		},
		{
			name: "core service with Manual flag and custom module name",
			source: `package testpkg

import "go.uber.org/fx"

//axon::core -Manual=CustomModule
type ConfigService struct {
	fx.In
	Logger LoggerInterface
}`,
			expected: []models.CoreServiceMetadata{
				{
					Name:         "ConfigService",
					StructName:   "ConfigService",
					HasLifecycle: false,
					IsManual:     true,
					ModuleName:   "CustomModule",
					Dependencies: []models.Dependency{{Name: "LoggerInterface", Type: "LoggerInterface"}},
				},
			},
		},
		{
			name: "core service with Manual flag only (default module name)",
			source: `package testpkg

import "go.uber.org/fx"

//axon::core -Manual
type ConfigService struct {
	fx.In
	Logger LoggerInterface
}`,
			expected: []models.CoreServiceMetadata{
				{
					Name:         "ConfigService",
					StructName:   "ConfigService",
					HasLifecycle: false,
					IsManual:     true,
					ModuleName:   "Module",
					Dependencies: []models.Dependency{{Name: "LoggerInterface", Type: "LoggerInterface"}},
				},
			},
		},
		{
			name: "core service without flags",
			source: `package testpkg

import "go.uber.org/fx"

//axon::core
type UserService struct {
	fx.In
	Repository UserRepository
}`,
			expected: []models.CoreServiceMetadata{
				{
					Name:         "UserService",
					StructName:   "UserService",
					HasLifecycle: false,
					IsManual:     false,
					ModuleName:   "",
					Dependencies: []models.Dependency{{Name: "UserRepository", Type: "UserRepository"}},
				},
			},
		},
		{
			name: "multiple core services with different flags",
			source: `package testpkg

import (
	"context"
	"go.uber.org/fx"
)

//axon::core -Init
type DatabaseService struct {
	fx.In
	Config *Config
}

func (s *DatabaseService) Start(ctx context.Context) error {
	return nil
}

//axon::core -Manual=CustomModule
type ConfigService struct {
	fx.In
	Logger LoggerInterface
}

//axon::core
type UserService struct {
	fx.In
	Repository UserRepository
}`,
			expected: []models.CoreServiceMetadata{
				{
					Name:         "DatabaseService",
					StructName:   "DatabaseService",
					HasLifecycle: true,
					IsManual:     false,
					ModuleName:   "",
					Dependencies: []models.Dependency{{Name: "*Config", Type: "*Config"}},
				},
				{
					Name:         "ConfigService",
					StructName:   "ConfigService",
					HasLifecycle: false,
					IsManual:     true,
					ModuleName:   "CustomModule",
					Dependencies: []models.Dependency{{Name: "LoggerInterface", Type: "LoggerInterface"}},
				},
				{
					Name:         "UserService",
					StructName:   "UserService",
					HasLifecycle: false,
					IsManual:     false,
					ModuleName:   "",
					Dependencies: []models.Dependency{{Name: "UserRepository", Type: "UserRepository"}},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary directory and file
			tempDir, err := os.MkdirTemp("", "axon_core_service_test")
			if err != nil {
				t.Fatalf("failed to create temp dir: %v", err)
			}
			defer os.RemoveAll(tempDir)

			// Write test file
			filePath := filepath.Join(tempDir, "services.go")
			err = os.WriteFile(filePath, []byte(tt.source), 0644)
			if err != nil {
				t.Fatalf("failed to write test file: %v", err)
			}

			// Parse the directory
			parser := NewParser()
			metadata, err := parser.ParseDirectory(tempDir)
			if err != nil {
				t.Fatalf("failed to parse directory: %v", err)
			}

			// Verify core services
			if len(metadata.CoreServices) != len(tt.expected) {
				t.Errorf("expected %d core services, got %d", len(tt.expected), len(metadata.CoreServices))
				return
			}

			// Check each expected service
			for i, expected := range tt.expected {
				if i >= len(metadata.CoreServices) {
					t.Errorf("missing core service at index %d", i)
					continue
				}

				actual := metadata.CoreServices[i]
				if actual.Name != expected.Name {
					t.Errorf("service %d: expected name %s, got %s", i, expected.Name, actual.Name)
				}
				if actual.StructName != expected.StructName {
					t.Errorf("service %d: expected struct name %s, got %s", i, expected.StructName, actual.StructName)
				}
				if actual.HasLifecycle != expected.HasLifecycle {
					t.Errorf("service %d: expected HasLifecycle %v, got %v", i, expected.HasLifecycle, actual.HasLifecycle)
				}
				if actual.IsManual != expected.IsManual {
					t.Errorf("service %d: expected IsManual %v, got %v", i, expected.IsManual, actual.IsManual)
				}
				if actual.ModuleName != expected.ModuleName {
					t.Errorf("service %d: expected ModuleName %s, got %s", i, expected.ModuleName, actual.ModuleName)
				}
				if len(actual.Dependencies) != len(expected.Dependencies) {
					t.Errorf("service %d: expected %d dependencies, got %d", i, len(expected.Dependencies), len(actual.Dependencies))
				} else {
					for j, dep := range expected.Dependencies {
						if actual.Dependencies[j].Type != dep.Type {
							t.Errorf("service %d dependency %d: expected %s, got %s", i, j, dep.Type, actual.Dependencies[j].Type)
						}
					}
				}
			}
		})
	}
}

// TestParser_DatabaseServiceDependencies tests dependency extraction for DatabaseService
func TestParser_DatabaseServiceDependencies(t *testing.T) {
	source := `package testpkg

import (
	"context"
	"go.uber.org/fx"
	"github.com/toyz/axon/examples/complete-app/internal/config"
)

//axon::core -Init
type DatabaseService struct {
	fx.In
	Config *config.Config
	connected bool
}

func (s *DatabaseService) Start(ctx context.Context) error {
	return nil
}`

	parser := NewParser()
	metadata, err := parser.ParseSource("test.go", source)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify core service
	if len(metadata.CoreServices) != 1 {
		t.Errorf("expected 1 core service, got %d", len(metadata.CoreServices))
		return
	}

	service := metadata.CoreServices[0]
	if service.Name != "DatabaseService" {
		t.Errorf("expected service name DatabaseService, got %s", service.Name)
	}

	// Check dependencies
	t.Logf("Found dependencies: %v", service.Dependencies)
	if len(service.Dependencies) != 1 {
		t.Errorf("expected 1 dependency, got %d: %v", len(service.Dependencies), service.Dependencies)
		return
	}

	if service.Dependencies[0].Name != "*config.Config" {
		t.Errorf("expected dependency '*config.Config', got '%s'", service.Dependencies[0].Name)
	}
}

// TestParser_CoreServiceWithInterface tests core services that also have interface annotations
func TestParser_CoreServiceWithInterface(t *testing.T) {
	source := `package testpkg

import "go.uber.org/fx"

//axon::core
//axon::interface
type UserService struct {
	fx.In
	Repository UserRepository
}

func (s *UserService) GetUser(id int) (*User, error) {
	return s.Repository.FindByID(id)
}`

	// Create temporary directory and file
	tempDir, err := os.MkdirTemp("", "axon_core_interface_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Write test file
	filePath := filepath.Join(tempDir, "services.go")
	err = os.WriteFile(filePath, []byte(source), 0644)
	if err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	// Parse the directory
	parser := NewParser()
	metadata, err := parser.ParseDirectory(tempDir)
	if err != nil {
		t.Fatalf("failed to parse directory: %v", err)
	}

	// Verify core service
	if len(metadata.CoreServices) != 1 {
		t.Errorf("expected 1 core service, got %d", len(metadata.CoreServices))
		return
	}

	service := metadata.CoreServices[0]
	if service.Name != "UserService" {
		t.Errorf("expected service name UserService, got %s", service.Name)
	}
	if service.HasLifecycle {
		t.Errorf("expected service to not have lifecycle")
	}
	if service.IsManual {
		t.Errorf("expected service to not be manual")
	}

	// Verify interface is also generated
	if len(metadata.Interfaces) != 1 {
		t.Errorf("expected 1 interface, got %d", len(metadata.Interfaces))
		return
	}

	iface := metadata.Interfaces[0]
	if iface.Name != "UserServiceInterface" {
		t.Errorf("expected interface name UserServiceInterface, got %s", iface.Name)
	}
	if iface.StructName != "UserService" {
		t.Errorf("expected struct name UserService, got %s", iface.StructName)
	}
}

// TestParser_LifecycleMethodDetection tests detection of Start and Stop methods
func TestParser_LifecycleMethodDetection(t *testing.T) {
	tests := []struct {
		name      string
		source    string
		hasStart  bool
		hasStop   bool
	}{
		{
			name: "service with both Start and Stop methods",
			source: `package testpkg

//axon::core -Init
type DatabaseService struct {
	fx.In
}

func (s *DatabaseService) Start(ctx context.Context) error {
	return nil
}

func (s *DatabaseService) Stop(ctx context.Context) error {
	return nil
}`,
			hasStart: true,
			hasStop:  true,
		},
		{
			name: "service with only Start method",
			source: `package testpkg

//axon::core -Init
type DatabaseService struct {
	fx.In
}

func (s *DatabaseService) Start(ctx context.Context) error {
	return nil
}`,
			hasStart: true,
			hasStop:  false,
		},
		{
			name: "service with no lifecycle methods",
			source: `package testpkg

//axon::core
type DatabaseService struct {
	fx.In
}`,
			hasStart: false,
			hasStop:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := NewParser()
			
			// Parse the source code
			metadata, err := parser.ParseSource("test.go", tt.source)
			
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// Verify core services
			if len(metadata.CoreServices) != 1 {
				t.Errorf("expected 1 core service, got %d", len(metadata.CoreServices))
				return
			}

			service := metadata.CoreServices[0]
			if service.HasStart != tt.hasStart {
				t.Errorf("expected HasStart %v, got %v", tt.hasStart, service.HasStart)
			}
			if service.HasStop != tt.hasStop {
				t.Errorf("expected HasStop %v, got %v", tt.hasStop, service.HasStop)
			}
		})
	}
}

// TestParser_LifecycleValidationErrors tests validation errors for lifecycle services
func TestParser_LifecycleValidationErrors(t *testing.T) {
	tests := []struct {
		name          string
		source        string
		expectedError string
	}{
		{
			name: "service with -Init flag but no Start method",
			source: `package testpkg

//axon::core -Init
type DatabaseService struct {
	fx.In
}`,
			expectedError: "failed to process annotations: service DatabaseService has -Init flag but missing Start(context.Context) error method",
		},
		{
			name: "service with -Init flag but wrong Start signature",
			source: `package testpkg

//axon::core -Init
type DatabaseService struct {
	fx.In
}

func (s *DatabaseService) Start() error {
	return nil
}`,
			expectedError: "failed to process annotations: service DatabaseService has -Init flag but missing Start(context.Context) error method",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := NewParser()
			
			// Parse the source code - should return error
			_, err := parser.ParseSource("test.go", tt.source)
			
			if err == nil {
				t.Error("expected error but got none")
				return
			}
			
			if err.Error() != tt.expectedError {
				t.Errorf("expected error %q, got %q", tt.expectedError, err.Error())
			}
		})
	}
}

// TestParser_CoreServiceValidation tests validation of core service annotations
func TestParser_RouteParserValidation(t *testing.T) {
	tests := []struct {
		name        string
		source      string
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid parser function",
			source: `package main

import (
	"github.com/labstack/echo/v4"
	"github.com/google/uuid"
)

//axon::route_parser UUID
func ParseUUID(c echo.Context, paramValue string) (uuid.UUID, error) {
	return uuid.Parse(paramValue)
}`,
			expectError: false,
		},
		{
			name: "parser with wrong number of parameters",
			source: `package main

import "github.com/labstack/echo/v4"

//axon::route_parser InvalidType
func InvalidParser(paramValue string) (string, error) {
	return paramValue, nil
}`,
			expectError: true,
			errorMsg:    "has 1 parameters, expected 2",
		},
		{
			name: "parser with wrong first parameter type",
			source: `package main

//axon::route_parser InvalidType
func InvalidParser(ctx string, paramValue string) (string, error) {
	return paramValue, nil
}`,
			expectError: true,
			errorMsg:    "first parameter is string, expected echo.Context",
		},
		{
			name: "parser with wrong second parameter type",
			source: `package main

import "github.com/labstack/echo/v4"

//axon::route_parser InvalidType
func InvalidParser(c echo.Context, paramValue int) (string, error) {
	return "", nil
}`,
			expectError: true,
			errorMsg:    "second parameter is int, expected string",
		},
		{
			name: "parser with wrong return type count",
			source: `package main

import "github.com/labstack/echo/v4"

//axon::route_parser InvalidType
func InvalidParser(c echo.Context, paramValue string) string {
	return paramValue
}`,
			expectError: true,
			errorMsg:    "returns 1 values, expected 2",
		},
		{
			name: "parser with wrong second return type",
			source: `package main

import "github.com/labstack/echo/v4"

//axon::route_parser InvalidType
func InvalidParser(c echo.Context, paramValue string) (string, string) {
	return paramValue, ""
}`,
			expectError: true,
			errorMsg:    "second return value is string, expected error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewParser()
			_, err := p.ParseSource("test.go", tt.source)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
				} else if !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("expected error containing '%s', got '%s'", tt.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

func TestParser_RouteParserMetadataExtraction(t *testing.T) {
	source := `package main

import (
	"github.com/labstack/echo/v4"
	"github.com/google/uuid"
)

//axon::route_parser UUID
func ParseUUID(c echo.Context, paramValue string) (uuid.UUID, error) {
	return uuid.Parse(paramValue)
}

//axon::route_parser CustomID
func ParseCustomID(c echo.Context, value string) (string, error) {
	return value, nil
}`

	p := NewParser()
	metadata, err := p.ParseSource("test.go", source)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(metadata.RouteParsers) != 2 {
		t.Errorf("expected 2 route parsers, got %d", len(metadata.RouteParsers))
	}

	// Check first parser
	parser1 := metadata.RouteParsers[0]
	if parser1.TypeName != "UUID" {
		t.Errorf("expected type name 'UUID', got '%s'", parser1.TypeName)
	}
	if parser1.FunctionName != "ParseUUID" {
		t.Errorf("expected function name 'ParseUUID', got '%s'", parser1.FunctionName)
	}
	if len(parser1.ParameterTypes) != 2 || parser1.ParameterTypes[0] != "echo.Context" || parser1.ParameterTypes[1] != "string" {
		t.Errorf("expected parameter types [echo.Context, string], got %v", parser1.ParameterTypes)
	}
	if len(parser1.ReturnTypes) != 2 || parser1.ReturnTypes[0] != "uuid.UUID" || parser1.ReturnTypes[1] != "error" {
		t.Errorf("expected return types [uuid.UUID, error], got %v", parser1.ReturnTypes)
	}

	// Check second parser
	parser2 := metadata.RouteParsers[1]
	if parser2.TypeName != "CustomID" {
		t.Errorf("expected type name 'CustomID', got '%s'", parser2.TypeName)
	}
	if parser2.FunctionName != "ParseCustomID" {
		t.Errorf("expected function name 'ParseCustomID', got '%s'", parser2.FunctionName)
	}
}

func TestParser_CoreServiceValidation(t *testing.T) {
	tests := []struct {
		name        string
		source      string
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid core service",
			source: `package testpkg

//axon::core
type UserService struct {}`,
			expectError: false,
		},
		{
			name: "core service with valid Init flag",
			source: `package testpkg

import "context"

//axon::core -Init
type DatabaseService struct {}

func (s *DatabaseService) Start(ctx context.Context) error {
	return nil
}`,
			expectError: false,
		},
		{
			name: "core service with valid Manual flag",
			source: `package testpkg

//axon::core -Manual
type ConfigService struct {}`,
			expectError: false,
		},
		{
			name: "core service with valid Manual flag and module name",
			source: `package testpkg

//axon::core -Manual=CustomModule
type ConfigService struct {}`,
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary directory and file
			tempDir, err := os.MkdirTemp("", "axon_core_validation_test")
			if err != nil {
				t.Fatalf("failed to create temp dir: %v", err)
			}
			defer os.RemoveAll(tempDir)

			// Write test file
			filePath := filepath.Join(tempDir, "services.go")
			err = os.WriteFile(filePath, []byte(tt.source), 0644)
			if err != nil {
				t.Fatalf("failed to write test file: %v", err)
			}

			// Parse the directory
			parser := NewParser()
			_, err = parser.ParseDirectory(tempDir)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
				} else if tt.errorMsg != "" && !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("expected error containing '%s', got '%s'", tt.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

func TestParser_RouteParserSignatureValidation_EdgeCases(t *testing.T) {
	tests := []struct {
		name        string
		source      string
		expectError bool
		errorMsg    string
	}{
		{
			name: "parser with pointer return type",
			source: `package main

import "github.com/labstack/echo/v4"

//axon::route_parser PointerType
func ParsePointer(c echo.Context, paramValue string) (*string, error) {
	return &paramValue, nil
}`,
			expectError: false,
		},
		{
			name: "parser with qualified return type",
			source: `package main

import (
	"github.com/labstack/echo/v4"
	"time"
)

//axon::route_parser TimeType
func ParseTime(c echo.Context, paramValue string) (time.Time, error) {
	return time.Parse("2006-01-02", paramValue)
}`,
			expectError: false,
		},
		{
			name: "parser with slice return type",
			source: `package main

import "github.com/labstack/echo/v4"

//axon::route_parser SliceType
func ParseSlice(c echo.Context, paramValue string) ([]string, error) {
	return strings.Split(paramValue, ","), nil
}`,
			expectError: false,
		},
		{
			name: "parser with three parameters",
			source: `package main

import "github.com/labstack/echo/v4"

//axon::route_parser ThreeParams
func ParseThreeParams(c echo.Context, paramValue string, extra int) (string, error) {
	return paramValue, nil
}`,
			expectError: true,
			errorMsg:    "has 3 parameters, expected 2",
		},
		{
			name: "parser with no parameters",
			source: `package main

//axon::route_parser NoParams
func ParseNoParams() (string, error) {
	return "", nil
}`,
			expectError: true,
			errorMsg:    "has 0 parameters, expected 2",
		},
		{
			name: "parser with three return values",
			source: `package main

import "github.com/labstack/echo/v4"

//axon::route_parser ThreeReturns
func ParseThreeReturns(c echo.Context, paramValue string) (string, error, bool) {
	return paramValue, nil, true
}`,
			expectError: true,
			errorMsg:    "returns 3 values, expected 2",
		},
		{
			name: "parser with no return values",
			source: `package main

import "github.com/labstack/echo/v4"

//axon::route_parser NoReturns
func ParseNoReturns(c echo.Context, paramValue string) {
}`,
			expectError: true,
			errorMsg:    "returns 0 values, expected 2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewParser()
			_, err := p.ParseSource("test.go", tt.source)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
				} else if !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("expected error containing '%s', got '%s'", tt.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

func TestParser_RouteParserSignatureExtraction_ComplexTypes(t *testing.T) {
	source := `package main

import (
	"github.com/labstack/echo/v4"
	"time"
)

//axon::route_parser PointerType
func ParsePointer(c echo.Context, paramValue string) (*string, error) {
	return &paramValue, nil
}

//axon::route_parser TimeType
func ParseTime(c echo.Context, paramValue string) (time.Time, error) {
	return time.Parse("2006-01-02", paramValue)
}

//axon::route_parser SliceType
func ParseSlice(c echo.Context, paramValue string) ([]string, error) {
	return []string{paramValue}, nil
}`

	p := NewParser()
	metadata, err := p.ParseSource("test.go", source)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(metadata.RouteParsers) != 3 {
		t.Errorf("expected 3 route parsers, got %d", len(metadata.RouteParsers))
	}

	// Check pointer type parser
	parser1 := metadata.RouteParsers[0]
	if parser1.TypeName != "PointerType" {
		t.Errorf("expected type name 'PointerType', got '%s'", parser1.TypeName)
	}
	if len(parser1.ReturnTypes) != 2 || parser1.ReturnTypes[0] != "*string" || parser1.ReturnTypes[1] != "error" {
		t.Errorf("expected return types [*string, error], got %v", parser1.ReturnTypes)
	}

	// Check qualified type parser
	parser2 := metadata.RouteParsers[1]
	if parser2.TypeName != "TimeType" {
		t.Errorf("expected type name 'TimeType', got '%s'", parser2.TypeName)
	}
	if len(parser2.ReturnTypes) != 2 || parser2.ReturnTypes[0] != "time.Time" || parser2.ReturnTypes[1] != "error" {
		t.Errorf("expected return types [time.Time, error], got %v", parser2.ReturnTypes)
	}

	// Check slice type parser
	parser3 := metadata.RouteParsers[2]
	if parser3.TypeName != "SliceType" {
		t.Errorf("expected type name 'SliceType', got '%s'", parser3.TypeName)
	}
	if len(parser3.ReturnTypes) != 2 || parser3.ReturnTypes[0] != "[]string" || parser3.ReturnTypes[1] != "error" {
		t.Errorf("expected return types [[]string, error], got %v", parser3.ReturnTypes)
	}
}

func TestParser_ExtractImports(t *testing.T) {
	p := NewParser()

	tests := []struct {
		name     string
		source   string
		expected []models.Import
	}{
		{
			name: "single import",
			source: `package main

import "context"

type Service struct{}`,
			expected: []models.Import{
				{Path: "context", Alias: ""},
			},
		},
		{
			name: "multiple imports",
			source: `package main

import (
	"context"
	"fmt"
	"net/http"
)

type Service struct{}`,
			expected: []models.Import{
				{Path: "context", Alias: ""},
				{Path: "fmt", Alias: ""},
				{Path: "net/http", Alias: ""},
			},
		},
		{
			name: "imports with aliases",
			source: `package main

import (
	"context"
	fx "go.uber.org/fx"
	. "github.com/onsi/ginkgo"
	_ "github.com/lib/pq"
)

type Service struct{}`,
			expected: []models.Import{
				{Path: "context", Alias: ""},
				{Path: "go.uber.org/fx", Alias: "fx"},
				{Path: "github.com/onsi/ginkgo", Alias: "."},
				{Path: "github.com/lib/pq", Alias: "_"},
			},
		},
		{
			name: "no imports",
			source: `package main

type Service struct{}`,
			expected: []models.Import{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Parse the source
			file, err := parser.ParseFile(token.NewFileSet(), "test.go", tt.source, parser.ParseComments)
			if err != nil {
				t.Fatalf("Failed to parse source: %v", err)
			}

			// Extract imports
			imports := p.ExtractImports(file)

			// Verify results
			if len(imports) != len(tt.expected) {
				t.Errorf("Expected %d imports, got %d", len(tt.expected), len(imports))
				return
			}

			for i, expected := range tt.expected {
				if i >= len(imports) {
					t.Errorf("Missing import at index %d", i)
					continue
				}
				
				actual := imports[i]
				if actual.Path != expected.Path {
					t.Errorf("Import %d: expected path %q, got %q", i, expected.Path, actual.Path)
				}
				if actual.Alias != expected.Alias {
					t.Errorf("Import %d: expected alias %q, got %q", i, expected.Alias, actual.Alias)
				}
			}
		})
	}
}

func TestParser_ParseSourceWithImports(t *testing.T) {
	p := NewParser()

	source := `package controllers

import (
	"context"
	"net/http"
	"github.com/toyz/axon/pkg/axon"
)

//axon::controller
type UserController struct {
	fx.In
	UserService UserServiceInterface
}

//axon::route GET /users/{id:int}
func (c *UserController) GetUser(ctx context.Context, id int) (*User, error) {
	return c.UserService.GetUser(ctx, id)
}`

	metadata, err := p.ParseSource("user_controller.go", source)
	if err != nil {
		t.Fatalf("Failed to parse source: %v", err)
	}

	// Verify imports were captured
	imports, exists := metadata.SourceImports["user_controller.go"]
	if !exists {
		t.Fatal("Expected imports to be captured for user_controller.go")
	}

	expectedImports := []models.Import{
		{Path: "context", Alias: ""},
		{Path: "net/http", Alias: ""},
		{Path: "github.com/toyz/axon/pkg/axon", Alias: ""},
	}

	if len(imports) != len(expectedImports) {
		t.Errorf("Expected %d imports, got %d", len(expectedImports), len(imports))
		return
	}

	for i, expected := range expectedImports {
		actual := imports[i]
		if actual.Path != expected.Path {
			t.Errorf("Import %d: expected path %q, got %q", i, expected.Path, actual.Path)
		}
		if actual.Alias != expected.Alias {
			t.Errorf("Import %d: expected alias %q, got %q", i, expected.Alias, actual.Alias)
		}
	}

	// Verify other metadata is still working
	if len(metadata.Controllers) != 1 {
		t.Errorf("Expected 1 controller, got %d", len(metadata.Controllers))
	}
}

func TestParser_detectModuleInfo(t *testing.T) {
	p := NewParser()

	// Create a temporary directory structure for testing
	tempDir := t.TempDir()
	
	// Create a go.mod file
	goModContent := `module github.com/test/project

go 1.21
`
	goModPath := filepath.Join(tempDir, "go.mod")
	if err := os.WriteFile(goModPath, []byte(goModContent), 0644); err != nil {
		t.Fatalf("Failed to create go.mod: %v", err)
	}
	
	// Create a subdirectory for the package
	packageDir := filepath.Join(tempDir, "internal", "services")
	if err := os.MkdirAll(packageDir, 0755); err != nil {
		t.Fatalf("Failed to create package directory: %v", err)
	}

	// Test module detection
	metadata := &models.PackageMetadata{
		PackageName:   "services",
		PackagePath:   packageDir,
		SourceImports: make(map[string][]models.Import),
	}

	err := p.detectModuleInfo(metadata)
	if err != nil {
		t.Fatalf("Failed to detect module info: %v", err)
	}

	// Verify results
	expectedModulePath := "github.com/test/project"
	if metadata.ModulePath != expectedModulePath {
		t.Errorf("Expected module path %q, got %q", expectedModulePath, metadata.ModulePath)
	}

	if metadata.ModuleRoot != tempDir {
		t.Errorf("Expected module root %q, got %q", tempDir, metadata.ModuleRoot)
	}

	expectedPackageImportPath := "github.com/test/project/internal/services"
	if metadata.PackageImportPath != expectedPackageImportPath {
		t.Errorf("Expected package import path %q, got %q", expectedPackageImportPath, metadata.PackageImportPath)
	}
}

func TestParser_parseGoModFile(t *testing.T) {
	p := NewParser()

	tests := []struct {
		name        string
		content     string
		expected    string
		expectError bool
	}{
		{
			name: "simple module",
			content: `module github.com/user/project

go 1.21
`,
			expected:    "github.com/user/project",
			expectError: false,
		},
		{
			name: "module with comments",
			content: `// This is a comment
module github.com/example/app

go 1.20
require (
	github.com/some/dep v1.0.0
)
`,
			expected:    "github.com/example/app",
			expectError: false,
		},
		{
			name: "no module declaration",
			content: `go 1.21

require (
	github.com/some/dep v1.0.0
)
`,
			expected:    "",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary file
			tempFile := filepath.Join(t.TempDir(), "go.mod")
			if err := os.WriteFile(tempFile, []byte(tt.content), 0644); err != nil {
				t.Fatalf("Failed to create temp file: %v", err)
			}

			// Test parsing
			result, err := p.parseGoModFile(tempFile)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if result != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}
		})
	}
}