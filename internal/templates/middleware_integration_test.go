package templates

import (
	"strings"
	"testing"

	"github.com/toyz/axon/internal/models"
)

func TestMiddlewareIntegration_CompleteWorkflow(t *testing.T) {
	// Test the complete middleware application workflow
	route := models.RouteMetadata{
		Method:      "POST",
		Path:        "/api/users",
		HandlerName: "CreateUser",
		Parameters: []models.Parameter{
			{Name: "user", Type: "CreateUserRequest", Source: models.ParameterSourceBody},
		},
		ReturnType: models.ReturnTypeInfo{
			Type:     models.ReturnTypeDataError,
			DataType: "User",
			HasError: true,
		},
		Middlewares: []string{"Auth", "Logging", "RateLimit"},
	}
	controllerName := "UserController"

	// Generate route wrapper with middleware
	registry := createTestParserRegistry()
	wrapper, err := GenerateRouteWrapper(route, controllerName, registry)
	if err != nil {
		t.Fatalf("failed to generate route wrapper: %v", err)
	}

	// Generate route registration
	registration, err := GenerateRouteRegistration(route, controllerName, route.Middlewares)
	if err != nil {
		t.Fatalf("failed to generate route registration: %v", err)
	}

	// Verify wrapper contains all expected elements
	expectedWrapperElements := []string{
		"func wrapUserControllerCreateUser(handler *UserController, auth *Auth, logging *Logging, ratelimit *RateLimit) echo.HandlerFunc",
		"baseHandler := func(c echo.Context) error {",
		"var body CreateUserRequest",
		"if err := c.Bind(&body); err != nil",
		"data, err := handler.CreateUser(body)",
		"// Apply middlewares in order",
		"finalHandler := baseHandler",
		"finalHandler = ratelimit.Handle(finalHandler)",
		"finalHandler = logging.Handle(finalHandler)",
		"finalHandler = auth.Handle(finalHandler)",
		"return finalHandler",
	}

	for _, expected := range expectedWrapperElements {
		if !strings.Contains(wrapper, expected) {
			t.Errorf("wrapper missing expected element: %s\n\nGenerated wrapper:\n%s", expected, wrapper)
		}
	}

	// Verify registration contains expected elements
	expectedContains := []string{
		`e.POST("/api/users", handler_usercontrollercreateuser)`,
		`handler_usercontrollercreateuser := wrapUserControllerCreateUser(UserController, auth, logging, ratelimit)`,
		`axon.DefaultRouteRegistry.RegisterRoute`,
		`Middlewares:         []string{"Auth", "Logging", "RateLimit"}`,
		`MiddlewareInstances: []axon.MiddlewareInstance{{`,
		`Name:     "Auth"`,
		`Handler:  auth.Handle`,
		`Instance: auth`,
	}
	
	for _, expected := range expectedContains {
		if !strings.Contains(registration, expected) {
			t.Errorf("expected registration to contain: %s\nGot: %s", expected, registration)
		}
	}

	// Verify middleware ordering (should be applied in reverse order for correct execution)
	middlewareOrderCheck := []string{
		"finalHandler = ratelimit.Handle(finalHandler)",
		"finalHandler = logging.Handle(finalHandler)", 
		"finalHandler = auth.Handle(finalHandler)",
	}

	lastIndex := -1
	for _, line := range middlewareOrderCheck {
		index := strings.Index(wrapper, line)
		if index == -1 {
			t.Errorf("middleware application line not found: %s", line)
			continue
		}
		if index <= lastIndex {
			t.Errorf("middleware application order is incorrect. Expected %s to come after previous middleware", line)
		}
		lastIndex = index
	}
}

