# Design Document

## Overview

The `axon::route_parser` feature extends the Axon framework's routing system with custom parameter parsing and validation capabilities. This feature allows developers to define reusable parsers for complex parameter types, providing automatic validation and type conversion before parameters reach controller methods.

## Architecture

### Parser Registry System

The framework will maintain a global parser registry that maps type names to parser functions. This registry is populated during the code generation phase by scanning for `//axon::route_parser` annotations.

```go
type ParserRegistry struct {
    parsers map[string]ParserMetadata
    mu      sync.RWMutex
}

type ParserMetadata struct {
    TypeName     string
    FunctionName string
    PackagePath  string
    ImportPath   string
}
```

### Parser Function Signature

All parser functions must conform to the signature:
```go
func(c echo.Context, paramValue string) (T, error)
```

Where `T` is the target type specified in the annotation.

### Code Generation Flow

1. **Discovery Phase**: Scan all packages for `//axon::route_parser` annotations
2. **Validation Phase**: Validate parser function signatures and check for conflicts
3. **Registry Building**: Build a registry of available parsers with their metadata
4. **Route Processing**: When processing routes, check for custom parameter types
5. **Code Generation**: Generate route wrappers that call appropriate parsers

## Components and Interfaces

### Parser Annotation

```go
// Annotation format: //axon::route_parser <TypeName>
// Example: //axon::route_parser UUID
func ParseUUID(c echo.Context, paramValue string) (uuid.UUID, error) {
    return uuid.Parse(paramValue)
}
```

### Parser Registry Interface

```go
type ParserRegistryInterface interface {
    RegisterParser(typeName string, metadata ParserMetadata) error
    GetParser(typeName string) (ParserMetadata, bool)
    ListParsers() []string
    ValidateParser(funcDecl *ast.FuncDecl) error
}
```

### Route Parameter Enhancement

Extend the existing route parameter parsing to support custom types:

```go
type RouteParameter struct {
    Name         string
    Type         string
    IsCustomType bool
    ParserFunc   string // Function name for custom parsers
    ImportPath   string // Import path for custom types
}
```

### Generated Code Structure

For a route with custom parsers, the generator will create:

```go
func GetUserWrapper(c echo.Context) error {
    // Parse custom parameter using registered parser
    id, err := parsers.ParseUUID(c, c.Param("id"))
    if err != nil {
        return echo.NewHTTPError(http.StatusBadRequest, 
            fmt.Sprintf("Invalid parameter 'id': %v", err))
    }
    
    // Call the actual handler
    result, err := controller.GetUser(id)
    if err != nil {
        return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
    }
    
    return c.JSON(http.StatusOK, result)
}
```

## Data Models

### Parser Metadata Model

```go
type ParserMetadata struct {
    // Basic information
    TypeName     string `json:"type_name"`
    FunctionName string `json:"function_name"`
    PackagePath  string `json:"package_path"`
    ImportPath   string `json:"import_path"`
    
    // Function signature validation
    ParameterTypes []string `json:"parameter_types"`
    ReturnTypes    []string `json:"return_types"`
    
    // Source location for error reporting
    FileName string `json:"file_name"`
    Line     int    `json:"line"`
}
```

### Enhanced Route Metadata

Extend existing `RouteMetadata` to include parser information:

```go
type RouteMetadata struct {
    // Existing fields...
    Method     string
    Path       string
    Handler    string
    
    // New parser-related fields
    Parameters []RouteParameter `json:"parameters"`
    HasCustomParsers bool       `json:"has_custom_parsers"`
    RequiredImports  []string   `json:"required_imports"`
}
```

### Built-in Parser Registry

```go
var BuiltinParsers = map[string]ParserMetadata{
    "uuid.UUID": {
        TypeName:     "uuid.UUID",
        FunctionName: "parseUUID",
        ImportPath:   "github.com/google/uuid",
    },
    "time.Time": {
        TypeName:     "time.Time", 
        FunctionName: "parseTime",
        ImportPath:   "time",
    },
}
```

## Error Handling

### Parser Discovery Errors

- **Duplicate Parser Registration**: Clear error showing conflicting parser locations
- **Invalid Function Signature**: Detailed error with expected vs actual signature
- **Missing Type Information**: Error when type cannot be determined from function signature

### Runtime Parser Errors

- **Parameter Parsing Failure**: HTTP 400 with descriptive error message
- **Missing Parser**: Code generation error with available parser suggestions
- **Import Resolution**: Clear error messages for missing imports

### Error Message Format

```go
type ParserError struct {
    Type        string `json:"type"`
    Message     string `json:"message"`
    FileName    string `json:"file_name,omitempty"`
    Line        int    `json:"line,omitempty"`
    Suggestions []string `json:"suggestions,omitempty"`
}
```

## Testing Strategy

### Unit Tests

1. **Parser Registry Tests**
   - Test parser registration and retrieval
   - Test conflict detection
   - Test signature validation

2. **Code Generation Tests**
   - Test route wrapper generation with custom parsers
   - Test import generation
   - Test error handling in generated code

3. **Parser Function Tests**
   - Test built-in parsers (UUID, time, etc.)
   - Test custom parser examples
   - Test error conditions and edge cases

### Integration Tests

1. **End-to-End Route Testing**
   - Test routes with custom parsers
   - Test parameter validation
   - Test error responses

2. **Multi-Package Parser Discovery**
   - Test parser discovery across packages
   - Test import resolution
   - Test parser precedence

3. **Mixed Parameter Types**
   - Test routes with both built-in and custom parsers
   - Test complex parameter combinations
   - Test middleware integration

### Example Test Cases

```go
func TestUUIDParser(t *testing.T) {
    tests := []struct {
        name        string
        input       string
        expectError bool
        expected    uuid.UUID
    }{
        {
            name:     "valid UUID",
            input:    "123e4567-e89b-12d3-a456-426614174000",
            expected: uuid.MustParse("123e4567-e89b-12d3-a456-426614174000"),
        },
        {
            name:        "invalid UUID",
            input:       "invalid-uuid",
            expectError: true,
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result, err := ParseUUID(nil, tt.input)
            if tt.expectError {
                assert.Error(t, err)
            } else {
                assert.NoError(t, err)
                assert.Equal(t, tt.expected, result)
            }
        })
    }
}
```

## Implementation Phases

### Phase 1: Core Parser Registry
- Implement parser registry system
- Add parser annotation parsing
- Create basic validation logic

### Phase 2: Built-in Parsers
- Implement UUID parser
- Add time.Time parser
- Create common validation patterns

### Phase 3: Code Generation Integration
- Extend route processing to detect custom types
- Generate parser calls in route wrappers
- Add import management for parser packages

### Phase 4: Advanced Features
- Composite type parsers
- Parser chaining and composition
- Performance optimizations

### Phase 5: Documentation and Examples
- Create comprehensive examples
- Add parser development guide
- Performance benchmarks and best practices