package registry

import (
	"strings"
	"testing"

	"github.com/toyz/axon/internal/models"
)

func TestMiddlewareRegistry_Register(t *testing.T) {
	tests := []struct {
		name        string
		middlewares []struct {
			name       string
			middleware *models.MiddlewareMetadata
		}
		expectError bool
		errorMsg    string
	}{
		{
			name: "register single middleware",
			middlewares: []struct {
				name       string
				middleware *models.MiddlewareMetadata
			}{
				{
					name: "AuthMiddleware",
					middleware: &models.MiddlewareMetadata{
						Name:        "AuthMiddleware",
						PackagePath: "/auth",
						StructName:  "AuthMiddleware",
					},
				},
			},
			expectError: false,
		},
		{
			name: "register multiple middlewares",
			middlewares: []struct {
				name       string
				middleware *models.MiddlewareMetadata
			}{
				{
					name: "AuthMiddleware",
					middleware: &models.MiddlewareMetadata{
						Name:        "AuthMiddleware",
						PackagePath: "/auth",
						StructName:  "AuthMiddleware",
					},
				},
				{
					name: "LoggingMiddleware",
					middleware: &models.MiddlewareMetadata{
						Name:        "LoggingMiddleware",
						PackagePath: "/logging",
						StructName:  "LoggingMiddleware",
					},
				},
			},
			expectError: false,
		},
		{
			name: "register duplicate middleware name",
			middlewares: []struct {
				name       string
				middleware *models.MiddlewareMetadata
			}{
				{
					name: "AuthMiddleware",
					middleware: &models.MiddlewareMetadata{
						Name:        "AuthMiddleware",
						PackagePath: "/auth",
						StructName:  "AuthMiddleware",
					},
				},
				{
					name: "AuthMiddleware",
					middleware: &models.MiddlewareMetadata{
						Name:        "AuthMiddleware",
						PackagePath: "/auth2",
						StructName:  "AuthMiddleware",
					},
				},
			},
			expectError: true,
			errorMsg:    "middleware 'AuthMiddleware' is already registered",
		},
		{
			name: "register with empty name",
			middlewares: []struct {
				name       string
				middleware *models.MiddlewareMetadata
			}{
				{
					name: "",
					middleware: &models.MiddlewareMetadata{
						Name:        "AuthMiddleware",
						PackagePath: "/auth",
						StructName:  "AuthMiddleware",
					},
				},
			},
			expectError: true,
			errorMsg:    "middleware name cannot be empty",
		},
		{
			name: "register with nil middleware",
			middlewares: []struct {
				name       string
				middleware *models.MiddlewareMetadata
			}{
				{
					name:       "AuthMiddleware",
					middleware: nil,
				},
			},
			expectError: true,
			errorMsg:    "middleware metadata cannot be nil",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			registry := NewMiddlewareRegistry()
			
			var err error
			for _, mw := range tt.middlewares {
				err = registry.Register(mw.name, mw.middleware)
				if err != nil {
					break
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
				}
			}
		})
	}
}

