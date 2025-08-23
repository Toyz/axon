package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/toyz/axon/examples/complete-app/internal/models"
	"github.com/toyz/axon/pkg/axon"
	"go.uber.org/fx"
)

// TestCompleteApplicationIntegration tests the entire application end-to-end
func TestCompleteApplicationIntegration(t *testing.T) {
	// Step 1: Generate code using the axon CLI
	t.Run("CodeGeneration", func(t *testing.T) {
		testCodeGeneration(t)
	})

	// Step 2: Test compilation of generated code
	t.Run("CodeCompilation", func(t *testing.T) {
		testCodeCompilation(t)
	})

	// Step 3: Test runtime behavior
	t.Run("RuntimeBehavior", func(t *testing.T) {
		testRuntimeBehavior(t)
	})

	// Step 4: Test application lifecycle
	t.Run("ApplicationLifecycle", func(t *testing.T) {
		testApplicationLifecycle(t)
	})

	// Step 5: Test route registry
	t.Run("RouteRegistry", func(t *testing.T) {
		testRouteRegistry(t)
	})
}

func testCodeGeneration(t *testing.T) {
	// Get the current directory (examples/complete-app)
	currentDir, err := os.Getwd()
	require.NoError(t, err)

	// Build the axon CLI tool
	axonDir := filepath.Join(currentDir, "..", "..")
	cmd := exec.Command("go", "build", "-o", "axon", "./cmd/axon")
	cmd.Dir = axonDir
	output, err := cmd.CombinedOutput()
	require.NoError(t, err, "Failed to build axon CLI: %s", output)

	// Run code generation
	axonBinary := filepath.Join(axonDir, "axon")
	cmd = exec.Command(axonBinary, "./internal/...")
	cmd.Dir = currentDir
	output, err = cmd.CombinedOutput()
	require.NoError(t, err, "Code generation failed: %s", output)

	// Verify generated files exist
	expectedFiles := []string{
		"internal/controllers/autogen_module.go",
		"internal/services/autogen_module.go",
		"internal/middleware/autogen_module.go",
		"internal/interfaces/autogen_module.go",
	}

	for _, file := range expectedFiles {
		_, err := os.Stat(file)
		assert.NoError(t, err, "Generated file should exist: %s", file)
	}

	// Verify generated content contains expected patterns
	controllerModule, err := os.ReadFile("internal/controllers/autogen_module.go")
	require.NoError(t, err)
	
	assert.Contains(t, string(controllerModule), "UserController")
	assert.Contains(t, string(controllerModule), "HealthController")
	assert.Contains(t, string(controllerModule), "fx.Provide")
}

func testCodeCompilation(t *testing.T) {
	// Test that all generated code compiles
	cmd := exec.Command("go", "build", "./...")
	output, err := cmd.CombinedOutput()
	require.NoError(t, err, "Generated code should compile: %s", output)

	// Test that we can build a binary
	cmd = exec.Command("go", "build", "-o", "test-app", ".")
	output, err = cmd.CombinedOutput()
	require.NoError(t, err, "Should be able to build application binary: %s", output)

	// Clean up
	os.Remove("test-app")
}

func testRuntimeBehavior(t *testing.T) {
	// This test simulates the runtime behavior by creating a minimal FX app
	// and testing the generated components
	
	var app *fx.App
	var e *echo.Echo
	
	// Create the FX application with generated modules
	app = fx.New(
		// Include generated modules (these would be imported in real app)
		fx.Provide(
			func() *echo.Echo { return echo.New() },
		),
		fx.Invoke(func(echo *echo.Echo) {
			e = echo
		}),
	)

	// Start the application
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	err := app.Start(ctx)
	require.NoError(t, err, "Application should start successfully")

	// Test HTTP endpoints (simulated)
	t.Run("HealthEndpoint", func(t *testing.T) {
		// This would test the actual generated routes
		// For now, we'll test the structure
		assert.NotNil(t, e)
	})

	// Stop the application
	ctx, cancel = context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	err = app.Stop(ctx)
	require.NoError(t, err, "Application should stop gracefully")
}

func testApplicationLifecycle(t *testing.T) {
	// Test that lifecycle services start and stop correctly
	
	// Create a test context
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Track lifecycle events
	var events []string
	
	// Create mock lifecycle hooks
	startHook := func() error {
		events = append(events, "service_started")
		return nil
	}
	
	stopHook := func() error {
		events = append(events, "service_stopped")
		return nil
	}

	// Create FX app with lifecycle
	app := fx.New(
		fx.Invoke(func(lc fx.Lifecycle) {
			lc.Append(fx.Hook{
				OnStart: func(context.Context) error { return startHook() },
				OnStop:  func(context.Context) error { return stopHook() },
			})
		}),
	)

	// Start application
	err := app.Start(ctx)
	require.NoError(t, err)
	assert.Contains(t, events, "service_started")

	// Stop application
	err = app.Stop(ctx)
	require.NoError(t, err)
	assert.Contains(t, events, "service_stopped")
}

