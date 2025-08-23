package axon

import (
	"strings"
	"github.com/labstack/echo/v4"
)

// MiddlewareInstance represents a middleware with its name and handler
type MiddlewareInstance struct {
	// Name is the middleware name as defined in annotations
	Name string
	
	// Handler is the middleware function that can be applied to routes
	Handler func(echo.HandlerFunc) echo.HandlerFunc
	
	// Instance is the actual middleware struct instance (if available)
	Instance interface{}
}

// MiddlewareRegistry provides access to all registered middlewares
type MiddlewareRegistry interface {
	// RegisterMiddleware adds a middleware to the registry
	RegisterMiddleware(name string, handler func(echo.HandlerFunc) echo.HandlerFunc, instance interface{})
	
	// GetMiddleware retrieves a middleware by name
	GetMiddleware(name string) (MiddlewareInstance, bool)
	
	// GetAllMiddlewares returns all registered middlewares
	GetAllMiddlewares() []MiddlewareInstance
}

// inMemoryMiddlewareRegistry implements MiddlewareRegistry
type inMemoryMiddlewareRegistry struct {
	middlewares map[string]MiddlewareInstance
}

// NewInMemoryMiddlewareRegistry creates a new in-memory middleware registry
func NewInMemoryMiddlewareRegistry() MiddlewareRegistry {
	return &inMemoryMiddlewareRegistry{
		middlewares: make(map[string]MiddlewareInstance),
	}
}

func (r *inMemoryMiddlewareRegistry) RegisterMiddleware(name string, handler func(echo.HandlerFunc) echo.HandlerFunc, instance interface{}) {
	r.middlewares[name] = MiddlewareInstance{
		Name:     name,
		Handler:  handler,
		Instance: instance,
	}
}

func (r *inMemoryMiddlewareRegistry) GetMiddleware(name string) (MiddlewareInstance, bool) {
	middleware, exists := r.middlewares[name]
	return middleware, exists
}

func (r *inMemoryMiddlewareRegistry) GetAllMiddlewares() []MiddlewareInstance {
	result := make([]MiddlewareInstance, 0, len(r.middlewares))
	for _, middleware := range r.middlewares {
		result = append(result, middleware)
	}
	return result
}

// DefaultMiddlewareRegistry is the global middleware registry
var DefaultMiddlewareRegistry MiddlewareRegistry = NewInMemoryMiddlewareRegistry()

// RouteInfo contains metadata about a registered route
type RouteInfo struct {
	// Method is the HTTP method (GET, POST, PUT, DELETE, etc.)
	Method string
	
	// Path is the original Axon route path with parameter placeholders (e.g., "/users/{id:int}")
	Path string
	
	// EchoPath is the Echo-compatible route path (e.g., "/users/:id")
	EchoPath string
	
	// HandlerName is the name of the handler function
	HandlerName string
	
	// ControllerName is the name of the controller that owns this route
	ControllerName string
	
	// PackageName is the name of the package containing the controller
	PackageName string
	
	// Middlewares is a list of middleware names applied to this route
	Middlewares []string
	
	// MiddlewareInstances provides access to the actual middleware instances
	MiddlewareInstances []MiddlewareInstance
	
	// ParameterTypes maps parameter names to their types (e.g., {"id": "int", "slug": "string"})
	ParameterTypes map[string]string
	
	// Handler is the actual Echo handler function
	Handler echo.HandlerFunc
}

// RouteRegistry provides access to all registered routes in the application
type RouteRegistry interface {
	// GetAllRoutes returns all registered routes
	GetAllRoutes() []RouteInfo
	
	// GetRoutesByPackage returns routes filtered by package name
	GetRoutesByPackage(packageName string) []RouteInfo
	
	// GetRoutesByController returns routes filtered by controller name
	GetRoutesByController(controllerName string) []RouteInfo
	
	// GetRoutesByMethod returns routes filtered by HTTP method
	GetRoutesByMethod(method string) []RouteInfo
	
	// RegisterRoute adds a route to the registry (used internally by generated code)
	RegisterRoute(route RouteInfo)
}

// DefaultRouteRegistry is the global route registry instance
var DefaultRouteRegistry RouteRegistry = NewInMemoryRouteRegistry()

// InMemoryRouteRegistry implements RouteRegistry using an in-memory slice
type InMemoryRouteRegistry struct {
	routes []RouteInfo
}

