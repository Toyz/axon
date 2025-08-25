package annotations

import (
	"errors"
	"testing"
)

// TestSchemaValidatorIntegration tests the complete workflow of registering schemas and validating annotations
func TestSchemaValidatorIntegration(t *testing.T) {
	// Create registry and validator
	registry := NewRegistry()
	validator := NewValidator()

	// Define a test schema
	testSchema := AnnotationSchema{
		Type:        CoreAnnotation,
		Description: "Test core annotation schema",
		Parameters: map[string]ParameterSpec{
			"Mode": {
				Type:         StringType,
				Required:     true,
				Description:  "Service lifecycle mode",
				Validator: func(v interface{}) error {
					mode := v.(string)
					if mode != "Singleton" && mode != "Transient" {
						return errors.New("must be 'Singleton' or 'Transient'")
					}
					return nil
				},
			},
			"Init": {
				Type:         StringType,
				Required:     false,
				DefaultValue: "Same",
				Description:  "Initialization mode",
				Validator: func(v interface{}) error {
					init := v.(string)
					if init != "Same" && init != "Background" {
						return errors.New("must be 'Same' or 'Background'")
					}
					return nil
				},
			},
			"Port": {
				Type:         IntType,
				Required:     false,
				DefaultValue: 8080,
				Description:  "Service port",
			},
			"Tags": {
				Type:        StringSliceType,
				Required:    false,
				Description: "Service tags",
			},
		},
	}

	// Register the schema
	err := registry.Register(CoreAnnotation, testSchema)
	if err != nil {
		t.Fatalf("Failed to register schema: %v", err)
	}

	tests := []struct {
		name        string
		annotation  *ParsedAnnotation
		expectError bool
		description string
	}{
		{
			name: "valid annotation with required parameters",
			annotation: &ParsedAnnotation{
				Type: CoreAnnotation,
				Parameters: map[string]interface{}{
					"Mode": "Singleton",
				},
				Location: SourceLocation{File: "test.go", Line: 1, Column: 1},
			},
			expectError: false,
			description: "Should pass with valid required parameters",
		},
		{
			name: "annotation with string parameters needing transformation",
			annotation: &ParsedAnnotation{
				Type: CoreAnnotation,
				Parameters: map[string]interface{}{
					"Mode": "Transient",
					"Port": "9090", // String that needs to be converted to int
					"Tags": "auth,logging,cache", // String that needs to be converted to []string
				},
				Location: SourceLocation{File: "test.go", Line: 1, Column: 1},
			},
			expectError: false,
			description: "Should pass after parameter transformation",
		},
		{
			name: "annotation missing required parameter",
			annotation: &ParsedAnnotation{
				Type:       CoreAnnotation,
				Parameters: map[string]interface{}{},
				Location:   SourceLocation{File: "test.go", Line: 1, Column: 1},
			},
			expectError: true,
			description: "Should fail due to missing required Mode parameter",
		},
		{
			name: "annotation with invalid parameter value",
			annotation: &ParsedAnnotation{
				Type: CoreAnnotation,
				Parameters: map[string]interface{}{
					"Mode": "InvalidMode",
				},
				Location: SourceLocation{File: "test.go", Line: 1, Column: 1},
			},
			expectError: true,
			description: "Should fail due to invalid Mode value",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Get the schema
			schema, err := registry.GetSchema(tt.annotation.Type)
			if err != nil {
				t.Fatalf("Failed to get schema: %v", err)
			}

			// Apply defaults first
			err = validator.ApplyDefaults(tt.annotation, schema)
			if err != nil {
				t.Fatalf("Failed to apply defaults: %v", err)
			}

			// Transform parameters
			err = validator.TransformParameters(tt.annotation, schema)
			if err != nil && !tt.expectError {
				t.Fatalf("Failed to transform parameters: %v", err)
			}

			// Validate the annotation
			err = validator.Validate(tt.annotation, schema)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none: %s", tt.description)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v (%s)", err, tt.description)
				}

				// Verify that defaults were applied
				if tt.name == "valid annotation with required parameters" {
					if tt.annotation.GetString("Init") != "Same" {
						t.Errorf("Expected default Init value 'Same', got '%s'", tt.annotation.GetString("Init"))
					}
					if tt.annotation.GetInt("Port") != 8080 {
						t.Errorf("Expected default Port value 8080, got %d", tt.annotation.GetInt("Port"))
					}
				}

				// Verify that transformations worked
				if tt.name == "annotation with string parameters needing transformation" {
					if tt.annotation.GetInt("Port") != 9090 {
						t.Errorf("Expected transformed Port value 9090, got %d", tt.annotation.GetInt("Port"))
					}
					tags := tt.annotation.GetStringSlice("Tags")
					expectedTags := []string{"auth", "logging", "cache"}
					if len(tags) != len(expectedTags) {
						t.Errorf("Expected %d tags, got %d", len(expectedTags), len(tags))
					}
					for i, expected := range expectedTags {
						if tags[i] != expected {
							t.Errorf("Expected tag[%d] to be '%s', got '%s'", i, expected, tags[i])
						}
					}
				}
			}
		})
	}
}

