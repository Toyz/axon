package axon

// RouteParserMetadata contains metadata about a route parameter parser
type RouteParserMetadata struct {
	TypeName     string `json:"type_name"`
	FunctionName string `json:"function_name"`
	PackagePath  string `json:"package_path"`

	// Function signature validation
	ParameterTypes []string `json:"parameter_types"`
	ReturnTypes    []string `json:"return_types"`

	// Source location for error reporting
	FileName string `json:"file_name"`
	Line     int    `json:"line"`
}
