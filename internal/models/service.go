package models

// Dependency represents a dependency with both field name and type
type Dependency struct {
	Name   string // field name in the struct
	Type   string // type of the dependency
	IsInit bool   // whether this should be initialized (not injected)
}

// CoreServiceMetadata represents a core service using composition
type CoreServiceMetadata struct {
	BaseMetadataTrait
	LifecycleTrait
	ManualModuleTrait
	ServiceModeTrait
	ConstructorTrait
}

// LoggerMetadata represents a logger service using composition
type LoggerMetadata struct {
	BaseMetadataTrait
	LifecycleTrait
	ManualModuleTrait
}

// ServiceMetadata represents service information for lifecycle management using composition
type ServiceMetadata struct {
	BaseMetadataTrait
	LifecycleTrait
}
