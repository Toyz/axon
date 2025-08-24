package generator

import "github.com/toyz/axon/internal/models"

// CodeGenerator defines the interface for generating FX modules and wiring code based on parsed annotations
type CodeGenerator interface {
	GenerateModule(metadata *models.PackageMetadata) (*models.GeneratedModule, error)
	GenerateModuleWithModule(metadata *models.PackageMetadata, moduleName string) (*models.GeneratedModule, error)
	GetParserRegistry() ParserRegistryInterface
}

// RouteGenerator defines the interface for generating wrapper functions for HTTP route handlers
type RouteGenerator interface {
	GenerateHandler(route *models.RouteMetadata) (string, error)
	GenerateParameterBinding(params []models.Parameter) (string, error)
}

// LifecycleManager defines the interface for handling component lifecycle for services with -Init flag
type LifecycleManager interface {
	GenerateLifecycleProvider(service *models.ServiceMetadata) (string, error)
}