package templates

import (
	"strings"
	"testing"

	"github.com/toyz/axon/internal/models"
)

func TestGenerateResponseHandling(t *testing.T) {
	tests := []struct {
		name           string
		route          models.RouteMetadata
		controllerName string
		expected       string
		expectError    bool
	}{
		{
			name: "data error return type",
			route: models.RouteMetadata{
				HandlerName: "GetUser",
				ReturnType: models.ReturnTypeInfo{
					Type:     models.ReturnTypeDataError,
					DataType: "User",
					HasError: true,
				},
				Parameters: []models.Parameter{
					{Name: "id", Type: "int", Source: models.ParameterSourcePath},
				},
			},
			controllerName: "UserController",
			expected: `		var data interface{}
		data, err = handler.GetUser(id)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
		}
		return c.JSON(http.StatusOK, data)`,
		},
		{
			name: "response error return type",
			route: models.RouteMetadata{
				HandlerName: "CreateUser",
				ReturnType: models.ReturnTypeInfo{
					Type:         models.ReturnTypeResponseError,
					UsesResponse: true,
					HasError:     true,
				},
				Parameters: []models.Parameter{
					{Name: "user", Type: "User", Source: models.ParameterSourceBody},
				},
			},
			controllerName: "UserController",
			expected: `		response, err := handler.CreateUser(body)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
		}
		if response == nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "handler returned nil response")
		}
		return c.JSON(response.StatusCode, response.Body)`,
		},
		{
			name: "error only return type",
			route: models.RouteMetadata{
				HandlerName: "DeleteUser",
				ReturnType: models.ReturnTypeInfo{
					Type:     models.ReturnTypeError,
					HasError: true,
				},
				Parameters: []models.Parameter{
					{Name: "id", Type: "int", Source: models.ParameterSourcePath},
				},
				Flags: []string{"-PassContext"},
			},
			controllerName: "UserController",
			expected: `		err = handler.DeleteUser(c, id)
		if err != nil {
			return err
		}
		return nil`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := GenerateResponseHandling(tt.route, tt.controllerName)
			
			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
				}
				return
			}
			
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}
			
			if strings.TrimSpace(result) != strings.TrimSpace(tt.expected) {
				t.Errorf("expected:\n%s\n\ngot:\n%s", tt.expected, result)
			}
		})
	}
}

func TestGenerateHandlerCall(t *testing.T) {
	tests := []struct {
		name           string
		route          models.RouteMetadata
		controllerName string
		expected       string
	}{
		{
			name: "simple path parameter",
			route: models.RouteMetadata{
				HandlerName: "GetUser",
				Parameters: []models.Parameter{
					{Name: "id", Type: "int", Source: models.ParameterSourcePath},
				},
			},
			controllerName: "UserController",
			expected:       "handler.GetUser(id)",
		},
		{
			name: "with context and path parameter",
			route: models.RouteMetadata{
				HandlerName: "GetUser",
				Parameters: []models.Parameter{
					{Name: "id", Type: "int", Source: models.ParameterSourcePath},
				},
				Flags: []string{"-PassContext"},
			},
			controllerName: "UserController",
			expected:       "handler.GetUser(c, id)",
		},
		{
			name: "with body parameter",
			route: models.RouteMetadata{
				HandlerName: "CreateUser",
				Parameters: []models.Parameter{
					{Name: "user", Type: "User", Source: models.ParameterSourceBody},
				},
			},
			controllerName: "UserController",
			expected:       "handler.CreateUser(body)",
		},
		{
			name: "multiple path parameters",
			route: models.RouteMetadata{
				HandlerName: "GetUserPost",
				Parameters: []models.Parameter{
					{Name: "userId", Type: "int", Source: models.ParameterSourcePath},
					{Name: "postId", Type: "int", Source: models.ParameterSourcePath},
				},
			},
			controllerName: "UserController",
			expected:       "handler.GetUserPost(userId, postId)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := generateHandlerCall(tt.route, tt.controllerName)
			
			if result != tt.expected {
				t.Errorf("expected: %s, got: %s", tt.expected, result)
			}
		})
	}
}

