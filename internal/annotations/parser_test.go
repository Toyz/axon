package annotations

import (
	"errors"
	"strings"
	"testing"
)

func TestParseAnnotationWithInit(t *testing.T) {
	// Create a registry with test schemas
	registry := NewRegistry()
	
	// Add core schema with Init parameter
	coreSchema := AnnotationSchema{
		Type: CoreAnnotation,
		Parameters: map[string]ParameterSpec{
			"Mode": {
				Type:         StringType,
				Required:     false,
				DefaultValue: "Singleton",
			},
			"Init": {
				Type:         StringType,
				Required:     false,
				DefaultValue: "Same", // -Init should become Init: "Same"
			},
		},
	}
	registry.Register(CoreAnnotation, coreSchema)
	
	parser := NewParser(registry)
	location := SourceLocation{File: "test.go", Line: 10, Column: 1}
	
	// Test: -Init should use schema default "Same", NOT boolean true
	annotation, err := parser.ParseAnnotation("//axon::core -Init", location)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	
	if annotation.Type != CoreAnnotation {
		t.Errorf("expected CoreAnnotation, got %v", annotation.Type)
	}
	
	// Critical test: -Init should become Init: "Same" from schema
	if annotation.Parameters["Init"] != "Same" {
		t.Errorf("expected Init='Same' (from schema default), got %v", annotation.Parameters["Init"])
	}
	
	// Verify it's NOT treated as boolean
	if val, ok := annotation.Parameters["Init"].(bool); ok {
		t.Errorf("Init should NOT be boolean, but got boolean value: %v", val)
	}
	
	// Test explicit value still works
	annotation2, err := parser.ParseAnnotation("//axon::core -Init=Background", location)
	if err != nil {
		t.Fatalf("unexpected error for explicit value: %v", err)
	}
	
	if annotation2.Parameters["Init"] != "Background" {
		t.Errorf("expected Init='Background', got %v", annotation2.Parameters["Init"])
	}
}

func TestFlexibleCommentPrefix(t *testing.T) {
	registry := NewRegistry()
	err := RegisterBuiltinSchemas(registry)
	if err != nil {
		t.Fatalf("failed to register builtin schemas: %v", err)
	}
	parser := NewParser(registry)
	location := SourceLocation{File: "test.go", Line: 1, Column: 1}
	
	tests := []struct {
		name  string
		input string
	}{
		{"standard", "//axon::core"},
		{"space after slashes", "// axon::core"},
		{"multiple spaces", "//  axon::core"},
		{"tab after slashes", "//\taxon::core"},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			annotation, err := parser.ParseAnnotation(tt.input, location)
			if err != nil {
				t.Errorf("unexpected error for %s: %v", tt.name, err)
				return
			}
			if annotation.Type != CoreAnnotation {
				t.Errorf("expected CoreAnnotation for %s, got %v", tt.name, annotation.Type)
			}
		})
	}
}

func TestRouteAnnotationPositionalParams(t *testing.T) {
	registry := NewRegistry()
	err := RegisterBuiltinSchemas(registry)
	if err != nil {
		t.Fatalf("failed to register builtin schemas: %v", err)
	}
	parser := NewParser(registry)
	location := SourceLocation{File: "test.go", Line: 1, Column: 1}
	
	annotation, err := parser.ParseAnnotation("//axon::route GET /users/{id:int}", location)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	
	if annotation.Type != RouteAnnotation {
		t.Errorf("expected RouteAnnotation, got %v", annotation.Type)
	}
	
	if annotation.Parameters["method"] != "GET" {
		t.Errorf("expected method=GET, got %v", annotation.Parameters["method"])
	}
	
	if annotation.Parameters["path"] != "/users/{id:int}" {
		t.Errorf("expected path=/users/{id:int}, got %v", annotation.Parameters["path"])
	}
}

func TestCommaSeperatedValues(t *testing.T) {
	registry := NewRegistry()
	err := RegisterBuiltinSchemas(registry)
	if err != nil {
		t.Fatalf("failed to register builtin schemas: %v", err)
	}
	parser := NewParser(registry)
	location := SourceLocation{File: "test.go", Line: 1, Column: 1}
	
	annotation, err := parser.ParseAnnotation("//axon::route GET /users -Middleware=Auth,Logging", location)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	
	middleware, ok := annotation.Parameters["Middleware"].([]string)
	if !ok {
		t.Errorf("expected Middleware to be []string, got %T", annotation.Parameters["Middleware"])
		return
	}
	
	if len(middleware) != 2 || middleware[0] != "Auth" || middleware[1] != "Logging" {
		t.Errorf("expected Middleware=[Auth, Logging], got %v", middleware)
	}
}

