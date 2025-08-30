package utils

import (
	"sync"
)

// Registry provides a generic, thread-safe registry implementation
// that can be used as a base for specific registry types.
type Registry[K comparable, V any] struct {
	mu    sync.RWMutex
	items map[K]V
}

// NewRegistry creates a new generic registry
func NewRegistry[K comparable, V any]() *Registry[K, V] {
	return &Registry[K, V]{
		items: make(map[K]V),
	}
}

// Register adds an item to the registry
func (r *Registry[K, V]) Register(key K, value V) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.items[key] = value
	return nil
}

// RegisterWithValidator adds an item to the registry with custom validation
func (r *Registry[K, V]) RegisterWithValidator(key K, value V, validator func(K, V, map[K]V) error) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if validator != nil {
		if err := validator(key, value, r.items); err != nil {
			return err
		}
	}

	r.items[key] = value
	return nil
}

// Get retrieves an item from the registry
func (r *Registry[K, V]) Get(key K) (V, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	value, exists := r.items[key]
	return value, exists
}

// Has checks if a key exists in the registry
func (r *Registry[K, V]) Has(key K) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	_, exists := r.items[key]
	return exists
}

// List returns all keys in the registry
func (r *Registry[K, V]) List() []K {
	r.mu.RLock()
	defer r.mu.RUnlock()

	keys := make([]K, 0, len(r.items))
	for key := range r.items {
		keys = append(keys, key)
	}
	return keys
}

// GetAll returns a copy of all items in the registry
func (r *Registry[K, V]) GetAll() map[K]V {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make(map[K]V, len(r.items))
	for k, v := range r.items {
		result[k] = v
	}
	return result
}

// Clear removes all items from the registry
func (r *Registry[K, V]) Clear() {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.items = make(map[K]V)
}

// ClearWithReset removes all items and optionally resets with initial items
func (r *Registry[K, V]) ClearWithReset(initialItems map[K]V) {
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
func (r *Registry[K, V]) Size() int {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return len(r.items)
}

// ForEach applies a function to each item in the registry
func (r *Registry[K, V]) ForEach(fn func(K, V)) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for k, v := range r.items {
		fn(k, v)
	}
}

// Filter returns items that match the given predicate
func (r *Registry[K, V]) Filter(predicate func(K, V) bool) map[K]V {
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