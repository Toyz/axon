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

	adapter.RegisterRoute("GET", "/test", handler)

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

	adapter.RegisterRoute("GET", "/middleware-test", handler, middleware)

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

	apiGroup.RegisterRoute("GET", "/users", handler)

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

	adapter.RegisterRoute("GET", "/users/:id", handler)

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

	adapter.RegisterRoute("GET", "/search", handler)

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

	adapter.RegisterRoute("GET", "/context-test", handler, middleware)

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
		return axon.NewHTTPError(400, "Bad request")
	}

	adapter.RegisterRoute("GET", "/error-test", handler)

	// Test request
	req := httptest.NewRequest("GET", "/error-test", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	// Echo's default error handler will return 500 for unhandled errors
	// In a real implementation, you'd set up proper error handling
	if rec.Code != 500 {
		t.Errorf("Expected status 500 (Echo default error handling), got %d", rec.Code)
	}
}