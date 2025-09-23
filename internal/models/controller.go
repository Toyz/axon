package models

// Backward compatibility aliases for external packages
type BaseMetadata = BaseMetadataTrait
type LifecycleMetadata = LifecycleTrait

// ControllerMetadata represents a controller and its routes using composition
type ControllerMetadata struct {
	BaseMetadataTrait
	PriorityTrait
	MiddlewareTrait
	Prefix string          // URL prefix for all routes in this controller
	Routes []RouteMetadata // all routes defined on this controller
}

// RouteMetadata represents an HTTP route handler
type RouteMetadata struct {
	Method      string         // HTTP method (GET, POST, etc.)
	Path        string         // URL path with parameters
	HandlerName string         // name of the handler method
	Parameters  []Parameter    // parameters extracted from path and body
	ReturnType  ReturnTypeInfo // information about return signature
	Middlewares []string       // middleware names to apply
	Flags       []string       // flags like -PassContext
	Priority    int            // route registration priority (lower = first, higher = last)
}

// Parameter represents a route parameter
type Parameter struct {
	Name         string          // parameter name
	Type         string          // Go type (int, string, etc.)
	Source       ParameterSource // where parameter comes from
	Required     bool            // whether parameter is required
	Position     int             // position in handler signature (for context parameters)
	IsCustomType bool            // whether this parameter uses a custom parser
	ParserFunc   string          // function name for custom parsers
}

// ReturnTypeInfo describes handler return signature
type ReturnTypeInfo struct {
	Type         ReturnType // type of return signature
	DataType     string     // type of data returned (if applicable)
	HasError     bool       // whether error is returned
	UsesResponse bool       // whether custom Response struct is used
}
