package annotations

import (
	"testing"
)

func TestCoreAnnotationSchema(t *testing.T) {
	tests := []struct {
		name        string
		parameters  map[string]interface{}
		expectError bool
		errorMsg    string
	}{
		{
			name:        "valid default parameters",
			parameters:  map[string]interface{}{},
			expectError: false,
		},
		{
			name: "valid singleton mode",
			parameters: map[string]interface{}{
				"Mode": "Singleton",
			},
			expectError: false,
		},
		{
			name: "valid transient mode",
			parameters: map[string]interface{}{
				"Mode": "Transient",
			},
			expectError: false,
		},
		{
			name: "valid background init",
			parameters: map[string]interface{}{
				"Init": "Background",
			},
			expectError: false,
		},
		{
			name: "valid manual module",
			parameters: map[string]interface{}{
				"Manual": "CustomModule",
			},
			expectError: false,
		},
		{
			name: "invalid mode",
			parameters: map[string]interface{}{
				"Mode": "Invalid",
			},
			expectError: true,
			errorMsg:    "must be 'Singleton' or 'Transient'",
		},
		{
			name: "invalid init",
			parameters: map[string]interface{}{
				"Init": "Invalid",
			},
			expectError: true,
			errorMsg:    "must be 'Same' or 'Background'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test Mode parameter validation
			if mode, exists := tt.parameters["Mode"]; exists {
				if validator := CoreAnnotationSchema.Parameters["Mode"].Validator; validator != nil {
					err := validator(mode)
					if tt.expectError && err == nil {
						t.Errorf("expected error for Mode parameter, got nil")
					}
					if !tt.expectError && err != nil {
						t.Errorf("unexpected error for Mode parameter: %v", err)
					}
					if tt.expectError && err != nil && tt.errorMsg != "" {
						if !contains(err.Error(), tt.errorMsg) {
							t.Errorf("expected error message to contain '%s', got '%s'", tt.errorMsg, err.Error())
						}
					}
				}
			}

			// Test Init parameter validation
			if init, exists := tt.parameters["Init"]; exists {
				if validator := CoreAnnotationSchema.Parameters["Init"].Validator; validator != nil {
					err := validator(init)
					if tt.expectError && err == nil {
						t.Errorf("expected error for Init parameter, got nil")
					}
					if !tt.expectError && err != nil {
						t.Errorf("unexpected error for Init parameter: %v", err)
					}
					if tt.expectError && err != nil && tt.errorMsg != "" {
						if !contains(err.Error(), tt.errorMsg) {
							t.Errorf("expected error message to contain '%s', got '%s'", tt.errorMsg, err.Error())
						}
					}
				}
			}
		})
	}
}

func TestRouteAnnotationSchema(t *testing.T) {
	tests := []struct {
		name        string
		parameters  map[string]interface{}
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid GET method",
			parameters: map[string]interface{}{
				"method": "GET",
			},
			expectError: false,
		},
		{
			name: "valid POST method",
			parameters: map[string]interface{}{
				"method": "POST",
			},
			expectError: false,
		},
		{
			name: "valid lowercase method",
			parameters: map[string]interface{}{
				"method": "get",
			},
			expectError: false, // Should be normalized to uppercase
		},
		{
			name: "invalid method",
			parameters: map[string]interface{}{
				"method": "INVALID",
			},
			expectError: true,
			errorMsg:    "must be one of",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if method, exists := tt.parameters["method"]; exists {
				if validator := RouteAnnotationSchema.Parameters["method"].Validator; validator != nil {
					err := validator(method)
					if tt.expectError && err == nil {
						t.Errorf("expected error for method parameter, got nil")
					}
					if !tt.expectError && err != nil {
						t.Errorf("unexpected error for method parameter: %v", err)
					}
					if tt.expectError && err != nil && tt.errorMsg != "" {
						if !contains(err.Error(), tt.errorMsg) {
							t.Errorf("expected error message to contain '%s', got '%s'", tt.errorMsg, err.Error())
						}
					}
				}
			}
		})
	}
}

func TestRouteParametersValidator(t *testing.T) {
	tests := []struct {
		name        string
		annotation  *ParsedAnnotation
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid route with method and path",
			annotation: &ParsedAnnotation{
				Type: RouteAnnotation,
				Parameters: map[string]interface{}{
					"method": "GET",
					"path":   "/users",
				},
			},
			expectError: false,
		},
		{
			name: "missing method",
			annotation: &ParsedAnnotation{
				Type: RouteAnnotation,
				Parameters: map[string]interface{}{
					"path": "/users",
				},
			},
			expectError: true,
			errorMsg:    "validation error for field 'method'",
		},
		{
			name: "missing path",
			annotation: &ParsedAnnotation{
				Type: RouteAnnotation,
				Parameters: map[string]interface{}{
					"method": "GET",
				},
			},
			expectError: true,
			errorMsg:    "validation error for field 'path'",
		},
		{
			name: "invalid path format",
			annotation: &ParsedAnnotation{
				Type: RouteAnnotation,
				Parameters: map[string]interface{}{
					"method": "GET",
					"path":   "users", // Missing leading slash
				},
			},
			expectError: true,
			errorMsg:    "must start with '/'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateRouteParameters(tt.annotation)
			if tt.expectError && err == nil {
				t.Errorf("expected error, got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if tt.expectError && err != nil && tt.errorMsg != "" {
				if !contains(err.Error(), tt.errorMsg) {
					t.Errorf("expected error message to contain '%s', got '%s'", tt.errorMsg, err.Error())
				}
			}
		})
	}
}

