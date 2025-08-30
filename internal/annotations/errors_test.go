package annotations

import (
	"strings"
	"testing"
)

// ErrorCollector provides error recovery and multiple error collection (test-only utility)
type ErrorCollector struct {
	errors    []AnnotationError
	maxErrors int
}

// NewErrorCollector creates a new error collector with optional max error limit
func NewErrorCollector(maxErrors ...int) *ErrorCollector {
	max := 10 // default max errors
	if len(maxErrors) > 0 && maxErrors[0] > 0 {
		max = maxErrors[0]
	}
	return &ErrorCollector{
		errors:    make([]AnnotationError, 0),
		maxErrors: max,
	}
}

// Add adds an error to the collection if under the limit
func (ec *ErrorCollector) Add(err AnnotationError) {
	if len(ec.errors) < ec.maxErrors {
		ec.errors = append(ec.errors, err)
	}
}

// AddSyntaxError creates and adds a syntax error
func (ec *ErrorCollector) AddSyntaxError(msg string, loc SourceLocation, suggestion string) {
	ec.Add(&SyntaxError{
		Msg:  msg,
		Loc:  loc,
		Hint: suggestion,
	})
}

// AddValidationError creates and adds a validation error
func (ec *ErrorCollector) AddValidationError(parameter, expected, actual string, loc SourceLocation, suggestion string) {
	ec.Add(&ValidationError{
		Parameter: parameter,
		Expected:  expected,
		Actual:    actual,
		Loc:       loc,
		Hint:      suggestion,
	})
}

// AddSchemaError creates and adds a schema error
func (ec *ErrorCollector) AddSchemaError(msg string, loc SourceLocation, suggestion string) {
	ec.Add(&SchemaError{
		Msg:  msg,
		Loc:  loc,
		Hint: suggestion,
	})
}

// HasErrors returns true if any errors have been collected
func (ec *ErrorCollector) HasErrors() bool {
	return len(ec.errors) > 0
}

// Count returns the number of errors collected
func (ec *ErrorCollector) Count() int {
	return len(ec.errors)
}

// Errors returns all collected errors
func (ec *ErrorCollector) Errors() []AnnotationError {
	return ec.errors
}

// ToError converts collected errors to a single error, or nil if no errors
func (ec *ErrorCollector) ToError() error {
	if len(ec.errors) == 0 {
		return nil
	}
	if len(ec.errors) == 1 {
		return ec.errors[0]
	}
	return &MultipleAnnotationErrors{Errors: ec.errors}
}

func TestAnnotationError_Interface(t *testing.T) {
	loc := SourceLocation{File: "test.go", Line: 10, Column: 5}

	tests := []struct {
		name     string
		err      AnnotationError
		wantCode ErrorCode
		wantLoc  SourceLocation
	}{
		{
			name: "SyntaxError implements AnnotationError",
			err: &SyntaxError{
				Msg:  "invalid syntax",
				Loc:  loc,
				Hint: "check syntax",
			},
			wantCode: SyntaxErrorCode,
			wantLoc:  loc,
		},
		{
			name: "ValidationError implements AnnotationError",
			err: &ValidationError{
				Parameter: "Mode",
				Expected:  "Singleton or Transient",
				Actual:    "Invalid",
				Loc:       loc,
				Hint:      "use valid mode",
			},
			wantCode: ValidationErrorCode,
			wantLoc:  loc,
		},
		{
			name: "SchemaError implements AnnotationError",
			err: &SchemaError{
				Msg:  "schema not found",
				Loc:  loc,
				Hint: "register schema",
			},
			wantCode: SchemaErrorCode,
			wantLoc:  loc,
		},
		{
			name: "RegistrationError implements AnnotationError",
			err: &RegistrationError{
				Msg:  "duplicate registration",
				Loc:  loc,
				Hint: "use different name",
			},
			wantCode: RegistrationErrorCode,
			wantLoc:  loc,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.err.Code(); got != tt.wantCode {
				t.Errorf("Code() = %v, want %v", got, tt.wantCode)
			}
			if got := tt.err.Location(); got != tt.wantLoc {
				t.Errorf("Location() = %v, want %v", got, tt.wantLoc)
			}
			if got := tt.err.Suggestion(); got == "" {
				t.Error("Suggestion() should not be empty")
			}
			if got := tt.err.Error(); got == "" {
				t.Error("Error() should not be empty")
			}
		})
	}
}

