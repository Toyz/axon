# Requirements Document

## Introduction

The Axon framework's dependency injection system has a critical bug where multiple `//axon::inject` annotations on struct fields are not being properly processed. Currently, only the first annotated field is included in the generated constructor, while subsequent fields are ignored. This breaks the dependency injection for controllers and services that need multiple dependencies.

## Requirements

### Requirement 1

**User Story:** As a Go developer using Axon, I want all fields marked with `//axon::inject` to be properly injected, so that my controllers and services receive all their required dependencies.

#### Acceptance Criteria

1. WHEN a struct has multiple fields annotated with `//axon::inject` THEN the generator SHALL include all annotated fields as parameters in the generated constructor
2. WHEN the generator creates a constructor function THEN it SHALL set all `//axon::inject` annotated fields in the returned struct instance
3. WHEN the generator processes dependency injection THEN it SHALL maintain the correct parameter order and types for all injected dependencies
4. WHEN a controller uses both singleton and transient services THEN both dependencies SHALL be properly injected and available

### Requirement 2

**User Story:** As a Go developer, I want the dependency injection to work consistently across different service lifecycle modes, so that I can mix singleton and transient services in the same controller.

#### Acceptance Criteria

1. WHEN a controller injects both a factory function (transient) and a regular service (singleton) THEN both SHALL be available in the controller instance
2. WHEN the generator processes mixed lifecycle dependencies THEN it SHALL correctly identify and inject both types
3. WHEN a transient service factory is injected alongside singleton services THEN the factory SHALL create new instances while singletons remain shared
4. WHEN multiple controllers use the same dependency pattern THEN the injection SHALL work consistently across all controllers

### Requirement 3

**User Story:** As a Go developer, I want clear error messages when dependency injection fails, so that I can quickly identify and fix configuration issues.

#### Acceptance Criteria

1. WHEN the generator encounters invalid `//axon::inject` annotations THEN it SHALL provide clear error messages indicating the issue
2. WHEN a dependency cannot be resolved THEN the generator SHALL report which field and type are missing
3. WHEN there are circular dependencies THEN the generator SHALL detect and report them with helpful context
4. WHEN the generated code fails to compile due to injection issues THEN the error messages SHALL point to the specific annotation or field causing the problem

### Requirement 4

**User Story:** As a Go developer, I want the dependency injection to support complex dependency graphs, so that I can build sophisticated applications with proper separation of concerns.

#### Acceptance Criteria

1. WHEN services depend on other services THEN the generator SHALL resolve the dependency chain correctly
2. WHEN a service needs multiple dependencies of different types THEN all SHALL be properly injected
3. WHEN dependencies have their own dependencies THEN the FX module SHALL wire the complete dependency graph
4. WHEN the application starts THEN all dependencies SHALL be available and properly initialized in the correct order