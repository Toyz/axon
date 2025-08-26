package annotations

import (
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// Benchmark tests for parser performance
func BenchmarkParseAnnotation(b *testing.B) {
	registry := NewRegistry()
	err := RegisterBuiltinSchemas(registry)
	if err != nil {
		b.Fatalf("failed to register builtin schemas: %v", err)
	}

	parser := NewParser(registry)
	location := SourceLocation{File: "test.go", Line: 1, Column: 1}

	testCases := []struct {
		name  string
		input string
	}{
		{"simple core", "//axon::core"},
		{"core with parameters", "//axon::core -Mode=Transient -Init=Background"},
		{"simple route", "//axon::route GET /users"},
		{"complex route", "//axon::route POST /users/{id:int} -Middleware=Auth,Logging,Cache -PassContext"},
		{"controller", "//axon::controller -Prefix=/api/v1"},
		{"middleware", "//axon::middleware -Routes=/api/*,/admin/*,/public/*"},
	}

	for _, tc := range testCases {
		b.Run(tc.name, func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, err := parser.ParseAnnotation(tc.input, location)
				if err != nil {
					b.Fatalf("unexpected error: %v", err)
				}
			}
		})
	}
}

// Benchmark registry operations
func BenchmarkRegistryOperations(b *testing.B) {
	registry := NewRegistry()
	err := RegisterBuiltinSchemas(registry)
	if err != nil {
		b.Fatalf("failed to register builtin schemas: %v", err)
	}

	b.Run("GetSchema", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := registry.GetSchema(CoreAnnotation)
			if err != nil {
				b.Fatalf("unexpected error: %v", err)
			}
		}
	})

	b.Run("IsRegistered", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = registry.IsRegistered(CoreAnnotation)
		}
	})

	b.Run("ListTypes", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = registry.ListTypes()
		}
	})
}

// Benchmark validation operations
func BenchmarkValidation(b *testing.B) {
	registry := NewRegistry()
	err := RegisterBuiltinSchemas(registry)
	if err != nil {
		b.Fatalf("failed to register builtin schemas: %v", err)
	}

	validator := NewValidator()
	location := SourceLocation{File: "test.go", Line: 1, Column: 1}

	schema, err := registry.GetSchema(CoreAnnotation)
	if err != nil {
		b.Fatalf("failed to get schema: %v", err)
	}

	annotation := &ParsedAnnotation{
		Type: CoreAnnotation,
		Parameters: map[string]interface{}{
			"Mode": "Transient",
			"Init": "Background",
		},
		Location: location,
	}

	b.Run("ApplyDefaults", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			// Create a copy for each iteration
			testAnnotation := &ParsedAnnotation{
				Type:       annotation.Type,
				Parameters: make(map[string]interface{}),
				Location:   annotation.Location,
			}
			for k, v := range annotation.Parameters {
				testAnnotation.Parameters[k] = v
			}

			err := validator.ApplyDefaults(testAnnotation, schema)
			if err != nil {
				b.Fatalf("unexpected error: %v", err)
			}
		}
	})

	b.Run("TransformParameters", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			// Create a copy for each iteration
			testAnnotation := &ParsedAnnotation{
				Type:       annotation.Type,
				Parameters: make(map[string]interface{}),
				Location:   annotation.Location,
			}
			for k, v := range annotation.Parameters {
				testAnnotation.Parameters[k] = v
			}

			err := validator.TransformParameters(testAnnotation, schema)
			if err != nil {
				b.Fatalf("unexpected error: %v", err)
			}
		}
	})

	b.Run("Validate", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			// Create a copy for each iteration
			testAnnotation := &ParsedAnnotation{
				Type:       annotation.Type,
				Parameters: make(map[string]interface{}),
				Location:   annotation.Location,
			}
			for k, v := range annotation.Parameters {
				testAnnotation.Parameters[k] = v
			}

			err := validator.Validate(testAnnotation, schema)
			if err != nil {
				b.Fatalf("unexpected error: %v", err)
			}
		}
	})
}