// NewInMemoryRouteRegistry creates a new in-memory route registry
func NewInMemoryRouteRegistry() *InMemoryRouteRegistry {
	return &InMemoryRouteRegistry{
		routes: make([]RouteInfo, 0),
	}
}

// GetAllRoutes returns all registered routes
func (r *InMemoryRouteRegistry) GetAllRoutes() []RouteInfo {
	return append([]RouteInfo(nil), r.routes...) // Return a copy
}

// GetRoutesByPackage returns routes filtered by package name
func (r *InMemoryRouteRegistry) GetRoutesByPackage(packageName string) []RouteInfo {
	var filtered []RouteInfo
	for _, route := range r.routes {
		if route.PackageName == packageName {
			filtered = append(filtered, route)
		}
	}
	return filtered
}

// GetRoutesByController returns routes filtered by controller name
func (r *InMemoryRouteRegistry) GetRoutesByController(controllerName string) []RouteInfo {
	var filtered []RouteInfo
	for _, route := range r.routes {
		if route.ControllerName == controllerName {
			filtered = append(filtered, route)
		}
	}
	return filtered
}

// GetRoutesByMethod returns routes filtered by HTTP method
func (r *InMemoryRouteRegistry) GetRoutesByMethod(method string) []RouteInfo {
	var filtered []RouteInfo
	for _, route := range r.routes {
		if route.Method == method {
			filtered = append(filtered, route)
		}
	}
	return filtered
}

// RegisterRoute adds a route to the registry
func (r *InMemoryRouteRegistry) RegisterRoute(route RouteInfo) {
	r.routes = append(r.routes, route)
}

// RegisterAllRoutes is a helper function that registers all routes with an Echo instance
// This function should be called in your main.go to set up all discovered routes
func RegisterAllRoutes(e *echo.Echo) {
	routes := DefaultRouteRegistry.GetAllRoutes()
	for _, route := range routes {
		// Use EchoPath for registration with Echo (converts {id:int} to :id)
		e.Add(route.Method, route.EchoPath, route.Handler)
	}
}

// GetRoutes returns all registered routes (convenience function)
func GetRoutes() []RouteInfo {
	return DefaultRouteRegistry.GetAllRoutes()
}

// GetRoutesByPackage returns routes for a specific package (convenience function)
func GetRoutesByPackage(packageName string) []RouteInfo {
	return DefaultRouteRegistry.GetRoutesByPackage(packageName)
}

// GetRoutesByController returns routes for a specific controller (convenience function)
func GetRoutesByController(controllerName string) []RouteInfo {
	return DefaultRouteRegistry.GetRoutesByController(controllerName)
}

// Middleware convenience functions

// RegisterMiddleware registers a middleware with the global registry
func RegisterMiddleware(name string, handler func(echo.HandlerFunc) echo.HandlerFunc, instance interface{}) {
	DefaultMiddlewareRegistry.RegisterMiddleware(name, handler, instance)
}

// GetMiddleware retrieves a middleware by name from the global registry
func GetMiddleware(name string) (MiddlewareInstance, bool) {
	return DefaultMiddlewareRegistry.GetMiddleware(name)
}

// GetAllMiddlewares returns all registered middlewares from the global registry
func GetAllMiddlewares() []MiddlewareInstance {
	return DefaultMiddlewareRegistry.GetAllMiddlewares()
}

// GetMiddlewaresByRoute returns the middleware instances for a specific route
func GetMiddlewaresByRoute(route RouteInfo) []MiddlewareInstance {
	return route.MiddlewareInstances
}

// ConvertAxonPathToEcho converts Axon route syntax to Echo route syntax
// Axon: /users/{id:int} -> Echo: /users/:id
// Axon: /posts/{slug:string} -> Echo: /posts/:slug
func ConvertAxonPathToEcho(axonPath string) string {
	// Replace Axon parameter syntax {param:type} with Echo syntax :param
	result := axonPath
	
	// Find all {param:type} patterns and replace with :param
	for {
		start := strings.Index(result, "{")
		if start == -1 {
			break
		}
		
		end := strings.Index(result[start:], "}")
		if end == -1 {
			break
		}
		end += start
		
		// Extract parameter definition: {id:int} -> id:int
		paramDef := result[start+1 : end]
		
		// Split by colon to get parameter name
		parts := strings.Split(paramDef, ":")
		if len(parts) > 0 {
			paramName := strings.TrimSpace(parts[0])
			// Replace {param:type} with :param
			result = result[:start] + ":" + paramName + result[end+1:]
		} else {
			// Invalid format, skip
			break
		}
	}
	
	return result
}