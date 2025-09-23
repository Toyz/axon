package errors

import (
	"fmt"
	"strings"
)

// Annotations-specific error types that use the unified base types
// These replace the error types in internal/annotations/errors.go

// AnnotationType represents the type of annotation
type AnnotationType int

const (
	UnknownAnnotation AnnotationType = iota
	CoreAnnotation
	ServiceAnnotation
	RouteAnnotation
	ControllerAnnotation
	MiddlewareAnnotation
	InterfaceAnnotation
	InjectAnnotation
	InitAnnotation
	LoggerAnnotation
	RouteParserAnnotation
)

// String returns the string representation of the annotation type
func (a AnnotationType) String() string {
	switch a {
	case CoreAnnotation:
		return "core"
	case ServiceAnnotation:
		return "service"
	case RouteAnnotation:
		return "route"
	case ControllerAnnotation:
		return "controller"
	case MiddlewareAnnotation:
		return "middleware"
	case InterfaceAnnotation:
		return "interface"
	case InjectAnnotation:
		return "inject"
	case InitAnnotation:
		return "init"
	case LoggerAnnotation:
		return "logger"
	case RouteParserAnnotation:
		return "route_parser"
	default:
		return "unknown"
	}
}

// NewAnnotationValidationError creates a validation error specific to annotations
func NewAnnotationValidationError(parameter, expected, actual string, loc SourceLocation, annotationType AnnotationType) *ValidationError {
	err := NewValidationError(parameter, expected, actual)
	err.WithLocation(loc)
	err.WithContext("annotation_type", annotationType.String())

	// Add context-aware suggestions
	suggestion := generateValidationSuggestion(parameter, expected, actual, annotationType)
	if suggestion != "" {
		err.WithSuggestion(suggestion)
	}

	return err
}

// NewAnnotationSyntaxError creates a syntax error specific to annotations
func NewAnnotationSyntaxError(message string, loc SourceLocation, context string) *SyntaxError {
	err := NewSyntaxError(message)
	err.WithLocation(loc)
	err.WithContext("parse_context", context)

	// Add context-aware suggestions
	suggestion := generateSyntaxSuggestion(message, context)
	if suggestion != "" {
		err.WithSuggestion(suggestion)
	}

	return err
}

// NewAnnotationSchemaError creates a schema error specific to annotations
func NewAnnotationSchemaError(message string, loc SourceLocation, annotationType AnnotationType) *SchemaError {
	err := NewSchemaError(message)
	err.WithLocation(loc)
	err.WithSchemaType("annotation")
	err.WithSchemaName(annotationType.String())

	// Add context-aware suggestions
	suggestion := generateSchemaSuggestion(message, annotationType)
	if suggestion != "" {
		err.BaseError.WithSuggestion(suggestion)
	}

	return err
}

// NewAnnotationRegistrationError creates a registration error for annotations
func NewAnnotationRegistrationError(annotationType AnnotationType, message string, loc SourceLocation) *RegistrationError {
	err := NewRegistrationError("annotation", annotationType.String(), message)
	err.WithLocation(loc)
	return err
}

// AnnotationErrorCollector helps collect multiple annotation errors
type AnnotationErrorCollector struct {
	*MultipleErrors
	maxErrors int
}

// NewAnnotationErrorCollector creates a new error collector
func NewAnnotationErrorCollector(maxErrors int) *AnnotationErrorCollector {
	if maxErrors <= 0 {
		maxErrors = 100 // default maximum
	}

	return &AnnotationErrorCollector{
		MultipleErrors: NewMultipleErrors(),
		maxErrors:      maxErrors,
	}
}

// AddValidation adds a validation error to the collection
func (c *AnnotationErrorCollector) AddValidation(parameter, expected, actual string, loc SourceLocation, annotationType AnnotationType) {
	if c.Count() >= c.maxErrors {
		return // Don't collect more than maxErrors
	}

	err := NewAnnotationValidationError(parameter, expected, actual, loc, annotationType)
	c.Add(err)
}

// AddSyntax adds a syntax error to the collection
func (c *AnnotationErrorCollector) AddSyntax(message string, loc SourceLocation, context string) {
	if c.Count() >= c.maxErrors {
		return
	}

	err := NewAnnotationSyntaxError(message, loc, context)
	c.Add(err)
}

