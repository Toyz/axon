# Design Document

## Overview

This design document outlines the architecture for a robust, extensible annotation parsing system that will replace the current fragile parsing logic in Axon. The new system will provide type-safe parameter handling, clear error reporting, and extensibility for future annotation types.

## Architecture

### Package Structure

```
internal/annotations/
├── types.go          // AnnotationType enum and core types
├── registry.go       // AnnotationRegistry implementation
├── parser.go         // ParserEngine implementation
├── validator.go      // SchemaValidator implementation
├── schemas.go        // Built-in annotation schemas
├── errors.go         // Error types and handling
└── parser_test.go    // Comprehensive tests
```

### High-Level Architecture

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   Annotation    │    │   Schema        │    │   Parser        │
│   Registry      │    │   Validator     │    │   Engine        │
│                 │    │                 │    │                 │
│ - Register      │    │ - Type Check    │    │ - Tokenize      │
│ - Lookup        │    │ - Validate      │    │ - Parse         │
│ - Schema Mgmt   │    │ - Transform     │    │ - Error Report  │
└─────────────────┘    └─────────────────┘    └─────────────────┘
         │                       │                       │
         └───────────────────────┼───────────────────────┘
                                 │
                    ┌─────────────────┐
                    │   Parsed        │
                    │   Annotation    │
                    │                 │
                    │ - Type          │
                    │ - Parameters    │
                    │ - Metadata      │
                    └─────────────────┘
```

### Package Dependencies

```
internal/parser/
├── parser.go         // Current parser (to be replaced)
└── ...

internal/annotations/ // New annotation parsing system
├── types.go
├── registry.go
├── parser.go
├── validator.go
├── schemas.go
└── errors.go

internal/cli/
├── generator.go      // Uses annotations package
└── ...

internal/generator/
├── generator.go      // Uses annotations package
└── ...
```

### Core Components

#### 1. Annotation Registry
- **Purpose**: Central registry for all supported annotation types and their schemas
- **Responsibilities**:
  - Register annotation types with their parameter schemas
  - Provide schema lookup for validation
  - Manage backward compatibility
  - Support schema evolution

#### 2. Schema Validator
- **Purpose**: Type-safe validation of annotation parameters
- **Responsibilities**:
  - Validate parameter types according to schema
  - Apply default values for optional parameters
  - Run custom validation logic
  - Generate descriptive error messages

#### 3. Parser Engine
- **Purpose**: Core parsing logic for annotation syntax
- **Responsibilities**:
  - Tokenize annotation strings
  - Parse parameters and flags
  - Handle quoted strings and escape sequences
  - Provide detailed error locations

#### 4. Parsed Annotation
- **Purpose**: Type-safe representation of parsed annotations
- **Responsibilities**:
  - Store annotation type and parameters
  - Provide type-safe parameter access
  - Include source location metadata

## Components and Interfaces

### Package: internal/annotations

#### types.go

```go
package annotations

type AnnotationType int

const (
    CoreAnnotation AnnotationType = iota
    RouteAnnotation
    ControllerAnnotation
    MiddlewareAnnotation
    InterfaceAnnotation
    // Add new annotation types here
)

func (a AnnotationType) String() string {
    switch a {
    case CoreAnnotation:
        return "core"
    case RouteAnnotation:
        return "route"
    case ControllerAnnotation:
        return "controller"
    case MiddlewareAnnotation:
        return "middleware"
    case InterfaceAnnotation:
        return "interface"
    default:
        return "unknown"
    }
}

// ParseAnnotationType converts string to AnnotationType
func ParseAnnotationType(s string) (AnnotationType, error) {
    switch s {
    case "core":
        return CoreAnnotation, nil
    case "route":
        return RouteAnnotation, nil
    case "controller":
        return ControllerAnnotation, nil
    case "middleware":
        return MiddlewareAnnotation, nil
    case "interface":
        return InterfaceAnnotation, nil
    default:
        return 0, fmt.Errorf("unknown annotation type: %s", s)
    }
}

type SourceLocation struct {
    File   string
    Line   int
    Column int
}

type ParsedAnnotation struct {
    Type       AnnotationType         // Annotation type enum
    Target     string                 // Target struct/function name
    Parameters map[string]interface{} // Typed parameters
    Flags      []string              // Boolean flags
    Location   SourceLocation        // Source location
    Raw        string                // Original annotation text
}
```

#### registry.go

```go
package annotations

type AnnotationRegistry interface {
    // Register a new annotation type with its schema
    Register(annotationType AnnotationType, schema AnnotationSchema) error
    
    // Get schema for an annotation type
    GetSchema(annotationType AnnotationType) (AnnotationSchema, error)
    
    // List all registered annotation types
    ListTypes() []AnnotationType
    
    // Check if annotation type is registered
    IsRegistered(annotationType AnnotationType) bool
}