func TestMiddlewareParametersValidator(t *testing.T) {
	tests := []struct {
		name        string
		annotation  *ParsedAnnotation
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid middleware with routes",
			annotation: &ParsedAnnotation{
				Type: MiddlewareAnnotation,
				Parameters: map[string]interface{}{
					"Routes": []string{"/api/*", "/admin/*"},
				},
			},
			expectError: false,
		},
		{
			name: "middleware without routes",
			annotation: &ParsedAnnotation{
				Type:       MiddlewareAnnotation,
				Parameters: map[string]interface{}{},
			},
			expectError: false,
		},
		{
			name: "empty route pattern",
			annotation: &ParsedAnnotation{
				Type: MiddlewareAnnotation,
				Parameters: map[string]interface{}{
					"Routes": []string{"/api/*", ""},
				},
			},
			expectError: true,
			errorMsg:    "validation error for field 'route'",
		},
		{
			name: "invalid route pattern",
			annotation: &ParsedAnnotation{
				Type: MiddlewareAnnotation,
				Parameters: map[string]interface{}{
					"Routes": []string{"api/*"}, // Missing leading slash
				},
			},
			expectError: true,
			errorMsg:    "must start with '/'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateMiddlewareParameters(tt.annotation)
			if tt.expectError && err == nil {
				t.Errorf("expected error, got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if tt.expectError && err != nil && tt.errorMsg != "" {
				if !contains(err.Error(), tt.errorMsg) {
					t.Errorf("expected error message to contain '%s', got '%s'", tt.errorMsg, err.Error())
				}
			}
		})
	}
}

func TestRegisterBuiltinSchemas(t *testing.T) {
	registry := NewRegistry()

	err := RegisterBuiltinSchemas(registry)
	if err != nil {
		t.Fatalf("failed to register builtin schemas: %v", err)
	}

	// Verify all schemas are registered
	expectedTypes := []AnnotationType{
		CoreAnnotation,
		RouteAnnotation,
		ControllerAnnotation,
		MiddlewareAnnotation,
		InterfaceAnnotation,
	}

	for _, expectedType := range expectedTypes {
		if !registry.IsRegistered(expectedType) {
			t.Errorf("expected %s to be registered", expectedType.String())
		}

		schema, err := registry.GetSchema(expectedType)
		if err != nil {
			t.Errorf("failed to get schema for %s: %v", expectedType.String(), err)
		}

		if schema.Type != expectedType {
			t.Errorf("expected schema type %s, got %s", expectedType.String(), schema.Type.String())
		}
	}
}

func TestGetBuiltinSchemas(t *testing.T) {
	schemas := GetBuiltinSchemas()

	expectedCount := 10
	if len(schemas) != expectedCount {
		t.Errorf("expected %d builtin schemas, got %d", expectedCount, len(schemas))
	}

	// Verify all expected types are present
	expectedTypes := map[AnnotationType]bool{
		ServiceAnnotation:     false,
		CoreAnnotation:        false, // Deprecated but still supported
		RouteAnnotation:       false,
		ControllerAnnotation:  false,
		MiddlewareAnnotation:  false,
		InterfaceAnnotation:   false,
		InjectAnnotation:      false,
		InitAnnotation:        false,
		LoggerAnnotation:      false,
		RouteParserAnnotation: false,
	}

	for _, schema := range schemas {
		if _, exists := expectedTypes[schema.Type]; exists {
			expectedTypes[schema.Type] = true
		} else {
			t.Errorf("unexpected schema type: %s", schema.Type.String())
		}
	}

	for schemaType, found := range expectedTypes {
		if !found {
			t.Errorf("missing schema for type: %s", schemaType.String())
		}
	}
}

func TestSchemaExamples(t *testing.T) {
	schemas := GetBuiltinSchemas()

	for _, schema := range schemas {
		if len(schema.Examples) == 0 {
			t.Errorf("schema %s has no examples", schema.Type.String())
		}

		for i, example := range schema.Examples {
			if example == "" {
				t.Errorf("schema %s has empty example at index %d", schema.Type.String(), i)
			}
		}
	}
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr ||
			func() bool {
				for i := 0; i <= len(s)-len(substr); i++ {
					if s[i:i+len(substr)] == substr {
						return true
					}
				}
				return false
			}())))
}