func TestErrorCollector(t *testing.T) {
	t.Run("NewErrorCollector with default max", func(t *testing.T) {
		ec := NewErrorCollector()
		if ec.maxErrors != 10 {
			t.Errorf("Expected default maxErrors to be 10, got %d", ec.maxErrors)
		}
		if ec.HasErrors() {
			t.Error("New collector should not have errors")
		}
		if ec.Count() != 0 {
			t.Errorf("Expected count 0, got %d", ec.Count())
		}
	})

	t.Run("NewErrorCollector with custom max", func(t *testing.T) {
		ec := NewErrorCollector(5)
		if ec.maxErrors != 5 {
			t.Errorf("Expected maxErrors to be 5, got %d", ec.maxErrors)
		}
	})

	t.Run("Add errors and check collection", func(t *testing.T) {
		ec := NewErrorCollector(3)
		loc := SourceLocation{File: "test.go", Line: 1, Column: 1}

		// Add first error
		ec.AddSyntaxError("syntax error 1", loc, "fix 1")
		if !ec.HasErrors() {
			t.Error("Should have errors after adding one")
		}
		if ec.Count() != 1 {
			t.Errorf("Expected count 1, got %d", ec.Count())
		}

		// Add second error
		ec.AddValidationError("param", "expected", "actual", loc, "fix 2")
		if ec.Count() != 2 {
			t.Errorf("Expected count 2, got %d", ec.Count())
		}

		// Add third error
		ec.AddSchemaError("schema error", loc, "fix 3")
		if ec.Count() != 3 {
			t.Errorf("Expected count 3, got %d", ec.Count())
		}

		// Try to add fourth error (should be ignored due to max limit)
		ec.AddSyntaxError("syntax error 4", loc, "fix 4")
		if ec.Count() != 3 {
			t.Errorf("Expected count to remain 3 due to max limit, got %d", ec.Count())
		}

		// Check error types
		errors := ec.Errors()
		if len(errors) != 3 {
			t.Errorf("Expected 3 errors, got %d", len(errors))
		}

		// Verify error types
		if errors[0].Code() != SyntaxErrorCode {
			t.Errorf("Expected first error to be SyntaxError, got %v", errors[0].Code())
		}
		if errors[1].Code() != ValidationErrorCode {
			t.Errorf("Expected second error to be ValidationError, got %v", errors[1].Code())
		}
		if errors[2].Code() != SchemaErrorCode {
			t.Errorf("Expected third error to be SchemaError, got %v", errors[2].Code())
		}
	})

	t.Run("ToError conversion", func(t *testing.T) {
		// Empty collector
		ec := NewErrorCollector()
		if err := ec.ToError(); err != nil {
			t.Errorf("Expected nil error for empty collector, got %v", err)
		}

		// Single error
		loc := SourceLocation{File: "test.go", Line: 1, Column: 1}
		ec.AddSyntaxError("single error", loc, "fix")
		err := ec.ToError()
		if err == nil {
			t.Error("Expected error for non-empty collector")
		}
		if _, ok := err.(*SyntaxError); !ok {
			t.Errorf("Expected SyntaxError, got %T", err)
		}

		// Multiple errors
		ec.AddValidationError("param", "expected", "actual", loc, "fix")
		err = ec.ToError()
		if err == nil {
			t.Error("Expected error for collector with multiple errors")
		}
		if _, ok := err.(*MultipleAnnotationErrors); !ok {
			t.Errorf("Expected MultipleAnnotationErrors, got %T", err)
		}
	})
}