// DefaultRegistry returns the global annotation registry
func DefaultRegistry() AnnotationRegistry {
    return defaultRegistry
}

// NewRegistry creates a new annotation registry
func NewRegistry() AnnotationRegistry {
    return &registry{
        schemas: make(map[AnnotationType]AnnotationSchema),
    }
}

type AnnotationSchema struct {
    Type        AnnotationType            // Annotation type enum
    Description string                    // Human-readable description
    Parameters  map[string]ParameterSpec  // Parameter specifications
    Validators  []CustomValidator         // Custom validation functions
    Examples    []string                  // Usage examples
}

type ParameterSpec struct {
    Type         ParameterType  // string, bool, int, []string, etc.
    Required     bool           // Whether parameter is required
    DefaultValue interface{}    // Default value if not provided
    Description  string         // Parameter description
    Validator    func(interface{}) error // Custom validator
}

type ParameterType int

const (
    StringType ParameterType = iota
    BoolType
    IntType
    StringSliceType
    // Add more types as needed
)
```

#### parser.go

```go
package annotations

type ParserEngine interface {
    // Parse a single annotation comment
    ParseAnnotation(comment string, location SourceLocation) (*ParsedAnnotation, error)
    
    // Parse multiple annotations from a file
    ParseFile(filePath string) ([]*ParsedAnnotation, error)
    
    // Validate parsed annotation against schema
    ValidateAnnotation(annotation *ParsedAnnotation) error
}

// NewParser creates a new parser engine with the given registry
func NewParser(registry AnnotationRegistry) ParserEngine {
    return &parser{
        registry: registry,
    }
}

// Parsing Logic Details:
// 1. Handle flexible comment prefixes: "//axon::", "// axon::", "//  axon::"
// 2. Strip leading dashes from parameter names: "-Mode=Transient" becomes "Mode": "Transient"
// 3. Support both flag formats: "-Init" (boolean) and "-Mode=Value" (key-value)
// 4. Normalize whitespace and handle quoted strings properly
```

#### validator.go

```go
package annotations

type SchemaValidator interface {
    // Validate annotation against its schema
    Validate(annotation *ParsedAnnotation, schema AnnotationSchema) error
    
    // Apply default values for missing optional parameters
    ApplyDefaults(annotation *ParsedAnnotation, schema AnnotationSchema) error
    
    // Transform parameter values to correct types
    TransformParameters(annotation *ParsedAnnotation, schema AnnotationSchema) error
}

type ValidationError struct {
    Parameter string        // Parameter name that failed validation
    Expected  string        // What was expected
    Actual    string        // What was provided
    Location  SourceLocation // Where the error occurred
    Suggestion string       // Suggested fix
}

func (e ValidationError) Error() string {
    return fmt.Sprintf("%s:%d:%d: parameter '%s' validation failed: expected %s, got %s. %s",
        e.Location.File, e.Location.Line, e.Location.Column,
        e.Parameter, e.Expected, e.Actual, e.Suggestion)
}

// NewValidator creates a new schema validator
func NewValidator() SchemaValidator {
    return &validator{}
}
```

## Data Models

#### errors.go

```go
package annotations

type AnnotationError interface {
    error
    Location() SourceLocation
    Suggestion() string
    Code() ErrorCode
}

type ErrorCode int

const (
    SyntaxError ErrorCode = iota
    ValidationError
    SchemaError
    RegistrationError
)

type SyntaxError struct {
    Msg        string
    Loc        SourceLocation
    Suggestion string
}

func (e SyntaxError) Error() string {
    return fmt.Sprintf("%s:%d:%d: syntax error: %s. %s",
        e.Loc.File, e.Loc.Line, e.Loc.Column, e.Msg, e.Suggestion)
}

func (e SyntaxError) Location() SourceLocation { return e.Loc }
func (e SyntaxError) Suggestion() string { return e.Suggestion }
func (e SyntaxError) Code() ErrorCode { return SyntaxError }
```

## Data Models

### Parsing Rules and Normalization

The parser will handle several normalization steps to make the system more user-friendly:

#### Comment Prefix Normalization
```go
// All of these formats will be accepted:
"//axon::core -Mode=Transient"           // Standard format
"// axon::core -Mode=Transient"          // Go linter adds space
"//  axon::core -Mode=Transient"         // Multiple spaces
"//\taxon::core -Mode=Transient"         // Tab character
```

#### Parameter Name Normalization
```go
// Input: "-Mode=Transient"
// Stored as: "Mode": "Transient"

// Input: "-Init=Background"  
// Stored as: "Init": "Background"

// Input: "-Init" (legacy boolean flag support)
// Stored as: "Init": "Same" (default value applied)

