package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/toyz/axon/internal/cli"
)

func TestCleanAutogenFiles(t *testing.T) {
	// Create temporary directory structure
	tempDir, err := os.MkdirTemp("", "axon_clean_test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Create directory structure with autogen files
	dirs := []string{
		"controllers",
		"services",
		"middleware",
		"nested/deep/controllers",
	}

	var autogenFiles []string
	for _, dir := range dirs {
		dirPath := filepath.Join(tempDir, dir)
		require.NoError(t, os.MkdirAll(dirPath, 0755))

		autogenFile := filepath.Join(dirPath, "autogen_module.go")
		require.NoError(t, os.WriteFile(autogenFile, []byte("package test\n// Generated file"), 0644))
		autogenFiles = append(autogenFiles, autogenFile)
	}

	// Create some non-autogen files that should not be deleted
	nonAutogenFiles := []string{
		filepath.Join(tempDir, "controllers", "user_controller.go"),
		filepath.Join(tempDir, "services", "user_service.go"),
		filepath.Join(tempDir, "main.go"),
	}

	for _, file := range nonAutogenFiles {
		require.NoError(t, os.WriteFile(file, []byte("package test\n// Regular file"), 0644))
	}

	// Change to temp directory for relative path testing
	originalDir, err := os.Getwd()
	require.NoError(t, err)
	defer os.Chdir(originalDir)
	require.NoError(t, os.Chdir(tempDir))

	t.Run("clean recursive pattern", func(t *testing.T) {
		// Test cleaning with recursive pattern
		cleaner := cli.NewCleaner()
		err := cleaner.CleanGeneratedFiles([]string{"./..."})
		assert.NoError(t, err)

		// Verify autogen files are deleted
		for _, file := range autogenFiles {
			assert.NoFileExists(t, file, "Autogen file should be deleted: %s", file)
		}

		// Verify non-autogen files still exist
		for _, file := range nonAutogenFiles {
			assert.FileExists(t, file, "Non-autogen file should still exist: %s", file)
		}
	})
}

func TestCleanAutogenFilesSpecificDirectory(t *testing.T) {
	// Create temporary directory structure
	tempDir, err := os.MkdirTemp("", "axon_clean_specific_test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Create directory structure
	controllersDir := filepath.Join(tempDir, "controllers")
	servicesDir := filepath.Join(tempDir, "services")
	require.NoError(t, os.MkdirAll(controllersDir, 0755))
	require.NoError(t, os.MkdirAll(servicesDir, 0755))

	// Create autogen files
	controllerAutogen := filepath.Join(controllersDir, "autogen_module.go")
	serviceAutogen := filepath.Join(servicesDir, "autogen_module.go")
	require.NoError(t, os.WriteFile(controllerAutogen, []byte("package controllers"), 0644))
	require.NoError(t, os.WriteFile(serviceAutogen, []byte("package services"), 0644))

	t.Run("clean specific directory only", func(t *testing.T) {
		// Clean only controllers directory
		cleaner := cli.NewCleaner()
		err := cleaner.CleanGeneratedFiles([]string{controllersDir})
		assert.NoError(t, err)

		// Verify only controllers autogen file is deleted
		assert.NoFileExists(t, controllerAutogen, "Controllers autogen file should be deleted")
		assert.FileExists(t, serviceAutogen, "Services autogen file should still exist")
	})
}

func TestCleanAutogenFilesNoFiles(t *testing.T) {
	// Create temporary directory with no autogen files
	tempDir, err := os.MkdirTemp("", "axon_clean_empty_test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Create some regular files
	require.NoError(t, os.WriteFile(filepath.Join(tempDir, "main.go"), []byte("package main"), 0644))

	t.Run("clean directory with no autogen files", func(t *testing.T) {
		cleaner := cli.NewCleaner()
		err := cleaner.CleanGeneratedFiles([]string{tempDir})
		assert.NoError(t, err)
	})
}

func TestFindAutogenFilesRecursive(t *testing.T) {
	// Create temporary directory structure
	tempDir, err := os.MkdirTemp("", "axon_find_test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Create nested directory structure
	dirs := []string{
		"controllers",
		"services/user",
		"middleware",
		".hidden",    // Should be skipped
		"vendor/pkg", // Should be skipped
	}

	expectedFiles := []string{}
	for _, dir := range dirs {
		if dir == ".hidden" || dir == "vendor/pkg" {
			continue // These should be skipped
		}

		dirPath := filepath.Join(tempDir, dir)
		require.NoError(t, os.MkdirAll(dirPath, 0755))

		autogenFile := filepath.Join(dirPath, "autogen_module.go")
		require.NoError(t, os.WriteFile(autogenFile, []byte("package test"), 0644))
		expectedFiles = append(expectedFiles, autogenFile)
	}

	// Create files in directories that should be skipped
	hiddenDir := filepath.Join(tempDir, ".hidden")
	require.NoError(t, os.MkdirAll(hiddenDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(hiddenDir, "autogen_module.go"), []byte("package hidden"), 0644))

	vendorDir := filepath.Join(tempDir, "vendor/pkg")
	require.NoError(t, os.MkdirAll(vendorDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(vendorDir, "autogen_module.go"), []byte("package vendor"), 0644))

	t.Run("clean autogen files recursively", func(t *testing.T) {
		// Test that cleaner can handle complex directory structures
		cleaner := cli.NewCleaner()
		err := cleaner.CleanGeneratedFiles([]string{tempDir + "/..."})
		assert.NoError(t, err)

		// Just verify no error occurred - the cleaner should handle the complex structure
		assert.NoError(t, err)
	})
}
