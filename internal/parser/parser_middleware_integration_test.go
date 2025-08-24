package parser

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/toyz/axon/pkg/axon"
)

// TestParserMiddlewareIntegration tests that custom parsers work correctly with middleware
func TestParserMiddlewareIntegration(t *testing.T) {
	e := echo.New()
	
	// Track middleware execution order
	var executionOrder []string
	
	// Mock logging middleware that tracks execution
	loggingMiddleware := func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			executionOrder = append(executionOrder, "logging_start")
			err := next(c)
			executionOrder = append(executionOrder, "logging_end")
			return err
		}
	}
	
	// Mock auth middleware that validates tokens
	authMiddleware := func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			executionOrder = append(executionOrder, "auth_start")
			auth := c.Request().Header.Get("Authorization")
			if auth != "Bearer valid-token" {
				executionOrder = append(executionOrder, "auth_rejected")
				return echo.NewHTTPError(http.StatusUnauthorized, "invalid token")
			}
			c.Set("user_id", 123)
			executionOrder = append(executionOrder, "auth_passed")
			err := next(c)
			executionOrder = append(executionOrder, "auth_end")
			return err
		}
	}

	// Mock custom parser for testing
	parseProductCode := func(c echo.Context, paramValue string) (string, error) {
		if !strings.HasPrefix(paramValue, "PROD-") || len(paramValue) != 10 {
			return "", fmt.Errorf("invalid product code format")
		}
		return strings.ToUpper(paramValue), nil
	}

	// Handler that simulates generated code with custom parser
	productHandler := func(c echo.Context) error {
		executionOrder = append(executionOrder, "parser_start")
		
		// Simulate generated parser call
		code, err := parseProductCode(c, c.Param("code"))
		if err != nil {
			executionOrder = append(executionOrder, "parser_failed")
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Invalid code: %v", err))
		}
		
		executionOrder = append(executionOrder, "parser_success")
		executionOrder = append(executionOrder, "handler_executed")
		
		// Verify auth middleware set user context
		userID := c.Get("user_id")
		
		return c.JSON(http.StatusOK, map[string]interface{}{
			"code":    code,
			"user_id": userID,
			"message": "Product found",
		})
	}

	// Chain middlewares with handler (simulating generated code)
	chainedHandler := loggingMiddleware(authMiddleware(productHandler))
	e.GET("/products/by-code/:code", chainedHandler)

	t.Run("ValidParserWithMiddleware", func(t *testing.T) {
		executionOrder = nil
		
		req := httptest.NewRequest(http.MethodGet, "/products/by-code/PROD-12345", nil)
		req.Header.Set("Authorization", "Bearer valid-token")
		rec := httptest.NewRecorder()
		
		e.ServeHTTP(rec, req)
		
		assert.Equal(t, http.StatusOK, rec.Code)
		
		var response map[string]interface{}
		err := json.Unmarshal(rec.Body.Bytes(), &response)
		require.NoError(t, err)
		
		assert.Equal(t, "PROD-12345", response["code"])
		assert.Equal(t, float64(123), response["user_id"])
		
		// Verify execution order: middleware -> parser -> handler -> middleware cleanup
		expectedOrder := []string{
			"logging_start", "auth_start", "auth_passed", 
			"parser_start", "parser_success", "handler_executed",
			"auth_end", "logging_end",
		}
		assert.Equal(t, expectedOrder, executionOrder)
	})

	t.Run("InvalidParserWithMiddleware", func(t *testing.T) {
		executionOrder = nil
		
		req := httptest.NewRequest(http.MethodGet, "/products/by-code/INVALID", nil)
		req.Header.Set("Authorization", "Bearer valid-token")
		rec := httptest.NewRecorder()
		
		e.ServeHTTP(rec, req)
		
		assert.Equal(t, http.StatusBadRequest, rec.Code)
		
		// Verify middleware ran but parser failed before handler
		expectedOrder := []string{
			"logging_start", "auth_start", "auth_passed",
			"parser_start", "parser_failed",
			"auth_end", "logging_end",
		}
		assert.Equal(t, expectedOrder, executionOrder)
	})

	t.Run("AuthFailureBeforeParser", func(t *testing.T) {
		executionOrder = nil
		
		req := httptest.NewRequest(http.MethodGet, "/products/by-code/PROD-12345", nil)
		req.Header.Set("Authorization", "Bearer invalid-token")
		rec := httptest.NewRecorder()
		
		e.ServeHTTP(rec, req)
		
		assert.Equal(t, http.StatusUnauthorized, rec.Code)
		
		// Verify parser never executed due to auth failure
		expectedOrder := []string{
			"logging_start", "auth_start", "auth_rejected", "logging_end",
		}
		assert.Equal(t, expectedOrder, executionOrder)
		
		// Ensure parser-related execution steps are not present
		assert.NotContains(t, executionOrder, "parser_start")
		assert.NotContains(t, executionOrder, "parser_success")
		assert.NotContains(t, executionOrder, "handler_executed")
	})
}

