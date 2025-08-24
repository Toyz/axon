package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCrossPackageParserDiscovery(t *testing.T) {
	// Create temporary directory structure
	tempDir, err := os.MkdirTemp("", "axon-parser-test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Create parsers package
	parsersDir := filepath.Join(tempDir, "parsers")
	err = os.MkdirAll(parsersDir, 0755)
	require.NoError(t, err)

	// Create parser file with UUID parser
	parserFile := filepath.Join(parsersDir, "uuid_parser.go")
	parserContent := `package parsers

import (
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

//axon::route_parser MyUUID
func ParseMyUUID(c echo.Context, paramValue string) (MyUUID, error) {
	parsed, err := uuid.Parse(paramValue)
	return MyUUID(parsed), err
}

type MyUUID uuid.UUID

//axon::route_parser CustomID
func ParseCustomID(c echo.Context, paramValue string) (CustomID, error) {
	return CustomID(paramValue), nil
}

type CustomID string
`
	err = os.WriteFile(parserFile, []byte(parserContent), 0644)
	require.NoError(t, err)

	// Create controllers package
	controllersDir := filepath.Join(tempDir, "controllers")
	err = os.MkdirAll(controllersDir, 0755)
	require.NoError(t, err)

	// Create controller file that uses the parsers
	controllerFile := filepath.Join(controllersDir, "user_controller.go")
	controllerContent := `package controllers

import (
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

//axon::controller
type UserController struct{}

//axon::route GET /users/{id:MyUUID}
func (uc *UserController) GetUser(id MyUUID) (string, error) {
	return "user", nil
}

//axon::route GET /custom/{id:CustomID}
func (uc *UserController) GetCustom(id CustomID) (string, error) {
	return "custom", nil
}

type CustomID string
`
	err = os.WriteFile(controllerFile, []byte(controllerContent), 0644)
	require.NoError(t, err)

	// Create go.mod file
	goModFile := filepath.Join(tempDir, "go.mod")
	goModContent := `module test-app

go 1.21
`
	err = os.WriteFile(goModFile, []byte(goModContent), 0644)
	require.NoError(t, err)

	// Change to temp directory for testing
	originalDir, err := os.Getwd()
	require.NoError(t, err)
	defer os.Chdir(originalDir)
	err = os.Chdir(tempDir)
	require.NoError(t, err)

	// Create generator and run discovery
	generator := NewGenerator()
	config := Config{
		Directories: []string{"parsers", "controllers"},
		ModuleName:  "test-app",
	}

	err = generator.Run(config)
	require.NoError(t, err)

	// Verify parsers were discovered
	assert.Len(t, generator.globalParsers, 2, "Should discover 2 parsers")

	// Check MyUUID parser
	uuidParser, exists := generator.globalParsers["MyUUID"]
	assert.True(t, exists, "MyUUID parser should be discovered")
	assert.Equal(t, "ParseMyUUID", uuidParser.FunctionName)
	assert.Contains(t, uuidParser.PackagePath, "parsers")

	// Check CustomID parser
	customParser, exists := generator.globalParsers["CustomID"]
	assert.True(t, exists, "CustomID parser should be discovered")
	assert.Equal(t, "ParseCustomID", customParser.FunctionName)
	assert.Contains(t, customParser.PackagePath, "parsers")

	// Verify generated controller module includes parser imports
	controllerModuleFile := filepath.Join(controllersDir, "autogen_module.go")
	assert.FileExists(t, controllerModuleFile, "Controller module should be generated")

	// Read generated content
	generatedContent, err := os.ReadFile(controllerModuleFile)
	require.NoError(t, err)
	content := string(generatedContent)

	// Verify imports include parser package
	assert.Contains(t, content, "test-app/parsers", "Generated module should import parser package")

	// Verify parser calls are generated in route wrappers
	assert.Contains(t, content, "parsers.ParseMyUUID", "Generated wrapper should call MyUUID parser")
	assert.Contains(t, content, "parsers.ParseCustomID", "Generated wrapper should call CustomID parser")
}

func TestParserConflictDetection(t *testing.T) {
	// Create temporary directory structure
	tempDir, err := os.MkdirTemp("", "axon-conflict-test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Create first parsers package
	parsers1Dir := filepath.Join(tempDir, "parsers1")
	err = os.MkdirAll(parsers1Dir, 0755)
	require.NoError(t, err)

	parser1File := filepath.Join(parsers1Dir, "parser.go")
	parser1Content := `package parsers1

import "github.com/labstack/echo/v4"

//axon::route_parser CustomType
func ParseCustomType(c echo.Context, paramValue string) (CustomType, error) {
	return CustomType(paramValue), nil
}

type CustomType string
`
	err = os.WriteFile(parser1File, []byte(parser1Content), 0644)
	require.NoError(t, err)

	// Create second parsers package with conflicting parser
	parsers2Dir := filepath.Join(tempDir, "parsers2")
	err = os.MkdirAll(parsers2Dir, 0755)
	require.NoError(t, err)

	parser2File := filepath.Join(parsers2Dir, "parser.go")
	parser2Content := `package parsers2

import "github.com/labstack/echo/v4"

//axon::route_parser CustomType
func ParseCustomType(c echo.Context, paramValue string) (CustomType, error) {
	return CustomType(paramValue), nil
}

type CustomType string
`
	err = os.WriteFile(parser2File, []byte(parser2Content), 0644)
	require.NoError(t, err)

	// Create go.mod file
	goModFile := filepath.Join(tempDir, "go.mod")
	goModContent := `module conflict-test

go 1.21
`
	err = os.WriteFile(goModFile, []byte(goModContent), 0644)
	require.NoError(t, err)

	// Change to temp directory for testing
	originalDir, err := os.Getwd()
	require.NoError(t, err)
	defer os.Chdir(originalDir)
	err = os.Chdir(tempDir)
	require.NoError(t, err)

	// Create generator and run discovery - should fail with conflict
	generator := NewGenerator()
	config := Config{
		Directories: []string{"parsers1", "parsers2"},
		ModuleName:  "conflict-test",
	}

	err = generator.Run(config)
	assert.Error(t, err, "Should detect parser conflict")
	assert.Contains(t, err.Error(), "parser conflict", "Error should mention parser conflict")
	assert.Contains(t, err.Error(), "CustomType", "Error should mention the conflicting type")
}

func TestParserImportPathResolution(t *testing.T) {
	// Create temporary directory structure
	tempDir, err := os.MkdirTemp("", "axon-import-test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Create nested parsers package
	parsersDir := filepath.Join(tempDir, "internal", "parsers")
	err = os.MkdirAll(parsersDir, 0755)
	require.NoError(t, err)

	parserFile := filepath.Join(parsersDir, "parser.go")
	parserContent := `package parsers

import "github.com/labstack/echo/v4"

//axon::route_parser TestType
func ParseTestType(c echo.Context, paramValue string) (TestType, error) {
	return TestType(paramValue), nil
}

type TestType string
`
	err = os.WriteFile(parserFile, []byte(parserContent), 0644)
	require.NoError(t, err)

	// Create go.mod file
	goModFile := filepath.Join(tempDir, "go.mod")
	goModContent := `module import-test

go 1.21
`
	err = os.WriteFile(goModFile, []byte(goModContent), 0644)
	require.NoError(t, err)

	// Change to temp directory for testing
	originalDir, err := os.Getwd()
	require.NoError(t, err)
	defer os.Chdir(originalDir)
	err = os.Chdir(tempDir)
	require.NoError(t, err)

	// Create generator and run discovery
	generator := NewGenerator()
	config := Config{
		Directories: []string{"internal/parsers"},
		ModuleName:  "import-test",
	}

	err = generator.Run(config)
	require.NoError(t, err)

	// Verify parser import path is correctly resolved
	parser, exists := generator.globalParsers["TestType"]
	assert.True(t, exists, "Parser should be discovered")
	assert.Contains(t, parser.PackagePath, "parsers", "Package path should contain parsers directory")
}