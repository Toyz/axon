# Design Document

## Overview

This design addresses the critical issues in the current templates.go file by restructuring the template system to be more maintainable, fixing import detection bugs, and improving code generation quality. The main problems being solved are:

1. Missing imports in generated code (especially `context` package)
2. Poor template organization with large hardcoded string constants
3. Inadequate import detection and management
4. Formatting issues in generated code

The solution involves refactoring the template system into a more modular architecture with proper import handling by leveraging existing imports from source files.

## Architecture

### Current Architecture Issues

The current system has several architectural problems:
- All templates are defined as large string constants in a single file
- Import detection is manual and error-prone
- No reuse of existing imports from source files
- Template execution lacks proper error handling and validation

### New Architecture

The refactor will work within the existing structure but improve organization:

```
internal/templates/
├── templates.go        # Main template functions (existing, refactored)
├── imports.go          # NEW: Import detection and management
├── template_defs.go    # NEW: Template definitions (extracted from constants)
├── response.go         # Existing route wrapper generation
├── templates_test.go   # Existing tests (enhanced)
└── response_test.go    # Existing tests
```

This approach maintains backward compatibility while improving the internal structure.

## Components and Interfaces

### Enhanced Template System

Direct improvements to the existing system:

```go
// Enhanced template execution with import management
func executeTemplate(name, templateStr string, data interface{}) (string, error) {
    // Enhanced with proper import handling and better error reporting
}

// Template definitions moved to separate file for better organization
var TemplateDefinitions = map[string]string{
    "provider-basic":     ProviderTemplate,
    "provider-lifecycle": FXLifecycleProviderTemplate,
    "interface":          InterfaceTemplate,
    // ... other templates
}
```

This directly improves the existing functions without worrying about API compatibility.

### Import Manager

Building on the existing import validation system in `parser.go`:

```go
type ImportManager struct {
    sourceImports   map[string][]Import  // imports from source files
    knownTypes      map[string]string    // type -> import path mapping
    packageResolver *PackageResolver     // for dynamic package path resolution
}

type Import struct {
    Path  string
    Alias string
}

type ImportManager interface {
    AddSourceImports(packagePath string, imports []Import)
    GetRequiredImports(generatedCode string) []Import
    GenerateImportBlock(requiredImports []Import) string
    FilterUnusedImports(imports []Import, generatedCode string) []Import
    ResolveLocalPackage(packageName string) (string, error) // NEW: dynamic local package resolution
}
```

This extends the existing `ValidateParserImports` functionality and adds dynamic package resolution that works with any project structure.

### Template Data Enhancement

Extend existing `PackageMetadata` to include import information:

```go
// Add to existing models.PackageMetadata
type PackageMetadata struct {
    // ... existing fields
    SourceImports   map[string][]Import  // NEW: imports from each source file
    ModulePath      string               // NEW: go module path from go.mod
    ModuleRoot      string               // NEW: filesystem path to module root
    PackageImportPath string             // NEW: full import path for this package
}

// Template context for enhanced generation
type TemplateContext struct {
    Data            interface{}
    SourceImports   []Import
    RequiredImports []Import
    PackageResolver *PackageResolver
}
```

This approach extends the existing data model and adds flexible package resolution that works with any project structure.

## Data Models

### Import Detection Strategy

Instead of trying to parse and detect all possible imports, we'll:

1. **Capture Source Imports**: During the parsing phase, extract all import statements from source Go files
2. **Dynamic Package Path Resolution**: Detect the actual module path and package structure from go.mod and folder structure
3. **Flexible Local Package Detection**: Generate import paths based on actual folder structure relative to module root, not hardcoded "internal" assumptions
4. **Reuse Existing Imports**: Include relevant imports from source files in generated code
5. **Add Framework Imports**: Add known framework imports (fx, context, etc.) as needed
6. **Filter Unused**: Remove imports that aren't actually referenced in the generated code

### Package Path Resolution

```go
type PackageResolver struct {
    ModuleRoot   string            // Root directory of the Go module
    ModulePath   string            // Module path from go.mod
    PackageMap   map[string]string // package name -> full import path
}

func (r *PackageResolver) ResolvePackagePath(packageDir string) string {
    // Convert filesystem path to import path
    // e.g., "./services" -> "github.com/user/project/services"
    // Works with any folder structure, not just "internal"
}
```

### Template Organization

Templates will be organized by functionality:

**Provider Templates** (`providers.go`):
- `provider-basic.tmpl` - Basic provider without lifecycle
- `provider-lifecycle.tmpl` - Provider with lifecycle hooks
- `provider-logger.tmpl` - Logger-specific provider
- `provider-transient.tmpl` - Transient service factory

**Interface Templates** (`interfaces.go`):
- `interface-definition.tmpl` - Interface type definition
- `interface-provider.tmpl` - Interface provider function

**Module Templates** (`modules.go`):
- `module-header.tmpl` - Package declaration and imports
- `module-body.tmpl` - Module definition with providers
- `fx-logger-adapter.tmpl` - FX logger adapter

## Error Handling

### Template Validation

```go
type TemplateValidator struct{}

func (v *TemplateValidator) ValidateData(templateName string, data interface{}) error {
    // Validate required fields based on template requirements
}

func (v *TemplateValidator) ValidateOutput(code string) error {
    // Basic syntax validation for generated Go code
}
```

### Error Recovery

- Provide detailed error messages with template name and data context
- Validate template data before execution
- Graceful handling of missing or invalid template data
- Clear error reporting for debugging

## Testing Strategy

### Unit Tests

1. **Template Execution Tests**: Test each template with various data inputs
2. **Import Detection Tests**: Verify import detection works for all Go type patterns
3. **Import Filtering Tests**: Ensure unused imports are properly removed
4. **Integration Tests**: Test complete code generation pipeline

### Test Data

Create comprehensive test fixtures:
- Sample PackageMetadata with various service types
- Mock source files with different import patterns
- Expected output files for comparison

### Validation Tests

- Generated code compiles without errors
- All required imports are present
- No unused imports remain
- Proper Go formatting is maintained

## Implementation Phases

### Phase 1: Import Management
- Create `imports.go` with ImportManager implementation
- Extend parser to capture source imports during AST processing
- Add import detection and filtering logic

### Phase 2: Template Organization  
- Create `template_defs.go` to extract template constants from main file
- Enhance `executeTemplate` function with import handling
- Improve template helper functions

### Phase 3: Integration
- Update `GenerateCoreServiceModule` and related functions to use ImportManager
- Fix the specific context.Context import bug
- Add proper import block generation

### Phase 4: Testing & Validation
- Enhance existing test suite with import validation
- Add tests for new ImportManager functionality
- Validate that generated code compiles correctly

## Migration Strategy

Since we're fixing bugs rather than changing functionality:
1. Direct refactor of existing functions to use improved import handling
2. Replace hardcoded template constants with better organized definitions
3. Fix the import detection issues immediately
4. Maintain the same public API signatures but improve internal implementation

This approach focuses on fixing the core issues without worrying about gradual migration.