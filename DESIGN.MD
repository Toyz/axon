# Technical Specification: "Axon" Framework

## 1. Vision & Goals
Axon is an annotation-driven framework for building scalable, modular, and maintainable web services in Go. Its primary goal is to eliminate boilerplate code associated with web server setup, dependency injection, and component lifecycle management.
By using Go's built-in code generation tools, Axon allows developers to focus exclusively on business logic, defining the application's structure declaratively through simple code comments.

**Core Technologies:**
- Go: The primary programming language.
- Uber FX: For dependency injection and application lifecycle management.
- Echo: As the underlying high-performance HTTP web framework.
- Go AST (go/ast, go/parser): For parsing source code to read annotations and code structure.

## 2. Annotation Specification
Annotations are specially formatted Go comments that the generator tool parses.

### 2.1. Controller Annotations

**`//fx::controller`**
- **Purpose**: Marks a struct as a controller, making it discoverable by the generator.
- **Placement**: Must be placed directly above a struct definition.
- **Behavior**: The generator will create a standard FX provider for this struct. The struct should use fx.In to declare its dependencies, which will be automatically injected by FX. The root *echo.Echo instance can also be injected as a dependency.

**Example:**
```go
//fx::controller
type UserController struct {
   fx.In
   Logger *zap.Logger
   Echo   *echo.Echo // Optional injection of the root Echo instance
}
```

**`//fx::route <METHOD> <PATH> [FLAGS]`**
- **Purpose**: Exposes a controller method as an HTTP endpoint.
- **Placement**: Must be placed directly above a method of a struct marked with `//fx::controller`.
- **Syntax**:
  - `<METHOD>`: A standard HTTP method (e.g., GET, POST, PUT, DELETE).
  - `<PATH>`: The URL path. Supports path parameters using Echo's syntax (`:param`) or an extended syntax for type coercion (`{param:type}`).
  - `[FLAGS]`: Optional space-separated flags to modify behavior.
- **Path Parameter Syntax**:
  - `{name:type}`: The generator will create a wrapper to parse the parameter and convert it to the specified Go type.
  - **Supported Types**: int, string, uuid.UUID, and other custom types defined in a generator registry.
- **Handler Signatures & Return Values**:
  1. `func(params...) (data, error)`: On success, data is returned with a 200 OK status. error results in a 500 Internal Server Error.
  2. `func(params...) (*Response, error)`: Allows full control over the response. The Response struct is defined as:
     ```go
     type Response struct {
        StatusCode int
        Body       any
     }
     ```
  3. `func(params...) error`: Returns a 204 No Content on success.
- **Flags**:
  - `-PassContext`: Allows the raw echo.Context to be injected as a parameter into the handler method. The generator will automatically detect its position in the function signature.
  - `-Middleware=<Name1>,<Name2>`: Applies a comma-separated list of named middlewares to the route.

**Example:**
```go
//fx::route GET /users/{id:int}
func (h *UserController) GetUser(id int) (*User, error) { /* ... */ }

//fx::route POST /users -PassContext
func (h *UserController) CreateUser(body CreateUserRequest, c echo.Context) error { /* ... */ }
```

### 2.2. Middleware Annotation

**`//fx::middleware <Name>`**
- **Purpose**: Marks a struct as a reusable middleware component and registers it with a given name.
- **Placement**: Must be placed directly above a struct definition.
- **Convention**: The struct must have a public method named Handle with the signature `func(next echo.HandlerFunc) echo.HandlerFunc`.
- **Behavior**: The generator creates an FX provider that registers the middleware instance as a named component, making it available to the `-Middleware=` flag.

**Example:**
```go
//fx::middleware Auth
type AuthMiddleware struct { /* ... */ }

func (m *AuthMiddleware) Handle(next echo.HandlerFunc) echo.HandlerFunc { /* ... */ }
```

### 2.3. Core Service Annotation

**`//fx::core [FLAGS]`**
- **Purpose**: Marks a component as a core service, automating its provision to the application's root dependency graph.
- **Placement**: Can be placed above a struct definition or in a package with a manually defined module.
- **Flags**:
  - `-Init`: Signals that the component has a lifecycle. The generator will require a `Start(context.Context) error` method and will optionally hook up a `Stop(context.Context) error` method if found.
  - `-Manual`: Tells the generator to not generate a provider for this package. Instead, it should find and include a public variable named Module.
  - `-Manual=<ModuleName>`: Same as `-Manual`, but looks for a public variable with the specified name (e.g., MyModule).

