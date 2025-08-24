package parser

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
	"testing"

	"github.com/toyz/axon/internal/models"
)

// TestExtractDependencies_DirectFunction tests the extractDependencies function directly
// This test verifies that the parser correctly extracts multiple //axon::inject annotations
func TestExtractDependencies_DirectFunction(t *testing.T) {
	// This test reproduces the SessionController scenario where both
	// SessionFactory and UserService should be injected
	source := `
type SessionController struct {
	//axon::inject
	SessionFactory func() *services.SessionService
	//axon::inject
	UserService    *services.UserService
}`

	// Parse the source code to get the AST
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "test.go", "package test\n"+source, parser.ParseComments)
	if err != nil {
		t.Fatalf("failed to parse source: %v", err)
	}

	// Find the struct type
	var structType *ast.StructType
	ast.Inspect(file, func(n ast.Node) bool {
		if genDecl, ok := n.(*ast.GenDecl); ok && genDecl.Tok == token.TYPE {
			for _, spec := range genDecl.Specs {
				if typeSpec, ok := spec.(*ast.TypeSpec); ok {
					if typeSpec.Name.Name == "SessionController" {
						if st, ok := typeSpec.Type.(*ast.StructType); ok {
							structType = st
							return false
						}
					}
				}
			}
		}
		return true
	})

	if structType == nil {
		t.Fatal("could not find SessionController struct type in source")
	}

	// Test the extractDependencies function directly
	p := NewParser()
	dependencies := p.extractDependencies(structType)

	t.Logf("Found %d dependencies:", len(dependencies))
	for i, dep := range dependencies {
		t.Logf("  [%d] Name: %s, Type: %s, IsInit: %v", i, dep.Name, dep.Type, dep.IsInit)
	}

	// Expected: Both SessionFactory and UserService should be found
	expectedDeps := []models.Dependency{
		{
			Name: "SessionFactory",
			Type: "func() *services.SessionService",
			IsInit: false,
		},
		{
			Name: "UserService", 
			Type: "*services.UserService",
			IsInit: false,
		},
	}

	// Verify that the extractDependencies function correctly finds both dependencies
	if len(dependencies) != len(expectedDeps) {
		t.Errorf("Expected %d dependencies, but got %d", len(expectedDeps), len(dependencies))
		
		// Show what we expected vs what we got
		t.Errorf("Expected dependencies:")
		for i, expected := range expectedDeps {
			t.Errorf("  [%d] %s: %s", i, expected.Name, expected.Type)
		}
		t.Errorf("Actual dependencies:")
		for i, actual := range dependencies {
			t.Errorf("  [%d] %s: %s", i, actual.Name, actual.Type)
		}
		return
	}

	// Verify each dependency matches expected
	for i, expected := range expectedDeps {
		if i >= len(dependencies) {
			t.Errorf("Missing dependency at index %d: expected %s", i, expected.Name)
			continue
		}
		
		actual := dependencies[i]
		if actual.Name != expected.Name {
			t.Errorf("Dependency %d name mismatch: expected %s, got %s", i, expected.Name, actual.Name)
		}
		if actual.Type != expected.Type {
			t.Errorf("Dependency %d type mismatch: expected %s, got %s", i, expected.Type, actual.Type)
		}
		if actual.IsInit != expected.IsInit {
			t.Errorf("Dependency %d IsInit mismatch: expected %v, got %v", i, expected.IsInit, actual.IsInit)
		}
	}

	// NOTE: This test passes, which means the parser extractDependencies function works correctly.
	// The actual bug appears to be in the generator, not the parser.
	// The SessionController in examples/complete-app/internal/controllers/autogen_module.go
	// shows that NewSessionController only takes sessionFactory but not userService.
}