func TestGenerateRouteWrapper(t *testing.T) {
	tests := []struct {
		name           string
		route          models.RouteMetadata
		controllerName string
		shouldContain  []string
		expectError    bool
	}{
		{
			name: "complete wrapper with data error return",
			route: models.RouteMetadata{
				Method:      "GET",
				Path:        "/users/{id:int}",
				HandlerName: "GetUser",
				Parameters: []models.Parameter{
					{Name: "id", Type: "int", Source: models.ParameterSourcePath},
				},
				ReturnType: models.ReturnTypeInfo{
					Type:     models.ReturnTypeDataError,
					DataType: "User",
					HasError: true,
				},
			},
			controllerName: "UserController",
			shouldContain: []string{
				"func wrapUserControllerGetUser(handler *UserController) echo.HandlerFunc",
				"id, err := strconv.Atoi(c.Param(\"id\"))",
				"data, err := handler.GetUser(id)",
				"return c.JSON(http.StatusOK, data)",
			},
		},
		{
			name: "wrapper with body parameter and response error return",
			route: models.RouteMetadata{
				Method:      "POST",
				Path:        "/users",
				HandlerName: "CreateUser",
				Parameters: []models.Parameter{
					{Name: "user", Type: "User", Source: models.ParameterSourceBody},
				},
				ReturnType: models.ReturnTypeInfo{
					Type:         models.ReturnTypeResponseError,
					UsesResponse: true,
					HasError:     true,
				},
			},
			controllerName: "UserController",
			shouldContain: []string{
				"func wrapUserControllerCreateUser(handler *UserController) echo.HandlerFunc",
				"var body User",
				"if err := c.Bind(&body); err != nil",
				"response, err := handler.CreateUser(body)",
				"return c.JSON(response.StatusCode, response.Body)",
			},
		},
		{
			name: "wrapper with single middleware",
			route: models.RouteMetadata{
				Method:      "POST",
				Path:        "/users",
				HandlerName: "CreateUser",
				Parameters: []models.Parameter{
					{Name: "user", Type: "User", Source: models.ParameterSourceBody},
				},
				ReturnType: models.ReturnTypeInfo{
					Type:     models.ReturnTypeDataError,
					DataType: "User",
					HasError: true,
				},
				Middlewares: []string{"Auth"},
			},
			controllerName: "UserController",
			shouldContain: []string{
				"func wrapUserControllerCreateUser(handler *UserController, auth *Auth) echo.HandlerFunc",
				"baseHandler := func(c echo.Context) error {",
				"// Apply middlewares in order",
				"finalHandler := baseHandler",
				"finalHandler = auth.Handle(finalHandler)",
				"return finalHandler",
			},
		},
		{
			name: "wrapper with multiple middlewares",
			route: models.RouteMetadata{
				Method:      "POST",
				Path:        "/users",
				HandlerName: "CreateUser",
				Parameters: []models.Parameter{
					{Name: "user", Type: "User", Source: models.ParameterSourceBody},
				},
				ReturnType: models.ReturnTypeInfo{
					Type:     models.ReturnTypeDataError,
					DataType: "User",
					HasError: true,
				},
				Middlewares: []string{"Auth", "Logging"},
			},
			controllerName: "UserController",
			shouldContain: []string{
				"func wrapUserControllerCreateUser(handler *UserController, auth *Auth, logging *Logging) echo.HandlerFunc",
				"baseHandler := func(c echo.Context) error {",
				"// Apply middlewares in order",
				"finalHandler := baseHandler",
				"finalHandler = logging.Handle(finalHandler)",
				"finalHandler = auth.Handle(finalHandler)",
				"return finalHandler",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			registry := createTestParserRegistry()
			result, err := GenerateRouteWrapper(tt.route, tt.controllerName, registry)
			
			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
				}
				return
			}
			
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}
			
			for _, expected := range tt.shouldContain {
				if !strings.Contains(result, expected) {
					t.Errorf("expected result to contain: %s\n\nActual result:\n%s", expected, result)
				}
			}
		})
	}
}

