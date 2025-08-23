# Axon Public API

This package provides public APIs for the Axon Framework that users can import and use in their applications.

## Response Struct

The `Response` struct allows you to control HTTP status codes and response bodies from your route handlers.

```go
package controllers

import "github.com/toyz/axon/pkg/axon"

//axon::controller
type UserController struct{}

//axon::route POST /users
func (c *UserController) CreateUser(user User) (*axon.Response, error) {
    // ... create user logic ...
    
    if user.Email == "" {
        return axon.BadRequest("Email is required"), nil
    }
    
    createdUser, err := c.userService.Create(user)
    if err != nil {
        return axon.InternalServerError("Failed to create user"), nil
    }
    
    return axon.Created(createdUser), nil
}
```

### Available Response Helpers

- `axon.OK(body)` - 200 OK with body
- `axon.Created(body)` - 201 Created with body  
- `axon.NoContent()` - 204 No Content
- `axon.BadRequest(message)` - 400 Bad Request with error message
- `axon.NotFound(message)` - 404 Not Found with error message
- `axon.InternalServerError(message)` - 500 Internal Server Error with error message
- `axon.NewResponse(statusCode, body)` - Custom status code and body

## Setting Up Your Server

Axon generates FX modules that you wire together in your own main.go. Here's the recommended approach:

### 1. Define Your Components with Annotations

**Controllers:**
```go
package controllers

//axon::controller
type UserController struct {
    UserService UserServiceInterface `fx.In`
}

//axon::route GET /users/{id:int}
func (c *UserController) GetUser(id int) (*axon.Response, error) {
    user, err := c.UserService.GetByID(id)
    if err != nil {
        return axon.NotFound("User not found"), nil
    }
    return axon.OK(user), nil
}

//axon::route POST /users -Middleware=Auth,RateLimit
func (c *UserController) CreateUser(user User) (*axon.Response, error) {
    created, err := c.UserService.Create(user)
    if err != nil {
        return axon.InternalServerError("Failed to create user"), nil
    }
    return axon.Created(created), nil
}
```

**Services:**
```go
package services

//axon::core -Init
type UserService struct {
    DB *sql.DB `fx.In`
}

func (s *UserService) Start(ctx context.Context) error {
    // Initialize service
    return nil
}

func (s *UserService) Stop(ctx context.Context) error {
    // Cleanup
    return nil
}

func (s *UserService) GetByID(id int) (*User, error) {
    // Implementation
}

func (s *UserService) Create(user User) (*User, error) {
    // Implementation
}
```

**Interfaces:**
```go
package services

//axon::interface
type UserService struct {
    // The generator will create UserServiceInterface
    // and provide both the concrete type and interface
}
```

**Middleware:**
```go
package middleware

//axon::middleware Auth
type AuthMiddleware struct {
    TokenService TokenServiceInterface `fx.In`
}

func (m *AuthMiddleware) Handle(next echo.HandlerFunc) echo.HandlerFunc {
    return func(c echo.Context) error {
        token := c.Request().Header.Get("Authorization")
        if !m.TokenService.ValidateToken(token) {
            return echo.NewHTTPError(http.StatusUnauthorized, "Invalid token")
        }
        return next(c)
    }
}

//axon::middleware RateLimit
type RateLimitMiddleware struct{}

func (m *RateLimitMiddleware) Handle(next echo.HandlerFunc) echo.HandlerFunc {
    return func(c echo.Context) error {
        // Rate limiting logic
        return next(c)
    }
}
```

### 2. Generate Your Modules

```bash
# Generate FX modules for all your annotated packages
axon ./internal/controllers ./internal/services ./internal/middleware
```

This creates `autogen_module.go` files in each package with the FX providers and route registration.

### 3. Wire Everything Together in main.go

