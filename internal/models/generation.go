package models

// GeneratedModule represents a generated FX module
type GeneratedModule struct {
	PackageName string     // name of the package
	FilePath    string     // path where module file should be written
	Content     string     // generated Go code content
	Providers   []Provider // FX providers in this module
}

// Provider represents an FX provider function
type Provider struct {
	Name         string   // name of the provider function
	StructName   string   // name of the struct being provided
	Dependencies []Dependency // dependencies required by provider
	IsLifecycle  bool     // whether provider handles lifecycle
}