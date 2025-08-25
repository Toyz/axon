package annotations

import (
	"fmt"
	"strings"
)

// Built-in annotation schemas

// CoreAnnotationSchema defines the schema for //axon::core annotations
var CoreAnnotationSchema = AnnotationSchema{
	Type:        CoreAnnotation,
	Description: "Marks a struct as a core service for dependency injection",
	Parameters: map[string]ParameterSpec{
		"Mode": {
			Type:         StringType,
			Required:     false,
			DefaultValue: "Singleton",
			Description:  "Service lifecycle mode: 'Singleton' (default) or 'Transient'",
			Validator: func(v interface{}) error {
				mode := v.(string)
				if mode != "Singleton" && mode != "Transient" {
					return fmt.Errorf("must be 'Singleton' or 'Transient', got '%s'", mode)
				}
				return nil
			},
		},
		"Init": {
			Type:         StringType,
			Required:     false,
			DefaultValue: "Same",
			Description:  "Lifecycle execution mode: 'Same' (default, synchronous) or 'Background' (async)",
			Validator: func(v interface{}) error {
				mode := v.(string)
				if mode != "Same" && mode != "Background" {
					return fmt.Errorf("must be 'Same' or 'Background', got '%s'", mode)
				}
				return nil
			},
		},
		"Manual": {
			Type:        StringType,
			Required:    false,
			Description: "Custom module name for manual registration",
		},
	},
	Examples: []string{
		"//axon::core",
		"//axon::core -Mode=Transient",
		"//axon::core -Init=Background",
		"//axon::core -Mode=Singleton -Init=Same",
		"//axon::core -Manual=\"CustomModule\"",
		"//axon::core -Mode=Transient -Init=Background -Manual=\"AsyncService\"",
	},
}

// RouteAnnotationSchema defines the schema for //axon::route annotations
var RouteAnnotationSchema = AnnotationSchema{
	Type:        RouteAnnotation,
	Description: "Defines an HTTP route handler",
	Parameters: map[string]ParameterSpec{
		"method": {
			Type:        StringType,
			Required:    true,
			Description: "HTTP method (GET, POST, PUT, DELETE, PATCH, HEAD, OPTIONS)",
			Validator: func(v interface{}) error {
				method := strings.ToUpper(v.(string))
				validMethods := []string{"GET", "POST", "PUT", "DELETE", "PATCH", "HEAD", "OPTIONS"}
				for _, valid := range validMethods {
					if method == valid {
						return nil
					}
				}
				return fmt.Errorf("must be one of: %s, got '%s'", strings.Join(validMethods, ", "), method)
			},
		},
		"path": {
			Type:        StringType,
			Required:    true,
			Description: "URL path pattern (e.g., /users, /users/{id:int})",
		},
		"Middleware": {
			Type:        StringSliceType,
			Required:    false,
			Description: "Comma-separated list of middleware names to apply to this route",
		},
		"PassContext": {
			Type:         BoolType,
			Required:     false,
			DefaultValue: false,
			Description:  "Whether to pass echo.Context as the first parameter to the handler",
		},
	},
	Examples: []string{
		"//axon::route GET /users",
		"//axon::route POST /users",
		"//axon::route GET /users/{id:int}",
		"//axon::route PUT /users/{id:int} -Middleware=Auth",
		"//axon::route DELETE /users/{id:int} -Middleware=Auth,Logging",
		"//axon::route GET /health -PassContext",
		"//axon::route POST /users -Middleware=Auth,Validation -PassContext",
	},
}

