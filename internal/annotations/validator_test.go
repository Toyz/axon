package annotations

import (
	"errors"
	"testing"
)

func TestValidator_Validate(t *testing.T) {
	validator := NewValidator()

	tests := []struct {
		name        string
		annotation  *ParsedAnnotation
		schema      AnnotationSchema
		expectError bool
		errorType   ErrorCode
	}{
		{
			name: "valid annotation with all required parameters",
			annotation: &ParsedAnnotation{
				Type: CoreAnnotation,
				Parameters: map[string]interface{}{
					"Mode": "Singleton",
				},
				Location: SourceLocation{File: "test.go", Line: 1, Column: 1},
			},
			schema: AnnotationSchema{
				Type: CoreAnnotation,
				Parameters: map[string]ParameterSpec{
					"Mode": {
						Type:     StringType,
						Required: true,
					},
				},
			},
			expectError: false,
		},
		{
			name: "missing required parameter",
			annotation: &ParsedAnnotation{
				Type:       CoreAnnotation,
				Parameters: map[string]interface{}{},
				Location:   SourceLocation{File: "test.go", Line: 1, Column: 1},
			},
			schema: AnnotationSchema{
				Type: CoreAnnotation,
				Parameters: map[string]ParameterSpec{
					"Mode": {
						Type:     StringType,
						Required: true,
					},
				},
			},
			expectError: true,
			errorType:   ValidationErrorCode,
		},
		{
			name: "unknown parameter",
			annotation: &ParsedAnnotation{
				Type: CoreAnnotation,
				Parameters: map[string]interface{}{
					"UnknownParam": "value",
				},
				Location: SourceLocation{File: "test.go", Line: 1, Column: 1},
			},
			schema: AnnotationSchema{
				Type:       CoreAnnotation,
				Parameters: map[string]ParameterSpec{},
			},
			expectError: true,
			errorType:   ValidationErrorCode,
		},
		{
			name: "wrong parameter type",
			annotation: &ParsedAnnotation{
				Type: CoreAnnotation,
				Parameters: map[string]interface{}{
					"Mode": 123, // Should be string
				},
				Location: SourceLocation{File: "test.go", Line: 1, Column: 1},
			},
			schema: AnnotationSchema{
				Type: CoreAnnotation,
				Parameters: map[string]ParameterSpec{
					"Mode": {
						Type:     StringType,
						Required: true,
					},
				},
			},
			expectError: true,
			errorType:   ValidationErrorCode,
		},
		{
			name: "custom validator failure",
			annotation: &ParsedAnnotation{
				Type: CoreAnnotation,
				Parameters: map[string]interface{}{
					"Mode": "InvalidMode",
				},
				Location: SourceLocation{File: "test.go", Line: 1, Column: 1},
			},
			schema: AnnotationSchema{
				Type: CoreAnnotation,
				Parameters: map[string]ParameterSpec{
					"Mode": {
						Type:     StringType,
						Required: true,
						Validator: func(v interface{}) error {
							mode := v.(string)
							if mode != "Singleton" && mode != "Transient" {
								return errors.New("must be 'Singleton' or 'Transient'")
							}
							return nil
						},
					},
				},
			},
			expectError: true,
			errorType:   ValidationErrorCode,
		},
		{
			name: "custom validator success",
			annotation: &ParsedAnnotation{
				Type: CoreAnnotation,
				Parameters: map[string]interface{}{
					"Mode": "Singleton",
				},
				Location: SourceLocation{File: "test.go", Line: 1, Column: 1},
			},
			schema: AnnotationSchema{
				Type: CoreAnnotation,
				Parameters: map[string]ParameterSpec{
					"Mode": {
						Type:     StringType,
						Required: true,
						Validator: func(v interface{}) error {
							mode := v.(string)
							if mode != "Singleton" && mode != "Transient" {
								return errors.New("must be 'Singleton' or 'Transient'")
							}
							return nil
						},
					},
				},
			},
			expectError: false,
		},
		{
			name: "annotation-level custom validator failure",
			annotation: &ParsedAnnotation{
				Type: RouteAnnotation,
				Parameters: map[string]interface{}{
					"method": "GET",
					"path":   "/users",
				},
				Location: SourceLocation{File: "test.go", Line: 1, Column: 1},
			},
			schema: AnnotationSchema{
				Type: RouteAnnotation,
				Parameters: map[string]ParameterSpec{
					"method": {Type: StringType, Required: true},
					"path":   {Type: StringType, Required: true},
				},
				Validators: []CustomValidator{
					func(annotation *ParsedAnnotation) error {
						// Custom rule: GET requests cannot have body
						if annotation.Parameters["method"] == "GET" && annotation.Parameters["body"] != nil {
							return errors.New("GET requests cannot have a body")
						}
						return nil
					},
				},
			},
			expectError: false, // This should pass since there's no body parameter
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.Validate(tt.annotation, tt.schema)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
					return
				}

				// Check error type if specified
				if tt.errorType != 0 {
					var annotationErr AnnotationError
					if errors.As(err, &annotationErr) {
						if annotationErr.Code() != tt.errorType {
							t.Errorf("expected error code %v, got %v", tt.errorType, annotationErr.Code())
						}
					} else {
						// Check for MultipleValidationErrors
						var multiErr *MultipleValidationErrors
						if errors.As(err, &multiErr) && len(multiErr.Errors) > 0 {
							if annotationErr, ok := multiErr.Errors[0].(AnnotationError); ok {
								if annotationErr.Code() != tt.errorType {
									t.Errorf("expected error code %v, got %v", tt.errorType, annotationErr.Code())
								}
							}
						}
					}
				}
			} else {
				if err != nil {
					t.Errorf("expected no error but got: %v", err)
				}
			}
		})
	}
}

