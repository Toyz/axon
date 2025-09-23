package models

// MiddlewareMetadata represents a middleware component using composition
type MiddlewareMetadata struct {
	BaseMetadataTrait
	PathTrait
	PriorityTrait
	Parameters map[string]interface{} // parameters from annotation
	IsGlobal   bool                   // whether this middleware should be applied globally
}
