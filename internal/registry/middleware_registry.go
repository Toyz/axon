package registry

import (
	"fmt"
	"strings"

	"github.com/toyz/axon/internal/models"
)

// middlewareRegistry implements the MiddlewareRegistry interface
type middlewareRegistry struct {
	middlewares map[string]*models.MiddlewareMetadata
}

// NewMiddlewareRegistry creates a new middleware registry
func NewMiddlewareRegistry() MiddlewareRegistry {
	return &middlewareRegistry{
		middlewares: make(map[string]*models.MiddlewareMetadata),
	}
}

// Register adds a middleware to the registry
func (r *middlewareRegistry) Register(name string, middleware *models.MiddlewareMetadata) error {
	if name == "" {
		return fmt.Errorf("middleware name cannot be empty")
	}

	if middleware == nil {
		return fmt.Errorf("middleware metadata cannot be nil")
	}

	// Check for duplicate names
	if existing, exists := r.middlewares[name]; exists {
		return fmt.Errorf("middleware '%s' is already registered in package '%s'", name, existing.PackagePath)
	}

	r.middlewares[name] = middleware
	return nil
}

// Validate checks that all middleware names exist in the registry
func (r *middlewareRegistry) Validate(middlewareNames []string) error {
	var missingMiddlewares []string

	for _, name := range middlewareNames {
		name = strings.TrimSpace(name)
		if name == "" {
			continue // Skip empty names
		}

		if _, exists := r.middlewares[name]; !exists {
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
	middleware, exists := r.middlewares[name]
	return middleware, exists
}
