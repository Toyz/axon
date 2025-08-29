package parser

import (
	"go/ast"

	"github.com/toyz/axon/internal/models"
	"github.com/toyz/axon/pkg/axon"
)

// AnnotationParser defines the interface for parsing Go source files and extracting annotation metadata
type AnnotationParser interface {
	ParseDirectory(path string) (*models.PackageMetadata, error)
	ExtractAnnotations(file *ast.File, fileName string) ([]models.Annotation, error)
	SetSkipParserValidation(skip bool)
	SetSkipMiddlewareValidation(skip bool)
	ValidateCustomParsersWithRegistry(metadata *models.PackageMetadata, parserRegistry map[string]axon.RouteParserMetadata) error
}