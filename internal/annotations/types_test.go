package annotations

import (
	"testing"
)

func TestParsedAnnotation_GetString(t *testing.T) {
	annotation := &ParsedAnnotation{
		Parameters: map[string]interface{}{
			"stringParam":  "test_value",
			"intParam":     42,
			"boolParam":    true,
			"emptyString":  "",
		},
	}

	tests := []struct {
		name         string
		paramName    string
		defaultValue []string
		expected     string
	}{
		{
			name:      "existing string parameter",
			paramName: "stringParam",
			expected:  "test_value",
		},
		{
			name:         "existing string parameter with default",
			paramName:    "stringParam",
			defaultValue: []string{"default"},
			expected:     "test_value",
		},
		{
			name:      "empty string parameter",
			paramName: "emptyString",
			expected:  "",
		},
		{
			name:      "non-existent parameter without default",
			paramName: "nonExistent",
			expected:  "",
		},
		{
			name:         "non-existent parameter with default",
			paramName:    "nonExistent",
			defaultValue: []string{"default_value"},
			expected:     "default_value",
		},
		{
			name:      "wrong type parameter (int) without default",
			paramName: "intParam",
			expected:  "",
		},
		{
			name:         "wrong type parameter (int) with default",
			paramName:    "intParam",
			defaultValue: []string{"fallback"},
			expected:     "fallback",
		},
		{
			name:      "wrong type parameter (bool) without default",
			paramName: "boolParam",
			expected:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := annotation.GetString(tt.paramName, tt.defaultValue...)
			if result != tt.expected {
				t.Errorf("GetString(%q, %v) = %q, want %q", tt.paramName, tt.defaultValue, result, tt.expected)
			}
		})
	}
}

func TestParsedAnnotation_GetBool(t *testing.T) {
	annotation := &ParsedAnnotation{
		Parameters: map[string]interface{}{
			"boolTrue":    true,
			"boolFalse":   false,
			"stringParam": "test",
			"intParam":    42,
		},
	}

	tests := []struct {
		name         string
		paramName    string
		defaultValue []bool
		expected     bool
	}{
		{
			name:      "existing bool parameter (true)",
			paramName: "boolTrue",
			expected:  true,
		},
		{
			name:      "existing bool parameter (false)",
			paramName: "boolFalse",
			expected:  false,
		},
		{
			name:         "existing bool parameter with default",
			paramName:    "boolTrue",
			defaultValue: []bool{false},
			expected:     true,
		},
		{
			name:      "non-existent parameter without default",
			paramName: "nonExistent",
			expected:  false,
		},
		{
			name:         "non-existent parameter with default (true)",
			paramName:    "nonExistent",
			defaultValue: []bool{true},
			expected:     true,
		},
		{
			name:         "non-existent parameter with default (false)",
			paramName:    "nonExistent",
			defaultValue: []bool{false},
			expected:     false,
		},
		{
			name:      "wrong type parameter (string) without default",
			paramName: "stringParam",
			expected:  false,
		},
		{
			name:         "wrong type parameter (string) with default",
			paramName:    "stringParam",
			defaultValue: []bool{true},
			expected:     true,
		},
		{
			name:      "wrong type parameter (int) without default",
			paramName: "intParam",
			expected:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := annotation.GetBool(tt.paramName, tt.defaultValue...)
			if result != tt.expected {
				t.Errorf("GetBool(%q, %v) = %t, want %t", tt.paramName, tt.defaultValue, result, tt.expected)
			}
		})
	}
}

