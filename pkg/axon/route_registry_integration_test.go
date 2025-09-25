package axon

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

// RouteRegistryIntegrationTestSuite tests the complete route registry functionality
type RouteRegistryIntegrationTestSuite struct {
	suite.Suite
	registry *InMemoryRouteRegistry
}

func (suite *RouteRegistryIntegrationTestSuite) SetupTest() {
	// Create a fresh registry for each test
	suite.registry = NewInMemoryRouteRegistry()
}

func (suite *RouteRegistryIntegrationTestSuite) TestCompleteRouteDiscoveryWorkflow() {
	// Simulate registering routes from different packages and controllers
	routes := []RouteInfo{
		{
			Method:             "GET",
			Path:               "/users/{id:int}",
			EchoPath:           "/users/:id",
			HandlerName:        "GetUser",
			ControllerName:     "UserController",
			PackageName:        "controllers",
			Middlewares:        []string{"Auth", "Logging"},
			ParameterInstances: []ParameterInstance{{Name: "id", Type: "int"}},
			Handler:            func(c RequestContext) error { return nil },
		},
		{
			Method:             "POST",
			Path:               "/users",
			EchoPath:           "/users",
			HandlerName:        "CreateUser",
			ControllerName:     "UserController",
			PackageName:        "controllers",
			Middlewares:        []string{"Auth", "Validation"},
			ParameterInstances: []ParameterInstance{},
			Handler:            func(c RequestContext) error { return nil },
		},
		{
			Method:             "GET",
			Path:               "/posts/{slug:string}",
			EchoPath:           "/posts/:slug",
			HandlerName:        "GetPost",
			ControllerName:     "PostController",
			PackageName:        "controllers",
			Middlewares:        []string{"Logging"},
			ParameterInstances: []ParameterInstance{{Name: "slug", Type: "string"}},
			Handler:            func(c RequestContext) error { return nil },
		},
		{
			Method:             "GET",
			Path:               "/health",
			EchoPath:           "/health",
			HandlerName:        "HealthCheck",
			ControllerName:     "HealthController",
			PackageName:        "health",
			Middlewares:        []string{},
			ParameterInstances: []ParameterInstance{},
			Handler:            func(c RequestContext) error { return nil },
		},
		{
			Method:             "PUT",
			Path:               "/users/{id:int}/profile",
			EchoPath:           "/users/:id/profile",
			HandlerName:        "UpdateProfile",
			ControllerName:     "UserController",
			PackageName:        "controllers",
			Middlewares:        []string{"Auth", "Validation", "RateLimit"},
			ParameterInstances: []ParameterInstance{{Name: "id", Type: "int"}},
			Handler:            func(c RequestContext) error { return nil },
		},
	}

	// Register all routes
	for _, route := range routes {
		suite.registry.RegisterRoute(route)
	}

	// Test GetAllRoutes
	allRoutes := suite.registry.GetAllRoutes()
	assert.Len(suite.T(), allRoutes, 5)

	// Test filtering by package
	controllerRoutes := suite.registry.GetRoutesByPackage("controllers")
	assert.Len(suite.T(), controllerRoutes, 4)

	healthRoutes := suite.registry.GetRoutesByPackage("health")
	assert.Len(suite.T(), healthRoutes, 1)
	assert.Equal(suite.T(), "HealthCheck", healthRoutes[0].HandlerName)

	// Test filtering by controller
	userRoutes := suite.registry.GetRoutesByController("UserController")
	assert.Len(suite.T(), userRoutes, 3)

	postRoutes := suite.registry.GetRoutesByController("PostController")
	assert.Len(suite.T(), postRoutes, 1)
	assert.Equal(suite.T(), "GetPost", postRoutes[0].HandlerName)

	// Test filtering by HTTP method
	getRoutes := suite.registry.GetRoutesByMethod("GET")
	assert.Len(suite.T(), getRoutes, 3)

	postRoutes = suite.registry.GetRoutesByMethod("POST")
	assert.Len(suite.T(), postRoutes, 1)
	assert.Equal(suite.T(), "CreateUser", postRoutes[0].HandlerName)

	putRoutes := suite.registry.GetRoutesByMethod("PUT")
	assert.Len(suite.T(), putRoutes, 1)
	assert.Equal(suite.T(), "UpdateProfile", putRoutes[0].HandlerName)
}

