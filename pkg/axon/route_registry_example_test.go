package axon

import (
	"fmt"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
)

// TestRouteRegistryUsageExample demonstrates how the route registry would be used
// in a real application with generated code
func TestRouteRegistryUsageExample(t *testing.T) {
	// Reset the global registry for this test
	originalRegistry := DefaultRouteRegistry
	defer func() {
		DefaultRouteRegistry = originalRegistry
	}()
	DefaultRouteRegistry = NewInMemoryRouteRegistry()

	// Simulate what generated code would do - register routes
	simulateGeneratedRouteRegistration()

	// Now demonstrate how a developer would use the route registry
	
	// 1. Get all routes for documentation or debugging
	allRoutes := GetRoutes()
	assert.Len(t, allRoutes, 4)
	
	fmt.Println("All registered routes:")
	for _, route := range allRoutes {
		fmt.Printf("  %s %s -> %s.%s (middlewares: %v)\n", 
			route.Method, route.Path, route.ControllerName, route.HandlerName, route.Middlewares)
	}

	// 2. Get routes by package (useful for modular applications)
	apiRoutes := GetRoutesByPackage("api")
	assert.Len(t, apiRoutes, 3)
	
	fmt.Println("\nAPI package routes:")
	for _, route := range apiRoutes {
		fmt.Printf("  %s %s\n", route.Method, route.Path)
	}

	// 3. Get routes by controller (useful for testing specific controllers)
	userRoutes := GetRoutesByController("UserController")
	assert.Len(t, userRoutes, 2)
	
	fmt.Println("\nUserController routes:")
	for _, route := range userRoutes {
		fmt.Printf("  %s %s -> %s\n", route.Method, route.Path, route.HandlerName)
	}

	// 4. Get routes by HTTP method (useful for analyzing API surface)
	getRoutes := DefaultRouteRegistry.GetRoutesByMethod("GET")
	assert.Len(t, getRoutes, 3)
	
	fmt.Println("\nGET routes:")
	for _, route := range getRoutes {
		fmt.Printf("  %s -> %s.%s\n", route.Path, route.ControllerName, route.HandlerName)
	}

	// 5. Set up Echo server with all registered routes
	e := echo.New()
	RegisterAllRoutes(e)
	
	// Verify Echo has the routes
	echoRoutes := e.Routes()
	assert.Len(t, echoRoutes, 4)
	
	fmt.Println("\nRoutes registered with Echo:")
	for _, route := range echoRoutes {
		fmt.Printf("  %s %s\n", route.Method, route.Path)
	}

	// 6. Demonstrate route metadata access
	for _, route := range allRoutes {
		if len(route.ParameterTypes) > 0 {
			fmt.Printf("\nRoute %s %s has parameters:\n", route.Method, route.Path)
			for name, typ := range route.ParameterTypes {
				fmt.Printf("  %s: %s\n", name, typ)
			}
		}
	}
}

// simulateGeneratedRouteRegistration simulates what the generated code would do
// This represents the calls that would be generated in autogen_module.go files
func simulateGeneratedRouteRegistration() {
	// Simulate registering routes from a UserController
	DefaultRouteRegistry.RegisterRoute(RouteInfo{
		Method:         "GET",
		Path:           "/users/{id:int}",
		EchoPath:       "/users/:id",
		HandlerName:    "GetUser",
		ControllerName: "UserController",
		PackageName:    "api",
		Middlewares:    []string{"Auth", "Logging"},
		ParameterTypes: map[string]string{"id": "int"},
		Handler:        func(c echo.Context) error { return c.JSON(200, map[string]string{"user": "data"}) },
	})

	DefaultRouteRegistry.RegisterRoute(RouteInfo{
		Method:         "POST",
		Path:           "/users",
		EchoPath:       "/users",
		HandlerName:    "CreateUser",
		ControllerName: "UserController",
		PackageName:    "api",
		Middlewares:    []string{"Auth", "Validation"},
		ParameterTypes: map[string]string{},
		Handler:        func(c echo.Context) error { return c.JSON(201, map[string]string{"created": "user"}) },
	})

	// Simulate registering routes from a PostController
	DefaultRouteRegistry.RegisterRoute(RouteInfo{
		Method:         "GET",
		Path:           "/posts/{slug:string}",
		EchoPath:       "/posts/:slug",
		HandlerName:    "GetPost",
		ControllerName: "PostController",
		PackageName:    "api",
		Middlewares:    []string{"Logging"},
		ParameterTypes: map[string]string{"slug": "string"},
		Handler:        func(c echo.Context) error { return c.JSON(200, map[string]string{"post": "data"}) },
	})

	// Simulate registering a health check route
	DefaultRouteRegistry.RegisterRoute(RouteInfo{
		Method:         "GET",
		Path:           "/health",
		EchoPath:       "/health",
		HandlerName:    "HealthCheck",
		ControllerName: "HealthController",
		PackageName:    "health",
		Middlewares:    []string{},
		ParameterTypes: map[string]string{},
		Handler:        func(c echo.Context) error { return c.JSON(200, map[string]string{"status": "ok"}) },
	})
}

