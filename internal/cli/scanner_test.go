package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDirectoryScanner_ScanDirectories(t *testing.T) {
	// Create temporary directory structure for testing
	tempDir, err := os.MkdirTemp("", "axon_scanner_test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Create test directory structure
	// tempDir/
	//   ├── controllers/
	//   │   ├── user_controller.go
	//   │   └── auth_controller.go
	//   ├── services/
	//   │   ├── user_service.go
	//   │   └── subservice/
	//   │       └── helper.go
	//   ├── models/
	//   │   └── user.go
	//   ├── vendor/
	//   │   └── dependency.go (should be skipped)
	//   └── empty_dir/
	//       (no Go files)

	// Create directories
	controllersDir := filepath.Join(tempDir, "controllers")
	servicesDir := filepath.Join(tempDir, "services")
	subserviceDir := filepath.Join(servicesDir, "subservice")
	modelsDir := filepath.Join(tempDir, "models")
	vendorDir := filepath.Join(tempDir, "vendor")
	emptyDir := filepath.Join(tempDir, "empty_dir")

	require.NoError(t, os.MkdirAll(controllersDir, 0755))
	require.NoError(t, os.MkdirAll(subserviceDir, 0755))
	require.NoError(t, os.MkdirAll(modelsDir, 0755))
	require.NoError(t, os.MkdirAll(vendorDir, 0755))
	require.NoError(t, os.MkdirAll(emptyDir, 0755))

	// Create Go files
	goFiles := map[string]string{
		filepath.Join(controllersDir, "user_controller.go"): "package controllers\n\ntype UserController struct{}",
		filepath.Join(controllersDir, "auth_controller.go"): "package controllers\n\ntype AuthController struct{}",
		filepath.Join(servicesDir, "user_service.go"):      "package services\n\ntype UserService struct{}",
		filepath.Join(subserviceDir, "helper.go"):          "package subservice\n\ntype Helper struct{}",
		filepath.Join(modelsDir, "user.go"):                "package models\n\ntype User struct{}",
		filepath.Join(vendorDir, "dependency.go"):          "package vendor\n\ntype Dependency struct{}",
	}

	for filePath, content := range goFiles {
		require.NoError(t, os.WriteFile(filePath, []byte(content), 0644))
	}

	// Create test files (should be ignored)
	testFile := filepath.Join(controllersDir, "user_controller_test.go")
	require.NoError(t, os.WriteFile(testFile, []byte("package controllers\n\nfunc TestUser(t *testing.T) {}"), 0644))

	// Create autogen file (should be ignored)
	autogenFile := filepath.Join(servicesDir, "autogen_module.go")
	require.NoError(t, os.WriteFile(autogenFile, []byte("package services\n\n// Generated file"), 0644))

	scanner := NewDirectoryScanner()

	t.Run("scan single directory", func(t *testing.T) {
		dirs, err := scanner.ScanDirectories([]string{controllersDir})
		require.NoError(t, err)
		assert.Len(t, dirs, 1)
		assert.Contains(t, dirs, controllersDir)
	})

	t.Run("scan multiple directories", func(t *testing.T) {
		dirs, err := scanner.ScanDirectories([]string{controllersDir, servicesDir})
		require.NoError(t, err)
		assert.Len(t, dirs, 3) // controllers, services, services/subservice
		assert.Contains(t, dirs, controllersDir)
		assert.Contains(t, dirs, servicesDir)
		assert.Contains(t, dirs, subserviceDir)
	})

	t.Run("scan root directory recursively", func(t *testing.T) {
		dirs, err := scanner.ScanDirectories([]string{tempDir})
		require.NoError(t, err)
		
		// Should find controllers, services, subservice, and models
		// Should NOT find vendor (skipped) or empty_dir (no Go files)
		assert.Len(t, dirs, 4)
		assert.Contains(t, dirs, controllersDir)
		assert.Contains(t, dirs, servicesDir)
		assert.Contains(t, dirs, subserviceDir)
		assert.Contains(t, dirs, modelsDir)
		assert.NotContains(t, dirs, vendorDir)
		assert.NotContains(t, dirs, emptyDir)
	})

	t.Run("scan with Go-style recursive pattern ./...", func(t *testing.T) {
		// Change to temp directory for relative path testing
		originalDir, err := os.Getwd()
		require.NoError(t, err)
		defer os.Chdir(originalDir)
		require.NoError(t, os.Chdir(tempDir))

		dirs, err := scanner.ScanDirectories([]string{"./..."})
		require.NoError(t, err)
		
		// Should find all the same directories as recursive scan
		assert.Len(t, dirs, 4)
		// Convert to relative paths for comparison
		for _, dir := range dirs {
			relDir, err := filepath.Rel(tempDir, dir)
			require.NoError(t, err)
			
			switch relDir {
			case "controllers", "services", "services/subservice", "models":
				// Expected directories
			default:
				t.Errorf("Unexpected directory found: %s", relDir)
			}
		}
	})

	t.Run("scan with specific subdirectory pattern", func(t *testing.T) {
		// Change to temp directory for relative path testing
		originalDir, err := os.Getwd()
		require.NoError(t, err)
		defer os.Chdir(originalDir)
		require.NoError(t, os.Chdir(tempDir))

		dirs, err := scanner.ScanDirectories([]string{"./services/..."})
		require.NoError(t, err)
		
		// Should find services and subservice directories
		assert.Len(t, dirs, 2)
		for _, dir := range dirs {
			relDir, err := filepath.Rel(tempDir, dir)
			require.NoError(t, err)
			
			assert.True(t, relDir == "services" || relDir == "services/subservice", 
				"Expected services or services/subservice, got %s", relDir)
		}
	})

	t.Run("nonexistent directory", func(t *testing.T) {
		_, err := scanner.ScanDirectories([]string{"/nonexistent/path"})
		assert.Error(t, err)
	})
}

func TestDirectoryScanner_hasGoFiles(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "axon_scanner_hasgo_test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	scanner := NewDirectoryScanner()

	t.Run("directory with Go files", func(t *testing.T) {
		goFile := filepath.Join(tempDir, "main.go")
		require.NoError(t, os.WriteFile(goFile, []byte("package main"), 0644))

		hasGo, err := scanner.hasGoFiles(tempDir)
		require.NoError(t, err)
		assert.True(t, hasGo)
	})

	t.Run("directory with only test files", func(t *testing.T) {
		testDir := filepath.Join(tempDir, "testonly")
		require.NoError(t, os.MkdirAll(testDir, 0755))
		
		testFile := filepath.Join(testDir, "main_test.go")
		require.NoError(t, os.WriteFile(testFile, []byte("package main"), 0644))

		hasGo, err := scanner.hasGoFiles(testDir)
		require.NoError(t, err)
		assert.False(t, hasGo)
	})

	t.Run("directory with only autogen files", func(t *testing.T) {
		autogenDir := filepath.Join(tempDir, "autogenonly")
		require.NoError(t, os.MkdirAll(autogenDir, 0755))
		
		autogenFile := filepath.Join(autogenDir, "autogen_module.go")
		require.NoError(t, os.WriteFile(autogenFile, []byte("package main"), 0644))

		hasGo, err := scanner.hasGoFiles(autogenDir)
		require.NoError(t, err)
		assert.False(t, hasGo)
	})

	t.Run("empty directory", func(t *testing.T) {
		emptyDir := filepath.Join(tempDir, "empty")
		require.NoError(t, os.MkdirAll(emptyDir, 0755))

		hasGo, err := scanner.hasGoFiles(emptyDir)
		require.NoError(t, err)
		assert.False(t, hasGo)
	})
}

func TestDirectoryScanner_shouldSkipDirectory(t *testing.T) {
	scanner := NewDirectoryScanner()

	testCases := []struct {
		name     string
		dirname  string
		expected bool
	}{
		{"vendor directory", "vendor", true},
		{"node_modules directory", "node_modules", true},
		{"git directory", ".git", true},
		{"hidden directory", ".hidden", true},
		{"testdata directory", "testdata", true},
		{"build directory", "build", true},
		{"normal directory", "controllers", false},
		{"normal directory with underscore", "user_service", false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := scanner.shouldSkipDirectory(tc.dirname)
			assert.Equal(t, tc.expected, result)
		})
	}
}