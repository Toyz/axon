package annotations

import (
	"fmt"
	"sync"
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
	mu      sync.RWMutex                        // Protects concurrent access
	schemas map[AnnotationType]AnnotationSchema // Schema storage
}

// NewRegistry creates a new annotation registry
func NewRegistry() AnnotationRegistry {
	return &registry{
		schemas: make(map[AnnotationType]AnnotationSchema),
	}
}

// defaultRegistry is the global registry instance
var (
	defaultRegistry     AnnotationRegistry
	defaultRegistryOnce sync.Once
)

// DefaultRegistry returns the global annotation registry
func DefaultRegistry() AnnotationRegistry {
	defaultRegistryOnce.Do(func() {
		defaultRegistry = NewRegistry()
	})
	return defaultRegistry
}

// Register adds a new annotation type with its schema to the registry
func (r *registry) Register(annotationType AnnotationType, schema AnnotationSchema) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Validate that the schema type matches the annotation type
	if schema.Type != annotationType {
		return fmt.Errorf("schema type %s does not match annotation type %s",
			schema.Type.String(), annotationType.String())
	}

	// Check if already registered
	if _, exists := r.schemas[annotationType]; exists {
		return fmt.Errorf("annotation type %s is already registered", annotationType.String())
	}

	// Validate schema parameters
	if err := r.validateSchema(schema); err != nil {
		return fmt.Errorf("invalid schema for %s: %w", annotationType.String(), err)
	}

	r.schemas[annotationType] = schema
	return nil
}

// GetSchema retrieves the schema for an annotation type
func (r *registry) GetSchema(annotationType AnnotationType) (AnnotationSchema, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	schema, exists := r.schemas[annotationType]
	if !exists {
		return AnnotationSchema{}, fmt.Errorf("annotation type %s is not registered", annotationType.String())
	}

	return schema, nil
}

// ListTypes returns all registered annotation types
func (r *registry) ListTypes() []AnnotationType {
	r.mu.RLock()
	defer r.mu.RUnlock()

	types := make([]AnnotationType, 0, len(r.schemas))
	for annotationType := range r.schemas {
		types = append(types, annotationType)
	}

	return types
}

// IsRegistered checks if an annotation type is registered
func (r *registry) IsRegistered(annotationType AnnotationType) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	_, exists := r.schemas[annotationType]
	return exists
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
