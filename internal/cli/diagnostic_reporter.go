package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/fatih/color"
	"github.com/toyz/axon/internal/models"
)

// DiagnosticReporter provides user-friendly error reporting and diagnostics
type DiagnosticReporter struct {
	verbose bool
}

// NewDiagnosticReporter creates a new diagnostic reporter
func NewDiagnosticReporter(verbose bool) *DiagnosticReporter {
	return &DiagnosticReporter{
		verbose: verbose,
	}
}

// ReportWarning provides user-friendly warning reporting
func (r *DiagnosticReporter) ReportWarning(message string, suggestions ...string) {
	// Clean warning format with orange color
	orange := color.New(color.FgYellow, color.Bold) // Orange-ish color using yellow + bold
	orange.Fprint(os.Stderr, "! ")
	fmt.Fprintf(os.Stderr, "%s\n", message)
}

// ReportError provides comprehensive error reporting with user-friendly output
func (r *DiagnosticReporter) ReportError(err error) {
	fmt.Fprintf(os.Stderr, "\nERROR: Code Generation Failed\n")
	fmt.Fprintf(os.Stderr, "=============================\n\n")

	// Check if it's a GeneratorError with rich information
	if genErr, ok := err.(*models.GeneratorError); ok {
		r.reportGeneratorError(genErr)
	} else {
		// Try to unwrap and find a GeneratorError
		if unwrapped := r.findGeneratorError(err); unwrapped != nil {
			r.reportGeneratorError(unwrapped)
		} else {
			// Fallback to basic error reporting
			r.reportBasicError(err)
		}
	}

	fmt.Fprintf(os.Stderr, "\n")
}

// reportGeneratorError reports a GeneratorError with full context and suggestions
func (r *DiagnosticReporter) reportGeneratorError(genErr *models.GeneratorError) {
	// Error type and location
	r.printErrorHeader(genErr)

	// Main error message
	fmt.Fprintf(os.Stderr, "Message: %s\n\n", genErr.Message)
	
	// In verbose mode, show the underlying cause if available
	if r.verbose && genErr.Cause != nil {
		fmt.Fprintf(os.Stderr, "Underlying cause: %s\n\n", genErr.Cause.Error())
	}

	// File and line information
	if genErr.File != "" {
		if genErr.Line > 0 {
			fmt.Fprintf(os.Stderr, "Location: %s:%d\n\n", genErr.File, genErr.Line)
		} else {
			fmt.Fprintf(os.Stderr, "File: %s\n\n", genErr.File)
		}
	}

	// Context information
	if genErr.Context != nil && len(genErr.Context) > 0 {
		r.printContext(genErr.Context)
	}

	// Suggestions
	if len(genErr.Suggestions) > 0 {
		r.printSuggestions(genErr.Suggestions)
	}

	// Additional help based on error type
	r.printAdditionalHelp(genErr.Type)
	
	// In verbose mode, show additional debugging information
	if r.verbose {
		r.printVerboseDebuggingInfo(genErr)
	}
}

// reportBasicError reports a basic error without rich context
func (r *DiagnosticReporter) reportBasicError(err error) {
	fmt.Fprintf(os.Stderr, "Message: %s\n\n", err.Error())

	// Try to provide some general guidance based on error message
	errorMsg := strings.ToLower(err.Error())
	
	if strings.Contains(errorMsg, "parser") {
		fmt.Fprintf(os.Stderr, "This appears to be a parser-related issue.\n")
		fmt.Fprintf(os.Stderr, "Common solutions:\n")
		fmt.Fprintf(os.Stderr, "  - Check your //axon::route_parser annotations\n")
		fmt.Fprintf(os.Stderr, "  - Ensure parser functions have the correct signature\n")
		fmt.Fprintf(os.Stderr, "  - Verify all required imports are present\n\n")
	} else if strings.Contains(errorMsg, "annotation") {
		fmt.Fprintf(os.Stderr, "This appears to be an annotation-related issue.\n")
		fmt.Fprintf(os.Stderr, "Common solutions:\n")
		fmt.Fprintf(os.Stderr, "  - Check your //axon:: annotation syntax\n")
		fmt.Fprintf(os.Stderr, "  - Ensure annotations are properly formatted\n")
		fmt.Fprintf(os.Stderr, "  - Verify annotation targets are correct\n\n")
	} else if strings.Contains(errorMsg, "module") {
		fmt.Fprintf(os.Stderr, "This appears to be a module-related issue.\n")
		fmt.Fprintf(os.Stderr, "Common solutions:\n")
		fmt.Fprintf(os.Stderr, "  - Check your go.mod file\n")
		fmt.Fprintf(os.Stderr, "  - Ensure module paths are correct\n")
		fmt.Fprintf(os.Stderr, "  - Try specifying --module flag explicitly\n\n")
	}
}

