package models

// MiddlewareMetadata represents a middleware component
type MiddlewareMetadata struct {
	Name         string       // name of the middleware
	PackagePath  string       // package where middleware is defined
	StructName   string       // name of the middleware struct
	Dependencies []Dependency // dependencies injected via fx.In
}
