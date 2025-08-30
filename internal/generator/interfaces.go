package generator

import (
	"github.com/toyz/axon/internal/models"
	"github.com/toyz/axon/pkg/axon"
)

// CodeGenerator defines the interface for generating FX modules and wiring code based on parsed annotations
type CodeGenerator interface {
	GenerateModule(metadata *models.PackageMetadata) (*models.GeneratedModule, error)
	GenerateModuleWithModule(metadata *models.PackageMetadata, moduleName string) (*models.GeneratedModule, error)
	GenerateModuleWithPackagePaths(metadata *models.PackageMetadata, moduleName string, packagePaths map[string]string) (*models.GeneratedModule, error)
	GenerateModuleWithRequiredPackages(metadata *models.PackageMetadata, moduleName string, packagePaths map[string]string, requiredPackages []string) (*models.GeneratedModule, error)
	GetParserRegistry() axon.ParserRegistryInterface
}

// Note: RouteGenerator interface was removed as it was unused.
// Route generation is now handled directly by functions in the templates package.

// LifecycleManager defines the interface for handling component lifecycle for services with -Init flag
type LifecycleManager interface {
	GenerateLifecycleProvider(service *models.ServiceMetadata) (string, error)
}