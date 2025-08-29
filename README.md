# Axon Framework

**Build powerful Go web APIs with annotations and zero boilerplate**

Axon is a modern, annotation-driven web framework for Go that eliminates boilerplate through intelligent code generation. It seamlessly integrates **Uber FX** for dependency injection and **Echo** for high-performance HTTP routing, letting you focus on business logic instead of wiring.

## Why Axon?

- **Annotation-Driven**: Define controllers, routes, and middleware with simple comments
- **Zero Boilerplate**: Automatic code generation for DI, routing, and middleware chains  
- **High Performance**: Built on Echo with optimized route registration and middleware application
- **Type-Safe**: Full type safety for parameters, responses, and dependency injection
- **Modular**: Clean separation with auto-generated FX modules
- **Developer-Friendly**: Hot reload support and comprehensive error reporting

## Quick Start

### 1. Install Axon CLI
```bash
go install github.com/toyz/axon/cmd/axon@latest
```

### 2. Create Your First Controller
```go
//axon::controller -Prefix=/api/v1/users -Middleware=AuthMiddleware -Priority=10
type UserController struct {
    //axon::inject
    UserService *services.UserService
}

//axon::route GET /search -Priority=10
func (c *UserController) SearchUsers(ctx echo.Context, query axon.QueryMap) ([]*User, error) {
    name := query.Get("name")
    return c.UserService.SearchUsers(name)
}

//axon::route GET /{id:int} -Priority=50
func (c *UserController) GetUser(id int) (*User, error) {
    return c.UserService.GetUser(id)
}
```

### 3. Generate Code & Run
```bash
# Generate all the magic
axon ./internal/...

# Use in your main.go
fx.New(
    controllers.AutogenModule,
    services.AutogenModule,
    middleware.AutogenModule,
).Run()
```

That's it! Axon handles routing, middleware, dependency injection, and parameter parsing automatically.

## Core Concepts

### Controllers with Smart Routing

Controllers are the heart of your API. Axon automatically generates route handlers, applies middleware, and manages dependencies.

```go
//axon::controller -Prefix=/api/v1/products -Middleware=AuthMiddleware -Priority=20
type ProductController struct {
    //axon::inject
    ProductService *services.ProductService
    //axon::inject  
    Logger *slog.Logger
}

//axon::route GET / -Priority=10
func (c *ProductController) ListProducts(ctx echo.Context, query axon.QueryMap) ([]*Product, error) {
    limit := query.GetIntDefault("limit", 10)
    return c.ProductService.List(limit)
}

//axon::route GET /{id:uuid.UUID} -Priority=50
func (c *ProductController) GetProduct(id uuid.UUID) (*Product, error) {
    return c.ProductService.GetByID(id)
}
```

**What Axon generates:**
- Echo route registration with proper middleware chains
- Type-safe parameter extraction and validation
- Automatic JSON serialization/deserialization
- FX dependency injection providers
- Route priority ordering (lower numbers = higher priority)

### Middleware Made Simple

Define middleware once, apply everywhere with perfect ordering control.

```go
//axon::middleware AuthMiddleware
type AuthMiddleware struct {
    //axon::inject
    JWTService *services.JWTService
}

func (m *AuthMiddleware) Handle(next echo.HandlerFunc) echo.HandlerFunc {
    return func(c echo.Context) error {
        token := c.Request().Header.Get("Authorization")
        if !m.JWTService.ValidateToken(token) {
            return echo.NewHTTPError(http.StatusUnauthorized, "Invalid token")
        }
        return next(c)
    }
}

//axon::middleware LoggingMiddleware -Global -Priority=50
type LoggingMiddleware struct {
    //axon::inject
    Logger *slog.Logger
}
```

**Apply middleware at any level:**
- **Global**: `//axon::middleware -Global -Priority=1` (Priority only works with -Global)
- **Controller**: `//axon::controller -Middleware=AuthMiddleware`  
- **Route**: `//axon::route GET /admin -Middleware=AdminMiddleware`

**Important Middleware Restrictions:**
- **Priority**: Only works when combined with `-Global` flag
- **Routes**: Middleware do not support the `-Routes` parameter (use controller/route level instead)

### Services with Lifecycle Management

Build robust services with automatic dependency injection and lifecycle hooks.

