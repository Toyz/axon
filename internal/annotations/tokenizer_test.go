package annotations

import (
	"strings"
	"testing"
)

// Comprehensive tokenizer tests for edge cases and various formats
func TestTokenizerComprehensive(t *testing.T) {
	registry := NewRegistry()
	err := RegisterBuiltinSchemas(registry)
	if err != nil {
		t.Fatalf("failed to register builtin schemas: %v", err)
	}
	parser := NewParser(registry)
	location := SourceLocation{File: "tokenizer_test.go", Line: 1, Column: 1}

	tests := []struct {
		name           string
		input          string
		expectError    bool
		expectedType   AnnotationType
		expectedParams map[string]interface{}
	}{
		// Basic tokenization tests
		{
			name:         "simple annotation",
			input:        "//axon::core",
			expectError:  false,
			expectedType: CoreAnnotation,
			expectedParams: map[string]interface{}{
				"Mode": "Singleton", // Default value
				"Init": "Same",      // Default value
			},
		},
		
		// Whitespace handling
		{
			name:         "extra spaces around parameters",
			input:        "//axon::core   -Mode=Transient   -Init=Background   ",
			expectError:  false,
			expectedType: CoreAnnotation,
			expectedParams: map[string]interface{}{
				"Mode": "Transient",
				"Init": "Background",
			},
		},
		{
			name:         "tabs and mixed whitespace",
			input:        "//axon::core\t-Mode=Transient\t\t-Init=Background",
			expectError:  false,
			expectedType: CoreAnnotation,
			expectedParams: map[string]interface{}{
				"Mode": "Transient",
				"Init": "Background",
			},
		},
		
		// Quote handling
		{
			name:         "double quoted values",
			input:        `//axon::core -Manual="Custom Module Name"`,
			expectError:  false,
			expectedType: CoreAnnotation,
			expectedParams: map[string]interface{}{
				"Mode":   "Singleton", // Default
				"Init":   "Same",      // Default
				"Manual": "Custom Module Name",
			},
		},
		{
			name:         "single quoted values",
			input:        `//axon::core -Manual='Custom Module Name'`,
			expectError:  false,
			expectedType: CoreAnnotation,
			expectedParams: map[string]interface{}{
				"Mode":   "Singleton", // Default
				"Init":   "Same",      // Default
				"Manual": "Custom Module Name",
			},
		},
		{
			name:         "quoted values with special characters",
			input:        `//axon::core -Manual="Module-With_Special.Chars@123"`,
			expectError:  false,
			expectedType: CoreAnnotation,
			expectedParams: map[string]interface{}{
				"Mode":   "Singleton", // Default
				"Init":   "Same",      // Default
				"Manual": "Module-With_Special.Chars@123",
			},
		},
		{
			name:         "empty quoted values",
			input:        `//axon::core -Manual=""`,
			expectError:  false,
			expectedType: CoreAnnotation,
			expectedParams: map[string]interface{}{
				"Mode":   "Singleton", // Default
				"Init":   "Same",      // Default
				"Manual": "",
			},
		},
		
		// Comma-separated values
		{
			name:         "comma separated without spaces",
			input:        "//axon::route GET /users -Middleware=Auth,Logging,Cache",
			expectError:  false,
			expectedType: RouteAnnotation,
			expectedParams: map[string]interface{}{
				"method":     "GET",
				"path":       "/users",
				"Middleware": []string{"Auth", "Logging", "Cache"},
			},
		},
		{
			name:         "comma separated with spaces",
			input:        "//axon::route GET /users -Middleware=Auth, Logging, Cache",
			expectError:  false,
			expectedType: RouteAnnotation,
			expectedParams: map[string]interface{}{
				"method":     "GET",
				"path":       "/users",
				"Middleware": []string{"Auth", "Logging", "Cache"},
			},
		},
		{
			name:         "comma separated with mixed quotes",
			input:        `//axon::route GET /users -Middleware="Auth Service",'Logging Service',Cache`,
			expectError:  false,
			expectedType: RouteAnnotation,
			expectedParams: map[string]interface{}{
				"method":     "GET",
				"path":       "/users",
				"Middleware": []string{"Auth Service", "Logging Service", "Cache"},
			},
		},
		{
			name:         "comma separated with empty values",
			input:        "//axon::route GET /users -Middleware=Auth,,Logging",
			expectError:  false,
			expectedType: RouteAnnotation,
			expectedParams: map[string]interface{}{
				"method":     "GET",
				"path":       "/users",
				"Middleware": []string{"Auth", "", "Logging"},
			},
		},
		
		// Boolean flags
		{
			name:         "boolean flag without value",
			input:        "//axon::route GET /users -PassContext",
			expectError:  false,
			expectedType: RouteAnnotation,
			expectedParams: map[string]interface{}{
				"method":      "GET",
				"path":        "/users",
				"PassContext": true,
			},
		},
		{
			name:         "multiple boolean flags",
			input:        "//axon::core -Init -Manual",
			expectError:  false,
			expectedType: CoreAnnotation,
			expectedParams: map[string]interface{}{
				"Mode":   "Singleton", // Default
				"Init":   "Same",      // Schema default for -Init flag
				"Manual": "",          // No default, so empty string
			},
		},
		
		// Complex parameter combinations
		{
			name:         "mixed parameter types",
			input:        `//axon::route POST /users/{id:int} -Middleware="Auth Service",Logging -PassContext`,
			expectError:  false,
			expectedType: RouteAnnotation,
			expectedParams: map[string]interface{}{
				"method":      "POST",
				"path":        "/users/{id:int}",
				"Middleware":  []string{"Auth Service", "Logging"},
				"PassContext": true,
			},
		},
		
		// Edge cases with parameter names (these should fail validation)
		{
			name:        "parameter names with numbers",
			input:       "//axon::core -Mode2=Transient -Init3=Background",
			expectError: true, // Unknown parameters should fail validation
		},
		{
			name:        "parameter names with underscores",
			input:       "//axon::core -Custom_Mode=Transient -Init_Type=Background",
			expectError: true, // Unknown parameters should fail validation
		},
		
		// Path parameter edge cases
		{
			name:         "complex path with multiple parameters",
			input:        "//axon::route GET /api/v1/users/{userId:int}/posts/{postId:string}/comments/{commentId:uuid}",
			expectError:  false,
			expectedType: RouteAnnotation,
			expectedParams: map[string]interface{}{
				"method": "GET",
				"path":   "/api/v1/users/{userId:int}/posts/{postId:string}/comments/{commentId:uuid}",
			},
		},
		{
			name:         "path with query parameters",
			input:        "//axon::route GET /search?q={query}&limit={limit:int}&offset={offset:int}",
			expectError:  false,
			expectedType: RouteAnnotation,
			expectedParams: map[string]interface{}{
				"method": "GET",
				"path":   "/search?q={query}&limit={limit:int}&offset={offset:int}",
			},
		},
		{
			name:         "path with special characters",
			input:        "//axon::route GET /api/v1/files/{filename:string}/download?format={format}&compress={compress:bool}",
			expectError:  false,
			expectedType: RouteAnnotation,
			expectedParams: map[string]interface{}{
				"method": "GET",
				"path":   "/api/v1/files/{filename:string}/download?format={format}&compress={compress:bool}",
			},
		},
		
		// Unicode and special character handling
		{
			name:         "unicode in parameter values",
			input:        `//axon::core -Manual="Módulo Personalizado"`,
			expectError:  false,
			expectedType: CoreAnnotation,
			expectedParams: map[string]interface{}{
				"Mode":   "Singleton", // Default
				"Init":   "Same",      // Default
				"Manual": "Módulo Personalizado",
			},
		},
		{
			name:         "special characters in path",
			input:        "//axon::route GET /api/files/{filename:string}",
			expectError:  false,
			expectedType: RouteAnnotation,
			expectedParams: map[string]interface{}{
				"method": "GET",
				"path":   "/api/files/{filename:string}",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			annotation, err := parser.ParseAnnotation(tt.input, location)
			
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
			
			// Check annotation type
			if annotation.Type != tt.expectedType {
				t.Errorf("expected type %v, got %v", tt.expectedType, annotation.Type)
			}
			
			// Check parameters
			for key, expectedValue := range tt.expectedParams {
				actualValue, exists := annotation.Parameters[key]
				if !exists {
					t.Errorf("expected parameter %s to exist", key)
					continue
				}
				
				// Handle slice comparison
				if expectedSlice, ok := expectedValue.([]string); ok {
					actualSlice, ok := actualValue.([]string)
					if !ok {
						t.Errorf("expected parameter %s to be []string, got %T", key, actualValue)
						continue
					}
					if !stringSlicesEqual(expectedSlice, actualSlice) {
						t.Errorf("expected parameter %s to be %v, got %v", key, expectedValue, actualValue)
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

// Test tokenizer error cases
func TestTokenizerErrorCases(t *testing.T) {
	registry := NewRegistry()
	err := RegisterBuiltinSchemas(registry)
	if err != nil {
		t.Fatalf("failed to register builtin schemas: %v", err)
	}
	parser := NewParser(registry)
	location := SourceLocation{File: "error_test.go", Line: 10, Column: 5}

	tests := []struct {
		name         string
		input        string
		expectedMsg  string
		expectedHint string
	}{
		{
			name:         "missing comment prefix",
			input:        "axon::core",
			expectedMsg:  "annotation must start with '//'",
			expectedHint: "Use format: //axon::type parameters",
		},
		{
			name:         "wrong comment prefix",
			input:        "/axon::core",
			expectedMsg:  "annotation must start with '//'",
			expectedHint: "Use format: //axon::type parameters",
		},
		{
			name:         "missing axon prefix",
			input:        "//annotation::core",
			expectedMsg:  "annotation must contain 'axon::' prefix",
			expectedHint: "Use format: //axon::type parameters",
		},
		{
			name:         "wrong axon prefix",
			input:        "//axon:core",
			expectedMsg:  "annotation must contain 'axon::' prefix",
			expectedHint: "Use format: //axon::type parameters",
		},
		{
			name:         "empty annotation",
			input:        "//axon::",
			expectedMsg:  "empty annotation",
			expectedHint: "Provide annotation type after 'axon::'",
		},
		{
			name:         "whitespace only annotation",
			input:        "//axon::   ",
			expectedMsg:  "empty annotation",
			expectedHint: "Provide annotation type after 'axon::'",
		},
		{
			name:         "unknown annotation type",
			input:        "//axon::unknown",
			expectedMsg:  "unknown annotation type: unknown",
			expectedHint: "Use one of: core, route, controller, middleware, interface",
		},
		{
			name:         "case sensitive annotation type",
			input:        "//axon::Core",
			expectedMsg:  "unknown annotation type: Core",
			expectedHint: "Use one of: core, route, controller, middleware, interface",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parser.ParseAnnotation(tt.input, location)
			
			if err == nil {
				t.Errorf("expected error but got none")
				return
			}
			
			errorStr := err.Error()
			
			// Check error message
			if !strings.Contains(errorStr, tt.expectedMsg) {
				t.Errorf("expected error message to contain '%s', got '%s'", tt.expectedMsg, errorStr)
			}
			
			// Check hint/suggestion
			if !strings.Contains(errorStr, tt.expectedHint) {
				t.Errorf("expected error to contain hint '%s', got '%s'", tt.expectedHint, errorStr)
			}
			
			// Check location is included
			if !strings.Contains(errorStr, "error_test.go:10:5") {
				t.Errorf("expected error to contain location 'error_test.go:10:5', got '%s'", errorStr)
			}
		})
	}
}

// Test parameter parsing edge cases
func TestParameterParsingEdgeCasesDetailed(t *testing.T) {
	registry := NewRegistry()
	err := RegisterBuiltinSchemas(registry)
	if err != nil {
		t.Fatalf("failed to register builtin schemas: %v", err)
	}
	parser := NewParser(registry)
	location := SourceLocation{File: "param_test.go", Line: 1, Column: 1}

	tests := []struct {
		name           string
		input          string
		expectError    bool
		expectedParams map[string]interface{}
	}{
		// Parameter value edge cases
		{
			name:        "parameter with equals in value",
			input:       `//axon::core -Manual="key=value"`,
			expectError: false,
			expectedParams: map[string]interface{}{
				"Manual": "key=value",
			},
		},
		{
			name:        "parameter with commas in quoted value",
			input:       `//axon::core -Manual="value,with,commas"`,
			expectError: false,
			expectedParams: map[string]interface{}{
				"Manual": "value,with,commas",
			},
		},
		{
			name:        "parameter with spaces in quoted value",
			input:       `//axon::core -Manual="value with spaces"`,
			expectError: false,
			expectedParams: map[string]interface{}{
				"Manual": "value with spaces",
			},
		},
		{
			name:        "parameter with quotes in value",
			input:       `//axon::core -Manual="value \"with\" quotes"`,
			expectError: false,
			expectedParams: map[string]interface{}{
				"Manual": `value "with" quotes`,
			},
		},
		
		// Boolean flag variations
		{
			name:        "boolean flag with explicit true",
			input:       "//axon::route GET /users -PassContext=true",
			expectError: false,
			expectedParams: map[string]interface{}{
				"method":      "GET",
				"path":        "/users",
				"PassContext": true, // Will be converted to bool by validator
			},
		},
		{
			name:        "boolean flag with explicit false",
			input:       "//axon::route GET /users -PassContext=false",
			expectError: false,
			expectedParams: map[string]interface{}{
				"method":      "GET",
				"path":        "/users",
				"PassContext": false, // Will be converted to bool by validator
			},
		},
		{
			name:        "boolean flag with numeric values",
			input:       "//axon::route GET /users -PassContext=1",
			expectError: false,
			expectedParams: map[string]interface{}{
				"method":      "GET",
				"path":        "/users",
				"PassContext": true, // Will be converted to bool by validator (1 -> true)
			},
		},
		
		// Comma-separated value edge cases
		{
			name:        "comma separated with trailing comma",
			input:       "//axon::route GET /users -Middleware=Auth,Logging,",
			expectError: false,
			expectedParams: map[string]interface{}{
				"method":     "GET",
				"path":       "/users",
				"Middleware": []string{"Auth", "Logging", ""},
			},
		},
		{
			name:        "comma separated with leading comma",
			input:       "//axon::route GET /users -Middleware=,Auth,Logging",
			expectError: false,
			expectedParams: map[string]interface{}{
				"method":     "GET",
				"path":       "/users",
				"Middleware": []string{"", "Auth", "Logging"},
			},
		},
		{
			name:        "comma separated with multiple consecutive commas",
			input:       "//axon::route GET /users -Middleware=Auth,,Logging",
			expectError: false,
			expectedParams: map[string]interface{}{
				"method":     "GET",
				"path":       "/users",
				"Middleware": []string{"Auth", "", "Logging"},
			},
		},
		{
			name:        "comma separated with only commas",
			input:       "//axon::route GET /users -Middleware=,,,",
			expectError: false,
			expectedParams: map[string]interface{}{
				"method":     "GET",
				"path":       "/users",
				"Middleware": []string{"", "", "", ""},
			},
		},
		
		// Parameter name edge cases
		{
			name:        "parameter name with numbers at end",
			input:       "//axon::core -Mode123=Transient",
			expectError: true, // Invalid parameter names should fail validation
		},
		{
			name:        "parameter name with numbers in middle",
			input:       "//axon::core -Mo123de=Transient",
			expectError: true, // Invalid parameter names should fail validation
		},
		{
			name:        "parameter name with underscores",
			input:       "//axon::core -Custom_Mode_Name=Transient",
			expectError: true, // Invalid parameter names should fail validation
		},
		
		// Multiple parameter combinations (with unknown parameters)
		{
			name:        "many parameters mixed types",
			input:       `//axon::core -Mode=Transient -Init=Background -Manual="Custom Module" -Debug -Count=42`,
			expectError: true, // Unknown parameters Debug and Count should fail validation
		},
		
		// Whitespace handling in parameters
		{
			name:        "parameter with leading/trailing spaces in value",
			input:       `//axon::core -Manual="  spaced value  "`,
			expectError: false,
			expectedParams: map[string]interface{}{
				"Manual": "  spaced value  ",
			},
		},
		{
			name:        "comma separated with spaces around values",
			input:       "//axon::route GET /users -Middleware= Auth , Logging , Cache ",
			expectError: false,
			expectedParams: map[string]interface{}{
				"method":     "GET",
				"path":       "/users",
				"Middleware": []string{"Auth", "Logging", "Cache"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			annotation, err := parser.ParseAnnotation(tt.input, location)
			
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
			
			// Check expected parameters
			for key, expectedValue := range tt.expectedParams {
				actualValue, exists := annotation.Parameters[key]
				if !exists {
					t.Errorf("expected parameter %s to exist", key)
					continue
				}
				
				// Handle slice comparison
				if expectedSlice, ok := expectedValue.([]string); ok {
					actualSlice, ok := actualValue.([]string)
					if !ok {
						t.Errorf("expected parameter %s to be []string, got %T", key, actualValue)
						continue
					}
					if !stringSlicesEqual(expectedSlice, actualSlice) {
						t.Errorf("expected parameter %s to be %v, got %v", key, expectedValue, actualValue)
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

