package registry

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/toyz/axon/internal/models"
)

func TestParserRegistry_RegisterParser(t *testing.T) {
	registry := NewParserRegistry()

	parser := models.RouteParserMetadata{
		TypeName:     "UUID",
		FunctionName: "ParseUUID",
		PackagePath:  "/test/parsers",
		FileName:     "uuid_parser.go",
		Line:         10,
	}

	// Test successful registration
	err := registry.RegisterParser(parser)
	assert.NoError(t, err)

	// Test duplicate registration
	duplicate := models.RouteParserMetadata{
		TypeName:     "UUID",
		FunctionName: "ParseUUID2",
		PackagePath:  "/test/parsers2",
		FileName:     "uuid_parser2.go",
		Line:         20,
	}

	err = registry.RegisterParser(duplicate)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Parser for type 'UUID' already registered")
	assert.Contains(t, err.Error(), "uuid_parser2.go:20")
}

func TestParserRegistry_GetParser(t *testing.T) {
	registry := NewParserRegistry()

	parser := models.RouteParserMetadata{
		TypeName:     "CustomType",
		FunctionName: "ParseCustomType",
		PackagePath:  "/test/parsers",
	}

	// Test getting non-existent parser
	_, exists := registry.GetParser("CustomType")
	assert.False(t, exists)

	// Register parser
	err := registry.RegisterParser(parser)
	assert.NoError(t, err)

	// Test getting existing parser
	retrieved, exists := registry.GetParser("CustomType")
	assert.True(t, exists)
	assert.Equal(t, parser.TypeName, retrieved.TypeName)
	assert.Equal(t, parser.FunctionName, retrieved.FunctionName)
	assert.Equal(t, parser.PackagePath, retrieved.PackagePath)
	
	// Test getting built-in parser
	builtinParser, exists := registry.GetParser("int")
	assert.True(t, exists)
	assert.Equal(t, "int", builtinParser.TypeName)
	
	// Test getting parser by alias
	uuidParser, exists := registry.GetParser("UUID")
	assert.True(t, exists)
	assert.Equal(t, "uuid.UUID", uuidParser.TypeName)
}

func TestParserRegistry_ListParsers(t *testing.T) {
	registry := NewParserRegistry()

	// Test registry with built-ins
	parsers := registry.ListParsers()
	assert.NotEmpty(t, parsers)
	assert.Contains(t, parsers, "int")
	assert.Contains(t, parsers, "string")
	assert.Contains(t, parsers, "uuid.UUID")
	assert.Contains(t, parsers, "float64")
	assert.Contains(t, parsers, "float32")

	// Add custom parsers
	parser1 := models.RouteParserMetadata{
		TypeName:     "CustomUUID",
		FunctionName: "ParseCustomUUID",
	}
	parser2 := models.RouteParserMetadata{
		TypeName:     "CompositeID",
		FunctionName: "ParseCompositeID",
	}

	err := registry.RegisterParser(parser1)
	assert.NoError(t, err)
	err = registry.RegisterParser(parser2)
	assert.NoError(t, err)

	// Test listing parsers includes both built-ins and custom
	parsers = registry.ListParsers()
	assert.Contains(t, parsers, "int") // built-in
	assert.Contains(t, parsers, "CustomUUID") // custom
	assert.Contains(t, parsers, "CompositeID") // custom
}

func TestParserRegistry_HasParser(t *testing.T) {
	registry := NewParserRegistry()

	// Test non-existent parser
	assert.False(t, registry.HasParser("UUID"))

	// Register parser
	parser := models.RouteParserMetadata{
		TypeName:     "UUID",
		FunctionName: "ParseUUID",
	}
	err := registry.RegisterParser(parser)
	assert.NoError(t, err)

	// Test existing parser
	assert.True(t, registry.HasParser("UUID"))
	assert.False(t, registry.HasParser("NonExistent"))
}

func TestParserRegistry_Clear(t *testing.T) {
	registry := NewParserRegistry()

	// Add custom parser
	parser := models.RouteParserMetadata{
		TypeName:     "CustomType",
		FunctionName: "ParseCustomType",
	}
	err := registry.RegisterParser(parser)
	assert.NoError(t, err)

	assert.True(t, registry.HasParser("CustomType"))
	assert.True(t, registry.HasParser("int")) // built-in should exist

	// Clear registry - should preserve built-ins
	registry.Clear()

	assert.False(t, registry.HasParser("CustomType"))
	assert.True(t, registry.HasParser("int")) // built-ins are preserved
	assert.NotEmpty(t, registry.ListParsers()) // built-ins remain
}

func TestParserRegistry_ClearCustomParsers(t *testing.T) {
	registry := NewParserRegistry()

	// Register a custom parser
	customParser := models.RouteParserMetadata{
		TypeName:     "CustomType",
		FunctionName: "ParseCustomType",
		PackagePath:  "test",
		FileName:     "test.go",
		Line:         10,
	}
	err := registry.RegisterParser(customParser)
	assert.NoError(t, err)

	assert.True(t, registry.HasParser("CustomType"))
	assert.True(t, registry.HasParser("int")) // built-in should exist

	// Clear only custom parsers
	registry.ClearCustomParsers()

	assert.False(t, registry.HasParser("CustomType"))
	assert.True(t, registry.HasParser("int")) // built-ins are preserved
	assert.NotEmpty(t, registry.ListParsers()) // built-ins remain
}

func TestParserRegistry_GetAllParsers(t *testing.T) {
	registry := NewParserRegistry()

	// Test registry with built-ins
	all := registry.GetAllParsers()
	assert.NotEmpty(t, all)
	assert.Contains(t, all, "int")
	assert.Contains(t, all, "string")
	assert.Contains(t, all, "uuid.UUID")

	// Add custom parsers
	parser1 := models.RouteParserMetadata{
		TypeName:     "CustomUUID",
		FunctionName: "ParseCustomUUID",
	}
	parser2 := models.RouteParserMetadata{
		TypeName:     "CompositeID",
		FunctionName: "ParseCompositeID",
	}

	err := registry.RegisterParser(parser1)
	assert.NoError(t, err)
	err = registry.RegisterParser(parser2)
	assert.NoError(t, err)

	// Test getting all parsers includes both built-ins and custom
	all = registry.GetAllParsers()
	assert.Contains(t, all, "int") // built-in
	assert.Equal(t, parser1, all["CustomUUID"])
	assert.Equal(t, parser2, all["CompositeID"])

	// Verify it's a copy (modifying returned map doesn't affect registry)
	delete(all, "CustomUUID")
	assert.True(t, registry.HasParser("CustomUUID"))
}

func TestParserRegistry_ThreadSafety(t *testing.T) {
	registry := NewParserRegistry()

	// Test concurrent access
	done := make(chan bool, 2)

	// Goroutine 1: Register parsers
	go func() {
		for i := 0; i < 100; i++ {
			parser := models.RouteParserMetadata{
				TypeName:     fmt.Sprintf("Type%d", i),
				FunctionName: fmt.Sprintf("ParseType%d", i),
			}
			registry.RegisterParser(parser)
		}
		done <- true
	}()

	// Goroutine 2: Read parsers
	go func() {
		for i := 0; i < 100; i++ {
			registry.ListParsers()
			registry.HasParser(fmt.Sprintf("Type%d", i))
		}
		done <- true
	}()

	// Wait for both goroutines
	<-done
	<-done

	// Verify final state (100 custom + 5 built-ins)
	parsers := registry.ListParsers()
	assert.Len(t, parsers, 105) // 100 custom + 5 built-ins (int, string, float64, float32, uuid.UUID)
}