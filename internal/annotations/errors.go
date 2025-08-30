package annotations

import (
	"fmt"
	"strings"
)

// AnnotationError defines the interface for annotation-related errors
type AnnotationError interface {
	error
	Location() SourceLocation
	Suggestion() string
	Code() ErrorCode
}

// ErrorCode represents different types of annotation errors
type ErrorCode int

const (
	SyntaxErrorCode ErrorCode = iota
	ValidationErrorCode
	SchemaErrorCode
	RegistrationErrorCode
)

// String returns the string representation of the error code
func (e ErrorCode) String() string {
	switch e {
	case SyntaxErrorCode:
		return "SyntaxError"
	case ValidationErrorCode:
		return "ValidationError"
	case SchemaErrorCode:
		return "SchemaError"
	case RegistrationErrorCode:
		return "RegistrationError"
	default:
		return "UnknownError"
	}
}

// ValidationError represents a parameter validation error
type ValidationError struct {
	Parameter string         // Parameter name that failed validation
	Expected  string         // What was expected
	Actual    string         // What was provided
	Loc       SourceLocation // Where the error occurred
	Hint      string         // Suggested fix
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("%s:%d:%d: parameter '%s' validation failed: expected %s, got %s. %s",
		e.Loc.File, e.Loc.Line, e.Loc.Column,
		e.Parameter, e.Expected, e.Actual, e.Hint)
}

func (e *ValidationError) Location() SourceLocation { return e.Loc }
func (e *ValidationError) Suggestion() string       { return e.Hint }
func (e *ValidationError) Code() ErrorCode          { return ValidationErrorCode }

// SyntaxError represents a syntax parsing error
type SyntaxError struct {
	Msg  string         // Error message
	Loc  SourceLocation // Where the error occurred
	Hint string         // Suggested fix
}

func (e *SyntaxError) Error() string {
	return fmt.Sprintf("%s:%d:%d: syntax error: %s. %s",
		e.Loc.File, e.Loc.Line, e.Loc.Column, e.Msg, e.Hint)
}

func (e *SyntaxError) Location() SourceLocation { return e.Loc }
func (e *SyntaxError) Suggestion() string       { return e.Hint }
func (e *SyntaxError) Code() ErrorCode          { return SyntaxErrorCode }

// SchemaError represents a schema-related error
type SchemaError struct {
	Msg  string         // Error message
	Loc  SourceLocation // Where the error occurred
	Hint string         // Suggested fix
}

func (e *SchemaError) Error() string {
	return fmt.Sprintf("%s:%d:%d: schema error: %s. %s",
		e.Loc.File, e.Loc.Line, e.Loc.Column, e.Msg, e.Hint)
}

func (e *SchemaError) Location() SourceLocation { return e.Loc }
func (e *SchemaError) Suggestion() string       { return e.Hint }
func (e *SchemaError) Code() ErrorCode          { return SchemaErrorCode }

// RegistrationError represents an error during annotation type registration
type RegistrationError struct {
	Msg  string         // Error message
	Loc  SourceLocation // Where the error occurred (optional)
	Hint string         // Suggested fix
}

func (e *RegistrationError) Error() string {
	if e.Loc.File != "" {
		return fmt.Sprintf("%s:%d:%d: registration error: %s. %s",
			e.Loc.File, e.Loc.Line, e.Loc.Column, e.Msg, e.Hint)
	}
	return fmt.Sprintf("registration error: %s. %s", e.Msg, e.Hint)
}

func (e *RegistrationError) Location() SourceLocation { return e.Loc }
func (e *RegistrationError) Suggestion() string       { return e.Hint }
func (e *RegistrationError) Code() ErrorCode          { return RegistrationErrorCode }

// MultipleAnnotationErrors represents multiple annotation errors collected together
type MultipleAnnotationErrors struct {
	Errors []AnnotationError
}

func (e *MultipleAnnotationErrors) Error() string {
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

	return fmt.Sprintf("multiple annotation errors (%d total):\n%s", len(e.Errors), strings.Join(messages, "\n"))
}