// TestCompleteWorkflow demonstrates the complete workflow from raw annotation to validated result
func TestCompleteWorkflow(t *testing.T) {
	// Setup
	registry := NewRegistry()
	validator := NewValidator()

	// Register a simple schema
	schema := AnnotationSchema{
		Type: RouteAnnotation,
		Parameters: map[string]ParameterSpec{
			"method": {
				Type:     StringType,
				Required: true,
			},
			"path": {
				Type:     StringType,
				Required: true,
			},
			"Middleware": {
				Type:     StringSliceType,
				Required: false,
			},
			"PassContext": {
				Type:         BoolType,
				Required:     false,
				DefaultValue: false,
			},
		},
	}

	err := registry.Register(RouteAnnotation, schema)
	if err != nil {
		t.Fatalf("Failed to register schema: %v", err)
	}

	// Simulate a parsed annotation (as would come from the parser)
	annotation := &ParsedAnnotation{
		Type: RouteAnnotation,
		Parameters: map[string]interface{}{
			"method":     "GET",
			"path":       "/users/{id:int}",
			"Middleware": "Auth,Logging", // String that needs conversion to []string
		},
		Location: SourceLocation{File: "controller.go", Line: 15, Column: 1},
		Raw:      "//axon::route GET /users/{id:int} -Middleware=Auth,Logging",
	}

	// Complete validation workflow
	retrievedSchema, err := registry.GetSchema(annotation.Type)
	if err != nil {
		t.Fatalf("Failed to get schema: %v", err)
	}

	// Step 1: Apply defaults
	err = validator.ApplyDefaults(annotation, retrievedSchema)
	if err != nil {
		t.Fatalf("Failed to apply defaults: %v", err)
	}

	// Step 2: Transform parameters
	err = validator.TransformParameters(annotation, retrievedSchema)
	if err != nil {
		t.Fatalf("Failed to transform parameters: %v", err)
	}

	// Step 3: Validate
	err = validator.Validate(annotation, retrievedSchema)
	if err != nil {
		t.Fatalf("Failed to validate annotation: %v", err)
	}

	// Verify results using type-safe getters
	if annotation.GetString("method") != "GET" {
		t.Errorf("Expected method 'GET', got '%s'", annotation.GetString("method"))
	}

	if annotation.GetString("path") != "/users/{id:int}" {
		t.Errorf("Expected path '/users/{id:int}', got '%s'", annotation.GetString("path"))
	}

	middleware := annotation.GetStringSlice("Middleware")
	expectedMiddleware := []string{"Auth", "Logging"}
	if len(middleware) != len(expectedMiddleware) {
		t.Errorf("Expected %d middleware items, got %d", len(expectedMiddleware), len(middleware))
	}
	for i, expected := range expectedMiddleware {
		if middleware[i] != expected {
			t.Errorf("Expected middleware[%d] to be '%s', got '%s'", i, expected, middleware[i])
		}
	}

	if annotation.GetBool("PassContext") != false {
		t.Errorf("Expected PassContext default value false, got %t", annotation.GetBool("PassContext"))
	}

	t.Logf("Successfully validated annotation: %+v", annotation)
}