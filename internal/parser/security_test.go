package parser

import (
	"strings"
	"testing"
)

func TestIsSecureDirectoryPath(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		expected bool
	}{
		{"valid current directory", ".", true},
		{"valid relative path", "./internal", true},
		{"valid nested path", "internal/pkg", true},
		{"valid absolute path", "/home/user/project", true},
		{"valid windows path", "C:\\Users\\user\\project", true},
		{"empty path", "", false},
		{"path with null byte", "path\x00injection", false},
		{"path with dangerous chars", "path<script>", false},
		{"path with pipe", "path|command", false},
		{"path with path traversal", "../malicious", true}, // Allowed here, checked after Clean()
		{"complex valid path", "./examples/complete-app/internal", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isSecureDirectoryPath(tt.path)
			if result != tt.expected {
				t.Errorf("isSecureDirectoryPath(%q) = %v, expected %v", tt.path, result, tt.expected)
			}
		})
	}
}

func TestIsValidGoModPath(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		expected bool
	}{
		{"valid go.mod path", "./go.mod", true},
		{"valid nested go.mod", "internal/go.mod", true},
		{"valid absolute go.mod", "/home/user/project/go.mod", true},
		{"empty path", "", false},
		{"path with null byte", "go.mod\x00", false},
		{"path with dangerous chars", "go<script>.mod", false},
		{"path not ending with go.mod", "./main.go", false},
		{"path with pipe", "go.mod|command", false},
		{"complex valid go.mod path", "./examples/complete-app/go.mod", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isValidGoModPath(tt.path)
			if result != tt.expected {
				t.Errorf("isValidGoModPath(%q) = %v, expected %v", tt.path, result, tt.expected)
			}
		})
	}
}

func TestParseDirectorySecurityValidation(t *testing.T) {
	parser := NewParser()

	// Test that path traversal is blocked after cleaning
	_, err := parser.ParseDirectory("../../../etc")
	if err == nil {
		t.Error("expected error for path traversal, got none")
	}
	if err != nil && !strings.Contains(err.Error(), "path traversal not allowed") {
		t.Errorf("expected path traversal error, got: %v", err)
	}

	// Test that null byte injection is blocked
	_, err = parser.ParseDirectory("valid\x00path")
	if err == nil {
		t.Error("expected error for null byte injection, got none")
	}
	if err != nil && !strings.Contains(err.Error(), "invalid directory path") {
		t.Errorf("expected invalid directory path error, got: %v", err)
	}
}

func TestParseGoModFileSecurityValidation(t *testing.T) {
	parser := NewParser()

	// Test that path traversal is blocked
	_, err := parser.parseGoModFile("../../../etc/passwd")
	if err == nil {
		t.Error("expected error for path traversal, got none")
	}

	// Test that non-go.mod files are blocked
	_, err = parser.parseGoModFile("./main.go")
	if err == nil {
		t.Error("expected error for non-go.mod file, got none")
	}
	if err != nil && !strings.Contains(err.Error(), "invalid go.mod file path") {
		t.Errorf("expected invalid go.mod file path error, got: %v", err)
	}
}
