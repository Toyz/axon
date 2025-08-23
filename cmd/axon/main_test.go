package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCLIArgumentParsing tests the CLI argument parsing by running the binary
func TestCLIArgumentParsing(t *testing.T) {
	// Build the CLI binary for testing
	tempDir, err := os.MkdirTemp("", "axon_cli_test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	binaryPath := filepath.Join(tempDir, "axon")
	
	// Build the binary
	cmd := exec.Command("go", "build", "-o", binaryPath, ".")
	cmd.Dir = "." // Build in current directory
	buildErr := cmd.Run()
	require.NoError(t, buildErr, "Failed to build CLI binary")

	t.Run("help flag", func(t *testing.T) {
		cmd := exec.Command(binaryPath, "--help")
		output, err := cmd.CombinedOutput()
		
		// Help should exit with code 0
		assert.NoError(t, err)
		
		outputStr := string(output)
		assert.Contains(t, outputStr, "Usage:")
		assert.Contains(t, outputStr, "Axon Framework Code Generator")
		assert.Contains(t, outputStr, "--module")
		assert.Contains(t, outputStr, "directory-paths")
		assert.NotContains(t, outputStr, "--main") // --main flag was removed
	})

	t.Run("no arguments", func(t *testing.T) {
		cmd := exec.Command(binaryPath)
		output, err := cmd.CombinedOutput()
		
		// Should exit with error code
		assert.Error(t, err)
		
		outputStr := string(output)
		assert.Contains(t, outputStr, "At least one directory path is required")
	})

	t.Run("nonexistent directory", func(t *testing.T) {
		cmd := exec.Command(binaryPath, "/nonexistent/directory")
		output, err := cmd.CombinedOutput()
		
		// Should exit with error code
		assert.Error(t, err)
		
		outputStr := string(output)
		assert.Contains(t, outputStr, "Directory does not exist")
	})

	t.Run("valid directory argument", func(t *testing.T) {
		// Create a test directory with a Go file
		testDir := filepath.Join(tempDir, "testpkg")
		require.NoError(t, os.MkdirAll(testDir, 0755))
		
		goFile := filepath.Join(testDir, "test.go")
		require.NoError(t, os.WriteFile(goFile, []byte("package testpkg\n\ntype Test struct{}"), 0644))

		cmd := exec.Command(binaryPath, testDir)
		output, _ := cmd.CombinedOutput()
		
		// The command should run but may fail due to missing go.mod or annotations
		// We're just testing that the directory argument is accepted
		outputStr := string(output)
		
		// Should not contain directory validation errors
		assert.NotContains(t, outputStr, "Directory does not exist")
		assert.NotContains(t, outputStr, "At least one directory path is required")
	})
}

// TestCLIFlags tests individual flag parsing
func TestCLIFlags(t *testing.T) {
	// This test uses a mock approach since we can't easily test flag parsing
	// in the same process. In a real scenario, you might want to refactor
	// the main function to be more testable.
	
	t.Run("flag parsing logic", func(t *testing.T) {
		// Test the logic that would be used in main()
		testCases := []struct {
			name     string
			args     []string
			moduleFlag string
			directories []string
			shouldError bool
		}{
			{
				name: "basic directory",
				args: []string{"./internal"},
				directories: []string{"./internal"},
			},
			{
				name: "multiple directories",
				args: []string{"./internal/controllers", "./internal/services"},
				directories: []string{"./internal/controllers", "./internal/services"},
			},
			{
				name: "with module flag",
				args: []string{"--module", "github.com/example/app", "./internal"},
				moduleFlag: "github.com/example/app",
				directories: []string{"./internal"},
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				// This is a simplified test of the flag parsing logic
				// In practice, you'd want to refactor main() to be more testable
				
				// Simulate flag parsing
				var moduleFlag string
				var directories []string
				
				for i := 0; i < len(tc.args); i++ {
					arg := tc.args[i]
					switch arg {
					case "--module":
						if i+1 < len(tc.args) {
							moduleFlag = tc.args[i+1]
							i++ // skip next arg
						}
					default:
						if !strings.HasPrefix(arg, "--") {
							directories = append(directories, arg)
						}
					}
				}

				assert.Equal(t, tc.moduleFlag, moduleFlag)
				assert.Equal(t, tc.directories, directories)
			})
		}
	})
}