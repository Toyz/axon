package registry

import "github.com/toyz/axon/internal/models"

// MiddlewareRegistry defines the interface for tracking and validating middleware components across packages
type MiddlewareRegistry interface {
	Register(name string, middleware *models.MiddlewareMetadata) error
	Validate(middlewareNames []string) error
	Get(name string) (*models.MiddlewareMetadata, bool)
}