// ControllerAnnotationSchema defines the schema for //axon::controller annotations
var ControllerAnnotationSchema = AnnotationSchema{
	Type:        ControllerAnnotation,
	Description: "Marks a struct as a controller for HTTP request handling",
	Parameters: map[string]ParameterSpec{
		"Prefix": {
			Type:        StringType,
			Required:    false,
			Description: "URL prefix to apply to all routes in this controller (e.g., /api/v1)",
		},
		"Middleware": {
			Type:        StringSliceType,
			Required:    false,
			Description: "Comma-separated list of middleware names to apply to all routes in this controller",
		},
		"Tags": {
			Type:        StringSliceType,
			Required:    false,
			Description: "Comma-separated list of tags for API documentation",
		},
	},
	Examples: []string{
		"//axon::controller",
		"//axon::controller -Prefix=/api/v1",
		"//axon::controller -Middleware=Auth",
		"//axon::controller -Prefix=/api/v1 -Middleware=Auth,Logging",
		"//axon::controller -Tags=Users,Management",
		"//axon::controller -Prefix=/admin -Middleware=Auth,AdminOnly -Tags=Admin",
	},
}

// MiddlewareAnnotationSchema defines the schema for //axon::middleware annotations
var MiddlewareAnnotationSchema = AnnotationSchema{
	Type:        MiddlewareAnnotation,
	Description: "Marks a struct as middleware for request processing",
	Parameters: map[string]ParameterSpec{
		"name": {
			Type:        StringType,
			Required:    true,
			Description: "Name for the middleware",
		},
		"Priority": {
			Type:         IntType,
			Required:     false,
			DefaultValue: 0,
			Description:  "Execution priority (lower numbers execute first)",
		},
		"Global": {
			Type:         BoolType,
			Required:     false,
			DefaultValue: false,
			Description:  "Whether to apply this middleware globally to all routes",
		},
		"Routes": {
			Type:        StringSliceType,
			Required:    false,
			Description: "Comma-separated list of route patterns to apply this middleware to",
		},
	},
	Examples: []string{
		"//axon::middleware",
		"//axon::middleware AuthMiddleware",
		"//axon::middleware -Name=CustomAuth",
		"//axon::middleware -Priority=10",
		"//axon::middleware -Global",
		"//axon::middleware -Routes=/api/*,/admin/*",
		"//axon::middleware -Name=RateLimit -Priority=5 -Routes=/api/*",
	},
}

// InterfaceAnnotationSchema defines the schema for //axon::interface annotations
var InterfaceAnnotationSchema = AnnotationSchema{
	Type:        InterfaceAnnotation,
	Description: "Marks a struct as implementing an interface for dependency injection",
	Parameters: map[string]ParameterSpec{
		"Name": {
			Type:        StringType,
			Required:    false,
			Description: "Custom interface name (defaults to struct name without 'Impl' suffix)",
		},
		"Singleton": {
			Type:         BoolType,
			Required:     false,
			DefaultValue: true,
			Description:  "Whether to register as singleton (true) or transient (false)",
		},
		"Primary": {
			Type:         BoolType,
			Required:     false,
			DefaultValue: false,
			Description:  "Whether this is the primary implementation when multiple implementations exist",
		},
	},
	Examples: []string{
		"//axon::interface",
		"//axon::interface -Name=UserRepository",
		"//axon::interface -Singleton=false",
		"//axon::interface -Primary",
		"//axon::interface -Name=CacheService -Primary",
		"//axon::interface -Name=DatabaseRepo -Singleton=false -Primary",
	},
}

// InjectAnnotationSchema defines the schema for //axon::inject annotations
var InjectAnnotationSchema = AnnotationSchema{
	Type:        InjectAnnotation,
	Description: "Marks a field for dependency injection",
	Parameters: map[string]ParameterSpec{
		"Name": {
			Type:        StringType,
			Required:    false,
			Description: "Custom name for the dependency",
		},
		"Optional": {
			Type:         BoolType,
			Required:     false,
			DefaultValue: false,
			Description:  "Whether the dependency is optional",
		},
	},
	Examples: []string{
		"//axon::inject",
		"//axon::inject -Name=CustomService",
		"//axon::inject -Optional",
		"//axon::inject -Name=CacheService -Optional",
	},
}

