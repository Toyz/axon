package internal

import (
	"strings"
	"testing"

	"github.com/toyz/axon/internal/parser"
	"github.com/toyz/axon/internal/templates"
)

// TestInterfaceGenerationIntegration tests the complete interface generation workflow
func TestInterfaceGenerationIntegration(t *testing.T) {
	source := `package services

import (
	"context"
	"go.uber.org/fx"
)

//axon::core
//axon::interface
type UserService struct {
	fx.In
	Repository UserRepository
	Logger     *Logger
}

func (s *UserService) GetUser(id int) (*User, error) {
	return s.Repository.FindByID(id)
}

func (s *UserService) CreateUser(user User) (*User, error) {
	return s.Repository.Create(user)
}

func (s *UserService) ListUsers(ctx context.Context, limit int) ([]User, error) {
	return s.Repository.List(ctx, limit)
}

func (s *UserService) privateMethod() string {
	return "private"
}

//axon::controller
//axon::interface
type UserController struct {
	fx.In
	UserService UserServiceInterface
}

func (c *UserController) GetUser(id int) (*User, error) {
	return c.UserService.GetUser(id)
}

func (c *UserController) CreateUser(user User) (*User, error) {
	return c.UserService.CreateUser(user)
}`

	// Parse the source
	p := parser.NewParser()
	metadata, err := p.ParseSource("services.go", source)
	if err != nil {
		t.Fatalf("failed to parse source: %v", err)
	}

	// Verify metadata was created correctly
	if len(metadata.CoreServices) != 1 {
		t.Errorf("expected 1 core service, got %d", len(metadata.CoreServices))
	}

	if len(metadata.Controllers) != 1 {
		t.Errorf("expected 1 controller, got %d", len(metadata.Controllers))
	}

	if len(metadata.Interfaces) != 2 {
		t.Errorf("expected 2 interfaces, got %d", len(metadata.Interfaces))
	}

	// Generate the complete module
	moduleCode, err := templates.GenerateCoreServiceModule(metadata)
	if err != nil {
		t.Fatalf("failed to generate module: %v", err)
	}

	// Verify the generated code contains expected elements
	expectedElements := []string{
		"package services",
		"// UserServiceInterface is the interface for UserService",
		"type UserServiceInterface interface {",
		"GetUser(id int) (*User, error)",
		"CreateUser(user User) (*User, error)",
		"ListUsers(ctx context.Context, limit int) ([]User, error)",
		"// UserControllerInterface is the interface for UserController",
		"type UserControllerInterface interface {",
		"func NewUserServiceInterface(impl *UserService) UserServiceInterface {",
		"func NewUserControllerInterface(impl *UserController) UserControllerInterface {",
		"fx.Provide(NewUserService),",
		"fx.Provide(NewUserServiceInterface),",
		"fx.Provide(NewUserControllerInterface),",
	}

	for _, expected := range expectedElements {
		if !strings.Contains(moduleCode, expected) {
			t.Errorf("generated module missing expected element: %s\n\nGenerated code:\n%s", expected, moduleCode)
		}
	}

	// Verify that private methods are not included in interfaces
	if strings.Contains(moduleCode, "privateMethod") {
		t.Errorf("private method should not be included in interface")
	}

	// Verify interface methods are correctly formatted
	userServiceInterface := extractInterfaceDefinition(moduleCode, "UserServiceInterface")
	if userServiceInterface == "" {
		t.Errorf("could not find UserServiceInterface definition")
	} else {
		// Check that all public methods are included
		publicMethods := []string{"GetUser", "CreateUser", "ListUsers"}
		for _, method := range publicMethods {
			if !strings.Contains(userServiceInterface, method) {
				t.Errorf("UserServiceInterface missing method: %s", method)
			}
		}
	}

	t.Logf("Generated module code:\n%s", moduleCode)
}

// extractInterfaceDefinition extracts the interface definition from generated code
func extractInterfaceDefinition(code, interfaceName string) string {
	start := strings.Index(code, "type "+interfaceName+" interface {")
	if start == -1 {
		return ""
	}

	// Find the matching closing brace
	braceCount := 0
	i := start
	for i < len(code) {
		if code[i] == '{' {
			braceCount++
		} else if code[i] == '}' {
			braceCount--
			if braceCount == 0 {
				return code[start : i+1]
			}
		}
		i++
	}

	return ""
}