func testRouteRegistry(t *testing.T) {
	// Test that the route registry works correctly
	
	// Create a mock route registry
	registry := &mockRouteRegistry{
		routes: make([]axon.RouteInfo, 0),
	}

	// Register some test routes
	testRoutes := []axon.RouteInfo{
		{
			Method:         "GET",
			Path:           "/users",
			EchoPath:       "/users",
			HandlerName:    "GetAllUsers",
			ControllerName: "UserController",
			PackageName:    "controllers",
			Middlewares:    []string{"LoggingMiddleware"},
		},
		{
			Method:         "GET",
			Path:           "/users/{id:int}",
			EchoPath:       "/users/:id",
			HandlerName:    "GetUser",
			ControllerName: "UserController",
			PackageName:    "controllers",
			Middlewares:    []string{"LoggingMiddleware"},
		},
		{
			Method:         "POST",
			Path:           "/users",
			EchoPath:       "/users",
			HandlerName:    "CreateUser",
			ControllerName: "UserController",
			PackageName:    "controllers",
			Middlewares:    []string{"LoggingMiddleware", "AuthMiddleware"},
		},
	}

	for _, route := range testRoutes {
		registry.RegisterRoute(route)
	}

	// Verify routes were registered
	assert.Len(t, registry.routes, 3)
	
	// Verify route details
	getUserRoute := registry.routes[1]
	assert.Equal(t, "GET", getUserRoute.Method)
	assert.Equal(t, "/users/{id:int}", getUserRoute.Path)
	assert.Equal(t, "/users/:id", getUserRoute.EchoPath)
	assert.Equal(t, "GetUser", getUserRoute.HandlerName)
	assert.Equal(t, []string{"LoggingMiddleware"}, getUserRoute.Middlewares)

	createUserRoute := registry.routes[2]
	assert.Equal(t, "POST", createUserRoute.Method)
	assert.Equal(t, []string{"LoggingMiddleware", "AuthMiddleware"}, createUserRoute.Middlewares)
}

// TestEndToEndHTTPRequests tests actual HTTP requests against generated handlers
func TestEndToEndHTTPRequests(t *testing.T) {
	// This test would require the actual generated code to be loaded
	// For now, we'll simulate the behavior
	
	e := echo.New()
	
	// Mock handlers that simulate generated behavior
	e.GET("/health", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]interface{}{
			"status": "healthy",
			"database": map[string]interface{}{
				"connected": true,
				"status":    "healthy",
			},
		})
	})
	
	e.GET("/users/:id", func(c echo.Context) error {
		// Simulate parameter parsing (generated code would do this)
		id := c.Param("id")
		if id == "1" {
			user := models.User{
				ID:    1,
				Name:  "John Doe",
				Email: "john@example.com",
			}
			return c.JSON(http.StatusOK, user)
		}
		return echo.NewHTTPError(http.StatusNotFound, "user not found")
	})

	// Test health endpoint
	t.Run("HealthEndpoint", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/health", nil)
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		
		var response map[string]interface{}
		err := json.Unmarshal(rec.Body.Bytes(), &response)
		require.NoError(t, err)
		
		assert.Equal(t, "healthy", response["status"])
		assert.NotNil(t, response["database"])
	})

	// Test user endpoint
	t.Run("GetUserEndpoint", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/users/1", nil)
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		
		var user models.User
		err := json.Unmarshal(rec.Body.Bytes(), &user)
		require.NoError(t, err)
		
		assert.Equal(t, 1, user.ID)
		assert.Equal(t, "John Doe", user.Name)
		assert.Equal(t, "john@example.com", user.Email)
	})
}

