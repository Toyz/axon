package parser

import (
	"go/ast"

	"github.com/toyz/axon/internal/models"
)

// AnnotationParser defines the interface for parsing Go source files and extracting annotation metadata
type AnnotationParser interface {
	ParseDirectory(path string) (*models.PackageMetadata, error)
	ExtractAnnotations(file *ast.File, fileName string) ([]models.Annotation, error)
	SetSkipParserValidation(skip bool)
	SetSkipMiddlewareValidation(skip bool)
	ValidateCustomParsersWithRegistry(metadata *models.PackageMetadata, parserRegistry map[string]models.RouteParserMetadata) error
}
