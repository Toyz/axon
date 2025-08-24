package templates

import (
	"go/format"
	"strings"
	"testing"

	"github.com/toyz/axon/internal/models"
)

func TestCompleteRouteWrapperGeneration(t *testing.T) {
	tests := []struct {
		name           string
		route          models.RouteMetadata
		controllerName string
		shouldCompile  bool
	}{
		{
			name: "GET route with int parameter and data-error return",
			route: models.RouteMetadata{
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
			},
			controllerName: "UserController",
			shouldCompile:  true,
		},
		{
			name: "POST route with body parameter and response-error return",
			route: models.RouteMetadata{
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
			},
			controllerName: "UserController",
			shouldCompile:  true,
		},
		{
			name: "DELETE route with context injection and error return",
			route: models.RouteMetadata{
				Method:      "DELETE",
				Path:        "/users/{id:int}",
				HandlerName: "DeleteUser",
				Parameters: []models.Parameter{
					{Name: "id", Type: "int", Source: models.ParameterSourcePath, Required: true},
				},
				ReturnType: models.ReturnTypeInfo{
					Type:     models.ReturnTypeError,
					HasError: true,
				},
				Flags: []string{"PassContext"},
			},
			controllerName: "UserController",
			shouldCompile:  true,
		},
		{
			name: "Complex route with multiple parameters",
			route: models.RouteMetadata{
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
			},
			controllerName: "UserController",
			shouldCompile:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Generate the complete route wrapper
			registry := createTestParserRegistry()
			wrapper, err := GenerateRouteWrapper(tt.route, tt.controllerName, registry)
			if err != nil {
				t.Fatalf("failed to generate route wrapper: %v", err)
			}

			// Verify the wrapper is not empty
			if strings.TrimSpace(wrapper) == "" {
				t.Fatal("generated wrapper is empty")
			}

			// Try to format the generated code to verify it's valid Go syntax
			if tt.shouldCompile {
				// Add necessary imports and package declaration for formatting
				completeCode := `package test

import (
	"net/http"
	"strconv"
	"github.com/labstack/echo/v4"
)

` + wrapper

				_, err := format.Source([]byte(completeCode))
				if err != nil {
					t.Errorf("generated code is not valid Go syntax: %v\n\nGenerated code:\n%s", err, completeCode)
				}
			}

			// Verify specific patterns based on return type
			switch tt.route.ReturnType.Type {
			case models.ReturnTypeDataError:
				if !strings.Contains(wrapper, "return c.JSON(http.StatusOK, data)") {
					t.Error("data-error return type should contain JSON response with 200 OK")
				}
				if !strings.Contains(wrapper, "return echo.NewHTTPError(http.StatusInternalServerError, err.Error())") {
					t.Error("data-error return type should contain 500 error handling")
				}

			case models.ReturnTypeResponseError:
				if !strings.Contains(wrapper, "return c.JSON(response.StatusCode, response.Body)") {
					t.Error("response-error return type should contain custom status code response")
				}
				if !strings.Contains(wrapper, "handler returned nil response") {
					t.Error("response-error return type should check for nil response")
				}

			case models.ReturnTypeError:
				if !strings.Contains(wrapper, "return err") {
					t.Error("error return type should return the error directly")
				}
				if !strings.Contains(wrapper, "return nil") {
					t.Error("error return type should return nil on success")
				}
			}

			// Verify parameter handling
			for _, param := range tt.route.Parameters {
				switch param.Source {
				case models.ParameterSourcePath:
					if param.Type == "int" {
						expectedBinding := `strconv.Atoi(c.Param("` + param.Name + `"))`
						if !strings.Contains(wrapper, expectedBinding) {
							t.Errorf("missing int parameter binding for %s", param.Name)
						}
					} else if param.Type == "string" {
						expectedBinding := `c.Param("` + param.Name + `")`
						if !strings.Contains(wrapper, expectedBinding) {
							t.Errorf("missing string parameter binding for %s", param.Name)
						}
					}

				case models.ParameterSourceBody:
					expectedBinding := `var body ` + param.Type
					if !strings.Contains(wrapper, expectedBinding) {
						t.Errorf("missing body parameter binding for type %s", param.Type)
					}
				}
			}

			// Verify context injection if PassContext flag is present
			if hasPassContextFlag(tt.route.Flags) {
				handlerCall := generateHandlerCall(tt.route, tt.controllerName)
				if !strings.Contains(handlerCall, "c,") && !strings.Contains(handlerCall, "(c)") {
					t.Error("PassContext flag should inject context parameter")
				}
			}
		})
	}
}

func TestResponseHandlingErrorCases(t *testing.T) {
	tests := []struct {
		name        string
		route       models.RouteMetadata
		expectError bool
		errorMsg    string
	}{
		{
			name: "unsupported return type",
			route: models.RouteMetadata{
				HandlerName: "TestHandler",
				ReturnType: models.ReturnTypeInfo{
					Type: models.ReturnType(999), // Invalid return type
				},
			},
			expectError: true,
			errorMsg:    "unsupported return type",
		},
		{
			name: "unsupported parameter type",
			route: models.RouteMetadata{
				HandlerName: "TestHandler",
				Parameters: []models.Parameter{
					{Name: "param", Type: "unsupported", Source: models.ParameterSourcePath},
				},
				ReturnType: models.ReturnTypeInfo{
					Type: models.ReturnTypeDataError,
				},
			},
			expectError: true,
			errorMsg:    "unsupported parameter type",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			registry := createTestParserRegistry()
			_, err := GenerateRouteWrapper(tt.route, "TestController", registry)
			
			if tt.expectError {
				if err == nil {
					t.Error("expected error but got none")
				} else if !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("expected error message to contain '%s', got: %v", tt.errorMsg, err)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

func TestResponseModelUsage(t *testing.T) {
	// Test that the Response model can be used correctly
	route := models.RouteMetadata{
		Method:      "POST",
		Path:        "/test",
		HandlerName: "TestHandler",
		ReturnType: models.ReturnTypeInfo{
			Type:         models.ReturnTypeResponseError,
			UsesResponse: true,
			HasError:     true,
		},
	}

	registry := createTestParserRegistry()
	wrapper, err := GenerateRouteWrapper(route, "TestController", registry)
	if err != nil {
		t.Fatalf("failed to generate wrapper: %v", err)
	}

	// Verify that the generated code properly handles the Response struct
	expectedPatterns := []string{
		"response, err := handler.TestHandler()",
		"if response == nil {",
		"return c.JSON(response.StatusCode, response.Body)",
	}

	for _, pattern := range expectedPatterns {
		if !strings.Contains(wrapper, pattern) {
			t.Errorf("generated wrapper should contain pattern: %s\n\nGenerated wrapper:\n%s", pattern, wrapper)
		}
	}
}