func TestHasPassContextFlag(t *testing.T) {
	tests := []struct {
		name     string
		flags    []string
		expected bool
	}{
		{
			name:     "has pass context flag",
			flags:    []string{"-PassContext", "SomeOtherFlag"},
			expected: true,
		},
		{
			name:     "no pass context flag",
			flags:    []string{"SomeOtherFlag", "AnotherFlag"},
			expected: false,
		},
		{
			name:     "empty flags",
			flags:    []string{},
			expected: false,
		},
		{
			name:     "nil flags",
			flags:    nil,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := hasPassContextFlag(tt.flags)
			if result != tt.expected {
				t.Errorf("expected: %v, got: %v", tt.expected, result)
			}
		})
	}
}

func TestGenerateBodyBindingCode(t *testing.T) {
	tests := []struct {
		name       string
		parameters []models.Parameter
		expected   string
	}{
		{
			name: "with body parameter",
			parameters: []models.Parameter{
				{Name: "user", Type: "User", Source: models.ParameterSourceBody},
				{Name: "id", Type: "int", Source: models.ParameterSourcePath},
			},
			expected: `		var body User
		if err := c.Bind(&body); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, err.Error())
		}
`,
		},
		{
			name: "no body parameter",
			parameters: []models.Parameter{
				{Name: "id", Type: "int", Source: models.ParameterSourcePath},
			},
			expected: "",
		},
		{
			name:       "empty parameters",
			parameters: []models.Parameter{},
			expected:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := generateBodyBindingCode(tt.parameters)
			if result != tt.expected {
				t.Errorf("expected:\n%s\n\ngot:\n%s", tt.expected, result)
			}
		})
	}
}

func TestGenerateMiddlewareParameters(t *testing.T) {
	tests := []struct {
		name        string
		middlewares []string
		expected    string
	}{
		{
			name:        "single middleware",
			middlewares: []string{"Auth"},
			expected:    "auth *Auth",
		},
		{
			name:        "multiple middlewares",
			middlewares: []string{"Auth", "Logging", "RateLimit"},
			expected:    "auth *Auth, logging *Logging, ratelimit *RateLimit",
		},
		{
			name:        "no middlewares",
			middlewares: []string{},
			expected:    "",
		},
		{
			name:        "nil middlewares",
			middlewares: nil,
			expected:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := generateMiddlewareParameters(tt.middlewares)
			if result != tt.expected {
				t.Errorf("expected: %s, got: %s", tt.expected, result)
			}
		})
	}
}

func TestGenerateMiddlewareApplication(t *testing.T) {
	tests := []struct {
		name        string
		middlewares []string
		expected    string
	}{
		{
			name:        "single middleware",
			middlewares: []string{"Auth"},
			expected: `
	// Apply middlewares in order
	finalHandler := baseHandler
	finalHandler = auth.Handle(finalHandler)
`,
		},
		{
			name:        "multiple middlewares in order",
			middlewares: []string{"Auth", "Logging"},
			expected: `
	// Apply middlewares in order
	finalHandler := baseHandler
	finalHandler = logging.Handle(finalHandler)
	finalHandler = auth.Handle(finalHandler)
`,
		},
		{
			name:        "three middlewares in order",
			middlewares: []string{"Auth", "Logging", "RateLimit"},
			expected: `
	// Apply middlewares in order
	finalHandler := baseHandler
	finalHandler = ratelimit.Handle(finalHandler)
	finalHandler = logging.Handle(finalHandler)
	finalHandler = auth.Handle(finalHandler)
`,
		},
		{
			name:        "no middlewares",
			middlewares: []string{},
			expected:    "",
		},
		{
			name:        "nil middlewares",
			middlewares: nil,
			expected:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := generateMiddlewareApplication(tt.middlewares)
			if result != tt.expected {
				t.Errorf("expected:\n%s\n\ngot:\n%s", tt.expected, result)
			}
		})
	}
}