func TestMiddlewareIntegration_EdgeCases(t *testing.T) {
	tests := []struct {
		name        string
		route       models.RouteMetadata
		controller  string
		shouldError bool
	}{
		{
			name: "route with no middlewares",
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
				Middlewares: []string{},
			},
			controller:  "UserController",
			shouldError: false,
		},
		{
			name: "route with single middleware",
			route: models.RouteMetadata{
				Method:      "DELETE",
				Path:        "/users/{id:int}",
				HandlerName: "DeleteUser",
				Parameters: []models.Parameter{
					{Name: "id", Type: "int", Source: models.ParameterSourcePath},
				},
				ReturnType: models.ReturnTypeInfo{
					Type:     models.ReturnTypeError,
					HasError: true,
				},
				Middlewares: []string{"Auth"},
				Flags:       []string{"PassContext"},
			},
			controller:  "UserController",
			shouldError: false,
		},
		{
			name: "route with complex middleware chain",
			route: models.RouteMetadata{
				Method:      "PUT",
				Path:        "/users/{id:int}/profile",
				HandlerName: "UpdateProfile",
				Parameters: []models.Parameter{
					{Name: "id", Type: "int", Source: models.ParameterSourcePath},
					{Name: "profile", Type: "ProfileUpdateRequest", Source: models.ParameterSourceBody},
				},
				ReturnType: models.ReturnTypeInfo{
					Type:         models.ReturnTypeResponseError,
					UsesResponse: true,
					HasError:     true,
				},
				Middlewares: []string{"Auth", "Validation", "Logging", "Metrics", "RateLimit"},
			},
			controller:  "UserController",
			shouldError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			registry := createTestParserRegistry()
			wrapper, err := GenerateRouteWrapper(tt.route, tt.controller, registry)
			if tt.shouldError {
				if err == nil {
					t.Errorf("expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			registration, err := GenerateRouteRegistration(tt.route, tt.controller, tt.route.Middlewares)
			if err != nil {
				t.Errorf("unexpected error generating registration: %v", err)
				return
			}

			// Verify basic structure
			if len(tt.route.Middlewares) > 0 {
				// Should have middleware parameters
				if !strings.Contains(wrapper, "baseHandler := func(c echo.Context) error {") {
					t.Errorf("wrapper with middlewares should have baseHandler")
				}
				if !strings.Contains(wrapper, "// Apply middlewares in order") {
					t.Errorf("wrapper with middlewares should have middleware application comment")
				}
				if !strings.Contains(wrapper, "return finalHandler") {
					t.Errorf("wrapper with middlewares should return finalHandler")
				}
			} else {
				// Should not have middleware application code
				if strings.Contains(wrapper, "baseHandler") {
					t.Errorf("wrapper without middlewares should not have baseHandler")
				}
				if strings.Contains(wrapper, "finalHandler") {
					t.Errorf("wrapper without middlewares should not have finalHandler")
				}
			}

			// Verify registration format
			expectedMethod := strings.ToUpper(tt.route.Method)
			if !strings.Contains(registration, "e."+expectedMethod) {
				t.Errorf("registration should contain e.%s", expectedMethod)
			}
			if !strings.Contains(registration, tt.route.Path) {
				t.Errorf("registration should contain path %s", tt.route.Path)
			}
		})
	}
}

func TestMiddlewareIntegration_ParameterGeneration(t *testing.T) {
	// Test that middleware parameters are generated correctly
	middlewares := []string{"Auth", "Logging", "RateLimit", "Validation"}
	
	params := generateMiddlewareParameters(middlewares)
	expected := "auth *Auth, logging *Logging, ratelimit *RateLimit, validation *Validation"
	
	if params != expected {
		t.Errorf("expected parameters: %s, got: %s", expected, params)
	}

	// Test middleware application order
	application := generateMiddlewareApplication(middlewares)
	
	// Should apply in reverse order
	expectedOrder := []string{
		"finalHandler = validation.Handle(finalHandler)",
		"finalHandler = ratelimit.Handle(finalHandler)", 
		"finalHandler = logging.Handle(finalHandler)",
		"finalHandler = auth.Handle(finalHandler)",
	}

	for i, expectedLine := range expectedOrder {
		if !strings.Contains(application, expectedLine) {
			t.Errorf("middleware application missing line %d: %s", i, expectedLine)
		}
	}

	// Verify order is correct
	lastIndex := -1
	for i, line := range expectedOrder {
		index := strings.Index(application, line)
		if index == -1 {
			t.Errorf("middleware application line %d not found: %s", i, line)
			continue
		}
		if index <= lastIndex {
			t.Errorf("middleware application order incorrect at line %d: %s", i, line)
		}
		lastIndex = index
	}
}