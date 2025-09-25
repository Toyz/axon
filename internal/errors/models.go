package errors

import (
	"fmt"
	"strings"
)

// Models-specific error types that use the unified base types
// These replace the error types in internal/models/errors.go

// ParserConflict represents a parser registration conflict
type ParserConflict struct {
	FileName     string
	Line         int
	FunctionName string
	PackagePath  string
}

// GeneratorError represents an error that occurred during code generation
// This is a compatibility wrapper around the unified error types
type GeneratorError struct {
	AxonError
}

// NewGeneratorError creates a new generator error wrapping an AxonError
func NewGeneratorError(axonErr AxonError) *GeneratorError {
	return &GeneratorError{AxonError: axonErr}
}

// Type returns the error type (for backward compatibility)
func (e *GeneratorError) Type() ErrorCode {
	return e.ErrorCode()
}

// File returns the file name where the error occurred
func (e *GeneratorError) File() string {
	return e.Location().File
}

// Line returns the line number where the error occurred
func (e *GeneratorError) Line() int {
	return e.Location().Line
}

// Message returns the error message
func (e *GeneratorError) Message() string {
	return e.Error()
}

// Cause returns the underlying error cause
func (e *GeneratorError) Cause() error {
	return e.Unwrap()
}

// Suggestions returns helpful suggestions for fixing the error
func (e *GeneratorError) Suggestions() []string {
	return e.AxonError.Suggestions()
}

// Context returns additional context information
func (e *GeneratorError) Context() map[string]interface{} {
	return e.AxonError.Context()
}

// NewModelsParserRegistrationError creates a new parser registration error (for models package compatibility)
func NewModelsParserRegistrationError(typeName, fileName string, line int, existingFile string, existingLine int) *GeneratorError {
	loc := SourceLocation{
		File: fileName,
		Line: line,
	}

	// Use the underlying ParserError directly
	err := &ParserError{
		BaseError: New(ParserRegistrationErrorCode, fmt.Sprintf("parser for type '%s' already registered", typeName)).
			WithLocation(loc).
			WithSuggestions(
				"Choose a different type name for your parser",
				"Remove the duplicate parser registration",
				fmt.Sprintf("Check existing parser at %s:%d", existingFile, existingLine),
			).
			WithContext("type_name", typeName).
			WithContext("existing_file", existingFile).
			WithContext("existing_line", existingLine),
		ParserType: "route_parser",
		TypeName:   typeName,
	}

	return NewGeneratorError(err)
}

// NewModelsParserValidationError creates a new parser validation error (for models package compatibility)
func NewModelsParserValidationError(functionName, fileName string, line int, expectedSignature, actualIssue string) *GeneratorError {
	loc := SourceLocation{
		File: fileName,
		Line: line,
	}

	message := fmt.Sprintf("parser function '%s' has invalid signature: %s", functionName, actualIssue)

	err := &ParserError{
		BaseError: New(ParserValidationErrorCode, message).
			WithLocation(loc).
			WithContext("function_name", functionName).
			WithContext("expected_signature", expectedSignature).
			WithContext("actual_issue", actualIssue),
		ParserType:   "route_parser",
		FunctionName: functionName,
	}

	return NewGeneratorError(err)
}

// NewParserImportError creates a new parser import error
func NewParserImportError(typeName, fileName string, line int, requiredImport string) *GeneratorError {
	loc := SourceLocation{
		File: fileName,
		Line: line,
	}

	message := fmt.Sprintf("Parser for type '%s' requires missing import: %s", typeName, requiredImport)
	err := New(ParserImportErrorCode, message).
		WithLocation(loc).
		WithContext("type_name", typeName).
		WithContext("required_import", requiredImport).
		WithSuggestions(
			fmt.Sprintf("Add import: %s", requiredImport),
			"Ensure the package containing the type is imported",
			"Check if the type name is correct",
		)

	return NewGeneratorError(err)
}

// NewParserNotFoundError creates a new parser not found error
func NewParserNotFoundError(typeName, routeMethod, routePath, paramName, fileName string, line int, availableParsers []string) *GeneratorError {
	loc := SourceLocation{
		File: fileName,
		Line: line,
	}

	message := fmt.Sprintf("No parser registered for custom type '%s' used in route %s %s (parameter '%s')",
		typeName, routeMethod, routePath, paramName)

	suggestions := []string{
		fmt.Sprintf("Register a parser for type '%s' using //axon::route_parser %s", typeName, typeName),
		"Check if the type name is spelled correctly",
	}

	if len(availableParsers) > 0 {
		suggestions = append(suggestions, fmt.Sprintf("Available parsers: %s", strings.Join(availableParsers, ", ")))
	} else {
		suggestions = append(suggestions, "No parsers are currently registered")
	}

	err := New(ParserValidationErrorCode, message).
		WithLocation(loc).
		WithContext("type_name", typeName).
		WithContext("route_method", routeMethod).
		WithContext("route_path", routePath).
		WithContext("parameter_name", paramName).
		WithContext("available_parsers", availableParsers).
		WithSuggestions(suggestions...)

	return NewGeneratorError(err)
}

