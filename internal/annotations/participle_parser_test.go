package annotations

import (
	"reflect"
	"testing"
)

func TestParticipleParserBasic(t *testing.T) {
	// Create a registry with builtin schemas
	registry := NewRegistry()
	err := RegisterBuiltinSchemas(registry)
	if err != nil {
		t.Fatalf("Failed to register builtin schemas: %v", err)
	}
	
	parser := NewParticipleParser(registry)
	location := SourceLocation{File: "test.go", Line: 1, Column: 1}

	tests := []struct {
		name     string
		input    string
		expected *ParsedAnnotation
	}{
		{
			name:  "simple middleware",
			input: "//axon::middleware",
			expected: &ParsedAnnotation{
				Type:       MiddlewareAnnotation,
				Target:     "",
				Parameters: map[string]interface{}{},
				Raw:        "//axon::middleware",
			},
		},
		{
			name:  "middleware with name",
			input: "//axon::middleware AuthMiddleware",
			expected: &ParsedAnnotation{
				Type:       MiddlewareAnnotation,
				Target:     "",
				Parameters: map[string]interface{}{"Name": "AuthMiddleware"},
				Raw:        "//axon::middleware AuthMiddleware",
			},
		},
		{
			name:  "middleware with flag",
			input: "//axon::middleware AuthMiddleware -Global",
			expected: &ParsedAnnotation{
				Type:       MiddlewareAnnotation,
				Target:     "",
				Parameters: map[string]interface{}{"Name": "AuthMiddleware", "Global": true},
				Raw:        "//axon::middleware AuthMiddleware -Global",
			},
		},
		{
			name:  "core annotation",
			input: "//axon::core",
			expected: &ParsedAnnotation{
				Type:       CoreAnnotation,
				Target:     "",
				Parameters: map[string]interface{}{},
				Raw:        "//axon::core",
			},
		},
		{
			name:  "core with mode",
			input: "//axon::core -Mode=Transient",
			expected: &ParsedAnnotation{
				Type:       CoreAnnotation,
				Target:     "",
				Parameters: map[string]interface{}{"Mode": "Transient"},
				Raw:        "//axon::core -Mode=Transient",
			},
		},
		{
			name:  "route annotation",
			input: "//axon::route GET /users",
			expected: &ParsedAnnotation{
				Type:       RouteAnnotation,
				Target:     "",
				Parameters: map[string]interface{}{"method": "GET", "path": "/users"},
				Raw:        "//axon::route GET /users",
			},
		},
		{
			name:  "route with middleware",
			input: "//axon::route POST /users -Middleware=Auth",
			expected: &ParsedAnnotation{
				Type:       RouteAnnotation,
				Target:     "",
				Parameters: map[string]interface{}{"method": "POST", "path": "/users", "Middleware": []string{"Auth"}},
				Raw:        "//axon::route POST /users -Middleware=Auth",
			},
		},
		{
			name:  "controller annotation",
			input: "//axon::controller",
			expected: &ParsedAnnotation{
				Type:       ControllerAnnotation,
				Target:     "",
				Parameters: map[string]interface{}{},
				Raw:        "//axon::controller",
			},
		},
		{
			name:  "controller with prefix",
			input: "//axon::controller -Prefix=/api/v1",
			expected: &ParsedAnnotation{
				Type:       ControllerAnnotation,
				Target:     "",
				Parameters: map[string]interface{}{"Prefix": "/api/v1"},
				Raw:        "//axon::controller -Prefix=/api/v1",
			},
		},
		{
			name:  "interface annotation",
			input: "//axon::interface",
			expected: &ParsedAnnotation{
				Type:       InterfaceAnnotation,
				Target:     "",
				Parameters: map[string]interface{}{},
				Raw:        "//axon::interface",
			},
		},
		{
			name:  "interface with name",
			input: "//axon::interface -Name=UserRepository",
			expected: &ParsedAnnotation{
				Type:       InterfaceAnnotation,
				Target:     "",
				Parameters: map[string]interface{}{"Name": "UserRepository"},
				Raw:        "//axon::interface -Name=UserRepository",
			},
		},
		{
			name:  "inject annotation",
			input: "//axon::inject",
			expected: &ParsedAnnotation{
				Type:       InjectAnnotation,
				Target:     "",
				Parameters: map[string]interface{}{},
				Raw:        "//axon::inject",
			},
		},

		{
			name:  "init annotation",
			input: "//axon::init",
			expected: &ParsedAnnotation{
				Type:       InitAnnotation,
				Target:     "",
				Parameters: map[string]interface{}{},
				Raw:        "//axon::init",
			},
		},
		{
			name:  "logger annotation",
			input: "//axon::logger",
			expected: &ParsedAnnotation{
				Type:       LoggerAnnotation,
				Target:     "",
				Parameters: map[string]interface{}{},
				Raw:        "//axon::logger",
			},
		},
		{
			name:  "route parser annotation",
			input: "//axon::route_parser UUID",
			expected: &ParsedAnnotation{
				Type:       RouteParserAnnotation,
				Target:     "UUID",
				Parameters: map[string]interface{}{"name": "UUID"},
				Raw:        "//axon::route_parser UUID",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parser.ParseAnnotation(tt.input, location)
			if err != nil {
				t.Logf("Parse error: %v", err)
				t.Logf("Input: %q", tt.input)
				t.FailNow()
			}

			if result.Type != tt.expected.Type {
				t.Errorf("expected type %v, got %v", tt.expected.Type, result.Type)
			}

			if result.Target != tt.expected.Target {
				t.Errorf("expected target %q, got %q", tt.expected.Target, result.Target)
			}

			if len(result.Parameters) != len(tt.expected.Parameters) {
				t.Errorf("expected %d parameters, got %d", len(tt.expected.Parameters), len(result.Parameters))
			}

			for key, expectedValue := range tt.expected.Parameters {
				if actualValue, exists := result.Parameters[key]; !exists {
					t.Errorf("expected parameter %q with value %v, but parameter not found", key, expectedValue)
				} else if !valuesEqual(actualValue, expectedValue) {
					t.Errorf("expected parameter %q to have value %v, got %v", key, expectedValue, actualValue)
				}
			}

			// No need to check flags since they're now just boolean parameters
		})
	}
}

// valuesEqual compares two interface{} values, handling slices and other complex types
func valuesEqual(a, b interface{}) bool {
	if a == nil || b == nil {
		return a == b
	}

	// Handle slices
	if aSlice, ok := a.([]string); ok {
		if bSlice, ok := b.([]string); ok {
			if len(aSlice) != len(bSlice) {
				return false
			}
			for i := range aSlice {
				if aSlice[i] != bSlice[i] {
					return false
				}
			}
			return true
		}
	}

	// Handle []interface{} slices
	if aSlice, ok := a.([]interface{}); ok {
		if bSlice, ok := b.([]interface{}); ok {
			if len(aSlice) != len(bSlice) {
				return false
			}
			for i := range aSlice {
				if !valuesEqual(aSlice[i], bSlice[i]) {
					return false
				}
			}
			return true
		}
	}

	// For other types, use reflection-based comparison
	return reflect.DeepEqual(a, b)
}
