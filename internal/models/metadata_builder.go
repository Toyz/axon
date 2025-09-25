package models

// MetadataBuilder provides a fluent interface for building complex metadata structures
type MetadataBuilder struct {
	base       *BaseMetadataTrait
	lifecycle  *LifecycleTrait
	priority   *PriorityTrait
	manual     *ManualModuleTrait
	middleware *MiddlewareTrait
	path       *PathTrait
	service    *ServiceModeTrait
	constructor *ConstructorTrait
}

// NewMetadataBuilder creates a new metadata builder
func NewMetadataBuilder(name, structName string) *MetadataBuilder {
	return &MetadataBuilder{
		base: &BaseMetadataTrait{
			Name:       name,
			StructName: structName,
		},
	}
}

// WithDependencies adds dependencies to the metadata
func (b *MetadataBuilder) WithDependencies(deps ...Dependency) *MetadataBuilder {
	b.base.Dependencies = append(b.base.Dependencies, deps...)
	return b
}

// WithLifecycle adds lifecycle configuration
func (b *MetadataBuilder) WithLifecycle(hasStart, hasStop bool) *MetadataBuilder {
	b.lifecycle = &LifecycleTrait{
		HasStart:     hasStart,
		HasStop:      hasStop,
		HasLifecycle: hasStart || hasStop,
		StartMode:    "Same",
	}
	return b
}

// WithStartMode sets the lifecycle start mode
func (b *MetadataBuilder) WithStartMode(mode string) *MetadataBuilder {
	if b.lifecycle == nil {
		b.lifecycle = &LifecycleTrait{StartMode: mode}
	} else {
		b.lifecycle.StartMode = mode
	}
	return b
}

// WithPriority sets the priority
func (b *MetadataBuilder) WithPriority(priority int) *MetadataBuilder {
	b.priority = &PriorityTrait{Priority: priority}
	return b
}

// WithManualModule configures manual module settings
func (b *MetadataBuilder) WithManualModule(moduleName string) *MetadataBuilder {
	b.manual = &ManualModuleTrait{
		IsManual:   true,
		ModuleName: moduleName,
	}
	return b
}

// WithMiddlewares adds middleware names
func (b *MetadataBuilder) WithMiddlewares(middlewares ...string) *MetadataBuilder {
	if b.middleware == nil {
		b.middleware = &MiddlewareTrait{}
	}
	b.middleware.Middlewares = append(b.middleware.Middlewares, middlewares...)
	return b
}

// WithPackagePath sets the package path
func (b *MetadataBuilder) WithPackagePath(path string) *MetadataBuilder {
	b.path = &PathTrait{PackagePath: path}
	return b
}

// WithServiceMode sets the service mode (Singleton/Transient)
func (b *MetadataBuilder) WithServiceMode(mode string) *MetadataBuilder {
	b.service = &ServiceModeTrait{Mode: mode}
	return b
}

// WithConstructor sets a custom constructor
func (b *MetadataBuilder) WithConstructor(constructor string) *MetadataBuilder {
	b.constructor = &ConstructorTrait{Constructor: constructor}
	return b
}

// BuildController creates a ControllerMetadata
func (b *MetadataBuilder) BuildController(prefix string, routes []RouteMetadata) *ControllerMetadata {
	controller := &ControllerMetadata{
		BaseMetadataTrait: *b.base,
		Prefix:           prefix,
		Routes:           routes,
	}

	if b.priority != nil {
		controller.PriorityTrait = *b.priority
	}

	if b.middleware != nil {
		controller.MiddlewareTrait = *b.middleware
	}

	return controller
}

// BuildCoreService creates a CoreServiceMetadata
func (b *MetadataBuilder) BuildCoreService() *CoreServiceMetadata {
	service := &CoreServiceMetadata{
		BaseMetadataTrait: *b.base,
	}

	if b.lifecycle != nil {
		service.LifecycleTrait = *b.lifecycle
	}

	if b.manual != nil {
		service.ManualModuleTrait = *b.manual
	}

	if b.service != nil {
		service.ServiceModeTrait = *b.service
	}

	if b.constructor != nil {
		service.ConstructorTrait = *b.constructor
	}

	return service
}

// BuildLogger creates a LoggerMetadata
func (b *MetadataBuilder) BuildLogger() *LoggerMetadata {
	logger := &LoggerMetadata{
		BaseMetadataTrait: *b.base,
	}

	if b.lifecycle != nil {
		logger.LifecycleTrait = *b.lifecycle
	}

	if b.manual != nil {
		logger.ManualModuleTrait = *b.manual
	}

	return logger
}

// BuildService creates a ServiceMetadata
func (b *MetadataBuilder) BuildService() *ServiceMetadata {
	service := &ServiceMetadata{
		BaseMetadataTrait: *b.base,
	}

	if b.lifecycle != nil {
		service.LifecycleTrait = *b.lifecycle
	}

	return service
}

// BuildMiddleware creates a MiddlewareMetadata
func (b *MetadataBuilder) BuildMiddleware(params map[string]interface{}, isGlobal bool) *MiddlewareMetadata {
	middleware := &MiddlewareMetadata{
		BaseMetadataTrait: *b.base,
		Parameters:        params,
		IsGlobal:          isGlobal,
	}

	if b.path != nil {
		middleware.PathTrait = *b.path
	}

	if b.priority != nil {
		middleware.PriorityTrait = *b.priority
	}

	return middleware
}

// BuildInterface creates a InterfaceMetadata
func (b *MetadataBuilder) BuildInterface(methods []Method) *InterfaceMetadata {
	iface := &InterfaceMetadata{
		BaseMetadataTrait: *b.base,
		Methods:           methods,
	}

	if b.path != nil {
		iface.PathTrait = *b.path
	}

	return iface
}