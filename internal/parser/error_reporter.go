package parser

import (
	"fmt"
	"strings"

	"github.com/toyz/axon/internal/models"
)

// ParserErrorReporter provides comprehensive error reporting for parser-related issues
type ParserErrorReporter struct {
	parser *Parser
}

// NewParserErrorReporter creates a new parser error reporter
func NewParserErrorReporter(parser *Parser) *ParserErrorReporter {
	return &ParserErrorReporter{
		parser: parser,
	}
}

// ReportParserValidationError creates a detailed parser validation error with context and suggestions
func (r *ParserErrorReporter) ReportParserValidationError(functionName, fileName string, line int, issue string, actualSignature string) error {
	expectedSignature := "func(c echo.Context, paramValue string) (T, error)"

	suggestions := []string{
		"Expected signature: " + expectedSignature,
	}

	// Add specific suggestions based on the issue
	switch {
	case strings.Contains(issue, "parameters"):
		suggestions = append(suggestions,
			"Ensure the function has exactly 2 parameters",
			"First parameter should be echo.Context",
			"Second parameter should be string",
		)
	case strings.Contains(issue, "first parameter"):
		suggestions = append(suggestions,
			"Import the Echo framework: github.com/labstack/echo/v4",
			"Use 'c echo.Context' as the first parameter",
		)
	case strings.Contains(issue, "second parameter"):
		suggestions = append(suggestions,
			"Use 'paramValue string' as the second parameter",
		)
	case strings.Contains(issue, "return"):
		suggestions = append(suggestions,
			"Function must return exactly 2 values",
			"First return value should be the parsed type (T)",
			"Second return value should be error",
		)
	case strings.Contains(issue, "function not found"):
		suggestions = append(suggestions,
			"Ensure the function is defined in the same file as the annotation",
			"Check that the function name matches the annotation target",
			"Ensure the function is not a method (no receiver)",
		)
	}

	// Add example based on common parser patterns
	example := r.generateParserExample(functionName)
	if example != "" {
		suggestions = append(suggestions, "Example implementation:", example)
	}

	return &models.GeneratorError{
		Type:        models.ErrorTypeParserValidation,
		File:        fileName,
		Line:        line,
		Message:     fmt.Sprintf("Parser function '%s' has invalid signature: %s", functionName, issue),
		Suggestions: suggestions,
		Context: map[string]interface{}{
			"function_name":      functionName,
			"expected_signature": expectedSignature,
			"actual_signature":   actualSignature,
			"issue":              issue,
		},
	}
}

