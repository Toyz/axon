package templates

import "github.com/toyz/axon/internal/registry"

// createTestParserRegistry creates a test parser registry with built-in parsers
func createTestParserRegistry() *registry.ParserRegistry {
	return registry.NewParserRegistry()
}