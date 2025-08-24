package templates

import (
	"strings"
	"testing"

	"github.com/toyz/axon/internal/models"
	"github.com/toyz/axon/internal/registry"
)

// createTestParserRegistry creates a test parser registry for testing
func createTestParserRegistry() ParserRegistryInterface {
	return registry.NewParserRegistry()
}

// Note: TestGenerateParameterBinding was removed as the function was unused dead code.

func TestGenerateParameterBindingCode(t *testing.T) {
	tests := []struct {
		name       string
		parameters []models.Parameter
		expected   string
	}{
		{
			name: "single int parameter",
			parameters: []models.Parameter{
				{
					Name:     "id",
					Type:     "int",
					Source:   models.ParameterSourcePath,
					Required: true,
				},
			},
			expected: `		id, err := axon.ParseInt(c, c.Param("id"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Invalid id: %v", err))
		}
`,
		},
		{
			name: "single string parameter",
			parameters: []models.Parameter{
				{
					Name:     "name",
					Type:     "string",
					Source:   models.ParameterSourcePath,
					Required: true,
				},
			},
			expected: `		name, err := axon.ParseString(c, c.Param("name"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Invalid name: %v", err))
		}
`,
		},
		{
			name: "multiple parameters",
			parameters: []models.Parameter{
				{
					Name:     "id",
					Type:     "int",
					Source:   models.ParameterSourcePath,
					Required: true,
				},
				{
					Name:     "slug",
					Type:     "string",
					Source:   models.ParameterSourcePath,
					Required: true,
				},
			},
			expected: `		id, err := axon.ParseInt(c, c.Param("id"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Invalid id: %v", err))
		}
		slug, err := axon.ParseString(c, c.Param("slug"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Invalid slug: %v", err))
		}
`,
		},
		{
			name:       "no parameters",
			parameters: []models.Parameter{},
			expected:   "",
		},
		{
			name: "mixed parameter sources (only path should be processed)",
			parameters: []models.Parameter{
				{
					Name:     "id",
					Type:     "int",
					Source:   models.ParameterSourcePath,
					Required: true,
				},
				{
					Name:     "body",
					Type:     "string",
					Source:   models.ParameterSourceBody,
					Required: true,
				},
			},
			expected: `		id, err := axon.ParseInt(c, c.Param("id"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Invalid id: %v", err))
		}
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			registry := createTestParserRegistry()
			result, err := GenerateParameterBindingCode(tt.parameters, registry)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if result != tt.expected {
				t.Errorf("expected:\n%s\ngot:\n%s", tt.expected, result)
			}
		})
	}
}

func TestGenerateParameterBindingCodeError(t *testing.T) {
	parameters := []models.Parameter{
		{
			Name:     "value",
			Type:     "unsupported",
			Source:   models.ParameterSourcePath,
			Required: true,
		},
	}

	registry := createTestParserRegistry()
	_, err := GenerateParameterBindingCode(parameters, registry)
	if err == nil {
		t.Errorf("expected error for unsupported parameter type")
	}

	if !strings.Contains(err.Error(), "unsupported parameter type: unsupported") {
		t.Errorf("expected error message to mention unsupported type, got: %v", err)
	}
}

func TestGetParameterSourceString(t *testing.T) {
	tests := []struct {
		source   models.ParameterSource
		expected string
	}{
		{models.ParameterSourcePath, "path"},
		{models.ParameterSourceBody, "body"},
		{models.ParameterSourceContext, "context"},
		{models.ParameterSource(999), "unknown"}, // Invalid source
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := getParameterSourceString(tt.source)
			if result != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestGenerateRouteRegistration(t *testing.T) {
	tests := []struct {
		name           string
		route          models.RouteMetadata
		controllerVar  string
		middlewares    []string
		expectedContains []string
		expectError    bool
	}{
		{
			name: "route without middleware",
			route: models.RouteMetadata{
				Method:      "GET",
				Path:        "/users/{id:int}",
				HandlerName: "GetUser",
				Parameters: []models.Parameter{
					{Name: "id", Type: "int", Source: models.ParameterSourcePath},
				},
			},
			controllerVar: "controller",
			middlewares:   []string{},
			expectedContains: []string{
				`e.GET("/users/:id", handler_controllergetuser)`,
				`handler_controllergetuser := wrapcontrollerGetUser(controller)`,
				`axon.DefaultRouteRegistry.RegisterRoute`,
				`Method:              "GET"`,
				`Path:                "/users/{id:int}"`,
				`EchoPath:            "/users/:id"`,
			},
		},
		{
			name: "route with single middleware",
			route: models.RouteMetadata{
				Method:      "POST",
				Path:        "/users",
				HandlerName: "CreateUser",
			},
			controllerVar: "controller",
			middlewares:   []string{"Auth"},
			expectedContains: []string{
				`e.POST("/users", handler_controllercreateuser)`,
				`handler_controllercreateuser := wrapcontrollerCreateUser(controller, auth)`,
				`Middlewares:         []string{"Auth"}`,
				`MiddlewareInstances: []axon.MiddlewareInstance{{`,
				`Name:     "Auth"`,
				`Handler:  auth.Handle`,
				`Instance: auth`,
			},
		},
		{
			name: "route with multiple middlewares",
			route: models.RouteMetadata{
				Method:      "PUT",
				Path:        "/users/{id:int}",
				HandlerName: "UpdateUser",
				Parameters: []models.Parameter{
					{Name: "id", Type: "int", Source: models.ParameterSourcePath},
				},
			},
			controllerVar: "controller",
			middlewares:   []string{"Auth", "Logging", "RateLimit"},
			expectedContains: []string{
				`e.PUT("/users/:id", handler_controllerupdateuser)`,
				`handler_controllerupdateuser := wrapcontrollerUpdateUser(controller, auth, logging, ratelimit)`,
				`Middlewares:         []string{"Auth", "Logging", "RateLimit"}`,
				`MiddlewareInstances: []axon.MiddlewareInstance{{`,
				`Name:     "Auth"`,
				`Handler:  auth.Handle`,
			},
		},
		{
			name: "DELETE route with middleware",
			route: models.RouteMetadata{
				Method:      "DELETE",
				Path:        "/users/{id:int}",
				HandlerName: "DeleteUser",
				Parameters: []models.Parameter{
					{Name: "id", Type: "int", Source: models.ParameterSourcePath},
				},
			},
			controllerVar: "controller",
			middlewares:   []string{"Auth", "AdminOnly"},
			expectedContains: []string{
				`e.DELETE("/users/:id", handler_controllerdeleteuser)`,
				`handler_controllerdeleteuser := wrapcontrollerDeleteUser(controller, auth, adminonly)`,
				`Middlewares:         []string{"Auth", "AdminOnly"}`,
				`MiddlewareInstances: []axon.MiddlewareInstance{{`,
				`Name:     "Auth"`,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := GenerateRouteRegistration(tt.route, tt.controllerVar, tt.middlewares)
			
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
			
			// Check that all expected strings are contained in the result
			for _, expected := range tt.expectedContains {
				if !strings.Contains(result, expected) {
					t.Errorf("expected result to contain: %s\nGot: %s", expected, result)
				}
			}
		})
	}
}

// TestGenerateCoreServiceProvider tests the generation of FX providers for core services
func TestGenerateCoreServiceProvider(t *testing.T) {
	tests := []struct {
		name     string
		service  models.CoreServiceMetadata
		expected string
	}{
		{
			name: "simple core service without dependencies",
			service: models.CoreServiceMetadata{
				Name:         "UserService",
				StructName:   "UserService",
				HasLifecycle: false,
				IsManual:     false,
				Dependencies: []models.Dependency{},
			},
			expected: `func NewUserService() *UserService {
	return &UserService{
		
	}
}`,
		},
		{
			name: "core service with dependencies",
			service: models.CoreServiceMetadata{
				Name:         "UserService",
				StructName:   "UserService",
				HasLifecycle: false,
				IsManual:     false,
				Dependencies: []models.Dependency{
					{Name: "UserRepository", Type: "UserRepository"},
					{Name: "Config", Type: "*Config"},
				},
			},
			expected: `func NewUserService(UserRepository UserRepository, Config *Config) *UserService {
	return &UserService{
		UserRepository: UserRepository,
		Config: Config,
	}
}`,
		},
		{
			name: "lifecycle service with dependencies and Start method only",
			service: models.CoreServiceMetadata{
				Name:         "DatabaseService",
				StructName:   "DatabaseService",
				HasLifecycle: true,
				HasStart:     true,
				HasStop:      false,
				IsManual:     false,
				Dependencies: []models.Dependency{
					{Name: "Config", Type: "*Config"},
				},
			},
			expected: `func NewDatabaseService(lc fx.Lifecycle, Config *Config) *DatabaseService {
	service := &DatabaseService{
		Config: Config,
	}
	
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			return service.Start(ctx)
		},
	})
	
	return service
}`,
		},
		{
			name: "lifecycle service with both Start and Stop methods",
			service: models.CoreServiceMetadata{
				Name:         "MessageConsumer",
				StructName:   "MessageConsumer",
				HasLifecycle: true,
				HasStart:     true,
				HasStop:      true,
				IsManual:     false,
				Dependencies: []models.Dependency{
					{Name: "Config", Type: "*Config"},
					{Name: "Logger", Type: "Logger"},
				},
			},
			expected: `func NewMessageConsumer(lc fx.Lifecycle, Config *Config, Logger Logger) *MessageConsumer {
	service := &MessageConsumer{
		Config: Config,
		Logger: Logger,
	}
	
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			return service.Start(ctx)
		},
		OnStop: func(ctx context.Context) error {
			return service.Stop(ctx)
		},
	})
	
	return service
}`,
		},
		{
			name: "manual service (should return empty)",
			service: models.CoreServiceMetadata{
				Name:         "ConfigService",
				StructName:   "ConfigService",
				HasLifecycle: false,
				IsManual:     true,
				ModuleName:   "CustomModule",
				Dependencies: []models.Dependency{},
			},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := GenerateCoreServiceProvider(tt.service)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// Normalize whitespace for comparison - remove extra spaces and normalize line endings
			normalizeWhitespace := func(s string) string {
				// Replace multiple spaces with single space, normalize line endings
				lines := strings.Split(s, "\n")
				var normalized []string
				for _, line := range lines {
					trimmed := strings.TrimSpace(line)
					if trimmed != "" || len(normalized) > 0 { // Keep empty lines in the middle but not at start
						normalized = append(normalized, trimmed)
					}
				}
				return strings.Join(normalized, "\n")
			}
			
			expected := normalizeWhitespace(tt.expected)
			actual := normalizeWhitespace(result)

			if actual != expected {
				t.Errorf("generated provider mismatch:\nExpected:\n%s\n\nActual:\n%s", expected, actual)
			}
		})
	}
}

// TestGenerateCoreServiceModule tests the generation of complete FX modules for core services
func TestGenerateCoreServiceModule(t *testing.T) {
	tests := []struct {
		name     string
		metadata *models.PackageMetadata
		contains []string // Strings that should be present in the output
	}{
		{
			name: "package with multiple core services",
			metadata: &models.PackageMetadata{
				PackageName: "services",
				PackagePath: "./services",
				CoreServices: []models.CoreServiceMetadata{
					{
						Name:         "UserService",
						StructName:   "UserService",
						HasLifecycle: false,
						IsManual:     false,
						Dependencies: []models.Dependency{
							{Name: "UserRepository", Type: "UserRepository"},
						},
					},
					{
						Name:         "DatabaseService",
						StructName:   "DatabaseService",
						HasLifecycle: true,
						IsManual:     false,
						Dependencies: []models.Dependency{
							{Name: "Config", Type: "*Config"},
						},
					},
					{
						Name:         "ConfigService",
						StructName:   "ConfigService",
						HasLifecycle: false,
						IsManual:     true,
						ModuleName:   "CustomModule",
						Dependencies: []models.Dependency{},
					},
				},
			},
			contains: []string{
				"package services",
				"import (",
				"\"context\"",
				"\"go.uber.org/fx\"",
				"func NewUserService(",
				"func NewDatabaseService(",
				"var AutogenModule = fx.Module(",
				"fx.Provide(NewUserService),",
				"fx.Provide(NewDatabaseService),",
				"CustomModule,",
			},
		},
		{
			name: "package with only manual services",
			metadata: &models.PackageMetadata{
				PackageName: "config",
				PackagePath: "./config",
				CoreServices: []models.CoreServiceMetadata{
					{
						Name:         "ConfigService",
						StructName:   "ConfigService",
						HasLifecycle: false,
						IsManual:     true,
						ModuleName:   "Module",
						Dependencies: []models.Dependency{},
					},
				},
			},
			contains: []string{
				"package config",
				"var AutogenModule = fx.Module(",
				"Module,",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := GenerateCoreServiceModule(tt.metadata)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// Check that all expected strings are present
			for _, expected := range tt.contains {
				if !strings.Contains(result, expected) {
					t.Errorf("generated module missing expected content: %s\n\nGenerated:\n%s", expected, result)
				}
			}
		})
	}
}

// TestExtractDependencyName tests the dependency name extraction logic
func TestExtractDependencyName(t *testing.T) {
	tests := []struct {
		depType  string
		expected string
	}{
		{"UserRepository", "UserRepository"},
		{"*Config", "Config"},
		{"pkg.Service", "Service"},
		{"*pkg.Interface", "Interface"},
		{"HTTPClient", "HTTPClient"},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.depType, func(t *testing.T) {
			result := extractDependencyName(tt.depType)
			if result != tt.expected {
				t.Errorf("extractDependencyName(%s) = %s, expected %s", tt.depType, result, tt.expected)
			}
		})
	}
}

// TestGenerateInterface tests the generation of interface code from metadata
func TestGenerateInterface(t *testing.T) {
	tests := []struct {
		name     string
		iface    models.InterfaceMetadata
		expected string
	}{
		{
			name: "simple interface with basic methods",
			iface: models.InterfaceMetadata{
				Name:       "UserServiceInterface",
				StructName: "UserService",
				Methods: []models.Method{
					{
						Name: "GetUser",
						Parameters: []models.Parameter{
							{Name: "id", Type: "int"},
						},
						Returns: []string{"*User", "error"},
					},
					{
						Name: "CreateUser",
						Parameters: []models.Parameter{
							{Name: "user", Type: "User"},
						},
						Returns: []string{"*User", "error"},
					},
				},
			},
			expected: `// UserServiceInterface is the interface for UserService
type UserServiceInterface interface {
	GetUser(id int) (*User, error)
	CreateUser(user User) (*User, error)
}`,
		},
		{
			name: "interface with complex method signatures",
			iface: models.InterfaceMetadata{
				Name:       "ProcessorInterface",
				StructName: "Processor",
				Methods: []models.Method{
					{
						Name: "Process",
						Parameters: []models.Parameter{
							{Name: "ctx", Type: "context.Context"},
							{Name: "data", Type: "[]byte"},
							{Name: "callback", Type: "func(error)"},
						},
						Returns: []string{"error"},
					},
					{
						Name:       "GetChannel",
						Parameters: []models.Parameter{},
						Returns:    []string{"<-chan Result"},
					},
				},
			},
			expected: `// ProcessorInterface is the interface for Processor
type ProcessorInterface interface {
	Process(ctx context.Context, data []byte, callback func(error)) (error)
	GetChannel() (<-chan Result)
}`,
		},
		{
			name: "interface with no methods",
			iface: models.InterfaceMetadata{
				Name:       "EmptyInterface",
				StructName: "Empty",
				Methods:    []models.Method{},
			},
			expected: `// EmptyInterface is the interface for Empty
type EmptyInterface interface {
}`,
		},
		{
			name: "interface with anonymous parameters",
			iface: models.InterfaceMetadata{
				Name:       "HandlerInterface",
				StructName: "Handler",
				Methods: []models.Method{
					{
						Name: "Handle",
						Parameters: []models.Parameter{
							{Type: "context.Context"},
							{Type: "Request"},
						},
						Returns: []string{"Response", "error"},
					},
				},
			},
			expected: `// HandlerInterface is the interface for Handler
type HandlerInterface interface {
	Handle(context.Context, Request) (Response, error)
}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := GenerateInterface(tt.iface)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// Normalize whitespace for comparison
			expected := strings.TrimSpace(tt.expected)
			actual := strings.TrimSpace(result)

			if actual != expected {
				t.Errorf("generated interface mismatch:\nExpected:\n%s\n\nActual:\n%s", expected, actual)
			}
		})
	}
}

