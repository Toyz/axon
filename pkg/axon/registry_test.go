package axon

import (
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
)

func TestInMemoryRouteRegistry_RegisterRoute(t *testing.T) {
	registry := NewInMemoryRouteRegistry()

	route := RouteInfo{
		Method:         "GET",
		Path:           "/users/{id}",
		HandlerName:    "GetUser",
		ControllerName: "UserController",
		PackageName:    "controllers",
		Middlewares:    []string{"auth", "logging"},
		Handler:        func(c echo.Context) error { return nil },
	}

	registry.RegisterRoute(route)

	routes := registry.GetAllRoutes()
	assert.Len(t, routes, 1)
	assert.Equal(t, route.Method, routes[0].Method)
	assert.Equal(t, route.Path, routes[0].Path)
	assert.Equal(t, route.HandlerName, routes[0].HandlerName)
	assert.Equal(t, route.ControllerName, routes[0].ControllerName)
	assert.Equal(t, route.PackageName, routes[0].PackageName)
	assert.Equal(t, route.Middlewares, routes[0].Middlewares)
}

func TestInMemoryRouteRegistry_GetRoutesByPackage(t *testing.T) {
	registry := NewInMemoryRouteRegistry()

	route1 := RouteInfo{
		Method:      "GET",
		Path:        "/users",
		PackageName: "controllers",
	}
	route2 := RouteInfo{
		Method:      "POST",
		Path:        "/users",
		PackageName: "controllers",
	}
	route3 := RouteInfo{
		Method:      "GET",
		Path:        "/health",
		PackageName: "health",
	}

	registry.RegisterRoute(route1)
	registry.RegisterRoute(route2)
	registry.RegisterRoute(route3)

	controllerRoutes := registry.GetRoutesByPackage("controllers")
	assert.Len(t, controllerRoutes, 2)

	healthRoutes := registry.GetRoutesByPackage("health")
	assert.Len(t, healthRoutes, 1)
	assert.Equal(t, "/health", healthRoutes[0].Path)
}

func TestInMemoryRouteRegistry_GetRoutesByController(t *testing.T) {
	registry := NewInMemoryRouteRegistry()

	route1 := RouteInfo{
		Method:         "GET",
		Path:           "/users",
		ControllerName: "UserController",
	}
	route2 := RouteInfo{
		Method:         "POST",
		Path:           "/users",
		ControllerName: "UserController",
	}
	route3 := RouteInfo{
		Method:         "GET",
		Path:           "/posts",
		ControllerName: "PostController",
	}

	registry.RegisterRoute(route1)
	registry.RegisterRoute(route2)
	registry.RegisterRoute(route3)

	userRoutes := registry.GetRoutesByController("UserController")
	assert.Len(t, userRoutes, 2)

	postRoutes := registry.GetRoutesByController("PostController")
	assert.Len(t, postRoutes, 1)
	assert.Equal(t, "/posts", postRoutes[0].Path)
}

func TestInMemoryRouteRegistry_GetRoutesByMethod(t *testing.T) {
	registry := NewInMemoryRouteRegistry()

	route1 := RouteInfo{Method: "GET", Path: "/users"}
	route2 := RouteInfo{Method: "POST", Path: "/users"}
	route3 := RouteInfo{Method: "GET", Path: "/posts"}

	registry.RegisterRoute(route1)
	registry.RegisterRoute(route2)
	registry.RegisterRoute(route3)

	getRoutes := registry.GetRoutesByMethod("GET")
	assert.Len(t, getRoutes, 2)

	postRoutes := registry.GetRoutesByMethod("POST")
	assert.Len(t, postRoutes, 1)
	assert.Equal(t, "/users", postRoutes[0].Path)
}

func TestConvenienceFunctions(t *testing.T) {
	// Reset the default registry for testing
	DefaultRouteRegistry = NewInMemoryRouteRegistry()

	route := RouteInfo{
		Method:         "GET",
		Path:           "/test",
		ControllerName: "TestController",
		PackageName:    "test",
	}

	DefaultRouteRegistry.RegisterRoute(route)

	// Test convenience functions
	allRoutes := GetRoutes()
	assert.Len(t, allRoutes, 1)

	packageRoutes := GetRoutesByPackage("test")
	assert.Len(t, packageRoutes, 1)

	controllerRoutes := GetRoutesByController("TestController")
	assert.Len(t, controllerRoutes, 1)
}

func TestMiddlewareRegistry(t *testing.T) {
	// Reset the default middleware registry for testing
	DefaultMiddlewareRegistry = NewInMemoryMiddlewareRegistry()

	// Test middleware registration
	mockHandler := func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			return next(c)
		}
	}

	mockInstance := struct{ Name string }{Name: "TestMiddleware"}

	RegisterMiddleware("TestAuth", mockHandler, mockInstance)

	// Test middleware retrieval
	middleware, exists := GetMiddleware("TestAuth")
	assert.True(t, exists)
	assert.Equal(t, "TestAuth", middleware.Name)
	assert.NotNil(t, middleware.Handler)
	assert.Equal(t, mockInstance, middleware.Instance)

	// Test non-existent middleware
	_, exists = GetMiddleware("NonExistent")
	assert.False(t, exists)

	// Test get all middlewares
	allMiddlewares := GetAllMiddlewares()
	assert.Len(t, allMiddlewares, 1)
	assert.Equal(t, "TestAuth", allMiddlewares[0].Name)
}

func TestRouteWithMiddlewareInstances(t *testing.T) {
	// Reset registries for testing
	DefaultRouteRegistry = NewInMemoryRouteRegistry()
	DefaultMiddlewareRegistry = NewInMemoryMiddlewareRegistry()

	// Register middleware
	authHandler := func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			return next(c)
		}
	}
	RegisterMiddleware("Auth", authHandler, "AuthInstance")

	// Create route with middleware instances
	authMiddleware, _ := GetMiddleware("Auth")
	route := RouteInfo{
		Method:              "GET",
		Path:                "/protected",
		ControllerName:      "TestController",
		PackageName:         "test",
		Middlewares:         []string{"Auth"},
		MiddlewareInstances: []MiddlewareInstance{authMiddleware},
	}

	DefaultRouteRegistry.RegisterRoute(route)

	// Test middleware access through route
	routes := GetRoutes()
	assert.Len(t, routes, 1)

	routeMiddlewares := GetMiddlewaresByRoute(routes[0])
	assert.Len(t, routeMiddlewares, 1)
	assert.Equal(t, "Auth", routeMiddlewares[0].Name)
	assert.Equal(t, "AuthInstance", routeMiddlewares[0].Instance)
}
