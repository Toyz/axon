# Implementation Plan

- [x] 1. Set up project structure and core interfaces
  - Create directory structure for internal packages (parser, generator, registry, templates)
  - Define core interfaces for AnnotationParser, CodeGenerator, and MiddlewareRegistry
  - Create data model structs for PackageMetadata, ControllerMetadata, RouteMetadata, etc.
  - _Requirements: 1.1, 5.1, 6.1_

- [x] 2. Implement annotation parsing foundation
  - Create AST walker to traverse Go source files and find annotated structs/methods
  - Implement string-based annotation extraction from comments (avoiding regex)
  - Build metadata structures from parsed annotations
  - Write unit tests for annotation parsing with valid and invalid syntax
  - _Requirements: 5.1, 5.2, 9.3_

- [x] 3. Implement controller annotation processing
  - Parse `//fx::controller` annotations on structs
  - Extract dependency information from fx.In fields
  - Parse `//fx::route` annotations with method, path, and flags
  - Validate that routes are only on controller-annotated structs
  - Write tests for controller and route annotation parsing
  - _Requirements: 1.1, 1.2, 1.3, 8.1_

- [x] 4. Implement path parameter parsing and type conversion
  - Parse path parameters with `{param:type}` syntax
  - Support int and string type conversions
  - Generate parameter binding code for route wrappers
  - Handle parameter parsing errors with appropriate HTTP responses
  - Write tests for parameter parsing and type conversion
  - _Requirements: 1.4, 8.3, 9.1_

- [x] 5. Implement route handler response generation
  - Generate wrapper functions for different handler return signatures
  - Handle `(data, error)` return type with 200 OK success and 500 error responses
  - Handle `(*Response, error)` return type with custom status codes and bodies
  - Handle `error` return type requiring echo.Context injection for custom responses
  - Write tests for all response handling patterns
  - _Requirements: 1.5, 1.6, 1.7, 9.1, 9.2_

- [x] 6. Implement middleware annotation processing
  - Parse `//fx::middleware <Name>` annotations on structs
  - Build middleware registry to track named middleware components
  - Validate middleware Handle method signature
  - Cross-reference middleware names in route `-Middleware=` flags
  - Write tests for middleware registration and validation
  - _Requirements: 2.1, 2.2, 2.4, 5.4_

- [x] 7. Implement middleware application to routes
  - Parse `-Middleware=<Name1>,<Name2>` flags on routes
  - Generate code to apply middlewares to routes in specified order
  - Validate middleware names against registry during generation
  - Write tests for middleware application and ordering
  - _Requirements: 2.3, 8.4_

- [x] 8. Implement core service annotation processing
  - Parse `//fx::core` annotations on structs
  - Handle `-Init` flag to identify lifecycle services
  - Handle `-Manual` and `-Manual=<ModuleName>` flags for manual modules
  - Generate FX providers for core services
  - Write tests for core service annotation parsing
  - _Requirements: 3.1, 3.3, 3.4_

- [x] 9. Implement lifecycle management for core services
  - Generate providers that accept fx.Lifecycle parameter
  - Register OnStart hooks for services with Start(context.Context) error methods
  - Register OnStop hooks for services with Stop(context.Context) error methods
  - Handle lifecycle errors during application startup and shutdown
  - Write tests for lifecycle hook generation and execution
  - _Requirements: 3.2, 3.5, 3.6, 9.5_

- [x] 10. Implement interface generation
  - Parse `//fx::interface` annotations on structs
  - Generate interfaces with `<StructName>Interface` naming convention
  - Extract all public methods from annotated structs
  - Generate FX providers for both concrete struct and interface casting
  - Write tests for interface generation and dependency injection
  - _Requirements: 4.1, 4.2, 4.3, 4.4_

- [x] 11. Implement code generation with templates
  - Create Go templates for FX providers, route wrappers, and module files
  - Implement template rendering with metadata structures
  - Generate `autogen_module.go` files in each package with annotations
  - Generate `autogen_root_module.go` files combining sub-package modules
  - Write tests for template rendering and generated code compilation
  - _Requirements: 6.1, 6.2, 6.4_

- [x] 12. Implement advanced route features
  - Support `-PassContext` flag to inject echo.Context into handlers
  - Detect echo.Context parameter position in handler signatures
  - Support Echo's standard `:param` syntax alongside `{param:type}`
  - Generate appropriate parameter binding for both syntaxes
  - Write tests for context injection and parameter binding variations
  - _Requirements: 8.1, 8.2, 8.5_

- [x] 13. Implement CLI interface and directory scanning
  - Create command-line interface with directory path arguments
  - Implement recursive directory scanning for `.go` files
  - Support optional `--main` flag for main.go generation location
  - Support `--module` flag for custom package name in imports
  - Write tests for CLI argument parsing and directory scanning
  - _Requirements: 5.1, 7.1, 7.2, 7.3_

- [ ] 14. Implement main.go generation
  - Generate main.go file only when `--main` flag is provided
  - Wire together all discovered autogen modules using FX
  - Import all necessary packages and create FX application
  - Handle different module combinations based on scanned directories
  - Write tests for main.go generation with various module configurations
  - _Requirements: 6.5, 7.4_

- [ ] 15. Implement comprehensive error handling
  - Create structured error types for different failure scenarios
  - Handle annotation syntax errors with file and line information
  - Validate handler signatures and provide helpful error messages
  - Handle file system errors during code generation
  - Write tests for error handling and error message quality
  - _Requirements: 9.1, 9.2, 9.3, 9.4_

- [x] 16. Create public API package for user access
  - Create `pkg/axon` package with public Response struct
  - Export RouteInfo struct for route metadata access
  - Create RouteRegistry interface for accessing all discovered routes
  - Generate route registration helpers for manual Echo setup
  - Implement route path conversion from Axon syntax to Echo syntax
  - Write tests for public API usage patterns and route conversion
  - _Requirements: 1.5, 1.6, 6.3, 8.6_

- [x] 17. Implement route registry and discovery system
  - Generate global route registry that collects all routes from modules
  - Create GetAllRoutes() function to return slice of RouteInfo
  - Include route metadata (method, path, handler name, middlewares)
  - Support filtering routes by package or controller
  - Write tests for route discovery and filtering
  - _Requirements: 6.3, 8.6_

- [x] 18. Integrate route registry into generated code
  - Update generated route wrappers to register routes with global registry
  - Generate route registration calls in autogen_module.go files
  - Ensure route metadata includes all necessary information (method, path, controller, middlewares)
  - Generate route registration with proper Axon-to-Echo path conversion
  - Write tests for generated route registration integration
  - _Requirements: 6.3, 8.6_

- [x] 19. Generate optional web server with Echo lifecycle
  - Add `--server` flag to CLI for generating complete web server
  - Generate main.go with Echo setup, route registration, and lifecycle
  - Include graceful shutdown handling and signal management
  - Support configuration through environment variables
  - Make server generation optional (nice-to-have feature)
  - Write tests for generated server functionality
  - _Requirements: 7.5, 9.6_

- [x] 20. Create integration tests and examples
  - Build complete example application using all framework features
  - Test end-to-end generation and compilation of example application
  - Verify runtime behavior of generated controllers, middleware, and services
  - Test application startup and shutdown with lifecycle services
  - Create comprehensive integration test suite
  - _Requirements: 1.1, 2.1, 3.1, 4.1, 5.1_