func (suite *RouteRegistryIntegrationTestSuite) TestRouteMetadataIntegrity() {
	// Test that all route metadata is preserved correctly
	route := RouteInfo{
		Method:             "POST",
		Path:               "/api/v1/users/{id:int}/posts/{slug:string}",
		EchoPath:           "/api/v1/users/:id/posts/:slug",
		HandlerName:        "CreateUserPost",
		ControllerName:     "PostController",
		PackageName:        "api.controllers",
		Middlewares:        []string{"Auth", "Validation", "RateLimit", "Logging"},
		ParameterInstances: []ParameterInstance{{Name: "id", Type: "int"}, {Name: "slug", Type: "string"}},
		Handler:            func(c RequestContext) error { return nil },
	}

	suite.registry.RegisterRoute(route)

	retrievedRoutes := suite.registry.GetAllRoutes()
	assert.Len(suite.T(), retrievedRoutes, 1)

	retrieved := retrievedRoutes[0]
	assert.Equal(suite.T(), route.Method, retrieved.Method)
	assert.Equal(suite.T(), route.Path, retrieved.Path)
	assert.Equal(suite.T(), route.EchoPath, retrieved.EchoPath)
	assert.Equal(suite.T(), route.HandlerName, retrieved.HandlerName)
	assert.Equal(suite.T(), route.ControllerName, retrieved.ControllerName)
	assert.Equal(suite.T(), route.PackageName, retrieved.PackageName)
	assert.Equal(suite.T(), route.Middlewares, retrieved.Middlewares)
	assert.Equal(suite.T(), route.ParameterInstances, retrieved.ParameterInstances)
	assert.NotNil(suite.T(), retrieved.Handler)
}

func (suite *RouteRegistryIntegrationTestSuite) TestFilteringEdgeCases() {
	// Test filtering with non-existent values
	nonExistentPackageRoutes := suite.registry.GetRoutesByPackage("nonexistent")
	assert.Len(suite.T(), nonExistentPackageRoutes, 0)

	nonExistentControllerRoutes := suite.registry.GetRoutesByController("NonExistentController")
	assert.Len(suite.T(), nonExistentControllerRoutes, 0)

	nonExistentMethodRoutes := suite.registry.GetRoutesByMethod("PATCH")
	assert.Len(suite.T(), nonExistentMethodRoutes, 0)

	// Test case sensitivity
	route := RouteInfo{
		Method:         "get", // lowercase
		Path:           "/test",
		ControllerName: "testcontroller", // lowercase
		PackageName:    "testpackage",    // lowercase
	}
	suite.registry.RegisterRoute(route)

	// Should match exactly (case sensitive)
	methodRoutes := suite.registry.GetRoutesByMethod("get")
	assert.Len(suite.T(), methodRoutes, 1)

	methodRoutesUpper := suite.registry.GetRoutesByMethod("GET")
	assert.Len(suite.T(), methodRoutesUpper, 0)

	controllerRoutes := suite.registry.GetRoutesByController("testcontroller")
	assert.Len(suite.T(), controllerRoutes, 1)

	packageRoutes := suite.registry.GetRoutesByPackage("testpackage")
	assert.Len(suite.T(), packageRoutes, 1)
}

func (suite *RouteRegistryIntegrationTestSuite) TestRouteRegistryImmutability() {
	// Test that returned slices are copies and don't affect the internal state
	route1 := RouteInfo{Method: "GET", Path: "/test1", HandlerName: "Test1"}
	route2 := RouteInfo{Method: "POST", Path: "/test2", HandlerName: "Test2"}

	suite.registry.RegisterRoute(route1)
	suite.registry.RegisterRoute(route2)

	// Get all routes and modify the returned slice
	allRoutes := suite.registry.GetAllRoutes()
	assert.Len(suite.T(), allRoutes, 2)

	// Modify the returned slice
	allRoutes[0].Method = "MODIFIED"
	allRoutes = append(allRoutes, RouteInfo{Method: "DELETE", Path: "/test3"})

	// Verify internal state is unchanged
	freshRoutes := suite.registry.GetAllRoutes()
	assert.Len(suite.T(), freshRoutes, 2)
	assert.Equal(suite.T(), "GET", freshRoutes[0].Method)
	assert.Equal(suite.T(), "POST", freshRoutes[1].Method)
}

