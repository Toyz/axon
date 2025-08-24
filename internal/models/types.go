package models

// AnnotationType represents the type of annotation found in source code
type AnnotationType int

const (
	AnnotationTypeController AnnotationType = iota
	AnnotationTypeRoute
	AnnotationTypeMiddleware
	AnnotationTypeCore
	AnnotationTypeInterface
	AnnotationTypeInject
	AnnotationTypeInit
	AnnotationTypeLogger
	AnnotationTypeRouteParser
)

// ParameterSource represents where a parameter comes from
type ParameterSource int

const (
	ParameterSourcePath ParameterSource = iota
	ParameterSourceBody
	ParameterSourceContext
)

// ReturnType represents the type of return signature for handlers
type ReturnType int

const (
	ReturnTypeDataError ReturnType = iota
	ReturnTypeResponseError
	ReturnTypeError
)

// ErrorType represents different types of generator errors
type ErrorType int

const (
	ErrorTypeAnnotationSyntax ErrorType = iota
	ErrorTypeValidation
	ErrorTypeGeneration
	ErrorTypeFileSystem
)

// RouteParserMetadata represents metadata for a route parameter parser
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