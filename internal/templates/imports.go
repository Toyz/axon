package templates

import (
	"fmt"
	"go/format"
	"os"
	"strings"

	"golang.org/x/tools/imports"
)

// GenerateMinimalImports creates the absolute minimal import block
// Only includes imports we know will definitely be needed
func GenerateMinimalImports(moduleImport string) string {
	return fmt.Sprintf(`import (
	"github.com/labstack/echo/v4"
	"go.uber.org/fx"

	"%s/pkg/axon"
)

`, moduleImport)
}

// GenerateMinimalImportsWithPackages creates imports with user project packages
func GenerateMinimalImportsWithPackages(moduleImport string, userPackages []string) string {
	var builder strings.Builder
	
	builder.WriteString("import (\n")
	builder.WriteString("\t\"github.com/labstack/echo/v4\"\n")
	builder.WriteString("\t\"go.uber.org/fx\"\n")
	builder.WriteString("\n")
	
	// Add user project packages
	for _, pkg := range userPackages {
		builder.WriteString(fmt.Sprintf("\t\"%s/%s\"\n", moduleImport, pkg))
	}
	
	builder.WriteString(fmt.Sprintf("\t\"%s/pkg/axon\"\n", moduleImport))
	builder.WriteString(")\n\n")
	
	return builder.String()
}

// FixEchoImports ensures echo imports use the correct v4 version
func FixEchoImports(content string) string {
	// Replace any echo imports that don't specify v4
	return strings.ReplaceAll(content, `"github.com/labstack/echo"`, `"github.com/labstack/echo/v4"`)
}

// PostProcessAllGeneratedFiles runs goimports on all generated files after they're written
// This two-phase approach ensures:
// 1. All files are generated first (so cross-references work)
// 2. Then goimports can properly resolve all imports with full context
func PostProcessAllGeneratedFiles(generatedFilePaths []string) error {
	for _, filePath := range generatedFilePaths {
		if err := processFileImports(filePath); err != nil {
			return fmt.Errorf("failed to process imports for %s: %w", filePath, err)
		}
	}
	return nil
}

// processFileImports runs goimports on a single file
func processFileImports(filePath string) error {
	// Read the generated file
	content, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}
	
	// Process with goimports - the filePath is crucial for context
	formatted, err := imports.Process(filePath, content, &imports.Options{
		Fragment:  false,    // Complete Go file
		AllErrors: true,     // Report all errors
		Comments:  true,     // Preserve comments
		TabIndent: true,     // Use tabs
		TabWidth:  8,        // Standard Go tab width
	})
	if err != nil {
		// Fallback to gofmt if goimports fails
		formatted, fmtErr := format.Source(content)
		if fmtErr != nil {
			return fmt.Errorf("goimports failed: %v, gofmt also failed: %v", err, fmtErr)
		}
		// Write the gofmt result and continue
		return os.WriteFile(filePath, formatted, 0644)
	}
	
	// Write the processed file back
	return os.WriteFile(filePath, formatted, 0644)
}