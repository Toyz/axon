package registry

import (
	"github.com/toyz/axon/internal/models"
	"github.com/toyz/axon/internal/utils"
	"github.com/toyz/axon/pkg/axon"
)

// ParserRegistry manages route parameter parsers
type ParserRegistry struct {
	*utils.Registry[string, axon.RouteParserMetadata]
}

// NewParserRegistry creates a new parser registry with built-in parsers
func NewParserRegistry() *ParserRegistry {
	registry := &ParserRegistry{
		Registry: utils.NewRegistry[string, axon.RouteParserMetadata](),
	}

	// Register built-in parsers from public API
	builtinParsers := make(map[string]axon.RouteParserMetadata)
	for _, parser := range axon.BuiltinParsers {
		builtinParsers[parser.TypeName] = parser
	}
	registry.Registry.ClearWithReset(builtinParsers)

	return registry
}

// RegisterParser registers a new parser for a type
func (r *ParserRegistry) RegisterParser(parser axon.RouteParserMetadata) error {
	validator := func(key string, value axon.RouteParserMetadata, existing map[string]axon.RouteParserMetadata) error {
		if existingParser, exists := existing[key]; exists {
			return models.NewParserRegistrationError(
				key,
				value.FileName,
				value.Line,
				existingParser.FileName,
				existingParser.Line,
			)
		}
		return nil
	}

	return r.Registry.RegisterWithValidator(parser.TypeName, parser, validator)
}

// GetParser retrieves a parser for a type, resolving aliases
func (r *ParserRegistry) GetParser(typeName string) (axon.RouteParserMetadata, bool) {
	// First try direct lookup
	if parser, exists := r.Registry.Get(typeName); exists {
		return parser, true
	}

	// Try resolving alias and lookup again
	resolvedType := axon.ResolveTypeAlias(typeName)
	if resolvedType != typeName {
		if parser, exists := r.Registry.Get(resolvedType); exists {
			return parser, true
		}
	}

	return axon.RouteParserMetadata{}, false
}

// ListParsers returns all registered parser type names
func (r *ParserRegistry) ListParsers() []string {
	return r.Registry.List()
}

// HasParser checks if a parser is registered for a type
func (r *ParserRegistry) HasParser(typeName string) bool {
	return r.Registry.Has(typeName)
}

// Clear removes all registered parsers
func (r *ParserRegistry) Clear() {
	// Re-register built-in parsers
	builtinParsers := make(map[string]axon.RouteParserMetadata)
	for _, parser := range axon.BuiltinParsers {
		builtinParsers[parser.TypeName] = parser
	}
	r.Registry.ClearWithReset(builtinParsers)
}

// ClearCustomParsers removes only custom parsers, keeping built-in parsers
func (r *ParserRegistry) ClearCustomParsers() {
	// Keep only built-in parsers
	builtinParsers := make(map[string]axon.RouteParserMetadata)
	for _, parser := range axon.BuiltinParsers {
		if existing, exists := r.Registry.Get(parser.TypeName); exists {
			builtinParsers[parser.TypeName] = existing
		}
	}

	r.Registry.ClearWithReset(builtinParsers)
}

// GetAllParsers returns a copy of all registered parsers
func (r *ParserRegistry) GetAllParsers() map[string]axon.RouteParserMetadata {
	return r.Registry.GetAll()
}