// printErrorHeader prints a formatted error header based on error type
func (r *DiagnosticReporter) printErrorHeader(genErr *models.GeneratorError) {
	var errorTypeStr string
	
	switch genErr.Type {
	case models.ErrorTypeParserValidation:
		errorTypeStr = "Parser Validation Error"
	case models.ErrorTypeParserRegistration:
		errorTypeStr = "Parser Registration Error"
	case models.ErrorTypeParserImport:
		errorTypeStr = "Parser Import Error"
	case models.ErrorTypeParserConflict:
		errorTypeStr = "Parser Conflict Error"
	case models.ErrorTypeAnnotationSyntax:
		errorTypeStr = "Annotation Syntax Error"
	case models.ErrorTypeValidation:
		errorTypeStr = "Validation Error"
	case models.ErrorTypeGeneration:
		errorTypeStr = "Code Generation Error"
	case models.ErrorTypeFileSystem:
		errorTypeStr = "File System Error"
	default:
		errorTypeStr = "Unknown Error"
	}
	
	fmt.Fprintf(os.Stderr, "Type: %s\n", errorTypeStr)
	fmt.Fprintf(os.Stderr, "%s\n\n", strings.Repeat("-", len(errorTypeStr)+6))
}

// printContext prints context information in a readable format
func (r *DiagnosticReporter) printContext(context map[string]interface{}) {
	fmt.Fprintf(os.Stderr, "Context:\n")
	
	// Print important context items first
	importantKeys := []string{"function_name", "type_name", "route_method", "route_path", "parameter_name"}
	printed := make(map[string]bool)
	
	for _, key := range importantKeys {
		if value, exists := context[key]; exists {
			fmt.Fprintf(os.Stderr, "   %s: %v\n", r.formatContextKey(key), value)
			printed[key] = true
		}
	}
	
	// Print remaining context items
	for key, value := range context {
		if !printed[key] {
			fmt.Fprintf(os.Stderr, "   %s: %v\n", r.formatContextKey(key), value)
		}
	}
	
	fmt.Fprintf(os.Stderr, "\n")
}

// formatContextKey formats context keys to be more readable
func (r *DiagnosticReporter) formatContextKey(key string) string {
	switch key {
	case "function_name":
		return "Function"
	case "type_name":
		return "Type"
	case "route_method":
		return "Route Method"
	case "route_path":
		return "Route Path"
	case "parameter_name":
		return "Parameter"
	case "expected_signature":
		return "Expected Signature"
	case "actual_signature":
		return "Actual Signature"
	case "required_import":
		return "Required Import"
	default:
		// Convert snake_case to Title Case
		parts := strings.Split(key, "_")
		for i, part := range parts {
			if len(part) > 0 {
				parts[i] = strings.ToUpper(part[:1]) + part[1:]
			}
		}
		return strings.Join(parts, " ")
	}
}

// printSuggestions prints actionable suggestions
func (r *DiagnosticReporter) printSuggestions(suggestions []string) {
	fmt.Fprintf(os.Stderr, "Suggestions:\n")
	
	for i, suggestion := range suggestions {
		// Format multi-line suggestions nicely
		lines := strings.Split(suggestion, "\n")
		if len(lines) == 1 {
			fmt.Fprintf(os.Stderr, "   %d. %s\n", i+1, suggestion)
		} else {
			fmt.Fprintf(os.Stderr, "   %d. %s\n", i+1, lines[0])
			for _, line := range lines[1:] {
				if strings.TrimSpace(line) != "" {
					fmt.Fprintf(os.Stderr, "      %s\n", line)
				}
			}
		}
	}
	
	fmt.Fprintf(os.Stderr, "\n")
}

// printAdditionalHelp prints additional help based on error type
func (r *DiagnosticReporter) printAdditionalHelp(errorType models.ErrorType) {
	switch errorType {
	case models.ErrorTypeParserValidation:
		fmt.Fprintf(os.Stderr, "Parser Function Requirements:\n")
		fmt.Fprintf(os.Stderr, "  - Must have exactly 2 parameters: (echo.Context, string)\n")
		fmt.Fprintf(os.Stderr, "  - Must return exactly 2 values: (YourType, error)\n")
		fmt.Fprintf(os.Stderr, "  - Must be a regular function (not a method)\n")
		fmt.Fprintf(os.Stderr, "  - Must be in the same file as the annotation\n\n")
		
	case models.ErrorTypeParserImport:
		fmt.Fprintf(os.Stderr, "Import Requirements:\n")
		fmt.Fprintf(os.Stderr, "  - Ensure all required packages are imported\n")
		fmt.Fprintf(os.Stderr, "  - Check that import paths are correct\n")
		fmt.Fprintf(os.Stderr, "  - Run 'go mod tidy' to ensure dependencies are available\n\n")
		
	case models.ErrorTypeParserConflict:
		fmt.Fprintf(os.Stderr, "Resolving Parser Conflicts:\n")
		fmt.Fprintf(os.Stderr, "  - Each type can only have one parser\n")
		fmt.Fprintf(os.Stderr, "  - Remove duplicate parser registrations\n")
		fmt.Fprintf(os.Stderr, "  - Consider using different type names if you need multiple parsers\n\n")
		
	case models.ErrorTypeAnnotationSyntax:
		fmt.Fprintf(os.Stderr, "Annotation Syntax Help:\n")
		fmt.Fprintf(os.Stderr, "  - Annotations must start with //axon::\n")
		fmt.Fprintf(os.Stderr, "  - Check the documentation for correct syntax\n")
		fmt.Fprintf(os.Stderr, "  - Ensure proper spacing and parameter format\n\n")
	}
	
	// Always show general help
	fmt.Fprintf(os.Stderr, "For more help:\n")
	fmt.Fprintf(os.Stderr, "  - Check the Axon documentation\n")
	fmt.Fprintf(os.Stderr, "  - Run with --verbose for more detailed output\n")
	fmt.Fprintf(os.Stderr, "  - Review example implementations in the examples/ directory\n")
}