func TestParsedAnnotation_GetInt(t *testing.T) {
	annotation := &ParsedAnnotation{
		Parameters: map[string]interface{}{
			"intParam":    42,
			"zeroInt":     0,
			"negativeInt": -10,
			"stringParam": "test",
			"boolParam":   true,
		},
	}

	tests := []struct {
		name         string
		paramName    string
		defaultValue []int
		expected     int
	}{
		{
			name:      "existing int parameter",
			paramName: "intParam",
			expected:  42,
		},
		{
			name:      "existing zero int parameter",
			paramName: "zeroInt",
			expected:  0,
		},
		{
			name:      "existing negative int parameter",
			paramName: "negativeInt",
			expected:  -10,
		},
		{
			name:         "existing int parameter with default",
			paramName:    "intParam",
			defaultValue: []int{100},
			expected:     42,
		},
		{
			name:      "non-existent parameter without default",
			paramName: "nonExistent",
			expected:  0,
		},
		{
			name:         "non-existent parameter with default",
			paramName:    "nonExistent",
			defaultValue: []int{99},
			expected:     99,
		},
		{
			name:      "wrong type parameter (string) without default",
			paramName: "stringParam",
			expected:  0,
		},
		{
			name:         "wrong type parameter (string) with default",
			paramName:    "stringParam",
			defaultValue: []int{123},
			expected:     123,
		},
		{
			name:      "wrong type parameter (bool) without default",
			paramName: "boolParam",
			expected:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := annotation.GetInt(tt.paramName, tt.defaultValue...)
			if result != tt.expected {
				t.Errorf("GetInt(%q, %v) = %d, want %d", tt.paramName, tt.defaultValue, result, tt.expected)
			}
		})
	}
}

func TestParsedAnnotation_GetStringSlice(t *testing.T) {
	annotation := &ParsedAnnotation{
		Parameters: map[string]interface{}{
			"stringSlice":      []string{"a", "b", "c"},
			"emptySlice":       []string{},
			"singleItemSlice":  []string{"single"},
			"stringParam":      "test",
			"intParam":         42,
		},
	}

	tests := []struct {
		name         string
		paramName    string
		defaultValue [][]string
		expected     []string
	}{
		{
			name:      "existing string slice parameter",
			paramName: "stringSlice",
			expected:  []string{"a", "b", "c"},
		},
		{
			name:      "existing empty slice parameter",
			paramName: "emptySlice",
			expected:  []string{},
		},
		{
			name:      "existing single item slice parameter",
			paramName: "singleItemSlice",
			expected:  []string{"single"},
		},
		{
			name:         "existing slice parameter with default",
			paramName:    "stringSlice",
			defaultValue: [][]string{{"default1", "default2"}},
			expected:     []string{"a", "b", "c"},
		},
		{
			name:      "non-existent parameter without default",
			paramName: "nonExistent",
			expected:  nil,
		},
		{
			name:         "non-existent parameter with default",
			paramName:    "nonExistent",
			defaultValue: [][]string{{"default1", "default2"}},
			expected:     []string{"default1", "default2"},
		},
		{
			name:         "non-existent parameter with empty default",
			paramName:    "nonExistent",
			defaultValue: [][]string{{}},
			expected:     []string{},
		},
		{
			name:      "wrong type parameter (string) without default",
			paramName: "stringParam",
			expected:  nil,
		},
		{
			name:         "wrong type parameter (string) with default",
			paramName:    "stringParam",
			defaultValue: [][]string{{"fallback1", "fallback2"}},
			expected:     []string{"fallback1", "fallback2"},
		},
		{
			name:      "wrong type parameter (int) without default",
			paramName: "intParam",
			expected:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := annotation.GetStringSlice(tt.paramName, tt.defaultValue...)
			
			// Compare slices
			if !stringSlicesEqual(result, tt.expected) {
				t.Errorf("GetStringSlice(%q, %v) = %v, want %v", tt.paramName, tt.defaultValue, result, tt.expected)
			}
		})
	}
}

func TestParsedAnnotation_HasParameter(t *testing.T) {
	annotation := &ParsedAnnotation{
		Parameters: map[string]interface{}{
			"existingParam": "value",
			"emptyString":   "",
			"zeroInt":       0,
			"falseBool":     false,
			"emptySlice":    []string{},
		},
	}

	tests := []struct {
		name      string
		paramName string
		expected  bool
	}{
		{
			name:      "existing parameter with value",
			paramName: "existingParam",
			expected:  true,
		},
		{
			name:      "existing parameter with empty string",
			paramName: "emptyString",
			expected:  true,
		},
		{
			name:      "existing parameter with zero int",
			paramName: "zeroInt",
			expected:  true,
		},
		{
			name:      "existing parameter with false bool",
			paramName: "falseBool",
			expected:  true,
		},
		{
			name:      "existing parameter with empty slice",
			paramName: "emptySlice",
			expected:  true,
		},
		{
			name:      "non-existent parameter",
			paramName: "nonExistent",
			expected:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := annotation.HasParameter(tt.paramName)
			if result != tt.expected {
				t.Errorf("HasParameter(%q) = %t, want %t", tt.paramName, result, tt.expected)
			}
		})
	}
}

