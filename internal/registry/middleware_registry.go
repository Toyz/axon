package registry

import (
	"fmt"
	"strings"

	"github.com/toyz/axon/internal/models"
	"github.com/toyz/axon/internal/utils"
)

// middlewareRegistry implements the MiddlewareRegistry interface
type middlewareRegistry struct {
	*utils.BaseRegistry[string, *models.MiddlewareMetadata]
}

// NewMiddlewareRegistry creates a new middleware registry
func NewMiddlewareRegistry() MiddlewareRegistry {
	registry := &middlewareRegistry{
		BaseRegistry: utils.NewBaseRegistry[string, *models.MiddlewareMetadata](
			"middleware",
			"middleware name",
			"middleware metadata",
		),
	}

	// Set up the default validator for middleware registration
	registry.BaseRegistry.SetValidator(createMiddlewareValidator())

	return registry
}

// createMiddlewareValidator creates the validation function for middleware
func createMiddlewareValidator() utils.RegistryValidator[string, *models.MiddlewareMetadata] {
	return func(key string, value *models.MiddlewareMetadata, existing map[string]*models.MiddlewareMetadata) error {
		// Validate key is not empty
		if key == "" {
			return fmt.Errorf("middleware name cannot be empty")
		}

		// Validate value is not nil
		if value == nil {
			return fmt.Errorf("middleware metadata cannot be nil")
		}

		// Check for duplicate names with better error message
		if existingMiddleware, exists := existing[key]; exists {
			return fmt.Errorf("middleware '%s' is already registered in package '%s'", key, existingMiddleware.PackagePath)
		}

		return nil
	}
}

// Register adds a middleware to the registry
func (r *middlewareRegistry) Register(name string, middleware *models.MiddlewareMetadata) error {
	return r.BaseRegistry.Register(name, middleware)
}

// Validate checks that all middleware names exist in the registry
func (r *middlewareRegistry) Validate(middlewareNames []string) error {
	var missingMiddlewares []string

	for _, name := range middlewareNames {
		name = strings.TrimSpace(name)
		if name == "" {
			continue // Skip empty names
		}

		if !r.BaseRegistry.Has(name) {
			missingMiddlewares = append(missingMiddlewares, name)
		}
	}

	if len(missingMiddlewares) > 0 {
		return fmt.Errorf("unknown middleware(s): %s", strings.Join(missingMiddlewares, ", "))
	}

	return nil
}

// Get retrieves a middleware by name
func (r *middlewareRegistry) Get(name string) (*models.MiddlewareMetadata, bool) {
	return r.BaseRegistry.Get(name)
}
