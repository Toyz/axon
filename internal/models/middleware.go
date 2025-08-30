package models

// MiddlewareMetadata represents a middleware component
type MiddlewareMetadata struct {
	BaseMetadata
	PackagePath string                 // package where middleware is defined
	Parameters  map[string]interface{} // parameters from annotation
	IsGlobal    bool                   // whether this middleware should be applied globally
	Priority    int                    // priority for global middleware ordering (lower = higher priority)
}