// Input: "-Middleware=Auth,Logging"
// Stored as: "Middleware": []string{"Auth", "Logging"}
```

#### Parsing Algorithm
1. **Normalize comment prefix**: Strip `//`, handle optional whitespace, find `axon::`
2. **Extract annotation type**: Parse the type after `axon::`
3. **Tokenize parameters**: Split remaining text into tokens
4. **Process flags**: Convert `-ParamName=Value` to `ParamName: Value`
5. **Handle legacy boolean flags**: Convert `-FlagName` to `FlagName: <default_value>` based on schema
6. **Type conversion**: Apply schema-based type conversion and comma-splitting for arrays
7. **Validation**: Validate against registered schema with custom validators

### Core Data Structures

```go
// AnnotationToken represents a parsed token from annotation text
type AnnotationToken struct {
    Type     TokenType
    Value    string
    Position int
}

type TokenType int

const (
    AnnotationPrefixToken TokenType = iota  // "axon::"
    TypeToken                               // "core", "route", etc.
    ParameterToken                          // "GET", "/users/{id:int}"
    FlagToken                              // "-Mode=Transient"
    BoolFlagToken                          // "-Init"
    QuotedStringToken                      // "\"Custom Module\""
    CommaToken                             // ","
    EOFToken
)

// ParseContext holds state during parsing
type ParseContext struct {
    Input    string
    Position int
    Line     int
    Column   int
    Tokens   []AnnotationToken
    Errors   []ParseError
}

type ParseError struct {
    Message    string
    Location   SourceLocation
    Suggestion string
}
```

#### schemas.go

```go
package annotations

// Built-in annotation schemas
var CoreAnnotationSchema = AnnotationSchema{
    Type: CoreAnnotation,
    Description: "Marks a struct as a core service for dependency injection",
    Parameters: map[string]ParameterSpec{
        "Mode": {
            Type:         StringType,
            Required:     false,
            DefaultValue: "Singleton",
            Description:  "Service lifecycle mode",
            Validator: func(v interface{}) error {
                mode := v.(string)
                if mode != "Singleton" && mode != "Transient" {
                    return fmt.Errorf("must be 'Singleton' or 'Transient'")
                }
                return nil
            },
        },
        "Init": {
            Type:        StringType,
            Required:    false,
            DefaultValue: "Same",
            Description: "Lifecycle execution mode: 'Same' (default, synchronous) or 'Background' (async)",
            Validator: func(v interface{}) error {
                mode := v.(string)
                if mode != "Same" && mode != "Background" {
                    return fmt.Errorf("must be 'Same' or 'Background'")
                }
                return nil
            },
        },
        "Manual": {
            Type:        StringType,
            Required:    false,
            Description: "Custom module name for manual registration",
        },
    },
    Examples: []string{
        "//axon::core",
        "// axon::core -Mode=Transient",                    // Go linter format
        "//axon::core -Init=Background",                    // Background lifecycle
        "//axon::core -Init=Same -Manual=\"CustomModule\"", // Same lifecycle with manual module
    },
}

var RouteAnnotationSchema = AnnotationSchema{
    Type: RouteAnnotation,
    Description: "Defines an HTTP route handler",
    Parameters: map[string]ParameterSpec{
        "method": {
            Type:     StringType,
            Required: true,
            Description: "HTTP method (GET, POST, etc.)",
            Validator: func(v interface{}) error {
                method := v.(string)
                validMethods := []string{"GET", "POST", "PUT", "DELETE", "PATCH", "HEAD", "OPTIONS"}
                for _, valid := range validMethods {
                    if method == valid {
                        return nil
                    }
                }
                return fmt.Errorf("must be one of: %s", strings.Join(validMethods, ", "))
            },
        },
        "path": {
            Type:     StringType,
            Required: true,
            Description: "URL path pattern",
        },
        "Middleware": {
            Type:        StringSliceType,
            Required:    false,
            Description: "Comma-separated list of middleware names",
        },
        "PassContext": {
            Type:        BoolType,
            Required:    false,
            DefaultValue: false,
            Description: "Whether to pass echo.Context to handler",
        },
    },
    Examples: []string{
        "//axon::route GET /users",
        "// axon::route POST /users/{id:int} -Middleware=Auth,Logging",  // Go linter format
        "//axon::route GET /health -PassContext",
    },
}

// RegisterBuiltinSchemas registers all built-in annotation schemas
func RegisterBuiltinSchemas(registry AnnotationRegistry) error {
    schemas := []AnnotationSchema{
        CoreAnnotationSchema,
        RouteAnnotationSchema,
        // Add other built-in schemas here
    }
    
    for _, schema := range schemas {
        if err := registry.Register(schema.Type, schema); err != nil {
            return fmt.Errorf("failed to register %s schema: %w", schema.Type, err)
        }
    }
    
    return nil
}
```

### Usage Examples