func TestValidator_ApplyDefaults(t *testing.T) {
	validator := NewValidator()

	tests := []struct {
		name       string
		annotation *ParsedAnnotation
		schema     AnnotationSchema
		expected   map[string]interface{}
	}{
		{
			name: "apply default for missing optional parameter",
			annotation: &ParsedAnnotation{
				Type:       CoreAnnotation,
				Parameters: map[string]interface{}{},
			},
			schema: AnnotationSchema{
				Type: CoreAnnotation,
				Parameters: map[string]ParameterSpec{
					"Mode": {
						Type:         StringType,
						Required:     false,
						DefaultValue: "Singleton",
					},
				},
			},
			expected: map[string]interface{}{
				"Mode": "Singleton",
			},
		},
		{
			name: "don't override existing parameter",
			annotation: &ParsedAnnotation{
				Type: CoreAnnotation,
				Parameters: map[string]interface{}{
					"Mode": "Transient",
				},
			},
			schema: AnnotationSchema{
				Type: CoreAnnotation,
				Parameters: map[string]ParameterSpec{
					"Mode": {
						Type:         StringType,
						Required:     false,
						DefaultValue: "Singleton",
					},
				},
			},
			expected: map[string]interface{}{
				"Mode": "Transient",
			},
		},
		{
			name: "no default value specified",
			annotation: &ParsedAnnotation{
				Type:       CoreAnnotation,
				Parameters: map[string]interface{}{},
			},
			schema: AnnotationSchema{
				Type: CoreAnnotation,
				Parameters: map[string]ParameterSpec{
					"Mode": {
						Type:     StringType,
						Required: false,
						// No DefaultValue
					},
				},
			},
			expected: map[string]interface{}{},
		},
		{
			name: "nil parameters map",
			annotation: &ParsedAnnotation{
				Type:       CoreAnnotation,
				Parameters: nil,
			},
			schema: AnnotationSchema{
				Type: CoreAnnotation,
				Parameters: map[string]ParameterSpec{
					"Mode": {
						Type:         StringType,
						Required:     false,
						DefaultValue: "Singleton",
					},
				},
			},
			expected: map[string]interface{}{
				"Mode": "Singleton",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ApplyDefaults(tt.annotation, tt.schema)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if len(tt.annotation.Parameters) != len(tt.expected) {
				t.Errorf("expected %d parameters, got %d", len(tt.expected), len(tt.annotation.Parameters))
				return
			}

			for key, expectedValue := range tt.expected {
				actualValue, exists := tt.annotation.Parameters[key]
				if !exists {
					t.Errorf("expected parameter %s to exist", key)
					continue
				}

				if actualValue != expectedValue {
					t.Errorf("expected parameter %s to be %v, got %v", key, expectedValue, actualValue)
				}
			}
		})
	}
}

