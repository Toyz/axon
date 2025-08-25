package annotations

import (
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
)

func TestNewRegistry(t *testing.T) {
	registry := NewRegistry()
	if registry == nil {
		t.Fatal("NewRegistry() returned nil")
	}

	// Should start empty
	types := registry.ListTypes()
	if len(types) != 0 {
		t.Errorf("Expected empty registry, got %d types", len(types))
	}
}

func TestDefaultRegistry(t *testing.T) {
	// Should return the same instance
	registry1 := DefaultRegistry()
	registry2 := DefaultRegistry()

	if registry1 != registry2 {
		t.Error("DefaultRegistry() should return the same instance")
	}
}

func TestRegister(t *testing.T) {
	registry := NewRegistry()

	// Create a valid schema
	schema := AnnotationSchema{
		Type:        CoreAnnotation,
		Description: "Test core annotation",
		Parameters: map[string]ParameterSpec{
			"Mode": {
				Type:         StringType,
				Required:     false,
				DefaultValue: "Singleton",
				Description:  "Service mode",
			},
		},
		Examples: []string{"//axon::core -Mode=Transient"},
	}

	// Should register successfully
	err := registry.Register(CoreAnnotation, schema)
	if err != nil {
		t.Errorf("Failed to register schema: %v", err)
	}

	// Should be registered now
	if !registry.IsRegistered(CoreAnnotation) {
		t.Error("Schema should be registered")
	}

	// Should not allow duplicate registration
	err = registry.Register(CoreAnnotation, schema)
	if err == nil {
		t.Error("Expected error when registering duplicate schema")
	}
}

func TestRegisterInvalidSchema(t *testing.T) {
	registry := NewRegistry()

	tests := []struct {
		name   string
		schema AnnotationSchema
	}{
		{
			name: "mismatched type",
			schema: AnnotationSchema{
				Type:        RouteAnnotation, // Mismatch: registering as CoreAnnotation
				Description: "Test",
			},
		},
		{
			name: "empty parameter name",
			schema: AnnotationSchema{
				Type:        CoreAnnotation,
				Description: "Test",
				Parameters: map[string]ParameterSpec{
					"": { // Empty parameter name
						Type: StringType,
					},
				},
			},
		},
		{
			name: "invalid parameter type",
			schema: AnnotationSchema{
				Type:        CoreAnnotation,
				Description: "Test",
				Parameters: map[string]ParameterSpec{
					"Mode": {
						Type: ParameterType(999), // Invalid type
					},
				},
			},
		},
		{
			name: "wrong default value type",
			schema: AnnotationSchema{
				Type:        CoreAnnotation,
				Description: "Test",
				Parameters: map[string]ParameterSpec{
					"Mode": {
						Type:         StringType,
						DefaultValue: 123, // Should be string
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := registry.Register(CoreAnnotation, tt.schema)
			if err == nil {
				t.Errorf("Expected error for %s", tt.name)
			}
		})
	}
}

func TestGetSchema(t *testing.T) {
	registry := NewRegistry()

	// Create and register a schema
	originalSchema := AnnotationSchema{
		Type:        CoreAnnotation,
		Description: "Test core annotation",
		Parameters: map[string]ParameterSpec{
			"Mode": {
				Type:         StringType,
				Required:     false,
				DefaultValue: "Singleton",
				Description:  "Service mode",
			},
		},
		Examples: []string{"//axon::core -Mode=Transient"},
	}

	err := registry.Register(CoreAnnotation, originalSchema)
	if err != nil {
		t.Fatalf("Failed to register schema: %v", err)
	}

	// Should retrieve the schema
	retrievedSchema, err := registry.GetSchema(CoreAnnotation)
	if err != nil {
		t.Errorf("Failed to get schema: %v", err)
	}

	// Verify schema contents
	if retrievedSchema.Type != originalSchema.Type {
		t.Errorf("Expected type %v, got %v", originalSchema.Type, retrievedSchema.Type)
	}

	if retrievedSchema.Description != originalSchema.Description {
		t.Errorf("Expected description %s, got %s", originalSchema.Description, retrievedSchema.Description)
	}

	// Should fail for unregistered type
	_, err = registry.GetSchema(RouteAnnotation)
	if err == nil {
		t.Error("Expected error for unregistered annotation type")
	}
}

func TestListTypes(t *testing.T) {
	registry := NewRegistry()

	// Should start empty
	types := registry.ListTypes()
	if len(types) != 0 {
		t.Errorf("Expected empty list, got %d types", len(types))
	}

	// Register some schemas
	schemas := []struct {
		annotationType AnnotationType
		schema         AnnotationSchema
	}{
		{
			CoreAnnotation,
			AnnotationSchema{
				Type:        CoreAnnotation,
				Description: "Core annotation",
			},
		},
		{
			RouteAnnotation,
			AnnotationSchema{
				Type:        RouteAnnotation,
				Description: "Route annotation",
			},
		},
	}

	for _, s := range schemas {
		err := registry.Register(s.annotationType, s.schema)
		if err != nil {
			t.Fatalf("Failed to register %s: %v", s.annotationType.String(), err)
		}
	}

	// Should list all registered types
	types = registry.ListTypes()
	if len(types) != 2 {
		t.Errorf("Expected 2 types, got %d", len(types))
	}

	// Verify all types are present
	typeMap := make(map[AnnotationType]bool)
	for _, annotationType := range types {
		typeMap[annotationType] = true
	}

	if !typeMap[CoreAnnotation] {
		t.Error("CoreAnnotation not found in list")
	}
	if !typeMap[RouteAnnotation] {
		t.Error("RouteAnnotation not found in list")
	}
}

func TestIsRegistered(t *testing.T) {
	registry := NewRegistry()

	// Should not be registered initially
	if registry.IsRegistered(CoreAnnotation) {
		t.Error("CoreAnnotation should not be registered initially")
	}

	// Register a schema
	schema := AnnotationSchema{
		Type:        CoreAnnotation,
		Description: "Test",
	}

	err := registry.Register(CoreAnnotation, schema)
	if err != nil {
		t.Fatalf("Failed to register schema: %v", err)
	}

	// Should be registered now
	if !registry.IsRegistered(CoreAnnotation) {
		t.Error("CoreAnnotation should be registered")
	}

	// Other types should still not be registered
	if registry.IsRegistered(RouteAnnotation) {
		t.Error("RouteAnnotation should not be registered")
	}
}

func TestConcurrentAccess(t *testing.T) {
	registry := NewRegistry()
	
	// Number of goroutines to run concurrently
	numGoroutines := 10
	numOperations := 100

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	// Run concurrent operations
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()

			for j := 0; j < numOperations; j++ {
				// Mix of read and write operations
				if j%2 == 0 {
					// Read operations
					registry.IsRegistered(CoreAnnotation)
					registry.ListTypes()
					registry.GetSchema(CoreAnnotation) // This will fail, but shouldn't crash
				} else {
					// Write operations (will fail after first success, but shouldn't crash)
					schema := AnnotationSchema{
						Type:        AnnotationType(id), // Use different types to avoid conflicts
						Description: "Concurrent test",
					}
					registry.Register(AnnotationType(id), schema)
				}
			}
		}(i)
	}

	wg.Wait()
	// If we get here without deadlock or panic, the test passes
}