```go
//axon::core -Init
type DatabaseService struct {
    //axon::inject
    Config *config.Config
    connected bool
}

func (s *DatabaseService) Start(ctx context.Context) error {
    fmt.Printf("Connecting to database: %s\n", s.Config.DatabaseURL)
    s.connected = true
    return nil
}

func (s *DatabaseService) Stop(ctx context.Context) error {
    s.connected = false
    return nil
}

//axon::core -Init=Background -Mode=Transient
type BackgroundWorker struct {
    //axon::inject
    DatabaseService *DatabaseService
}

func (s *BackgroundWorker) Start(ctx context.Context) error {
    // This will run in its own goroutine
    go s.processJobs()
    return nil
}
// Injected as: func() *BackgroundWorker (new instance per request)
```

## Advanced Features

### Priority-Based Ordering

Control the exact order of controllers, routes, and middleware with priorities:

```go
// Routes with priorities (lower = registered first)
//axon::route GET /users/profile -Priority=10    // Matches before /{id}
//axon::route GET /users/admin -Priority=20      // Matches before /{id}  
//axon::route GET /users/{id:int} -Priority=50   // Catch-all for IDs

// Controllers with priorities
//axon::controller -Priority=10                  // API controllers first
//axon::controller -Priority=999                 // Catch-all controllers last

// Global middleware with priorities (Priority only works with -Global)
//axon::middleware SecurityMiddleware -Global -Priority=1    // Security first
//axon::middleware LoggingMiddleware -Global -Priority=50    // Logging later
```

### Type-Safe Query Parameters

No more manual parameter parsing or type conversion errors:

```go
//axon::route GET /search
func (c *Controller) Search(ctx echo.Context, query axon.QueryMap) (*SearchResult, error) {
    // All type-safe with automatic defaults
    term := query.Get("q")                        // string
    page := query.GetIntDefault("page", 1)        // int, defaults to 1
    limit := query.GetIntDefault("limit", 10)     // int, defaults to 10
    active := query.GetBool("active")             // bool, defaults to false
    price := query.GetFloat64("max_price")        // float64, defaults to 0.0
    
    return c.SearchService.Search(term, page, limit, active, price)
}
```

### Flexible Response Handling

Return data in the most natural way for your use case:

```go
// Simple data + error (most common)
func (c *Controller) GetUser(id int) (*User, error) {
    return c.UserService.GetUser(id) // Auto JSON + status codes
}

// Custom responses with full control
func (c *Controller) CreateUser(user User) (*axon.Response, error) {
    created, err := c.UserService.Create(user)
    if err != nil {
        return nil, err
    }
    
    return axon.Created(created).
        WithHeader("Location", fmt.Sprintf("/users/%d", created.ID)).
        WithSecureCookie("session", sessionID, "/", 3600), nil
}

// Error-only for operations
func (c *Controller) DeleteUser(id int) error {
    return c.UserService.Delete(id) // Auto 204 No Content
}
```

### Custom Parameter Parsers

Extend Axon with your own parameter types:

```go
//axon::route_parser ProductCode
func ParseProductCode(c echo.Context, value string) (ProductCode, error) {
    if !strings.HasPrefix(value, "PROD-") {
        return "", fmt.Errorf("invalid product code format")
    }
    return ProductCode(value), nil
}

// Use in routes
//axon::route GET /products/{code:ProductCode}
func (c *Controller) GetByCode(code ProductCode) (*Product, error) {
    return c.ProductService.GetByCode(string(code))
}
```

## Annotation Reference

### Controller Annotations

#### `//axon::controller [flags]`
Transform structs into powerful HTTP controllers.

**Flags:**
- `-Prefix=/path` - URL prefix for all routes (creates Echo groups)
- `-Middleware=Name1,Name2` - Apply middleware to all routes  
- `-Priority=N` - Registration order (lower = first, default: 100)

```go
//axon::controller -Prefix=/api/v1/users -Middleware=AuthMiddleware -Priority=10
type UserController struct {
    //axon::inject
    UserService *services.UserService
}
```

#### `//axon::route METHOD /path [flags]`
Define HTTP route handlers with automatic parameter binding.

**Flags:**
- `-Middleware=Name1,Name2` - Route-specific middleware
- `-Priority=N` - Route registration order (lower = first, default: 100)
- `-PassContext` - Inject `echo.Context` as first parameter

```go
//axon::route GET /search -Priority=10 -Middleware=LoggingMiddleware
func (c *Controller) SearchUsers(ctx echo.Context, query axon.QueryMap) ([]*User, error) {}

//axon::route GET /{id:int} -Priority=50
func (c *Controller) GetUser(id int) (*User, error) {}

//axon::route POST / -PassContext -Middleware=ValidationMiddleware
func (c *Controller) CreateUser(ctx echo.Context, user User) (*axon.Response, error) {}
```

