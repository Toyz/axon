# Axon Framework Complete Example Application

This is a comprehensive example application demonstrating all features of the Axon framework, including:

- **Controllers** with HTTP route handling
- **Middleware** for cross-cutting concerns
- **Core Services** with lifecycle management
- **Interface Generation** for dependency injection
- **Parameter Binding** and type conversion
- **Response Handling** with custom status codes
- **Route Registry** for runtime introspection

## Project Structure

```
examples/complete-app/
├── internal/
│   ├── controllers/          # HTTP controllers with route annotations
│   │   ├── user_controller.go
│   │   ├── health_controller.go
│   │   └── autogen_module.go  # Generated FX module
│   ├── services/             # Business logic services
│   │   ├── user_service.go
│   │   ├── database_service.go
│   │   └── autogen_module.go  # Generated FX module
│   ├── middleware/           # HTTP middleware components
│   │   ├── logging_middleware.go
│   │   ├── auth_middleware.go
│   │   └── autogen_module.go  # Generated FX module
│   ├── interfaces/           # Interface generation examples
│   │   ├── user_interface.go
│   │   └── autogen_module.go  # Generated FX module
│   ├── models/               # Data models
│   │   └── user.go
│   └── config/               # Configuration
│       └── config.go
├── main.go                   # Application entry point
├── go.mod                    # Go module definition
└── README.md                 # This file
```

## Features Demonstrated

### 1. Controllers (`internal/controllers/`)

Controllers handle HTTP requests and are annotated with `//axon::controller`:

```go
//axon::controller
type UserController struct {
    userService *services.UserService `fx:"in"`
}

//axon::route GET /users/{id:int} -Middleware=LoggingMiddleware
func (c *UserController) GetUser(id int) (*models.User, error) {
    return c.userService.GetUser(id)
}
```

**Features:**
- Automatic dependency injection
- Path parameter parsing with type conversion (`{id:int}`)
- Middleware application (`-Middleware=LoggingMiddleware,AuthMiddleware`)
- Multiple response types: `(data, error)`, `(*Response, error)`, `error`
- Context injection with `-PassContext` flag

### 2. Middleware (`internal/middleware/`)

Middleware components provide cross-cutting functionality:

```go
//axon::middleware LoggingMiddleware
type LoggingMiddleware struct {
    enabled bool `fx:"in"`
}

func (m *LoggingMiddleware) Handle(next echo.HandlerFunc) echo.HandlerFunc {
    return func(c echo.Context) error {
        // Logging logic here
        return next(c)
    }
}
```

**Features:**
- Named middleware registration
- Automatic FX provider generation
- Integration with route middleware chains

### 3. Core Services (`internal/services/`)

Services handle business logic and can have lifecycle management:

```go
//axon::core -Init
type UserService struct {
    config *config.Config `fx:"in"`
}

func (s *UserService) Start(ctx context.Context) error {
    // Initialization logic
    return nil
}

func (s *UserService) Stop(ctx context.Context) error {
    // Cleanup logic
    return nil
}
```

**Features:**
- Automatic FX provider generation
- Lifecycle management with `-Init` flag
- Dependency injection
- Graceful startup and shutdown

### 4. Interface Generation (`internal/interfaces/`)

Generate interfaces from structs for better testability:

```go
//axon::core
//axon::interface
type UserRepository struct {
    data map[int]*models.User `fx:"in"`
}
```

**Features:**
- Automatic interface generation (`UserRepositoryInterface`)
- FX providers for both struct and interface
- Enables easy mocking for tests

## Running the Example

### 1. Generate Code

First, generate the Axon modules:

```bash
# From the axon root directory
go build -o axon ./cmd/axon
./axon examples/complete-app/internal/...
```

This will create `autogen_module.go` files in each package.

### 2. Run Tests

Run the comprehensive test suite:

```bash
cd examples/complete-app
go test -v ./example_test.go
```

### 3. Build and Run

```bash
cd examples/complete-app
go build -o app .
./app
```

The application will start an HTTP server on port 8080 (configurable via `PORT` environment variable).

## API Endpoints

The example application exposes the following endpoints:

### Health Endpoints
- `GET /health` - Application health status
- `GET /ready` - Readiness check

### User Management
- `GET /users` - List all users
- `GET /users/{id}` - Get user by ID (with logging middleware)
- `POST /users` - Create new user (with logging + auth middleware)
- `PUT /users/{id}` - Update user (with logging + auth middleware)
- `DELETE /users/{id}` - Delete user (with logging + auth middleware)

### Authentication

For endpoints requiring authentication, include the header:
```
Authorization: Bearer valid-token
```

## Generated Code

The Axon framework generates the following for each package:

### Controllers
- Provider functions for dependency injection
- Route wrapper functions with parameter parsing
- Middleware application logic
- Route registration with Echo
- Route registry integration

### Middleware
- Provider functions for middleware instances
- Middleware registration with Axon registry
- FX module configuration

### Services
- Provider functions with lifecycle hooks
- Automatic Start/Stop method wiring
- FX module configuration

### Interfaces
- Interface definitions from struct methods
- Providers for both concrete and interface types
- FX module configuration

## Testing

The example includes comprehensive tests demonstrating:

1. **Code Generation** - Verifies all modules are generated correctly
2. **Compilation** - Ensures generated code compiles without errors
3. **Route Generation** - Tests route wrapper and middleware integration
4. **Middleware Generation** - Verifies middleware provider generation
5. **Service Generation** - Tests lifecycle service generation
6. **Runtime Behavior** - Integration tests for HTTP endpoints
7. **Parameter Binding** - Tests path parameter parsing
8. **Response Handling** - Tests different response types
9. **Middleware Chaining** - Tests middleware execution order

Run tests with:
```bash
go test -v ./example_test.go
```

## Key Concepts

### Annotation-Driven Development
All framework features are configured through code comments using the `axon::` prefix:
- `//axon::controller` - Marks HTTP controllers
- `//axon::route METHOD /path` - Defines HTTP routes
- `//axon::middleware Name` - Defines middleware components
- `//axon::core` - Marks core services
- `//axon::interface` - Generates interfaces from structs

### Dependency Injection
The framework uses Uber FX for dependency injection. Dependencies are declared using struct tags:
```go
type UserController struct {
    userService *UserService `fx:"in"`
}
```

### Lifecycle Management
Services can participate in application lifecycle with the `-Init` flag:
```go
//axon::core -Init
type DatabaseService struct {}

func (s *DatabaseService) Start(ctx context.Context) error { /* ... */ }
func (s *DatabaseService) Stop(ctx context.Context) error { /* ... */ }
```

### Route Registry
All routes are automatically registered in a global registry for runtime introspection:
```go
routes := axon.GetRoutes()
for _, route := range routes {
    fmt.Printf("%s %s -> %s\n", route.Method, route.Path, route.HandlerName)
}
```

This example demonstrates the complete capabilities of the Axon framework and serves as a reference for building annotation-driven web applications in Go.