package annotations

import (
	"fmt"
	"strings"

	"github.com/toyz/axon/internal/utils"
)

// Common validation functions to eliminate duplication

// ValidateMode validates service lifecycle mode (Singleton/Transient)
func ValidateMode(v interface{}) error {
	mode, ok := v.(string)
	if !ok {
		return fmt.Errorf("mode must be a string")
	}
	return utils.ValidateServiceMode("mode")(mode)
}

// ValidateInit validates initialization mode (Same/Background)
func ValidateInit(v interface{}) error {
	mode, ok := v.(string)
	if !ok {
		return fmt.Errorf("init mode must be a string")
	}
	return utils.ValidateInitMode("init")(mode)
}

// ValidateHTTPMethod validates HTTP method names
func ValidateHTTPMethod(v interface{}) error {
	method, ok := v.(string)
	if !ok {
		return fmt.Errorf("method must be a string")
	}
	method = strings.ToUpper(method)
	return utils.ValidateHTTPMethod("method")(method)
}

// ValidateURLPath validates URL path format
func ValidateURLPath(v interface{}) error {
	path, ok := v.(string)
	if !ok {
		return fmt.Errorf("path must be a string")
	}
	return utils.ValidateURLPath("path")(path)
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

// ConstructorParameterSpec returns a standard constructor parameter specification
func ConstructorParameterSpec() ParameterSpec {
	return ParameterSpec{
		Type:        StringType,
		Required:    false,
		Description: "Custom constructor function name (e.g., NewCustomUserService)",
		Validator:   ValidateConstructor,
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

// ValidateConstructor validates constructor function names
func ValidateConstructor(value interface{}) error {
	constructor, ok := value.(string)
	if !ok {
		return fmt.Errorf("constructor must be a string")
	}
	return utils.ValidateConstructorName("constructor")(constructor)
}
