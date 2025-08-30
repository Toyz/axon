package cli

import (
	"bytes"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/toyz/axon/internal/models"
)

func TestDiagnosticReporter_ReportWarning(t *testing.T) {
	// Capture stderr output
	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	reporter := NewDiagnosticReporter(false)

	// Test warning without suggestions
	reporter.ReportWarning("This is a test warning")

	// Test warning with suggestions
	reporter.ReportWarning("This is another warning",
		"First suggestion",
		"Second suggestion",
	)

	// Close writer and read output
	w.Close()
	os.Stderr = oldStderr

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	// Verify output contains expected elements (new clean format)
	if !strings.Contains(output, "! This is a test warning") {
		t.Errorf("Expected warning message not found in output")
	}

	if !strings.Contains(output, "! This is another warning") {
		t.Errorf("Expected second warning message not found in output")
	}

	// Note: Suggestions are no longer displayed in the clean format
}

func TestDiagnosticReporter_ReportGeneratorError(t *testing.T) {
	// Capture stderr output
	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	reporter := NewDiagnosticReporter(false)

	// Create a test GeneratorError
	genErr := &models.GeneratorError{
		Type:    models.ErrorTypeParserValidation,
		File:    "test.go",
		Line:    42,
		Message: "Parser function 'ParseUUID' has invalid signature",
		Suggestions: []string{
			"Expected signature: func(c echo.Context, paramValue string) (T, error)",
			"Ensure the first parameter is echo.Context",
			"Ensure the second parameter is string",
		},
		Context: map[string]interface{}{
			"function_name":      "ParseUUID",
			"expected_signature": "func(c echo.Context, paramValue string) (T, error)",
			"actual_signature":   "func ParseUUID(s string) (uuid.UUID, error)",
		},
	}

	// Report the error
	reporter.ReportError(genErr)

	// Close writer and read output
	w.Close()
	os.Stderr = oldStderr

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	// Verify output contains expected elements
	expectedElements := []string{
		"ERROR: Code Generation Failed",
		"Parser Validation Error",
		"Message: Parser function 'ParseUUID' has invalid signature",
		"Location: test.go:42",
		"Context:",
		"Function: ParseUUID",
		"Suggestions:",
		"Expected signature: func(c echo.Context, paramValue string) (T, error)",
		"Parser Function Requirements:",
		"Must have exactly 2 parameters",
	}

	for _, expected := range expectedElements {
		if !strings.Contains(output, expected) {
			t.Errorf("Output should contain '%s', got:\n%s", expected, output)
		}
	}
}

func TestDiagnosticReporter_ReportBasicError(t *testing.T) {
	// Capture stderr output
	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	reporter := NewDiagnosticReporter(false)

	// Create a basic error
	err := fmt.Errorf("parser validation failed: invalid function signature")

	// Report the error
	reporter.ReportError(err)

	// Close writer and read output
	w.Close()
	os.Stderr = oldStderr

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	// Verify output contains expected elements
	expectedElements := []string{
		"ERROR: Code Generation Failed",
		"Message: parser validation failed: invalid function signature",
		"This appears to be a parser-related issue",
		"Check your //axon::route_parser annotations",
	}

	for _, expected := range expectedElements {
		if !strings.Contains(output, expected) {
			t.Errorf("Output should contain '%s', got:\n%s", expected, output)
		}
	}
}

func TestDiagnosticReporter_ReportSuccess(t *testing.T) {
	// Capture stdout output
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	reporter := NewDiagnosticReporter(false)

	// Create a test summary
	summary := GenerationSummary{
		PackagesProcessed: 3,
		ModulesGenerated:  2,
		ParsersDiscovered: 5,
		ControllersFound:  4,
		ServicesFound:     6,
		MiddlewaresFound:  2,
		GeneratedFiles: []string{
			"internal/controllers/autogen_module.go",
			"internal/services/autogen_module.go",
		},
	}

	// Report success
	reporter.ReportSuccess(summary)

	// Close writer and read output
	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	// Verify output contains expected elements
	expectedElements := []string{
		"Code Generation Completed Successfully!",
		"Processed 3 packages",
		"Generated 2 FX modules",
		"Discovered 5 custom parsers",
		"Found 4 controllers",
		"Found 6 services",
		"Found 2 middlewares",
		"Generated files:",
		"internal/controllers/autogen_module.go",
		"internal/services/autogen_module.go",
		"Your Axon application is ready to use!",
	}

	for _, expected := range expectedElements {
		if !strings.Contains(output, expected) {
			t.Errorf("Output should contain '%s', got:\n%s", expected, output)
		}
	}
}

func TestDiagnosticReporter_FormatContextKey(t *testing.T) {
	reporter := NewDiagnosticReporter(false)

	tests := []struct {
		input    string
		expected string
	}{
		{"function_name", "Function"},
		{"type_name", "Type"},
		{"route_method", "Route Method"},
		{"route_path", "Route Path"},
		{"parameter_name", "Parameter"},
		{"custom_key", "Custom Key"},
		{"another_test_key", "Another Test Key"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := reporter.formatContextKey(tt.input)
			if result != tt.expected {
				t.Errorf("formatContextKey(%s) = %s, expected %s", tt.input, result, tt.expected)
			}
		})
	}
}
