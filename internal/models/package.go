package models

import "github.com/toyz/axon/pkg/axon"

// Import represents a Go import statement
type Import struct {
	Path  string // import path (e.g., "context", "github.com/user/repo/pkg")
	Alias string // import alias (empty if no alias)
}

// PackageMetadata represents all annotations found in a package
type PackageMetadata struct {
	PackageName       string                     // name of the Go package
	PackagePath       string                     // file system path to the package
	Controllers       []ControllerMetadata       // all controllers found in the package
	Middlewares       []MiddlewareMetadata       // all middlewares found in the package
	CoreServices      []CoreServiceMetadata      // all core services found in the package
	Interfaces        []InterfaceMetadata        // all interfaces to be generated
	Loggers           []LoggerMetadata           // all loggers found in the package
	RouteParsers      []axon.RouteParserMetadata // all route parsers found in the package
	SourceImports     map[string][]Import        // imports from each source file (filename -> imports)
	ModulePath        string                     // go module path from go.mod
	ModuleRoot        string                     // filesystem path to module root
	PackageImportPath string                     // full import path for this package
}

// ModuleReference represents a reference to a generated module
type ModuleReference struct {
	PackageName string // name of the package
	PackagePath string // import path for the package
	ModuleName  string // name of the module variable
}