func TestGenerateHandlerCall_WithContextParameters(t *testing.T) {
	tests := []struct {
		name           string
		route          models.RouteMetadata
		controllerName string
		expected       string
	}{
		{
			name: "handler with context parameter at beginning",
			route: models.RouteMetadata{
				HandlerName: "UserController.GetUser",
				Parameters: []models.Parameter{
					{
						Name:     "c",
						Type:     "echo.Context",
						Source:   models.ParameterSourceContext,
						Required: true,
						Position: 0,
					},
					{
						Name:     "id",
						Type:     "int",
						Source:   models.ParameterSourcePath,
						Required: true,
					},
				},
			},
			controllerName: "UserController",
			expected:       "handler.GetUser(c, id)",
		},
		{
			name: "handler with context parameter in middle",
			route: models.RouteMetadata{
				HandlerName: "UserController.UpdateUser",
				Parameters: []models.Parameter{
					{
						Name:     "id",
						Type:     "int",
						Source:   models.ParameterSourcePath,
						Required: true,
					},
					{
						Name:     "ctx",
						Type:     "echo.Context",
						Source:   models.ParameterSourceContext,
						Required: true,
						Position: 1,
					},
					{
						Name:     "data",
						Type:     "string",
						Source:   models.ParameterSourcePath,
						Required: true,
					},
				},
			},
			controllerName: "UserController",
			expected:       "handler.UpdateUser(c, id, data)",
		},
		{
			name: "handler with PassContext flag (legacy)",
			route: models.RouteMetadata{
				HandlerName: "UserController.HealthCheck",
				Flags:       []string{"-PassContext"},
				Parameters: []models.Parameter{
					{
						Name:     "id",
						Type:     "int",
						Source:   models.ParameterSourcePath,
						Required: true,
					},
				},
			},
			controllerName: "UserController",
			expected:       "handler.HealthCheck(c, id)",
		},
		{
			name: "handler with both context parameter and PassContext flag",
			route: models.RouteMetadata{
				HandlerName: "UserController.ComplexHandler",
				Flags:       []string{"-PassContext"},
				Parameters: []models.Parameter{
					{
						Name:     "ctx",
						Type:     "echo.Context",
						Source:   models.ParameterSourceContext,
						Required: true,
						Position: 0,
					},
					{
						Name:     "id",
						Type:     "int",
						Source:   models.ParameterSourcePath,
						Required: true,
					},
				},
			},
			controllerName: "UserController",
			expected:       "handler.ComplexHandler(c, id)",
		},
		{
			name: "handler with no parameters",
			route: models.RouteMetadata{
				HandlerName: "UserController.GetAll",
				Parameters:  []models.Parameter{},
			},
			controllerName: "UserController",
			expected:       "handler.GetAll()",
		},
		{
			name: "handler with body parameter",
			route: models.RouteMetadata{
				HandlerName: "UserController.CreateUser",
				Parameters: []models.Parameter{
					{
						Name:     "c",
						Type:     "echo.Context",
						Source:   models.ParameterSourceContext,
						Required: true,
						Position: 0,
					},
					{
						Name:     "body",
						Type:     "User",
						Source:   models.ParameterSourceBody,
						Required: true,
					},
				},
			},
			controllerName: "UserController",
			expected:       "handler.CreateUser(c, body)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := generateHandlerCall(tt.route, tt.controllerName)

			if result != tt.expected {
				t.Errorf("expected: %s, got: %s", tt.expected, result)
			}
		})
	}
}