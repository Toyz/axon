# Axon Framework

Axon is an annotation-driven web framework for Go that uses code generation to create dependency injection modules, HTTP route handlers, and middleware chains. It leverages Uber FX for dependency injection and Echo for HTTP routing.

## Quick Start

1. **Install Axon CLI**:
   ```bash
   go install github.com/toyz/axon/cmd/axon@latest
   ```

2. **Create a controller**:
   ```go
   //axon::controller
   type UserController struct {
       //axon::inject
       UserService *services.UserService
   }

   //axon::route GET /users/{id:int}
   func (c *UserController) GetUser(id int) (*User, error) {
       return c.UserService.GetUser(id)
   }
   ```

3. **Generate code**:
   ```bash
   axon ./internal/...
   ```

4. **Use the generated modules**:
   ```go
   fx.New(
       controllers.AutogenModule,
       services.AutogenModule,
   ).Run()
   ```

## Annotations Reference

### Controller Annotations

#### `//axon::controller`
Marks a struct as an HTTP controller. Generates FX providers and route registration.

```go
//axon::controller
type UserController struct {
    //axon::inject
    UserService *services.UserService
}
```

**Generated:**
- `NewUserController()` provider function
- Route wrapper functions for each handler
- `RegisterRoutes()` function for Echo integration
- FX module with all providers

#### `//axon::route METHOD /path [flags]`
Defines an HTTP route handler method.

**Syntax:**
```go
//axon::route METHOD /path [flags]
func (c *Controller) HandlerName(params...) (response, error) {}
```

**Parameters:**
- `METHOD`: HTTP method (GET, POST, PUT, DELETE, etc.)
- `/path`: URL path with optional parameters
- `flags`: Optional flags (see Route Flags section)

**Path Parameters:**
```go
//axon::route GET /users/{id:int}/posts/{slug:string}
func (c *Controller) GetUserPost(id int, slug string) (*Post, error) {}
```

**Supported parameter types:**
- `int` - Integer conversion with validation
- `string` - Direct string value
- `float64`, `float32` - Floating point conversion
- `uuid.UUID` - UUID parsing (requires custom parser)
- Custom types via `//axon::parser`

**Route Flags:**

- **`-Middleware=Name1,Name2`**: Apply middleware to route
  ```go
  //axon::route POST /users -Middleware=AuthMiddleware,LoggingMiddleware
  func (c *Controller) CreateUser(user User) error {}
  ```

- **`-PassContext`**: Inject Echo context as first parameter
  ```go
  //axon::route GET /custom -PassContext
  func (c *Controller) CustomHandler(ctx echo.Context) error {}
  ```

**Response Types:**

1. **Data + Error**: `(T, error)`
   ```go
   func (c *Controller) GetUser(id int) (*User, error) {}
   // Returns: 200 OK with JSON body, or 500/400 on error
   ```

2. **Custom Response**: `(*axon.Response, error)`
   ```go
   func (c *Controller) CreateUser(user User) (*axon.Response, error) {
       return &axon.Response{
           StatusCode: 201,
           Body:       user,
           Headers:    map[string]string{"Location": "/users/123"},
       }, nil
   }
   ```

3. **Error Only**: `error`
   ```go
   func (c *Controller) DeleteUser(id int) error {}
   // Returns: 204 No Content on success, error status on failure
   ```

### Service Annotations

#### `//axon::core [flags]`
Marks a struct as a core service. Generates FX providers with optional lifecycle management.

```go
//axon::core
type UserService struct {
    //axon::inject
    DatabaseService *DatabaseService
    //axon::init
    cache map[string]*User
}
```

**Service Flags:**

- **`-Init`**: Enable lifecycle management
  ```go
  //axon::core -Init
  type DatabaseService struct {}

  func (s *DatabaseService) Start(ctx context.Context) error {
      // Initialization logic
      return nil
  }

  func (s *DatabaseService) Stop(ctx context.Context) error {
      // Cleanup logic
      return nil
  }
  ```

- **`-Mode=Singleton`**: Singleton lifecycle (default)
  ```go
  //axon::core -Mode=Singleton
  type ConfigService struct {}
  // Injected as: *ConfigService
  ```

- **`-Mode=Transient`**: Transient lifecycle
  ```go
  //axon::core -Mode=Transient
  type SessionService struct {}
  // Injected as: func() *SessionService
  ```

- **`-Manual=ModuleName`**: Reference existing FX module
  ```go
  //axon::core -Manual=CustomModule
  type ExternalService struct {}
  // References existing CustomModule instead of generating provider
  ```

#### `//axon::logger [flags]`
Marks a struct as a logger service with special FX integration.

```go
//axon::logger
type AppLogger struct {
    //axon::inject
    Config *config.Config
    //axon::init
    logger *slog.Logger
}
```

**Logger Features:**
- Automatic `fx.WithLogger()` integration
- Immediate initialization for FX logging
- Lifecycle management support with `-Init` flag

### Dependency Injection Annotations

#### `//axon::inject`
Marks a struct field for dependency injection. The field will become a parameter in the generated provider function.

```go
type UserController struct {
    //axon::inject
    UserService *services.UserService
    //axon::inject
    DatabaseService *services.DatabaseService
}
```

**Generated provider:**
```go
func NewUserController(userService *services.UserService, databaseService *services.DatabaseService) *UserController {
    return &UserController{
        UserService: userService,
        DatabaseService: databaseService,
    }
}
```

