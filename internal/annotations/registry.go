package annotations

import (
	"fmt"

	"github.com/toyz/axon/internal/utils"
)

// AnnotationRegistry defines the interface for managing annotation schemas
type AnnotationRegistry interface {
	// Register a new annotation type with its schema
	Register(annotationType AnnotationType, schema AnnotationSchema) error

	// GetSchema retrieves the schema for an annotation type
	GetSchema(annotationType AnnotationType) (AnnotationSchema, error)

	// ListTypes returns all registered annotation types
	ListTypes() []AnnotationType

	// IsRegistered checks if an annotation type is registered
	IsRegistered(annotationType AnnotationType) bool
}

// registry is the concrete implementation of AnnotationRegistry
type registry struct {
	*utils.Registry[AnnotationType, AnnotationSchema]
}

// NewRegistry creates a new annotation registry
func NewRegistry() AnnotationRegistry {
	return &registry{
		Registry: utils.NewRegistry[AnnotationType, AnnotationSchema](),
	}
}

// Register adds a new annotation type with its schema to the registry
func (r *registry) Register(annotationType AnnotationType, schema AnnotationSchema) error {
	validator := func(key AnnotationType, value AnnotationSchema, existing map[AnnotationType]AnnotationSchema) error {
		// Validate that the schema type matches the annotation type
		if value.Type != key {
			return fmt.Errorf("schema type %s does not match annotation type %s",
				value.Type.String(), key.String())
		}

		// Check if already registered
		if _, exists := existing[key]; exists {
			return fmt.Errorf("annotation type %s is already registered", key.String())
		}

		// Validate schema parameters
		if err := r.validateSchema(value); err != nil {
			return fmt.Errorf("invalid schema for %s: %w", key.String(), err)
		}

		return nil
	}

	return r.Registry.RegisterWithValidator(annotationType, schema, validator)
}

// GetSchema retrieves the schema for an annotation type
func (r *registry) GetSchema(annotationType AnnotationType) (AnnotationSchema, error) {
	schema, exists := r.Registry.Get(annotationType)
	if !exists {
		return AnnotationSchema{}, fmt.Errorf("annotation type %s is not registered", annotationType.String())
	}

	return schema, nil
}

// ListTypes returns all registered annotation types
func (r *registry) ListTypes() []AnnotationType {
	return r.Registry.List()
}

// IsRegistered checks if an annotation type is registered
func (r *registry) IsRegistered(annotationType AnnotationType) bool {
	return r.Registry.Has(annotationType)
}

// validateSchema performs basic validation on a schema
func (r *registry) validateSchema(schema AnnotationSchema) error {
	// Validate that parameter names are not empty
	for paramName, paramSpec := range schema.Parameters {
		if paramName == "" {
			return fmt.Errorf("parameter name cannot be empty")
		}

		// Validate parameter type
		if paramSpec.Type < StringType || paramSpec.Type > StringSliceType {
			return fmt.Errorf("invalid parameter type for %s: %d", paramName, paramSpec.Type)
		}

		// Validate default value type matches parameter type
		if paramSpec.DefaultValue != nil {
			if err := r.validateDefaultValue(paramName, paramSpec.Type, paramSpec.DefaultValue); err != nil {
				return err
			}
		}
	}

	return nil
}

// validateDefaultValue checks if the default value matches the parameter type
func (r *registry) validateDefaultValue(paramName string, paramType ParameterType, defaultValue interface{}) error {
	switch paramType {
	case StringType:
		if _, ok := defaultValue.(string); !ok {
			return fmt.Errorf("default value for string parameter %s must be string, got %T", paramName, defaultValue)
		}
	case BoolType:
		if _, ok := defaultValue.(bool); !ok {
			return fmt.Errorf("default value for bool parameter %s must be bool, got %T", paramName, defaultValue)
		}
	case IntType:
		if _, ok := defaultValue.(int); !ok {
			return fmt.Errorf("default value for int parameter %s must be int, got %T", paramName, defaultValue)
		}
	case StringSliceType:
		if _, ok := defaultValue.([]string); !ok {
			return fmt.Errorf("default value for []string parameter %s must be []string, got %T", paramName, defaultValue)
		}
	default:
		return fmt.Errorf("unknown parameter type for %s: %d", paramName, paramType)
	}

	return nil
}
