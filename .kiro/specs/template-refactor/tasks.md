# Implementation Plan

- [x] 1. Create import management system
  - Create `internal/templates/imports.go` with ImportManager implementation
  - Implement PackageResolver for dynamic package path detection
  - Add functions to extract imports from AST and resolve local package paths
  - _Requirements: 4.1, 4.2, 4.3, 4.4_

- [x] 2. Enhance parser to capture source imports
  - Modify parser to extract and store import statements during AST processing
  - Add import information to PackageMetadata structure
  - Implement module path detection from go.mod file
  - _Requirements: 4.1, 4.2_

- [x] 3. Create organized template definitions
  - Create `internal/templates/template_defs.go` to extract template constants
  - Move all template string constants from templates.go to the new file
  - Organize templates by functionality (providers, interfaces, modules)
  - _Requirements: 2.1, 2.2, 2.3_

- [ ] 4. Fix import detection in template generation
  - Update `GenerateCoreServiceModule` function to use ImportManager
  - Fix the context.Context import bug in interface generation
  - Implement proper import block generation with correct formatting
  - _Requirements: 1.1, 1.2, 1.3, 4.5_

- [ ] 5. Enhance template execution with import handling
  - Update `executeTemplate` function to handle import management
  - Add template helper functions for import-related operations
  - Improve error handling and validation for template execution
  - _Requirements: 3.1, 3.2, 3.3, 3.4_

- [ ] 6. Update interface template generation
  - Fix InterfaceTemplate to properly detect and include required imports
  - Ensure method signatures with external types get proper imports
  - Test with various interface method signatures including context.Context
  - _Requirements: 1.1, 4.4, 5.1, 5.2_

- [ ] 7. Update module template generation
  - Fix spacing and formatting issues in generated module files
  - Ensure proper import grouping (standard library, third-party, local)
  - Add import filtering to remove unused imports
  - _Requirements: 1.2, 1.4, 4.5, 5.3, 5.4_

- [ ] 8. Enhance error handling and validation
  - Add comprehensive error messages for template execution failures
  - Implement template data validation before execution
  - Add validation for generated Go code syntax
  - _Requirements: 3.1, 3.2, 3.3, 3.4_

- [ ] 9. Create comprehensive tests for import management
  - Write unit tests for ImportManager functionality
  - Test package path resolution with various project structures
  - Test import detection and filtering logic
  - _Requirements: 6.1, 6.2, 6.3, 6.4_

- [ ] 10. Create tests for template generation fixes
  - Write tests that verify context.Context imports are included
  - Test interface generation with various method signatures
  - Test module generation with proper import formatting
  - _Requirements: 6.1, 6.2, 6.3, 6.4_

- [ ] 11. Add integration tests for complete code generation
  - Create end-to-end tests that generate and compile complete modules
  - Test with various project structures and package layouts
  - Verify that all generated code compiles without import errors
  - _Requirements: 6.1, 6.2, 6.3, 6.4_

- [ ] 12. Update existing tests to use enhanced functionality
  - Update existing template tests to work with new import handling
  - Add import validation to existing test cases
  - Ensure all existing functionality continues to work correctly
  - _Requirements: 6.1, 6.2, 6.3, 6.4_