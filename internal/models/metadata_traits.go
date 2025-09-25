package models

// MetadataTraits contains composable trait structs that can be embedded
// to avoid duplication while maintaining flexibility

// BaseMetadataTrait provides core metadata functionality
// This replaces the existing BaseMetadata but with interface implementation
type BaseMetadataTrait struct {
	Name         string       // name of the component
	StructName   string       // name of the struct
	Dependencies []Dependency // dependencies injected via fx.In
}

// GetName returns the component name
func (b *BaseMetadataTrait) GetName() string {
	return b.Name
}

// GetStructName returns the struct name
func (b *BaseMetadataTrait) GetStructName() string {
	return b.StructName
}

// GetDependencies returns the dependencies
func (b *BaseMetadataTrait) GetDependencies() []Dependency {
	return b.Dependencies
}

// LifecycleTrait provides lifecycle-related functionality
type LifecycleTrait struct {
	HasStart     bool   // whether service has Start(context.Context) error method
	HasStop      bool   // whether service has Stop(context.Context) error method
	HasLifecycle bool   // whether lifecycle is enabled
	StartMode    string // lifecycle start mode: "Same" (default) or "Background"
}

// HasStartMethod returns whether the component has a Start method
func (l *LifecycleTrait) HasStartMethod() bool {
	return l.HasStart
}

// HasStopMethod returns whether the component has a Stop method
func (l *LifecycleTrait) HasStopMethod() bool {
	return l.HasStop
}

// IsLifecycleEnabled returns whether lifecycle is enabled
func (l *LifecycleTrait) IsLifecycleEnabled() bool {
	return l.HasLifecycle
}

// GetStartMode returns the start mode
func (l *LifecycleTrait) GetStartMode() string {
	if l.StartMode == "" {
		return "Same"
	}
	return l.StartMode
}

// PriorityTrait provides priority ordering functionality
type PriorityTrait struct {
	Priority int // registration priority (lower = first, higher = last)
}

// GetPriority returns the priority
func (p *PriorityTrait) GetPriority() int {
	return p.Priority
}

// ManualModuleTrait provides manual module functionality
type ManualModuleTrait struct {
	IsManual   bool   // whether component uses manual module
	ModuleName string // name of manual module (if applicable)
}

// IsManualModule returns whether this uses manual module
func (m *ManualModuleTrait) IsManualModule() bool {
	return m.IsManual
}

// GetModuleName returns the module name
func (m *ManualModuleTrait) GetModuleName() string {
	return m.ModuleName
}

// MiddlewareTrait provides middleware functionality
type MiddlewareTrait struct {
	Middlewares []string // middleware names to apply
}

// GetMiddlewares returns the middlewares
func (m *MiddlewareTrait) GetMiddlewares() []string {
	return m.Middlewares
}

// PathTrait provides package path functionality
type PathTrait struct {
	PackagePath string // package where component is defined
}

// GetPackagePath returns the package path
func (p *PathTrait) GetPackagePath() string {
	return p.PackagePath
}

// ServiceModeTrait provides service mode functionality
type ServiceModeTrait struct {
	Mode string // lifecycle mode: "Singleton" (default) or "Transient"
}

// GetServiceMode returns the service mode
func (s *ServiceModeTrait) GetServiceMode() string {
	if s.Mode == "" {
		return "Singleton"
	}
	return s.Mode
}

// ConstructorTrait provides custom constructor functionality
type ConstructorTrait struct {
	Constructor string // custom constructor function name
}

// GetConstructor returns the constructor name
func (c *ConstructorTrait) GetConstructor() string {
	return c.Constructor
}