func TestMultipleAnnotationErrors(t *testing.T) {
	loc := SourceLocation{File: "test.go", Line: 1, Column: 1}
	
	syntaxErr := &SyntaxError{Msg: "syntax error", Loc: loc, Hint: "fix syntax"}
	validationErr := &ValidationError{Parameter: "Mode", Expected: "valid", Actual: "invalid", Loc: loc, Hint: "fix validation"}
	schemaErr := &SchemaError{Msg: "schema error", Loc: loc, Hint: "fix schema"}

	t.Run("Empty errors", func(t *testing.T) {
		mae := &MultipleAnnotationErrors{Errors: []AnnotationError{}}
		if mae.Error() != "no errors" {
			t.Errorf("Expected 'no errors', got %s", mae.Error())
		}
	})

	t.Run("Single error", func(t *testing.T) {
		mae := &MultipleAnnotationErrors{Errors: []AnnotationError{syntaxErr}}
		expected := syntaxErr.Error()
		if mae.Error() != expected {
			t.Errorf("Expected %s, got %s", expected, mae.Error())
		}
	})

	t.Run("Multiple errors", func(t *testing.T) {
		mae := &MultipleAnnotationErrors{Errors: []AnnotationError{syntaxErr, validationErr, schemaErr}}
		errorMsg := mae.Error()
		
		if !strings.Contains(errorMsg, "multiple annotation errors (3 total)") {
			t.Errorf("Error message should contain count, got: %s", errorMsg)
		}
		if !strings.Contains(errorMsg, "syntax error") {
			t.Errorf("Error message should contain syntax error, got: %s", errorMsg)
		}
		if !strings.Contains(errorMsg, "validation failed") {
			t.Errorf("Error message should contain validation error, got: %s", errorMsg)
		}
		if !strings.Contains(errorMsg, "schema error") {
			t.Errorf("Error message should contain schema error, got: %s", errorMsg)
		}
	})

	t.Run("GetByType", func(t *testing.T) {
		mae := &MultipleAnnotationErrors{Errors: []AnnotationError{syntaxErr, validationErr, schemaErr}}
		
		syntaxErrors := mae.GetByType(SyntaxErrorCode)
		if len(syntaxErrors) != 1 {
			t.Errorf("Expected 1 syntax error, got %d", len(syntaxErrors))
		}
		
		validationErrors := mae.GetByType(ValidationErrorCode)
		if len(validationErrors) != 1 {
			t.Errorf("Expected 1 validation error, got %d", len(validationErrors))
		}
		
		registrationErrors := mae.GetByType(RegistrationErrorCode)
		if len(registrationErrors) != 0 {
			t.Errorf("Expected 0 registration errors, got %d", len(registrationErrors))
		}
	})

	t.Run("HasType", func(t *testing.T) {
		mae := &MultipleAnnotationErrors{Errors: []AnnotationError{syntaxErr, validationErr}}
		
		if !mae.HasType(SyntaxErrorCode) {
			t.Error("Should have syntax error")
		}
		if !mae.HasType(ValidationErrorCode) {
			t.Error("Should have validation error")
		}
		if mae.HasType(SchemaErrorCode) {
			t.Error("Should not have schema error")
		}
	})

	t.Run("Unwrap", func(t *testing.T) {
		mae := &MultipleAnnotationErrors{Errors: []AnnotationError{syntaxErr, validationErr}}
		unwrapped := mae.Unwrap()
		
		if len(unwrapped) != 2 {
			t.Errorf("Expected 2 unwrapped errors, got %d", len(unwrapped))
		}
	})

	t.Run("Is", func(t *testing.T) {
		mae := &MultipleAnnotationErrors{Errors: []AnnotationError{syntaxErr, validationErr}}
		
		if !mae.Is(syntaxErr) {
			t.Error("Should find syntax error")
		}
		if !mae.Is(validationErr) {
			t.Error("Should find validation error")
		}
		if mae.Is(schemaErr) {
			t.Error("Should not find schema error")
		}
	})
}

