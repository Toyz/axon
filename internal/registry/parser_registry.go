package registry

import (
	"sync"

	"github.com/toyz/axon/internal/models"
	"github.com/toyz/axon/pkg/axon"
)

// ParserRegistryInterface defines the interface for parser registry operations
type ParserRegistryInterface interface {
	RegisterParser(parser models.RouteParserMetadata) error
	GetParser(typeName string) (models.RouteParserMetadata, bool)
	ListParsers() []string
	HasParser(typeName string) bool
	Clear()
	ClearCustomParsers()
	GetAllParsers() map[string]models.RouteParserMetadata
}

// ParserRegistry manages route parameter parsers
type ParserRegistry struct {
	parsers map[string]models.RouteParserMetadata
	mu      sync.RWMutex
}

// NewParserRegistry creates a new parser registry with built-in parsers
func NewParserRegistry() *ParserRegistry {
	registry := &ParserRegistry{
		parsers: make(map[string]models.RouteParserMetadata),
	}

	// Register built-in parsers from public API
	for _, parser := range axon.BuiltinParsers {
		registry.parsers[parser.TypeName] = parser
	}

	return registry
}

// RegisterParser registers a new parser for a type
func (r *ParserRegistry) RegisterParser(parser models.RouteParserMetadata) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if existing, exists := r.parsers[parser.TypeName]; exists {
		return models.NewParserRegistrationError(
			parser.TypeName,
			parser.FileName,
			parser.Line,
			existing.FileName,
			existing.Line,
		)
	}

	r.parsers[parser.TypeName] = parser
	return nil
}

// GetParser retrieves a parser for a type, resolving aliases
func (r *ParserRegistry) GetParser(typeName string) (models.RouteParserMetadata, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// First try direct lookup
	if parser, exists := r.parsers[typeName]; exists {
		return parser, true
	}

	// Try resolving alias and lookup again
	resolvedType := axon.ResolveTypeAlias(typeName)
	if resolvedType != typeName {
		if parser, exists := r.parsers[resolvedType]; exists {
			return parser, true
		}
	}

	return models.RouteParserMetadata{}, false
}

// ListParsers returns all registered parser type names
func (r *ParserRegistry) ListParsers() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	types := make([]string, 0, len(r.parsers))
	for typeName := range r.parsers {
		types = append(types, typeName)
	}
	return types
}

// HasParser checks if a parser is registered for a type
func (r *ParserRegistry) HasParser(typeName string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	_, exists := r.parsers[typeName]
	return exists
}

// Clear removes all registered parsers
func (r *ParserRegistry) Clear() {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.parsers = make(map[string]models.RouteParserMetadata)

	// Re-register built-in parsers
	for _, parser := range axon.BuiltinParsers {
		r.parsers[parser.TypeName] = parser
	}
}

// ClearCustomParsers removes only custom parsers, keeping built-in parsers
func (r *ParserRegistry) ClearCustomParsers() {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Keep only built-in parsers
	builtinParsers := make(map[string]models.RouteParserMetadata)
	for _, parser := range axon.BuiltinParsers {
		if existing, exists := r.parsers[parser.TypeName]; exists {
			builtinParsers[parser.TypeName] = existing
		}
	}

	r.parsers = builtinParsers
}

// GetAllParsers returns a copy of all registered parsers
func (r *ParserRegistry) GetAllParsers() map[string]models.RouteParserMetadata {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make(map[string]models.RouteParserMetadata, len(r.parsers))
	for k, v := range r.parsers {
		result[k] = v
	}
	return result
}