func (suite *RouteRegistryIntegrationTestSuite) TestGlobalRegistryIntegration() {
	// Test integration with global registry functions
	originalRegistry := DefaultRouteRegistry
	defer func() {
		DefaultRouteRegistry = originalRegistry
	}()

	// Replace with test registry
	DefaultRouteRegistry = suite.registry

	route := RouteInfo{
		Method:         "GET",
		Path:           "/global-test",
		ControllerName: "TestController",
		PackageName:    "test",
	}

	// Test global convenience functions
	DefaultRouteRegistry.RegisterRoute(route)

	allRoutes := GetRoutes()
	assert.Len(suite.T(), allRoutes, 1)
	assert.Equal(suite.T(), route.Method, allRoutes[0].Method)

	packageRoutes := GetRoutesByPackage("test")
	assert.Len(suite.T(), packageRoutes, 1)

	controllerRoutes := GetRoutesByController("TestController")
	assert.Len(suite.T(), controllerRoutes, 1)
}

// TestEchoIntegration removed - Echo integration should be tested via adapters package

func (suite *RouteRegistryIntegrationTestSuite) TestMiddlewareInstancesIntegration() {
	// Test that middleware instances are properly stored and retrieved

	// First register some middleware
	authMiddleware := MiddlewareInstance{
		Name:     "Auth",
		Handler:  func(next HandlerFunc) HandlerFunc { return next },
		Instance: "AuthInstance",
	}

	loggingMiddleware := MiddlewareInstance{
		Name:     "Logging",
		Handler:  func(next HandlerFunc) HandlerFunc { return next },
		Instance: "LoggingInstance",
	}

	// Create route with middleware instances
	route := RouteInfo{
		Method:              "GET",
		Path:                "/protected",
		ControllerName:      "TestController",
		PackageName:         "test",
		Middlewares:         []string{"Auth", "Logging"},
		MiddlewareInstances: []MiddlewareInstance{authMiddleware, loggingMiddleware},
	}

	suite.registry.RegisterRoute(route)

	// Retrieve and verify middleware instances
	routes := suite.registry.GetAllRoutes()
	assert.Len(suite.T(), routes, 1)

	retrievedRoute := routes[0]
	assert.Len(suite.T(), retrievedRoute.MiddlewareInstances, 2)

	// Verify middleware instances are preserved
	assert.Equal(suite.T(), "Auth", retrievedRoute.MiddlewareInstances[0].Name)
	assert.Equal(suite.T(), "AuthInstance", retrievedRoute.MiddlewareInstances[0].Instance)
	assert.Equal(suite.T(), "Logging", retrievedRoute.MiddlewareInstances[1].Name)
	assert.Equal(suite.T(), "LoggingInstance", retrievedRoute.MiddlewareInstances[1].Instance)

	// Test GetMiddlewaresByRoute convenience function
	routeMiddlewares := GetMiddlewaresByRoute(retrievedRoute)
	assert.Len(suite.T(), routeMiddlewares, 2)
	assert.Equal(suite.T(), authMiddleware.Name, routeMiddlewares[0].Name)
	assert.Equal(suite.T(), loggingMiddleware.Name, routeMiddlewares[1].Name)
}

func TestRouteRegistryIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(RouteRegistryIntegrationTestSuite))
}

// TestRouteDiscoveryPerformance tests the performance of route discovery operations
func TestRouteDiscoveryPerformance(t *testing.T) {
	registry := NewInMemoryRouteRegistry()

	// Register a large number of routes
	numRoutes := 1000
	for i := 0; i < numRoutes; i++ {
		route := RouteInfo{
			Method:         "GET",
			Path:           fmt.Sprintf("/api/v1/resource%d/{id:int}", i),
			HandlerName:    fmt.Sprintf("GetResource%d", i),
			ControllerName: fmt.Sprintf("Resource%dController", i%10), // 10 different controllers
			PackageName:    fmt.Sprintf("package%d", i%5),             // 5 different packages
			Middlewares:    []string{"Auth", "Logging"},
		}
		registry.RegisterRoute(route)
	}

	// Test performance of different operations
	t.Run("GetAllRoutes", func(t *testing.T) {
		routes := registry.GetAllRoutes()
		assert.Len(t, routes, numRoutes)
	})

	t.Run("GetRoutesByPackage", func(t *testing.T) {
		routes := registry.GetRoutesByPackage("package0")
		assert.Greater(t, len(routes), 0)
	})

	t.Run("GetRoutesByController", func(t *testing.T) {
		routes := registry.GetRoutesByController("Resource0Controller")
		assert.Greater(t, len(routes), 0)
	})

	t.Run("GetRoutesByMethod", func(t *testing.T) {
		routes := registry.GetRoutesByMethod("GET")
		assert.Len(t, routes, numRoutes)
	})
}