func TestContextAwareErrorGeneration(t *testing.T) {
	loc := SourceLocation{File: "test.go", Line: 10, Column: 5}

	t.Run("NewSyntaxErrorWithContext", func(t *testing.T) {
		err := NewSyntaxErrorWithContext("missing annotation type", loc, "route context")
		if err.Code() != SyntaxErrorCode {
			t.Errorf("Expected SyntaxErrorCode, got %v", err.Code())
		}
		if !strings.Contains(err.Suggestion(), "axon::core") || !strings.Contains(err.Suggestion(), "axon::route") {
			t.Errorf("Expected suggestion to contain annotation examples, got: %s", err.Suggestion())
		}
	})

	t.Run("NewValidationErrorWithContext for core annotation", func(t *testing.T) {
		err := NewValidationErrorWithContext("Mode", "Singleton or Transient", "Invalid", loc, CoreAnnotation)
		if err.Code() != ValidationErrorCode {
			t.Errorf("Expected ValidationErrorCode, got %v", err.Code())
		}
		if !strings.Contains(err.Suggestion(), "Singleton") || !strings.Contains(err.Suggestion(), "Transient") {
			t.Errorf("Expected suggestion to contain valid modes, got: %s", err.Suggestion())
		}
	})

	t.Run("NewValidationErrorWithContext for route annotation", func(t *testing.T) {
		err := NewValidationErrorWithContext("method", "valid HTTP method", "INVALID", loc, RouteAnnotation)
		if !strings.Contains(err.Suggestion(), "GET") || !strings.Contains(err.Suggestion(), "POST") {
			t.Errorf("Expected suggestion to contain HTTP methods, got: %s", err.Suggestion())
		}
	})

	t.Run("NewSchemaErrorWithContext", func(t *testing.T) {
		err := NewSchemaErrorWithContext("unknown annotation type", loc, CoreAnnotation)
		if err.Code() != SchemaErrorCode {
			t.Errorf("Expected SchemaErrorCode, got %v", err.Code())
		}
		if !strings.Contains(err.Suggestion(), "core") {
			t.Errorf("Expected suggestion to contain supported types, got: %s", err.Suggestion())
		}
	})
}

func TestSyntaxSuggestionGeneration(t *testing.T) {
	tests := []struct {
		name     string
		msg      string
		context  string
		expected []string // strings that should be in the suggestion
	}{
		{
			name:     "missing annotation type",
			msg:      "missing annotation type",
			context:  "",
			expected: []string{"axon::core", "axon::route"},
		},
		{
			name:     "invalid annotation prefix",
			msg:      "invalid annotation prefix",
			context:  "",
			expected: []string{"//axon::", "double colon"},
		},
		{
			name:     "unterminated quoted string",
			msg:      "unterminated quoted string",
			context:  "",
			expected: []string{"quoted", "closed"},
		},
		{
			name:     "invalid parameter format",
			msg:      "invalid parameter format",
			context:  "",
			expected: []string{"-ParamName=Value", "-FlagName"},
		},
		{
			name:     "unexpected token in route context",
			msg:      "unexpected token",
			context:  "route",
			expected: []string{"METHOD", "/path", "Middleware"},
		},
		{
			name:     "unexpected token in core context",
			msg:      "unexpected token",
			context:  "core",
			expected: []string{"Mode", "Singleton", "Transient"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			suggestion := generateSyntaxSuggestion(tt.msg, tt.context)
			for _, expected := range tt.expected {
				if !strings.Contains(suggestion, expected) {
					t.Errorf("Expected suggestion to contain '%s', got: %s", expected, suggestion)
				}
			}
		})
	}
}

func TestValidationSuggestionGeneration(t *testing.T) {
	tests := []struct {
		name           string
		parameter      string
		expected       string
		actual         string
		annotationType AnnotationType
		expectedInSuggestion []string
	}{
		{
			name:           "core Mode parameter",
			parameter:      "Mode",
			expected:       "Singleton or Transient",
			actual:         "Invalid",
			annotationType: CoreAnnotation,
			expectedInSuggestion: []string{"Singleton", "Transient", "-Mode=Transient"},
		},
		{
			name:           "core Init parameter",
			parameter:      "Init",
			expected:       "Same or Background",
			actual:         "Invalid",
			annotationType: CoreAnnotation,
			expectedInSuggestion: []string{"Same", "Background", "-Init=Background"},
		},
		{
			name:           "route method parameter",
			parameter:      "method",
			expected:       "valid HTTP method",
			actual:         "INVALID",
			annotationType: RouteAnnotation,
			expectedInSuggestion: []string{"GET", "POST", "PUT", "DELETE"},
		},
		{
			name:           "route Middleware parameter",
			parameter:      "Middleware",
			expected:       "comma-separated names",
			actual:         "Invalid",
			annotationType: RouteAnnotation,
			expectedInSuggestion: []string{"comma-separated", "Auth,Logging"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			suggestion := generateValidationSuggestion(tt.parameter, tt.expected, tt.actual, tt.annotationType)
			for _, expected := range tt.expectedInSuggestion {
				if !strings.Contains(suggestion, expected) {
					t.Errorf("Expected suggestion to contain '%s', got: %s", expected, suggestion)
				}
			}
		})
	}
}

