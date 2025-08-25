# Implementation Plan

- [x] 1. Set up package structure and core types
  - Create `internal/annotations` package directory
  - Define `AnnotationType` enum with all current annotation types
  - Implement `String()` and `ParseAnnotationType()` methods
  - Define core data structures: `SourceLocation`, `ParsedAnnotation`, `ParameterSpec`, `AnnotationSchema`
  - _Requirements: 3.1, 3.4_

- [x] 2. Implement annotation registry system
  - Create `AnnotationRegistry` interface and implementation
  - Implement thread-safe registry with schema storage
  - Add registration, lookup, and validation methods
  - Create `NewRegistry()` and `DefaultRegistry()` functions
  - _Requirements: 3.1, 3.2, 5.5_

- [x] 3. Build core tokenizer and parser engine
  - Implement flexible comment prefix parsing (handle `//axon::`, `// axon::`, etc.)
  - Create tokenizer for annotation syntax with proper error positions
  - Build parameter parsing logic with dash stripping (`-Mode` → `Mode`)
  - Handle quoted strings and escape sequences correctly
  - Implement parameter value parsing (`-Init=Background` → `Init: "Background"`)
  - Support legacy boolean flag compatibility (`-Init` → `Init: "Same"` using schema defaults)
  - _Requirements: 1.1, 1.2, 1.3, 1.4, 4.1_

- [x] 4. Implement schema validator
  - Create `SchemaValidator` interface and implementation
  - Add parameter type validation (string, bool, int, []string)
  - Implement required parameter checking
  - Add default value application for optional parameters
  - Support custom validator functions
  - _Requirements: 2.1, 2.2, 2.3, 2.4, 2.5_

- [x] 5. Create comprehensive error handling system
  - Define `AnnotationError` interface with location and suggestion support
  - Implement specific error types: `SyntaxError`, `ValidationError`, `SchemaError`
  - Add error recovery and multiple error collection
  - Create context-aware error messages with fix suggestions
  - Include file name and line number in all errors
  - _Requirements: 4.1, 4.2, 4.3, 4.4, 4.5_

- [x] 6. Define built-in annotation schemas
  - Create `CoreAnnotationSchema` with Mode, Init, Manual parameters
  - Create `RouteAnnotationSchema` with method, path, Middleware, PassContext parameters
  - Add schemas for Controller, Middleware, and Interface annotations
  - Implement `RegisterBuiltinSchemas()` function
  - Add comprehensive examples for each schema
  - _Requirements: 6.1, 6.2, 6.3, 6.4, 3.3_

- [x] 7. Add type-safe parameter access methods
  - Implement getter methods on `ParsedAnnotation`: `GetString()`, `GetBool()`, `GetStringSlice()`
  - Add default value support in getter methods
  - Create type conversion utilities for parameter values
  - Add parameter existence checking methods
  - _Requirements: 2.5, 1.2_

- [x] 8. Write comprehensive unit tests
  - Test tokenizer with various comment formats and edge cases
  - Test parameter parsing with all supported types and formats
  - Test schema validation with valid and invalid inputs
  - Test error handling and message quality
  - Test concurrent access to registry
  - _Requirements: 5.1, 5.4, 5.5_

- [x] 9. Create integration layer with existing parser
  - Update `internal/parser/parser.go` to use new annotations package
  - Create adapter functions to maintain existing API compatibility
  - Add feature flag to switch between old and new parsers
  - Implement fallback mechanism for migration safety
  - _Requirements: 6.5, 5.3_

- [ ] 10. Add performance optimizations and caching
  - Implement schema caching in registry
  - Add parsed annotation caching for repeated parsing
  - Optimize tokenizer performance for large files
  - Add memory usage monitoring and cleanup
  - _Requirements: 5.1, 5.2, 5.4_

- [ ] 11. Create migration and testing utilities
  - Build comparison tool to validate old vs new parser results
  - Create test harness for running both parsers in parallel
  - Add performance benchmarking suite
  - Create migration guide and examples
  - _Requirements: 5.1, 6.5_

- [ ] 12. Integration testing and validation
  - Test with complete-app example to ensure compatibility
  - Run full test suite with new parser enabled
  - Validate all existing annotation formats still work
  - Test error message quality with real-world examples
  - Performance testing with large codebases
  - _Requirements: 6.1, 6.2, 6.3, 6.4, 6.5, 5.1_