func TestMiddlewareRegistry_Validate(t *testing.T) {
	tests := []struct {
		name            string
		registeredNames []string
		validateNames   []string
		expectError     bool
		errorMsg        string
	}{
		{
			name:            "validate existing middleware",
			registeredNames: []string{"AuthMiddleware", "LoggingMiddleware"},
			validateNames:   []string{"AuthMiddleware"},
			expectError:     false,
		},
		{
			name:            "validate multiple existing middlewares",
			registeredNames: []string{"AuthMiddleware", "LoggingMiddleware", "CorsMiddleware"},
			validateNames:   []string{"AuthMiddleware", "LoggingMiddleware"},
			expectError:     false,
		},
		{
			name:            "validate non-existing middleware",
			registeredNames: []string{"AuthMiddleware"},
			validateNames:   []string{"NonExistentMiddleware"},
			expectError:     true,
			errorMsg:        "unknown middleware(s): NonExistentMiddleware",
		},
		{
			name:            "validate mix of existing and non-existing middlewares",
			registeredNames: []string{"AuthMiddleware"},
			validateNames:   []string{"AuthMiddleware", "NonExistentMiddleware", "AnotherMissing"},
			expectError:     true,
			errorMsg:        "unknown middleware(s): NonExistentMiddleware, AnotherMissing",
		},
		{
			name:            "validate empty list",
			registeredNames: []string{"AuthMiddleware"},
			validateNames:   []string{},
			expectError:     false,
		},
		{
			name:            "validate with whitespace names",
			registeredNames: []string{"AuthMiddleware", "LoggingMiddleware"},
			validateNames:   []string{" AuthMiddleware ", "LoggingMiddleware"},
			expectError:     false,
		},
		{
			name:            "validate with empty string in list",
			registeredNames: []string{"AuthMiddleware"},
			validateNames:   []string{"AuthMiddleware", "", "LoggingMiddleware"},
			expectError:     true,
			errorMsg:        "unknown middleware(s): LoggingMiddleware",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			registry := NewMiddlewareRegistry()
			
			// Register middlewares
			for _, name := range tt.registeredNames {
				middleware := &models.MiddlewareMetadata{
					Name:        name,
					PackagePath: "/test",
					StructName:  name,
				}
				err := registry.Register(name, middleware)
				if err != nil {
					t.Fatalf("failed to register middleware %s: %v", name, err)
				}
			}
			
			// Validate middleware names
			err := registry.Validate(tt.validateNames)

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

func TestMiddlewareRegistry_Get(t *testing.T) {
	registry := NewMiddlewareRegistry()
	
	// Register some middlewares
	authMiddleware := &models.MiddlewareMetadata{
		Name:        "AuthMiddleware",
		PackagePath: "/auth",
		StructName:  "AuthMiddleware",
		Dependencies: []string{"TokenService"},
	}
	
	loggingMiddleware := &models.MiddlewareMetadata{
		Name:        "LoggingMiddleware",
		PackagePath: "/logging",
		StructName:  "LoggingMiddleware",
		Dependencies: []string{"Logger"},
	}
	
	err := registry.Register("AuthMiddleware", authMiddleware)
	if err != nil {
		t.Fatalf("failed to register AuthMiddleware: %v", err)
	}
	
	err = registry.Register("LoggingMiddleware", loggingMiddleware)
	if err != nil {
		t.Fatalf("failed to register LoggingMiddleware: %v", err)
	}

	tests := []struct {
		name     string
		getName  string
		expected *models.MiddlewareMetadata
		exists   bool
	}{
		{
			name:     "get existing middleware",
			getName:  "AuthMiddleware",
			expected: authMiddleware,
			exists:   true,
		},
		{
			name:     "get another existing middleware",
			getName:  "LoggingMiddleware",
			expected: loggingMiddleware,
			exists:   true,
		},
		{
			name:     "get non-existing middleware",
			getName:  "NonExistentMiddleware",
			expected: nil,
			exists:   false,
		},
		{
			name:     "get with empty name",
			getName:  "",
			expected: nil,
			exists:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, exists := registry.Get(tt.getName)

			if exists != tt.exists {
				t.Errorf("expected exists=%v, got exists=%v", tt.exists, exists)
			}

			if tt.exists {
				if result == nil {
					t.Errorf("expected middleware but got nil")
					return
				}
				
				if result.Name != tt.expected.Name {
					t.Errorf("expected name %s, got %s", tt.expected.Name, result.Name)
				}
				
				if result.PackagePath != tt.expected.PackagePath {
					t.Errorf("expected package path %s, got %s", tt.expected.PackagePath, result.PackagePath)
				}
				
				if result.StructName != tt.expected.StructName {
					t.Errorf("expected struct name %s, got %s", tt.expected.StructName, result.StructName)
				}
				
				if len(result.Dependencies) != len(tt.expected.Dependencies) {
					t.Errorf("expected %d dependencies, got %d", len(tt.expected.Dependencies), len(result.Dependencies))
				}
			} else {
				if result != nil {
					t.Errorf("expected nil but got middleware: %+v", result)
				}
			}
		})
	}
}