package parser

import (
	"strings"
	"testing"

	"github.com/toyz/axon/internal/models"
	"github.com/toyz/axon/pkg/axon"
)

func TestParserErrorReporter_ReportParserValidationError(t *testing.T) {
	parser := NewParser()
	reporter := NewParserErrorReporter(parser)

	tests := []struct {
		name                 string
		functionName         string
		fileName             string
		line                 int
		issue                string
		actualSignature      string
		expectedInMessage    []string
		expectedInSuggestions []string
	}{
		{
			name:            "function not found",
			functionName:    "ParseUUID",
			fileName:        "test.go",
			line:            10,
			issue:           "function not found",
			actualSignature: "",
			expectedInMessage: []string{
				"ParseUUID",
				"function not found",
			},
			expectedInSuggestions: []string{
				"Expected signature",
				"func(c echo.Context, paramValue string) (T, error)",
			},
		},
		{
			name:            "wrong parameter count",
			functionName:    "ParseUUID",
			fileName:        "test.go",
			line:            10,
			issue:           "has 1 parameters, expected 2",
			actualSignature: "func ParseUUID(s string) (uuid.UUID, error)",
			expectedInMessage: []string{
				"ParseUUID",
				"parameters",
			},
			expectedInSuggestions: []string{
				"exactly 2 parameters",
			},
		},
		{
			name:            "wrong first parameter",
			functionName:    "ParseUUID",
			fileName:        "test.go",
			line:            10,
			issue:           "first parameter is string, expected echo.Context",
			actualSignature: "func ParseUUID(s string, t string) (uuid.UUID, error)",
			expectedInMessage: []string{
				"ParseUUID",
				"first parameter",
			},
			expectedInSuggestions: []string{
				"echo.Context",
				"Import the Echo framework",
			},
		},
		{
			name:            "wrong return count",
			functionName:    "ParseUUID",
			fileName:        "test.go",
			line:            10,
			issue:           "returns 1 values, expected 2",
			actualSignature: "func ParseUUID(c echo.Context, s string) uuid.UUID",
			expectedInMessage: []string{
				"ParseUUID",
				"return",
			},
			expectedInSuggestions: []string{
				"exactly 2 values",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := reporter.ReportParserValidationError(
				tt.functionName,
				tt.fileName,
				tt.line,
				tt.issue,
				tt.actualSignature,
			)

			if err == nil {
				t.Fatal("Expected error, got nil")
			}

			genErr, ok := err.(*models.GeneratorError)
			if !ok {
				t.Fatalf("Expected GeneratorError, got %T", err)
			}

			if genErr.Type != models.ErrorTypeParserValidation {
				t.Errorf("Expected ErrorTypeParserValidation, got %v", genErr.Type)
			}

			if genErr.File != tt.fileName {
				t.Errorf("Expected file %s, got %s", tt.fileName, genErr.File)
			}

			if genErr.Line != tt.line {
				t.Errorf("Expected line %d, got %d", tt.line, genErr.Line)
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

			// Check context
			if genErr.Context == nil {
				t.Error("Expected context, got nil")
			}

			if genErr.Context["function_name"] != tt.functionName {
				t.Errorf("Expected function_name %s in context, got %v", tt.functionName, genErr.Context["function_name"])
			}
		})
	}
}

func TestParserErrorReporter_ReportParserNotFoundError(t *testing.T) {
	parser := NewParser()
	reporter := NewParserErrorReporter(parser)

	tests := []struct {
		name                 string
		typeName             string
		routeMethod          string
		routePath            string
		paramName            string
		availableParsers     []string
		expectedInMessage    []string
		expectedInSuggestions []string
	}{
		{
			name:             "UUID type not found",
			typeName:         "uuid.UUID",
			routeMethod:      "GET",
			routePath:        "/users/{id:uuid.UUID}",
			paramName:        "id",
			availableParsers: []string{"string", "int"},
			expectedInMessage: []string{
				"uuid.UUID",
				"GET",
				"/users/{id:uuid.UUID}",
				"id",
			},
			expectedInSuggestions: []string{
				"Available parsers: string, int",
				"github.com/google/uuid",
			},
		},
		{
			name:             "Time type not found",
			typeName:         "time.Time",
			routeMethod:      "POST",
			routePath:        "/events/{date:time.Time}",
			paramName:        "date",
			availableParsers: []string{},
			expectedInMessage: []string{
				"time.Time",
				"POST",
				"/events/{date:time.Time}",
				"date",
			},
			expectedInSuggestions: []string{
				"No parsers are currently registered",
				"time.Time",
			},
		},
		{
			name:             "Custom type not found",
			typeName:         "CustomID",
			routeMethod:      "DELETE",
			routePath:        "/items/{id:CustomID}",
			paramName:        "id",
			availableParsers: []string{"uuid.UUID", "time.Time"},
			expectedInMessage: []string{
				"CustomID",
				"DELETE",
				"/items/{id:CustomID}",
				"id",
			},
			expectedInSuggestions: []string{
				"Available parsers: uuid.UUID, time.Time",
				"//axon::route_parser CustomID",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := reporter.ReportParserNotFoundError(
				tt.typeName,
				tt.routeMethod,
				tt.routePath,
				tt.paramName,
				"test.go",
				10,
				tt.availableParsers,
			)

			if err == nil {
				t.Fatal("Expected error, got nil")
			}

			genErr, ok := err.(*models.GeneratorError)
			if !ok {
				t.Fatalf("Expected GeneratorError, got %T", err)
			}

			if genErr.Type != models.ErrorTypeParserValidation {
				t.Errorf("Expected ErrorTypeParserValidation, got %v", genErr.Type)
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

			// Check context
			if genErr.Context == nil {
				t.Error("Expected context, got nil")
			}

			if genErr.Context["type_name"] != tt.typeName {
				t.Errorf("Expected type_name %s in context, got %v", tt.typeName, genErr.Context["type_name"])
			}
		})
	}
}

func TestParserErrorReporter_ReportParserImportError(t *testing.T) {
	parser := NewParser()
	reporter := NewParserErrorReporter(parser)

	tests := []struct {
		name                 string
		typeName             string
		requiredImport       string
		expectedInMessage    []string
		expectedInSuggestions []string
	}{
		{
			name:           "UUID import missing",
			typeName:       "uuid.UUID",
			requiredImport: "github.com/google/uuid",
			expectedInMessage: []string{
				"uuid.UUID",
				"github.com/google/uuid",
			},
			expectedInSuggestions: []string{
				"go get github.com/google/uuid",
				"import \"github.com/google/uuid\"",
			},
		},
		{
			name:           "time import missing",
			typeName:       "time.Time",
			requiredImport: "time",
			expectedInMessage: []string{
				"time.Time",
				"time",
			},
			expectedInSuggestions: []string{
				"import \"time\"",
			},
		},
		{
			name:           "url import missing",
			typeName:       "url.URL",
			requiredImport: "net/url",
			expectedInMessage: []string{
				"url.URL",
				"net/url",
			},
			expectedInSuggestions: []string{
				"import \"net/url\"",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := reporter.ReportParserImportError(
				tt.typeName,
				"test.go",
				10,
				tt.requiredImport,
			)

			if err == nil {
				t.Fatal("Expected error, got nil")
			}

			genErr, ok := err.(*models.GeneratorError)
			if !ok {
				t.Fatalf("Expected GeneratorError, got %T", err)
			}

			if genErr.Type != models.ErrorTypeParserImport {
				t.Errorf("Expected ErrorTypeParserImport, got %v", genErr.Type)
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

			// Check context
			if genErr.Context == nil {
				t.Error("Expected context, got nil")
			}

			if genErr.Context["type_name"] != tt.typeName {
				t.Errorf("Expected type_name %s in context, got %v", tt.typeName, genErr.Context["type_name"])
			}

			if genErr.Context["required_import"] != tt.requiredImport {
				t.Errorf("Expected required_import %s in context, got %v", tt.requiredImport, genErr.Context["required_import"])
			}
		})
	}
}

func TestParserErrorReporter_ReportParserConflictError(t *testing.T) {
	parser := NewParser()
	reporter := NewParserErrorReporter(parser)

	conflicts := []models.ParserConflict{
		{
			FileName:     "parser1.go",
			Line:         10,
			FunctionName: "ParseUUID1",
			PackagePath:  "pkg1",
		},
		{
			FileName:     "parser2.go",
			Line:         20,
			FunctionName: "ParseUUID2",
			PackagePath:  "pkg2",
		},
	}

	err := reporter.ReportParserConflictError("uuid.UUID", conflicts)

	if err == nil {
		t.Fatal("Expected error, got nil")
	}

	genErr, ok := err.(*models.GeneratorError)
	if !ok {
		t.Fatalf("Expected GeneratorError, got %T", err)
	}

	if genErr.Type != models.ErrorTypeParserConflict {
		t.Errorf("Expected ErrorTypeParserConflict, got %v", genErr.Type)
	}

	// Check basic error message content
	errorMessage := err.Error()
	expectedInMessage := []string{
		"uuid.UUID",
		"Multiple parsers",
	}

	for _, expected := range expectedInMessage {
		if !strings.Contains(errorMessage, expected) {
			t.Errorf("Error message should contain '%s', got: %s", expected, errorMessage)
		}
	}

	// Check suggestions contain conflict details
	if len(genErr.Suggestions) == 0 {
		t.Error("Expected suggestions, got none")
	}
	
	allSuggestions := strings.Join(genErr.Suggestions, " ")
	expectedInSuggestions := []string{
		"parser1.go:10",
		"parser2.go:20",
	}
	
	for _, expected := range expectedInSuggestions {
		if !strings.Contains(allSuggestions, expected) {
			t.Errorf("Suggestions should contain '%s', got: %v", expected, genErr.Suggestions)
		}
	}

	// Check context
	if genErr.Context == nil {
		t.Error("Expected context, got nil")
	}

	if genErr.Context["type_name"] != "uuid.UUID" {
		t.Errorf("Expected type_name uuid.UUID in context, got %v", genErr.Context["type_name"])
	}
}

func TestParserErrorReporter_GenerateParserExample(t *testing.T) {
	parser := NewParser()
	reporter := NewParserErrorReporter(parser)

	tests := []struct {
		name         string
		functionName string
		expected     []string
	}{
		{
			name:         "UUID parser",
			functionName: "ParseUUID",
			expected:     []string{"//axon::route_parser uuid.UUID", "uuid.Parse"},
		},
		{
			name:         "Time parser",
			functionName: "ParseTime",
			expected:     []string{"//axon::route_parser time.Time", "time.Parse"},
		},
		{
			name:         "Int parser",
			functionName: "ParseCustomInt",
			expected:     []string{"//axon::route_parser CustomInt", "strconv.Atoi"},
		},
		{
			name:         "Generic parser",
			functionName: "ParseCustomType",
			expected:     []string{"//axon::route_parser YourType", "func ParseCustomType"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			example := reporter.generateParserExample(tt.functionName)

			if example == "" {
				t.Error("Expected example, got empty string")
			}

			for _, expected := range tt.expected {
				if !strings.Contains(example, expected) {
					t.Errorf("Example should contain '%s', got: %s", expected, example)
				}
			}
		})
	}
}

func TestParserErrorReporter_GenerateParserDiagnostics(t *testing.T) {
	parser := NewParser()
	reporter := NewParserErrorReporter(parser)

	tests := []struct {
		name     string
		metadata *models.PackageMetadata
		expected []string
	}{
		{
			name: "no parsers",
			metadata: &models.PackageMetadata{
				RouteParsers: []axon.RouteParserMetadata{},
				Controllers:  []models.ControllerMetadata{},
			},
			expected: []string{"No custom parsers found"},
		},
		{
			name: "unused parser",
			metadata: &models.PackageMetadata{
				RouteParsers: []axon.RouteParserMetadata{
					{TypeName: "UnusedType", FunctionName: "ParseUnused"},
				},
				Controllers: []models.ControllerMetadata{
					{
						Routes: []models.RouteMetadata{
							{
								Method: "GET",
								Path:   "/test",
								Parameters: []models.Parameter{
									{Name: "id", Type: "string", IsCustomType: false},
								},
							},
						},
					},
				},
			},
			expected: []string{"Parser for type 'UnusedType' is defined but not used"},
		},
		{
			name: "missing parser",
			metadata: &models.PackageMetadata{
				RouteParsers: []axon.RouteParserMetadata{},
				Controllers: []models.ControllerMetadata{
					{
						Routes: []models.RouteMetadata{
							{
								Method: "GET",
								Path:   "/test/{id:CustomType}",
								Parameters: []models.Parameter{
									{Name: "id", Type: "CustomType", IsCustomType: true, ParserFunc: ""},
								},
							},
						},
					},
				},
			},
			expected: []string{"uses custom type 'CustomType' but no parser is registered"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			diagnostics := reporter.GenerateParserDiagnostics(tt.metadata)

			for _, expected := range tt.expected {
				found := false
				for _, diagnostic := range diagnostics {
					if strings.Contains(diagnostic, expected) {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected diagnostic containing '%s', got: %v", expected, diagnostics)
				}
			}
		})
	}
}