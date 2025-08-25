package annotations

import "fmt"

// AnnotationType represents the type of annotation
type AnnotationType int

const (
	CoreAnnotation AnnotationType = iota
	RouteAnnotation
	ControllerAnnotation
	MiddlewareAnnotation
	InterfaceAnnotation
	InjectAnnotation
	InitAnnotation
	LoggerAnnotation
	RouteParserAnnotation
)

// String returns the string representation of the annotation type
func (a AnnotationType) String() string {
	switch a {
	case CoreAnnotation:
		return "core"
	case RouteAnnotation:
		return "route"
	case ControllerAnnotation:
		return "controller"
	case MiddlewareAnnotation:
		return "middleware"
	case InterfaceAnnotation:
		return "interface"
	case InjectAnnotation:
		return "inject"
	case InitAnnotation:
		return "init"
	case LoggerAnnotation:
		return "logger"
	case RouteParserAnnotation:
		return "route_parser"
	default:
		return "unknown"
	}
}

// ParseAnnotationType converts string to AnnotationType
func ParseAnnotationType(s string) (AnnotationType, error) {
	switch s {
	case "core":
		return CoreAnnotation, nil
	case "route":
		return RouteAnnotation, nil
	case "controller":
		return ControllerAnnotation, nil
	case "middleware":
		return MiddlewareAnnotation, nil
	case "interface":
		return InterfaceAnnotation, nil
	case "inject":
		return InjectAnnotation, nil
	case "init":
		return InitAnnotation, nil
	case "logger":
		return LoggerAnnotation, nil
	case "route_parser":
		return RouteParserAnnotation, nil
	default:
		return 0, fmt.Errorf("unknown annotation type: %s", s)
	}
}

// SourceLocation represents the location of an annotation in source code
type SourceLocation struct {
	File   string // File path
	Line   int    // Line number (1-based)
	Column int    // Column number (1-based)
}

// ParsedAnnotation represents a fully parsed annotation with type-safe parameters
type ParsedAnnotation struct {
	Type       AnnotationType         // Annotation type enum
	Target     string                 // Target struct/function name
	Parameters map[string]interface{} // Typed parameters
	Flags      []string               // Boolean flags
	Location   SourceLocation         // Source location
	Raw        string                 // Original annotation text
}

// GetString returns a string parameter value with optional default
func (p *ParsedAnnotation) GetString(paramName string, defaultValue ...string) string {
	if value, exists := p.Parameters[paramName]; exists {
		if strValue, ok := value.(string); ok {
			return strValue
		}
	}
	if len(defaultValue) > 0 {
		return defaultValue[0]
	}
	return ""
}

// GetBool returns a boolean parameter value with optional default
func (p *ParsedAnnotation) GetBool(paramName string, defaultValue ...bool) bool {
	if value, exists := p.Parameters[paramName]; exists {
		if boolValue, ok := value.(bool); ok {
			return boolValue
		}
	}
	if len(defaultValue) > 0 {
		return defaultValue[0]
	}
	return false
}

// GetInt returns an integer parameter value with optional default
func (p *ParsedAnnotation) GetInt(paramName string, defaultValue ...int) int {
	if value, exists := p.Parameters[paramName]; exists {
		if intValue, ok := value.(int); ok {
			return intValue
		}
	}
	if len(defaultValue) > 0 {
		return defaultValue[0]
	}
	return 0
}

// GetStringSlice returns a string slice parameter value with optional default
func (p *ParsedAnnotation) GetStringSlice(paramName string, defaultValue ...[]string) []string {
	if value, exists := p.Parameters[paramName]; exists {
		if sliceValue, ok := value.([]string); ok {
			return sliceValue
		}
	}
	if len(defaultValue) > 0 {
		return defaultValue[0]
	}
	return nil
}

// HasParameter checks if a parameter exists
func (p *ParsedAnnotation) HasParameter(paramName string) bool {
	_, exists := p.Parameters[paramName]
	return exists
}

// GetStringWithConversion returns a string parameter value with type conversion and optional default
func (p *ParsedAnnotation) GetStringWithConversion(paramName string, defaultValue ...string) string {
	if value, exists := p.Parameters[paramName]; exists {
		if converted, err := ConvertToString(value); err == nil {
			return converted
		}
	}
	if len(defaultValue) > 0 {
		return defaultValue[0]
	}
	return ""
}

// GetBoolWithConversion returns a boolean parameter value with type conversion and optional default
func (p *ParsedAnnotation) GetBoolWithConversion(paramName string, defaultValue ...bool) bool {
	if value, exists := p.Parameters[paramName]; exists {
		if converted, err := ConvertToBool(value); err == nil {
			return converted
		}
	}
	if len(defaultValue) > 0 {
		return defaultValue[0]
	}
	return false
}

// GetIntWithConversion returns an integer parameter value with type conversion and optional default
func (p *ParsedAnnotation) GetIntWithConversion(paramName string, defaultValue ...int) int {
	if value, exists := p.Parameters[paramName]; exists {
		if converted, err := ConvertToInt(value); err == nil {
			return converted
		}
	}
	if len(defaultValue) > 0 {
		return defaultValue[0]
	}
	return 0
}

