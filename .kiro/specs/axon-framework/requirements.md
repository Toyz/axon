# Requirements Document

## Introduction

Axon is an annotation-driven framework for building scalable, modular, and maintainable web services in Go. The framework eliminates boilerplate code associated with web server setup, dependency injection, and component lifecycle management by using Go's built-in code generation tools. Developers can focus exclusively on business logic while defining application structure declaratively through simple code comments.

## Requirements

### Requirement 1

**User Story:** As a Go developer, I want to define controllers using annotations, so that I can eliminate boilerplate code for HTTP endpoint setup and dependency injection.

#### Acceptance Criteria

1. WHEN a struct is annotated with `//fx::controller` THEN the generator SHALL create a standard FX provider for this struct
2. WHEN a controller struct uses `fx.In` THEN the generator SHALL automatically inject declared dependencies
3. WHEN a method on a `//fx::controller` annotated struct is annotated with `//fx::route <METHOD> <PATH>` (e.g., `//fx::route GET /users/{id:int}`) THEN the generator SHALL expose it as an HTTP endpoint
4. WHEN a route uses path parameters with syntax `{param:type}` THEN the generator SHALL create a wrapper to parse and convert parameters to the specified Go type
5. IF a route handler returns `(data, error)` THEN the system SHALL return data with 200 OK on success and 500 Internal Server Error on error
6. IF a route handler returns `(*Response, error)` THEN the system SHALL allow full control over the response including status code and body
7. IF a route handler returns `error` THEN the user SHALL inject echo.Context to handle the response themselves instead of using generated wrappers

### Requirement 2

**User Story:** As a Go developer, I want to define reusable middleware components using annotations, so that I can apply cross-cutting concerns to my routes without repetitive code.

#### Acceptance Criteria

1. WHEN a struct is annotated with `//fx::middleware <Name>` THEN the generator SHALL register it as a named middleware component
2. WHEN a middleware struct has a `Handle(next echo.HandlerFunc) echo.HandlerFunc` method THEN the generator SHALL create an FX provider for it
3. WHEN a route uses `-Middleware=<Name1>,<Name2>` flag THEN the generator SHALL apply the specified middlewares to that route
4. WHEN the generator processes middleware annotations THEN it SHALL build a registry of available named middlewares for validation

### Requirement 3

**User Story:** As a Go developer, I want to define core services using annotations, so that I can automate dependency injection and lifecycle management without manual wiring.

#### Acceptance Criteria

1. WHEN a struct is annotated with `//fx::core` THEN the generator SHALL create an FX provider and add it to the application's root dependency graph
2. WHEN a core service uses `-Init` flag THEN the generator SHALL require a `Start(context.Context) error` method and optionally hook up a `Stop(context.Context) error` method
3. WHEN a core service uses `-Manual` flag THEN the generator SHALL look for a public variable named `Module` instead of generating a provider
4. WHEN a core service uses `-Manual=<ModuleName>` flag THEN the generator SHALL look for a public variable with the specified name
5. WHEN a service with `-Init` flag is started THEN the system SHALL call the Start method during application startup
6. WHEN a service with `-Init` flag is stopped THEN the system SHALL call the Stop method during application shutdown if it exists

### Requirement 4

**User Story:** As a Go developer, I want to generate interfaces from my structs using annotations, so that I can facilitate mocking and testing without manually maintaining interface definitions.

#### Acceptance Criteria

1. WHEN a struct is annotated with `//fx::interface` THEN the generator SHALL create a Go interface containing all public methods of the struct
2. WHEN an interface is generated THEN the interface name SHALL be `<StructName>Interface` to avoid naming conflicts with the original struct
3. WHEN an interface is generated THEN the FX module SHALL provide both the concrete struct and a provider that casts to the interface
4. WHEN other components depend on the generated interface THEN the dependency injection SHALL work seamlessly

### Requirement 5

**User Story:** As a Go developer, I want the generator to automatically discover and analyze my annotated code, so that I don't need to manually register components or maintain configuration files.

#### Acceptance Criteria

1. WHEN the generator runs THEN it SHALL recursively scan specified directories for `.go` files
2. WHEN the generator finds annotated structs and methods THEN it SHALL parse them to build an AST and collect all `fx::` tags
3. WHEN the generator analyzes middleware THEN it SHALL build a registry of available named middlewares first
4. WHEN the generator analyzes core services THEN it SHALL determine which need generated providers and which export manual modules
5. WHEN the generator analyzes controllers THEN it SHALL cross-reference the middleware registry to validate `-Middleware=` flags
6. WHEN the generator inspects handler signatures THEN it SHALL determine parameter injection and response handling logic

### Requirement 6

**User Story:** As a Go developer, I want the generator to create organized module files, so that my application structure remains clean and maintainable.

#### Acceptance Criteria

1. WHEN the generator processes a package with annotations THEN it SHALL create an `autogen_module.go` file in that package
2. WHEN the generator creates module files THEN each SHALL contain the `AutogenModule` with FX providers for all discovered components
3. WHEN the generator finds lifecycle components THEN it SHALL generate providers that accept `fx.Lifecycle` and register OnStart/OnStop hooks
4. WHEN the generator processes top-level directories THEN it SHALL create `autogen_root_module.go` files that combine all sub-package modules
5. WHEN the generator completes THEN it SHALL create a main application entry point that wires together all autogen modules

### Requirement 7

**User Story:** As a Go developer, I want flexible CLI options for the generator, so that I can customize the generation process for different project structures and deployment scenarios.

#### Acceptance Criteria

1. WHEN I run the generator with directory paths and no `--main` option THEN it SHALL NOT generate a main.go file
2. WHEN I use the `--main` option THEN the generator SHALL create the main.go file at the specified location
3. WHEN I use the `--module` option THEN the generator SHALL use the specified package name for imports
4. WHEN the generator creates main.go THEN it SHALL automatically wire together all discovered modules using FX
5. WHEN I use the `--server` option THEN the generator SHALL create a complete web server with Echo lifecycle management


### Requirement 8

**User Story:** As a Go developer, I want support for advanced route features, so that I can handle complex HTTP scenarios without additional boilerplate.

#### Acceptance Criteria

1. WHEN a route uses `-PassContext` flag THEN the generator SHALL inject the raw echo.Context as a parameter
2. WHEN a route handler needs the echo.Context THEN the generator SHALL automatically detect its position in the function signature
3. WHEN path parameters use supported types (int, string) THEN the generator SHALL handle type conversion automatically
4. WHEN multiple middlewares are specified THEN the generator SHALL apply them in the order listed
5. WHEN route parameters use Echo's syntax (`:param`) THEN the generator SHALL support standard Echo parameter binding
6. WHEN I need to access route information THEN the system SHALL provide a public API to get all registered routes

### Requirement 9

**User Story:** As a Go developer, I want proper error handling and response management, so that my API endpoints behave consistently and predictably.

#### Acceptance Criteria

1. WHEN a handler returns an error THEN the system SHALL return appropriate HTTP status codes
2. WHEN a handler uses the Response struct THEN the system SHALL respect the specified StatusCode and Body
3. WHEN parameter parsing fails THEN the system SHALL return appropriate error responses
4. WHEN middleware processing fails THEN the system SHALL handle errors gracefully
5. WHEN the application starts or stops THEN lifecycle errors SHALL be properly propagated and handled
6. WHEN I need to register routes manually THEN the system SHALL provide helper functions for Echo setup