// Unwrap returns the underlying errors for error inspection
func (e *MultipleAnnotationErrors) Unwrap() []error {
	errors := make([]error, len(e.Errors))
	for i, err := range e.Errors {
		errors[i] = err
	}
	return errors
}

// Is checks if any of the wrapped errors matches the target
func (e *MultipleAnnotationErrors) Is(target error) bool {
	for _, err := range e.Errors {
		if err == target {
			return true
		}
	}
	return false
}

// As finds the first error in the chain that matches the target type
func (e *MultipleAnnotationErrors) As(target interface{}) bool {
	for _, err := range e.Errors {
		if as, ok := err.(interface{ As(interface{}) bool }); ok && as.As(target) {
			return true
		}
	}
	return false
}

// GetByType returns all errors of a specific type
func (e *MultipleAnnotationErrors) GetByType(code ErrorCode) []AnnotationError {
	var result []AnnotationError
	for _, err := range e.Errors {
		if err.Code() == code {
			result = append(result, err)
		}
	}
	return result
}

// HasType returns true if any error of the specified type exists
func (e *MultipleAnnotationErrors) HasType(code ErrorCode) bool {
	for _, err := range e.Errors {
		if err.Code() == code {
			return true
		}
	}
	return false
}

// MultipleValidationErrors represents multiple validation errors collected together
// Deprecated: Use MultipleAnnotationErrors instead
type MultipleValidationErrors struct {
	Errors []error
}

func (e *MultipleValidationErrors) Error() string {
	if len(e.Errors) == 0 {
		return "no errors"
	}

	if len(e.Errors) == 1 {
		return e.Errors[0].Error()
	}

	var messages []string
	for _, err := range e.Errors {
		messages = append(messages, err.Error())
	}

	return fmt.Sprintf("multiple validation errors:\n%s", strings.Join(messages, "\n"))
}

// Unwrap returns the underlying errors for error inspection
func (e *MultipleValidationErrors) Unwrap() []error {
	return e.Errors
}

// Is checks if any of the wrapped errors matches the target
func (e *MultipleValidationErrors) Is(target error) bool {
	for _, err := range e.Errors {
		if err == target {
			return true
		}
	}
	return false
}

// As finds the first error in the chain that matches the target type
func (e *MultipleValidationErrors) As(target interface{}) bool {
	for _, err := range e.Errors {
		if as, ok := err.(interface{ As(interface{}) bool }); ok && as.As(target) {
			return true
		}
	}
	return false
}

// Context-aware error message generators with fix suggestions

// NewSyntaxErrorWithContext creates a syntax error with context-aware suggestions
func NewSyntaxErrorWithContext(msg string, loc SourceLocation, context string) *SyntaxError {
	suggestion := generateSyntaxSuggestion(msg, context)
	return &SyntaxError{
		Msg:  msg,
		Loc:  loc,
		Hint: suggestion,
	}
}

// NewValidationErrorWithContext creates a validation error with context-aware suggestions
func NewValidationErrorWithContext(parameter, expected, actual string, loc SourceLocation, annotationType AnnotationType) *ValidationError {
	suggestion := generateValidationSuggestion(parameter, expected, actual, annotationType)
	return &ValidationError{
		Parameter: parameter,
		Expected:  expected,
		Actual:    actual,
		Loc:       loc,
		Hint:      suggestion,
	}
}

// NewSchemaErrorWithContext creates a schema error with context-aware suggestions
func NewSchemaErrorWithContext(msg string, loc SourceLocation, annotationType AnnotationType) *SchemaError {
	suggestion := generateSchemaSuggestion(msg, annotationType)
	return &SchemaError{
		Msg:  msg,
		Loc:  loc,
		Hint: suggestion,
	}
}