// TestParserPassContextIntegration tests parser integration with -PassContext flag
func TestParserPassContextIntegration(t *testing.T) {
	e := echo.New()
	
	// Mock date range parser for testing
	parseDateRange := func(c echo.Context, paramValue string) (map[string]string, error) {
		parts := strings.Split(paramValue, "_")
		if len(parts) != 2 {
			return nil, fmt.Errorf("date range must be in format 'YYYY-MM-DD_YYYY-MM-DD'")
		}
		
		// Simple validation for test
		if parts[0] == "invalid" || parts[1] == "invalid" {
			return nil, fmt.Errorf("invalid date format")
		}
		
		return map[string]string{
			"start": parts[0],
			"end":   parts[1],
		}, nil
	}
	
	// Mock parser that uses echo.Context (simulating -PassContext functionality)
	contextAwareHandler := func(c echo.Context) error {
		// Simulate generated code for route with -PassContext and custom parser
		
		// Parse custom parameter
		dateRange, err := parseDateRange(c, c.Param("dateRange"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Invalid dateRange: %v", err))
		}
		
		// Simulate controller method that receives both context and parsed parameter
		// This simulates: func (c *Controller) GetSales(ctx echo.Context, dateRange DateRange) error
		
		// Set some context values to verify context is passed through
		c.Set("request_id", "test-123")
		c.Set("parsed_date_range", dateRange)
		
		// Simulate controller logic that uses context
		requestID := c.Get("request_id")
		
		return c.JSON(http.StatusOK, map[string]interface{}{
			"date_range":   fmt.Sprintf("%s to %s", dateRange["start"], dateRange["end"]),
			"request_id":   requestID,
			"context_type": fmt.Sprintf("%T", c),
		})
	}

	e.GET("/sales/:dateRange", contextAwareHandler)

	t.Run("PassContextWithValidParser", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/sales/2024-01-01_2024-12-31", nil)
		rec := httptest.NewRecorder()
		
		e.ServeHTTP(rec, req)
		
		assert.Equal(t, http.StatusOK, rec.Code)
		
		var response map[string]interface{}
		err := json.Unmarshal(rec.Body.Bytes(), &response)
		require.NoError(t, err)
		
		assert.Equal(t, "2024-01-01 to 2024-12-31", response["date_range"])
		assert.Equal(t, "test-123", response["request_id"])
		assert.Contains(t, response["context_type"], "echo")
	})

	t.Run("PassContextWithInvalidParser", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/sales/invalid-date", nil)
		rec := httptest.NewRecorder()
		
		e.ServeHTTP(rec, req)
		
		assert.Equal(t, http.StatusBadRequest, rec.Code)
		
		var response map[string]interface{}
		err := json.Unmarshal(rec.Body.Bytes(), &response)
		require.NoError(t, err)
		
		assert.Contains(t, response["message"], "Invalid dateRange")
	})
}

