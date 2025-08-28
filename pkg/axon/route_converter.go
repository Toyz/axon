package axon

import (
	"fmt"
	"regexp"
	"strings"
)

// RouteConverter handles conversion between Axon route syntax and Echo route syntax
type RouteConverter struct{}

// NewRouteConverter creates a new route converter
func NewRouteConverter() *RouteConverter {
	return &RouteConverter{}
}

// AxonToEcho converts Axon route syntax to Echo route syntax
// Converts: /users/{id:int} -> /users/:id
// Converts: /users/{id:string} -> /users/:id
// Converts: /posts/{slug:string}/comments/{id:int} -> /posts/:slug/comments/:id
// Converts: /files/{*} -> /files/*
func (rc *RouteConverter) AxonToEcho(axonPath string) string {
	// Handle wildcard first: /files/{*} -> /files/*
	if strings.Contains(axonPath, "{*}") {
		axonPath = strings.ReplaceAll(axonPath, "{*}", "*")
	}

	// Regex to match Axon parameter syntax: {param:type}
	axonParamRegex := regexp.MustCompile(`\{([^:}]+):[^}]+\}`)

	// Replace with Echo parameter syntax: :param
	echoPath := axonParamRegex.ReplaceAllString(axonPath, `:$1`)

	return echoPath
}

// EchoToAxon converts Echo route syntax to Axon route syntax (for reverse conversion)
// Converts: /users/:id -> /users/{id:string} (defaults to string type)
func (rc *RouteConverter) EchoToAxon(echoPath string, paramTypes map[string]string) string {
	// Regex to match Echo parameter syntax: :param
	echoParamRegex := regexp.MustCompile(`:([a-zA-Z_][a-zA-Z0-9_]*)`)

	// Replace with Axon parameter syntax using provided types or default to string
	axonPath := echoParamRegex.ReplaceAllStringFunc(echoPath, func(match string) string {
		paramName := strings.TrimPrefix(match, ":")
		paramType := "string" // default type
		if paramTypes != nil {
			if t, exists := paramTypes[paramName]; exists {
				paramType = t
			}
		}
		return "{" + paramName + ":" + paramType + "}"
	})

	return axonPath
}

// ExtractParameterInfo extracts parameter names and types from Axon route syntax
// Returns a map of parameter names to their types
func (rc *RouteConverter) ExtractParameterInfo(axonPath string) map[string]string {
	paramInfo := make(map[string]string)

	// Handle wildcard parameter
	if strings.Contains(axonPath, "{*}") {
		paramInfo["*"] = "string" // Wildcard parameters are always strings
	}

	// Regex to match Axon parameter syntax: {param:type}
	axonParamRegex := regexp.MustCompile(`\{([^:}]+):([^}]+)\}`)

	matches := axonParamRegex.FindAllStringSubmatch(axonPath, -1)
	for _, match := range matches {
		if len(match) == 3 {
			paramName := match[1]
			paramType := match[2]
			paramInfo[paramName] = paramType
		}
	}

	return paramInfo
}

// ValidateAxonPath validates that an Axon path has correct syntax
func (rc *RouteConverter) ValidateAxonPath(axonPath string) error {
	// Check for unclosed braces
	openBraces := strings.Count(axonPath, "{")
	closeBraces := strings.Count(axonPath, "}")
	if openBraces != closeBraces {
		return fmt.Errorf("mismatched braces in path: %s", axonPath)
	}

	// Special validation for wildcard routes
	if strings.Contains(axonPath, "{*}") {
		if !strings.HasSuffix(axonPath, "{*}") {
			return fmt.Errorf("wildcard {*} must be at the end of path: %s", axonPath)
		}
		if strings.Count(axonPath, "{*}") > 1 {
			return fmt.Errorf("only one wildcard {*} allowed per route: %s", axonPath)
		}

		// Remove wildcard from further validation since it doesn't follow {name:type} format
		axonPath = strings.ReplaceAll(axonPath, "{*}", "")
	}

	// Check parameter syntax for remaining parameters
	axonParamRegex := regexp.MustCompile(`\{([^:}]+):([^}]+)\}`)
	invalidParamRegex := regexp.MustCompile(`\{[^}]*\}`)

	// Find all parameter-like patterns
	allParams := invalidParamRegex.FindAllString(axonPath, -1)
	validParams := axonParamRegex.FindAllString(axonPath, -1)

	if len(allParams) != len(validParams) {
		return fmt.Errorf("invalid parameter syntax in path: %s (use {name:type} format, or {*} for wildcard)", axonPath)
	}

	return nil
}

// Global converter instance
var DefaultRouteConverter = NewRouteConverter()

// Convenience functions
func AxonToEcho(axonPath string) string {
	return DefaultRouteConverter.AxonToEcho(axonPath)
}

func EchoToAxon(echoPath string, paramTypes map[string]string) string {
	return DefaultRouteConverter.EchoToAxon(echoPath, paramTypes)
}

func ExtractParameterInfo(axonPath string) map[string]string {
	return DefaultRouteConverter.ExtractParameterInfo(axonPath)
}