// Helper function to compare string slices
func stringSlicesEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func TestTypeConversionUtilities(t *testing.T) {
	t.Run("ConvertToString", func(t *testing.T) {
		tests := []struct {
			name     string
			input    interface{}
			expected string
		}{
			{"string", "test", "test"},
			{"int", 42, "42"},
			{"bool true", true, "true"},
			{"bool false", false, "false"},
			{"float", 3.14, "3.14"},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				result, err := ConvertToString(tt.input)
				if err != nil {
					t.Errorf("ConvertToString(%v) returned error: %v", tt.input, err)
				}
				if result != tt.expected {
					t.Errorf("ConvertToString(%v) = %q, want %q", tt.input, result, tt.expected)
				}
			})
		}
	})

	t.Run("ConvertToBool", func(t *testing.T) {
		tests := []struct {
			name      string
			input     interface{}
			expected  bool
			shouldErr bool
		}{
			{"bool true", true, true, false},
			{"bool false", false, false, false},
			{"string true", "true", true, false},
			{"string false", "false", false, false},
			{"string yes", "yes", true, false},
			{"string no", "no", false, false},
			{"string 1", "1", true, false},
			{"string 0", "0", false, false},
			{"int non-zero", 42, true, false},
			{"int zero", 0, false, false},
			{"float non-zero", 3.14, true, false},
			{"invalid string", "invalid", false, true},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				result, err := ConvertToBool(tt.input)
				if tt.shouldErr {
					if err == nil {
						t.Errorf("ConvertToBool(%v) should have returned error", tt.input)
					}
				} else {
					if err != nil {
						t.Errorf("ConvertToBool(%v) returned unexpected error: %v", tt.input, err)
					}
					if result != tt.expected {
						t.Errorf("ConvertToBool(%v) = %t, want %t", tt.input, result, tt.expected)
					}
				}
			})
		}
	})

	t.Run("ConvertToInt", func(t *testing.T) {
		tests := []struct {
			name      string
			input     interface{}
			expected  int
			shouldErr bool
		}{
			{"int", 42, 42, false},
			{"int64", int64(42), 42, false},
			{"float64", 42.0, 42, false},
			{"string valid", "42", 42, false},
			{"bool true", true, 1, false},
			{"bool false", false, 0, false},
			{"string invalid", "invalid", 0, true},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				result, err := ConvertToInt(tt.input)
				if tt.shouldErr {
					if err == nil {
						t.Errorf("ConvertToInt(%v) should have returned error", tt.input)
					}
				} else {
					if err != nil {
						t.Errorf("ConvertToInt(%v) returned unexpected error: %v", tt.input, err)
					}
					if result != tt.expected {
						t.Errorf("ConvertToInt(%v) = %d, want %d", tt.input, result, tt.expected)
					}
				}
			})
		}
	})

	t.Run("ConvertToStringSlice", func(t *testing.T) {
		tests := []struct {
			name     string
			input    interface{}
			expected []string
		}{
			{"string slice", []string{"a", "b", "c"}, []string{"a", "b", "c"}},
			{"single string", "test", []string{"test"}},
			{"comma separated", "a,b,c", []string{"a", "b", "c"}},
			{"comma separated with spaces", "a, b, c", []string{"a", "b", "c"}},
			{"empty string", "", []string{}},
			{"interface slice", []interface{}{"a", 42, true}, []string{"a", "42", "true"}},
			{"single int", 42, []string{"42"}},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				result, err := ConvertToStringSlice(tt.input)
				if err != nil {
					t.Errorf("ConvertToStringSlice(%v) returned error: %v", tt.input, err)
				}
				if !stringSlicesEqual(result, tt.expected) {
					t.Errorf("ConvertToStringSlice(%v) = %v, want %v", tt.input, result, tt.expected)
				}
			})
		}
	})
}

