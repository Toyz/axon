package adapters

import (
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/toyz/axon/pkg/axon"
)

func TestEchoAdapter_BasicFunctionality(t *testing.T) {
	// Create Echo instance and adapter
	e := echo.New()
	adapter := NewEchoAdapter(e)

	// Test basic properties
	if adapter.Name() != "Echo" {
		t.Errorf("Expected adapter name 'Echo', got '%s'", adapter.Name())
	}

	// Test handler registration
	handler := func(ctx axon.RequestContext) error {
		return ctx.Response().JSON(200, map[string]string{"message": "hello"})
	}

	adapter.RegisterRoute("GET", axon.NewAxonPath("/test"), handler)

	// Create test request
	req := httptest.NewRequest("GET", "/test", nil)
	rec := httptest.NewRecorder()

	// Execute request
	e.ServeHTTP(rec, req)

	// Verify response
	if rec.Code != 200 {
		t.Errorf("Expected status 200, got %d", rec.Code)
	}

	expectedBody := `{"message":"hello"}`
	body := strings.TrimSpace(rec.Body.String())
	if body != expectedBody {
		t.Errorf("Expected body '%s', got '%s'", expectedBody, body)
	}
}

func TestEchoAdapter_Middleware(t *testing.T) {
	e := echo.New()
	adapter := NewEchoAdapter(e)

	// Create middleware that adds a header
	middleware := func(next axon.HandlerFunc) axon.HandlerFunc {
		return func(ctx axon.RequestContext) error {
			ctx.Response().SetHeader("X-Test", "middleware-works")
			return next(ctx)
		}
	}

	// Register handler with middleware
	handler := func(ctx axon.RequestContext) error {
		return ctx.Response().JSON(200, map[string]string{"test": "success"})
	}

	adapter.RegisterRoute("GET", axon.NewAxonPath("/middleware-test"), handler, middleware)

	// Test request
	req := httptest.NewRequest("GET", "/middleware-test", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	// Verify middleware header was set
	if rec.Header().Get("X-Test") != "middleware-works" {
		t.Errorf("Expected middleware header 'middleware-works', got '%s'", rec.Header().Get("X-Test"))
	}
}

func TestEchoAdapter_RouteGroup(t *testing.T) {
	e := echo.New()
	adapter := NewEchoAdapter(e)

	// Create route group
	apiGroup := adapter.RegisterGroup("/api")

	// Register route in group
	handler := func(ctx axon.RequestContext) error {
		return ctx.Response().JSON(200, map[string]string{"group": "api"})
	}

	apiGroup.RegisterRoute("GET", axon.NewAxonPath("/users"), handler)

	// Test request to group route
	req := httptest.NewRequest("GET", "/api/users", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != 200 {
		t.Errorf("Expected status 200, got %d", rec.Code)
	}
}

func TestEchoAdapter_ParameterBinding(t *testing.T) {
	e := echo.New()
	adapter := NewEchoAdapter(e)

	// Handler that uses path parameters
	handler := func(ctx axon.RequestContext) error {
		id := ctx.Param("id")
		return ctx.Response().JSON(200, map[string]string{"id": id})
	}

	adapter.RegisterRoute("GET", axon.NewAxonPath("/users/{id}"), handler)

	// Test request with parameter
	req := httptest.NewRequest("GET", "/users/123", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != 200 {
		t.Errorf("Expected status 200, got %d", rec.Code)
	}

	expectedBody := `{"id":"123"}`
	body := strings.TrimSpace(rec.Body.String())
	if body != expectedBody {
		t.Errorf("Expected body '%s', got '%s'", expectedBody, body)
	}
}

func TestEchoAdapter_QueryParameters(t *testing.T) {
	e := echo.New()
	adapter := NewEchoAdapter(e)

	// Handler that uses query parameters
	handler := func(ctx axon.RequestContext) error {
		name := ctx.QueryParam("name")
		return ctx.Response().JSON(200, map[string]string{"name": name})
	}

	adapter.RegisterRoute("GET", axon.NewAxonPath("/search"), handler)

	// Test request with query parameter
	req := httptest.NewRequest("GET", "/search?name=john", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != 200 {
		t.Errorf("Expected status 200, got %d", rec.Code)
	}

	expectedBody := `{"name":"john"}`
	body := strings.TrimSpace(rec.Body.String())
	if body != expectedBody {
		t.Errorf("Expected body '%s', got '%s'", expectedBody, body)
	}
}

func TestEchoAdapter_ContextStorage(t *testing.T) {
	e := echo.New()
	adapter := NewEchoAdapter(e)

	// Middleware that sets context value
	middleware := func(next axon.HandlerFunc) axon.HandlerFunc {
		return func(ctx axon.RequestContext) error {
			ctx.Set("user", "test-user")
			return next(ctx)
		}
	}

	// Handler that reads context value
	handler := func(ctx axon.RequestContext) error {
		user := ctx.Get("user").(string)
		return ctx.Response().JSON(200, map[string]string{"user": user})
	}

	adapter.RegisterRoute("GET", axon.NewAxonPath("/context-test"), handler, middleware)

	// Test request
	req := httptest.NewRequest("GET", "/context-test", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != 200 {
		t.Errorf("Expected status 200, got %d", rec.Code)
	}

	expectedBody := `{"user":"test-user"}`
	body := strings.TrimSpace(rec.Body.String())
	if body != expectedBody {
		t.Errorf("Expected body '%s', got '%s'", expectedBody, body)
	}
}

func TestEchoAdapter_ErrorHandling(t *testing.T) {
	e := echo.New()
	adapter := NewEchoAdapter(e)

	// Handler that returns an error
	handler := func(ctx axon.RequestContext) error {
		return axon.NewHTTPError(500, "internal server error")
	}

	adapter.RegisterRoute("GET", axon.NewAxonPath("/error-test"), handler)

	// Test request
	req := httptest.NewRequest("GET", "/error-test", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	// Should preserve the original HTTP error code
	if rec.Code != 500 {
		t.Errorf("Expected status 500 from HTTPError, got %d", rec.Code)
	}
}

func TestEchoAdapter_MiddlewareErrorHandling(t *testing.T) {
	e := echo.New()
	adapter := NewEchoAdapter(e)

	// Middleware that returns an HTTPError
	authMiddleware := func(next axon.HandlerFunc) axon.HandlerFunc {
		return func(ctx axon.RequestContext) error {
			// Simulate auth failure
			return axon.NewHTTPError(401, "unauthorized")
		}
	}

	// Handler should not be reached
	handler := func(ctx axon.RequestContext) error {
		return ctx.Response().JSON(200, map[string]string{"message": "success"})
	}

	adapter.RegisterRoute("POST", axon.NewAxonPath("/protected"), handler, authMiddleware)

	req := httptest.NewRequest("POST", "/protected", nil)
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	// Should return 401 from middleware
	if rec.Code != 401 {
		t.Errorf("Expected status 401 from middleware, got %d", rec.Code)
	}

	body := strings.TrimSpace(rec.Body.String())
	if !strings.Contains(body, "unauthorized") {
		t.Errorf("Expected 'unauthorized' message in response body, got '%s'", body)
	}
}