package utils

import (
	"fmt"
	"go/token"
	"regexp"
	"strings"
)

// ValidationError represents a validation error with context
type ValidationError struct {
	Field   string
	Value   interface{}
	Message string
}

func (e ValidationError) Error() string {
	if e.Field != "" {
		return fmt.Sprintf("validation error for field '%s': %s", e.Field, e.Message)
	}
	return fmt.Sprintf("validation error: %s", e.Message)
}

// Validator represents a validation function
type Validator[T any] func(T) error

// ValidatorChain allows chaining multiple validators
type ValidatorChain[T any] struct {
	validators []Validator[T]
}

// NewValidatorChain creates a new validator chain
func NewValidatorChain[T any](validators ...Validator[T]) *ValidatorChain[T] {
	return &ValidatorChain[T]{validators: validators}
}

// Add adds a validator to the chain
func (vc *ValidatorChain[T]) Add(validator Validator[T]) *ValidatorChain[T] {
	vc.validators = append(vc.validators, validator)
	return vc
}

// Validate runs all validators in the chain
func (vc *ValidatorChain[T]) Validate(value T) error {
	for _, validator := range vc.validators {
		if err := validator(value); err != nil {
			return err
		}
	}
	return nil
}

// Common validation functions

// NotEmpty validates that a string is not empty
func NotEmpty(field string) Validator[string] {
	return func(value string) error {
		if value == "" {
			return ValidationError{
				Field:   field,
				Value:   value,
				Message: "cannot be empty",
			}
		}
		return nil
	}
}

// NotNil validates that a pointer is not nil
func NotNil[T any](field string) Validator[*T] {
	return func(value *T) error {
		if value == nil {
			return ValidationError{
				Field:   field,
				Value:   value,
				Message: "cannot be nil",
			}
		}
		return nil
	}
}

// HasPrefix validates that a string has a specific prefix
func HasPrefix(field, prefix string) Validator[string] {
	return func(value string) error {
		if !strings.HasPrefix(value, prefix) {
			return ValidationError{
				Field:   field,
				Value:   value,
				Message: fmt.Sprintf("must start with '%s'", prefix),
			}
		}
		return nil
	}
}

// HasSuffix validates that a string has a specific suffix
func HasSuffix(field, suffix string) Validator[string] {
	return func(value string) error {
		if !strings.HasSuffix(value, suffix) {
			return ValidationError{
				Field:   field,
				Value:   value,
				Message: fmt.Sprintf("must end with '%s'", suffix),
			}
		}
		return nil
	}
}

// MatchesRegex validates that a string matches a regex pattern
func MatchesRegex(field, pattern string) Validator[string] {
	regex := regexp.MustCompile(pattern)
	return func(value string) error {
		if !regex.MatchString(value) {
			return ValidationError{
				Field:   field,
				Value:   value,
				Message: fmt.Sprintf("must match pattern '%s'", pattern),
			}
		}
		return nil
	}
}

// IsValidGoIdentifier validates that a string is a valid Go identifier
func IsValidGoIdentifier(field string) Validator[string] {
	return func(value string) error {
		if value == "" {
			return ValidationError{
				Field:   field,
				Value:   value,
				Message: "cannot be empty",
			}
		}
		
		if !token.IsIdentifier(value) {
			return ValidationError{
				Field:   field,
				Value:   value,
				Message: "must be a valid Go identifier",
			}
		}
		
		return nil
	}
}

// IsOneOf validates that a value is one of the allowed values
func IsOneOf[T comparable](field string, allowed ...T) Validator[T] {
	return func(value T) error {
		for _, allowedValue := range allowed {
			if value == allowedValue {
				return nil
			}
		}
		
		return ValidationError{
			Field:   field,
			Value:   value,
			Message: fmt.Sprintf("must be one of: %v", allowed),
		}
	}
}

// MinLength validates that a string has a minimum length
func MinLength(field string, min int) Validator[string] {
	return func(value string) error {
		if len(value) < min {
			return ValidationError{
				Field:   field,
				Value:   value,
				Message: fmt.Sprintf("must be at least %d characters long", min),
			}
		}
		return nil
	}
}

// MaxLength validates that a string has a maximum length
func MaxLength(field string, max int) Validator[string] {
	return func(value string) error {
		if len(value) > max {
			return ValidationError{
				Field:   field,
				Value:   value,
				Message: fmt.Sprintf("must be at most %d characters long", max),
			}
		}
		return nil
	}
}

// SliceNotEmpty validates that a slice is not empty
func SliceNotEmpty[T any](field string) Validator[[]T] {
	return func(value []T) error {
		if len(value) == 0 {
			return ValidationError{
				Field:   field,
				Value:   value,
				Message: "cannot be empty",
			}
		}
		return nil
	}
}

// SliceMinLength validates that a slice has a minimum length
func SliceMinLength[T any](field string, min int) Validator[[]T] {
	return func(value []T) error {
		if len(value) < min {
			return ValidationError{
				Field:   field,
				Value:   value,
				Message: fmt.Sprintf("must have at least %d items", min),
			}
		}
		return nil
	}
}

// SliceMaxLength validates that a slice has a maximum length
func SliceMaxLength[T any](field string, max int) Validator[[]T] {
	return func(value []T) error {
		if len(value) > max {
			return ValidationError{
				Field:   field,
				Value:   value,
				Message: fmt.Sprintf("must have at most %d items", max),
			}
		}
		return nil
	}
}

// ValidateEach validates each item in a slice using the provided validator
func ValidateEach[T any](field string, itemValidator Validator[T]) Validator[[]T] {
	return func(value []T) error {
		for i, item := range value {
			if err := itemValidator(item); err != nil {
				return ValidationError{
					Field:   fmt.Sprintf("%s[%d]", field, i),
					Value:   item,
					Message: err.Error(),
				}
			}
		}
		return nil
	}
}

// Custom validates using a custom function
func Custom[T any](field string, message string, validatorFunc func(T) bool) Validator[T] {
	return func(value T) error {
		if !validatorFunc(value) {
			return ValidationError{
				Field:   field,
				Value:   value,
				Message: message,
			}
		}
		return nil
	}
}

// Conditional validates only if the condition is true
func Conditional[T any](condition func(T) bool, validator Validator[T]) Validator[T] {
	return func(value T) error {
		if condition(value) {
			return validator(value)
		}
		return nil
	}
}

// Common validation patterns for specific use cases

// ValidateHTTPMethod validates that a string is a valid HTTP method
func ValidateHTTPMethod(field string) Validator[string] {
	return IsOneOf(field, "GET", "POST", "PUT", "DELETE", "PATCH", "HEAD", "OPTIONS")
}

// ValidateURLPath validates that a string is a valid URL path
func ValidateURLPath(field string) Validator[string] {
	return NewValidatorChain(
		NotEmpty(field),
		HasPrefix(field, "/"),
	).Validate
}

// ValidateMiddlewareRoute validates that a string is a valid middleware route pattern
func ValidateMiddlewareRoute(field string) Validator[string] {
	return NewValidatorChain(
		NotEmpty(field),
		HasPrefix(field, "/"),
	).Validate
}

// ValidateConstructorName validates that a string is a valid constructor function name
func ValidateConstructorName(field string) Validator[string] {
	return NewValidatorChain(
		NotEmpty(field),
		IsValidGoIdentifier(field),
		HasPrefix(field, "New"),
	).Validate
}