**Example:**
```go
//fx::core
type Config struct { /* ... */ }

//fx::core -Init
type MessageConsumer struct { /* ... */ }

// In a package with a manual module:
//fx::core -Manual
package database
// ... manual provider and 'var Module = fx.Module(...)'
```

#### Lifecycle Management with `-Init`

When using the `-Init` flag, your struct must implement specific lifecycle methods that will be automatically bound to the FX application lifecycle:

**Required Method:**
- `Start(context.Context) error`: Called when the application starts. This is where you should initialize resources, start background processes, connect to databases, etc.

**Optional Method:**
- `Stop(context.Context) error`: Called when the application shuts down. This is where you should clean up resources, close connections, stop background processes, etc.

**Complete Lifecycle Example:**
```go
//fx::core -Init
type DatabaseService struct {
    fx.In
    Config *Config
    Logger *zap.Logger
    
    db *sql.DB
}

// Required: Start method for initialization
func (s *DatabaseService) Start(ctx context.Context) error {
    s.Logger.Info("Starting database service")
    
    db, err := sql.Open("postgres", s.Config.DatabaseURL)
    if err != nil {
        return fmt.Errorf("failed to open database: %w", err)
    }
    
    if err := db.PingContext(ctx); err != nil {
        return fmt.Errorf("failed to ping database: %w", err)
    }
    
    s.db = db
    s.Logger.Info("Database service started successfully")
    return nil
}

// Optional: Stop method for cleanup
func (s *DatabaseService) Stop(ctx context.Context) error {
    s.Logger.Info("Stopping database service")
    
    if s.db != nil {
        if err := s.db.Close(); err != nil {
            s.Logger.Error("Error closing database", zap.Error(err))
            return err
        }
    }
    
    s.Logger.Info("Database service stopped")
    return nil
}

// Regular service methods
func (s *DatabaseService) GetConnection() *sql.DB {
    return s.db
}
```

**Background Process Example:**
```go
//fx::core -Init
type MetricsCollector struct {
    fx.In
    Logger *zap.Logger
    
    ticker *time.Ticker
    done   chan struct{}
}

func (m *MetricsCollector) Start(ctx context.Context) error {
    m.Logger.Info("Starting metrics collector")
    
    m.ticker = time.NewTicker(30 * time.Second)
    m.done = make(chan struct{})
    
    go m.collectMetrics()
    
    m.Logger.Info("Metrics collector started")
    return nil
}

func (m *MetricsCollector) Stop(ctx context.Context) error {
    m.Logger.Info("Stopping metrics collector")
    
    if m.ticker != nil {
        m.ticker.Stop()
    }
    
    if m.done != nil {
        close(m.done)
    }
    
    m.Logger.Info("Metrics collector stopped")
    return nil
}

func (m *MetricsCollector) collectMetrics() {
    for {
        select {
        case <-m.ticker.C:
            // Collect and send metrics
            m.Logger.Debug("Collecting metrics")
        case <-m.done:
            return
        }
    }
}
```

**Generated Provider (what the generator creates):**
```go
// The generator creates a provider that hooks into FX lifecycle
func NewDatabaseService(config *Config, logger *zap.Logger, lc fx.Lifecycle) *DatabaseService {
    svc := &DatabaseService{
        Config: config,
        Logger: logger,
    }
    
    lc.Append(fx.Hook{
        OnStart: svc.Start,
        OnStop:  svc.Stop, // Only if Stop method exists
    })
    
    return svc
}
```

### 2.4. Interface Generation Annotation

**`//fx::interface`**
- **Purpose**: Instructs the generator to create a Go interface based on the public methods of the annotated struct. This is primarily used to facilitate mocking and testing.
- **Placement**: Can be placed directly above a struct that is also marked with `//fx::core` or `//fx::controller`.
- **Behavior**:
  - The generator will inspect the struct and create an interface containing all of its public methods.
  - The generated interface name will be `I<StructName>` by default (e.g., IUserService).
  - The generated fx.Module will be updated to provide both the concrete struct and a provider that casts the concrete type to the generated interface. This allows other components to depend on the interface directly.

**Example:**

**User-Written Code:**
```go
//fx::core
//fx::interface
type UserService struct { /* ... */ }

func (s *UserService) GetUser(id int) (*User, error) { /* ... */ }
func (s *UserService) CreateUser(data CreateUserData) (*User, error) { /* ... */ }
```