// Benchmark type-safe getters
func BenchmarkTypeSafeGetters(b *testing.B) {
	annotation := &ParsedAnnotation{
		Type: CoreAnnotation,
		Parameters: map[string]interface{}{
			"Mode":       "Transient",
			"Init":       "Background",
			"Manual":     "CustomModule",
			"Enabled":    true,
			"Count":      42,
			"Middleware": []string{"Auth", "Logging", "Cache"},
		},
	}

	b.Run("GetString", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = annotation.GetString("Mode", "Singleton")
		}
	})

	b.Run("GetBool", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = annotation.GetBool("Enabled", false)
		}
	})

	b.Run("GetInt", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = annotation.GetInt("Count", 0)
		}
	})

	b.Run("GetStringSlice", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = annotation.GetStringSlice("Middleware", nil)
		}
	})

	b.Run("HasParameter", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = annotation.HasParameter("Mode")
		}
	})
}

// Performance test for large number of annotations
func TestParsePerformanceWithManyAnnotations(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping performance test in short mode")
	}

	registry := NewRegistry()
	err := RegisterBuiltinSchemas(registry)
	if err != nil {
		t.Fatalf("failed to register builtin schemas: %v", err)
	}

	parser := NewParser(registry)
	location := SourceLocation{File: "test.go", Line: 1, Column: 1}

	// Test with different annotation complexities
	testAnnotations := []string{
		"//axon::core",
		"//axon::core -Mode=Transient",
		"//axon::core -Mode=Singleton -Init=Background",
		"//axon::route GET /users",
		"//axon::route POST /users/{id:int}",
		"//axon::route PUT /users/{id:int} -Middleware=Auth,Logging",
		"//axon::route DELETE /users/{id:int} -Middleware=Auth,Logging,Cache -PassContext",
		"//axon::controller -Prefix=/api/v1",
		"//axon::middleware -Routes=/api/*,/admin/*",
		"//axon::interface -Name=UserService",
	}

	numAnnotations := 10000

	start := time.Now()
	var memBefore runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&memBefore)

	for i := 0; i < numAnnotations; i++ {
		annotationText := testAnnotations[i%len(testAnnotations)]
		_, err := parser.ParseAnnotation(annotationText, location)
		if err != nil {
			t.Fatalf("unexpected error at iteration %d: %v", i, err)
		}
	}

	duration := time.Since(start)
	var memAfter runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&memAfter)

	annotationsPerSecond := float64(numAnnotations) / duration.Seconds()

	// Handle potential underflow when GC runs between measurements
	var memUsed uint64
	if memAfter.Alloc > memBefore.Alloc {
		memUsed = memAfter.Alloc - memBefore.Alloc
	} else {
		memUsed = 0 // Memory usage decreased due to GC
	}

	t.Logf("Parsed %d annotations in %v", numAnnotations, duration)
	t.Logf("Performance: %.2f annotations/second", annotationsPerSecond)
	t.Logf("Memory before: %d bytes", memBefore.Alloc)
	t.Logf("Memory after: %d bytes", memAfter.Alloc)
	t.Logf("Memory used: %d bytes (%.2f KB)", memUsed, float64(memUsed)/1024)

	// Performance requirements (adjust as needed)
	if annotationsPerSecond < 1000 {
		t.Errorf("Performance too slow: %.2f annotations/second (expected > 1000)", annotationsPerSecond)
	}

	// Memory usage should be reasonable (less than 1MB for 10k annotations)
	// Only check if memory actually increased
	if memUsed > 1024*1024 {
		t.Errorf("Memory usage too high: %d bytes (expected < 1MB)", memUsed)
	}
}

