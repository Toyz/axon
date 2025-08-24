package templates

import (
	"fmt"
	"testing"

	"github.com/toyz/axon/internal/models"
	"github.com/toyz/axon/internal/registry"
)

// ExampleGenerateRouteWrapper demonstrates how to use the route wrapper generation
func ExampleGenerateRouteWrapper() {
	// Example 1: Simple GET route with data-error return
	route1 := models.RouteMetadata{
		Method:      "GET",
		Path:        "/users/{id:int}",
		HandlerName: "GetUser",
		Parameters: []models.Parameter{
			{Name: "id", Type: "int", Source: models.ParameterSourcePath, Required: true},
		},
		ReturnType: models.ReturnTypeInfo{
			Type:     models.ReturnTypeDataError,
			DataType: "User",
			HasError: true,
		},
	}

	registry := registry.NewParserRegistry()
	wrapper1, _ := GenerateRouteWrapper(route1, "UserController", registry)
	fmt.Println("Generated wrapper for GET /users/{id:int}:")
	fmt.Println(wrapper1)

	// Example 2: POST route with body and custom response
	route2 := models.RouteMetadata{
		Method:      "POST",
		Path:        "/users",
		HandlerName: "CreateUser",
		Parameters: []models.Parameter{
			{Name: "user", Type: "CreateUserRequest", Source: models.ParameterSourceBody, Required: true},
		},
		ReturnType: models.ReturnTypeInfo{
			Type:         models.ReturnTypeResponseError,
			UsesResponse: true,
			HasError:     true,
		},
	}

	wrapper2, _ := GenerateRouteWrapper(route2, "UserController", registry)
	fmt.Println("\nGenerated wrapper for POST /users:")
	fmt.Println(wrapper2)
}

func TestExampleUsage(t *testing.T) {
	// This test demonstrates the complete usage of the response generation system
	
	// Create a route with all supported features
	route := models.RouteMetadata{
		Method:      "PUT",
		Path:        "/users/{userId:int}/posts/{postId:int}",
		HandlerName: "UpdateUserPost",
		Parameters: []models.Parameter{
			{Name: "userId", Type: "int", Source: models.ParameterSourcePath, Required: true},
			{Name: "postId", Type: "int", Source: models.ParameterSourcePath, Required: true},
			{Name: "post", Type: "UpdatePostRequest", Source: models.ParameterSourceBody, Required: true},
		},
		ReturnType: models.ReturnTypeInfo{
			Type:         models.ReturnTypeResponseError,
			UsesResponse: true,
			HasError:     true,
		},
		Flags: []string{"SomeFlag"}, // Not PassContext to test different scenarios
	}

	// Generate the wrapper
	registry := registry.NewParserRegistry()
	wrapper, err := GenerateRouteWrapper(route, "UserController", registry)
	if err != nil {
		t.Fatalf("Failed to generate wrapper: %v", err)
	}

	// Verify the wrapper contains expected elements
	expectedElements := []string{
		"func wrapUserControllerUpdateUserPost(handler *UserController) echo.HandlerFunc",
		"userId, err := strconv.Atoi(c.Param(\"userId\"))",
		"postId, err := strconv.Atoi(c.Param(\"postId\"))",
		"var body UpdatePostRequest",
		"response, err := handler.UpdateUserPost(userId, postId, body)",
		"return c.JSON(response.StatusCode, response.Body)",
	}

	for _, element := range expectedElements {
		if !contains(wrapper, element) {
			t.Errorf("Generated wrapper missing expected element: %s\n\nGenerated wrapper:\n%s", element, wrapper)
		}
	}

	t.Logf("Successfully generated wrapper:\n%s", wrapper)
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 || 
		(len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || 
		containsAt(s, substr))))
}

func containsAt(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}