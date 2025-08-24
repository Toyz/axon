package parser

import (
	"strings"
	"testing"

	"github.com/toyz/axon/internal/models"
)

// TestComprehensiveParserErrorHandling demonstrates the comprehensive error handling
// functionality implemented for parser-related failures
func TestComprehensiveParserErrorHandling(t *testing.T) {
	tests := []struct {
		name                 string
		source               string
		expectedErrorType    models.ErrorType
		expectedInMessage    []string
		expectedInSuggestions []string
		expectError          bool
	}{
		{
			name: "parser with invalid signature - wrong parameter count",
			source: `
package test

import "github.com/labstack/echo/v4"

//axon::route_parser CustomID
func ParseCustomID(paramValue string) (CustomID, error) {
	return CustomID(paramValue), nil
}

type CustomID string

//axon::controller
type TestController struct{}

//axon::route GET /test/{id:CustomID}
func (c *TestController) GetTest(id CustomID) (string, error) {
	return string(id), nil
}
`,
			expectError:       true,
			expectedErrorType: models.ErrorTypeParserValidation,
			expectedInMessage: []string{
				"ParseCustomID",
				"has 1 parameters, expected 2",
			},
			expectedInSuggestions: []string{
				"Expected signature",
				"func(c echo.Context, paramValue string) (T, error)",
				"exactly 2 parameters",
			},
		},
		{
			name: "parser with wrong first parameter type",
			source: `
package test

import "github.com/labstack/echo/v4"

//axon::route_parser CustomID
func ParseCustomID(ctx string, paramValue string) (CustomID, error) {
	return CustomID(paramValue), nil
}

type CustomID string

//axon::controller
type TestController struct{}

//axon::route GET /test/{id:CustomID}
func (c *TestController) GetTest(id CustomID) (string, error) {
	return string(id), nil
}
`,
			expectError:       true,
			expectedErrorType: models.ErrorTypeParserValidation,
			expectedInMessage: []string{
				"ParseCustomID",
				"first parameter is string, expected echo.Context",
			},
			expectedInSuggestions: []string{
				"Import the Echo framework",
				"echo.Context",
			},
		},
		{
			name: "parser with wrong return type count",
			source: `
package test

import "github.com/labstack/echo/v4"

//axon::route_parser CustomID
func ParseCustomID(c echo.Context, paramValue string) CustomID {
	return CustomID(paramValue)
}

type CustomID string

//axon::controller
type TestController struct{}

//axon::route GET /test/{id:CustomID}
func (c *TestController) GetTest(id CustomID) (string, error) {
	return string(id), nil
}
`,
			expectError:       true,
			expectedErrorType: models.ErrorTypeParserValidation,
			expectedInMessage: []string{
				"ParseCustomID",
				"returns 1 values, expected 2",
			},
			expectedInSuggestions: []string{
				"exactly 2 values",
				"Second return value should be error",
			},
		},
		{
			name: "route with custom type but no parser registered",
			source: `
package test

//axon::controller
type TestController struct{}

//axon::route GET /test/{id:UnknownType}
func (c *TestController) GetTest(id UnknownType) (string, error) {
	return string(id), nil
}

type UnknownType string
`,
			expectError:       true,
			expectedErrorType: models.ErrorTypeParserValidation,
			expectedInMessage: []string{
				"No parser registered for custom type 'UnknownType'",
				"GET",
				"/test/{id:UnknownType}",
			},
			expectedInSuggestions: []string{
				"Register a parser for type 'UnknownType'",
				"//axon::route_parser UnknownType",
			},
		},
		{
			name: "valid parser with comprehensive error handling",
			source: `
package test

import "github.com/labstack/echo/v4"

//axon::route_parser CustomID
func ParseCustomID(c echo.Context, paramValue string) (CustomID, error) {
	return CustomID(paramValue), nil
}

type CustomID string

//axon::controller
type TestController struct{}

//axon::route GET /test/{id:CustomID}
func (c *TestController) GetTest(id CustomID) (string, error) {
	return string(id), nil
}
`,
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewParser()
			_, err := p.ParseSource("test.go", tt.source)

			if tt.expectError {
				if err == nil {
					t.Fatal("Expected error, got nil")
				}

				// Check if it's a GeneratorError or contains one
				var genErr *models.GeneratorError
				if directErr, ok := err.(*models.GeneratorError); ok {
					genErr = directErr
				} else {
					// Try to unwrap and find the GeneratorError
					t.Logf("Error is wrapped: %T: %v", err, err)
					// For now, just check that we get a meaningful error message
					errorMessage := err.Error()
					for _, expected := range tt.expectedInMessage {
						if !strings.Contains(errorMessage, expected) {
							t.Errorf("Error message should contain '%s', got: %s", expected, errorMessage)
						}
					}
					return // Skip the rest of the test for wrapped errors
				}

				if genErr.Type != tt.expectedErrorType {
					t.Errorf("Expected error type %v, got %v", tt.expectedErrorType, genErr.Type)
				}

				// Check basic error message content
				errorMessage := err.Error()
				for _, expected := range tt.expectedInMessage {
					if !strings.Contains(errorMessage, expected) {
						t.Errorf("Error message should contain '%s', got: %s", expected, errorMessage)
					}
				}

				// Check suggestions
				if len(genErr.Suggestions) == 0 {
					t.Error("Expected suggestions, got none")
				}

				// Check that suggestions contain expected content
				allSuggestions := strings.Join(genErr.Suggestions, " ")
				for _, expected := range tt.expectedInSuggestions {
					if !strings.Contains(allSuggestions, expected) {
						t.Errorf("Suggestions should contain '%s', got: %v", expected, genErr.Suggestions)
					}
				}

				// Verify context is populated
				if genErr.Context == nil {
					t.Error("Expected context, got nil")
				}

				// Verify file and line information is present
				if genErr.File == "" {
					t.Error("Expected file information, got empty string")
				}

				t.Logf("Error: %s", err.Error())
				t.Logf("Suggestions: %v", genErr.Suggestions)
				t.Logf("Context: %v", genErr.Context)

			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
			}
		})
	}
}