#### `//axon::init`
Marks a struct field for initialization (not injection). The field will be initialized with generated code.

```go
type UserService struct {
    //axon::inject
    Config *config.Config
    //axon::init
    cache map[string]*User
    //axon::init
    mutex sync.RWMutex
}
```

**Generated provider:**
```go
func NewUserService(config *config.Config) *UserService {
    return &UserService{
        Config: config,
        cache:  make(map[string]*User),
        mutex:  sync.RWMutex{},
    }
}
```

### Middleware Annotations

#### `//axon::middleware Name`
Defines a named middleware component.

```go
//axon::middleware AuthMiddleware
type AuthMiddleware struct {
    //axon::inject
    Config *config.Config
}

func (m *AuthMiddleware) Handle(next echo.HandlerFunc) echo.HandlerFunc {
    return func(c echo.Context) error {
        // Authentication logic
        if !m.isAuthenticated(c) {
            return echo.NewHTTPError(http.StatusUnauthorized)
        }
        return next(c)
    }
}
```

**Generated:**
- `NewAuthMiddleware()` provider function
- Middleware registration with Axon registry
- FX module with middleware provider

**Usage in routes:**
```go
//axon::route POST /users -Middleware=AuthMiddleware
func (c *Controller) CreateUser(user User) error {}
```

### Interface Generation

#### `//axon::interface`
Generates an interface from a struct's public methods.

```go
//axon::core
//axon::interface
type UserRepository struct {
    //axon::inject
    DB *sql.DB
}

func (r *UserRepository) GetUser(id int) (*User, error) {}
func (r *UserRepository) CreateUser(user *User) error {}
```

**Generated interface:**
```go
type UserRepositoryInterface interface {
    GetUser(id int) (*User, error)
    CreateUser(user *User) error
}
```

**Generated providers:**
```go
func NewUserRepository(db *sql.DB) *UserRepository {}
func NewUserRepositoryInterface(impl *UserRepository) UserRepositoryInterface {
    return impl
}
```

### Custom Parameter Parsers

#### `//axon::parser Type`
Defines a custom parameter parser for route parameters.

```go
//axon::parser uuid.UUID
func ParseUUID(c echo.Context, value string) (uuid.UUID, error) {
    return uuid.Parse(value)
}
```

**Usage in routes:**
```go
//axon::route GET /users/{id:uuid.UUID}
func (c *Controller) GetUser(id uuid.UUID) (*User, error) {}
```

**Parser Requirements:**
- Function signature: `func(echo.Context, string) (T, error)`
- Must return the target type and an error
- Registered globally for the specified type

## Code Generation

### Generated Files

Axon generates `autogen_module.go` files in each package containing annotations:

```go
// Code generated by Axon framework. DO NOT EDIT.
package controllers

import (
    "go.uber.org/fx"
    // ... other imports
)

// Provider functions
func NewUserController(userService *services.UserService) *UserController {}

// Route wrappers
func userControllerGetUserWrapper(controller *UserController) echo.HandlerFunc {}

// Route registration
func RegisterRoutes(e *echo.Echo, userController *UserController) {}

// FX Module
var AutogenModule = fx.Module("controllers",
    fx.Provide(NewUserController),
    fx.Invoke(RegisterRoutes),
)
```

### Module Integration

Use generated modules in your application:

```go
package main

import (
    "github.com/your-app/internal/controllers"
    "github.com/your-app/internal/services"
    "github.com/your-app/internal/middleware"
    "go.uber.org/fx"
)

func main() {
    fx.New(
        // Generated modules
        controllers.AutogenModule,
        services.AutogenModule,
        middleware.AutogenModule,
        
        // Manual providers
        fx.Provide(
            config.New,
            echo.New,
        ),
        
        // Start HTTP server
        fx.Invoke(startServer),
    ).Run()
}
```

## Best Practices

### Project Structure
```
your-app/
├── internal/
│   ├── controllers/     # HTTP controllers
│   ├── services/        # Business logic
│   ├── middleware/      # HTTP middleware
│   ├── interfaces/      # Interface definitions
│   ├── parsers/         # Custom parameter parsers
│   └── config/          # Configuration
├── pkg/                 # Public packages
├── cmd/                 # Application entrypoints
└── main.go
```

### Dependency Organization
- Use `//axon::inject` for external dependencies
- Use `//axon::init` for internal state (maps, slices, etc.)
- Prefer interfaces for better testability
- Use transient services for stateful, per-request logic

### Error Handling
- Return structured errors from handlers
- Use `echo.NewHTTPError()` for HTTP-specific errors
- Implement proper error logging in middleware

### Testing
- Generate interfaces for easy mocking
- Test business logic in services separately from HTTP handlers
- Use integration tests for full request/response cycles

## Examples

See the [complete example application](./examples/complete-app/) for a comprehensive demonstration of all Axon features.

## CLI Usage

```bash
# Generate code for specific packages
axon ./internal/controllers ./internal/services

# Generate code for all packages recursively
axon ./internal/...

# Generate with custom module name
axon -module=github.com/your-org/your-app ./internal/...
```

## Contributing

1. Fork the repository
2. Create a feature branch
3. Add tests for new functionality
4. Ensure all tests pass
5. Submit a pull request

## License

MIT License - see LICENSE file for details.