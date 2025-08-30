package templates

import (
	"fmt"
	"strings"
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