func TestParsedAnnotation_ConversionGetters(t *testing.T) {
	annotation := &ParsedAnnotation{
		Parameters: map[string]interface{}{
			"stringParam":     "test_value",
			"intAsString":     "42",
			"boolAsString":    "true",
			"sliceAsString":   "a,b,c",
			"intParam":        42,
			"boolParam":       true,
			"floatParam":      3.14,
		},
	}

	t.Run("GetStringWithConversion", func(t *testing.T) {
		tests := []struct {
			name         string
			paramName    string
			defaultValue []string
			expected     string
		}{
			{"existing string", "stringParam", nil, "test_value"},
			{"int to string", "intParam", nil, "42"},
			{"bool to string", "boolParam", nil, "true"},
			{"float to string", "floatParam", nil, "3.14"},
			{"non-existent with default", "nonExistent", []string{"default"}, "default"},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				result := annotation.GetStringWithConversion(tt.paramName, tt.defaultValue...)
				if result != tt.expected {
					t.Errorf("GetStringWithConversion(%q, %v) = %q, want %q", tt.paramName, tt.defaultValue, result, tt.expected)
				}
			})
		}
	})

	t.Run("GetBoolWithConversion", func(t *testing.T) {
		tests := []struct {
			name         string
			paramName    string
			defaultValue []bool
			expected     bool
		}{
			{"existing bool", "boolParam", nil, true},
			{"string to bool", "boolAsString", nil, true},
			{"int to bool", "intParam", nil, true},
			{"non-existent with default", "nonExistent", []bool{true}, true},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				result := annotation.GetBoolWithConversion(tt.paramName, tt.defaultValue...)
				if result != tt.expected {
					t.Errorf("GetBoolWithConversion(%q, %v) = %t, want %t", tt.paramName, tt.defaultValue, result, tt.expected)
				}
			})
		}
	})

	t.Run("GetIntWithConversion", func(t *testing.T) {
		tests := []struct {
			name         string
			paramName    string
			defaultValue []int
			expected     int
		}{
			{"existing int", "intParam", nil, 42},
			{"string to int", "intAsString", nil, 42},
			{"bool to int", "boolParam", nil, 1},
			{"non-existent with default", "nonExistent", []int{99}, 99},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				result := annotation.GetIntWithConversion(tt.paramName, tt.defaultValue...)
				if result != tt.expected {
					t.Errorf("GetIntWithConversion(%q, %v) = %d, want %d", tt.paramName, tt.defaultValue, result, tt.expected)
				}
			})
		}
	})

	t.Run("GetStringSliceWithConversion", func(t *testing.T) {
		tests := []struct {
			name         string
			paramName    string
			defaultValue [][]string
			expected     []string
		}{
			{"string to slice", "sliceAsString", nil, []string{"a", "b", "c"}},
			{"single string to slice", "stringParam", nil, []string{"test_value"}},
			{"non-existent with default", "nonExistent", [][]string{{"default1", "default2"}}, []string{"default1", "default2"}},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				result := annotation.GetStringSliceWithConversion(tt.paramName, tt.defaultValue...)
				if !stringSlicesEqual(result, tt.expected) {
					t.Errorf("GetStringSliceWithConversion(%q, %v) = %v, want %v", tt.paramName, tt.defaultValue, result, tt.expected)
				}
			})
		}
	})
}