With the new design, parameter access becomes much cleaner:

```go
// Before (current system):
if modeFlag, exists := annotation.Parameters["-Mode"]; exists {
    service.Mode = modeFlag
}

// After (new system):
if mode, ok := annotation.Parameters["Mode"].(string); ok {
    service.Mode = mode
}

// Or even better with type-safe getters:
mode := annotation.GetString("Mode", "Singleton")        // with default
initMode := annotation.GetString("Init", "Same")         // lifecycle execution mode
middlewares := annotation.GetStringSlice("Middleware", nil)
```

### Integration with Existing Code

The new `internal/annotations` package will be integrated into the existing codebase as follows:

```go
// internal/parser/parser.go (updated to use new annotations package)
package parser

import (
    "github.com/toyz/axon/internal/annotations"
)

type Parser struct {
    annotationParser annotations.ParserEngine
    registry        annotations.AnnotationRegistry
}

func NewParser() *Parser {
    registry := annotations.NewRegistry()
    if err := annotations.RegisterBuiltinSchemas(registry); err != nil {
        panic(fmt.Sprintf("failed to register builtin schemas: %v", err))
    }
    
    return &Parser{
        annotationParser: annotations.NewParser(registry),
        registry:        registry,
    }
}
```

## Error Handling

### Error Recovery and Reporting

The parser will implement error recovery strategies:

1. **Continue parsing after syntax errors** to find multiple issues
2. **Provide context-aware suggestions** based on common mistakes
3. **Group related errors** to avoid overwhelming users
4. **Include examples** of correct syntax in error messages

## Testing Strategy

### Unit Testing Approach

1. **Parser Engine Tests**
   - Test tokenization of various annotation formats
   - Test parameter extraction and type conversion
   - Test error handling and recovery
   - Test edge cases and malformed input

2. **Schema Validator Tests**
   - Test parameter type validation
   - Test required parameter checking
   - Test default value application
   - Test custom validator execution

3. **Registry Tests**
   - Test annotation type registration
   - Test schema lookup and caching
   - Test backward compatibility
   - Test concurrent access

4. **Integration Tests**
   - Test complete parsing workflow
   - Test with real annotation examples
   - Test performance with large files
   - Test error reporting quality

### Test Data Strategy

```go
// Test cases will be organized by annotation type and scenario
var testCases = []struct {
    name        string
    input       string
    expected    *ParsedAnnotation
    expectError bool
    errorType   ErrorCode
}{
    {
        name:  "core annotation with mode",
        input: "//axon::core -Mode=Transient",
        expected: &ParsedAnnotation{
            Type: CoreAnnotation,
            Parameters: map[string]interface{}{
                "Mode": "Transient",
            },
        },
    },
    {
        name:        "invalid mode value",
        input:       "//axon::core -Mode=Invalid",
        expectError: true,
        errorType:   ValidationError,
    },
    // ... more test cases
}
```

### Performance Testing

- **Benchmark parsing speed** with files of varying sizes
- **Memory usage profiling** to prevent leaks
- **Concurrent parsing tests** to ensure thread safety
- **Regression tests** to catch performance degradation

## Implementation Plan

### Phase 1: Core Parser Engine
1. Implement tokenizer for annotation syntax
2. Build basic parameter parsing logic
3. Add error reporting infrastructure
4. Create initial test suite

### Phase 2: Schema System
1. Design and implement annotation registry
2. Create schema validation framework
3. Add type conversion and default value handling
4. Implement custom validator support

### Phase 3: Built-in Annotations
1. Define schemas for existing annotation types
2. Migrate current parsing logic to new system
3. Ensure backward compatibility
4. Add comprehensive error messages

### Phase 4: Integration and Testing
1. Integrate new parser into existing codebase
2. Run extensive testing with real-world examples
3. Performance optimization and profiling
4. Documentation and examples

### Phase 5: Advanced Features
1. Add caching for improved performance
2. Implement concurrent parsing support
3. Add IDE integration helpers (if needed)
4. Create migration tools for complex cases

## Migration Strategy

### Backward Compatibility Approach

1. **Dual Parser Support**: Run both old and new parsers in parallel during transition
2. **Gradual Migration**: Migrate annotation types one at a time
3. **Validation Mode**: Compare results between parsers to catch regressions
4. **Fallback Mechanism**: Fall back to old parser if new parser fails

### Migration Steps

1. **Phase 1**: Implement new parser alongside existing one
2. **Phase 2**: Add feature flag to enable new parser for testing
3. **Phase 3**: Run both parsers and compare results in CI
4. **Phase 4**: Switch to new parser by default with fallback option
5. **Phase 5**: Remove old parser after confidence period

This design provides a solid foundation for a robust, extensible annotation parsing system that addresses all the requirements while maintaining backward compatibility and providing clear migration path.