// TestExtractDependencies_MultipleInjectVariations tests various scenarios with multiple inject annotations
func TestExtractDependencies_MultipleInjectVariations(t *testing.T) {
	tests := []struct {
		name     string
		source   string
		expected []models.Dependency
	}{
		{
			name: "two inject annotations",
			source: `
type Controller struct {
	//axon::inject
	ServiceA *ServiceA
	//axon::inject  
	ServiceB *ServiceB
}`,
			expected: []models.Dependency{
				{Name: "ServiceA", Type: "*ServiceA", IsInit: false},
				{Name: "ServiceB", Type: "*ServiceB", IsInit: false},
			},
		},
		{
			name: "three inject annotations",
			source: `
type Controller struct {
	//axon::inject
	ServiceA *ServiceA
	//axon::inject
	ServiceB *ServiceB
	//axon::inject
	ServiceC *ServiceC
}`,
			expected: []models.Dependency{
				{Name: "ServiceA", Type: "*ServiceA", IsInit: false},
				{Name: "ServiceB", Type: "*ServiceB", IsInit: false},
				{Name: "ServiceC", Type: "*ServiceC", IsInit: false},
			},
		},
		{
			name: "mixed inject and init annotations",
			source: `
type Controller struct {
	//axon::inject
	ServiceA *ServiceA
	//axon::init
	ServiceB *ServiceB
	//axon::inject
	ServiceC *ServiceC
}`,
			expected: []models.Dependency{
				{Name: "ServiceA", Type: "*ServiceA", IsInit: false},
				{Name: "ServiceB", Type: "*ServiceB", IsInit: true},
				{Name: "ServiceC", Type: "*ServiceC", IsInit: false},
			},
		},
		{
			name: "inject with flags",
			source: `
type Controller struct {
	//axon::inject -Init
	ServiceA *ServiceA
	//axon::inject
	ServiceB *ServiceB
}`,
			expected: []models.Dependency{
				{Name: "ServiceA", Type: "*ServiceA", IsInit: true},
				{Name: "ServiceB", Type: "*ServiceB", IsInit: false},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Parse the source code
			fset := token.NewFileSet()
			file, err := parser.ParseFile(fset, "test.go", "package test\n"+tt.source, parser.ParseComments)
			if err != nil {
				t.Fatalf("failed to parse source: %v", err)
			}

			// Find the struct type
			var structType *ast.StructType
			ast.Inspect(file, func(n ast.Node) bool {
				if genDecl, ok := n.(*ast.GenDecl); ok && genDecl.Tok == token.TYPE {
					for _, spec := range genDecl.Specs {
						if typeSpec, ok := spec.(*ast.TypeSpec); ok {
							if st, ok := typeSpec.Type.(*ast.StructType); ok {
								structType = st
								return false
							}
						}
					}
				}
				return true
			})

			if structType == nil {
				t.Fatal("could not find struct type in source")
			}

			// Test the extractDependencies function
			p := NewParser()
			dependencies := p.extractDependencies(structType)

			// Log what we found for debugging
			t.Logf("Found %d dependencies:", len(dependencies))
			for i, dep := range dependencies {
				t.Logf("  [%d] Name: %s, Type: %s, IsInit: %v", i, dep.Name, dep.Type, dep.IsInit)
			}

			// Check if we got the expected number of dependencies
			if len(dependencies) != len(tt.expected) {
				t.Errorf("Expected %d dependencies, got %d", len(tt.expected), len(dependencies))
				return
			}

			// Verify each dependency
			for i, expected := range tt.expected {
				if i >= len(dependencies) {
					t.Errorf("Missing dependency at index %d: expected %s", i, expected.Name)
					continue
				}
				
				actual := dependencies[i]
				if actual.Name != expected.Name {
					t.Errorf("Dependency %d name mismatch: expected %s, got %s", i, expected.Name, actual.Name)
				}
				if actual.Type != expected.Type {
					t.Errorf("Dependency %d type mismatch: expected %s, got %s", i, expected.Type, actual.Type)
				}
				if actual.IsInit != expected.IsInit {
					t.Errorf("Dependency %d IsInit mismatch: expected %v, got %v", i, expected.IsInit, actual.IsInit)
				}
			}
		})
	}
}