**Generated Code (autogen_module.go):**
```go
// The generator creates the interface from the public methods.
type IUserService interface {
   GetUser(id int) (*User, error)
   CreateUser(data CreateUserData) (*User, error)
}

// The generated module provides both the concrete type and the interface.
var AutogenModule = fx.Module("users_autogen",
   fx.Provide(
       // 1. Provides the concrete implementation
       NewUserService,
       // 2. Provides the cast to the interface
       func(s *UserService) IUserService { return s },
   ),
)
```

**Controller Depending on the Interface:**
```go
//fx::controller
type UserController struct {
   fx.In
   // This now works seamlessly
   UserService IUserService
}
```

## 3. Generator ("Compiler") Logic

The generator is a command-line tool run via `go generate`. It performs a multi-stage process.

1. **Discovery:**
   - Recursively scans specified directories (e.g., `/controllers`, `/middleware`, `/core`).
   - Parses all `.go` files to build an AST.
   - Collects all structs and methods annotated with `fx::` tags.

2. **Analysis & Model Building:**
   - **Middleware**: First, it analyzes all `//fx::middleware` annotations to build a registry of available named middlewares.
   - **Core Services**: It analyzes `//fx::core` annotations to determine which services need generated providers (with or without lifecycle hooks) and which packages export manual modules. It also checks for `//fx::interface` to determine if interface providers are needed.
   - **Controllers**: It analyzes `//fx::controller` and `//fx::route` annotations, cross-referencing the middleware registry to validate `-Middleware=` flags. It inspects handler signatures to determine parameter injection and response handling logic.

3. **Code Generation:**
   - For each package containing annotations, the generator creates an `autogen_module.go` file.
   - This file contains the `AutogenModule`, which includes:
     - FX providers for all discovered components (including those with `-Init` lifecycle hooks).
     - Generated interfaces if `//fx::interface` is present.
     - For routes, it generates wrapper functions that handle parameter binding, type conversion, and response marshalling before calling the developer's actual handler logic.
     - For lifecycle components (`-Init`), it generates providers that accept `fx.Lifecycle` and register the OnStart/OnStop hooks.
   - It generates a single `autogen_root_module.go` in each top-level directory (e.g., `/controllers`, `/core`) that combines all discovered sub-package modules (both generated and manual) into a single, convenient module (e.g., `AllControllersModule`).
   - Finally, it generates the main application entry point at the location specified by the CLI `--main` option (defaults to `cmd/server/main.go`) that automatically wires together all the autogen modules.

## 4. Project Structure & Conventions

A recommended project layout is as follows:

```
/app
├── cmd/server/main.go      // Generated application entry point (configurable location)
│
├── internal/
│   ├── controllers/
│   │   ├── admin/
│   │   │   ├── user_controller.go
│   │   │   └── autogen_module.go    // Generated
│   │   ├── autogen_root_module.go   // Generated
│   │   ├── product_controller.go
│   │   └── autogen_module.go        // Generated
│   │
│   ├── core/
│   │   ├── config/
│   │   │   ├── config.go
│   │   │   └── autogen_module.go    // Generated
│   │   ├── database/
│   │   │   ├── database.go
│   │   │   └── autogen_module.go    // Generated
│   │   └── autogen_root_module.go   // Generated
│   │
│   └── middleware/
│       ├── auth.go
│       ├── autogen_module.go        // Generated
│       └── autogen_root_module.go   // Generated
│
└── go.mod
```

### Generator CLI Usage

The generator accepts several CLI options to customize the generation process:

```bash
# Basic usage (uses default main.go location: cmd/server/main.go)
axon generate ./internal/controllers ./internal/core ./internal/middleware

# Custom main.go location
axon generate --main cmd/api/main.go ./internal/controllers ./internal/core ./internal/middleware

# Multiple entry points for different applications
axon generate --main cmd/web/main.go ./internal/controllers ./internal/core
axon generate --main cmd/worker/main.go ./internal/core ./internal/jobs

# Specify custom package name for imports
axon generate --main cmd/app/main.go --module myapp ./internal/controllers ./internal/core
```

The generator creates the `main.go` file automatically at the specified location, wiring together all discovered modules.

**Generated main.go (example for `--main cmd/api/main.go`):**
```go
package main

import (
    "go.uber.org/fx"
    
    "your-app/internal/core"
    "your-app/internal/middleware"
    "your-app/internal/controllers"
)

func main() {
    app := fx.New(
        core.AllCoreModule,        // Includes all core services with lifecycle hooks
        middleware.AllMiddlewareModule,
        controllers.AllControllersModule,
    )
    app.Run()
}
```