// TestParserErrorHandlingWithMiddleware tests that parser errors are properly handled by middleware
func TestParserErrorHandlingWithMiddleware(t *testing.T) {
	e := echo.New()
	
	var errorsCaught []string
	
	// Error handling middleware that catches and logs errors
	errorMiddleware := func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			err := next(c)
			if err != nil {
				if httpErr, ok := err.(*echo.HTTPError); ok {
					errorsCaught = append(errorsCaught, fmt.Sprintf("HTTP_%d: %v", httpErr.Code, httpErr.Message))
				} else {
					errorsCaught = append(errorsCaught, fmt.Sprintf("Error: %v", err))
				}
			}
			return err
		}
	}

	// Handler with multiple parsers
	multiParserHandler := func(c echo.Context) error {
		// Parse UUID parameter
		id, err := axon.ParseUUID(c, c.Param("id"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Invalid id: must be a valid UUID")
		}
		
		// Parse custom ProductCode parameter (mock)
		codeParam := c.Param("code")
		if !strings.HasPrefix(codeParam, "PROD-") || len(codeParam) != 10 {
			return echo.NewHTTPError(http.StatusBadRequest, "Invalid code: must be PROD-XXXXX format")
		}
		
		return c.JSON(http.StatusOK, map[string]interface{}{
			"id":   id.String(),
			"code": codeParam,
		})
	}

	chainedHandler := errorMiddleware(multiParserHandler)
	e.GET("/products/:id/:code", chainedHandler)

	t.Run("FirstParserFailsSecondNotCalled", func(t *testing.T) {
		errorsCaught = nil
		
		req := httptest.NewRequest(http.MethodGet, "/products/invalid-uuid/PROD-12345", nil)
		rec := httptest.NewRecorder()
		
		e.ServeHTTP(rec, req)
		
		assert.Equal(t, http.StatusBadRequest, rec.Code)
		assert.Len(t, errorsCaught, 1)
		assert.Contains(t, errorsCaught[0], "HTTP_400")
		assert.Contains(t, errorsCaught[0], "Invalid id: must be a valid UUID")
	})

	t.Run("FirstParserSucceedsSecondFails", func(t *testing.T) {
		errorsCaught = nil
		
		validUUID := uuid.New().String()
		req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/products/%s/INVALID", validUUID), nil)
		rec := httptest.NewRecorder()
		
		e.ServeHTTP(rec, req)
		
		assert.Equal(t, http.StatusBadRequest, rec.Code)
		assert.Len(t, errorsCaught, 1)
		assert.Contains(t, errorsCaught[0], "HTTP_400")
		assert.Contains(t, errorsCaught[0], "Invalid code")
	})

	t.Run("BothParsersSucceed", func(t *testing.T) {
		errorsCaught = nil
		
		validUUID := uuid.New().String()
		req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/products/%s/PROD-12345", validUUID), nil)
		rec := httptest.NewRecorder()
		
		e.ServeHTTP(rec, req)
		
		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Len(t, errorsCaught, 0)
		
		var response map[string]interface{}
		err := json.Unmarshal(rec.Body.Bytes(), &response)
		require.NoError(t, err)
		
		assert.Equal(t, validUUID, response["id"])
		assert.Equal(t, "PROD-12345", response["code"])
	})
}