// TestRouteRegistryDocumentationGeneration shows how the route registry could be used
// to generate API documentation
func TestRouteRegistryDocumentationGeneration(t *testing.T) {
	// Reset the global registry for this test
	originalRegistry := DefaultRouteRegistry
	defer func() {
		DefaultRouteRegistry = originalRegistry
	}()
	DefaultRouteRegistry = NewInMemoryRouteRegistry()

	// Register some example routes
	simulateGeneratedRouteRegistration()

	// Generate simple API documentation
	fmt.Println("=== API Documentation ===")
	
	packages := make(map[string][]RouteInfo)
	allRoutes := GetRoutes()
	
	// Group routes by package
	for _, route := range allRoutes {
		packages[route.PackageName] = append(packages[route.PackageName], route)
	}

	// Generate documentation by package
	for packageName, routes := range packages {
		fmt.Printf("\n## Package: %s\n", packageName)
		
		controllers := make(map[string][]RouteInfo)
		for _, route := range routes {
			controllers[route.ControllerName] = append(controllers[route.ControllerName], route)
		}
		
		for controllerName, controllerRoutes := range controllers {
			fmt.Printf("\n### %s\n", controllerName)
			
			for _, route := range controllerRoutes {
				fmt.Printf("- **%s** `%s`", route.Method, route.Path)
				if len(route.Middlewares) > 0 {
					fmt.Printf(" (middlewares: %v)", route.Middlewares)
				}
				if len(route.ParameterTypes) > 0 {
					fmt.Printf(" (params: %v)", route.ParameterTypes)
				}
				fmt.Printf("\n")
			}
		}
	}

	// Verify we have the expected structure
	assert.Contains(t, packages, "api")
	assert.Contains(t, packages, "health")
	assert.Len(t, packages["api"], 3)
	assert.Len(t, packages["health"], 1)
}

// TestRouteRegistryMiddlewareAnalysis shows how to analyze middleware usage across routes
func TestRouteRegistryMiddlewareAnalysis(t *testing.T) {
	// Reset the global registry for this test
	originalRegistry := DefaultRouteRegistry
	defer func() {
		DefaultRouteRegistry = originalRegistry
	}()
	DefaultRouteRegistry = NewInMemoryRouteRegistry()

	// Register some example routes
	simulateGeneratedRouteRegistration()

	// Analyze middleware usage
	middlewareUsage := make(map[string]int)
	allRoutes := GetRoutes()
	
	for _, route := range allRoutes {
		for _, middleware := range route.Middlewares {
			middlewareUsage[middleware]++
		}
	}

	fmt.Println("=== Middleware Usage Analysis ===")
	for middleware, count := range middlewareUsage {
		fmt.Printf("%s: used in %d routes\n", middleware, count)
	}

	// Verify expected middleware usage
	assert.Equal(t, 2, middlewareUsage["Auth"])     // Used in 2 routes
	assert.Equal(t, 2, middlewareUsage["Logging"])  // Used in 2 routes  
	assert.Equal(t, 1, middlewareUsage["Validation"]) // Used in 1 route

	// Find routes without any middleware
	unprotectedRoutes := []RouteInfo{}
	for _, route := range allRoutes {
		if len(route.Middlewares) == 0 {
			unprotectedRoutes = append(unprotectedRoutes, route)
		}
	}

	fmt.Printf("\nUnprotected routes (no middleware): %d\n", len(unprotectedRoutes))
	for _, route := range unprotectedRoutes {
		fmt.Printf("  %s %s\n", route.Method, route.Path)
	}

	assert.Len(t, unprotectedRoutes, 1) // Only health check should be unprotected
	assert.Equal(t, "/health", unprotectedRoutes[0].Path)
}