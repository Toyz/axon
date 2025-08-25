package annotations

import (
	"testing"
)

func TestBuiltinSchemasIntegration(t *testing.T) {
	// Create registry and register builtin schemas
	registry := NewRegistry()
	err := RegisterBuiltinSchemas(registry)
	if err != nil {
		t.Fatalf("failed to register builtin schemas: %v", err)
	}

	// Create parser
	parser := NewParser(registry)
	location := SourceLocation{File: "test.go", Line: 1, Column: 1}

	tests := []struct {
		name           string
		input          string
		expectedType   AnnotationType
		expectedParams map[string]interface{}
	}{
		{
			name:         "core annotation with defaults",
			input:        "//axon::core",
			expectedType: CoreAnnotation,
			expectedParams: map[string]interface{}{
				"Mode": "Singleton",
				"Init": "Same",
			},
		},
		{
			name:         "core annotation with transient mode",
			input:        "//axon::core -Mode=Transient",
			expectedType: CoreAnnotation,
			expectedParams: map[string]interface{}{
				"Mode": "Transient",
				"Init": "Same",
			},
		},
		{
			name:         "core annotation with background init",
			input:        "//axon::core -Init=Background",
			expectedType: CoreAnnotation,
			expectedParams: map[string]interface{}{
				"Mode": "Singleton",
				"Init": "Background",
			},
		},
		{
			name:         "core annotation with manual module",
			input:        "//axon::core -Manual=\"CustomModule\"",
			expectedType: CoreAnnotation,
			expectedParams: map[string]interface{}{
				"Mode":   "Singleton",
				"Init":   "Same",
				"Manual": "CustomModule",
			},
		},
		{
			name:         "route annotation basic",
			input:        "//axon::route GET /users",
			expectedType: RouteAnnotation,
			expectedParams: map[string]interface{}{
				"method":      "GET",
				"path":        "/users",
				"PassContext": false,
			},
		},
		{
			name:         "route annotation with middleware",
			input:        "//axon::route POST /users -Middleware=Auth,Logging",
			expectedType: RouteAnnotation,
			expectedParams: map[string]interface{}{
				"method":      "POST",
				"path":        "/users",
				"Middleware":  []string{"Auth", "Logging"},
				"PassContext": false,
			},
		},
		{
			name:         "route annotation with pass context",
			input:        "//axon::route DELETE /users/{id:int} -PassContext",
			expectedType: RouteAnnotation,
			expectedParams: map[string]interface{}{
				"method":      "DELETE",
				"path":        "/users/{id:int}",
				"PassContext": true,
			},
		},
		{
			name:         "controller annotation basic",
			input:        "//axon::controller",
			expectedType: ControllerAnnotation,
			expectedParams: map[string]interface{}{},
		},
		{
			name:         "controller annotation with prefix",
			input:        "//axon::controller -Prefix=/api/v1",
			expectedType: ControllerAnnotation,
			expectedParams: map[string]interface{}{
				"Prefix": "/api/v1",
			},
		},
		{
			name:         "middleware annotation basic",
			input:        "//axon::middleware",
			expectedType: MiddlewareAnnotation,
			expectedParams: map[string]interface{}{
				"Priority": 0,
				"Global":   false,
			},
		},
		{
			name:         "middleware annotation with name",
			input:        "//axon::middleware AuthMiddleware",
			expectedType: MiddlewareAnnotation,
			expectedParams: map[string]interface{}{
				"Name":     "AuthMiddleware",
				"Priority": 0,
				"Global":   false,
			},
		},
		{
			name:         "interface annotation basic",
			input:        "//axon::interface",
			expectedType: InterfaceAnnotation,
			expectedParams: map[string]interface{}{
				"Singleton": true,
				"Primary":   false,
			},
		},
		{
			name:         "interface annotation with name",
			input:        "//axon::interface -Name=UserRepository",
			expectedType: InterfaceAnnotation,
			expectedParams: map[string]interface{}{
				"Name":      "UserRepository",
				"Singleton": true,
				"Primary":   false,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			annotation, err := parser.ParseAnnotation(tt.input, location)
			if err != nil {
				t.Fatalf("unexpected error parsing %s: %v", tt.input, err)
			}

			if annotation.Type != tt.expectedType {
				t.Errorf("expected type %s, got %s", tt.expectedType.String(), annotation.Type.String())
			}

			// Check expected parameters
			for key, expectedValue := range tt.expectedParams {
				actualValue, exists := annotation.Parameters[key]
				if !exists {
					t.Errorf("expected parameter %s to exist", key)
					continue
				}

				// Handle different types
				switch expected := expectedValue.(type) {
				case string:
					if actual, ok := actualValue.(string); !ok || actual != expected {
						t.Errorf("expected parameter %s to be %q, got %v", key, expected, actualValue)
					}
				case bool:
					if actual, ok := actualValue.(bool); !ok || actual != expected {
						t.Errorf("expected parameter %s to be %t, got %v", key, expected, actualValue)
					}
				case int:
					if actual, ok := actualValue.(int); !ok || actual != expected {
						t.Errorf("expected parameter %s to be %d, got %v", key, expected, actualValue)
					}
				case []string:
					if actual, ok := actualValue.([]string); !ok {
						t.Errorf("expected parameter %s to be []string, got %T", key, actualValue)
					} else {
						if len(actual) != len(expected) {
							t.Errorf("expected parameter %s to have length %d, got %d", key, len(expected), len(actual))
						} else {
							for i, expectedItem := range expected {
								if i >= len(actual) || actual[i] != expectedItem {
									t.Errorf("expected parameter %s[%d] to be %q, got %q", key, i, expectedItem, actual[i])
								}
							}
						}
					}
				default:
					t.Errorf("unsupported expected type for parameter %s: %T", key, expectedValue)
				}
			}
		})
	}
}

