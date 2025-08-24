# Implementation Plan

- [x] 1. Create diagnostic test to reproduce the dependency injection bug
  - Write a unit test that creates a struct with multiple `//axon::inject` annotations
  - Test the `extractDependencies` function directly to see which dependencies are found
  - Verify that the test fails with the current implementation, showing only one dependency is extracted
  - _Requirements: 1.1, 1.2_

- [x] 2. Add debug logging to dependency extraction process
  - Add detailed logging to the `extractDependencies` function in `internal/parser/parser.go`
  - Log each field being processed and each annotation found
  - Log the final list of dependencies extracted
  - _Requirements: 3.2, 3.3_

- [x] 3. Fix the dependency extraction logic in the parser
  - Analyze the `extractDependencies` function to identify why multiple annotations are not processed
  - Fix any early breaks or logic errors that prevent all fields from being processed
  - Ensure all `//axon::inject` annotated fields are included in the dependencies list
  - _Requirements: 1.1, 1.2, 1.3_

- [-] 4. Verify the generator correctly processes multiple dependencies
  - Check that `generateControllerProvider` in `internal/generator/generator.go` handles multiple dependencies
  - Ensure both `Dependencies` and `InjectedDeps` arrays are populated correctly
  - Verify the template receives all required data for constructor generation
  - _Requirements: 1.3, 2.1_

- [ ] 5. Test the fix with SessionController example
  - Regenerate the SessionController constructor using the fixed parser
  - Verify that both `SessionFactory` and `UserService` parameters are included
  - Test that the generated constructor sets both struct fields correctly
  - _Requirements: 1.4, 2.1, 2.2_

- [ ] 6. Add comprehensive unit tests for multiple dependency scenarios
  - Test structs with 2, 3, and more `//axon::inject` annotations
  - Test mixed annotations (`//axon::inject` and `//axon::init`)
  - Test different field types (pointers, interfaces, functions)
  - Test edge cases like empty structs and structs with no annotations
  - _Requirements: 1.1, 1.2, 2.3, 4.2_

- [ ] 7. Add validation to ensure all annotated fields are processed
  - Create a validation function that checks if all `//axon::inject` fields are in the dependencies list
  - Add error reporting when annotated fields are missing from the dependencies
  - Provide clear error messages indicating which fields were not processed
  - _Requirements: 3.1, 3.2, 3.4_

- [ ] 8. Create integration test for end-to-end dependency injection
  - Write a test that goes from annotated struct to generated constructor code
  - Test the complete flow: parsing → generation → compilation
  - Verify that the generated code compiles and works correctly
  - Test with real-world controller examples like SessionController
  - _Requirements: 2.4, 4.1, 4.2_

- [ ] 9. Update error handling and reporting
  - Improve error messages when dependency extraction fails
  - Add context about which file and struct are being processed
  - Provide suggestions for fixing common annotation issues
  - _Requirements: 3.1, 3.4_

- [ ] 10. Run regression tests and verify no existing functionality is broken
  - Run all existing parser and generator tests
  - Test with existing examples to ensure they still work
  - Verify that single-dependency injection still works correctly
  - Test edge cases and complex dependency scenarios
  - _Requirements: 4.3, 4.4_