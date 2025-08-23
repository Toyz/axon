package models

import "fmt"

// GeneratorError represents an error that occurred during code generation
type GeneratorError struct {
	Type    ErrorType // type of error
	File    string    // file where error occurred
	Line    int       // line number where error occurred
	Message string    // error message
	Cause   error     // underlying error cause
}

// Error implements the error interface
func (e *GeneratorError) Error() string {
	if e.File != "" && e.Line > 0 {
		return fmt.Sprintf("%s:%d: %s", e.File, e.Line, e.Message)
	}
	if e.File != "" {
		return fmt.Sprintf("%s: %s", e.File, e.Message)
	}
	return e.Message
}

// Unwrap returns the underlying error cause
func (e *GeneratorError) Unwrap() error {
	return e.Cause
}