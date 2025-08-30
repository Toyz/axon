package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestModuleResolver_ResolveModuleName(t *testing.T) {
	resolver := NewModuleResolver()

	t.Run("custom module name provided", func(t *testing.T) {
		customModule := "github.com/custom/module"
		result, err := resolver.ResolveModuleName(customModule)
		require.NoError(t, err)
		assert.Equal(t, customModule, result)
	})

	t.Run("read from go.mod file", func(t *testing.T) {
		// Create temporary directory with go.mod
		tempDir, err := os.MkdirTemp("", "axon_resolver_test")
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
		goModPath := filepath.Join(tempDir, "go.mod")
		require.NoError(t, os.WriteFile(goModPath, []byte(goModContent), 0644))

		// Change to temp directory
		originalDir, err := os.Getwd()
		require.NoError(t, err)
		defer os.Chdir(originalDir)
		require.NoError(t, os.Chdir(tempDir))

		result, err := resolver.ResolveModuleName("")
		require.NoError(t, err)
		assert.Equal(t, "github.com/example/testapp", result)
	})

	t.Run("no go.mod file found", func(t *testing.T) {
		// Create temporary directory without go.mod
		tempDir, err := os.MkdirTemp("", "axon_resolver_nomod_test")
		require.NoError(t, err)
		defer os.RemoveAll(tempDir)

		// Change to temp directory
		originalDir, err := os.Getwd()
		require.NoError(t, err)
		defer os.Chdir(originalDir)
		require.NoError(t, os.Chdir(tempDir))

		_, err = resolver.ResolveModuleName("")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "go.mod file not found")
	})
}

func TestModuleResolver_parseGoModFile(t *testing.T) {
	resolver := NewModuleResolver()

	t.Run("valid go.mod file", func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "axon_parse_test")
		require.NoError(t, err)
		defer os.RemoveAll(tempDir)

		goModContent := `module github.com/example/myapp

go 1.21

require (
	github.com/labstack/echo/v4 v4.11.1
)
`
		goModPath := filepath.Join(tempDir, "go.mod")
		require.NoError(t, os.WriteFile(goModPath, []byte(goModContent), 0644))

		moduleName, err := resolver.parseGoModFile(goModPath)
		require.NoError(t, err)
		assert.Equal(t, "github.com/example/myapp", moduleName)
	})

	t.Run("go.mod without module declaration", func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "axon_parse_invalid_test")
		require.NoError(t, err)
		defer os.RemoveAll(tempDir)

		goModContent := `go 1.21

require (
	github.com/labstack/echo/v4 v4.11.1
)
`
		goModPath := filepath.Join(tempDir, "go.mod")
		require.NoError(t, os.WriteFile(goModPath, []byte(goModContent), 0644))

		_, err = resolver.parseGoModFile(goModPath)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no module declaration found")
	})

	t.Run("nonexistent file", func(t *testing.T) {
		_, err := resolver.parseGoModFile("/nonexistent/go.mod")
		assert.Error(t, err)
	})
}

func TestModuleResolver_BuildPackagePath(t *testing.T) {
	resolver := NewModuleResolver()

	// Get current working directory for tests
	currentDir, err := os.Getwd()
	require.NoError(t, err)

	testCases := []struct {
		name       string
		moduleName string
		packageDir string
		expected   string
	}{
		{
			name:       "current directory",
			moduleName: "github.com/example/app",
			packageDir: ".",
			expected:   "github.com/example/app",
		},
		{
			name:       "subdirectory",
			moduleName: "github.com/example/app",
			packageDir: "internal/controllers",
			expected:   "github.com/example/app/internal/controllers",
		},
		{
			name:       "nested subdirectory",
			moduleName: "github.com/example/app",
			packageDir: "internal/services/user",
			expected:   "github.com/example/app/internal/services/user",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Convert relative path to absolute for testing
			var packageDir string
			if tc.packageDir == "." {
				packageDir = currentDir
			} else {
				packageDir = filepath.Join(currentDir, tc.packageDir)
			}

			result, err := resolver.BuildPackagePath(tc.moduleName, packageDir)
			require.NoError(t, err)
			assert.Equal(t, tc.expected, result)
		})
	}

	t.Run("absolute path", func(t *testing.T) {
		// Create temporary directory
		tempDir, err := os.MkdirTemp("", "axon_buildpath_test")
		require.NoError(t, err)
		defer os.RemoveAll(tempDir)

		// Change to temp directory
		originalDir, err := os.Getwd()
		require.NoError(t, err)
		defer os.Chdir(originalDir)
		require.NoError(t, os.Chdir(tempDir))

		// Create subdirectory
		subDir := filepath.Join(tempDir, "internal", "controllers")
		require.NoError(t, os.MkdirAll(subDir, 0755))

		result, err := resolver.BuildPackagePath("github.com/example/app", subDir)
		require.NoError(t, err)
		assert.Equal(t, "github.com/example/app/internal/controllers", result)
	})
}
