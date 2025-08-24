package parser

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/toyz/axon/pkg/axon"
)

// TestParserContextAccess tests that parsers can access echo.Context properly
func TestParserContextAccess(t *testing.T) {
	e := echo.New()
	
	// Mock parser that uses context to access request headers
	contextUsingParser := func(c echo.Context, paramValue string) (string, error) {
		// Access request headers through context
		userAgent := c.Request().Header.Get("User-Agent")
		if userAgent == "" {
			return "", fmt.Errorf("missing User-Agent header")
		}
		
		// Access query parameters through context
		version := c.QueryParam("version")
		if version == "" {
			version = "v1"
		}
		
		// Validate parameter based on context
		if paramValue == "context-dependent" && version != "v2" {
			return "", fmt.Errorf("parameter 'context-dependent' requires version=v2")
		}
		
		return fmt.Sprintf("%s_%s_%s", paramValue, version, userAgent), nil
	}
	
	// Handler that uses context-aware parser
	contextHandler := func(c echo.Context) error {
		parsed, err := contextUsingParser(c, c.Param("param"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Invalid param: %v", err))
		}
		
		return c.JSON(http.StatusOK, map[string]interface{}{
			"parsed_value": parsed,
		})
	}

	e.GET("/context-test/:param", contextHandler)

	t.Run("ParserAccessesContextSuccessfully", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/context-test/test-value?version=v1", nil)
		req.Header.Set("User-Agent", "test-agent")
		rec := httptest.NewRecorder()
		
		e.ServeHTTP(rec, req)
		
		assert.Equal(t, http.StatusOK, rec.Code)
		
		var response map[string]interface{}
		err := json.Unmarshal(rec.Body.Bytes(), &response)
		require.NoError(t, err)
		
		assert.Equal(t, "test-value_v1_test-agent", response["parsed_value"])
	})

	t.Run("ParserFailsWhenContextMissingData", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/context-test/test-value", nil)
		// Missing User-Agent header
		rec := httptest.NewRecorder()
		
		e.ServeHTTP(rec, req)
		
		assert.Equal(t, http.StatusBadRequest, rec.Code)
		
		var response map[string]interface{}
		err := json.Unmarshal(rec.Body.Bytes(), &response)
		require.NoError(t, err)
		
		assert.Contains(t, response["message"], "missing User-Agent header")
	})

	t.Run("ParserValidatesBasedOnContext", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/context-test/context-dependent?version=v1", nil)
		req.Header.Set("User-Agent", "test-agent")
		rec := httptest.NewRecorder()
		
		e.ServeHTTP(rec, req)
		
		assert.Equal(t, http.StatusBadRequest, rec.Code)
		
		var response map[string]interface{}
		err := json.Unmarshal(rec.Body.Bytes(), &response)
		require.NoError(t, err)
		
		assert.Contains(t, response["message"], "requires version=v2")
	})

	t.Run("ParserSucceedsWithCorrectContext", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/context-test/context-dependent?version=v2", nil)
		req.Header.Set("User-Agent", "test-agent")
		rec := httptest.NewRecorder()
		
		e.ServeHTTP(rec, req)
		
		assert.Equal(t, http.StatusOK, rec.Code)
		
		var response map[string]interface{}
		err := json.Unmarshal(rec.Body.Bytes(), &response)
		require.NoError(t, err)
		
		assert.Equal(t, "context-dependent_v2_test-agent", response["parsed_value"])
	})
}

