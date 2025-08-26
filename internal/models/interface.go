package models

// InterfaceMetadata represents an interface to be generated from a struct
type InterfaceMetadata struct {
	Name         string       // name of the interface (StructNameInterface)
	StructName   string       // name of the source struct
	PackagePath  string       // package where struct is defined
	Methods      []Method     // public methods to include in interface
	Dependencies []Dependency // dependencies of the source struct
}

// Method represents a method signature for interface generation
type Method struct {
	Name       string      // method name
	Parameters []Parameter // method parameters
	Returns    []string    // return types
}
