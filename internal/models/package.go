package models

// PackageMetadata represents all annotations found in a package
type PackageMetadata struct {
	PackageName   string                  // name of the Go package
	PackagePath   string                  // file system path to the package
	Controllers   []ControllerMetadata    // all controllers found in the package
	Middlewares   []MiddlewareMetadata    // all middlewares found in the package
	CoreServices  []CoreServiceMetadata   // all core services found in the package
	Interfaces    []InterfaceMetadata     // all interfaces to be generated
	Loggers       []LoggerMetadata        // all loggers found in the package
}

// ModuleReference represents a reference to a generated module
type ModuleReference struct {
	PackageName string // name of the package
	PackagePath string // import path for the package
	ModuleName  string // name of the module variable
}