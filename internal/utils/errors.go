package utils

import "fmt"

// ErrorWrappers provides common error wrapping patterns used throughout the codebase
// to reduce duplication and ensure consistent error formatting.

// WrapRegisterError wraps an error with a "failed to register" message
func WrapRegisterError(name string, err error) error {
	return fmt.Errorf("failed to register %s: %w", name, err)
}

// WrapParseError wraps an error with a "failed to parse" message
func WrapParseError(item string, err error) error {
	return fmt.Errorf("failed to parse %s: %w", item, err)
}

// WrapGenerateError wraps an error with a "failed to generate" message
func WrapGenerateError(item string, err error) error {
	return fmt.Errorf("failed to generate %s: %w", item, err)
}

// WrapCreateError wraps an error with a "failed to create" message
func WrapCreateError(item string, err error) error {
	return fmt.Errorf("failed to create %s: %w", item, err)
}

// WrapLoadError wraps an error with a "failed to load" message
func WrapLoadError(item string, err error) error {
	return fmt.Errorf("failed to load %s: %w", item, err)
}

// WrapValidateError wraps an error with a "failed to validate" message
func WrapValidateError(item string, err error) error {
	return fmt.Errorf("failed to validate %s: %w", item, err)
}

// WrapProcessError wraps an error with a "failed to process" message
func WrapProcessError(item string, err error) error {
	return fmt.Errorf("failed to process %s: %w", item, err)
}