func TestValidator_TransformParameters(t *testing.T) {
	validator := NewValidator()

	tests := []struct {
		name        string
		annotation  *ParsedAnnotation
		schema      AnnotationSchema
		expected    map[string]interface{}
		expectError bool
	}{
		{
			name: "transform string to int",
			annotation: &ParsedAnnotation{
				Type: CoreAnnotation,
				Parameters: map[string]interface{}{
					"Port": "8080",
				},
				Location: SourceLocation{File: "test.go", Line: 1, Column: 1},
			},
			schema: AnnotationSchema{
				Type: CoreAnnotation,
				Parameters: map[string]ParameterSpec{
					"Port": {Type: IntType},
				},
			},
			expected: map[string]interface{}{
				"Port": 8080,
			},
			expectError: false,
		},
		{
			name: "transform string to bool",
			annotation: &ParsedAnnotation{
				Type: CoreAnnotation,
				Parameters: map[string]interface{}{
					"Enabled": "true",
				},
				Location: SourceLocation{File: "test.go", Line: 1, Column: 1},
			},
			schema: AnnotationSchema{
				Type: CoreAnnotation,
				Parameters: map[string]ParameterSpec{
					"Enabled": {Type: BoolType},
				},
			},
			expected: map[string]interface{}{
				"Enabled": true,
			},
			expectError: false,
		},
		{
			name: "transform string to string slice",
			annotation: &ParsedAnnotation{
				Type: CoreAnnotation,
				Parameters: map[string]interface{}{
					"Tags": "auth,logging,cache",
				},
				Location: SourceLocation{File: "test.go", Line: 1, Column: 1},
			},
			schema: AnnotationSchema{
				Type: CoreAnnotation,
				Parameters: map[string]ParameterSpec{
					"Tags": {Type: StringSliceType},
				},
			},
			expected: map[string]interface{}{
				"Tags": []string{"auth", "logging", "cache"},
			},
			expectError: false,
		},
		{
			name: "transform empty string to empty slice",
			annotation: &ParsedAnnotation{
				Type: CoreAnnotation,
				Parameters: map[string]interface{}{
					"Tags": "",
				},
				Location: SourceLocation{File: "test.go", Line: 1, Column: 1},
			},
			schema: AnnotationSchema{
				Type: CoreAnnotation,
				Parameters: map[string]ParameterSpec{
					"Tags": {Type: StringSliceType},
				},
			},
			expected: map[string]interface{}{
				"Tags": []string{},
			},
			expectError: false,
		},
		{
			name: "invalid string to int conversion",
			annotation: &ParsedAnnotation{
				Type: CoreAnnotation,
				Parameters: map[string]interface{}{
					"Port": "invalid",
				},
				Location: SourceLocation{File: "test.go", Line: 1, Column: 1},
			},
			schema: AnnotationSchema{
				Type: CoreAnnotation,
				Parameters: map[string]ParameterSpec{
					"Port": {Type: IntType},
				},
			},
			expectError: true,
		},
		{
			name: "already correct type - no transformation needed",
			annotation: &ParsedAnnotation{
				Type: CoreAnnotation,
				Parameters: map[string]interface{}{
					"Port": 8080,
				},
				Location: SourceLocation{File: "test.go", Line: 1, Column: 1},
			},
			schema: AnnotationSchema{
				Type: CoreAnnotation,
				Parameters: map[string]ParameterSpec{
					"Port": {Type: IntType},
				},
			},
			expected: map[string]interface{}{
				"Port": 8080,
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.TransformParameters(tt.annotation, tt.schema)

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

			for key, expectedValue := range tt.expected {
				actualValue, exists := tt.annotation.Parameters[key]
				if !exists {
					t.Errorf("expected parameter %s to exist", key)
					continue
				}

				// For slice comparison, we need to compare elements
				if expectedSlice, ok := expectedValue.([]string); ok {
					actualSlice, ok := actualValue.([]string)
					if !ok {
						t.Errorf("expected parameter %s to be []string, got %T", key, actualValue)
						continue
					}

					if len(expectedSlice) != len(actualSlice) {
						t.Errorf("expected parameter %s to have %d elements, got %d", key, len(expectedSlice), len(actualSlice))
						continue
					}

					for i, expected := range expectedSlice {
						if actualSlice[i] != expected {
							t.Errorf("expected parameter %s[%d] to be %s, got %s", key, i, expected, actualSlice[i])
						}
					}
				} else {
					if actualValue != expectedValue {
						t.Errorf("expected parameter %s to be %v, got %v", key, expectedValue, actualValue)
					}
				}
			}
		})
	}
}

