package adapters

import (
	"bytes"
	"net/http"
	"strings"
	"testing"

	"github.com/toyz/axon/pkg/axon"
)

func TestFiberAdapter_BasicFunctionality(t *testing.T) {
	// Create Fiber adapter
	adapter := NewDefaultFiberAdapter()

	// Test basic properties
	if adapter.Name() != "Fiber" {
		t.Errorf("Expected adapter name 'Fiber', got '%s'", adapter.Name())
	}

	// Test handler registration
	handler := func(ctx axon.RequestContext) error {
		return ctx.Response().JSON(200, map[string]string{"message": "hello"})
	}

	adapter.RegisterRoute("GET", axon.NewAxonPath("/test"), handler)

	// Create test request
	req, _ := http.NewRequest("GET", "/test", nil)
	resp, err := adapter.app.Test(req, -1)
	if err != nil {
		t.Fatalf("Failed to execute request: %v", err)
	}
	defer resp.Body.Close()

	// Verify response
	if resp.StatusCode != 200 {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	// Read response body
	buf := new(bytes.Buffer)
	buf.ReadFrom(resp.Body)
	body := strings.TrimSpace(buf.String())

	expectedBody := `{"message":"hello"}`
	if body != expectedBody {
		t.Errorf("Expected body '%s', got '%s'", expectedBody, body)
	}
}

func TestFiberAdapter_Middleware(t *testing.T) {
	adapter := NewDefaultFiberAdapter()

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

	req, _ := http.NewRequest("GET", "/middleware-test", nil)
	resp, err := adapter.app.Test(req, -1)
	if err != nil {
		t.Fatalf("Failed to execute request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	if !middlewareCalled {
		t.Error("Expected middleware to be called")
	}

	buf := new(bytes.Buffer)
	buf.ReadFrom(resp.Body)
	body := strings.TrimSpace(buf.String())

	expectedBody := `{"middleware":"executed"}`
	if body != expectedBody {
		t.Errorf("Expected body '%s', got '%s'", expectedBody, body)
	}
}

func TestFiberAdapter_RouteGroup(t *testing.T) {
	adapter := NewDefaultFiberAdapter()

	handler := func(ctx axon.RequestContext) error {
		return ctx.Response().JSON(200, map[string]string{"message": "api endpoint"})
	}

	// Create route group
	apiGroup := adapter.RegisterGroup("/api")
	apiGroup.RegisterRoute("GET", axon.NewAxonPath("/users"), handler)

	req, _ := http.NewRequest("GET", "/api/users", nil)
	resp, err := adapter.app.Test(req, -1)
	if err != nil {
		t.Fatalf("Failed to execute request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	buf := new(bytes.Buffer)
	buf.ReadFrom(resp.Body)
	body := strings.TrimSpace(buf.String())

	expectedBody := `{"message":"api endpoint"}`
	if body != expectedBody {
		t.Errorf("Expected body '%s', got '%s'", expectedBody, body)
	}
}

func TestFiberAdapter_ParameterBinding(t *testing.T) {
	adapter := NewDefaultFiberAdapter()

	handler := func(ctx axon.RequestContext) error {
		id := ctx.Param("id")
		return ctx.Response().JSON(200, map[string]string{"id": id})
	}

	// Test Axon path format with parameter
	adapter.RegisterRoute("GET", axon.NewAxonPath("/users/{id}"), handler)

	req, _ := http.NewRequest("GET", "/users/123", nil)
	resp, err := adapter.app.Test(req, -1)
	if err != nil {
		t.Fatalf("Failed to execute request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	buf := new(bytes.Buffer)
	buf.ReadFrom(resp.Body)
	body := strings.TrimSpace(buf.String())

	expectedBody := `{"id":"123"}`
	if body != expectedBody {
		t.Errorf("Expected body '%s', got '%s'", expectedBody, body)
	}
}

func TestFiberAdapter_QueryParameters(t *testing.T) {
	adapter := NewDefaultFiberAdapter()

	handler := func(ctx axon.RequestContext) error {
		search := ctx.QueryParam("q")
		limit := ctx.QueryParam("limit")
		return ctx.Response().JSON(200, map[string]string{
			"search": search,
			"limit":  limit,
		})
	}

	adapter.RegisterRoute("GET", axon.NewAxonPath("/search"), handler)

	req, _ := http.NewRequest("GET", "/search?q=test&limit=10", nil)
	resp, err := adapter.app.Test(req, -1)
	if err != nil {
		t.Fatalf("Failed to execute request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	buf := new(bytes.Buffer)
	buf.ReadFrom(resp.Body)
	body := strings.TrimSpace(buf.String())

	// Note: Fiber might order JSON keys differently
	if !strings.Contains(body, `"search":"test"`) {
		t.Errorf("Expected search parameter in response body, got '%s'", body)
	}
	if !strings.Contains(body, `"limit":"10"`) {
		t.Errorf("Expected limit parameter in response body, got '%s'", body)
	}
}

func TestFiberAdapter_ContextStorage(t *testing.T) {
	adapter := NewDefaultFiberAdapter()

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

	req, _ := http.NewRequest("GET", "/context-test", nil)
	resp, err := adapter.app.Test(req, -1)
	if err != nil {
		t.Fatalf("Failed to execute request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	buf := new(bytes.Buffer)
	buf.ReadFrom(resp.Body)
	body := buf.String()

	if !strings.Contains(body, `"user_id":"12345"`) {
		t.Errorf("Expected user_id in response body, got '%s'", body)
	}
	if !strings.Contains(body, `"session":"active"`) {
		t.Errorf("Expected session in response body, got '%s'", body)
	}
}

func TestFiberAdapter_ErrorHandling(t *testing.T) {
	adapter := NewDefaultFiberAdapter()

	handler := func(ctx axon.RequestContext) error {
		return axon.NewHTTPError(500, "internal server error")
	}

	adapter.RegisterRoute("GET", axon.NewAxonPath("/error-test"), handler)

	req, _ := http.NewRequest("GET", "/error-test", nil)
	resp, err := adapter.app.Test(req, -1)
	if err != nil {
		t.Fatalf("Failed to execute request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 500 {
		t.Errorf("Expected status 500, got %d", resp.StatusCode)
	}

	buf := new(bytes.Buffer)
	buf.ReadFrom(resp.Body)
	body := strings.TrimSpace(buf.String())

	if !strings.Contains(body, "internal server error") {
		t.Errorf("Expected error message in response body, got '%s'", body)
	}
}

func TestFiberAdapter_WildcardPath(t *testing.T) {
	adapter := NewDefaultFiberAdapter()

	handler := func(ctx axon.RequestContext) error {
		// In Fiber, wildcard is accessible via "*" parameter
		path := ctx.Param("*")
		return ctx.Response().JSON(200, map[string]string{"path": path})
	}

	// Test Axon wildcard path format
	adapter.RegisterRoute("GET", axon.NewAxonPath("/files/{*}"), handler)

	req, _ := http.NewRequest("GET", "/files/documents/readme.txt", nil)
	resp, err := adapter.app.Test(req, -1)
	if err != nil {
		t.Fatalf("Failed to execute request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	buf := new(bytes.Buffer)
	buf.ReadFrom(resp.Body)
	body := strings.TrimSpace(buf.String())

	expectedBody := `{"path":"documents/readme.txt"}`
	if body != expectedBody {
		t.Errorf("Expected body '%s', got '%s'", expectedBody, body)
	}
}