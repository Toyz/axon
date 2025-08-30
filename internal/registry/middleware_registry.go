package registry

import (
	"fmt"
	"strings"

	"github.com/toyz/axon/internal/models"
	"github.com/toyz/axon/internal/utils"
)

// middlewareRegistry implements the MiddlewareRegistry interface
type middlewareRegistry struct {
	*utils.Registry[string, *models.MiddlewareMetadata]
}

// NewMiddlewareRegistry creates a new middleware registry
func NewMiddlewareRegistry() MiddlewareRegistry {
	return &middlewareRegistry{
		Registry: utils.NewRegistry[string, *models.MiddlewareMetadata](),
	}
}

// Register adds a middleware to the registry
func (r *middlewareRegistry) Register(name string, middleware *models.MiddlewareMetadata) error {
	validator := func(key string, value *models.MiddlewareMetadata, existing map[string]*models.MiddlewareMetadata) error {
		// Validate key
		if err := utils.NotEmpty("name")(key); err != nil {
			return fmt.Errorf("middleware %w", err)
		}

		// Validate value
		if err := utils.NotNil[models.MiddlewareMetadata]("metadata")(value); err != nil {
			return fmt.Errorf("middleware %w", err)
		}

		// Check for duplicate names
		if existingMiddleware, exists := existing[key]; exists {
			return fmt.Errorf("middleware '%s' is already registered in package '%s'", key, existingMiddleware.PackagePath)
		}

		return nil
	}

	return r.Registry.RegisterWithValidator(name, middleware, validator)
}

// Validate checks that all middleware names exist in the registry
func (r *middlewareRegistry) Validate(middlewareNames []string) error {
	var missingMiddlewares []string

	for _, name := range middlewareNames {
		name = strings.TrimSpace(name)
		if name == "" {
			continue // Skip empty names
		}

		if !r.Registry.Has(name) {
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
	return r.Registry.Get(name)
}
