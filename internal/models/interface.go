package models

// InterfaceMetadata represents an interface to be generated from a struct
type InterfaceMetadata struct {
	BaseMetadata
	PackagePath string   // package where struct is defined
	Methods     []Method // public methods to include in interface
}

// Method represents a method signature for interface generation
type Method struct {
	Name       string      // method name
	Parameters []Parameter // method parameters
	Returns    []string    // return types
}