func TestBuiltinSchemasValidation(t *testing.T) {
	// Create registry and register builtin schemas
	registry := NewRegistry()
	err := RegisterBuiltinSchemas(registry)
	if err != nil {
		t.Fatalf("failed to register builtin schemas: %v", err)
	}

	// Create parser
	parser := NewParser(registry)
	location := SourceLocation{File: "test.go", Line: 1, Column: 1}

	// Test validation errors
	errorTests := []struct {
		name        string
		input       string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "invalid core mode",
			input:       "//axon::core -Mode=Invalid",
			expectError: true,
			errorMsg:    "must be 'Singleton' or 'Transient'",
		},
		{
			name:        "invalid core init",
			input:       "//axon::core -Init=Invalid",
			expectError: true,
			errorMsg:    "must be 'Same' or 'Background'",
		},
		{
			name:        "invalid route method",
			input:       "//axon::route INVALID /users",
			expectError: true,
			errorMsg:    "must be one of",
		},
		{
			name:        "missing route method",
			input:       "//axon::route",
			expectError: true,
			errorMsg:    "requires method parameter",
		},
		{
			name:        "invalid route path",
			input:       "//axon::route GET users",
			expectError: true,
			errorMsg:    "must start with '/'",
		},
		{
			name:        "invalid middleware route pattern",
			input:       "//axon::middleware -Routes=api/*",
			expectError: true,
			errorMsg:    "must start with '/'",
		},
	}

	for _, tt := range errorTests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parser.ParseAnnotation(tt.input, location)
			if tt.expectError && err == nil {
				t.Errorf("expected error for %s, got nil", tt.input)
			}
			if !tt.expectError && err != nil {
				t.Errorf("unexpected error for %s: %v", tt.input, err)
			}
			if tt.expectError && err != nil && tt.errorMsg != "" {
				if !contains(err.Error(), tt.errorMsg) {
					t.Errorf("expected error message to contain '%s', got '%s'", tt.errorMsg, err.Error())
				}
			}
		})
	}
}

func TestBuiltinSchemasTypeConversion(t *testing.T) {
	// Create registry and register builtin schemas
	registry := NewRegistry()
	err := RegisterBuiltinSchemas(registry)
	if err != nil {
		t.Fatalf("failed to register builtin schemas: %v", err)
	}

	// Create parser
	parser := NewParser(registry)
	location := SourceLocation{File: "test.go", Line: 1, Column: 1}

	// Test type conversions
	tests := []struct {
		name           string
		input          string
		paramName      string
		expectedType   string
		expectedValue  interface{}
	}{
		{
			name:          "boolean flag conversion",
			input:         "//axon::route GET /users -PassContext",
			paramName:     "PassContext",
			expectedType:  "bool",
			expectedValue: true,
		},
		{
			name:          "string slice conversion",
			input:         "//axon::route GET /users -Middleware=Auth,Logging,Validation",
			paramName:     "Middleware",
			expectedType:  "[]string",
			expectedValue: []string{"Auth", "Logging", "Validation"},
		},
		{
			name:          "integer conversion",
			input:         "//axon::middleware -Priority=10",
			paramName:     "Priority",
			expectedType:  "int",
			expectedValue: 10,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			annotation, err := parser.ParseAnnotation(tt.input, location)
			if err != nil {
				t.Fatalf("unexpected error parsing %s: %v", tt.input, err)
			}

			value, exists := annotation.Parameters[tt.paramName]
			if !exists {
				t.Fatalf("expected parameter %s to exist", tt.paramName)
			}

			switch tt.expectedType {
			case "bool":
				if actual, ok := value.(bool); !ok {
					t.Errorf("expected parameter %s to be bool, got %T", tt.paramName, value)
				} else if actual != tt.expectedValue.(bool) {
					t.Errorf("expected parameter %s to be %t, got %t", tt.paramName, tt.expectedValue.(bool), actual)
				}
			case "[]string":
				if actual, ok := value.([]string); !ok {
					t.Errorf("expected parameter %s to be []string, got %T", tt.paramName, value)
				} else {
					expected := tt.expectedValue.([]string)
					if len(actual) != len(expected) {
						t.Errorf("expected parameter %s to have length %d, got %d", tt.paramName, len(expected), len(actual))
					} else {
						for i, expectedItem := range expected {
							if actual[i] != expectedItem {
								t.Errorf("expected parameter %s[%d] to be %q, got %q", tt.paramName, i, expectedItem, actual[i])
							}
						}
					}
				}
			case "int":
				if actual, ok := value.(int); !ok {
					t.Errorf("expected parameter %s to be int, got %T", tt.paramName, value)
				} else if actual != tt.expectedValue.(int) {
					t.Errorf("expected parameter %s to be %d, got %d", tt.paramName, tt.expectedValue.(int), actual)
				}
			}
		})
	}
}