// TestParserPerformanceWithMiddleware tests that parser integration doesn't significantly impact performance
func TestParserPerformanceWithMiddleware(t *testing.T) {
	e := echo.New()
	
	// Simple timing middleware
	timingMiddleware := func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			start := time.Now()
			err := next(c)
			duration := time.Since(start)
			c.Response().Header().Set("X-Response-Time", duration.String())
			return err
		}
	}
	
	// Mock parsers for performance testing
	parseProductCode := func(c echo.Context, paramValue string) (string, error) {
		if len(paramValue) != 10 || paramValue[:5] != "PROD-" {
			return "", fmt.Errorf("invalid format")
		}
		return paramValue, nil
	}
	
	parseDateRange := func(c echo.Context, paramValue string) (map[string]string, error) {
		parts := []string{"2024-01-01", "2024-12-31"} // Mock parsing
		if paramValue == "invalid" {
			return nil, fmt.Errorf("invalid date range")
		}
		return map[string]string{"start": parts[0], "end": parts[1]}, nil
	}
	
	// Handler with multiple parsers
	performanceHandler := func(c echo.Context) error {
		// Parse multiple parameters
		id, err := axon.ParseUUID(c, c.Param("id"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Invalid id")
		}
		
		code, err := parseProductCode(c, c.Param("code"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Invalid code")
		}
		
		dateRange, err := parseDateRange(c, c.Param("dateRange"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Invalid dateRange")
		}
		
		return c.JSON(http.StatusOK, map[string]interface{}{
			"id":         id.String(),
			"code":       code,
			"date_range": fmt.Sprintf("%s to %s", dateRange["start"], dateRange["end"]),
		})
	}

	chainedHandler := timingMiddleware(performanceHandler)
	e.GET("/perf/:id/:code/:dateRange", chainedHandler)

	t.Run("ParserPerformanceIsAcceptable", func(t *testing.T) {
		validUUID := uuid.New().String()
		url := fmt.Sprintf("/perf/%s/PROD-12345/2024-01-01_2024-12-31", validUUID)
		
		// Run multiple requests to get average performance
		var totalDuration time.Duration
		iterations := 10
		
		for i := 0; i < iterations; i++ {
			req := httptest.NewRequest(http.MethodGet, url, nil)
			rec := httptest.NewRecorder()
			
			start := time.Now()
			e.ServeHTTP(rec, req)
			duration := time.Since(start)
			totalDuration += duration
			
			assert.Equal(t, http.StatusOK, rec.Code)
		}
		
		avgDuration := totalDuration / time.Duration(iterations)
		
		// Parser integration should not add significant overhead
		// This is a reasonable threshold for unit tests
		assert.Less(t, avgDuration, 10*time.Millisecond, 
			"Parser integration should not add significant overhead. Average duration: %v", avgDuration)
		
		t.Logf("Average request duration with parsers and middleware: %v", avgDuration)
	})
}

// TestParserWithPassContextFlag tests the -PassContext flag functionality
func TestParserWithPassContextFlag(t *testing.T) {
	e := echo.New()
	
	// Mock parser that simulates context-dependent validation
	contextDependentParser := func(c echo.Context, paramValue string) (string, error) {
		// Simulate parser that needs context for validation
		requestID := c.Request().Header.Get("X-Request-ID")
		if requestID == "" {
			return "", fmt.Errorf("missing request ID in context")
		}
		
		// Simulate validation that depends on context
		if paramValue == "admin-only" {
			userRole := c.Request().Header.Get("X-User-Role")
			if userRole != "admin" {
				return "", fmt.Errorf("admin-only parameter requires admin role")
			}
		}
		
		return fmt.Sprintf("%s_%s", paramValue, requestID), nil
	}
	
	// Handler that simulates -PassContext functionality
	passContextHandler := func(c echo.Context) error {
		// Parse parameter using context-dependent parser
		parsed, err := contextDependentParser(c, c.Param("param"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Invalid param: %v", err))
		}
		
		// Simulate controller method that also receives context
		// This would be: func (ctrl *Controller) HandleRequest(ctx echo.Context, param string) error
		
		// Controller can access context directly
		method := c.Request().Method
		path := c.Request().URL.Path
		
		// Set response using context
		return c.JSON(http.StatusOK, map[string]interface{}{
			"parsed_param": parsed,
			"method":       method,
			"path":         path,
			"context_available": true,
		})
	}

	e.GET("/pass-context/:param", passContextHandler)
	e.POST("/pass-context/:param", passContextHandler)

	t.Run("PassContextWithValidHeaders", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/pass-context/test-param", nil)
		req.Header.Set("X-Request-ID", "req-123")
		req.Header.Set("X-User-Role", "user")
		rec := httptest.NewRecorder()
		
		e.ServeHTTP(rec, req)
		
		assert.Equal(t, http.StatusOK, rec.Code)
		
		var response map[string]interface{}
		err := json.Unmarshal(rec.Body.Bytes(), &response)
		require.NoError(t, err)
		
		assert.Equal(t, "test-param_req-123", response["parsed_param"])
		assert.Equal(t, "GET", response["method"])
		assert.Equal(t, "/pass-context/test-param", response["path"])
		assert.Equal(t, true, response["context_available"])
	})

	t.Run("PassContextMissingRequiredHeader", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/pass-context/test-param", nil)
		// Missing X-Request-ID header
		rec := httptest.NewRecorder()
		
		e.ServeHTTP(rec, req)
		
		assert.Equal(t, http.StatusBadRequest, rec.Code)
		
		var response map[string]interface{}
		err := json.Unmarshal(rec.Body.Bytes(), &response)
		require.NoError(t, err)
		
		assert.Contains(t, response["message"], "missing request ID")
	})

	t.Run("PassContextAdminOnlyParameter", func(t *testing.T) {
		// Test with non-admin user
		req := httptest.NewRequest(http.MethodGet, "/pass-context/admin-only", nil)
		req.Header.Set("X-Request-ID", "req-456")
		req.Header.Set("X-User-Role", "user")
		rec := httptest.NewRecorder()
		
		e.ServeHTTP(rec, req)
		
		assert.Equal(t, http.StatusBadRequest, rec.Code)
		
		var response map[string]interface{}
		err := json.Unmarshal(rec.Body.Bytes(), &response)
		require.NoError(t, err)
		
		assert.Contains(t, response["message"], "requires admin role")
	})

	t.Run("PassContextAdminOnlyParameterWithAdmin", func(t *testing.T) {
		// Test with admin user
		req := httptest.NewRequest(http.MethodGet, "/pass-context/admin-only", nil)
		req.Header.Set("X-Request-ID", "req-789")
		req.Header.Set("X-User-Role", "admin")
		rec := httptest.NewRecorder()
		
		e.ServeHTTP(rec, req)
		
		assert.Equal(t, http.StatusOK, rec.Code)
		
		var response map[string]interface{}
		err := json.Unmarshal(rec.Body.Bytes(), &response)
		require.NoError(t, err)
		
		assert.Equal(t, "admin-only_req-789", response["parsed_param"])
		assert.Equal(t, true, response["context_available"])
	})
}

// TestParserMiddlewareErrorPropagation tests that parser errors propagate correctly through middleware
func TestParserMiddlewareErrorPropagation(t *testing.T) {
	e := echo.New()
	
	var middlewareEvents []string
	
	// Middleware that tracks error handling
	errorTrackingMiddleware := func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			middlewareEvents = append(middlewareEvents, "middleware_start")
			
			err := next(c)
			
			if err != nil {
				middlewareEvents = append(middlewareEvents, "middleware_error_caught")
				// Middleware can modify error response
				if httpErr, ok := err.(*echo.HTTPError); ok {
					if httpErr.Code == http.StatusBadRequest {
						middlewareEvents = append(middlewareEvents, "middleware_bad_request_handled")
					}
				}
			} else {
				middlewareEvents = append(middlewareEvents, "middleware_success")
			}
			
			middlewareEvents = append(middlewareEvents, "middleware_end")
			return err
		}
	}
	
	// Handler with parser that can fail
	parserHandler := func(c echo.Context) error {
		middlewareEvents = append(middlewareEvents, "handler_start")
		
		// Mock parser that validates UUID format
		idParam := c.Param("id")
		if _, err := uuid.Parse(idParam); err != nil {
			middlewareEvents = append(middlewareEvents, "parser_failed")
			return echo.NewHTTPError(http.StatusBadRequest, "Invalid UUID format")
		}
		
		middlewareEvents = append(middlewareEvents, "parser_success")
		middlewareEvents = append(middlewareEvents, "handler_success")
		
		return c.JSON(http.StatusOK, map[string]interface{}{
			"id": idParam,
		})
	}

	chainedHandler := errorTrackingMiddleware(parserHandler)
	e.GET("/items/:id", chainedHandler)

	t.Run("ParserErrorPropagatesCorrectly", func(t *testing.T) {
		middlewareEvents = nil
		
		req := httptest.NewRequest(http.MethodGet, "/items/invalid-uuid", nil)
		rec := httptest.NewRecorder()
		
		e.ServeHTTP(rec, req)
		
		assert.Equal(t, http.StatusBadRequest, rec.Code)
		
		expectedEvents := []string{
			"middleware_start",
			"handler_start", 
			"parser_failed",
			"middleware_error_caught",
			"middleware_bad_request_handled",
			"middleware_end",
		}
		assert.Equal(t, expectedEvents, middlewareEvents)
	})

	t.Run("ParserSuccessFlowsCorrectly", func(t *testing.T) {
		middlewareEvents = nil
		
		validUUID := uuid.New().String()
		req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/items/%s", validUUID), nil)
		rec := httptest.NewRecorder()
		
		e.ServeHTTP(rec, req)
		
		assert.Equal(t, http.StatusOK, rec.Code)
		
		expectedEvents := []string{
			"middleware_start",
			"handler_start",
			"parser_success",
			"handler_success",
			"middleware_success",
			"middleware_end",
		}
		assert.Equal(t, expectedEvents, middlewareEvents)
	})
}