package utils

import (
	"strings"
	"testing"
)

func TestValidationError(t *testing.T) {
	tests := []struct {
		name     string
		err      ValidationError
		expected string
	}{
		{
			name: "error with field",
			err: ValidationError{
				Field:   "username",
				Value:   "",
				Message: "cannot be empty",
			},
			expected: "validation error for field 'username': cannot be empty",
		},
		{
			name: "error without field",
			err: ValidationError{
				Message: "invalid format",
			},
			expected: "validation error: invalid format",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.err.Error(); got != tt.expected {
				t.Errorf("ValidationError.Error() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestNotEmpty(t *testing.T) {
	validator := NotEmpty("test_field")

	tests := []struct {
		name    string
		value   string
		wantErr bool
	}{
		{"valid string", "hello", false},
		{"empty string", "", true},
		{"whitespace only", "   ", false}, // NotEmpty only checks for empty, not whitespace
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator(tt.value)
			if (err != nil) != tt.wantErr {
				t.Errorf("NotEmpty() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestNotNil(t *testing.T) {
	validator := NotNil[string]("test_field")

	tests := []struct {
		name    string
		value   *string
		wantErr bool
	}{
		{"valid pointer", stringPtr("hello"), false},
		{"nil pointer", nil, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator(tt.value)
			if (err != nil) != tt.wantErr {
				t.Errorf("NotNil() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestHasPrefix(t *testing.T) {
	validator := HasPrefix("path", "/")

	tests := []struct {
		name    string
		value   string
		wantErr bool
	}{
		{"valid path", "/users", false},
		{"invalid path", "users", true},
		{"empty string", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator(tt.value)
			if (err != nil) != tt.wantErr {
				t.Errorf("HasPrefix() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestHasSuffix(t *testing.T) {
	validator := HasSuffix("filename", ".go")

	tests := []struct {
		name    string
		value   string
		wantErr bool
	}{
		{"valid go file", "main.go", false},
		{"invalid file", "main.txt", true},
		{"empty string", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator(tt.value)
			if (err != nil) != tt.wantErr {
				t.Errorf("HasSuffix() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestMatchesRegex(t *testing.T) {
	validator := MatchesRegex("email", `^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)

	tests := []struct {
		name    string
		value   string
		wantErr bool
	}{
		{"valid email", "test@example.com", false},
		{"invalid email", "invalid-email", true},
		{"empty string", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator(tt.value)
			if (err != nil) != tt.wantErr {
				t.Errorf("MatchesRegex() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestIsValidGoIdentifier(t *testing.T) {
	validator := IsValidGoIdentifier("identifier")

	tests := []struct {
		name    string
		value   string
		wantErr bool
	}{
		{"valid identifier", "myVariable", false},
		{"valid with underscore", "my_variable", false},
		{"valid with numbers", "var123", false},
		{"invalid - starts with number", "123var", true},
		{"invalid - has spaces", "my variable", true},
		{"invalid - has special chars", "my-variable", true},
		{"empty string", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator(tt.value)
			if (err != nil) != tt.wantErr {
				t.Errorf("IsValidGoIdentifier() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestIsOneOf(t *testing.T) {
	validator := IsOneOf("method", "GET", "POST", "PUT", "DELETE")

	tests := []struct {
		name    string
		value   string
		wantErr bool
	}{
		{"valid method", "GET", false},
		{"valid method", "POST", false},
		{"invalid method", "PATCH", true},
		{"empty string", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator(tt.value)
			if (err != nil) != tt.wantErr {
				t.Errorf("IsOneOf() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestMinLength(t *testing.T) {
	validator := MinLength("password", 8)

	tests := []struct {
		name    string
		value   string
		wantErr bool
	}{
		{"valid length", "password123", false},
		{"exact minimum", "12345678", false},
		{"too short", "123", true},
		{"empty string", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator(tt.value)
			if (err != nil) != tt.wantErr {
				t.Errorf("MinLength() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestMaxLength(t *testing.T) {
	validator := MaxLength("username", 20)

	tests := []struct {
		name    string
		value   string
		wantErr bool
	}{
		{"valid length", "user123", false},
		{"exact maximum", strings.Repeat("a", 20), false},
		{"too long", strings.Repeat("a", 21), true},
		{"empty string", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator(tt.value)
			if (err != nil) != tt.wantErr {
				t.Errorf("MaxLength() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestSliceNotEmpty(t *testing.T) {
	validator := SliceNotEmpty[string]("items")

	tests := []struct {
		name    string
		value   []string
		wantErr bool
	}{
		{"valid slice", []string{"a", "b"}, false},
		{"single item", []string{"a"}, false},
		{"empty slice", []string{}, true},
		{"nil slice", nil, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator(tt.value)
			if (err != nil) != tt.wantErr {
				t.Errorf("SliceNotEmpty() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateEach(t *testing.T) {
	validator := ValidateEach("routes", NotEmpty("route"))

	tests := []struct {
		name    string
		value   []string
		wantErr bool
	}{
		{"all valid", []string{"/users", "/posts"}, false},
		{"one invalid", []string{"/users", ""}, true},
		{"empty slice", []string{}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator(tt.value)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateEach() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidatorChain(t *testing.T) {
	chain := NewValidatorChain(
		NotEmpty("path"),
		HasPrefix("path", "/"),
		MinLength("path", 2),
	)

	tests := []struct {
		name    string
		value   string
		wantErr bool
	}{
		{"valid path", "/users", false},
		{"empty string", "", true},
		{"no prefix", "users", true},
		{"too short", "/", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := chain.Validate(tt.value)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidatorChain.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestCustom(t *testing.T) {
	validator := Custom("number", "must be even", func(n int) bool {
		return n%2 == 0
	})

	tests := []struct {
		name    string
		value   int
		wantErr bool
	}{
		{"even number", 4, false},
		{"odd number", 3, true},
		{"zero", 0, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator(tt.value)
			if (err != nil) != tt.wantErr {
				t.Errorf("Custom() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestConditional(t *testing.T) {
	// Only validate if string is not empty
	validator := Conditional(
		func(s string) bool { return s != "" },
		HasPrefix("path", "/"),
	)

	tests := []struct {
		name    string
		value   string
		wantErr bool
	}{
		{"valid path", "/users", false},
		{"invalid path", "users", true},
		{"empty string (skipped)", "", false}, // Condition is false, so validation is skipped
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator(tt.value)
			if (err != nil) != tt.wantErr {
				t.Errorf("Conditional() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateHTTPMethod(t *testing.T) {
	validator := ValidateHTTPMethod("method")

	tests := []struct {
		name    string
		value   string
		wantErr bool
	}{
		{"GET", "GET", false},
		{"POST", "POST", false},
		{"PUT", "PUT", false},
		{"DELETE", "DELETE", false},
		{"PATCH", "PATCH", false},
		{"HEAD", "HEAD", false},
		{"OPTIONS", "OPTIONS", false},
		{"invalid method", "INVALID", true},
		{"lowercase", "get", true},
		{"empty", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator(tt.value)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateHTTPMethod() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateURLPath(t *testing.T) {
	validator := ValidateURLPath("path")

	tests := []struct {
		name    string
		value   string
		wantErr bool
	}{
		{"valid path", "/users", false},
		{"root path", "/", false},
		{"nested path", "/api/v1/users", false},
		{"no leading slash", "users", true},
		{"empty", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator(tt.value)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateURLPath() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateConstructorName(t *testing.T) {
	validator := ValidateConstructorName("constructor")

	tests := []struct {
		name    string
		value   string
		wantErr bool
	}{
		{"valid constructor", "NewUserService", false},
		{"valid with underscore", "New_Service", false},
		{"no New prefix", "UserService", true},
		{"invalid identifier", "New-Service", true},
		{"empty", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator(tt.value)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateConstructorName() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// Helper function for tests
func stringPtr(s string) *string {
	return &s
}
