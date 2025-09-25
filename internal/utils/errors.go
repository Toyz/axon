package utils

import (
	"fmt"

	"github.com/toyz/axon/internal/errors"
)

// ErrorWrappers provides common error wrapping patterns used throughout the codebase
// to reduce duplication and ensure consistent error formatting.
//
// Deprecated: Use the unified error system in internal/errors package instead.
// This file provides backward compatibility wrappers.

// WrapRegisterError wraps an error with a "failed to register" message
// Deprecated: Use errors.WrapRegisterError instead
func WrapRegisterError(name string, err error) error {
	return fmt.Errorf("failed to register %s: %w", name, err)
}

// WrapParseError wraps an error with a "failed to parse" message
// Deprecated: Use errors.WrapParseError instead
func WrapParseError(item string, err error) error {
	return errors.WrapParseError(item, err)
}

// WrapGenerateError wraps an error with a "failed to generate" message
// Deprecated: Use errors.WrapGenerateError instead
func WrapGenerateError(item string, err error) error {
	return errors.WrapGenerateError("", item, err)
}

// WrapCreateError wraps an error with a "failed to create" message
// Deprecated: Use errors.WrapWithOperation instead
func WrapCreateError(item string, err error) error {
	return errors.WrapWithOperation("create", item, err)
}

// WrapLoadError wraps an error with a "failed to load" message
// Deprecated: Use errors.WrapWithOperation instead
func WrapLoadError(item string, err error) error {
	return errors.WrapWithOperation("load", item, err)
}

// WrapValidateError wraps an error with a "failed to validate" message
// Deprecated: Use errors.WrapValidationError instead
func WrapValidateError(item string, err error) error {
	return errors.WrapValidationError(item, err)
}

// WrapProcessError wraps an error with a "failed to process" message
// Deprecated: Use errors.WrapWithOperation instead
func WrapProcessError(item string, err error) error {
	return errors.WrapWithOperation("process", item, err)
}