// NewParserConflictError creates a new parser conflict error
func NewParserConflictError(typeName string, conflicts []ParserConflict) *GeneratorError {
	var conflictDetails []string
	for _, conflict := range conflicts {
		conflictDetails = append(conflictDetails, fmt.Sprintf("%s:%d", conflict.FileName, conflict.Line))
	}

	message := fmt.Sprintf("Multiple parsers registered for type '%s'", typeName)

	err := New(ParserConflictErrorCode, message).
		WithContext("type_name", typeName).
		WithContext("conflicts", conflicts).
		WithSuggestions(
			"Keep only one parser registration for each type",
			"Remove duplicate parser annotations",
			fmt.Sprintf("Conflicting registrations found at: %s", strings.Join(conflictDetails, ", ")),
		)

	return NewGeneratorError(err)
}

// Generation-specific error helper functions

// NewGenerationFileError creates an error for file generation issues
func NewGenerationFileError(operation, fileName string, cause error) *GeneratorError {
	err := WrapFileSystemError(operation, fileName, cause)
	return NewGeneratorError(err)
}

// NewTemplateExecutionError creates an error for template execution issues
func NewTemplateExecutionError(templateName, operation string, cause error) *GeneratorError {
	err := WrapTemplateError(templateName, operation, cause)
	return NewGeneratorError(err)
}

// NewValidationGenerationError creates a validation error during generation
func NewValidationGenerationError(field, expected, actual, fileName string, line int) *GeneratorError {
	loc := SourceLocation{
		File: fileName,
		Line: line,
	}

	err := NewValidationError(field, expected, actual).WithLocation(loc)
	return NewGeneratorError(err)
}

// NewSyntaxGenerationError creates a syntax error during generation
func NewSyntaxGenerationError(message, fileName string, line int) *GeneratorError {
	loc := SourceLocation{
		File: fileName,
		Line: line,
	}

	err := NewSyntaxError(message).WithLocation(loc)
	return NewGeneratorError(err)
}

// Collection helpers for generator errors

// GeneratorErrorCollection helps collect multiple generator errors
type GeneratorErrorCollection struct {
	*MultipleErrors
}

// NewGeneratorErrorCollection creates a new error collection
func NewGeneratorErrorCollection() *GeneratorErrorCollection {
	return &GeneratorErrorCollection{
		MultipleErrors: NewMultipleErrors(),
	}
}

// AddGenerator adds a generator error to the collection
func (c *GeneratorErrorCollection) AddGenerator(err *GeneratorError) {
	c.Add(err.AxonError)
}

// AddParser adds a parser error to the collection
func (c *GeneratorErrorCollection) AddParser(err *ParserError) {
	c.Add(err)
}

// AddValidation adds a validation error to the collection
func (c *GeneratorErrorCollection) AddValidation(field, expected, actual, fileName string, line int) {
	err := NewValidationGenerationError(field, expected, actual, fileName, line)
	c.AddGenerator(err)
}

// AddSyntax adds a syntax error to the collection
func (c *GeneratorErrorCollection) AddSyntax(message, fileName string, line int) {
	err := NewSyntaxGenerationError(message, fileName, line)
	c.AddGenerator(err)
}

// ToGeneratorError returns the collected errors as a single generator error
func (c *GeneratorErrorCollection) ToGeneratorError() *GeneratorError {
	if c.IsEmpty() {
		return nil
	}

	if c.Count() == 1 {
		if genErr, ok := c.Errors[0].(*GeneratorError); ok {
			return genErr
		}
		return NewGeneratorError(c.Errors[0])
	}

	return NewGeneratorError(c.MultipleErrors)
}

// Legacy compatibility functions to maintain backward compatibility

// ErrorType represents legacy error types (for backward compatibility)
type ErrorType = ErrorCode

// Legacy error type constants
const (
	ErrorTypeAnnotationSyntax   = SyntaxErrorCode
	ErrorTypeValidation         = ValidationErrorCode
	ErrorTypeGeneration         = GenerationErrorCode
	ErrorTypeFileSystem         = FileSystemErrorCode
	ErrorTypeParserRegistration = ParserRegistrationErrorCode
	ErrorTypeParserValidation   = ParserValidationErrorCode
	ErrorTypeParserImport       = ParserImportErrorCode
	ErrorTypeParserConflict     = ParserConflictErrorCode
)