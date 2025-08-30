package utils

import (
	"errors"
	"testing"
)

func TestErrorWrappers(t *testing.T) {
	originalErr := errors.New("original error")

	tests := []struct {
		name     string
		wrapper  func(string, error) error
		item     string
		expected string
	}{
		{
			name:     "WrapRegisterError",
			wrapper:  WrapRegisterError,
			item:     "middleware",
			expected: "failed to register middleware: original error",
		},
		{
			name:     "WrapParseError",
			wrapper:  WrapParseError,
			item:     "template",
			expected: "failed to parse template: original error",
		},
		{
			name:     "WrapGenerateError",
			wrapper:  WrapGenerateError,
			item:     "provider",
			expected: "failed to generate provider: original error",
		},
		{
			name:     "WrapCreateError",
			wrapper:  WrapCreateError,
			item:     "file",
			expected: "failed to create file: original error",
		},
		{
			name:     "WrapLoadError",
			wrapper:  WrapLoadError,
			item:     "config",
			expected: "failed to load config: original error",
		},
		{
			name:     "WrapValidateError",
			wrapper:  WrapValidateError,
			item:     "annotation",
			expected: "failed to validate annotation: original error",
		},
		{
			name:     "WrapProcessError",
			wrapper:  WrapProcessError,
			item:     "directory",
			expected: "failed to process directory: original error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.wrapper(tt.item, originalErr)
			if result.Error() != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result.Error())
			}

			// Test that the error can be unwrapped
			if !errors.Is(result, originalErr) {
				t.Errorf("wrapped error should be unwrappable to original error")
			}
		})
	}
}

func TestErrorWrappersWithEmptyItem(t *testing.T) {
	originalErr := errors.New("test error")
	
	result := WrapRegisterError("", originalErr)
	expected := "failed to register : test error"
	
	if result.Error() != expected {
		t.Errorf("expected %q, got %q", expected, result.Error())
	}
}

func TestErrorWrappersWithNilError(t *testing.T) {
	// These should still work with nil errors (though not recommended usage)
	result := WrapRegisterError("test", nil)
	expected := "failed to register test: %!w(<nil>)"
	
	if result.Error() != expected {
		t.Errorf("expected %q, got %q", expected, result.Error())
	}
}