// AddSchema adds a schema error to the collection
func (c *AnnotationErrorCollector) AddSchema(message string, loc SourceLocation, annotationType AnnotationType) {
	if c.Count() >= c.maxErrors {
		return
	}

	err := NewAnnotationSchemaError(message, loc, annotationType)
	c.Add(err)
}

// AddRegistration adds a registration error to the collection
func (c *AnnotationErrorCollector) AddRegistration(annotationType AnnotationType, message string, loc SourceLocation) {
	if c.Count() >= c.maxErrors {
		return
	}

	err := NewAnnotationRegistrationError(annotationType, message, loc)
	c.Add(err)
}

// ToError returns the collected errors as a single error
func (c *AnnotationErrorCollector) ToError() AxonError {
	if c.IsEmpty() {
		return nil
	}

	if c.Count() == 1 {
		return c.Errors[0]
	}

	// MultipleErrors implements AxonError interface
	return c.MultipleErrors
}

// Annotation Error Summary for better reporting
type AnnotationErrorSummary struct {
	SyntaxErrors     []AxonError
	ValidationErrors []AxonError
	SchemaErrors     []AxonError
	OtherErrors      []AxonError
	TotalCount       int
}

// SummarizeAnnotationErrors creates an error summary from a collection of errors
func SummarizeAnnotationErrors(errors []AxonError) AnnotationErrorSummary {
	summary := AnnotationErrorSummary{
		TotalCount: len(errors),
	}

	for _, err := range errors {
		switch err.ErrorCode() {
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
func (s AnnotationErrorSummary) String() string {
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

// Context-aware error message generators with fix suggestions

// generateSyntaxSuggestion provides context-aware suggestions for syntax errors
func generateSyntaxSuggestion(msg, context string) string {
	msg = strings.ToLower(msg)
	context = strings.ToLower(context)

	switch {
	case strings.Contains(msg, "missing annotation type"):
		return "Try: //axon::service or //axon::route GET /path"
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
		if strings.Contains(context, "service") {
			return "Service format: //axon::service [-Mode=Singleton|Transient] [-Init=Same|Background] [-Manual=ModuleName]"
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
	case ServiceAnnotation:
		return generateServiceValidationSuggestion(parameter, expected, actual)
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

// generateServiceValidationSuggestion provides suggestions for service annotation validation errors
func generateServiceValidationSuggestion(parameter, expected, actual string) string {
	switch parameter {
	case "Mode":
		return "Mode must be 'Singleton' (default) or 'Transient'. Example: -Mode=Transient"
	case "Init":
		return "Init must be 'Same' (default, synchronous) or 'Background' (async). Example: -Init=Background"
	case "Manual":
		return "Manual should be a custom module name. Example: -Manual=\"CustomModule\""
	case "Constructor":
		return "Constructor should be a function name. Example: -Constructor=NewCustomService"
	default:
		return fmt.Sprintf("Service annotation parameter '%s' should be %s, got '%s'", parameter, expected, actual)
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
	case "Priority":
		return "Priority should be an integer. Example: -Priority=10"
	default:
		return fmt.Sprintf("Route annotation parameter '%s' should be %s, got '%s'", parameter, expected, actual)
	}
}

// generateControllerValidationSuggestion provides suggestions for controller annotation validation errors
func generateControllerValidationSuggestion(parameter, expected, actual string) string {
	switch parameter {
	case "Prefix":
		return "Controller prefix should be a base URL path. Example: -Prefix=/api/v1"
	case "Middleware":
		return "Middleware should be comma-separated names. Example: -Middleware=Auth,Logging"
	case "Priority":
		return "Priority should be an integer. Example: -Priority=10"
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
		return "Supported annotation types: service, route, controller, middleware, interface. Did you mean one of these?"
	case strings.Contains(msg, "schema not found"):
		return fmt.Sprintf("Annotation type '%s' is not registered. Make sure to register the schema first.", annotationType.String())
	case strings.Contains(msg, "parameter not defined"):
		switch annotationType {
		case ServiceAnnotation:
			return "Service annotation supports: Mode, Init, Manual, Constructor parameters"
		case RouteAnnotation:
			return "Route annotation supports: method, path, Middleware, PassContext, Priority parameters"
		case ControllerAnnotation:
			return "Controller annotation supports: Prefix, Middleware, Priority parameters"
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