// findGeneratorError recursively searches for a GeneratorError in wrapped errors
func (r *DiagnosticReporter) findGeneratorError(err error) *models.GeneratorError {
	if err == nil {
		return nil
	}
	
	// Check if this is a GeneratorError
	if genErr, ok := err.(*models.GeneratorError); ok {
		return genErr
	}
	
	// Try to unwrap and search recursively
	if unwrapper, ok := err.(interface{ Unwrap() error }); ok {
		return r.findGeneratorError(unwrapper.Unwrap())
	}
	
	return nil
}

// printVerboseDebuggingInfo prints additional debugging information in verbose mode
func (r *DiagnosticReporter) printVerboseDebuggingInfo(genErr *models.GeneratorError) {
	fmt.Fprintf(os.Stderr, "Verbose Debug Information:\n")
	fmt.Fprintf(os.Stderr, "  Error Type Code: %d\n", int(genErr.Type))
	
	if genErr.Context != nil {
		fmt.Fprintf(os.Stderr, "  Full Context Data:\n")
		for key, value := range genErr.Context {
			fmt.Fprintf(os.Stderr, "    %s: %+v\n", key, value)
		}
	}
	
	if genErr.Cause != nil {
		fmt.Fprintf(os.Stderr, "  Error Chain:\n")
		err := genErr.Cause
		level := 1
		for err != nil {
			fmt.Fprintf(os.Stderr, "    %d. %s\n", level, err.Error())
			if unwrapper, ok := err.(interface{ Unwrap() error }); ok {
				err = unwrapper.Unwrap()
				level++
			} else {
				break
			}
		}
	}
	
	fmt.Fprintf(os.Stderr, "\n")
}

// Debug prints debug information when verbose mode is enabled
func (r *DiagnosticReporter) Debug(format string, args ...interface{}) {
	if r.verbose {
		fmt.Fprintf(os.Stderr, "[DEBUG] "+format+"\n", args...)
	}
}

// DebugSection prints a debug section header when verbose mode is enabled
func (r *DiagnosticReporter) DebugSection(section string) {
	if r.verbose {
		fmt.Fprintf(os.Stderr, "[DEBUG] === %s ===\n", section)
	}
}

// ReportSuccess reports successful generation with summary information
func (r *DiagnosticReporter) ReportSuccess(summary GenerationSummary) {
	fmt.Printf("\nCode Generation Completed Successfully!\n")
	fmt.Printf("=======================================\n\n")
	
	if summary.PackagesProcessed > 0 {
		fmt.Printf("Processed %d packages\n", summary.PackagesProcessed)
	}
	
	if summary.ModulesGenerated > 0 {
		fmt.Printf("Generated %d FX modules\n", summary.ModulesGenerated)
	}
	
	if summary.ParsersDiscovered > 0 {
		fmt.Printf("Discovered %d custom parsers\n", summary.ParsersDiscovered)
	}
	
	if summary.ControllersFound > 0 {
		fmt.Printf("Found %d controllers\n", summary.ControllersFound)
	}
	
	if summary.ServicesFound > 0 {
		fmt.Printf("Found %d services\n", summary.ServicesFound)
	}
	
	if summary.MiddlewaresFound > 0 {
		fmt.Printf("Found %d middlewares\n", summary.MiddlewaresFound)
	}
	
	if len(summary.GeneratedFiles) > 0 {
		fmt.Printf("\nGenerated files:\n")
		for _, file := range summary.GeneratedFiles {
			fmt.Printf("  - %s\n", file)
		}
	}
	
	fmt.Printf("\nYour Axon application is ready to use!\n")
}

// GenerationSummary contains information about the generation process
type GenerationSummary struct {
	PackagesProcessed int
	ModulesGenerated  int
	ParsersDiscovered int
	ControllersFound  int
	ServicesFound     int
	MiddlewaresFound  int
	GeneratedFiles    []string
}