// ReportParserNotFoundError creates a detailed error when a parser is not found for a type
func (r *ParserErrorReporter) ReportParserNotFoundError(typeName, routeMethod, routePath, paramName, fileName string, line int, availableParsers []string) error {
	suggestions := []string{
		fmt.Sprintf("Register a parser for type '%s' using //axon::route_parser %s", typeName, typeName),
		"Check if the type name is spelled correctly",
	}

	// Add suggestions based on the type name
	if strings.Contains(typeName, "UUID") || strings.Contains(typeName, "uuid") {
		suggestions = append(suggestions,
			"For UUID types, use 'uuid.UUID' and import 'github.com/google/uuid'",
			"Example: //axon::route_parser uuid.UUID",
		)
	}

	if strings.Contains(typeName, "Time") || strings.Contains(typeName, "time") {
		suggestions = append(suggestions,
			"For time types, use 'time.Time' and import 'time'",
			"Example: //axon::route_parser time.Time",
		)
	}

	// Add available parsers information
	if len(availableParsers) > 0 {
		suggestions = append(suggestions, fmt.Sprintf("Available parsers: %s", strings.Join(availableParsers, ", ")))
	} else {
		suggestions = append(suggestions, "No parsers are currently registered")
		suggestions = append(suggestions, "Consider adding built-in parsers or creating custom ones")
	}

	// Add example parser implementation
	example := r.generateParserExample("Parse" + r.capitalizeFirst(typeName))
	if example != "" {
		suggestions = append(suggestions, "Example parser implementation:", example)
	}

	return &models.GeneratorError{
		Type:        models.ErrorTypeParserValidation,
		File:        fileName,
		Line:        line,
		Message:     fmt.Sprintf("No parser registered for custom type '%s' used in route %s %s (parameter '%s')", typeName, routeMethod, routePath, paramName),
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

// ReportParserImportError creates a detailed error when required imports are missing
func (r *ParserErrorReporter) ReportParserImportError(typeName, fileName string, line int, requiredImport string) error {
	suggestions := []string{
		fmt.Sprintf("Add import: %s", requiredImport),
		"Ensure the package containing the type is imported",
	}

	// Add specific suggestions based on the import
	switch requiredImport {
	case "github.com/google/uuid":
		suggestions = append(suggestions,
			"Install the UUID package: go get github.com/google/uuid",
			"Add to imports: import \"github.com/google/uuid\"",
		)
	case "time":
		suggestions = append(suggestions,
			"Add to imports: import \"time\"",
		)
	case "net/url":
		suggestions = append(suggestions,
			"Add to imports: import \"net/url\"",
		)
	}

	return &models.GeneratorError{
		Type:        models.ErrorTypeParserImport,
		File:        fileName,
		Line:        line,
		Message:     fmt.Sprintf("Parser for type '%s' requires missing import: %s", typeName, requiredImport),
		Suggestions: suggestions,
		Context: map[string]interface{}{
			"type_name":       typeName,
			"required_import": requiredImport,
		},
	}
}

// ReportParserConflictError creates a detailed error when parser conflicts are detected
func (r *ParserErrorReporter) ReportParserConflictError(typeName string, conflicts []models.ParserConflict) error {
	var conflictDetails []string
	for _, conflict := range conflicts {
		conflictDetails = append(conflictDetails, fmt.Sprintf("%s:%d (%s)", conflict.FileName, conflict.Line, conflict.FunctionName))
	}

	suggestions := []string{
		"Keep only one parser registration for each type",
		"Remove duplicate parser annotations",
		"Consider using different type names if you need multiple parsers",
		fmt.Sprintf("Conflicting registrations found at: %s", strings.Join(conflictDetails, ", ")),
	}

	return &models.GeneratorError{
		Type:        models.ErrorTypeParserConflict,
		Message:     fmt.Sprintf("Multiple parsers registered for type '%s'", typeName),
		Suggestions: suggestions,
		Context: map[string]interface{}{
			"type_name": typeName,
			"conflicts": conflicts,
		},
	}
}

// generateParserExample generates an example parser implementation based on the function name
func (r *ParserErrorReporter) generateParserExample(functionName string) string {
	// Generate example based on common patterns
	if strings.Contains(strings.ToLower(functionName), "uuid") {
		return `//axon::route_parser uuid.UUID
func ParseUUID(c echo.Context, paramValue string) (uuid.UUID, error) {
    return uuid.Parse(paramValue)
}`
	}

	if strings.Contains(strings.ToLower(functionName), "time") {
		return `//axon::route_parser time.Time
func ParseTime(c echo.Context, paramValue string) (time.Time, error) {
    return time.Parse(time.RFC3339, paramValue)
}`
	}

	if strings.Contains(strings.ToLower(functionName), "int") {
		return `//axon::route_parser CustomInt
func ParseCustomInt(c echo.Context, paramValue string) (CustomInt, error) {
    val, err := strconv.Atoi(paramValue)
    if err != nil {
        return CustomInt(0), err
    }
    return CustomInt(val), nil
}`
	}

	// Generic example
	return fmt.Sprintf(`//axon::route_parser YourType
func %s(c echo.Context, paramValue string) (YourType, error) {
    // Parse paramValue and return YourType
    // Return error if parsing fails
    return YourType{}, nil
}`, functionName)
}

// capitalizeFirst capitalizes the first letter of a string
func (r *ParserErrorReporter) capitalizeFirst(s string) string {
	if len(s) == 0 {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

// GenerateParserDiagnostics generates comprehensive diagnostics for parser-related issues
func (r *ParserErrorReporter) GenerateParserDiagnostics(metadata *models.PackageMetadata) []string {
	var diagnostics []string

	// Check for common parser issues
	if len(metadata.RouteParsers) == 0 {
		diagnostics = append(diagnostics, "No custom parsers found. Consider adding parsers for complex types.")
	}

	// Check for unused parsers
	usedParsers := make(map[string]bool)
	for _, controller := range metadata.Controllers {
		for _, route := range controller.Routes {
			for _, param := range route.Parameters {
				if param.IsCustomType {
					usedParsers[param.Type] = true
				}
			}
		}
	}

	for _, parser := range metadata.RouteParsers {
		if !usedParsers[parser.TypeName] {
			diagnostics = append(diagnostics, fmt.Sprintf("Parser for type '%s' is defined but not used in any routes", parser.TypeName))
		}
	}

	// Check for missing parsers for custom types
	for _, controller := range metadata.Controllers {
		for _, route := range controller.Routes {
			for _, param := range route.Parameters {
				if param.IsCustomType && param.ParserFunc == "" {
					diagnostics = append(diagnostics, fmt.Sprintf("Route %s %s uses custom type '%s' but no parser is registered", route.Method, route.Path, param.Type))
				}
			}
		}
	}

	return diagnostics
}