func TestValidator_validateParameterType(t *testing.T) {
	validator := &validator{}
	location := SourceLocation{File: "test.go", Line: 1, Column: 1}

	tests := []struct {
		name         string
		paramName    string
		expectedType ParameterType
		value        interface{}
		expectError  bool
	}{
		{"valid string", "param", StringType, "value", false},
		{"invalid string", "param", StringType, 123, true},
		{"valid bool", "param", BoolType, true, false},
		{"invalid bool", "param", BoolType, "true", true},
		{"valid int", "param", IntType, 42, false},
		{"invalid int", "param", IntType, "42", true},
		{"valid string slice", "param", StringSliceType, []string{"a", "b"}, false},
		{"invalid string slice", "param", StringSliceType, "a,b", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.validateParameterType(tt.paramName, tt.expectedType, tt.value, location)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

func TestValidator_convertFromString(t *testing.T) {
	validator := &validator{}

	tests := []struct {
		name         string
		input        string
		targetType   ParameterType
		expected     interface{}
		expectError  bool
	}{
		{"string to string", "hello", StringType, "hello", false},
		{"string to bool true", "true", BoolType, true, false},
		{"string to bool false", "false", BoolType, false, false},
		{"string to bool invalid", "invalid", BoolType, nil, true},
		{"string to int", "42", IntType, 42, false},
		{"string to int invalid", "invalid", IntType, nil, true},
		{"string to slice", "a,b,c", StringSliceType, []string{"a", "b", "c"}, false},
		{"string to slice with spaces", "a, b , c", StringSliceType, []string{"a", "b", "c"}, false},
		{"empty string to slice", "", StringSliceType, []string{}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := validator.convertFromString(tt.input, tt.targetType)

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

			// Special handling for slice comparison
			if expectedSlice, ok := tt.expected.([]string); ok {
				resultSlice, ok := result.([]string)
				if !ok {
					t.Errorf("expected []string, got %T", result)
					return
				}

				if len(expectedSlice) != len(resultSlice) {
					t.Errorf("expected %d elements, got %d", len(expectedSlice), len(resultSlice))
					return
				}

				for i, expected := range expectedSlice {
					if resultSlice[i] != expected {
						t.Errorf("expected element %d to be %s, got %s", i, expected, resultSlice[i])
					}
				}
			} else {
				if result != tt.expected {
					t.Errorf("expected %v, got %v", tt.expected, result)
				}
			}
		})
	}
}
func 
TestParsedAnnotation_TypeSafeGetters(t *testing.T) {
	annotation := &ParsedAnnotation{
		Type: CoreAnnotation,
		Parameters: map[string]interface{}{
			"StringParam": "test_value",
			"BoolParam":   true,
			"IntParam":    42,
			"SliceParam":  []string{"a", "b", "c"},
		},
	}

	// Test GetString
	t.Run("GetString", func(t *testing.T) {
		// Existing parameter
		if got := annotation.GetString("StringParam"); got != "test_value" {
			t.Errorf("GetString() = %v, want %v", got, "test_value")
		}

		// Non-existing parameter without default
		if got := annotation.GetString("NonExistent"); got != "" {
			t.Errorf("GetString() = %v, want empty string", got)
		}

		// Non-existing parameter with default
		if got := annotation.GetString("NonExistent", "default"); got != "default" {
			t.Errorf("GetString() = %v, want %v", got, "default")
		}

		// Wrong type parameter
		if got := annotation.GetString("IntParam"); got != "" {
			t.Errorf("GetString() = %v, want empty string for wrong type", got)
		}
	})

	// Test GetBool
	t.Run("GetBool", func(t *testing.T) {
		// Existing parameter
		if got := annotation.GetBool("BoolParam"); got != true {
			t.Errorf("GetBool() = %v, want %v", got, true)
		}

		// Non-existing parameter without default
		if got := annotation.GetBool("NonExistent"); got != false {
			t.Errorf("GetBool() = %v, want false", got)
		}

		// Non-existing parameter with default
		if got := annotation.GetBool("NonExistent", true); got != true {
			t.Errorf("GetBool() = %v, want %v", got, true)
		}

		// Wrong type parameter
		if got := annotation.GetBool("StringParam"); got != false {
			t.Errorf("GetBool() = %v, want false for wrong type", got)
		}
	})

	// Test GetInt
	t.Run("GetInt", func(t *testing.T) {
		// Existing parameter
		if got := annotation.GetInt("IntParam"); got != 42 {
			t.Errorf("GetInt() = %v, want %v", got, 42)
		}

		// Non-existing parameter without default
		if got := annotation.GetInt("NonExistent"); got != 0 {
			t.Errorf("GetInt() = %v, want 0", got)
		}

		// Non-existing parameter with default
		if got := annotation.GetInt("NonExistent", 100); got != 100 {
			t.Errorf("GetInt() = %v, want %v", got, 100)
		}

		// Wrong type parameter
		if got := annotation.GetInt("StringParam"); got != 0 {
			t.Errorf("GetInt() = %v, want 0 for wrong type", got)
		}
	})

	// Test GetStringSlice
	t.Run("GetStringSlice", func(t *testing.T) {
		// Existing parameter
		got := annotation.GetStringSlice("SliceParam")
		expected := []string{"a", "b", "c"}
		if len(got) != len(expected) {
			t.Errorf("GetStringSlice() length = %v, want %v", len(got), len(expected))
		}
		for i, v := range expected {
			if got[i] != v {
				t.Errorf("GetStringSlice()[%d] = %v, want %v", i, got[i], v)
			}
		}

		// Non-existing parameter without default
		if got := annotation.GetStringSlice("NonExistent"); got != nil {
			t.Errorf("GetStringSlice() = %v, want nil", got)
		}

		// Non-existing parameter with default
		defaultSlice := []string{"default"}
		if got := annotation.GetStringSlice("NonExistent", defaultSlice); len(got) != 1 || got[0] != "default" {
			t.Errorf("GetStringSlice() = %v, want %v", got, defaultSlice)
		}

		// Wrong type parameter
		if got := annotation.GetStringSlice("StringParam"); got != nil {
			t.Errorf("GetStringSlice() = %v, want nil for wrong type", got)
		}
	})

	// Test HasParameter
	t.Run("HasParameter", func(t *testing.T) {
		if !annotation.HasParameter("StringParam") {
			t.Errorf("HasParameter() = false, want true for existing parameter")
		}

		if annotation.HasParameter("NonExistent") {
			t.Errorf("HasParameter() = true, want false for non-existing parameter")
		}
	})
}