**Example Generated Core Module (including lifecycle services):**
```go
// internal/core/autogen_root_module.go
package core

import "go.uber.org/fx"

var AllCoreModule = fx.Module("all_core",
    // Include all package-level autogen modules
    config.AutogenModule,      // Contains Config provider
    database.AutogenModule,    // Contains DatabaseService with lifecycle hooks
    messaging.AutogenModule,   // Contains MessageConsumer with lifecycle hooks
    
    // Include any manual modules
    cache.Module,              // Manual module if using -Manual flag
)
```

## 5. Future Work & Potential Enhancements

### `//fx::RouteParser <TypeName>`

- **Purpose**: To create a discoverable, reusable parser for a custom type used in route parameters. This extends the framework beyond the built-in int, string, etc., parsers.
- **Placement**: Must be placed directly above a func definition.
- **Convention**: The annotated function must have the signature `func(c echo.Context, paramValue string) (T, error)`, where T is the TypeName specified in the annotation.
- **Behavior**: The generator will discover this function and add it to an internal registry. When it encounters a route parameter of this type (e.g., `{id:CompositeID}`), it will generate a call to this registered parser function instead of attempting a simple type conversion.

**Example:**
```go
// In a central 'parsers' package
import "strings"

type CompositeID struct {
   Prefix string
   UUID   string
}

//fx::RouteParser CompositeID
func ParseCompositeID(c echo.Context, paramValue string) (CompositeID, error) {
   parts := strings.Split(paramValue, "_")
   if len(parts) != 2 {
       return CompositeID{}, errors.New("invalid composite ID format")
   }
   return CompositeID{Prefix: parts[0], UUID: parts[1]}, nil
}

// In a controller, for a URL like /items/user_123e4567-e89b-12d3-a456-426614174000
//fx::route GET /items/{id:CompositeID}
func (h *ItemController) GetItem(id CompositeID) (*Item, error) {
   // The 'id' object is already parsed and ready to use.
   // id.Prefix == "user"
   // id.UUID == "123e4567-e89b-12d3-a456-426614174000"
}
```

### Custom Generation Templates

- **Purpose**: To provide a "power-user" escape hatch for overriding the default code generation logic for a specific component.
- **Annotation Flag**: `-Template=<path/to/template.go.tmpl>`
- **Applicable Annotations**: `//fx::core`, `//fx::controller`, etc.
- **Behavior**: When the generator sees this flag, it will use the specified Go template file to generate the provider for that component instead of its default, built-in template. The generator will still parse the component and pass the same structured data to the custom template. This allows for handling highly specialized edge cases without modifying the generator tool itself.

**Example:**
```go
//fx::core -Init -Template=./custom_consumer.tmpl
type SpecialMessageConsumer struct {
   // ...
}
```

### Automated Documentation Generation

- **Purpose**: To generate modern, interactive API documentation that is always in sync with the source code.
- **Tooling**: A separate generator (vp-gen) would parse the Go source code and output Markdown files compatible with a modern static site generator like VitePress.
- **Behavior**: The generator would:
  1. Scan for `//fx::route` annotations.
  2. Extract the HTTP method, path, parameters, and middleware requirements.
  3. Use the GoDoc comment block above the handler function as the endpoint's description.
  4. Inspect the handler's parameter and return types to generate request/response schemas.
  5. Output `.md` files that can be committed to the repository and deployed as a static documentation site.

**`//vp::page [FLAGS]`**
- **Purpose**: Provides page-level metadata for all routes within a single Go file, similar to frontmatter in Markdown.
- **Placement**: At the top of a controller file, before any `//fx::controller` or `//fx::route` annotations.
- **Flags**:
  - `-Title="Page Title"`: Sets the title of the generated documentation page.
  - `-Group="Sidebar Group"`: Groups all routes from this file under a specific heading in the documentation sidebar.
  - `-Order=1`: Defines the sort order of this page within its group.

**Example:**
```go
//vp::page -Title="User Management" -Group="Admin APIs" -Order=1
package admin

// ...

// Fetches a user by their unique identifier.
// This endpoint requires authentication.
//fx::route GET /users/{id:int} -Middleware=Auth
func (h *UserController) GetUser(id int) (*User, error) { /* ... */ }
```

This would generate a Markdown page titled "User Management" and place it in the "Admin APIs" section of the VitePress sidebar.
