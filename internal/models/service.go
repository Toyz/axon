package models

// Dependency represents a dependency with both field name and type
type Dependency struct {
	Name   string // field name in the struct
	Type   string // type of the dependency
	IsInit bool   // whether this should be initialized (not injected)
}

// CoreServiceMetadata represents a core service
type CoreServiceMetadata struct {
	Name         string       // name of the service
	StructName   string       // name of the struct
	HasLifecycle bool         // whether service has lifecycle methods
	HasStart     bool         // whether service has Start(context.Context) error method
	HasStop      bool         // whether service has Stop(context.Context) error method
	StartMode    string       // lifecycle start mode: "Same" (default) or "Background"
	IsManual     bool         // whether service uses manual module
	ModuleName   string       // name of manual module (if applicable)
	Mode         string       // lifecycle mode: "Singleton" (default) or "Transient"
	Dependencies []Dependency // dependencies injected via fx.In
}

// LoggerMetadata represents a logger service
type LoggerMetadata struct {
	Name         string       // name of the logger
	StructName   string       // name of the struct
	HasLifecycle bool         // whether logger has lifecycle methods
	HasStart     bool         // whether logger has Start method
	HasStop      bool         // whether logger has Stop method
	IsManual     bool         // whether logger uses manual module
	ModuleName   string       // name of manual module (if applicable)
	Dependencies []Dependency // dependencies injected via fx.In
}

// ServiceMetadata represents service information for lifecycle management
type ServiceMetadata struct {
	Name         string       // name of the service
	HasStart     bool         // whether service has Start method
	HasStop      bool         // whether service has Stop method
	Dependencies []Dependency // service dependencies
}
