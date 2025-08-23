package main

import (
	"os"
	"os/exec"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/toyz/axon/internal/cli"
)

// TestExampleApplicationGeneration tests code generation for the complete example app
func TestExampleApplicationGeneration(t *testing.T) {
	// Run code generation
	generator := cli.NewGenerator()
	config := cli.Config{
		Directories: []string{"./internal/middleware", "./internal/services", "./internal/controllers", "./internal/interfaces"},
		ModuleName:  "github.com/toyz/axon/examples/complete-app",
	}

	err := generator.Run(config)
	require.NoError(t, err, "Code generation should succeed")

	// Verify generated files exist and have correct content
	testGeneratedFile(t, "internal/controllers/autogen_module.go", []string{
		"package controllers",
		"UserController",
		"HealthController",
		"fx.Provide",
		"AutogenModule",
	})

	testGeneratedFile(t, "internal/services/autogen_module.go", []string{
		"package services",
		"UserService",
		"DatabaseService",
		"fx.Invoke", // Lifecycle services use fx.Invoke, not fx.Provide
		"AutogenModule",
	})

	testGeneratedFile(t, "internal/middleware/autogen_module.go", []string{
		"package middleware",
		"LoggingMiddleware",
		"AuthMiddleware",
		"fx.Provide",
		"AutogenModule",
	})

	testGeneratedFile(t, "internal/interfaces/autogen_module.go", []string{
		"package interfaces",
		"UserRepository",
		"fx.Provide",
		"AutogenModule",
	})
}

func testGeneratedFile(t *testing.T, filePath string, expectedContent []string) {
	// Check file exists
	_, err := os.Stat(filePath)
	require.NoError(t, err, "Generated file should exist: %s", filePath)

	// Read file content
	content, err := os.ReadFile(filePath)
	require.NoError(t, err, "Should be able to read generated file: %s", filePath)

	contentStr := string(content)

	// Verify expected content
	for _, expected := range expectedContent {
		assert.Contains(t, contentStr, expected, "Generated file %s should contain: %s", filePath, expected)
	}

	t.Logf("Generated file %s verified successfully", filePath)
}

// TestExampleApplicationCompilation tests that the generated code compiles
func TestExampleApplicationCompilation(t *testing.T) {
	// First run generation to ensure files exist
	generator := cli.NewGenerator()
	config := cli.Config{
		Directories: []string{"./internal/middleware", "./internal/services", "./internal/controllers", "./internal/interfaces"},
		ModuleName:  "github.com/toyz/axon/examples/complete-app",
	}

	err := generator.Run(config)
	require.NoError(t, err)

	// Add any missing dependencies
	cmd := exec.Command("go", "mod", "tidy")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Logf("go mod tidy output: %s", output)
	}

	// Try to compile the generated modules
	cmd = exec.Command("go", "build", "./internal/...")
	output, err = cmd.CombinedOutput()
	require.NoError(t, err, "Generated modules should compile: %s", output)

	t.Log("All generated modules compiled successfully")
}

// TestRouteGeneration tests that routes are generated correctly
func TestRouteGeneration(t *testing.T) {
	// Run generation - process middleware first, then controllers
	generator := cli.NewGenerator()
	config := cli.Config{
		Directories: []string{"./internal/middleware", "./internal/controllers"},
		ModuleName:  "github.com/toyz/axon/examples/complete-app",
	}

	err := generator.Run(config)
	require.NoError(t, err)

	// Read the generated controller module
	content, err := os.ReadFile("internal/controllers/autogen_module.go")
	require.NoError(t, err)

	contentStr := string(content)

	// Verify route wrappers are generated
	expectedRoutes := []string{
		"wrapUserControllerGetAllUsers",
		"wrapUserControllerGetUser", 
		"wrapUserControllerCreateUser",
		"wrapUserControllerDeleteUser",
		"wrapHealthControllerGetHealth",
	}

	for _, route := range expectedRoutes {
		assert.Contains(t, contentStr, route, "Should generate wrapper for route: %s", route)
	}

	// Verify middleware application
	assert.Contains(t, contentStr, "LoggingMiddleware", "Should reference LoggingMiddleware")
	assert.Contains(t, contentStr, "AuthMiddleware", "Should reference AuthMiddleware")

	// Verify parameter parsing
	assert.Contains(t, contentStr, "strconv.Atoi", "Should generate parameter parsing code")

	t.Log("Route generation verified successfully")
}

// TestMiddlewareGeneration tests that middleware is generated correctly
func TestMiddlewareGeneration(t *testing.T) {
	// Run generation
	generator := cli.NewGenerator()
	config := cli.Config{
		Directories: []string{"./internal/middleware"},
		ModuleName:  "github.com/toyz/axon/examples/complete-app",
	}

	err := generator.Run(config)
	require.NoError(t, err)

	// Read the generated middleware module
	content, err := os.ReadFile("internal/middleware/autogen_module.go")
	require.NoError(t, err)

	contentStr := string(content)

	// Verify middleware providers are generated
	assert.Contains(t, contentStr, "NewLoggingMiddleware", "Should generate LoggingMiddleware provider")
	assert.Contains(t, contentStr, "NewAuthMiddleware", "Should generate AuthMiddleware provider")

	// Verify middleware registration
	assert.Contains(t, contentStr, "axon.RegisterMiddleware", "Should register middlewares with axon registry")

	t.Log("Middleware generation verified successfully")
}

// TestServiceGeneration tests that services are generated correctly
func TestServiceGeneration(t *testing.T) {
	// Run generation
	generator := cli.NewGenerator()
	config := cli.Config{
		Directories: []string{"./internal/services"},
		ModuleName:  "github.com/toyz/axon/examples/complete-app",
	}

	err := generator.Run(config)
	require.NoError(t, err)

	// Read the generated services module
	content, err := os.ReadFile("internal/services/autogen_module.go")
	require.NoError(t, err)

	contentStr := string(content)

	// Verify service providers are generated
	assert.Contains(t, contentStr, "NewUserService", "Should generate UserService provider")
	assert.Contains(t, contentStr, "NewDatabaseService", "Should generate DatabaseService provider")

	// Verify lifecycle hooks for services with -Init flag
	assert.Contains(t, contentStr, "fx.Lifecycle", "Should include lifecycle management")
	assert.Contains(t, contentStr, "OnStart", "Should register OnStart hooks")
	assert.Contains(t, contentStr, "OnStop", "Should register OnStop hooks")

	t.Log("Service generation verified successfully")
}

// TestCleanup removes generated files after tests
func TestCleanup(t *testing.T) {
	// Remove generated files
	generatedFiles := []string{
		"internal/controllers/autogen_module.go",
		"internal/services/autogen_module.go", 
		"internal/middleware/autogen_module.go",
		"internal/interfaces/autogen_module.go",
	}

	for _, file := range generatedFiles {
		if _, err := os.Stat(file); err == nil {
			err = os.Remove(file)
			if err != nil {
				t.Logf("Warning: Could not remove generated file %s: %v", file, err)
			} else {
				t.Logf("Cleaned up generated file: %s", file)
			}
		}
	}
}