// Comprehensive concurrent access tests for parser and registry
func TestConcurrentParserAccess(t *testing.T) {
	registry := NewRegistry()
	err := RegisterBuiltinSchemas(registry)
	if err != nil {
		t.Fatalf("failed to register builtin schemas: %v", err)
	}
	
	parser := NewParser(registry)
	location := SourceLocation{File: "test.go", Line: 1, Column: 1}
	
	// Test annotations to parse concurrently
	testAnnotations := []string{
		"//axon::core -Mode=Singleton",
		"//axon::core -Mode=Transient -Init=Background",
		"//axon::route GET /users",
		"//axon::route POST /users -Middleware=Auth,Logging",
		"//axon::controller -Prefix=/api/v1",
		"//axon::middleware -Routes=/api/*",
		"//axon::interface -Name=UserService",
	}
	
	numGoroutines := 20
	numOperations := 50
	
	var wg sync.WaitGroup
	var errorCount int32
	var successCount int32
	
	wg.Add(numGoroutines)
	
	for i := 0; i < numGoroutines; i++ {
		go func(goroutineID int) {
			defer wg.Done()
			
			for j := 0; j < numOperations; j++ {
				// Pick a random annotation to parse
				annotationText := testAnnotations[j%len(testAnnotations)]
				
				// Parse the annotation
				annotation, err := parser.ParseAnnotation(annotationText, location)
				if err != nil {
					atomic.AddInt32(&errorCount, 1)
					t.Errorf("goroutine %d, operation %d: unexpected error: %v", goroutineID, j, err)
					continue
				}
				
				// Validate the result
				if annotation == nil {
					atomic.AddInt32(&errorCount, 1)
					t.Errorf("goroutine %d, operation %d: got nil annotation", goroutineID, j)
					continue
				}
				
				// Test type-safe getters concurrently
				_ = annotation.GetString("Mode", "Singleton")
				_ = annotation.GetBool("PassContext", false)
				_ = annotation.GetStringSlice("Middleware", nil)
				_ = annotation.HasParameter("Mode")
				
				atomic.AddInt32(&successCount, 1)
			}
		}(i)
	}
	
	wg.Wait()
	
	totalOperations := int32(numGoroutines * numOperations)
	if errorCount > 0 {
		t.Errorf("had %d errors out of %d total operations", errorCount, totalOperations)
	}
	
	if successCount != totalOperations-errorCount {
		t.Errorf("success count mismatch: expected %d, got %d", totalOperations-errorCount, successCount)
	}
	
	t.Logf("Concurrent test completed: %d successful operations, %d errors", successCount, errorCount)
}

