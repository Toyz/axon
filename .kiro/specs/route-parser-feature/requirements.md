# Requirements Document

## Introduction

The `axon::route_parser` feature extends the Axon framework's routing system to support custom parameter parsing and validation. This feature allows developers to define reusable parsers for complex parameter types (like UUIDs, composite IDs, date ranges) that automatically validate and convert route parameters before they reach controller methods.

## Requirements

### Requirement 1

**User Story:** As a developer, I want to define custom parameter parsers for route parameters, so that I can validate and convert complex types automatically without manual parsing in each controller method.

#### Acceptance Criteria

1. WHEN I annotate a function with `//axon::route_parser <TypeName>` THEN the generator SHALL register this function as a parser for the specified type
2. WHEN the parser function has signature `func(c echo.Context, paramValue string) (T, error)` THEN the generator SHALL accept it as valid
3. WHEN the parser function has an invalid signature THEN the generator SHALL return a clear error message
4. WHEN I use `{param:TypeName}` in a route path THEN the generator SHALL use the registered parser for that parameter
5. WHEN no parser is registered for a type THEN the generator SHALL return an error during code generation

### Requirement 2

**User Story:** As a developer, I want automatic error handling for parser failures, so that invalid parameters return appropriate HTTP error responses without manual error handling.

#### Acceptance Criteria

1. WHEN a parser function returns an error THEN the generated route wrapper SHALL return HTTP 400 Bad Request
2. WHEN a parser function returns an error THEN the error message SHALL be included in the HTTP response
3. WHEN a parser succeeds THEN the parsed value SHALL be passed to the controller method
4. WHEN multiple parameters use custom parsers THEN each SHALL be validated independently
5. WHEN any parser fails THEN the request SHALL be rejected before calling the controller method

### Requirement 3

**User Story:** As a developer, I want built-in support for common types like UUID, so that I don't need to implement basic parsers myself.

#### Acceptance Criteria

1. WHEN I use `{id:uuid.UUID}` in a route THEN the generator SHALL use a built-in UUID parser
2. WHEN an invalid UUID is provided THEN the system SHALL return HTTP 400 with a clear error message
3. WHEN a valid UUID is provided THEN it SHALL be converted to `uuid.UUID` type
4. WHEN I import the UUID package THEN the generator SHALL detect and use the appropriate parser
5. WHEN the UUID package is not imported THEN the generator SHALL provide a helpful error message

### Requirement 4

**User Story:** As a developer, I want to create composite parsers that validate multiple parts of a parameter, so that I can handle complex ID formats and business rules.

#### Acceptance Criteria

1. WHEN I create a parser for composite types THEN it SHALL validate all components
2. WHEN any component of a composite type is invalid THEN the parser SHALL return a descriptive error
3. WHEN all components are valid THEN the parser SHALL return a structured object
4. WHEN I use composite types in routes THEN the controller SHALL receive the fully parsed object
5. WHEN composite parsing fails THEN the error message SHALL indicate which component failed

### Requirement 5

**User Story:** As a developer, I want parser discovery across packages, so that I can organize parsers in a central location and use them throughout my application.

#### Acceptance Criteria

1. WHEN parsers are defined in any scanned package THEN the generator SHALL discover them
2. WHEN multiple parsers are defined for the same type THEN the generator SHALL return an error
3. WHEN parsers are defined in imported packages THEN they SHALL be available for use
4. WHEN I reference a parser type THEN the generator SHALL include the necessary imports
5. WHEN parser packages are not imported THEN the generator SHALL provide import suggestions

### Requirement 6

**User Story:** As a developer, I want generated code to be efficient and readable, so that the parsing overhead is minimal and debugging is straightforward.

#### Acceptance Criteria

1. WHEN route wrappers are generated THEN they SHALL call parsers directly without reflection
2. WHEN multiple parameters need parsing THEN they SHALL be processed in parameter order
3. WHEN parsing succeeds THEN there SHALL be no unnecessary allocations or conversions
4. WHEN I examine generated code THEN it SHALL be readable and well-commented
5. WHEN errors occur THEN stack traces SHALL point to the relevant parser function

### Requirement 7

**User Story:** As a developer, I want comprehensive error messages during code generation, so that I can quickly identify and fix parser-related issues.

#### Acceptance Criteria

1. WHEN a parser function has wrong signature THEN the error SHALL specify the expected signature
2. WHEN a parser type is not found THEN the error SHALL list available parsers
3. WHEN parser imports are missing THEN the error SHALL suggest the required imports
4. WHEN parser conflicts exist THEN the error SHALL show all conflicting definitions
5. WHEN parser registration fails THEN the error SHALL include file and line information

### Requirement 8

**User Story:** As a developer, I want parser integration with existing route features, so that custom parsers work seamlessly with middleware, context injection, and response handling.

#### Acceptance Criteria

1. WHEN routes use both custom parsers and middleware THEN both SHALL work together correctly
2. WHEN routes use both custom parsers and `-PassContext` THEN the context SHALL be available to parsers
3. WHEN parser errors occur THEN middleware SHALL still be able to handle the error response
4. WHEN parsers succeed THEN the parsed values SHALL integrate with existing parameter binding
5. WHEN routes have mixed parameter types THEN built-in and custom parsers SHALL work together