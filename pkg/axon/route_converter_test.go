package axon

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRouteConverter_AxonToEcho(t *testing.T) {
	converter := NewRouteConverter()

	testCases := []struct {
		name     string
		axonPath string
		expected string
	}{
		{
			name:     "simple int parameter",
			axonPath: "/users/{id:int}",
			expected: "/users/:id",
		},
		{
			name:     "simple string parameter",
			axonPath: "/users/{name:string}",
			expected: "/users/:name",
		},
		{
			name:     "multiple parameters",
			axonPath: "/users/{id:int}/posts/{slug:string}",
			expected: "/users/:id/posts/:slug",
		},
		{
			name:     "no parameters",
			axonPath: "/users",
			expected: "/users",
		},
		{
			name:     "mixed parameters",
			axonPath: "/api/v1/users/{userId:int}/posts/{postId:int}/comments/{commentId:string}",
			expected: "/api/v1/users/:userId/posts/:postId/comments/:commentId",
		},
		{
			name:     "parameter at end",
			axonPath: "/search/{query:string}",
			expected: "/search/:query",
		},
		{
			name:     "parameter at start",
			axonPath: "/{category:string}/items",
			expected: "/:category/items",
		},
		{
			name:     "untyped parameter",
			axonPath: "/{id}/fish",
			expected: "/:id/fish",
		},
		{
			name:     "mixed typed and untyped parameters",
			axonPath: "/users/{id}/posts/{slug:string}",
			expected: "/users/:id/posts/:slug",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := converter.AxonToEcho(tc.axonPath)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestRouteConverter_EchoToAxon(t *testing.T) {
	converter := NewRouteConverter()

	testCases := []struct {
		name       string
		echoPath   string
		paramTypes map[string]string
		expected   string
	}{
		{
			name:       "simple parameter with type info",
			echoPath:   "/users/:id",
			paramTypes: map[string]string{"id": "int"},
			expected:   "/users/{id:int}",
		},
		{
			name:       "simple parameter without type info (defaults to string)",
			echoPath:   "/users/:name",
			paramTypes: nil,
			expected:   "/users/{name:string}",
		},
		{
			name:     "multiple parameters with mixed types",
			echoPath: "/users/:id/posts/:slug",
			paramTypes: map[string]string{
				"id":   "int",
				"slug": "string",
			},
			expected: "/users/{id:int}/posts/{slug:string}",
		},
		{
			name:       "no parameters",
			echoPath:   "/users",
			paramTypes: nil,
			expected:   "/users",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := converter.EchoToAxon(tc.echoPath, tc.paramTypes)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestRouteConverter_ExtractParameterInfo(t *testing.T) {
	converter := NewRouteConverter()

	testCases := []struct {
		name     string
		axonPath string
		expected map[string]string
	}{
		{
			name:     "single int parameter",
			axonPath: "/users/{id:int}",
			expected: map[string]string{"id": "int"},
		},
		{
			name:     "single string parameter",
			axonPath: "/users/{name:string}",
			expected: map[string]string{"name": "string"},
		},
		{
			name:     "multiple parameters",
			axonPath: "/users/{id:int}/posts/{slug:string}",
			expected: map[string]string{"id": "int", "slug": "string"},
		},
		{
			name:     "no parameters",
			axonPath: "/users",
			expected: map[string]string{},
		},
		{
			name:     "complex path with multiple parameters",
			axonPath: "/api/v1/users/{userId:int}/posts/{postId:int}/comments/{commentId:string}",
			expected: map[string]string{
				"userId":    "int",
				"postId":    "int",
				"commentId": "string",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := converter.ExtractParameterInfo(tc.axonPath)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestRouteConverter_ValidateAxonPath(t *testing.T) {
	converter := NewRouteConverter()

	testCases := []struct {
		name        string
		axonPath    string
		expectError bool
	}{
		{
			name:        "valid path with parameters",
			axonPath:    "/users/{id:int}/posts/{slug:string}",
			expectError: false,
		},
		{
			name:        "valid path without parameters",
			axonPath:    "/users",
			expectError: false,
		},
		{
			name:        "invalid path - missing closing brace",
			axonPath:    "/users/{id:int",
			expectError: true,
		},
		{
			name:        "invalid path - missing opening brace",
			axonPath:    "/users/id:int}",
			expectError: true,
		},
		{
			name:        "invalid path - missing type",
			axonPath:    "/users/{id}",
			expectError: true,
		},
		{
			name:        "invalid path - missing colon",
			axonPath:    "/users/{id int}",
			expectError: true,
		},
		{
			name:        "invalid path - empty parameter",
			axonPath:    "/users/{}",
			expectError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := converter.ValidateAxonPath(tc.axonPath)
			if tc.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestRouteConverterConvenienceFunctions(t *testing.T) {
	// Test global convenience functions
	axonPath := "/users/{id:int}/posts/{slug:string}"
	echoPath := AxonToEcho(axonPath)
	assert.Equal(t, "/users/:id/posts/:slug", echoPath)

	paramInfo := ExtractParameterInfo(axonPath)
	expected := map[string]string{"id": "int", "slug": "string"}
	assert.Equal(t, expected, paramInfo)

	// Test reverse conversion
	convertedBack := EchoToAxon(echoPath, paramInfo)
	assert.Equal(t, axonPath, convertedBack)
}