// Comprehensive tokenizer tests for various comment formats and edge cases
func TestTokenizerEdgeCases(t *testing.T) {
	registry := NewRegistry()
	err := RegisterBuiltinSchemas(registry)
	if err != nil {
		t.Fatalf("failed to register builtin schemas: %v", err)
	}
	parser := NewParser(registry)
	location := SourceLocation{File: "test.go", Line: 1, Column: 1}

	tests := []struct {
		name        string
		input       string
		expectError bool
		errorMsg    string
	}{
		// Valid comment prefix variations
		{"standard format", "//axon::core", false, ""},
		{"space after slashes", "// axon::core", false, ""},
		{"multiple spaces", "//  axon::core", false, ""},
		{"tab after slashes", "//\taxon::core", false, ""},
		{"mixed whitespace", "//\t  axon::core", false, ""},
		
		// Invalid comment prefixes
		{"missing slashes", "axon::core", true, "annotation must start with '//'"},
		{"single slash", "/axon::core", true, "annotation must start with '//'"},
		{"wrong prefix", "//annotation::core", true, "annotation must contain 'axon::' prefix"},
		{"missing colon", "//axon:core", true, "annotation must contain 'axon::' prefix"},
		{"extra colons", "//axon:::core", true, "unknown annotation type"}, // Extra colon makes it ":core"
		
		// Edge cases with whitespace
		{"leading whitespace", "  //axon::core", false, ""},
		{"trailing whitespace", "//axon::core  ", false, ""},
		{"whitespace around prefix", "  //  axon::  core  ", false, ""},
		
		// Empty and minimal cases
		{"empty after prefix", "//axon::", true, "empty annotation"},
		{"only whitespace after prefix", "//axon::   ", true, "empty annotation"},
		
		// Case sensitivity
		{"uppercase axon", "//AXON::core", true, "annotation must contain 'axon::' prefix"},
		{"mixed case axon", "//Axon::core", true, "annotation must contain 'axon::' prefix"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parser.ParseAnnotation(tt.input, location)
			
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

// Comprehensive parameter parsing tests with all supported types and formats
func TestParameterParsingEdgeCases(t *testing.T) {
	registry := NewRegistry()
	err := RegisterBuiltinSchemas(registry)
	if err != nil {
		t.Fatalf("failed to register builtin schemas: %v", err)
	}
	parser := NewParser(registry)
	location := SourceLocation{File: "test.go", Line: 1, Column: 1}

	tests := []struct {
		name           string
		input          string
		expectedParams map[string]interface{}
		expectError    bool
	}{
		// Quoted string parameters
		{
			name:  "double quoted parameter",
			input: `//axon::core -Manual="Custom Module"`,
			expectedParams: map[string]interface{}{
				"Manual": "Custom Module",
			},
		},
		{
			name:  "single quoted parameter",
			input: `//axon::core -Manual='Custom Module'`,
			expectedParams: map[string]interface{}{
				"Manual": "Custom Module",
			},
		},
		{
			name:  "quoted parameter with special chars",
			input: `//axon::core -Manual="Module-With_Special.Chars"`,
			expectedParams: map[string]interface{}{
				"Manual": "Module-With_Special.Chars",
			},
		},
		{
			name:  "empty quoted parameter",
			input: `//axon::core -Manual=""`,
			expectedParams: map[string]interface{}{
				"Manual": "",
			},
		},
		
		// Comma-separated values with various formats
		{
			name:  "comma separated without spaces",
			input: "//axon::route GET /users -Middleware=Auth,Logging,Cache",
			expectedParams: map[string]interface{}{
				"method":     "GET",
				"path":       "/users",
				"Middleware": []string{"Auth", "Logging", "Cache"},
			},
		},
		{
			name:  "comma separated with spaces",
			input: "//axon::route GET /users -Middleware=Auth, Logging, Cache",
			expectedParams: map[string]interface{}{
				"method":     "GET",
				"path":       "/users",
				"Middleware": []string{"Auth", "Logging", "Cache"},
			},
		},
		{
			name:  "comma separated with quoted values",
			input: `//axon::route GET /users -Middleware="Auth Service","Logging Service"`,
			expectedParams: map[string]interface{}{
				"method":     "GET",
				"path":       "/users",
				"Middleware": []string{"Auth Service", "Logging Service"},
			},
		},
		{
			name:  "single value in comma format",
			input: "//axon::route GET /users -Middleware=Auth",
			expectedParams: map[string]interface{}{
				"method":     "GET",
				"path":       "/users",
				"Middleware": []string{"Auth"}, // Single values are converted to slices by validator
			},
		},
		
		// Boolean flags
		{
			name:  "boolean flag without value",
			input: "//axon::route GET /users -PassContext",
			expectedParams: map[string]interface{}{
				"method":      "GET",
				"path":        "/users",
				"PassContext": true,
			},
		},
		{
			name:  "boolean flag with explicit true",
			input: "//axon::route GET /users -PassContext=true",
			expectedParams: map[string]interface{}{
				"method":      "GET",
				"path":        "/users",
				"PassContext": true, // Will be converted to bool by validator
			},
		},
		{
			name:  "boolean flag with explicit false",
			input: "//axon::route GET /users -PassContext=false",
			expectedParams: map[string]interface{}{
				"method":      "GET",
				"path":        "/users",
				"PassContext": false, // Will be converted to bool by validator
			},
		},
		
		// Mixed parameter types
		{
			name:  "mixed parameters",
			input: `//axon::core -Mode=Transient -Init=Background -Manual="Custom Module"`,
			expectedParams: map[string]interface{}{
				"Mode":   "Transient",
				"Init":   "Background",
				"Manual": "Custom Module",
			},
		},
		
		// Edge cases with parameter names (these should fail validation)
		{
			name:        "parameter with numbers",
			input:       "//axon::core -Mode2=Transient",
			expectError: true, // Unknown parameter should fail validation
		},
		{
			name:        "parameter with underscores",
			input:       "//axon::core -Custom_Mode=Transient",
			expectError: true, // Unknown parameter should fail validation
		},
		
		// Complex route patterns
		{
			name:  "route with path parameters",
			input: "//axon::route GET /users/{id:int}/posts/{postId:string}",
			expectedParams: map[string]interface{}{
				"method": "GET",
				"path":   "/users/{id:int}/posts/{postId:string}",
			},
		},
		{
			name:  "route with query parameters in path",
			input: "//axon::route GET /search?q={query}&limit={limit:int}",
			expectedParams: map[string]interface{}{
				"method": "GET",
				"path":   "/search?q={query}&limit={limit:int}",
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
			
			// Check all expected parameters
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

// Test error handling and message quality
func TestParserErrorHandling(t *testing.T) {
	registry := NewRegistry()
	err := RegisterBuiltinSchemas(registry)
	if err != nil {
		t.Fatalf("failed to register builtin schemas: %v", err)
	}
	parser := NewParser(registry)
	location := SourceLocation{File: "controller.go", Line: 25, Column: 3}

	tests := []struct {
		name                string
		input               string
		expectedErrorType   ErrorCode
		expectedErrorMsg    string
		expectedSuggestion  string
	}{
		{
			name:               "unknown annotation type",
			input:              "//axon::unknown",
			expectedErrorType:  0, // ParseError, not AnnotationError
			expectedErrorMsg:   "unknown annotation type: unknown",
			expectedSuggestion: "Use one of: core, route, controller, middleware, interface",
		},
		{
			name:               "missing annotation type",
			input:              "//axon::",
			expectedErrorType:  0, // ParseError
			expectedErrorMsg:   "empty annotation",
			expectedSuggestion: "Provide annotation type after 'axon::'",
		},
		{
			name:               "invalid comment prefix",
			input:              "/axon::core",
			expectedErrorType:  0, // ParseError
			expectedErrorMsg:   "annotation must start with '//'",
			expectedSuggestion: "Use format: //axon::type parameters",
		},
		{
			name:               "missing axon prefix",
			input:              "//annotation::core",
			expectedErrorType:  0, // ParseError
			expectedErrorMsg:   "annotation must contain 'axon::' prefix",
			expectedSuggestion: "Use format: //axon::type parameters",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parser.ParseAnnotation(tt.input, location)
			
			if err == nil {
				t.Errorf("expected error but got none")
				return
			}
			
			// Check error message content
			if !strings.Contains(err.Error(), tt.expectedErrorMsg) {
				t.Errorf("expected error message to contain '%s', got '%s'", tt.expectedErrorMsg, err.Error())
			}
			
			// Check suggestion content
			if !strings.Contains(err.Error(), tt.expectedSuggestion) {
				t.Errorf("expected error to contain suggestion '%s', got '%s'", tt.expectedSuggestion, err.Error())
			}
			
			// Check location is included
			if !strings.Contains(err.Error(), "controller.go:25:3") {
				t.Errorf("expected error to contain location 'controller.go:25:3', got '%s'", err.Error())
			}
		})
	}
}

// Test schema validation with comprehensive valid and invalid inputs
func TestSchemaValidationComprehensive(t *testing.T) {
	registry := NewRegistry()
	err := RegisterBuiltinSchemas(registry)
	if err != nil {
		t.Fatalf("failed to register builtin schemas: %v", err)
	}
	parser := NewParser(registry)
	location := SourceLocation{File: "test.go", Line: 1, Column: 1}

	tests := []struct {
		name        string
		input       string
		expectError bool
		errorType   ErrorCode
		errorMsg    string
	}{
		// Valid core annotations
		{
			name:        "valid core with defaults",
			input:       "//axon::core",
			expectError: false,
		},
		{
			name:        "valid core with singleton mode",
			input:       "//axon::core -Mode=Singleton",
			expectError: false,
		},
		{
			name:        "valid core with transient mode",
			input:       "//axon::core -Mode=Transient",
			expectError: false,
		},
		{
			name:        "valid core with background init",
			input:       "//axon::core -Init=Background",
			expectError: false,
		},
		{
			name:        "valid core with manual module",
			input:       `//axon::core -Manual="CustomModule"`,
			expectError: false,
		},
		
		// Invalid core annotations
		{
			name:        "invalid core mode",
			input:       "//axon::core -Mode=Invalid",
			expectError: true,
			errorType:   ValidationErrorCode,
			errorMsg:    "must be 'Singleton' or 'Transient'",
		},
		{
			name:        "invalid core init",
			input:       "//axon::core -Init=Invalid",
			expectError: true,
			errorType:   ValidationErrorCode,
			errorMsg:    "must be 'Same' or 'Background'",
		},
		
		// Valid route annotations
		{
			name:        "valid route basic",
			input:       "//axon::route GET /users",
			expectError: false,
		},
		{
			name:        "valid route with middleware",
			input:       "//axon::route POST /users -Middleware=Auth,Logging",
			expectError: false,
		},
		{
			name:        "valid route with pass context",
			input:       "//axon::route GET /health -PassContext",
			expectError: false,
		},
		{
			name:        "valid route with all parameters",
			input:       "//axon::route PUT /users/{id:int} -Middleware=Auth -PassContext",
			expectError: false,
		},
		
		// Invalid route annotations
		{
			name:        "invalid route method",
			input:       "//axon::route INVALID /users",
			expectError: true,
			errorType:   ValidationErrorCode,
			errorMsg:    "must be one of",
		},
		{
			name:        "route missing method",
			input:       "//axon::route",
			expectError: true,
			errorType:   ValidationErrorCode,
			errorMsg:    "requires method parameter",
		},
		{
			name:        "route missing path",
			input:       "//axon::route GET",
			expectError: true,
			errorType:   ValidationErrorCode,
			errorMsg:    "requires path parameter",
		},
		{
			name:        "route invalid path format",
			input:       "//axon::route GET users",
			expectError: true,
			errorType:   0, // This is caught by schema validation, not parameter validation
			errorMsg:    "must start with '/'",
		},
		
		// Valid controller annotations
		{
			name:        "valid controller basic",
			input:       "//axon::controller",
			expectError: false,
		},
		{
			name:        "valid controller with prefix",
			input:       "//axon::controller -Prefix=/api/v1",
			expectError: false,
		},
		
		// Valid middleware annotations
		{
			name:        "valid middleware basic",
			input:       "//axon::middleware",
			expectError: false,
		},
		{
			name:        "valid middleware with routes",
			input:       "//axon::middleware -Routes=/api/*,/admin/*",
			expectError: false,
		},
		
		// Invalid middleware annotations
		{
			name:        "middleware with invalid route pattern",
			input:       "//axon::middleware -Routes=api/*",
			expectError: true,
			errorType:   0, // This is caught by schema validation
			errorMsg:    "must start with '/'",
		},
		{
			name:        "middleware with empty route pattern",
			input:       "//axon::middleware -Routes=/api/*,",
			expectError: true,
			errorType:   0, // This is caught by schema validation
			errorMsg:    "route pattern cannot be empty",
		},
		
		// Valid interface annotations
		{
			name:        "valid interface basic",
			input:       "//axon::interface",
			expectError: false,
		},
		{
			name:        "valid interface with name",
			input:       "//axon::interface -Name=UserService",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parser.ParseAnnotation(tt.input, location)
			
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
				
				// Check error message if specified
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