// GetStringSliceWithConversion returns a string slice parameter value with type conversion and optional default
func (p *ParsedAnnotation) GetStringSliceWithConversion(paramName string, defaultValue ...[]string) []string {
	if value, exists := p.Parameters[paramName]; exists {
		if converted, err := ConvertToStringSlice(value); err == nil {
			return converted
		}
	}
	if len(defaultValue) > 0 {
		return defaultValue[0]
	}
	return nil
}

// ParameterType represents the type of a parameter
type ParameterType int

const (
	StringType ParameterType = iota
	BoolType
	IntType
	StringSliceType
)

// String returns the string representation of the parameter type
func (p ParameterType) String() string {
	switch p {
	case StringType:
		return "string"
	case BoolType:
		return "bool"
	case IntType:
		return "int"
	case StringSliceType:
		return "[]string"
	default:
		return "unknown"
	}
}

// ParameterSpec defines the specification for an annotation parameter
type ParameterSpec struct {
	Type         ParameterType           // Parameter type
	Required     bool                    // Whether parameter is required
	DefaultValue interface{}             // Default value if not provided
	Description  string                  // Parameter description
	Validator    func(interface{}) error // Custom validator function
}

// CustomValidator represents a custom validation function for annotations
type CustomValidator func(*ParsedAnnotation) error

// AnnotationSchema defines the schema for an annotation type
type AnnotationSchema struct {
	Type        AnnotationType            // Annotation type enum
	Description string                    // Human-readable description
	Parameters  map[string]ParameterSpec  // Parameter specifications
	Validators  []CustomValidator         // Custom validation functions
	Examples    []string                  // Usage examples
}

// Type conversion utilities

// ConvertToString converts any value to a string
func ConvertToString(value interface{}) (string, error) {
	if strValue, ok := value.(string); ok {
		return strValue, nil
	}
	return fmt.Sprintf("%v", value), nil
}

// ConvertToBool converts various types to boolean
func ConvertToBool(value interface{}) (bool, error) {
	switch v := value.(type) {
	case bool:
		return v, nil
	case string:
		return parseBoolString(v)
	case int:
		return v != 0, nil
	case int64:
		return v != 0, nil
	case float64:
		return v != 0, nil
	default:
		return false, fmt.Errorf("cannot convert %T to bool", value)
	}
}

// ConvertToInt converts various types to integer
func ConvertToInt(value interface{}) (int, error) {
	switch v := value.(type) {
	case int:
		return v, nil
	case int64:
		return int(v), nil
	case float64:
		return int(v), nil
	case string:
		return parseIntString(v)
	case bool:
		if v {
			return 1, nil
		}
		return 0, nil
	default:
		return 0, fmt.Errorf("cannot convert %T to int", value)
	}
}

// ConvertToStringSlice converts various types to string slice
func ConvertToStringSlice(value interface{}) ([]string, error) {
	switch v := value.(type) {
	case []string:
		return v, nil
	case string:
		// Handle comma-separated values
		if v == "" {
			return []string{}, nil
		}
		if containsComma(v) {
			return parseCommaSeparated(v), nil
		}
		return []string{v}, nil
	case []interface{}:
		result := make([]string, len(v))
		for i, item := range v {
			result[i] = fmt.Sprintf("%v", item)
		}
		return result, nil
	default:
		// Convert single value to slice
		return []string{fmt.Sprintf("%v", value)}, nil
	}
}

// Helper functions for type conversion

func parseBoolString(s string) (bool, error) {
	switch s {
	case "true", "True", "TRUE", "1", "yes", "Yes", "YES", "on", "On", "ON":
		return true, nil
	case "false", "False", "FALSE", "0", "no", "No", "NO", "off", "Off", "OFF":
		return false, nil
	default:
		return false, fmt.Errorf("invalid boolean string: %s", s)
	}
}

func parseIntString(s string) (int, error) {
	var result int
	if _, err := fmt.Sscanf(s, "%d", &result); err != nil {
		return 0, fmt.Errorf("invalid integer string: %s", s)
	}
	return result, nil
}

func containsComma(s string) bool {
	for _, char := range s {
		if char == ',' {
			return true
		}
	}
	return false
}

func parseCommaSeparated(s string) []string {
	parts := make([]string, 0)
	current := ""
	inQuotes := false
	
	for i, char := range s {
		switch char {
		case '"', '\'':
			inQuotes = !inQuotes
			current += string(char)
		case ',':
			if !inQuotes {
				parts = append(parts, trimAndUnquote(current))
				current = ""
			} else {
				current += string(char)
			}
		default:
			current += string(char)
		}
		
		// Handle last part
		if i == len(s)-1 && current != "" {
			parts = append(parts, trimAndUnquote(current))
		}
	}
	
	return parts
}

func trimAndUnquote(s string) string {
	// Trim whitespace
	for len(s) > 0 && (s[0] == ' ' || s[0] == '\t') {
		s = s[1:]
	}
	for len(s) > 0 && (s[len(s)-1] == ' ' || s[len(s)-1] == '\t') {
		s = s[:len(s)-1]
	}
	
	// Remove quotes if present
	if len(s) >= 2 {
		if (s[0] == '"' && s[len(s)-1] == '"') || (s[0] == '\'' && s[len(s)-1] == '\'') {
			return s[1 : len(s)-1]
		}
	}
	return s
}