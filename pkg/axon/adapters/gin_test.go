package adapters

import (
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/toyz/axon/pkg/axon"
)

func init() {
	// Set Gin to test mode to reduce noise in test output
	gin.SetMode(gin.TestMode)
}

func TestGinAdapter_BasicFunctionality(t *testing.T) {
	// Create Gin adapter
	adapter := NewDefaultGinAdapter()

	// Test basic properties
	if adapter.Name() != "Gin" {
		t.Errorf("Expected adapter name 'Gin', got '%s'", adapter.Name())
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
	adapter.engine.ServeHTTP(rec, req)

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

func TestGinAdapter_Middleware(t *testing.T) {
	adapter := NewDefaultGinAdapter()

	var middlewareCalled bool
	middleware := func(next axon.HandlerFunc) axon.HandlerFunc {
		return func(ctx axon.RequestContext) error {
			middlewareCalled = true
			ctx.Set("middleware", "executed")
			return next(ctx)
		}
	}

	handler := func(ctx axon.RequestContext) error {
		middlewareValue := ctx.Get("middleware")
		return ctx.Response().JSON(200, map[string]interface{}{
			"middleware": middlewareValue,
		})
	}

	adapter.RegisterRoute("GET", axon.NewAxonPath("/middleware-test"), handler, middleware)

	req := httptest.NewRequest("GET", "/middleware-test", nil)
	rec := httptest.NewRecorder()

	adapter.engine.ServeHTTP(rec, req)

	if rec.Code != 200 {
		t.Errorf("Expected status 200, got %d", rec.Code)
	}

	if !middlewareCalled {
		t.Error("Expected middleware to be called")
	}

	expectedBody := `{"middleware":"executed"}`
	body := strings.TrimSpace(rec.Body.String())
	if body != expectedBody {
		t.Errorf("Expected body '%s', got '%s'", expectedBody, body)
	}
}

func TestGinAdapter_RouteGroup(t *testing.T) {
	adapter := NewDefaultGinAdapter()

	handler := func(ctx axon.RequestContext) error {
		return ctx.Response().JSON(200, map[string]string{"message": "api endpoint"})
	}

	// Create route group
	apiGroup := adapter.RegisterGroup("/api")
	apiGroup.RegisterRoute("GET", axon.NewAxonPath("/users"), handler)

	req := httptest.NewRequest("GET", "/api/users", nil)
	rec := httptest.NewRecorder()

	adapter.engine.ServeHTTP(rec, req)

	if rec.Code != 200 {
		t.Errorf("Expected status 200, got %d", rec.Code)
	}

	expectedBody := `{"message":"api endpoint"}`
	body := strings.TrimSpace(rec.Body.String())
	if body != expectedBody {
		t.Errorf("Expected body '%s', got '%s'", expectedBody, body)
	}
}

func TestGinAdapter_ParameterBinding(t *testing.T) {
	adapter := NewDefaultGinAdapter()

	handler := func(ctx axon.RequestContext) error {
		id := ctx.Param("id")
		return ctx.Response().JSON(200, map[string]string{"id": id})
	}

	// Test Axon path format with parameter
	adapter.RegisterRoute("GET", axon.NewAxonPath("/users/{id}"), handler)

	req := httptest.NewRequest("GET", "/users/123", nil)
	rec := httptest.NewRecorder()

	adapter.engine.ServeHTTP(rec, req)

	if rec.Code != 200 {
		t.Errorf("Expected status 200, got %d", rec.Code)
	}

	expectedBody := `{"id":"123"}`
	body := strings.TrimSpace(rec.Body.String())
	if body != expectedBody {
		t.Errorf("Expected body '%s', got '%s'", expectedBody, body)
	}
}

func TestGinAdapter_QueryParameters(t *testing.T) {
	adapter := NewDefaultGinAdapter()

	handler := func(ctx axon.RequestContext) error {
		search := ctx.QueryParam("q")
		limit := ctx.QueryParam("limit")
		return ctx.Response().JSON(200, map[string]string{
			"search": search,
			"limit":  limit,
		})
	}

	adapter.RegisterRoute("GET", axon.NewAxonPath("/search"), handler)

	req := httptest.NewRequest("GET", "/search?q=test&limit=10", nil)
	rec := httptest.NewRecorder()

	adapter.engine.ServeHTTP(rec, req)

	if rec.Code != 200 {
		t.Errorf("Expected status 200, got %d", rec.Code)
	}

	expectedBody := `{"limit":"10","search":"test"}`
	body := strings.TrimSpace(rec.Body.String())
	if body != expectedBody {
		t.Errorf("Expected body '%s', got '%s'", expectedBody, body)
	}
}

func TestGinAdapter_ContextStorage(t *testing.T) {
	adapter := NewDefaultGinAdapter()

	middleware := func(next axon.HandlerFunc) axon.HandlerFunc {
		return func(ctx axon.RequestContext) error {
			ctx.Set("user_id", "12345")
			ctx.Set("session", "active")
			return next(ctx)
		}
	}

	handler := func(ctx axon.RequestContext) error {
		userID := ctx.Get("user_id")
		session := ctx.Get("session")
		return ctx.Response().JSON(200, map[string]interface{}{
			"user_id": userID,
			"session": session,
		})
	}

	adapter.RegisterRoute("GET", axon.NewAxonPath("/context-test"), handler, middleware)

	req := httptest.NewRequest("GET", "/context-test", nil)
	rec := httptest.NewRecorder()

	adapter.engine.ServeHTTP(rec, req)

	if rec.Code != 200 {
		t.Errorf("Expected status 200, got %d", rec.Code)
	}

	body := rec.Body.String()
	if !strings.Contains(body, `"user_id":"12345"`) {
		t.Errorf("Expected user_id in response body, got '%s'", body)
	}
	if !strings.Contains(body, `"session":"active"`) {
		t.Errorf("Expected session in response body, got '%s'", body)
	}
}

func TestGinAdapter_ErrorHandling(t *testing.T) {
	adapter := NewDefaultGinAdapter()

	handler := func(ctx axon.RequestContext) error {
		return axon.NewHTTPError(500, "internal server error")
	}

	adapter.RegisterRoute("GET", axon.NewAxonPath("/error-test"), handler)

	req := httptest.NewRequest("GET", "/error-test", nil)
	rec := httptest.NewRecorder()

	adapter.engine.ServeHTTP(rec, req)

	if rec.Code != 500 {
		t.Errorf("Expected status 500, got %d", rec.Code)
	}

	body := strings.TrimSpace(rec.Body.String())
	if !strings.Contains(body, "internal server error") {
		t.Errorf("Expected error message in response body, got '%s'", body)
	}
}

func TestGinAdapter_WildcardPath(t *testing.T) {
	adapter := NewDefaultGinAdapter()

	handler := func(ctx axon.RequestContext) error {
		// In Gin, wildcard is accessible via "path" parameter
		path := ctx.Param("path")
		return ctx.Response().JSON(200, map[string]string{"path": path})
	}

	// Test Axon wildcard path format
	adapter.RegisterRoute("GET", axon.NewAxonPath("/files/{*}"), handler)

	req := httptest.NewRequest("GET", "/files/documents/readme.txt", nil)
	rec := httptest.NewRecorder()

	adapter.engine.ServeHTTP(rec, req)

	if rec.Code != 200 {
		t.Errorf("Expected status 200, got %d", rec.Code)
	}

	expectedBody := `{"path":"/documents/readme.txt"}`
	body := strings.TrimSpace(rec.Body.String())
	if body != expectedBody {
		t.Errorf("Expected body '%s', got '%s'", expectedBody, body)
	}
}