// Stress test with concurrent parsing
func TestConcurrentParsingStress(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping stress test in short mode")
	}

	registry := NewRegistry()
	err := RegisterBuiltinSchemas(registry)
	if err != nil {
		t.Fatalf("failed to register builtin schemas: %v", err)
	}

	parser := NewParser(registry)
	location := SourceLocation{File: "test.go", Line: 1, Column: 1}

	testAnnotations := []string{
		"//axon::core -Mode=Transient -Init=Background",
		"//axon::route POST /users/{id:int} -Middleware=Auth,Logging,Cache -PassContext",
		"//axon::controller -Prefix=/api/v1",
		"//axon::middleware -Routes=/api/*,/admin/*,/public/*",
		"//axon::interface -Name=UserService",
	}

	numGoroutines := 50
	numOperationsPerGoroutine := 1000

	var wg sync.WaitGroup
	var totalErrors int64
	var totalSuccess int64

	start := time.Now()

	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func(goroutineID int) {
			defer wg.Done()

			var localErrors int64
			var localSuccess int64

			for j := 0; j < numOperationsPerGoroutine; j++ {
				annotationText := testAnnotations[j%len(testAnnotations)]

				annotation, err := parser.ParseAnnotation(annotationText, location)
				if err != nil {
					localErrors++
					continue
				}

				// Validate the result
				if annotation == nil {
					localErrors++
					continue
				}

				// Test type-safe getters
				_ = annotation.GetString("Mode", "Singleton")
				_ = annotation.GetBool("PassContext", false)
				_ = annotation.GetStringSlice("Middleware", nil)

				localSuccess++
			}

			// Update global counters atomically
			atomic.AddInt64(&totalErrors, localErrors)
			atomic.AddInt64(&totalSuccess, localSuccess)
		}(i)
	}

	wg.Wait()
	duration := time.Since(start)

	totalOperations := int64(numGoroutines * numOperationsPerGoroutine)
	operationsPerSecond := float64(totalOperations) / duration.Seconds()

	t.Logf("Stress test completed:")
	t.Logf("  Goroutines: %d", numGoroutines)
	t.Logf("  Operations per goroutine: %d", numOperationsPerGoroutine)
	t.Logf("  Total operations: %d", totalOperations)
	t.Logf("  Duration: %v", duration)
	t.Logf("  Operations/second: %.2f", operationsPerSecond)
	t.Logf("  Successful operations: %d", totalSuccess)
	t.Logf("  Failed operations: %d", totalErrors)
	t.Logf("  Success rate: %.2f%%", float64(totalSuccess)/float64(totalOperations)*100)

	if totalErrors > 0 {
		t.Errorf("had %d errors during stress test", totalErrors)
	}

	if totalSuccess != totalOperations {
		t.Errorf("success count mismatch: expected %d, got %d", totalOperations, totalSuccess)
	}

	// Performance requirement: should handle at least 10k operations/second under stress
	if operationsPerSecond < 10000 {
		t.Errorf("Performance under stress too slow: %.2f ops/second (expected > 10000)", operationsPerSecond)
	}
}