// InitAnnotationSchema defines the schema for //axon::init annotations
var InitAnnotationSchema = AnnotationSchema{
	Type:        InitAnnotation,
	Description: "Marks a function for initialization",
	Parameters: map[string]ParameterSpec{
		"Priority": {
			Type:         IntType,
			Required:     false,
			DefaultValue: 0,
			Description:  "Initialization priority (lower numbers execute first)",
		},
		"Async": {
			Type:         BoolType,
			Required:     false,
			DefaultValue: false,
			Description:  "Whether to run initialization asynchronously",
		},
	},
	Examples: []string{
		"//axon::init",
		"//axon::init -Priority=10",
		"//axon::init -Async",
		"//axon::init -Priority=5 -Async",
	},
}

// LoggerAnnotationSchema defines the schema for //axon::logger annotations
var LoggerAnnotationSchema = AnnotationSchema{
	Type:        LoggerAnnotation,
	Description: "Marks a struct as a logger service",
	Parameters:  map[string]ParameterSpec{}, // No parameters - just //axon::logger
	Examples: []string{
		"//axon::logger",
	},
}

// RouteParserAnnotationSchema defines the schema for //axon::route_parser annotations
var RouteParserAnnotationSchema = AnnotationSchema{
	Type:        RouteParserAnnotation,
	Description: "Marks a function as a route parameter parser",
	Parameters: map[string]ParameterSpec{
		"name": {
			Type:        StringType,
			Required:    true,
			Description: "Type name that this parser handles",
		},
	},
	Examples: []string{
		"//axon::route_parser name=UUID",
		"//axon::route_parser name=CustomID", 
		"//axon::route_parser name=time.Time",
	},
}

// RegisterBuiltinSchemas registers all built-in annotation schemas with the given registry
func RegisterBuiltinSchemas(registry AnnotationRegistry) error {
	schemas := []AnnotationSchema{
		CoreAnnotationSchema,
		RouteAnnotationSchema,
		ControllerAnnotationSchema,
		MiddlewareAnnotationSchema,
		InterfaceAnnotationSchema,
		InjectAnnotationSchema,
		InitAnnotationSchema,
		LoggerAnnotationSchema,
		RouteParserAnnotationSchema,
	}

	for _, schema := range schemas {
		if err := registry.Register(schema.Type, schema); err != nil {
			return fmt.Errorf("failed to register %s schema: %w", schema.Type.String(), err)
		}
	}

	return nil
}

// GetBuiltinSchemas returns all built-in annotation schemas
func GetBuiltinSchemas() []AnnotationSchema {
	return []AnnotationSchema{
		CoreAnnotationSchema,
		RouteAnnotationSchema,
		ControllerAnnotationSchema,
		MiddlewareAnnotationSchema,
		InterfaceAnnotationSchema,
		InjectAnnotationSchema,
		InitAnnotationSchema,
		LoggerAnnotationSchema,
		RouteParserAnnotationSchema,
	}
}

// ValidateRouteParameters is a custom validator for route annotations
func ValidateRouteParameters(annotation *ParsedAnnotation) error {
	// Ensure method and path are provided
	method := annotation.GetString("method")
	path := annotation.GetString("path")

	if method == "" {
		return fmt.Errorf("route annotation requires method parameter")
	}

	if path == "" {
		return fmt.Errorf("route annotation requires path parameter")
	}

	// Validate path format
	if !strings.HasPrefix(path, "/") {
		return fmt.Errorf("route path must start with '/', got '%s'", path)
	}

	return nil
}

// ValidateMiddlewareParameters is a custom validator for middleware annotations
func ValidateMiddlewareParameters(annotation *ParsedAnnotation) error {
	// If Routes is specified, validate the patterns
	if routes := annotation.GetStringSlice("Routes"); routes != nil {
		for _, route := range routes {
			if route == "" {
				return fmt.Errorf("middleware route pattern cannot be empty")
			}
			if !strings.HasPrefix(route, "/") {
				return fmt.Errorf("middleware route pattern must start with '/', got '%s'", route)
			}
		}
	}

	return nil
}

// init registers custom validators for schemas that need them
func init() {
	// Add custom validators to route schema
	RouteAnnotationSchema.Validators = []CustomValidator{
		ValidateRouteParameters,
	}

	// Add custom validators to middleware schema
	MiddlewareAnnotationSchema.Validators = []CustomValidator{
		ValidateMiddlewareParameters,
	}
}