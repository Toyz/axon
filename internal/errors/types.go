package errors

import (
	"fmt"
	"strings"
)

// AxonError defines the base interface for all Axon framework errors
type AxonError interface {
	error
	ErrorCode() ErrorCode
	Location() SourceLocation
	Context() map[string]interface{}
	Suggestions() []string
	Unwrap() error
}

// ErrorCode represents the type of error that occurred
type ErrorCode int

const (
	// Core error types
	UnknownErrorCode ErrorCode = iota
	SyntaxErrorCode
	ValidationErrorCode
	RegistrationErrorCode
	SchemaErrorCode

	// Generation error types
	GenerationErrorCode
	TemplateErrorCode
	FileSystemErrorCode

	// Parser error types
	ParserRegistrationErrorCode
	ParserValidationErrorCode
	ParserImportErrorCode
	ParserConflictErrorCode

	// Runtime error types
	ConfigurationErrorCode
	DependencyErrorCode
)

// String returns the string representation of the error code
func (e ErrorCode) String() string {
	switch e {
	case SyntaxErrorCode:
		return "SyntaxError"
	case ValidationErrorCode:
		return "ValidationError"
	case RegistrationErrorCode:
		return "RegistrationError"
	case SchemaErrorCode:
		return "SchemaError"
	case GenerationErrorCode:
		return "GenerationError"
	case TemplateErrorCode:
		return "TemplateError"
	case FileSystemErrorCode:
		return "FileSystemError"
	case ParserRegistrationErrorCode:
		return "ParserRegistrationError"
	case ParserValidationErrorCode:
		return "ParserValidationError"
	case ParserImportErrorCode:
		return "ParserImportError"
	case ParserConflictErrorCode:
		return "ParserConflictError"
	case ConfigurationErrorCode:
		return "ConfigurationError"
	case DependencyErrorCode:
		return "DependencyError"
	default:
		return "UnknownError"
	}
}

// SourceLocation represents where an error occurred in source code
type SourceLocation struct {
	File   string // file path where error occurred
	Line   int    // line number (1-based)
	Column int    // column number (1-based)
}

// String returns a formatted string representation of the location
func (s SourceLocation) String() string {
	if s.File == "" {
		return "unknown location"
	}
	if s.Line == 0 {
		return s.File
	}
	if s.Column == 0 {
		return fmt.Sprintf("%s:%d", s.File, s.Line)
	}
	return fmt.Sprintf("%s:%d:%d", s.File, s.Line, s.Column)
}

// IsEmpty returns true if the location has no useful information
func (s SourceLocation) IsEmpty() bool {
	return s.File == ""
}

// BaseError provides a common implementation of the AxonError interface
type BaseError struct {
	Code        ErrorCode              // type of error
	Message     string                 // error message
	Loc         SourceLocation         // where the error occurred
	Cause       error                  // underlying error cause
	ContextData map[string]interface{} // additional context information
	Hints       []string               // helpful suggestions for fixing the error
}

// Error implements the error interface
func (e *BaseError) Error() string {
	if e.Loc.IsEmpty() {
		return e.Message
	}
	return fmt.Sprintf("%s: %s", e.Loc.String(), e.Message)
}

// ErrorCode returns the error code
func (e *BaseError) ErrorCode() ErrorCode {
	return e.Code
}

// Location returns the source location where the error occurred
func (e *BaseError) Location() SourceLocation {
	return e.Loc
}

// Context returns the error context data
func (e *BaseError) Context() map[string]interface{} {
	if e.ContextData == nil {
		return make(map[string]interface{})
	}
	return e.ContextData
}

// Suggestions returns helpful suggestions for fixing the error
func (e *BaseError) Suggestions() []string {
	return e.Hints
}

// Unwrap returns the underlying error cause for error chain inspection
func (e *BaseError) Unwrap() error {
	return e.Cause
}

// WithLocation adds location information to the error
func (e *BaseError) WithLocation(loc SourceLocation) *BaseError {
	e.Loc = loc
	return e
}

// WithCause adds an underlying error cause
func (e *BaseError) WithCause(cause error) *BaseError {
	e.Cause = cause
	return e
}

// WithContext adds context data to the error
func (e *BaseError) WithContext(key string, value interface{}) *BaseError {
	if e.ContextData == nil {
		e.ContextData = make(map[string]interface{})
	}
	e.ContextData[key] = value
	return e
}

// WithSuggestion adds a helpful suggestion for fixing the error
func (e *BaseError) WithSuggestion(suggestion string) *BaseError {
	e.Hints = append(e.Hints, suggestion)
	return e
}

// WithSuggestions adds multiple helpful suggestions
func (e *BaseError) WithSuggestions(suggestions ...string) *BaseError {
	e.Hints = append(e.Hints, suggestions...)
	return e
}

