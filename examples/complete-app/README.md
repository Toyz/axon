# Axon Framework Complete Example Application

This is a comprehensive example application demonstrating all features of the Axon framework, including:

- **Controllers** with HTTP route handling and dependency injection
- **Middleware** for cross-cutting concerns
- **Core Services** with lifecycle management and modes (Singleton/Transient)
- **Interface Generation** for dependency injection
- **Custom Parameter Parsers** for type conversion
- **Response Handling** with custom status codes
- **Route Registry** for runtime introspection

## Project Structure

```
examples/complete-app/
├── internal/
│   ├── controllers/          # HTTP controllers with route annotations
│   │   ├── user_controller.go
│   │   ├── product_controller.go
│   │   ├── health_controller.go
│   │   ├── session_controller.go  # Demonstrates transient services
│   │   └── autogen_module.go  # Generated FX module
│   ├── services/             # Business logic services
│   │   ├── user_service.go
│   │   ├── database_service.go
│   │   ├── session_service.go  # Transient service example
│   │   └── autogen_module.go  # Generated FX module
│   ├── middleware/           # HTTP middleware components
│   │   ├── logging_middleware.go
│   │   ├── auth_middleware.go
│   │   └── autogen_module.go  # Generated FX module
│   ├── interfaces/           # Interface generation examples
│   │   ├── user_interface.go
│   │   └── autogen_module.go  # Generated FX module
│   ├── logging/              # Logger services
│   │   ├── logger.go
│   │   └── autogen_module.go  # Generated FX module
│   ├── parsers/              # Custom parameter parsers
│   │   └── uuid_parser.go
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
    //axon::inject
    UserService *services.UserService
    //axon::inject
    DatabaseService *services.DatabaseService
}

//axon::route GET /users/{id:int} -Middleware=LoggingMiddleware
func (c *UserController) GetUser(id int) (*models.User, error) {
    return c.UserService.GetUser(id)
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
    //axon::inject
    Config *config.Config
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
    //axon::inject
    Config *config.Config
    //axon::inject
    DatabaseService *DatabaseService
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

#### Service Lifecycle Modes

Axon supports different lifecycle modes for services, similar to C# dependency injection:

**Singleton Mode (Default):**
```go
//axon::core                    // Default to Singleton
//axon::core -Mode=Singleton    // Explicit Singleton
type DatabaseService struct {
    //axon::inject
    Config *config.Config
}
```
- Creates a single shared instance across the application
- Perfect for stateless services, database connections, configuration

**Transient Mode:**
```go
//axon::core -Mode=Transient
type SessionService struct {
    //axon::inject
    DatabaseService *DatabaseService
}
```
- Creates a new instance every time it's requested
- Perfect for stateful services, per-request sessions, temporary objects
- Injected as a factory function: `func() *SessionService`

**Usage Example:**
```go
//axon::controller
type SessionController struct {
    //axon::inject
    SessionFactory func() *services.SessionService  // Transient service factory
    //axon::inject
    UserService    *services.UserService           // Singleton service
}

func (c *SessionController) HandleRequest() {
    // Get a fresh session instance for this request
    session := c.SessionFactory()
    
    // Use the shared user service
    user, _ := c.UserService.GetUser(123)
}
```

### 4. Interface Generation (`internal/interfaces/`)

Generate interfaces from structs for better testability:

```go
//axon::core
//axon::interface
type UserRepository struct {
    //axon::init
    data map[int]*models.User
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

### 2. Run the Application

```bash
cd examples/complete-app
go run main.go
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

### Product Management
- `GET /products` - List all products
- `GET /products/{id}` - Get product by ID
- `POST /products` - Create new product

### Session Management
- `POST /sessions/{userID:int}` - Start a new session (demonstrates transient services)
- `GET /sessions/info/{userID:int}` - Get session information
- `GET /sessions/compare` - Compare multiple session instances

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

## Custom Parameter Parsers

The example includes custom parameter parsers for advanced type conversion:

```go
//axon::parser uuid.UUID
func ParseUUID(c echo.Context, value string) (uuid.UUID, error) {
    return uuid.Parse(value)
}
```

This allows routes to use UUID parameters directly:
```go
//axon::route GET /users/{id:uuid.UUID}
func (c *UserController) GetUserByUUID(id uuid.UUID) (*User, error) {
    // id is already parsed as uuid.UUID
}
```

## Key Concepts

### Annotation-Driven Development
All framework features are configured through code comments using the `axon::` prefix:
- `//axon::controller` - Marks HTTP controllers
- `//axon::route METHOD /path` - Defines HTTP routes
- `//axon::middleware Name` - Defines middleware components
- `//axon::core` - Marks core services
- `//axon::interface` - Generates interfaces from structs
- `//axon::inject` - Marks dependencies for injection
- `//axon::init` - Marks fields for initialization (not injection)
- `//axon::logger` - Marks logger services
- `//axon::parser Type` - Defines custom parameter parsers

### Dependency Injection
The framework uses Uber FX for dependency injection. Dependencies are declared using `//axon::inject` annotations:
```go
type UserController struct {
    //axon::inject
    UserService *services.UserService
    //axon::inject
    DatabaseService *services.DatabaseService
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