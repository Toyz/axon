package models

// Dependency represents a dependency with both field name and type
type Dependency struct {
	Name     string // field name in the struct
	Type     string // type of the dependency
	IsInit   bool   // whether this should be initialized (not injected)
}

// LifecycleMetadata contains common lifecycle-related fields
type LifecycleMetadata struct {
	HasStart bool // whether service has Start(context.Context) error method
	HasStop  bool // whether service has Stop(context.Context) error method
}

// CoreServiceMetadata represents a core service
type CoreServiceMetadata struct {
	BaseMetadata
	LifecycleMetadata
	HasLifecycle bool   // whether service has lifecycle methods
	StartMode    string // lifecycle start mode: "Same" (default) or "Background"
	IsManual     bool   // whether service uses manual module
	ModuleName   string // name of manual module (if applicable)
	Mode         string // lifecycle mode: "Singleton" (default) or "Transient"
	Constructor  string // custom constructor function name (if provided)
}

// LoggerMetadata represents a logger service
type LoggerMetadata struct {
	BaseMetadata
	LifecycleMetadata
	HasLifecycle bool   // whether logger has lifecycle methods
	IsManual     bool   // whether logger uses manual module
	ModuleName   string // name of manual module (if applicable)
}

// ServiceMetadata represents service information for lifecycle management
type ServiceMetadata struct {
	BaseMetadata
	LifecycleMetadata
}