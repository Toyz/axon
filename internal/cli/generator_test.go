package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/toyz/axon/internal/models"
)

func TestGenerator_Run(t *testing.T) {
	// Create temporary directory structure for testing
	tempDir, err := os.MkdirTemp("", "axon_generator_test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Create go.mod file
	goModContent := `module github.com/example/testapp

go 1.21

require (
	github.com/labstack/echo/v4 v4.11.1
	go.uber.org/fx v1.20.0
)
`
	require.NoError(t, os.WriteFile(filepath.Join(tempDir, "go.mod"), []byte(goModContent), 0644))

	// Create test package structure
	controllersDir := filepath.Join(tempDir, "internal", "controllers")
	servicesDir := filepath.Join(tempDir, "internal", "services")
	require.NoError(t, os.MkdirAll(controllersDir, 0755))
	require.NoError(t, os.MkdirAll(servicesDir, 0755))

	// Create controller file with annotations
	controllerContent := `package controllers

import (
	"go.uber.org/fx"
)

//axon::controller
type UserController struct {
	fx.In
}

//axon::route GET /users/{id:int}
func (c *UserController) GetUser(id int) (string, error) {
	return "user", nil
}
`
	require.NoError(t, os.WriteFile(filepath.Join(controllersDir, "user_controller.go"), []byte(controllerContent), 0644))

	// Create service file with annotations
	serviceContent := `package services

import "context"

//axon::core -Init
type UserService struct{}

func (s *UserService) Start(ctx context.Context) error {
	return nil
}
`
	require.NoError(t, os.WriteFile(filepath.Join(servicesDir, "user_service.go"), []byte(serviceContent), 0644))

	// Create package without annotations
	modelsDir := filepath.Join(tempDir, "internal", "models")
	require.NoError(t, os.MkdirAll(modelsDir, 0755))
	modelContent := `package models

type User struct {
	ID   int    ` + "`json:\"id\"`" + `
	Name string ` + "`json:\"name\"`" + `
}
`
	require.NoError(t, os.WriteFile(filepath.Join(modelsDir, "user.go"), []byte(modelContent), 0644))

	// Change to temp directory for relative path resolution
	originalDir, err := os.Getwd()
	require.NoError(t, err)
	defer os.Chdir(originalDir)
	require.NoError(t, os.Chdir(tempDir))

	generator := NewGenerator(false)

	t.Run("generate modules", func(t *testing.T) {
		config := Config{
			Directories: []string{"./internal"},
			ModuleName:  "",
		}

		err := generator.Run(config)
		require.NoError(t, err)

		// Check that autogen_module.go files were created
		controllerModulePath := filepath.Join(controllersDir, "autogen_module.go")
		assert.FileExists(t, controllerModulePath)

		serviceModulePath := filepath.Join(servicesDir, "autogen_module.go")
		assert.FileExists(t, serviceModulePath)

		// Check that no module was created for models (no annotations)
		modelModulePath := filepath.Join(modelsDir, "autogen_module.go")
		assert.NoFileExists(t, modelModulePath)

		// Verify controller module content
		controllerModuleContent, err := os.ReadFile(controllerModulePath)
		require.NoError(t, err)
		controllerModuleStr := string(controllerModuleContent)
		assert.Contains(t, controllerModuleStr, "package controllers")
		assert.Contains(t, controllerModuleStr, "NewUserController")
		assert.Contains(t, controllerModuleStr, "RegisterRoutes")
		assert.Contains(t, controllerModuleStr, "AutogenModule")

		// Verify service module content
		serviceModuleContent, err := os.ReadFile(serviceModulePath)
		require.NoError(t, err)
		serviceModuleStr := string(serviceModuleContent)
		assert.Contains(t, serviceModuleStr, "package services")
		assert.Contains(t, serviceModuleStr, "NewUserService")
		assert.Contains(t, serviceModuleStr, "AutogenModule")
	})

	t.Run("generate with custom module name", func(t *testing.T) {
		// Clean up previous autogen files
		os.Remove(filepath.Join(controllersDir, "autogen_module.go"))
		os.Remove(filepath.Join(servicesDir, "autogen_module.go"))

		config := Config{
			Directories: []string{"./internal"},
			ModuleName:  "github.com/custom/myapp",
		}

		err := generator.Run(config)
		require.NoError(t, err)

		// Verify modules were created with custom module context
		controllerModulePath := filepath.Join(controllersDir, "autogen_module.go")
		assert.FileExists(t, controllerModulePath)

		serviceModulePath := filepath.Join(servicesDir, "autogen_module.go")
		assert.FileExists(t, serviceModulePath)
	})



	t.Run("no packages found", func(t *testing.T) {
		emptyDir := filepath.Join(tempDir, "empty")
		require.NoError(t, os.MkdirAll(emptyDir, 0755))

		config := Config{
			Directories: []string{emptyDir},
			ModuleName:  "",
		}

		err := generator.Run(config)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "No Go packages found")
	})

	t.Run("no annotations found", func(t *testing.T) {
		// Only scan models directory (no annotations)
		config := Config{
			Directories: []string{"./internal/models"},
			ModuleName:  "",
		}

		err := generator.Run(config)
		require.NoError(t, err) // Should succeed but generate no modules

		// Verify no autogen files were created in models
		modelModulePath := filepath.Join(modelsDir, "autogen_module.go")
		assert.NoFileExists(t, modelModulePath)
	})
}

func TestGenerator_hasNoAnnotations(t *testing.T) {
	generator := NewGenerator(false)

	testCases := []struct {
		name     string
		metadata func() *models.PackageMetadata
		expected bool
	}{
		{
			name: "no annotations",
			metadata: func() *models.PackageMetadata {
				return &models.PackageMetadata{
					PackageName:  "test",
					Controllers:  []models.ControllerMetadata{},
					CoreServices: []models.CoreServiceMetadata{},
					Middlewares:  []models.MiddlewareMetadata{},
					Interfaces:   []models.InterfaceMetadata{},
				}
			},
			expected: true,
		},
		{
			name: "has controllers",
			metadata: func() *models.PackageMetadata {
				return &models.PackageMetadata{
					PackageName: "test",
					Controllers: []models.ControllerMetadata{
						{BaseMetadata: models.BaseMetadata{Name: "TestController"}},
					},
				}
			},
			expected: false,
		},
		{
			name: "has core services",
			metadata: func() *models.PackageMetadata {
				return &models.PackageMetadata{
					PackageName: "test",
					CoreServices: []models.CoreServiceMetadata{
						{BaseMetadata: models.BaseMetadata{Name: "TestService"}},
					},
				}
			},
			expected: false,
		},
		{
			name: "has middlewares",
			metadata: func() *models.PackageMetadata {
				return &models.PackageMetadata{
					PackageName: "test",
					Middlewares: []models.MiddlewareMetadata{
						{BaseMetadata: models.BaseMetadata{Name: "TestMiddleware"}},
					},
				}
			},
			expected: false,
		},
		{
			name: "has interfaces",
			metadata: func() *models.PackageMetadata {
				return &models.PackageMetadata{
					PackageName: "test",
					Interfaces: []models.InterfaceMetadata{
						{BaseMetadata: models.BaseMetadata{Name: "TestInterface"}},
					},
				}
			},
			expected: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := generator.hasNoAnnotations(tc.metadata())
			assert.Equal(t, tc.expected, result)
		})
	}
}