// Test concurrent registry modifications
func TestConcurrentRegistryModifications(t *testing.T) {
	registry := NewRegistry()
	
	numGoroutines := 15
	numSchemas := 5
	
	var wg sync.WaitGroup
	var registrationErrors int32
	var lookupErrors int32
	
	wg.Add(numGoroutines)
	
	// Concurrent registration and lookup
	for i := 0; i < numGoroutines; i++ {
		go func(goroutineID int) {
			defer wg.Done()
			
			// Each goroutine tries to register different schemas
			for j := 0; j < numSchemas; j++ {
				schemaType := AnnotationType(goroutineID*numSchemas + j + 100) // Avoid conflicts with builtin types
				
				schema := AnnotationSchema{
					Type:        schemaType,
					Description: fmt.Sprintf("Test schema %d from goroutine %d", j, goroutineID),
					Parameters: map[string]ParameterSpec{
						"TestParam": {
							Type:         StringType,
							Required:     false,
							DefaultValue: "default",
						},
					},
				}
				
				// Try to register
				err := registry.Register(schemaType, schema)
				if err != nil {
					atomic.AddInt32(&registrationErrors, 1)
					// This might be expected if another goroutine registered first
				}
				
				// Try to lookup immediately after registration
				_, err = registry.GetSchema(schemaType)
				if err != nil {
					atomic.AddInt32(&lookupErrors, 1)
				}
				
				// Test other registry operations
				_ = registry.IsRegistered(schemaType)
				_ = registry.ListTypes()
			}
		}(i)
	}
	
	wg.Wait()
	
	t.Logf("Concurrent registry test completed: %d registration errors, %d lookup errors", 
		registrationErrors, lookupErrors)
	
	// Verify final state
	types := registry.ListTypes()
	if len(types) == 0 {
		t.Error("expected some schemas to be registered")
	}
}

// Test concurrent validation operations
func TestConcurrentValidation(t *testing.T) {
	registry := NewRegistry()
	err := RegisterBuiltinSchemas(registry)
	if err != nil {
		t.Fatalf("failed to register builtin schemas: %v", err)
	}
	
	validator := NewValidator()
	location := SourceLocation{File: "test.go", Line: 1, Column: 1}
	
	// Create test annotations for validation
	testAnnotations := []*ParsedAnnotation{
		{
			Type: CoreAnnotation,
			Parameters: map[string]interface{}{
				"Mode": "Singleton",
				"Init": "Same",
			},
			Location: location,
		},
		{
			Type: RouteAnnotation,
			Parameters: map[string]interface{}{
				"method":     "GET",
				"path":       "/users",
				"Middleware": []string{"Auth", "Logging"},
			},
			Location: location,
		},
		{
			Type: ControllerAnnotation,
			Parameters: map[string]interface{}{
				"Prefix": "/api/v1",
			},
			Location: location,
		},
	}
	
	numGoroutines := 10
	numOperations := 100
	
	var wg sync.WaitGroup
	var validationErrors int32
	
	wg.Add(numGoroutines)
	
	for i := 0; i < numGoroutines; i++ {
		go func(goroutineID int) {
			defer wg.Done()
			
			for j := 0; j < numOperations; j++ {
				annotation := testAnnotations[j%len(testAnnotations)]
				
				// Get schema
				schema, err := registry.GetSchema(annotation.Type)
				if err != nil {
					atomic.AddInt32(&validationErrors, 1)
					continue
				}
				
				// Create a copy to avoid concurrent modification
				annotationCopy := &ParsedAnnotation{
					Type:       annotation.Type,
					Parameters: make(map[string]interface{}),
					Location:   annotation.Location,
				}
				
				// Copy parameters
				for k, v := range annotation.Parameters {
					annotationCopy.Parameters[k] = v
				}
				
				// Apply defaults
				err = validator.ApplyDefaults(annotationCopy, schema)
				if err != nil {
					atomic.AddInt32(&validationErrors, 1)
					continue
				}
				
				// Transform parameters
				err = validator.TransformParameters(annotationCopy, schema)
				if err != nil {
					atomic.AddInt32(&validationErrors, 1)
					continue
				}
				
				// Validate
				err = validator.Validate(annotationCopy, schema)
				if err != nil {
					atomic.AddInt32(&validationErrors, 1)
					continue
				}
			}
		}(i)
	}
	
	wg.Wait()
	
	if validationErrors > 0 {
		t.Errorf("had %d validation errors during concurrent operations", validationErrors)
	}
	
	t.Logf("Concurrent validation test completed with %d errors", validationErrors)
}

func TestValidateDefaultValue(t *testing.T) {
	registry := &registry{
		schemas: make(map[AnnotationType]AnnotationSchema),
	}

	tests := []struct {
		name         string
		paramName    string
		paramType    ParameterType
		defaultValue interface{}
		expectError  bool
	}{
		{"valid string", "Mode", StringType, "Singleton", false},
		{"valid bool", "Required", BoolType, true, false},
		{"valid int", "Count", IntType, 42, false},
		{"valid string slice", "Tags", StringSliceType, []string{"a", "b"}, false},
		{"invalid string", "Mode", StringType, 123, true},
		{"invalid bool", "Required", BoolType, "true", true},
		{"invalid int", "Count", IntType, "42", true},
		{"invalid string slice", "Tags", StringSliceType, "a,b", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := registry.validateDefaultValue(tt.paramName, tt.paramType, tt.defaultValue)
			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}