func TestTypeSafeGettersIntegration(t *testing.T) {
	// Create a registry with test schemas
	registry := NewRegistry()
	err := RegisterBuiltinSchemas(registry)
	if err != nil {
		t.Fatalf("failed to register builtin schemas: %v", err)
	}

	parser := NewParser(registry)
	location := SourceLocation{File: "test.go", Line: 10, Column: 1}

	tests := []struct {
		name           string
		annotationText string
		testFunc       func(*testing.T, *ParsedAnnotation)
	}{
		{
			name:           "core annotation with type-safe getters",
			annotationText: "//axon::core -Mode=Transient -Init=Background",
			testFunc: func(t *testing.T, annotation *ParsedAnnotation) {
				// Test basic getters
				mode := annotation.GetString("Mode", "Singleton")
				if mode != "Transient" {
					t.Errorf("expected Mode='Transient', got %q", mode)
				}

				init := annotation.GetString("Init", "Same")
				if init != "Background" {
					t.Errorf("expected Init='Background', got %q", init)
				}

				// Test parameter existence
				if !annotation.HasParameter("Mode") {
					t.Error("expected Mode parameter to exist")
				}

				if !annotation.HasParameter("Init") {
					t.Error("expected Init parameter to exist")
				}

				if annotation.HasParameter("NonExistent") {
					t.Error("expected NonExistent parameter to not exist")
				}

				// Test default values
				manual := annotation.GetString("Manual", "DefaultModule")
				if manual != "DefaultModule" {
					t.Errorf("expected Manual default='DefaultModule', got %q", manual)
				}
			},
		},
		{
			name:           "route annotation with middleware slice",
			annotationText: "//axon::route GET /users/{id:int} -Middleware=Auth,Logging -PassContext",
			testFunc: func(t *testing.T, annotation *ParsedAnnotation) {
				// Test string getters
				method := annotation.GetString("method", "GET")
				if method != "GET" {
					t.Errorf("expected method='GET', got %q", method)
				}

				path := annotation.GetString("path", "/")
				if path != "/users/{id:int}" {
					t.Errorf("expected path='/users/{id:int}', got %q", path)
				}

				// Test string slice getter
				middleware := annotation.GetStringSlice("Middleware", []string{})
				expectedMiddleware := []string{"Auth", "Logging"}
				if !stringSlicesEqual(middleware, expectedMiddleware) {
					t.Errorf("expected Middleware=%v, got %v", expectedMiddleware, middleware)
				}

				// Test boolean getter
				passContext := annotation.GetBool("PassContext", false)
				if !passContext {
					t.Error("expected PassContext=true")
				}

				// Test conversion getters
				methodWithConversion := annotation.GetStringWithConversion("method", "POST")
				if methodWithConversion != "GET" {
					t.Errorf("expected method with conversion='GET', got %q", methodWithConversion)
				}

				passContextWithConversion := annotation.GetBoolWithConversion("PassContext", false)
				if !passContextWithConversion {
					t.Error("expected PassContext with conversion=true")
				}
			},
		},
		{
			name:           "annotation with mixed parameter types",
			annotationText: "//axon::core -Mode=Singleton",
			testFunc: func(t *testing.T, annotation *ParsedAnnotation) {
				// Test that we can access parameters with different getter methods
				mode := annotation.GetString("Mode", "Transient")
				if mode != "Singleton" {
					t.Errorf("expected Mode='Singleton', got %q", mode)
				}

				// Test conversion from string to other types (should fail gracefully)
				modeAsBool := annotation.GetBoolWithConversion("Mode", true)
				// "Singleton" should not convert to bool, so should return default
				if !modeAsBool {
					t.Error("expected default bool value when conversion fails")
				}

				// Test accessing non-existent parameter with defaults
				nonExistent := annotation.GetString("NonExistent", "default_value")
				if nonExistent != "default_value" {
					t.Errorf("expected default value, got %q", nonExistent)
				}

				nonExistentBool := annotation.GetBool("NonExistent", true)
				if !nonExistentBool {
					t.Error("expected default bool value")
				}

				nonExistentSlice := annotation.GetStringSlice("NonExistent", []string{"default"})
				if !stringSlicesEqual(nonExistentSlice, []string{"default"}) {
					t.Errorf("expected default slice value, got %v", nonExistentSlice)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			annotation, err := parser.ParseAnnotation(tt.annotationText, location)
			if err != nil {
				t.Fatalf("unexpected error parsing annotation: %v", err)
			}

			tt.testFunc(t, annotation)
		})
	}
}