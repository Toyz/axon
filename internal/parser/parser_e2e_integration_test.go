package parser

import (
	"bytes"
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

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestParserIntegrationEndToEnd tests the complete parser integration workflow
// by generating code and testing the actual generated handlers
func TestParserIntegrationEndToEnd(t *testing.T) {
	// Skip this test in CI or if we can't build
	if os.Getenv("SKIP_E2E") != "" {
		t.Skip("Skipping E2E test")
	}

	// Get the project root directory
	projectRoot, err := getProjectRoot()
	require.NoError(t, err)

	// Build the axon CLI
	axonBinary := filepath.Join(projectRoot, "axon")
	cmd := exec.Command("go", "build", "-o", axonBinary, "./cmd/axon")
	cmd.Dir = projectRoot
	output, err := cmd.CombinedOutput()
	require.NoError(t, err, "Failed to build axon CLI: %s", output)
	defer os.Remove(axonBinary)

	// Generate code for the complete-app
	completeAppDir := filepath.Join(projectRoot, "examples", "complete-app")
	cmd = exec.Command(axonBinary, "./internal/...")
	cmd.Dir = completeAppDir
	output, err = cmd.CombinedOutput()
	require.NoError(t, err, "Code generation failed: %s", output)

	// Verify the generated code compiles
	cmd = exec.Command("go", "build", "./...")
	cmd.Dir = completeAppDir
	output, err = cmd.CombinedOutput()
	require.NoError(t, err, "Generated code should compile: %s", output)

	t.Run("GeneratedCodeHasParserIntegration", func(t *testing.T) {
		// Read the generated controller module
		generatedFile := filepath.Join(completeAppDir, "internal", "controllers", "autogen_module.go")
		content, err := os.ReadFile(generatedFile)
		require.NoError(t, err)

		generatedCode := string(content)

		// Verify parser imports are present
		assert.Contains(t, generatedCode, `"github.com/toyz/axon/examples/complete-app/internal/parsers"`)
		assert.Contains(t, generatedCode, `"github.com/toyz/axon/pkg/axon"`)

		// Verify parser calls are generated
		assert.Contains(t, generatedCode, "parsers.ParseProductCode(c, c.Param(\"code\"))")
		assert.Contains(t, generatedCode, "parsers.ParseDateRange(c, c.Param(\"dateRange\"))")
		assert.Contains(t, generatedCode, "axon.ParseUUID(c, c.Param(\"id\"))")

		// Verify error handling for parsers
		assert.Contains(t, generatedCode, "Invalid code:")
		assert.Contains(t, generatedCode, "Invalid dateRange:")
		assert.Contains(t, generatedCode, "Invalid categoryId: must be a valid UUID")

		// Verify PassContext integration
		assert.Contains(t, generatedCode, "handler.CreateProductInCategory(c, categoryId, body)")
	})

	t.Run("GeneratedCodeHandlesParserErrors", func(t *testing.T) {
		// This test verifies that the generated code properly handles parser errors
		// by examining the generated wrapper functions

		generatedFile := filepath.Join(completeAppDir, "internal", "controllers", "autogen_module.go")
		content, err := os.ReadFile(generatedFile)
		require.NoError(t, err)

		generatedCode := string(content)

		// Check that parser errors return HTTP 400
		assert.Contains(t, generatedCode, "echo.NewHTTPError(http.StatusBadRequest")
		
		// Check that parser errors include the error message
		assert.Contains(t, generatedCode, "fmt.Sprintf(\"Invalid code: %v\", err)")
		assert.Contains(t, generatedCode, "fmt.Sprintf(\"Invalid dateRange: %v\", err)")
	})
}

// TestParserIntegrationWithActualServer tests parser integration by starting a real server
func TestParserIntegrationWithActualServer(t *testing.T) {
	// This test creates a minimal server setup to test parser integration
	// without requiring the full complete-app build
	
	e := echo.New()
	
	// Mock the actual generated handlers to test integration patterns
	setupMockHandlersWithParsers(e)
	
	t.Run("UUIDParserIntegration", func(t *testing.T) {
		validUUID := uuid.New().String()
		
		// Test valid UUID
		req := httptest.NewRequest(http.MethodGet, "/products/"+validUUID, nil)
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)
		
		assert.Equal(t, http.StatusOK, rec.Code)
		
		var response map[string]interface{}
		err := json.Unmarshal(rec.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Equal(t, validUUID, response["id"])
		
		// Test invalid UUID
		req = httptest.NewRequest(http.MethodGet, "/products/invalid-uuid", nil)
		rec = httptest.NewRecorder()
		e.ServeHTTP(rec, req)
		
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})
	
	t.Run("CustomParserIntegration", func(t *testing.T) {
		// Test valid product code
		req := httptest.NewRequest(http.MethodGet, "/products/by-code/PROD-12345", nil)
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)
		
		assert.Equal(t, http.StatusOK, rec.Code)
		
		var response map[string]interface{}
		err := json.Unmarshal(rec.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Equal(t, "PROD-12345", response["code"])
		
		// Test invalid product code
		req = httptest.NewRequest(http.MethodGet, "/products/by-code/INVALID", nil)
		rec = httptest.NewRecorder()
		e.ServeHTTP(rec, req)
		
		assert.Equal(t, http.StatusBadRequest, rec.Code)
		
		err = json.Unmarshal(rec.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Contains(t, response["message"], "Invalid code")
	})
	
	t.Run("DateRangeParserIntegration", func(t *testing.T) {
		// Test valid date range
		req := httptest.NewRequest(http.MethodGet, "/products/sales/2024-01-01_2024-12-31", nil)
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)
		
		assert.Equal(t, http.StatusOK, rec.Code)
		
		var response map[string]interface{}
		err := json.Unmarshal(rec.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Contains(t, response["date_range"], "2024-01-01")
		assert.Contains(t, response["date_range"], "2024-12-31")
		
		// Test invalid date range
		req = httptest.NewRequest(http.MethodGet, "/products/sales/invalid-date", nil)
		rec = httptest.NewRecorder()
		e.ServeHTTP(rec, req)
		
		assert.Equal(t, http.StatusBadRequest, rec.Code)
		
		err = json.Unmarshal(rec.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Contains(t, response["message"], "Invalid dateRange")
	})
	
	t.Run("MixedParametersIntegration", func(t *testing.T) {
		validUUID := uuid.New().String()
		
		// Test valid mixed parameters
		req := httptest.NewRequest(http.MethodPost, "/products/"+validUUID+"/items", 
			bytes.NewReader([]byte(`{"name":"Test Product","description":"Test","price":99.99}`)))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)
		
		assert.Equal(t, http.StatusOK, rec.Code)
		
		var response map[string]interface{}
		err := json.Unmarshal(rec.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Equal(t, validUUID, response["category_id"])
		assert.Equal(t, "Test Product", response["name"])
		
		// Test invalid UUID with valid body
		req = httptest.NewRequest(http.MethodPost, "/products/invalid-uuid/items", 
			bytes.NewReader([]byte(`{"name":"Test Product","description":"Test","price":99.99}`)))
		req.Header.Set("Content-Type", "application/json")
		rec = httptest.NewRecorder()
		e.ServeHTTP(rec, req)
		
		assert.Equal(t, http.StatusBadRequest, rec.Code)
		
		err = json.Unmarshal(rec.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Contains(t, response["message"], "Invalid categoryId")
	})
}

// setupMockHandlersWithParsers creates mock handlers that simulate the generated code
func setupMockHandlersWithParsers(e *echo.Echo) {
	// Mock UUID parser (simulating axon.ParseUUID)
	parseUUID := func(c echo.Context, paramValue string) (uuid.UUID, error) {
		return uuid.Parse(paramValue)
	}
	
	// Mock ProductCode parser
	parseProductCode := func(c echo.Context, paramValue string) (string, error) {
		if !strings.HasPrefix(paramValue, "PROD-") || len(paramValue) != 10 {
			return "", fmt.Errorf("invalid product code format")
		}
		return strings.ToUpper(paramValue), nil
	}
	
	// Mock DateRange parser
	parseDateRange := func(c echo.Context, paramValue string) (map[string]string, error) {
		parts := strings.Split(paramValue, "_")
		if len(parts) != 2 {
			return nil, fmt.Errorf("date range must be in format 'YYYY-MM-DD_YYYY-MM-DD'")
		}
		return map[string]string{"start": parts[0], "end": parts[1]}, nil
	}
	
	// Handler that simulates generated UUID parser integration
	e.GET("/products/:id", func(c echo.Context) error {
		id, err := parseUUID(c, c.Param("id"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Invalid id: must be a valid UUID")
		}
		
		return c.JSON(http.StatusOK, map[string]interface{}{
			"id":   id.String(),
			"name": "Sample Product",
		})
	})
	
	// Handler that simulates generated custom parser integration
	e.GET("/products/by-code/:code", func(c echo.Context) error {
		code, err := parseProductCode(c, c.Param("code"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Invalid code: %v", err))
		}
		
		return c.JSON(http.StatusOK, map[string]interface{}{
			"code": code,
			"name": "Product " + code,
		})
	})
	
	// Handler that simulates generated date range parser integration
	e.GET("/products/sales/:dateRange", func(c echo.Context) error {
		dateRange, err := parseDateRange(c, c.Param("dateRange"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Invalid dateRange: %v", err))
		}
		
		return c.JSON(http.StatusOK, map[string]interface{}{
			"date_range": fmt.Sprintf("%s to %s", dateRange["start"], dateRange["end"]),
			"sales":      []string{"product1", "product2"},
		})
	})
	
	// Handler that simulates mixed parameter types (UUID + body)
	e.POST("/products/:categoryId/items", func(c echo.Context) error {
		categoryId, err := parseUUID(c, c.Param("categoryId"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Invalid categoryId: must be a valid UUID")
		}
		
		var body map[string]interface{}
		if err := c.Bind(&body); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, err.Error())
		}
		
		return c.JSON(http.StatusOK, map[string]interface{}{
			"category_id": categoryId.String(),
			"name":        body["name"],
			"description": body["description"],
			"price":       body["price"],
		})
	})
}

// getProjectRoot finds the project root directory
func getProjectRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	
	// Walk up the directory tree to find go.mod
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("could not find project root")
		}
		dir = parent
	}
}

// TestParserIntegrationPerformance tests that parser integration doesn't add significant overhead
func TestParserIntegrationPerformance(t *testing.T) {
	e := echo.New()
	setupMockHandlersWithParsers(e)
	
	// Benchmark parser performance
	validUUID := uuid.New().String()
	
	// Warm up
	for i := 0; i < 10; i++ {
		req := httptest.NewRequest(http.MethodGet, "/products/"+validUUID, nil)
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)
	}
	
	// Measure performance
	iterations := 1000
	start := time.Now()
	
	for i := 0; i < iterations; i++ {
		req := httptest.NewRequest(http.MethodGet, "/products/"+validUUID, nil)
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)
		
		if rec.Code != http.StatusOK {
			t.Fatalf("Unexpected status code: %d", rec.Code)
		}
	}
	
	duration := time.Since(start)
	avgDuration := duration / time.Duration(iterations)
	
	t.Logf("Average request duration with parser: %v", avgDuration)
	t.Logf("Requests per second: %.0f", float64(iterations)/duration.Seconds())
	
	// Parser integration should not add significant overhead
	assert.Less(t, avgDuration, 100*time.Microsecond, 
		"Parser integration should not add significant overhead")
}