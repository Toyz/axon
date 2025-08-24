package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDirectoryValidation(t *testing.T) {
	// Create temporary directory structure for testing
	tempDir, err := os.MkdirTemp("", "axon_main_test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Create test directory structure
	controllersDir := filepath.Join(tempDir, "internal", "controllers")
	servicesDir := filepath.Join(tempDir, "internal", "services")
	require.NoError(t, os.MkdirAll(controllersDir, 0755))
	require.NoError(t, os.MkdirAll(servicesDir, 0755))

	// Create test Go files
	controllerFile := filepath.Join(controllersDir, "test_controller.go")
	serviceFile := filepath.Join(servicesDir, "test_service.go")
	require.NoError(t, os.WriteFile(controllerFile, []byte("package controllers"), 0644))
	require.NoError(t, os.WriteFile(serviceFile, []byte("package services"), 0644))

	// Change to temp directory for relative path testing
	originalDir, err := os.Getwd()
	require.NoError(t, err)
	defer os.Chdir(originalDir)
	require.NoError(t, os.Chdir(tempDir))

	tests := []struct {
		name        string
		directories []string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "valid single directory",
			directories: []string{"./internal/controllers"},
			expectError: false,
		},
		{
			name:        "valid recursive pattern ./...",
			directories: []string{"./..."},
			expectError: false,
		},
		{
			name:        "valid subdirectory pattern",
			directories: []string{"./internal/..."},
			expectError: false,
		},
		{
			name:        "valid specific subdirectory pattern",
			directories: []string{"./internal/controllers/..."},
			expectError: false,
		},
		{
			name:        "nonexistent directory",
			directories: []string{"./nonexistent"},
			expectError: true,
			errorMsg:    "Directory does not exist: ./nonexistent",
		},
		{
			name:        "nonexistent base directory in pattern",
			directories: []string{"./nonexistent/..."},
			expectError: true,
			errorMsg:    "Base directory does not exist: ./nonexistent (from pattern ./nonexistent/...)",
		},
		{
			name:        "mixed valid and invalid directories",
			directories: []string{"./internal/controllers", "./nonexistent"},
			expectError: true,
			errorMsg:    "Directory does not exist: ./nonexistent",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test the validation logic by simulating what main() does
			var validationError error
			
			// Validate directory paths (same logic as in main.go)
			for _, dir := range tt.directories {
				if strings.HasSuffix(dir, "/...") {
					baseDir := filepath.Dir(dir + "dummy") // Remove /... suffix
					baseDir = filepath.Clean(baseDir)
					if baseDir == "." {
						baseDir = "."
					} else {
						baseDir = filepath.Dir(dir[:len(dir)-4]) // Remove /...
					}
					
					// More accurate base directory extraction
					if dir == "./..." {
						baseDir = "."
					} else if dir[len(dir)-4:] == "/..." {
						baseDir = dir[:len(dir)-4]
					}
					
					if _, err := os.Stat(baseDir); os.IsNotExist(err) {
						validationError = err
						break
					}
				} else {
					if _, err := os.Stat(dir); os.IsNotExist(err) {
						validationError = err
						break
					}
				}
			}

			if tt.expectError {
				assert.Error(t, validationError, "Expected validation to fail for directories: %v", tt.directories)
			} else {
				assert.NoError(t, validationError, "Expected validation to pass for directories: %v", tt.directories)
			}
		})
	}
}

func TestGoStylePatternParsing(t *testing.T) {
	tests := []struct {
		name        string
		pattern     string
		expectedBase string
		isPattern   bool
	}{
		{
			name:        "root recursive pattern",
			pattern:     "./...",
			expectedBase: ".",
			isPattern:   true,
		},
		{
			name:        "subdirectory recursive pattern",
			pattern:     "./internal/...",
			expectedBase: "./internal",
			isPattern:   true,
		},
		{
			name:        "deep subdirectory recursive pattern",
			pattern:     "./internal/controllers/...",
			expectedBase: "./internal/controllers",
			isPattern:   true,
		},
		{
			name:        "absolute path pattern",
			pattern:     "/home/user/project/...",
			expectedBase: "/home/user/project",
			isPattern:   true,
		},
		{
			name:        "regular directory",
			pattern:     "./internal/controllers",
			expectedBase: "./internal/controllers",
			isPattern:   false,
		},
		{
			name:        "root directory",
			pattern:     ".",
			expectedBase: ".",
			isPattern:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isPattern := strings.HasSuffix(tt.pattern, "/...")
			assert.Equal(t, tt.isPattern, isPattern, "Pattern detection mismatch")

			if isPattern {
				baseDir := tt.pattern[:len(tt.pattern)-4] // Remove "/..."
				if baseDir == "" {
					baseDir = "."
				}
				assert.Equal(t, tt.expectedBase, baseDir, "Base directory extraction mismatch")
			}
		})
	}
}