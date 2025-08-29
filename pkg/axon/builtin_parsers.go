package axon

import (
	"strconv"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

// BuiltinParsers contains metadata for all built-in parsers
var BuiltinParsers = map[string]RouteParserMetadata{
	"int": {
		TypeName:       "int",
		FunctionName:   "ParseInt",
		PackagePath:    "builtin",
		ParameterTypes: []string{"echo.Context", "string"},
		ReturnTypes:    []string{"int", "error"},
		FileName:       "builtin",
		Line:           0,
	},
	"string": {
		TypeName:       "string",
		FunctionName:   "ParseString",
		PackagePath:    "builtin",
		ParameterTypes: []string{"echo.Context", "string"},
		ReturnTypes:    []string{"string", "error"},
		FileName:       "builtin",
		Line:           0,
	},
	"float64": {
		TypeName:       "float64",
		FunctionName:   "ParseFloat64",
		PackagePath:    "builtin",
		ParameterTypes: []string{"echo.Context", "string"},
		ReturnTypes:    []string{"float64", "error"},
		FileName:       "builtin",
		Line:           0,
	},
	"float32": {
		TypeName:       "float32",
		FunctionName:   "ParseFloat32",
		PackagePath:    "builtin",
		ParameterTypes: []string{"echo.Context", "string"},
		ReturnTypes:    []string{"float32", "error"},
		FileName:       "builtin",
		Line:           0,
	},
	"uuid.UUID": {
		TypeName:       "uuid.UUID",
		FunctionName:   "ParseUUID",
		PackagePath:    "builtin",
		ParameterTypes: []string{"echo.Context", "string"},
		ReturnTypes:    []string{"uuid.UUID", "error"},
		FileName:       "builtin",
		Line:           0,
	},
}

// ParserAliases maps convenient aliases to their full type names
var ParserAliases = map[string]string{
	"UUID":   "uuid.UUID",
	"float":  "float64", // Default float to float64
	"double": "float64", // Common alias for float64
}

// ParseInt parses a string parameter to int
func ParseInt(c echo.Context, paramValue string) (int, error) {
	return strconv.Atoi(paramValue)
}

// ParseString returns the string parameter as-is (no conversion needed)
func ParseString(c echo.Context, paramValue string) (string, error) {
	return paramValue, nil
}

// ParseFloat64 parses a string parameter to float64
func ParseFloat64(c echo.Context, paramValue string) (float64, error) {
	return strconv.ParseFloat(paramValue, 64)
}

// ParseFloat32 parses a string parameter to float32
func ParseFloat32(c echo.Context, paramValue string) (float32, error) {
	val, err := strconv.ParseFloat(paramValue, 32)
	if err != nil {
		return 0, err
	}
	return float32(val), nil
}

// ParseUUID parses a string parameter to uuid.UUID
func ParseUUID(c echo.Context, paramValue string) (uuid.UUID, error) {
	return uuid.Parse(paramValue)
}

// GetBuiltinParser returns a built-in parser by type name, checking aliases first
func GetBuiltinParser(typeName string) (RouteParserMetadata, bool) {
	// Check if it's an alias first
	if actualType, isAlias := ParserAliases[typeName]; isAlias {
		typeName = actualType
	}
	
	parser, exists := BuiltinParsers[typeName]
	return parser, exists
}

// IsBuiltinType checks if a type is a built-in type, including aliases
func IsBuiltinType(typeName string) bool {
	// Check aliases first
	if actualType, isAlias := ParserAliases[typeName]; isAlias {
		typeName = actualType
	}
	
	_, exists := BuiltinParsers[typeName]
	return exists
}

// ResolveTypeAlias resolves a type alias to its actual type name
func ResolveTypeAlias(typeName string) string {
	if actualType, isAlias := ParserAliases[typeName]; isAlias {
		return actualType
	}
	return typeName
}

// GetAllBuiltinTypes returns all built-in type names including aliases
func GetAllBuiltinTypes() []string {
	types := make([]string, 0, len(BuiltinParsers)+len(ParserAliases))
	
	// Add actual types
	for typeName := range BuiltinParsers {
		types = append(types, typeName)
	}
	
	// Add aliases
	for alias := range ParserAliases {
		types = append(types, alias)
	}
	
	return types
}