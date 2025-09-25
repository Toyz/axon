package axon

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseInt(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expected    int
		expectError bool
	}{
		{
			name:     "valid positive integer",
			input:    "123",
			expected: 123,
		},
		{
			name:     "valid negative integer",
			input:    "-456",
			expected: -456,
		},
		{
			name:     "zero",
			input:    "0",
			expected: 0,
		},
		{
			name:     "large integer",
			input:    "2147483647",
			expected: 2147483647,
		},
		{
			name:        "invalid integer - letters",
			input:       "abc",
			expectError: true,
		},
		{
			name:        "invalid integer - float",
			input:       "123.45",
			expectError: true,
		},
		{
			name:        "invalid integer - empty string",
			input:       "",
			expectError: true,
		},
		{
			name:        "invalid integer - mixed",
			input:       "123abc",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseInt(nil, tt.input)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestParseString(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "regular string",
			input:    "hello",
			expected: "hello",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "string with spaces",
			input:    "hello world",
			expected: "hello world",
		},
		{
			name:     "string with special characters",
			input:    "hello@world.com",
			expected: "hello@world.com",
		},
		{
			name:     "numeric string",
			input:    "12345",
			expected: "12345",
		},
		{
			name:     "unicode string",
			input:    "こんにちは",
			expected: "こんにちは",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseString(nil, tt.input)
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestParseFloat64(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expected    float64
		expectError bool
	}{
		{
			name:     "positive float",
			input:    "123.45",
			expected: 123.45,
		},
		{
			name:     "negative float",
			input:    "-456.78",
			expected: -456.78,
		},
		{
			name:     "zero",
			input:    "0.0",
			expected: 0.0,
		},
		{
			name:     "integer as float",
			input:    "42",
			expected: 42.0,
		},
		{
			name:     "scientific notation",
			input:    "1.23e10",
			expected: 1.23e10,
		},
		{
			name:     "very small number",
			input:    "0.000001",
			expected: 0.000001,
		},
		{
			name:        "invalid float - letters",
			input:       "abc",
			expectError: true,
		},
		{
			name:        "invalid float - empty string",
			input:       "",
			expectError: true,
		},
		{
			name:        "invalid float - mixed",
			input:       "123.45abc",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseFloat64(nil, tt.input)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestParseFloat32(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expected    float32
		expectError bool
	}{
		{
			name:     "positive float",
			input:    "123.45",
			expected: 123.45,
		},
		{
			name:     "negative float",
			input:    "-456.78",
			expected: -456.78,
		},
		{
			name:     "zero",
			input:    "0.0",
			expected: 0.0,
		},
		{
			name:     "integer as float",
			input:    "42",
			expected: 42.0,
		},
		{
			name:        "invalid float - letters",
			input:       "abc",
			expectError: true,
		},
		{
			name:        "invalid float - empty string",
			input:       "",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseFloat32(nil, tt.input)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestParseUUID(t *testing.T) {
	validUUID := "123e4567-e89b-12d3-a456-426614174000"
	expectedUUID := uuid.MustParse(validUUID)

	tests := []struct {
		name        string
		input       string
		expected    uuid.UUID
		expectError bool
	}{
		{
			name:     "valid UUID v4",
			input:    validUUID,
			expected: expectedUUID,
		},
		{
			name:     "valid UUID v1",
			input:    "550e8400-e29b-11d4-a716-446655440000",
			expected: uuid.MustParse("550e8400-e29b-11d4-a716-446655440000"),
		},
		{
			name:     "nil UUID",
			input:    "00000000-0000-0000-0000-000000000000",
			expected: uuid.Nil,
		},
		{
			name:        "invalid UUID - too short",
			input:       "123e4567-e89b-12d3-a456",
			expectError: true,
		},
		{
			name:        "invalid UUID - wrong format",
			input:       "123e4567-e89b-12d3-a456-42661417400",
			expectError: true,
		},
		{
			name:        "invalid UUID - invalid characters",
			input:       "123e4567-e89b-12d3-a456-42661417400g",
			expectError: true,
		},
		{
			name:        "invalid UUID - empty string",
			input:       "",
			expectError: true,
		},
		{
			name:        "invalid UUID - random string",
			input:       "not-a-uuid",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseUUID(nil, tt.input)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestBuiltinParsersMetadata(t *testing.T) {
	expectedParsers := []string{"int", "string", "float64", "float32", "uuid.UUID"}

	for _, typeName := range expectedParsers {
		t.Run("metadata_for_"+typeName, func(t *testing.T) {
			parser, exists := BuiltinParsers[typeName]
			require.True(t, exists, "Parser for type %s should exist", typeName)

			assert.Equal(t, typeName, parser.TypeName)
			assert.NotEmpty(t, parser.FunctionName)
			assert.Equal(t, "builtin", parser.PackagePath)
			assert.Equal(t, []string{"RequestContext", "string"}, parser.ParameterTypes)
			assert.Len(t, parser.ReturnTypes, 2)
			assert.Equal(t, "error", parser.ReturnTypes[1])
		})
	}
}

func TestGetBuiltinParser(t *testing.T) {
	tests := []struct {
		name     string
		typeName string
		exists   bool
		expected string // expected resolved type name
	}{
		{
			name:     "int parser exists",
			typeName: "int",
			exists:   true,
			expected: "int",
		},
		{
			name:     "string parser exists",
			typeName: "string",
			exists:   true,
			expected: "string",
		},
		{
			name:     "uuid parser exists",
			typeName: "uuid.UUID",
			exists:   true,
			expected: "uuid.UUID",
		},
		{
			name:     "UUID alias resolves to uuid.UUID",
			typeName: "UUID",
			exists:   true,
			expected: "uuid.UUID",
		},
		{
			name:     "float alias resolves to float64",
			typeName: "float",
			exists:   true,
			expected: "float64",
		},
		{
			name:     "double alias resolves to float64",
			typeName: "double",
			exists:   true,
			expected: "float64",
		},
		{
			name:     "non-existent parser",
			typeName: "bool",
			exists:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser, exists := GetBuiltinParser(tt.typeName)
			assert.Equal(t, tt.exists, exists)
			if exists {
				assert.Equal(t, tt.expected, parser.TypeName)
			}
		})
	}
}

func TestIsBuiltinType(t *testing.T) {
	tests := []struct {
		name     string
		typeName string
		expected bool
	}{
		{
			name:     "int is builtin",
			typeName: "int",
			expected: true,
		},
		{
			name:     "string is builtin",
			typeName: "string",
			expected: true,
		},
		{
			name:     "float64 is builtin",
			typeName: "float64",
			expected: true,
		},
		{
			name:     "float32 is builtin",
			typeName: "float32",
			expected: true,
		},
		{
			name:     "uuid.UUID is builtin",
			typeName: "uuid.UUID",
			expected: true,
		},
		{
			name:     "UUID alias is builtin",
			typeName: "UUID",
			expected: true,
		},
		{
			name:     "float alias is builtin",
			typeName: "float",
			expected: true,
		},
		{
			name:     "double alias is builtin",
			typeName: "double",
			expected: true,
		},
		{
			name:     "bool is not builtin",
			typeName: "bool",
			expected: false,
		},
		{
			name:     "custom type is not builtin",
			typeName: "CustomType",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsBuiltinType(tt.typeName)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestResolveTypeAlias(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "UUID alias resolves",
			input:    "UUID",
			expected: "uuid.UUID",
		},
		{
			name:     "float alias resolves",
			input:    "float",
			expected: "float64",
		},
		{
			name:     "double alias resolves",
			input:    "double",
			expected: "float64",
		},
		{
			name:     "non-alias returns as-is",
			input:    "int",
			expected: "int",
		},
		{
			name:     "non-alias uuid.UUID returns as-is",
			input:    "uuid.UUID",
			expected: "uuid.UUID",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ResolveTypeAlias(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetAllBuiltinTypes(t *testing.T) {
	types := GetAllBuiltinTypes()

	// Should include all actual types
	expectedTypes := []string{"int", "string", "float64", "float32", "uuid.UUID"}
	for _, expectedType := range expectedTypes {
		assert.Contains(t, types, expectedType, "Should contain type %s", expectedType)
	}

	// Should include all aliases
	expectedAliases := []string{"UUID", "float", "double"}
	for _, expectedAlias := range expectedAliases {
		assert.Contains(t, types, expectedAlias, "Should contain alias %s", expectedAlias)
	}

	// Should have the right total count
	expectedCount := len(BuiltinParsers) + len(ParserAliases)
	assert.Len(t, types, expectedCount)
}

// Mock RequestContext for testing parsers
type mockRequestContext struct{}

func (m *mockRequestContext) Method() string                         { return "GET" }
func (m *mockRequestContext) Path() string                           { return "/test" }
func (m *mockRequestContext) RealIP() string                         { return "127.0.0.1" }
func (m *mockRequestContext) Param(key string) string                { return "" }
func (m *mockRequestContext) ParamNames() []string                   { return nil }
func (m *mockRequestContext) ParamValues() []string                  { return nil }
func (m *mockRequestContext) SetParam(name, value string)            {}
func (m *mockRequestContext) QueryParam(key string) string           { return "" }
func (m *mockRequestContext) QueryParams() map[string][]string       { return nil }
func (m *mockRequestContext) QueryString() string                    { return "" }
func (m *mockRequestContext) Request() RequestInterface              { return nil }
func (m *mockRequestContext) Response() ResponseInterface            { return nil }
func (m *mockRequestContext) Bind(i interface{}) error               { return nil }
func (m *mockRequestContext) Validate(i interface{}) error           { return nil }
func (m *mockRequestContext) Get(key string) interface{}             { return nil }
func (m *mockRequestContext) Set(key string, val interface{})        {}
func (m *mockRequestContext) FormValue(name string) string           { return "" }
func (m *mockRequestContext) FormParams() (map[string][]string, error) { return nil, nil }
func (m *mockRequestContext) FormFile(name string) (FileHeader, error) { return nil, nil }
func (m *mockRequestContext) MultipartForm() (MultipartForm, error)  { return nil, nil }

// Test with mock context
func TestBuiltinParsersWithMockContext(t *testing.T) {
	c := &mockRequestContext{}

	t.Run("ParseInt with context", func(t *testing.T) {
		result, err := ParseInt(c, "42")
		assert.NoError(t, err)
		assert.Equal(t, 42, result)
	})

	t.Run("ParseString with context", func(t *testing.T) {
		result, err := ParseString(c, "test-string")
		assert.NoError(t, err)
		assert.Equal(t, "test-string", result)
	})

	t.Run("ParseFloat64 with context", func(t *testing.T) {
		result, err := ParseFloat64(c, "123.45")
		assert.NoError(t, err)
		assert.Equal(t, 123.45, result)
	})

	t.Run("ParseUUID with context", func(t *testing.T) {

		validUUID := "123e4567-e89b-12d3-a456-426614174000"
		result, err := ParseUUID(c, validUUID)
		assert.NoError(t, err)
		assert.Equal(t, uuid.MustParse(validUUID), result)
	})
}