// New creates a new BaseError with the specified code and message
func New(code ErrorCode, message string) *BaseError {
	return &BaseError{
		Code:    code,
		Message: message,
		Hints:   make([]string, 0),
	}
}

// Newf creates a new BaseError with formatted message
func Newf(code ErrorCode, format string, args ...interface{}) *BaseError {
	return New(code, fmt.Sprintf(format, args...))
}

// Wrap creates a new error that wraps another error
func Wrap(code ErrorCode, message string, cause error) *BaseError {
	return &BaseError{
		Code:    code,
		Message: message,
		Cause:   cause,
		Hints:   make([]string, 0),
	}
}

// Wrapf creates a new error that wraps another error with formatted message
func Wrapf(code ErrorCode, cause error, format string, args ...interface{}) *BaseError {
	return Wrap(code, fmt.Sprintf(format, args...), cause)
}

// MultipleErrors represents multiple errors collected together
type MultipleErrors struct {
	Errors []AxonError
}

// Error implements the error interface
func (e *MultipleErrors) Error() string {
	if len(e.Errors) == 0 {
		return "no errors"
	}

	if len(e.Errors) == 1 {
		return e.Errors[0].Error()
	}

	var messages []string
	for i, err := range e.Errors {
		messages = append(messages, fmt.Sprintf("  %d. %s", i+1, err.Error()))
	}

	return fmt.Sprintf("multiple errors (%d total):\n%s", len(e.Errors), strings.Join(messages, "\n"))
}

// ErrorCode returns the error code (uses the first error's code)
func (e *MultipleErrors) ErrorCode() ErrorCode {
	if len(e.Errors) == 0 {
		return UnknownErrorCode
	}
	return e.Errors[0].ErrorCode()
}

// Location returns the location of the first error
func (e *MultipleErrors) Location() SourceLocation {
	if len(e.Errors) == 0 {
		return SourceLocation{}
	}
	return e.Errors[0].Location()
}

// Context returns combined context from all errors
func (e *MultipleErrors) Context() map[string]interface{} {
	combined := make(map[string]interface{})
	for i, err := range e.Errors {
		for k, v := range err.Context() {
			// Prefix keys with error index to avoid conflicts
			combined[fmt.Sprintf("error_%d_%s", i, k)] = v
		}
	}
	return combined
}

// Suggestions returns combined suggestions from all errors
func (e *MultipleErrors) Suggestions() []string {
	var suggestions []string
	for _, err := range e.Errors {
		suggestions = append(suggestions, err.Suggestions()...)
	}
	return suggestions
}

// Unwrap returns the first underlying error for error inspection (to satisfy the error interface)
func (e *MultipleErrors) Unwrap() error {
	if len(e.Errors) == 0 {
		return nil
	}
	return e.Errors[0]
}

// UnwrapAll returns all underlying errors for inspection
func (e *MultipleErrors) UnwrapAll() []error {
	errors := make([]error, len(e.Errors))
	for i, err := range e.Errors {
		errors[i] = err
	}
	return errors
}

// Add adds an error to the collection
func (e *MultipleErrors) Add(err AxonError) {
	e.Errors = append(e.Errors, err)
}

// IsEmpty returns true if there are no errors
func (e *MultipleErrors) IsEmpty() bool {
	return len(e.Errors) == 0
}

// Count returns the number of errors
func (e *MultipleErrors) Count() int {
	return len(e.Errors)
}

// GetByCode returns all errors of a specific type
func (e *MultipleErrors) GetByCode(code ErrorCode) []AxonError {
	var result []AxonError
	for _, err := range e.Errors {
		if err.ErrorCode() == code {
			result = append(result, err)
		}
	}
	return result
}

// HasCode returns true if any error of the specified type exists
func (e *MultipleErrors) HasCode(code ErrorCode) bool {
	for _, err := range e.Errors {
		if err.ErrorCode() == code {
			return true
		}
	}
	return false
}

// Is checks if any of the wrapped errors matches the target
func (e *MultipleErrors) Is(target error) bool {
	for _, err := range e.Errors {
		if err == target {
			return true
		}
	}
	return false
}

// As finds the first error in the chain that matches the target type
func (e *MultipleErrors) As(target interface{}) bool {
	for _, err := range e.Errors {
		if as, ok := err.(interface{ As(interface{}) bool }); ok && as.As(target) {
			return true
		}
	}
	return false
}

// NewMultipleErrors creates a new MultipleErrors collection
func NewMultipleErrors() *MultipleErrors {
	return &MultipleErrors{
		Errors: make([]AxonError, 0),
	}
}

// CollectErrors creates a MultipleErrors from a slice of AxonErrors
func CollectErrors(errors ...AxonError) *MultipleErrors {
	return &MultipleErrors{
		Errors: errors,
	}
}