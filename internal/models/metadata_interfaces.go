package models

// Metadata is the base interface for all metadata types
type Metadata interface {
	GetName() string
	GetStructName() string
	GetDependencies() []Dependency
}

// LifecycleAware represents components that can have lifecycle methods
type LifecycleAware interface {
	HasStartMethod() bool
	HasStopMethod() bool
	IsLifecycleEnabled() bool
}

// PriorityAware represents components that have priority ordering
type PriorityAware interface {
	GetPriority() int
}

// ManualModuleAware represents components that can use manual module configuration
type ManualModuleAware interface {
	IsManualModule() bool
	GetModuleName() string
}

// MiddlewareAware represents components that can have middleware applied
type MiddlewareAware interface {
	GetMiddlewares() []string
}

// PathAware represents components that have a package path
type PathAware interface {
	GetPackagePath() string
}