// generateSyntaxSuggestion provides context-aware suggestions for syntax errors
func generateSyntaxSuggestion(msg, context string) string {
	msg = strings.ToLower(msg)
	context = strings.ToLower(context)

	switch {
	case strings.Contains(msg, "missing annotation type"):
		return "Try: //axon::core or //axon::route GET /path"
	case strings.Contains(msg, "invalid annotation prefix"):
		return "Annotation must start with '//axon::' (note the double colon)"
	case strings.Contains(msg, "unterminated quoted string"):
		return "Make sure quoted strings are properly closed with matching quotes"
	case strings.Contains(msg, "invalid parameter format"):
		return "Parameters should be in format '-ParamName=Value' or '-FlagName' for boolean flags"
	case strings.Contains(msg, "unexpected token"):
		if strings.Contains(context, "route") {
			return "Route format: //axon::route METHOD /path [-Middleware=Name1,Name2] [-PassContext]"
		}
		if strings.Contains(context, "core") {
			return "Core format: //axon::core [-Mode=Singleton|Transient] [-Init=Same|Background] [-Manual=ModuleName]"
		}
		return "Check annotation syntax and parameter format"
	case strings.Contains(msg, "missing required parameter"):
		if strings.Contains(context, "route") {
			return "Route annotations require HTTP method and path: //axon::route GET /users"
		}
		return "Check annotation schema for required parameters"
	default:
		return "Check annotation syntax and refer to documentation for examples"
	}
}

// generateValidationSuggestion provides context-aware suggestions for validation errors
func generateValidationSuggestion(parameter, expected, actual string, annotationType AnnotationType) string {
	switch annotationType {
	case CoreAnnotation:
		return generateCoreValidationSuggestion(parameter, expected, actual)
	case RouteAnnotation:
		return generateRouteValidationSuggestion(parameter, expected, actual)
	case ControllerAnnotation:
		return generateControllerValidationSuggestion(parameter, expected, actual)
	case MiddlewareAnnotation:
		return generateMiddlewareValidationSuggestion(parameter, expected, actual)
	case InterfaceAnnotation:
		return generateInterfaceValidationSuggestion(parameter, expected, actual)
	default:
		return fmt.Sprintf("Parameter '%s' should be %s, not '%s'", parameter, expected, actual)
	}
}

// generateCoreValidationSuggestion provides suggestions for core annotation validation errors
func generateCoreValidationSuggestion(parameter, expected, actual string) string {
	switch parameter {
	case "Mode":
		return "Mode must be 'Singleton' (default) or 'Transient'. Example: -Mode=Transient"
	case "Init":
		return "Init must be 'Same' (default, synchronous) or 'Background' (async). Example: -Init=Background"
	case "Manual":
		return "Manual should be a custom module name. Example: -Manual=\"CustomModule\""
	default:
		return fmt.Sprintf("Core annotation parameter '%s' should be %s, got '%s'", parameter, expected, actual)
	}
}

// generateRouteValidationSuggestion provides suggestions for route annotation validation errors
func generateRouteValidationSuggestion(parameter, expected, actual string) string {
	switch parameter {
	case "method":
		return "HTTP method must be one of: GET, POST, PUT, DELETE, PATCH, HEAD, OPTIONS. Example: GET"
	case "path":
		return "Path should be a valid URL pattern. Examples: /users, /users/{id:int}, /api/v1/health"
	case "Middleware":
		return "Middleware should be comma-separated names. Example: -Middleware=Auth,Logging"
	case "PassContext":
		return "PassContext is a boolean flag. Use: -PassContext (no value needed)"
	default:
		return fmt.Sprintf("Route annotation parameter '%s' should be %s, got '%s'", parameter, expected, actual)
	}
}

// generateControllerValidationSuggestion provides suggestions for controller annotation validation errors
func generateControllerValidationSuggestion(parameter, expected, actual string) string {
	switch parameter {
	case "Path":
		return "Controller path should be a base URL path. Example: -Path=/api/v1"
	case "Middleware":
		return "Middleware should be comma-separated names. Example: -Middleware=Auth,Logging"
	default:
		return fmt.Sprintf("Controller annotation parameter '%s' should be %s, got '%s'", parameter, expected, actual)
	}
}