// Memory leak detection test
func TestMemoryLeakDetection(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping memory leak test in short mode")
	}

	registry := NewRegistry()
	err := RegisterBuiltinSchemas(registry)
	if err != nil {
		t.Fatalf("failed to register builtin schemas: %v", err)
	}

	parser := NewParser(registry)
	location := SourceLocation{File: "test.go", Line: 1, Column: 1}

	// Force garbage collection and get baseline
	runtime.GC()
	runtime.GC() // Call twice to ensure cleanup
	var memBefore runtime.MemStats
	runtime.ReadMemStats(&memBefore)

	// Perform many parsing operations
	numIterations := 10000
	for i := 0; i < numIterations; i++ {
		annotation, err := parser.ParseAnnotation("//axon::route POST /users/{id:int} -Middleware=Auth,Logging -PassContext", location)
		if err != nil {
			t.Fatalf("unexpected error at iteration %d: %v", i, err)
		}

		// Use the annotation to prevent optimization
		_ = annotation.GetString("method", "GET")
		_ = annotation.GetStringSlice("Middleware", nil)
		_ = annotation.HasParameter("PassContext")
	}

	// Force garbage collection and measure memory
	runtime.GC()
	runtime.GC()
	var memAfter runtime.MemStats
	runtime.ReadMemStats(&memAfter)

	// Handle potential underflow when GC runs between measurements
	var memGrowth int64
	if memAfter.Alloc > memBefore.Alloc {
		memGrowth = int64(memAfter.Alloc - memBefore.Alloc)
	} else {
		memGrowth = -int64(memBefore.Alloc - memAfter.Alloc) // Negative growth (GC cleaned up)
	}

	t.Logf("Memory before: %d bytes", memBefore.Alloc)
	t.Logf("Memory after: %d bytes", memAfter.Alloc)
	t.Logf("Memory growth: %d bytes", memGrowth)
	if memGrowth > 0 {
		t.Logf("Memory growth per operation: %.2f bytes", float64(memGrowth)/float64(numIterations))
	} else {
		t.Logf("Memory decreased (GC effect): %d bytes", -memGrowth)
	}

	// Memory growth should be minimal (less than 50KB total for 10k operations)
	// Only check if memory actually increased
	maxAllowedGrowth := int64(50 * 1024) // 50KB
	if memGrowth > maxAllowedGrowth {
		t.Errorf("Potential memory leak detected: memory grew by %d bytes (max allowed: %d)",
			memGrowth, maxAllowedGrowth)
	}
}

// Test parser behavior with malformed inputs under stress
func TestMalformedInputStress(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping malformed input stress test in short mode")
	}

	registry := NewRegistry()
	err := RegisterBuiltinSchemas(registry)
	if err != nil {
		t.Fatalf("failed to register builtin schemas: %v", err)
	}

	parser := NewParser(registry)
	location := SourceLocation{File: "test.go", Line: 1, Column: 1}

	// Various malformed inputs that should be handled gracefully
	malformedInputs := []string{
		"",
		"//",
		"//axon",
		"//axon:",
		"//axon::",
		"//axon::unknown",
		"//axon::core -",
		"//axon::core -=",
		"//axon::core -Mode",
		"//axon::core -Mode=",
		"//axon::core -Mode=Invalid",
		"//axon::route",
		"//axon::route GET",
		"//axon::route INVALID /path",
		"//axon::route GET invalid_path",
		"/axon::core",
		"axon::core",
		"//annotation::core",
		"//axon::core -Mode=Singleton -Mode=Transient", // Duplicate parameter
		"//axon::core -UnknownParam=Value",
		"//axon::route GET /users -Middleware=",
		"//axon::route GET /users -Middleware=,",
		"//axon::route GET /users -Middleware=Auth,",
		"//axon::route GET /users -Middleware=,Auth",
	}

	numIterations := 1000
	var totalErrors int
	var panicCount int

	for i := 0; i < numIterations; i++ {
		for _, input := range malformedInputs {
			func() {
				defer func() {
					if r := recover(); r != nil {
						panicCount++
						t.Errorf("Parser panicked on input %q: %v", input, r)
					}
				}()

				_, err := parser.ParseAnnotation(input, location)
				if err != nil {
					totalErrors++
					// This is expected for malformed inputs
				}
			}()
		}
	}

	expectedErrors := numIterations * len(malformedInputs)

	t.Logf("Malformed input stress test completed:")
	t.Logf("  Iterations: %d", numIterations)
	t.Logf("  Malformed inputs per iteration: %d", len(malformedInputs))
	t.Logf("  Total operations: %d", expectedErrors)
	t.Logf("  Total errors: %d", totalErrors)
	t.Logf("  Panics: %d", panicCount)

	if panicCount > 0 {
		t.Errorf("Parser should not panic on malformed input, but had %d panics", panicCount)
	}

	// Most malformed inputs should produce errors
	if totalErrors < expectedErrors/2 {
		t.Errorf("Expected more errors for malformed inputs: got %d, expected at least %d",
			totalErrors, expectedErrors/2)
	}
}