// TestMixedParameterTypes tests routes with both built-in and custom parsers
func TestMixedParameterTypes(t *testing.T) {
	e := echo.New()
	
	// Handler that uses multiple parameter types
	mixedParamsHandler := func(c echo.Context) error {
		// Built-in UUID parser
		categoryId, err := axon.ParseUUID(c, c.Param("categoryId"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Invalid categoryId: must be a valid UUID")
		}
		
		// Built-in int parser (simulated)
		pageStr := c.QueryParam("page")
		page := 1
		if pageStr != "" {
			if pageStr == "invalid" {
				return echo.NewHTTPError(http.StatusBadRequest, "Invalid page: must be an integer")
			}
			// Simple conversion for test
			if pageStr == "2" {
				page = 2
			}
		}
		
		// Custom ProductCode parser (mock)
		codeParam := c.Param("code")
		if !strings.HasPrefix(codeParam, "PROD-") || len(codeParam) != 10 {
			return echo.NewHTTPError(http.StatusBadRequest, "Invalid code: must be PROD-XXXXX format")
		}
		
		// Custom DateRange parser (mock)
		dateRangeParam := c.Param("dateRange")
		parts := strings.Split(dateRangeParam, "_")
		if len(parts) != 2 {
			return echo.NewHTTPError(http.StatusBadRequest, "Invalid dateRange: must be YYYY-MM-DD_YYYY-MM-DD format")
		}
		
		return c.JSON(http.StatusOK, map[string]interface{}{
			"category_id": categoryId.String(),
			"page":        page,
			"code":        codeParam,
			"date_range":  fmt.Sprintf("%s to %s", parts[0], parts[1]),
		})
	}

	e.GET("/categories/:categoryId/products/:code/sales/:dateRange", mixedParamsHandler)

	t.Run("AllParametersValid", func(t *testing.T) {
		validUUID := uuid.New().String()
		url := fmt.Sprintf("/categories/%s/products/PROD-12345/sales/2024-01-01_2024-12-31?page=2", validUUID)
		
		req := httptest.NewRequest(http.MethodGet, url, nil)
		rec := httptest.NewRecorder()
		
		e.ServeHTTP(rec, req)
		
		assert.Equal(t, http.StatusOK, rec.Code)
		
		var response map[string]interface{}
		err := json.Unmarshal(rec.Body.Bytes(), &response)
		require.NoError(t, err)
		
		assert.Equal(t, validUUID, response["category_id"])
		assert.Equal(t, float64(2), response["page"])
		assert.Equal(t, "PROD-12345", response["code"])
		assert.Equal(t, "2024-01-01 to 2024-12-31", response["date_range"])
	})

	t.Run("UUIDParameterInvalid", func(t *testing.T) {
		url := "/categories/invalid-uuid/products/PROD-12345/sales/2024-01-01_2024-12-31"
		
		req := httptest.NewRequest(http.MethodGet, url, nil)
		rec := httptest.NewRecorder()
		
		e.ServeHTTP(rec, req)
		
		assert.Equal(t, http.StatusBadRequest, rec.Code)
		
		var response map[string]interface{}
		err := json.Unmarshal(rec.Body.Bytes(), &response)
		require.NoError(t, err)
		
		assert.Contains(t, response["message"], "Invalid categoryId")
	})

	t.Run("CustomCodeParameterInvalid", func(t *testing.T) {
		validUUID := uuid.New().String()
		url := fmt.Sprintf("/categories/%s/products/INVALID/sales/2024-01-01_2024-12-31", validUUID)
		
		req := httptest.NewRequest(http.MethodGet, url, nil)
		rec := httptest.NewRecorder()
		
		e.ServeHTTP(rec, req)
		
		assert.Equal(t, http.StatusBadRequest, rec.Code)
		
		var response map[string]interface{}
		err := json.Unmarshal(rec.Body.Bytes(), &response)
		require.NoError(t, err)
		
		assert.Contains(t, response["message"], "Invalid code")
	})

	t.Run("CustomDateRangeParameterInvalid", func(t *testing.T) {
		validUUID := uuid.New().String()
		url := fmt.Sprintf("/categories/%s/products/PROD-12345/sales/invalid-date", validUUID)
		
		req := httptest.NewRequest(http.MethodGet, url, nil)
		rec := httptest.NewRecorder()
		
		e.ServeHTTP(rec, req)
		
		assert.Equal(t, http.StatusBadRequest, rec.Code)
		
		var response map[string]interface{}
		err := json.Unmarshal(rec.Body.Bytes(), &response)
		require.NoError(t, err)
		
		assert.Contains(t, response["message"], "Invalid dateRange")
	})

	t.Run("QueryParameterInvalid", func(t *testing.T) {
		validUUID := uuid.New().String()
		url := fmt.Sprintf("/categories/%s/products/PROD-12345/sales/2024-01-01_2024-12-31?page=invalid", validUUID)
		
		req := httptest.NewRequest(http.MethodGet, url, nil)
		rec := httptest.NewRecorder()
		
		e.ServeHTTP(rec, req)
		
		assert.Equal(t, http.StatusBadRequest, rec.Code)
		
		var response map[string]interface{}
		err := json.Unmarshal(rec.Body.Bytes(), &response)
		require.NoError(t, err)
		
		assert.Contains(t, response["message"], "Invalid page")
	})
}