// TestSessionController_RealWorldBug tests the actual SessionController from the examples
// to see if the parser correctly extracts both dependencies
func TestSessionController_RealWorldBug(t *testing.T) {
	// Read and parse just the SessionController file directly
	sessionControllerSource := `package controllers

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/toyz/axon/examples/complete-app/internal/services"
	"github.com/toyz/axon/pkg/axon"
)

// SessionController demonstrates using a Transient service
// Each request gets its own SessionService instance via the factory
//axon::controller
type SessionController struct {
	//axon::inject
	SessionFactory func() *services.SessionService // Inject the factory function for transient service
	//axon::inject
	UserService    *services.UserService          // Regular singleton service
}`

	p := NewParser()
	metadata, err := p.ParseSource("session_controller.go", sessionControllerSource)
	if err != nil {
		t.Fatalf("failed to parse controllers directory: %v", err)
	}

	// Find the SessionController
	var sessionController *models.ControllerMetadata
	for i := range metadata.Controllers {
		if metadata.Controllers[i].Name == "SessionController" {
			sessionController = &metadata.Controllers[i]
			break
		}
	}

	if sessionController == nil {
		t.Fatal("SessionController not found in parsed metadata")
	}

	t.Logf("SessionController found with %d dependencies:", len(sessionController.Dependencies))
	for i, dep := range sessionController.Dependencies {
		t.Logf("  [%d] Name: %s, Type: %s, IsInit: %v", i, dep.Name, dep.Type, dep.IsInit)
	}

	// Expected dependencies based on the SessionController source
	expectedDeps := []models.Dependency{
		{
			Name: "SessionFactory",
			Type: "func() *services.SessionService",
			IsInit: false,
		},
		{
			Name: "UserService", 
			Type: "*services.UserService",
			IsInit: false,
		},
	}

	// Check if both dependencies are found
	if len(sessionController.Dependencies) != len(expectedDeps) {
		t.Errorf("BUG CONFIRMED: SessionController should have %d dependencies but parser found %d", 
			len(expectedDeps), len(sessionController.Dependencies))
		
		t.Errorf("Expected dependencies:")
		for i, expected := range expectedDeps {
			t.Errorf("  [%d] %s: %s", i, expected.Name, expected.Type)
		}
		t.Errorf("Actual dependencies found by parser:")
		for i, actual := range sessionController.Dependencies {
			t.Errorf("  [%d] %s: %s", i, actual.Name, actual.Type)
		}
		
		// This would indicate the bug is in the parser
		t.Errorf("If this test fails, the bug is in the parser's extractDependencies function")
		return
	}

	// Verify each dependency
	for i, expected := range expectedDeps {
		if i >= len(sessionController.Dependencies) {
			t.Errorf("Missing dependency at index %d: expected %s", i, expected.Name)
			continue
		}
		
		actual := sessionController.Dependencies[i]
		if actual.Name != expected.Name {
			t.Errorf("Dependency %d name mismatch: expected %s, got %s", i, expected.Name, actual.Name)
		}
		if actual.Type != expected.Type {
			t.Errorf("Dependency %d type mismatch: expected %s, got %s", i, expected.Type, actual.Type)
		}
		if actual.IsInit != expected.IsInit {
			t.Errorf("Dependency %d IsInit mismatch: expected %v, got %v", i, expected.IsInit, actual.IsInit)
		}
	}

	t.Logf("SUCCESS: Parser correctly extracts both dependencies from SessionController")
	t.Logf("The bug must be in the generator, not the parser")
	t.Logf("Check the generated NewSessionController function in autogen_module.go")
}

// TestExtractDependencies_WithDebugLogging tests the extractDependencies function with debug logging enabled
func TestExtractDependencies_WithDebugLogging(t *testing.T) {
	// Create a mock reporter that captures debug messages
	var debugMessages []string
	mockReporter := &mockDiagnosticReporter{
		debugMessages: &debugMessages,
	}
	
	// Create parser with the mock reporter
	p := NewParserWithReporter(mockReporter)

	// Test source with multiple inject annotations
	source := `
type SessionController struct {
	//axon::inject
	SessionFactory func() *services.SessionService
	//axon::inject
	UserService    *services.UserService
}`

	// Parse the source code to get the AST
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "test.go", "package test\n"+source, parser.ParseComments)
	if err != nil {
		t.Fatalf("failed to parse source: %v", err)
	}

	// Find the struct type
	var structType *ast.StructType
	ast.Inspect(file, func(n ast.Node) bool {
		if genDecl, ok := n.(*ast.GenDecl); ok && genDecl.Tok == token.TYPE {
			for _, spec := range genDecl.Specs {
				if typeSpec, ok := spec.(*ast.TypeSpec); ok {
					if typeSpec.Name.Name == "SessionController" {
						if st, ok := typeSpec.Type.(*ast.StructType); ok {
							structType = st
							return false
						}
					}
				}
			}
		}
		return true
	})

	if structType == nil {
		t.Fatal("could not find SessionController struct type in source")
	}

	// Test the extractDependencies function with debug logging
	dependencies := p.extractDependencies(structType)

	// Verify that debug messages were generated
	if len(debugMessages) == 0 {
		t.Error("Expected debug messages to be generated, but none were found")
	}

	// Log the debug messages for verification
	t.Logf("Debug messages generated (%d):", len(debugMessages))
	for i, msg := range debugMessages {
		t.Logf("  [%d] %s", i, msg)
	}

	// Verify that both dependencies were found
	if len(dependencies) != 2 {
		t.Errorf("Expected 2 dependencies, got %d", len(dependencies))
	}

	// Verify that specific debug messages were generated
	expectedMessages := []string{
		"Starting dependency extraction",
		"Processing individual field annotations",
		"Found //axon::inject annotation",
		"Added inject dependency",
	}

	for _, expected := range expectedMessages {
		found := false
		for _, msg := range debugMessages {
			if strings.Contains(msg, expected) {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected debug message containing '%s' was not found", expected)
		}
	}
}

// mockDiagnosticReporter is a mock implementation for testing debug logging
type mockDiagnosticReporter struct {
	debugMessages *[]string
}

func (m *mockDiagnosticReporter) Debug(format string, args ...interface{}) {
	message := fmt.Sprintf(format, args...)
	*m.debugMessages = append(*m.debugMessages, message)
}

func (m *mockDiagnosticReporter) DebugSection(section string) {
	message := fmt.Sprintf("=== %s ===", section)
	*m.debugMessages = append(*m.debugMessages, message)
}