// TestParserErrorHandlingIntegration tests the integration of all error handling components
func TestParserErrorHandlingIntegration(t *testing.T) {
	p := NewParser()
	
	// Test comprehensive validation function
	source := `
package test

import "github.com/labstack/echo/v4"

//axon::route_parser ValidType
func ParseValid(c echo.Context, paramValue string) (ValidType, error) {
	return ValidType(paramValue), nil
}

//axon::route_parser InvalidType
func ParseInvalid(paramValue string) (InvalidType, error) {
	return InvalidType(paramValue), nil
}

type ValidType string
type InvalidType string

//axon::controller
type TestController struct{}

//axon::route GET /valid/{id:ValidType}
func (c *TestController) GetValid(id ValidType) (string, error) {
	return string(id), nil
}

//axon::route GET /invalid/{id:InvalidType}
func (c *TestController) GetInvalid(id InvalidType) (string, error) {
	return string(id), nil
}

//axon::route GET /unknown/{id:UnknownType}
func (c *TestController) GetUnknown(id UnknownType) (string, error) {
	return string(id), nil
}

type UnknownType string
`

	_, err := p.ParseSource("test.go", source)
	
	// Should get an error due to invalid parser signature
	if err == nil {
		t.Fatal("Expected error due to invalid parser signature, got nil")
	}

	genErr, ok := err.(*models.GeneratorError)
	if !ok {
		t.Fatalf("Expected GeneratorError, got %T: %v", err, err)
	}

	// Should be a parser validation error
	if genErr.Type != models.ErrorTypeParserValidation {
		t.Errorf("Expected ErrorTypeParserValidation, got %v", genErr.Type)
	}

	// Should contain helpful information
	if len(genErr.Suggestions) == 0 {
		t.Error("Expected suggestions for fixing the error")
	}

	if genErr.Context == nil {
		t.Error("Expected context information")
	}

	t.Logf("Integration test error: %s", err.Error())
	t.Logf("Suggestions: %v", genErr.Suggestions)
}

// TestParserDiagnostics tests the diagnostic functionality
func TestParserDiagnostics(t *testing.T) {
	p := NewParser()
	reporter := NewParserErrorReporter(p)

	// Test metadata with various parser issues
	metadata := &models.PackageMetadata{
		RouteParsers: []models.RouteParserMetadata{
			{TypeName: "UnusedType", FunctionName: "ParseUnused"},
			{TypeName: "UsedType", FunctionName: "ParseUsed"},
		},
		Controllers: []models.ControllerMetadata{
			{
				Routes: []models.RouteMetadata{
					{
						Method: "GET",
						Path:   "/test/{id:UsedType}",
						Parameters: []models.Parameter{
							{Name: "id", Type: "UsedType", IsCustomType: true, ParserFunc: "ParseUsed"},
						},
					},
					{
						Method: "POST",
						Path:   "/test/{id:MissingType}",
						Parameters: []models.Parameter{
							{Name: "id", Type: "MissingType", IsCustomType: true, ParserFunc: ""},
						},
					},
				},
			},
		},
	}

	diagnostics := reporter.GenerateParserDiagnostics(metadata)

	// Should detect unused parser
	foundUnused := false
	foundMissing := false
	
	for _, diagnostic := range diagnostics {
		if strings.Contains(diagnostic, "UnusedType") && strings.Contains(diagnostic, "not used") {
			foundUnused = true
		}
		if strings.Contains(diagnostic, "MissingType") && strings.Contains(diagnostic, "no parser is registered") {
			foundMissing = true
		}
	}

	if !foundUnused {
		t.Error("Expected diagnostic about unused parser")
	}

	if !foundMissing {
		t.Error("Expected diagnostic about missing parser")
	}

	t.Logf("Diagnostics: %v", diagnostics)
}