func TestErrorSummary(t *testing.T) {
	loc := SourceLocation{File: "test.go", Line: 1, Column: 1}
	
	syntaxErr := &SyntaxError{Msg: "syntax error", Loc: loc, Hint: "fix"}
	validationErr := &ValidationError{Parameter: "Mode", Expected: "valid", Actual: "invalid", Loc: loc, Hint: "fix"}
	schemaErr := &SchemaError{Msg: "schema error", Loc: loc, Hint: "fix"}
	registrationErr := &RegistrationError{Msg: "registration error", Loc: loc, Hint: "fix"}

	t.Run("Empty summary", func(t *testing.T) {
		summary := SummarizeErrors([]AnnotationError{})
		if summary.TotalCount != 0 {
			t.Errorf("Expected total count 0, got %d", summary.TotalCount)
		}
		if summary.String() != "No errors found" {
			t.Errorf("Expected 'No errors found', got %s", summary.String())
		}
	})

	t.Run("Mixed errors summary", func(t *testing.T) {
		errors := []AnnotationError{syntaxErr, validationErr, schemaErr, registrationErr}
		summary := SummarizeErrors(errors)
		
		if summary.TotalCount != 4 {
			t.Errorf("Expected total count 4, got %d", summary.TotalCount)
		}
		if len(summary.SyntaxErrors) != 1 {
			t.Errorf("Expected 1 syntax error, got %d", len(summary.SyntaxErrors))
		}
		if len(summary.ValidationErrors) != 1 {
			t.Errorf("Expected 1 validation error, got %d", len(summary.ValidationErrors))
		}
		if len(summary.SchemaErrors) != 1 {
			t.Errorf("Expected 1 schema error, got %d", len(summary.SchemaErrors))
		}
		if len(summary.OtherErrors) != 1 {
			t.Errorf("Expected 1 other error, got %d", len(summary.OtherErrors))
		}

		summaryStr := summary.String()
		if !strings.Contains(summaryStr, "4 total error(s)") {
			t.Errorf("Expected summary to contain total count, got: %s", summaryStr)
		}
		if !strings.Contains(summaryStr, "1 syntax error(s)") {
			t.Errorf("Expected summary to contain syntax error count, got: %s", summaryStr)
		}
		if !strings.Contains(summaryStr, "1 validation error(s)") {
			t.Errorf("Expected summary to contain validation error count, got: %s", summaryStr)
		}
	})
}

func TestErrorCode_String(t *testing.T) {
	tests := []struct {
		code     ErrorCode
		expected string
	}{
		{SyntaxErrorCode, "SyntaxError"},
		{ValidationErrorCode, "ValidationError"},
		{SchemaErrorCode, "SchemaError"},
		{RegistrationErrorCode, "RegistrationError"},
		{ErrorCode(999), "UnknownError"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if got := tt.code.String(); got != tt.expected {
				t.Errorf("ErrorCode.String() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestErrorFormatting(t *testing.T) {
	loc := SourceLocation{File: "test.go", Line: 10, Column: 5}

	t.Run("SyntaxError formatting", func(t *testing.T) {
		err := &SyntaxError{
			Msg:  "invalid syntax",
			Loc:  loc,
			Hint: "check your syntax",
		}
		expected := "test.go:10:5: syntax error: invalid syntax. check your syntax"
		if got := err.Error(); got != expected {
			t.Errorf("Expected %s, got %s", expected, got)
		}
	})

	t.Run("ValidationError formatting", func(t *testing.T) {
		err := &ValidationError{
			Parameter: "Mode",
			Expected:  "Singleton or Transient",
			Actual:    "Invalid",
			Loc:       loc,
			Hint:      "use valid mode",
		}
		expected := "test.go:10:5: parameter 'Mode' validation failed: expected Singleton or Transient, got Invalid. use valid mode"
		if got := err.Error(); got != expected {
			t.Errorf("Expected %s, got %s", expected, got)
		}
	})

	t.Run("RegistrationError without location", func(t *testing.T) {
		err := &RegistrationError{
			Msg:  "duplicate registration",
			Hint: "use different name",
		}
		expected := "registration error: duplicate registration. use different name"
		if got := err.Error(); got != expected {
			t.Errorf("Expected %s, got %s", expected, got)
		}
	})
}