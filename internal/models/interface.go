package models

// InterfaceMetadata represents an interface to be generated from a struct using composition
type InterfaceMetadata struct {
	BaseMetadataTrait
	PathTrait
	Methods []Method // public methods to include in interface
}

// Method represents a method signature for interface generation
type Method struct {
	Name       string      // method name
	Parameters []Parameter // method parameters
	Returns    []string    // return types
}