// TestGenerateInterfaceProvider tests the generation of FX provider for interface casting
func TestGenerateInterfaceProvider(t *testing.T) {
	tests := []struct {
		name     string
		iface    models.InterfaceMetadata
		expected string
	}{
		{
			name: "simple interface provider",
			iface: models.InterfaceMetadata{
				Name:       "UserServiceInterface",
				StructName: "UserService",
			},
			expected: `func NewUserServiceInterface(impl *UserService) UserServiceInterface {
	return impl
}`,
		},
		{
			name: "complex interface provider",
			iface: models.InterfaceMetadata{
				Name:       "MessageProcessorInterface",
				StructName: "MessageProcessor",
			},
			expected: `func NewMessageProcessorInterface(impl *MessageProcessor) MessageProcessorInterface {
	return impl
}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := GenerateInterfaceProvider(tt.iface)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// Normalize whitespace for comparison
			expected := strings.TrimSpace(tt.expected)
			actual := strings.TrimSpace(result)

			if actual != expected {
				t.Errorf("generated provider mismatch:\nExpected:\n%s\n\nActual:\n%s", expected, actual)
			}
		})
	}
}

// TestGenerateCoreServiceModuleWithInterfaces tests module generation including interfaces
func TestGenerateCoreServiceModuleWithInterfaces(t *testing.T) {
	metadata := &models.PackageMetadata{
		PackageName: "services",
		PackagePath: "./services",
		CoreServices: []models.CoreServiceMetadata{
			{
				Name:         "UserService",
				StructName:   "UserService",
				HasLifecycle: false,
				IsManual:     false,
				Dependencies: []models.Dependency{
					{Name: "UserRepository", Type: "UserRepository"},
				},
			},
		},
		Interfaces: []models.InterfaceMetadata{
			{
				Name:       "UserServiceInterface",
				StructName: "UserService",
				Methods: []models.Method{
					{
						Name: "GetUser",
						Parameters: []models.Parameter{
							{Name: "id", Type: "int"},
						},
						Returns: []string{"*User", "error"},
					},
				},
			},
		},
	}

	result, err := GenerateCoreServiceModule(metadata)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Check that interface and provider are included
	expectedContents := []string{
		"package services",
		"// UserServiceInterface is the interface for UserService",
		"type UserServiceInterface interface {",
		"GetUser(id int) (*User, error)",
		"func NewUserService(",
		"func NewUserServiceInterface(impl *UserService) UserServiceInterface {",
		"fx.Provide(NewUserService),",
		"fx.Provide(NewUserServiceInterface),",
	}

	for _, expected := range expectedContents {
		if !strings.Contains(result, expected) {
			t.Errorf("generated module missing expected content: %s\n\nGenerated:\n%s", expected, result)
		}
	}
}

func TestGenerateParameterBindingCode_WithContextParameters(t *testing.T) {
	tests := []struct {
		name        string
		parameters  []models.Parameter
		expected    string
		expectError bool
	}{
		{
			name: "path parameters only",
			parameters: []models.Parameter{
				{
					Name:     "id",
					Type:     "int",
					Source:   models.ParameterSourcePath,
					Required: true,
				},
				{
					Name:     "slug",
					Type:     "string",
					Source:   models.ParameterSourcePath,
					Required: true,
				},
			},
			expected: `		id, err := axon.ParseInt(c, c.Param("id"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Invalid id: %v", err))
		}
		slug, err := axon.ParseString(c, c.Param("slug"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Invalid slug: %v", err))
		}
`,
			expectError: false,
		},
		{
			name: "context parameters only",
			parameters: []models.Parameter{
				{
					Name:     "c",
					Type:     "echo.Context",
					Source:   models.ParameterSourceContext,
					Required: true,
					Position: 0,
				},
			},
			expected: "",
			expectError: false,
		},
		{
			name: "mixed parameters",
			parameters: []models.Parameter{
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
			expected: `		id, err := axon.ParseInt(c, c.Param("id"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Invalid id: %v", err))
		}
`,
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			registry := createTestParserRegistry()
			result, err := GenerateParameterBindingCode(tt.parameters, registry)

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

			if result != tt.expected {
				t.Errorf("expected:\n%s\ngot:\n%s", tt.expected, result)
			}
		})
	}
}