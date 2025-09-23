package annotations

import (
	"fmt"

	"github.com/toyz/axon/internal/errors"
	"github.com/toyz/axon/internal/utils"
)

// Built-in annotation schemas

// ServiceAnnotationSchema defines the schema for //axon::service annotations
var ServiceAnnotationSchema = AnnotationSchema{
	Type:        ServiceAnnotation,
	Description: "Marks a struct as a service for dependency injection",
	Parameters: map[string]ParameterSpec{
		"Mode":        ModeParameterSpec(),
		"Init":        InitParameterSpec(),
		"Manual":      ManualParameterSpec(),
		"Constructor": ConstructorParameterSpec(),
	},
	Examples: []string{
		"//axon::service",
		"//axon::service -Mode=Transient",
		"//axon::service -Init=Background",
		"//axon::service -Constructor=NewCustomUserService",
		"//axon::service -Mode=Singleton -Init=Same",
		"//axon::service -Manual=\"CustomModule\"",
		"//axon::service -Mode=Transient -Init=Background -Manual=\"AsyncService\"",
		"//axon::service -Constructor=NewCustomService -Mode=Transient",
	},
}

// CoreAnnotationSchema is an alias to ServiceAnnotationSchema for backward compatibility (DEPRECATED: use ServiceAnnotationSchema)
var CoreAnnotationSchema = AnnotationSchema{
	Type:        CoreAnnotation,
	Description: ServiceAnnotationSchema.Description + " (DEPRECATED: use //axon::service)",
	Parameters:  ServiceAnnotationSchema.Parameters, // Same parameters as ServiceAnnotationSchema
	Examples: []string{
		"//axon::core",
		"//axon::core -Mode=Transient",
		"//axon::core -Init=Background",
		"//axon::core -Constructor=NewCustomService",
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
		"method":      HTTPMethodParameterSpec(),
		"path":        URLPathParameterSpec(),
		"Middleware":  MiddlewareParameterSpec(),
		"PassContext": PassContextParameterSpec(),
		"Priority":    PriorityParameterSpec(),
	},
	Examples: []string{
		"//axon::route GET /users",
		"//axon::route POST /users",
		"//axon::route GET /users/{id:int}",
		"//axon::route PUT /users/{id:int} -Middleware=Auth",
		"//axon::route DELETE /users/{id:int} -Middleware=Auth,Logging",
		"//axon::route GET /health -PassContext",
		"//axon::route POST /users -Middleware=Auth,Validation -PassContext",
		"//axon::route GET /users/profile -Priority=10  // Higher priority than /users/{id}",
	},
}

// ControllerAnnotationSchema defines the schema for //axon::controller annotations
var ControllerAnnotationSchema = AnnotationSchema{
	Type:        ControllerAnnotation,
	Description: "Marks a struct as a controller for HTTP request handling",
	Parameters: map[string]ParameterSpec{
		"Prefix":     PrefixParameterSpec(),
		"Middleware": MiddlewareParameterSpec(),
		"Priority":   PriorityParameterSpec(),
	},
	Examples: []string{
		"//axon::controller",
		"//axon::controller -Prefix=/api/v1",
		"//axon::controller -Prefix=/users/{id:int}",
		"//axon::controller -Middleware=Auth",
		"//axon::controller -Prefix=/api/v1 -Middleware=Auth,Logging",
		"//axon::controller -Priority=50",
		"//axon::controller -Priority=999 -Prefix=/ // Catch-all route, loads last",
		"//axon::controller -Prefix=/users/{userId:int} -Middleware=Auth",
	},
}

// MiddlewareAnnotationSchema defines the schema for //axon::middleware annotations
var MiddlewareAnnotationSchema = AnnotationSchema{
	Type:        MiddlewareAnnotation,
	Description: "Marks a struct as middleware for request processing",
	Parameters: map[string]ParameterSpec{
		"Name":     NameParameterSpec("Name for the middleware (can be provided as positional parameter)"),
		"Global":   GlobalParameterSpec(),
		"Priority": PriorityParameterSpec(),
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
		"Name": NameParameterSpec("Custom interface name (defaults to struct name without 'Impl' suffix)"),
	},
	Examples: []string{
		"//axon::interface",
		"//axon::interface -Name=UserRepository",
	},
}

// InjectAnnotationSchema defines the schema for //axon::inject annotations
var InjectAnnotationSchema = AnnotationSchema{
	Type:        InjectAnnotation,
	Description: "Marks a field for dependency injection",
	Parameters:  map[string]ParameterSpec{},
	Examples: []string{
		"//axon::inject",
	},
}

// InitAnnotationSchema defines the schema for //axon::init annotations
var InitAnnotationSchema = AnnotationSchema{
	Type:        InitAnnotation,
	Description: "Marks a function for initialization",
	Parameters:  map[string]ParameterSpec{},
	Examples: []string{
		"//axon::init",
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
		"name": NameParameterSpec("Type name that this parser handles (provided as positional parameter)"),
	},
	Examples: []string{
		"//axon::route_parser UUID",
		"//axon::route_parser CustomID",
		"//axon::route_parser time.Time",
		"//axon::route_parser MyCustomType",
	},
}

// RegisterBuiltinSchemas registers all built-in annotation schemas with the given registry
func RegisterBuiltinSchemas(registry AnnotationRegistry) error {
	for _, schema := range GetBuiltinSchemas() {
		if err := registry.Register(schema.Type, schema); err != nil {
			return errors.WrapRegisterError("component", schema.Type.String()+" schema", err)
		}
	}

	return nil
}

// GetBuiltinSchemas returns all built-in annotation schemas
func GetBuiltinSchemas() []AnnotationSchema {
	return []AnnotationSchema{
		ServiceAnnotationSchema,
		CoreAnnotationSchema, // Deprecated: kept for backward compatibility
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
	// Validate method parameter
	method := annotation.GetString("method")
	if err := utils.ValidateHTTPMethod("method")(method); err != nil {
		return fmt.Errorf("route annotation: %w", err)
	}

	// Validate path parameter
	path := annotation.GetString("path")
	if err := utils.ValidateURLPath("path")(path); err != nil {
		return fmt.Errorf("route annotation: %w", err)
	}

	return nil
}

// ValidateMiddlewareParameters is a custom validator for middleware annotations
func ValidateMiddlewareParameters(annotation *ParsedAnnotation) error {
	// If Routes is specified, validate the patterns
	if routes := annotation.GetStringSlice("Routes"); routes != nil {
		validator := utils.ValidateEach("Routes", utils.ValidateMiddlewareRoute("route"))
		if err := validator(routes); err != nil {
			return fmt.Errorf("middleware annotation: %w", err)
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
