package models

import (
	"fmt"
	"strings"
)

// GeneratorError represents an error that occurred during code generation
type GeneratorError struct {
	Type        ErrorType // type of error
	File        string    // file where error occurred
	Line        int       // line number where error occurred
	Message     string    // error message
	Cause       error     // underlying error cause
	Suggestions []string  // helpful suggestions for fixing the error
	Context     map[string]interface{} // additional context information
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

// NewParserRegistrationError creates a new parser registration error
func NewParserRegistrationError(typeName, fileName string, line int, existingFile string, existingLine int) *GeneratorError {
	return &GeneratorError{
		Type:    ErrorTypeParserRegistration,
		File:    fileName,
		Line:    line,
		Message: fmt.Sprintf("Parser for type '%s' already registered", typeName),
		Suggestions: []string{
			"Choose a different type name for your parser",
			"Remove the duplicate parser registration",
			fmt.Sprintf("Check existing parser at %s:%d", existingFile, existingLine),
		},
		Context: map[string]interface{}{
			"type_name":      typeName,
			"existing_file":  existingFile,
			"existing_line":  existingLine,
		},
	}
}

// NewParserValidationError creates a new parser validation error
func NewParserValidationError(functionName, fileName string, line int, expectedSignature, actualIssue string) *GeneratorError {
	return &GeneratorError{
		Type:    ErrorTypeParserValidation,
		File:    fileName,
		Line:    line,
		Message: fmt.Sprintf("Parser function '%s' has invalid signature: %s", functionName, actualIssue),
		Suggestions: []string{
			fmt.Sprintf("Expected signature: %s", expectedSignature),
			"Ensure the first parameter is echo.Context",
			"Ensure the second parameter is string",
			"Ensure the function returns (T, error)",
		},
		Context: map[string]interface{}{
			"function_name":      functionName,
			"expected_signature": expectedSignature,
			"actual_issue":       actualIssue,
		},
	}
}

// NewParserImportError creates a new parser import error
func NewParserImportError(typeName, fileName string, line int, requiredImport string) *GeneratorError {
	return &GeneratorError{
		Type:    ErrorTypeParserImport,
		File:    fileName,
		Line:    line,
		Message: fmt.Sprintf("Parser for type '%s' requires missing import", typeName),
		Suggestions: []string{
			fmt.Sprintf("Add import: %s", requiredImport),
			"Ensure the package containing the type is imported",
			"Check if the type name is correct",
		},
		Context: map[string]interface{}{
			"type_name":       typeName,
			"required_import": requiredImport,
		},
	}
}

// NewParserNotFoundError creates a new parser not found error
func NewParserNotFoundError(typeName, routeMethod, routePath, paramName, fileName string, line int, availableParsers []string) *GeneratorError {
	suggestions := []string{
		fmt.Sprintf("Register a parser for type '%s' using //axon::route_parser %s", typeName, typeName),
		"Check if the type name is spelled correctly",
	}
	
	if len(availableParsers) > 0 {
		suggestions = append(suggestions, fmt.Sprintf("Available parsers: %s", strings.Join(availableParsers, ", ")))
	} else {
		suggestions = append(suggestions, "No parsers are currently registered")
	}
	
	return &GeneratorError{
		Type:    ErrorTypeParserValidation,
		File:    fileName,
		Line:    line,
		Message: fmt.Sprintf("No parser registered for custom type '%s' used in route %s %s (parameter '%s')", typeName, routeMethod, routePath, paramName),
		Suggestions: suggestions,
		Context: map[string]interface{}{
			"type_name":         typeName,
			"route_method":      routeMethod,
			"route_path":        routePath,
			"parameter_name":    paramName,
			"available_parsers": availableParsers,
		},
	}
}

// NewParserConflictError creates a new parser conflict error
func NewParserConflictError(typeName string, conflicts []ParserConflict) *GeneratorError {
	var conflictDetails []string
	for _, conflict := range conflicts {
		conflictDetails = append(conflictDetails, fmt.Sprintf("%s:%d", conflict.FileName, conflict.Line))
	}
	
	return &GeneratorError{
		Type:    ErrorTypeParserConflict,
		Message: fmt.Sprintf("Multiple parsers registered for type '%s'", typeName),
		Suggestions: []string{
			"Keep only one parser registration for each type",
			"Remove duplicate parser annotations",
			fmt.Sprintf("Conflicting registrations found at: %s", strings.Join(conflictDetails, ", ")),
		},
		Context: map[string]interface{}{
			"type_name": typeName,
			"conflicts": conflicts,
		},
	}
}

// ParserConflict represents a parser registration conflict
type ParserConflict struct {
	FileName     string
	Line         int
	FunctionName string
	PackagePath  string
}