### Middleware Annotations

#### `//axon::middleware Name [flags]`
Create reusable middleware components.

**Flags:**
- `-Priority=N` - Execution order for global middleware (lower = first, **only works with -Global**)
- `-Global` - Apply to all routes automatically

**Note**: Middleware do not support `-Routes` parameter. Use controller or route-level middleware instead.

```go
//axon::middleware AuthMiddleware
type AuthMiddleware struct {
    //axon::inject
    JWTService *services.JWTService
}

func (m *AuthMiddleware) Handle(next echo.HandlerFunc) echo.HandlerFunc {
    return func(c echo.Context) error {
        // Middleware logic here
        return next(c)
    }
}

// Global middleware with priority ordering
//axon::middleware SecurityMiddleware -Global -Priority=1
type SecurityMiddleware struct {}

//axon::middleware LoggingMiddleware -Global -Priority=50
type LoggingMiddleware struct {}
```

### Service Annotations

#### `//axon::core [flags]`
Define business services with lifecycle management.

**Flags:**
- `-Init[=Same|Background]` - Enable lifecycle hooks with execution mode
  - `Same`: Start/Stop runs on the same thread (blocking) - default if no value specified
  - `Background`: Start/Stop runs in its own goroutine (non-blocking)
- `-Mode=Singleton|Transient` - Instance lifecycle (default: Singleton)
- `-Manual=ModuleName` - Reference existing FX module

```go
//axon::core -Init
type DatabaseService struct {
    //axon::inject
    Config *config.Config
    connected bool
}

//axon::core -Init=Background
type CrawlerService struct {
    // Background service for async operations
}

//axon::core -Mode=Transient
type SessionService struct {
    //axon::inject
    DatabaseService *DatabaseService
    sessionID string
}

func (s *DatabaseService) Start(ctx context.Context) error {
    // Runs on same thread (blocking)
    var err error
    s.db, err = sql.Open("postgres", s.Config.DatabaseURL)
    return err
}

func (s *DatabaseService) Stop(ctx context.Context) error {
    return s.db.Close()
}

//axon::core -Init=Background -Mode=Singleton
type BackgroundWorker struct {
    //axon::inject
    DatabaseService *DatabaseService
}

func (s *BackgroundWorker) Start(ctx context.Context) error {
    // Runs in its own goroutine (non-blocking)
    s.processJobs()
    return nil
}
```

### Dependency Injection

#### `//axon::inject`
Mark fields for dependency injection.

#### `//axon::init`  
Mark fields for initialization (not injection).

```go
type UserService struct {
    //axon::inject
    DatabaseService *DatabaseService  // Injected dependency
    //axon::inject
    Logger *slog.Logger              // Injected dependency
    //axon::init
    cache map[string]*User           // Initialized field
    //axon::init
    mutex sync.RWMutex               // Initialized field
}
```

## CLI Commands

```bash
# Generate code for all packages
axon ./internal/...

# Generate specific packages  
axon ./internal/controllers ./internal/services

# Clean generated files
axon --clean ./...

# Verbose output for debugging
axon --verbose ./internal/...

# Custom module name
axon -module=github.com/your-org/app ./internal/...
```

## Project Structure

```
your-app/
├── cmd/
│   └── server/
│       └── main.go          # Application entry point
├── internal/
│   ├── controllers/         # HTTP controllers
│   │   ├── user_controller.go
│   │   └── autogen_module.go    # Generated
│   ├── services/           # Business logic
│   │   ├── user_service.go
│   │   └── autogen_module.go    # Generated
│   ├── middleware/         # HTTP middleware
│   │   ├── auth_middleware.go
│   │   └── autogen_module.go    # Generated
│   ├── models/             # Data models
│   ├── config/             # Configuration
│   └── parsers/            # Custom parameter parsers
├── pkg/                    # Public packages
├── go.mod
└── README.md
```

## Best Practices

### Use Priorities Strategically
```go
// Global middleware with priority (Priority ONLY works with -Global)
//axon::middleware SecurityMiddleware -Global -Priority=1

// Local middleware (no Priority support)
//axon::middleware AuthMiddleware

// Specific routes before parameterized ones
//axon::route GET /users/me -Priority=10
//axon::route GET /users/{id:int} -Priority=50

// Catch-all controllers last
//axon::controller -Priority=999
```

