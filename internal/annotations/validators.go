package annotations

import (
	"fmt"
	"strings"
)

// Common validation functions to eliminate duplication

// ValidateMode validates service lifecycle mode (Singleton/Transient)
func ValidateMode(v interface{}) error {
	mode := v.(string)
	if mode != "Singleton" && mode != "Transient" {
		return fmt.Errorf("must be 'Singleton' or 'Transient', got '%s'", mode)
	}
	return nil
}

// ValidateInit validates initialization mode (Same/Background)
func ValidateInit(v interface{}) error {
	mode := v.(string)
	if mode != "Same" && mode != "Background" {
		return fmt.Errorf("must be 'Same' or 'Background', got '%s'", mode)
	}
	return nil
}

// ValidateHTTPMethod validates HTTP method names
func ValidateHTTPMethod(v interface{}) error {
	method := strings.ToUpper(v.(string))
	validMethods := []string{"GET", "POST", "PUT", "DELETE", "PATCH", "HEAD", "OPTIONS"}
	for _, valid := range validMethods {
		if method == valid {
			return nil
		}
	}
	return fmt.Errorf("must be one of: %s, got '%s'", strings.Join(validMethods, ", "), method)
}

// ValidateURLPath validates URL path format
func ValidateURLPath(v interface{}) error {
	path := v.(string)
	if !strings.HasPrefix(path, "/") {
		return fmt.Errorf("path must start with '/', got '%s'", path)
	}
	return nil
}

// Common parameter specifications to eliminate duplication

// ModeParameterSpec returns a standard Mode parameter specification
func ModeParameterSpec() ParameterSpec {
	return ParameterSpec{
		Type:         StringType,
		Required:     false,
		DefaultValue: "Singleton",
		Description:  "Service lifecycle mode: 'Singleton' (default) or 'Transient'",
		Validator:    ValidateMode,
	}
}

// InitParameterSpec returns a standard Init parameter specification
func InitParameterSpec() ParameterSpec {
	return ParameterSpec{
		Type:         StringType,
		Required:     false,
		DefaultValue: "Same",
		Description:  "Lifecycle execution mode: 'Same' (default, synchronous) or 'Background' (async)",
		Validator:    ValidateInit,
	}
}

// ManualParameterSpec returns a standard Manual parameter specification
func ManualParameterSpec() ParameterSpec {
	return ParameterSpec{
		Type:        StringType,
		Required:    false,
		Description: "Custom module name for manual registration",
	}
}

// HTTPMethodParameterSpec returns a standard HTTP method parameter specification
func HTTPMethodParameterSpec() ParameterSpec {
	return ParameterSpec{
		Type:        StringType,
		Required:    true,
		Description: "HTTP method (GET, POST, PUT, DELETE, PATCH, HEAD, OPTIONS)",
		Validator:   ValidateHTTPMethod,
	}
}

// URLPathParameterSpec returns a standard URL path parameter specification
func URLPathParameterSpec() ParameterSpec {
	return ParameterSpec{
		Type:        StringType,
		Required:    true,
		Description: "URL path pattern (e.g., /users, /users/{id:int})",
		Validator:   ValidateURLPath,
	}
}

// MiddlewareParameterSpec returns a standard Middleware parameter specification
func MiddlewareParameterSpec() ParameterSpec {
	return ParameterSpec{
		Type:        StringSliceType,
		Required:    false,
		Description: "Comma-separated list of middleware names to apply",
	}
}

// PriorityParameterSpec returns a standard Priority parameter specification
func PriorityParameterSpec() ParameterSpec {
	return ParameterSpec{
		Type:         IntType,
		Required:     false,
		DefaultValue: 100,
		Description:  "Priority for ordering (lower numbers = higher priority)",
	}
}

// PassContextParameterSpec returns a standard PassContext parameter specification
func PassContextParameterSpec() ParameterSpec {
	return ParameterSpec{
		Type:         BoolType,
		Required:     false,
		DefaultValue: false,
		Description:  "Whether to pass echo.Context as the first parameter to the handler",
	}
}

// GlobalParameterSpec returns a standard Global parameter specification
func GlobalParameterSpec() ParameterSpec {
	return ParameterSpec{
		Type:         BoolType,
		Required:     false,
		DefaultValue: false,
		Description:  "Whether to apply this middleware globally to all routes",
	}
}

// NameParameterSpec returns a standard Name parameter specification
func NameParameterSpec(description string) ParameterSpec {
	return ParameterSpec{
		Type:        StringType,
		Required:    false,
		Description: description,
	}
}

// PrefixParameterSpec returns a standard Prefix parameter specification
func PrefixParameterSpec() ParameterSpec {
	return ParameterSpec{
		Type:        StringType,
		Required:    false,
		Description: "URL prefix to apply. Supports parameters (e.g., /api/v1, /users/{id:int})",
	}
}