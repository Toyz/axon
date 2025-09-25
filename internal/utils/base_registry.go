package utils

import (
	"fmt"
	"sync"
)

// RegistryValidator is a function that validates a key-value pair before registration
type RegistryValidator[K comparable, V any] func(key K, value V, existing map[K]V) error

// BaseRegistry provides a generic, thread-safe registry implementation
// with built-in validation support that can be extended by specific registry types
type BaseRegistry[K comparable, V any] struct {
	mu              sync.RWMutex
	items           map[K]V
	validator       RegistryValidator[K, V]
	registryName    string
	keyDescriptor   string // e.g., "middleware name", "parser type", etc.
	valueDescriptor string // e.g., "metadata", "parser", etc.
}

// NewBaseRegistry creates a new base registry with the specified configuration
func NewBaseRegistry[K comparable, V any](registryName, keyDesc, valueDesc string) *BaseRegistry[K, V] {
	return &BaseRegistry[K, V]{
		items:           make(map[K]V),
		registryName:    registryName,
		keyDescriptor:   keyDesc,
		valueDescriptor: valueDesc,
	}
}

// SetValidator sets the validation function for this registry
func (r *BaseRegistry[K, V]) SetValidator(validator RegistryValidator[K, V]) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.validator = validator
}

// Register adds an item to the registry with validation
func (r *BaseRegistry[K, V]) Register(key K, value V) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Run validation if configured
	if r.validator != nil {
		if err := r.validator(key, value, r.items); err != nil {
			return fmt.Errorf("%s registry: %w", r.registryName, err)
		}
	}

	r.items[key] = value
	return nil
}

// RegisterWithCustomValidator registers an item with a one-time custom validator
func (r *BaseRegistry[K, V]) RegisterWithCustomValidator(key K, value V, customValidator RegistryValidator[K, V]) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Run custom validation
	if customValidator != nil {
		if err := customValidator(key, value, r.items); err != nil {
			return fmt.Errorf("%s registry: %w", r.registryName, err)
		}
	}

	// Also run default validation if configured
	if r.validator != nil {
		if err := r.validator(key, value, r.items); err != nil {
			return fmt.Errorf("%s registry: %w", r.registryName, err)
		}
	}

	r.items[key] = value
	return nil
}

// Get retrieves an item from the registry
func (r *BaseRegistry[K, V]) Get(key K) (V, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	value, exists := r.items[key]
	return value, exists
}

// GetOrError retrieves an item or returns an error if not found
func (r *BaseRegistry[K, V]) GetOrError(key K) (V, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	value, exists := r.items[key]
	if !exists {
		var zero V
		return zero, fmt.Errorf("%s '%v' is not registered", r.keyDescriptor, key)
	}
	return value, nil
}

// Has checks if a key exists in the registry
func (r *BaseRegistry[K, V]) Has(key K) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	_, exists := r.items[key]
	return exists
}

// List returns all keys in the registry
func (r *BaseRegistry[K, V]) List() []K {
	r.mu.RLock()
	defer r.mu.RUnlock()

	keys := make([]K, 0, len(r.items))
	for key := range r.items {
		keys = append(keys, key)
	}
	return keys
}

// GetAll returns a copy of all items in the registry
func (r *BaseRegistry[K, V]) GetAll() map[K]V {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make(map[K]V, len(r.items))
	for k, v := range r.items {
		result[k] = v
	}
	return result
}

// Clear removes all items from the registry
func (r *BaseRegistry[K, V]) Clear() {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.items = make(map[K]V)
}

// ClearWithReset removes all items and resets with initial items
func (r *BaseRegistry[K, V]) ClearWithReset(initialItems map[K]V) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.items = make(map[K]V)
	if initialItems != nil {
		for k, v := range initialItems {
			r.items[k] = v
		}
	}
}

// Size returns the number of items in the registry
func (r *BaseRegistry[K, V]) Size() int {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return len(r.items)
}

// ForEach applies a function to each item in the registry
func (r *BaseRegistry[K, V]) ForEach(fn func(K, V)) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for k, v := range r.items {
		fn(k, v)
	}
}

// Filter returns items that match the given predicate
func (r *BaseRegistry[K, V]) Filter(predicate func(K, V) bool) map[K]V {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make(map[K]V)
	for k, v := range r.items {
		if predicate(k, v) {
			result[k] = v
		}
	}
	return result
}

// Update updates an existing item in the registry
func (r *BaseRegistry[K, V]) Update(key K, value V) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.items[key]; !exists {
		return fmt.Errorf("%s '%v' does not exist", r.keyDescriptor, key)
	}

	r.items[key] = value
	return nil
}

// Delete removes an item from the registry
func (r *BaseRegistry[K, V]) Delete(key K) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.items[key]; exists {
		delete(r.items, key)
		return true
	}
	return false
}

// Common validators for reuse across different registry types

// NotEmptyKeyValidator validates that a string key is not empty
func NotEmptyKeyValidator[V any](keyDesc string) RegistryValidator[string, V] {
	return func(key string, value V, existing map[string]V) error {
		if key == "" {
			return fmt.Errorf("%s cannot be empty", keyDesc)
		}
		return nil
	}
}

// NotNilValueValidator validates that a pointer value is not nil
func NotNilValueValidator[K comparable, V any](valueDesc string) RegistryValidator[K, *V] {
	return func(key K, value *V, existing map[K]*V) error {
		if value == nil {
			return fmt.Errorf("%s cannot be nil", valueDesc)
		}
		return nil
	}
}

// NoDuplicateValidator validates that a key doesn't already exist
func NoDuplicateValidator[K comparable, V any](keyDesc string) RegistryValidator[K, V] {
	return func(key K, value V, existing map[K]V) error {
		if _, exists := existing[key]; exists {
			return fmt.Errorf("%s '%v' is already registered", keyDesc, key)
		}
		return nil
	}
}

// ChainValidators combines multiple validators into one
func ChainValidators[K comparable, V any](validators ...RegistryValidator[K, V]) RegistryValidator[K, V] {
	return func(key K, value V, existing map[K]V) error {
		for _, validator := range validators {
			if validator != nil {
				if err := validator(key, value, existing); err != nil {
					return err
				}
			}
		}
		return nil
	}
}

// Registry is an alias for BaseRegistry to maintain backward compatibility
type Registry[K comparable, V any] = BaseRegistry[K, V]

// NewRegistry creates a new registry (backward compatibility)
func NewRegistry[K comparable, V any]() *Registry[K, V] {
	return NewBaseRegistry[K, V]("registry", "key", "value")
}

// RegisterWithValidator is a backward compatibility method
func (r *BaseRegistry[K, V]) RegisterWithValidator(key K, value V, validator func(K, V, map[K]V) error) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Run the provided validator
	if validator != nil {
		if err := validator(key, value, r.items); err != nil {
			return err
		}
	}

	// Also run the registry's built-in validator if set
	if r.validator != nil {
		if err := r.validator(key, value, r.items); err != nil {
			return fmt.Errorf("%s registry: %w", r.registryName, err)
		}
	}

	r.items[key] = value
	return nil
}