### Layer Your Middleware
```go
// Global: Security, CORS, Rate Limiting (with -Global flag)
//axon::middleware SecurityMiddleware -Global -Priority=1

// Controller: Authentication, Authorization  
//axon::controller -Middleware=AuthMiddleware

// Route: Validation, Caching
//axon::route POST /users -Middleware=ValidationMiddleware
```

### Design for Testing
```go
//axon::core
//axon::interface  // Generates interface for easy mocking
type UserService struct {
    //axon::inject
    UserRepo UserRepositoryInterface  // Use interface for testability
}
```

### Service Lifecycle Best Practices
```go
// Use -Init for services that need startup/shutdown logic
//axon::core -Init
type DatabaseService struct {}

// Use -Init=Background for non-blocking services
//axon::core -Init=Background  
type CrawlerService struct {}

// Use -Mode=Transient for request-scoped services
//axon::core -Mode=Transient
type SessionService struct {}

// Simple services don't need lifecycle hooks
//axon::core
type UtilityService struct {}
```

### Lifecycle Management
```go
// Blocking initialization (database connections, etc.)
//axon::core -Init
type DatabaseService struct {
    //axon::inject
    Config *config.Config
}

// Non-blocking initialization (background workers, etc.)
//axon::core -Init=Background
type CrawlerService struct {}
```

## Examples

### Complete REST API
```go
//axon::controller -Prefix=/api/v1/products -Middleware=AuthMiddleware -Priority=20
type ProductController struct {
    //axon::inject
    ProductService *services.ProductService
}

//axon::route GET / -Priority=10
func (c *ProductController) ListProducts(ctx echo.Context, query axon.QueryMap) ([]*Product, error) {
    limit := query.GetIntDefault("limit", 10)
    offset := query.GetIntDefault("offset", 0)
    category := query.Get("category")
    
    return c.ProductService.List(limit, offset, category)
}

//axon::route GET /featured -Priority=20
func (c *ProductController) GetFeatured() ([]*Product, error) {
    return c.ProductService.GetFeatured()
}

//axon::route GET /{id:uuid.UUID} -Priority=50
func (c *ProductController) GetProduct(id uuid.UUID) (*Product, error) {
    return c.ProductService.GetByID(id)
}

//axon::route POST / -Middleware=ValidationMiddleware
func (c *ProductController) CreateProduct(product CreateProductRequest) (*axon.Response, error) {
    created, err := c.ProductService.Create(product)
    if err != nil {
        return nil, err
    }
    
    return axon.Created(created).
        WithHeader("Location", fmt.Sprintf("/api/v1/products/%s", created.ID)), nil
}

//axon::route PUT /{id:uuid.UUID}
func (c *ProductController) UpdateProduct(id uuid.UUID, product UpdateProductRequest) (*Product, error) {
    return c.ProductService.Update(id, product)
}

//axon::route DELETE /{id:uuid.UUID} -Middleware=AdminMiddleware
func (c *ProductController) DeleteProduct(id uuid.UUID) error {
    return c.ProductService.Delete(id)
}
```

### Advanced Middleware Chain
```go
// Global middleware with priorities (Priority only works with -Global)
//axon::middleware SecurityMiddleware -Global -Priority=1
type SecurityMiddleware struct {}

//axon::middleware CORSMiddleware -Global -Priority=5  
type CORSMiddleware struct {}

// Local middleware (no Priority support)
//axon::middleware AuthMiddleware
type AuthMiddleware struct {}

//axon::middleware RateLimitMiddleware
type RateLimitMiddleware struct {}

//axon::middleware LoggingMiddleware -Global -Priority=50
type LoggingMiddleware struct {}
```

**Global middleware execution order:** Security → CORS → Logging → Handler
**Route-specific middleware:** Applied in the order specified on the route/controller

### Service Lifecycle Examples
```go
// Database service - blocking initialization (default Same mode)
//axon::core -Init
type DatabaseService struct {
    //axon::inject
    Config *config.Config
    connected bool
}

func (s *DatabaseService) Start(ctx context.Context) error {
    // Blocks application startup until database is connected
    fmt.Printf("Connecting to database: %s\n", s.Config.DatabaseURL)
    s.connected = true
    return nil
}

// Background worker - non-blocking initialization  
//axon::core -Init=Background
type CrawlerService struct {}

func (s *CrawlerService) Start(ctx context.Context) error {
    // Starts in background, doesn't block application startup
    go func() {
        for {
            select {
            case <-ctx.Done():
                return
            default:
                // Background processing
                time.Sleep(time.Second)
            }
        }
    }()
    return nil
}

// Transient service - new instance per request
//axon::core -Mode=Transient
type SessionService struct {
    //axon::inject
    DatabaseService *DatabaseService
    sessionID   string
    createdAt   time.Time
}

func (s *MetricsCollector) Start(ctx context.Context) error {
    // Runs in background, doesn't block application startup
    go s.collectMetrics()
    return nil
}
```