```go
package main

import (
    "context"
    "database/sql"
    "log"
    "os"

    "github.com/labstack/echo/v4"
    "github.com/labstack/echo/v4/middleware"
    "go.uber.org/fx"
    _ "github.com/lib/pq"

    // Your generated modules
    "myapp/internal/controllers"
    "myapp/internal/services"
    "myapp/internal/middleware"
)

func main() {
    fx.New(
        // Provide external dependencies
        fx.Provide(
            NewEcho,
            NewDatabase,
        ),

        // Include all your generated modules
        controllers.AutogenModule,
        services.AutogenModule,
        middleware.AutogenModule,

        // Start the server
        fx.Invoke(StartServer),
    ).Run()
}

func NewEcho() *echo.Echo {
    e := echo.New()
    e.Use(middleware.Logger())
    e.Use(middleware.Recover())
    return e
}

func NewDatabase() *sql.DB {
    db, err := sql.Open("postgres", os.Getenv("DATABASE_URL"))
    if err != nil {
        log.Fatal(err)
    }
    return db
}

func StartServer(e *echo.Echo, lc fx.Lifecycle) {
    lc.Append(fx.Hook{
        OnStart: func(ctx context.Context) error {
            go func() {
                if err := e.Start(":8080"); err != nil {
                    log.Printf("Server error: %v", err)
                }
            }()
            return nil
        },
        OnStop: func(ctx context.Context) error {
            return e.Shutdown(ctx)
        },
    })
}
```

## Route Registry

Access information about all your registered routes:

```go
package main

import (
    "log"
    "github.com/toyz/axon/pkg/axon"
)

func main() {
    // Get all registered routes
    routes := axon.GetRoutes()
    log.Printf("Found %d routes:", len(routes))
    
    for _, route := range routes {
        log.Printf("  %s %s -> %s.%s", 
            route.Method, route.Path, route.ControllerName, route.HandlerName)
    }
    
    // Filter routes by package
    controllerRoutes := axon.GetRoutesByPackage("controllers")
    
    // Filter routes by controller
    userRoutes := axon.GetRoutesByController("UserController")
    
    // Filter routes by HTTP method
    getRoutes := axon.GetRoutesByMethod("GET")
}
```

### RouteInfo Structure

Each route contains the following information:

```go
type RouteInfo struct {
    Method         string           // HTTP method (GET, POST, etc.)
    Path           string           // Route path (/users/{id})
    HandlerName    string           // Handler method name (GetUser)
    ControllerName string           // Controller name (UserController)
    PackageName    string           // Package name (controllers)
    Middlewares    []string         // Applied middleware names
    Handler        echo.HandlerFunc // Actual Echo handler function
}
```

## Manual Route Registration

If you need to register routes manually with an existing Echo instance:

```go
package main

import (
    "github.com/labstack/echo/v4"
    "github.com/toyz/axon/pkg/axon"
)

func main() {
    e := echo.New()
    
    // Register all discovered routes with Echo
    axon.RegisterAllRoutes(e)
    
    // Start server
    e.Start(":8080")
}
```

## Route Path Conversion

Axon uses its own route syntax which is automatically converted to Echo syntax:

- Axon: `/users/{id:int}` → Echo: `/users/:id`
- Axon: `/posts/{slug:string}` → Echo: `/posts/:slug`
- Axon: `/users/{userId:int}/posts/{postId:int}` → Echo: `/users/:userId/posts/:postId`

You can also manually convert paths:

```go
echoPath := axon.ConvertAxonPathToEcho("/users/{id:int}")
// echoPath is now "/users/:id"
```

## Best Practices

1. **Keep annotations simple**: Use the framework's annotations instead of manual configuration
2. **Organize by feature**: Group related controllers, services, and middleware in the same package
3. **Use interfaces**: Generate interfaces for your services to enable easy testing and mocking
4. **Leverage FX lifecycle**: Use `-Init` flag on services that need startup/shutdown hooks
5. **Compose middleware**: Apply multiple middleware to routes using `-Middleware=Auth,RateLimit`
6. **Control your main.go**: The framework generates modules, you control the application structure