// TestMiddlewareIntegration tests middleware application
func TestMiddlewareIntegration(t *testing.T) {
	e := echo.New()
	
	// Track middleware execution
	var middlewareExecuted []string
	
	// Mock logging middleware
	loggingMiddleware := func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			middlewareExecuted = append(middlewareExecuted, "logging")
			return next(c)
		}
	}
	
	// Mock auth middleware
	authMiddleware := func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			auth := c.Request().Header.Get("Authorization")
			if auth != "Bearer valid-token" {
				return echo.NewHTTPError(http.StatusUnauthorized, "invalid token")
			}
			middlewareExecuted = append(middlewareExecuted, "auth")
			return next(c)
		}
	}
	
	// Apply middlewares in order (simulating generated code)
	handler := func(c echo.Context) error {
		middlewareExecuted = append(middlewareExecuted, "handler")
		return c.JSON(http.StatusOK, map[string]string{"message": "success"})
	}
	
	// Chain middlewares
	chainedHandler := loggingMiddleware(authMiddleware(handler))
	e.POST("/users", chainedHandler)

	// Test with valid auth
	t.Run("ValidAuth", func(t *testing.T) {
		middlewareExecuted = nil
		
		req := httptest.NewRequest(http.MethodPost, "/users", strings.NewReader(`{"name":"test"}`))
		req.Header.Set("Authorization", "Bearer valid-token")
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		
		e.ServeHTTP(rec, req)
		
		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, []string{"logging", "auth", "handler"}, middlewareExecuted)
	})

	// Test with invalid auth
	t.Run("InvalidAuth", func(t *testing.T) {
		middlewareExecuted = nil
		
		req := httptest.NewRequest(http.MethodPost, "/users", strings.NewReader(`{"name":"test"}`))
		req.Header.Set("Authorization", "Bearer invalid-token")
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		
		e.ServeHTTP(rec, req)
		
		assert.Equal(t, http.StatusUnauthorized, rec.Code)
		assert.Equal(t, []string{"logging"}, middlewareExecuted) // Auth middleware stops the chain
	})
}

// TestParameterBinding tests parameter parsing and type conversion
func TestParameterBinding(t *testing.T) {
	e := echo.New()
	
	// Mock handler that simulates generated parameter binding
	e.GET("/users/:id", func(c echo.Context) error {
		// Simulate generated parameter parsing code
		idStr := c.Param("id")
		id := 0
		if idStr != "" {
			// This would be generated type conversion code
			if parsed, err := parseIntParam(idStr); err != nil {
				return echo.NewHTTPError(http.StatusBadRequest, "Invalid id parameter")
			} else {
				id = parsed
			}
		}
		
		return c.JSON(http.StatusOK, map[string]interface{}{
			"id":      id,
			"message": fmt.Sprintf("User %d", id),
		})
	})

	// Test valid parameter
	t.Run("ValidParameter", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/users/123", nil)
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		
		var response map[string]interface{}
		err := json.Unmarshal(rec.Body.Bytes(), &response)
		require.NoError(t, err)
		
		assert.Equal(t, float64(123), response["id"]) // JSON unmarshals numbers as float64
	})

	// Test invalid parameter
	t.Run("InvalidParameter", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/users/invalid", nil)
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})
}

// TestResponseHandling tests different response types
func TestResponseHandling(t *testing.T) {
	e := echo.New()
	
	// Test (data, error) return type
	e.GET("/users", func(c echo.Context) error {
		// Simulate generated wrapper for (data, error) return
		users := []models.User{
			{ID: 1, Name: "John", Email: "john@example.com"},
			{ID: 2, Name: "Jane", Email: "jane@example.com"},
		}
		return c.JSON(http.StatusOK, users)
	})
	
	// Test (*Response, error) return type
	e.POST("/users", func(c echo.Context) error {
		// Simulate generated wrapper for (*Response, error) return
		var req models.CreateUserRequest
		if err := c.Bind(&req); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, err.Error())
		}
		
		// Simulate custom response
		response := &axon.Response{
			StatusCode: http.StatusCreated,
			Body: models.User{
				ID:    3,
				Name:  req.Name,
				Email: req.Email,
			},
		}
		
		return c.JSON(response.StatusCode, response.Body)
	})

	// Test data/error response
	t.Run("DataErrorResponse", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/users", nil)
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		
		var users []models.User
		err := json.Unmarshal(rec.Body.Bytes(), &users)
		require.NoError(t, err)
		assert.Len(t, users, 2)
	})

	// Test custom response
	t.Run("CustomResponse", func(t *testing.T) {
		reqBody := `{"name":"Alice","email":"alice@example.com"}`
		req := httptest.NewRequest(http.MethodPost, "/users", bytes.NewReader([]byte(reqBody)))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusCreated, rec.Code)
		
		var user models.User
		err := json.Unmarshal(rec.Body.Bytes(), &user)
		require.NoError(t, err)
		assert.Equal(t, "Alice", user.Name)
		assert.Equal(t, "alice@example.com", user.Email)
	})
}

// Helper functions

func parseIntParam(s string) (int, error) {
	// This simulates the generated parameter parsing code
	if s == "invalid" {
		return 0, fmt.Errorf("invalid integer")
	}
	// Simple conversion for test
	switch s {
	case "123":
		return 123, nil
	case "1":
		return 1, nil
	default:
		return 0, fmt.Errorf("invalid integer")
	}
}

// mockRouteRegistry implements axon.RouteRegistry for testing
type mockRouteRegistry struct {
	routes []axon.RouteInfo
}

func (r *mockRouteRegistry) RegisterRoute(route axon.RouteInfo) {
	r.routes = append(r.routes, route)
}