## Parameter Types and Parsing

### Built-in Parameter Types
```go
//axon::route GET /users/{id:int}/posts/{slug:string}
func (c *Controller) GetUserPost(id int, slug string) (*Post, error) {}

//axon::route GET /products/{id:uuid.UUID}
func (c *Controller) GetProduct(id uuid.UUID) (*Product, error) {}

//axon::route GET /search
func (c *Controller) Search(ctx echo.Context, query axon.QueryMap) (*SearchResult, error) {
    term := query.Get("q")                    // string
    page := query.GetIntDefault("page", 1)    // int with default
    active := query.GetBool("active")         // bool
    price := query.GetFloat64("max_price")    // float64
}
```

### Custom Parameter Parsers
```go
//axon::route_parser DateRange
func ParseDateRange(c echo.Context, value string) (DateRange, error) {
    parts := strings.Split(value, "_")
    if len(parts) != 2 {
        return DateRange{}, fmt.Errorf("invalid date range format")
    }
    
    start, err := time.Parse("2006-01-02", parts[0])
    if err != nil {
        return DateRange{}, err
    }
    
    end, err := time.Parse("2006-01-02", parts[1])
    if err != nil {
        return DateRange{}, err
    }
    
    return DateRange{Start: start, End: end}, nil
}

// Usage
//axon::route GET /sales/{period:DateRange}
func (c *Controller) GetSales(period DateRange) ([]*Sale, error) {
    return c.SalesService.GetSalesInRange(period.Start, period.End)
}
```

## Response Handling

### Standard Response Types

```go
// Data + Error (most common)
func (c *Controller) GetUser(id int) (*User, error) {
    user, err := c.UserService.GetUser(id)
    if err != nil {
        return nil, axon.ErrNotFound("User not found")
    }
    return user, nil
}
// Returns: 200 OK with JSON body, or custom HTTP status on axon.HttpError

// Custom Response with full control
func (c *Controller) CreateUser(user User) (*axon.Response, error) {
    return &axon.Response{
        StatusCode: 201,
        Body:       user,
        Headers: map[string]string{
            "Location": "/users/123",
        },
    }, nil
}

// Error Only (for operations)
func (c *Controller) DeleteUser(id int) error {
    return c.UserService.Delete(id)
}
// Returns: 204 No Content on success, custom HTTP status on axon.HttpError
```

### HTTP Error Handling

```go
// Common HTTP errors
return axon.ErrBadRequest("Invalid input")
return axon.ErrUnauthorized("Authentication required")
return axon.ErrForbidden("Access denied")
return axon.ErrNotFound("Resource not found")
return axon.ErrConflict("Resource already exists")

// Custom HTTP error
return axon.NewHttpError(418, "I'm a teapot")
```

### Response Builder API

```go
// Fluent response building
return axon.Created(user).
    WithHeader("Location", "/users/123").
    WithSecureCookie("session", "token", "/", 3600), nil

// Common responses
return axon.OK(data)
return axon.Created(user)
return axon.NoContent()
return axon.RedirectTo("/login")
```

## Generated Code Structure

Axon generates `autogen_module.go` files in each package:

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
func wrapUserControllerGetUser(controller *UserController) echo.HandlerFunc {}

// Route registration with middleware
func RegisterRoutes(e *echo.Echo, userController *UserController, authMiddleware *AuthMiddleware) {
    userGroup := e.Group("/api/v1/users")
    userGroup.GET("/search", wrapUserControllerSearchUsers(userController), authMiddleware.Handle)
    userGroup.GET("/:id", wrapUserControllerGetUser(userController), authMiddleware.Handle)
}

// FX Module
var AutogenModule = fx.Module("controllers",
    fx.Provide(NewUserController),
    fx.Invoke(RegisterRoutes),
)
```

## Integration

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

## Contributing

We welcome contributions! Please see our [Contributing Guide](CONTRIBUTING.md) for details.

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Add tests for new functionality
4. Ensure all tests pass (`go test ./...`)
5. Submit a pull request

## License

MIT License - see [LICENSE](LICENSE) file for details.

---

**Ready to build something amazing?** Check out our [complete example application](./examples/complete-app/) to see Axon in action!