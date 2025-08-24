# Design Document

## Overview

The Axon framework's dependency injection system has a bug where multiple `//axon::inject` annotations on struct fields are not being properly processed. The issue appears to be in the dependency extraction logic in the parser, where only some of the annotated fields are being detected and included in the generated constructor functions.

## Architecture

The dependency injection system in Axon consists of several components:

1. **Parser (`internal/parser/parser.go`)**: Scans Go source files and extracts `//axon::inject` annotations from struct fields
2. **Generator (`internal/generator/generator.go`)**: Converts parsed metadata into constructor functions
3. **Templates (`internal/templates/templates.go`)**: Contains Go templates for generating the actual constructor code

## Root Cause Analysis

Based on code analysis, the issue is likely in the `extractDependencies` function in `internal/parser/parser.go`. The function processes struct fields and looks for `//axon::inject` annotations, but there may be a logic error that causes it to miss some annotations or break early from the loop.

### Current Flow

1. `extractDependencies()` iterates through struct fields
2. For each field, it checks if there are doc comments
3. It looks for `//axon::inject` annotations in the comments
4. When found, it adds the field to the dependencies list
5. The generator uses these dependencies to create constructor parameters

### Suspected Issues

1. **Early break from comment loop**: The parser may be breaking out of the comment processing loop after finding the first annotation
2. **Field processing order**: The parser may not be processing all fields in the struct
3. **Comment parsing logic**: The annotation detection logic may have edge cases

## Components and Interfaces

### Parser Interface
```go
type Parser struct {
    // existing fields
}

func (p *Parser) extractDependencies(structType *ast.StructType) []models.Dependency
func (p *Parser) parseInjectAnnotation(comment string) (bool, bool, bool)
```

### Generator Interface
```go
type Generator struct {
    // existing fields
}

func (g *Generator) generateControllerProvider(controller models.ControllerMetadata) (string, error)
```

### Template Data Structure
```go
type CoreServiceProviderData struct {
    StructName    string
    Dependencies  []DependencyData // All dependencies (for struct initialization)
    InjectedDeps  []DependencyData // Only injected dependencies (for function parameters)
    HasStart      bool
    HasStop       bool
}
```

## Data Models

### Dependency Model
```go
type Dependency struct {
    Name   string // Field name
    Type   string // Field type
    IsInit bool   // Whether this is an init dependency
}
```

### DependencyData (Template)
```go
type DependencyData struct {
    Name      string // Parameter name (camelCase)
    FieldName string // Struct field name
    Type      string // Go type
    IsInit    bool   // Whether this is an init dependency
}
```

## Error Handling

### Current Issues
1. **Silent failures**: Missing dependencies don't cause compilation errors until runtime
2. **No validation**: The parser doesn't validate that all `//axon::inject` fields are processed
3. **Poor error messages**: When injection fails, the error messages don't point to the specific annotation

### Proposed Improvements
1. **Validation step**: Add a validation phase that ensures all annotated fields are included
2. **Debug logging**: Add detailed logging to show which fields are being processed
3. **Error reporting**: Provide clear error messages when annotation parsing fails

## Testing Strategy

### Unit Tests
1. **Parser tests**: Test `extractDependencies` with various struct configurations
2. **Generator tests**: Test constructor generation with multiple dependencies
3. **Template tests**: Test template execution with different dependency combinations

### Integration Tests
1. **End-to-end tests**: Test complete flow from annotation to generated code
2. **Real-world scenarios**: Test with actual controller structs like SessionController
3. **Edge cases**: Test with mixed annotation types and complex dependency graphs

### Test Cases to Add
1. **Multiple inject annotations**: Struct with 2+ `//axon::inject` fields
2. **Mixed annotations**: Struct with both `//axon::inject` and `//axon::init` fields
3. **Different field types**: Test with various Go types (pointers, interfaces, functions)
4. **Comment variations**: Test with different comment formats and spacing

## Implementation Plan

### Phase 1: Diagnosis
1. Add debug logging to `extractDependencies` function
2. Create unit tests that reproduce the issue
3. Identify the exact point where dependency extraction fails

### Phase 2: Fix
1. Fix the bug in the dependency extraction logic
2. Ensure all `//axon::inject` annotations are processed
3. Maintain backward compatibility with existing code

### Phase 3: Validation
1. Add validation to ensure all annotated fields are included
2. Improve error messages for debugging
3. Add comprehensive test coverage

### Phase 4: Testing
1. Test with the SessionController example
2. Verify that both SessionFactory and UserService are injected
3. Run integration tests to ensure no regressions

## Specific Fix Areas

### 1. Parser Logic (`internal/parser/parser.go`)
- Review the loop structure in `extractDependencies`
- Ensure all fields are processed, not just the first one with annotations
- Fix any early breaks or continue statements that skip fields

### 2. Comment Processing
- Verify that all comments on a field are processed
- Ensure the annotation detection logic handles multiple annotations per field
- Check for edge cases in comment parsing

### 3. Dependency Collection
- Ensure dependencies are collected from all annotated fields
- Verify that the dependency list includes all required fields
- Check that field names and types are correctly extracted

### 4. Generator Integration
- Verify that the generator receives all dependencies
- Ensure the template receives both `Dependencies` and `InjectedDeps` correctly
- Check that constructor parameters match struct fields

## Success Criteria

1. **SessionController works**: Both SessionFactory and UserService are injected
2. **All tests pass**: Unit and integration tests validate the fix
3. **No regressions**: Existing functionality continues to work
4. **Clear errors**: Better error messages when injection fails
5. **Documentation**: Updated examples and documentation reflect the fix