// generateMiddlewareValidationSuggestion provides suggestions for middleware annotation validation errors
func generateMiddlewareValidationSuggestion(parameter, expected, actual string) string {
	switch parameter {
	case "Priority":
		return "Priority should be an integer. Lower numbers execute first. Example: -Priority=10"
	case "Global":
		return "Global is a boolean flag. Use: -Global (no value needed)"
	default:
		return fmt.Sprintf("Middleware annotation parameter '%s' should be %s, got '%s'", parameter, expected, actual)
	}
}

// generateInterfaceValidationSuggestion provides suggestions for interface annotation validation errors
func generateInterfaceValidationSuggestion(parameter, expected, actual string) string {
	switch parameter {
	case "Implementation":
		return "Implementation should be the implementing struct name. Example: -Implementation=UserService"
	default:
		return fmt.Sprintf("Interface annotation parameter '%s' should be %s, got '%s'", parameter, expected, actual)
	}
}

// generateSchemaSuggestion provides context-aware suggestions for schema errors
func generateSchemaSuggestion(msg string, annotationType AnnotationType) string {
	msg = strings.ToLower(msg)

	switch {
	case strings.Contains(msg, "unknown annotation type"):
		return fmt.Sprintf("Supported annotation types: core, route, controller, middleware, interface. Did you mean one of these?")
	case strings.Contains(msg, "schema not found"):
		return fmt.Sprintf("Annotation type '%s' is not registered. Make sure to register the schema first.", annotationType.String())
	case strings.Contains(msg, "parameter not defined"):
		switch annotationType {
		case CoreAnnotation:
			return "Core annotation supports: Mode, Init, Manual parameters"
		case RouteAnnotation:
			return "Route annotation supports: method, path, Middleware, PassContext parameters"
		case ControllerAnnotation:
			return "Controller annotation supports: Path, Middleware parameters"
		case MiddlewareAnnotation:
			return "Middleware annotation supports: Priority, Global parameters"
		case InterfaceAnnotation:
			return "Interface annotation supports: Implementation parameter"
		default:
			return "Check annotation schema documentation for supported parameters"
		}
	default:
		return "Check annotation schema and parameter definitions"
	}
}

// ErrorSummary provides a summary of errors by type for better reporting
type ErrorSummary struct {
	SyntaxErrors     []AnnotationError
	ValidationErrors []AnnotationError
	SchemaErrors     []AnnotationError
	OtherErrors      []AnnotationError
	TotalCount       int
}

// SummarizeErrors creates an error summary from a collection of errors
func SummarizeErrors(errors []AnnotationError) ErrorSummary {
	summary := ErrorSummary{
		TotalCount: len(errors),
	}

	for _, err := range errors {
		switch err.Code() {
		case SyntaxErrorCode:
			summary.SyntaxErrors = append(summary.SyntaxErrors, err)
		case ValidationErrorCode:
			summary.ValidationErrors = append(summary.ValidationErrors, err)
		case SchemaErrorCode:
			summary.SchemaErrors = append(summary.SchemaErrors, err)
		default:
			summary.OtherErrors = append(summary.OtherErrors, err)
		}
	}

	return summary
}

// String returns a formatted summary of errors
func (s ErrorSummary) String() string {
	if s.TotalCount == 0 {
		return "No errors found"
	}

	var parts []string
	if len(s.SyntaxErrors) > 0 {
		parts = append(parts, fmt.Sprintf("%d syntax error(s)", len(s.SyntaxErrors)))
	}
	if len(s.ValidationErrors) > 0 {
		parts = append(parts, fmt.Sprintf("%d validation error(s)", len(s.ValidationErrors)))
	}
	if len(s.SchemaErrors) > 0 {
		parts = append(parts, fmt.Sprintf("%d schema error(s)", len(s.SchemaErrors)))
	}
	if len(s.OtherErrors) > 0 {
		parts = append(parts, fmt.Sprintf("%d other error(s)", len(s.OtherErrors)))
	}

	return fmt.Sprintf("Found %d total error(s): %s", s.TotalCount, strings.Join(parts, ", "))
}
