package parser

const (
	// AnnotationPrefix is the prefix used for all framework annotations
	AnnotationPrefix = "axon::"
	
	// Annotation type constants
	AnnotationTypeController = "controller"
	AnnotationTypeRoute      = "route"
	AnnotationTypeMiddleware = "middleware"
	AnnotationTypeCore       = "core"
	AnnotationTypeInterface  = "interface"
	AnnotationTypeInject     = "inject"
	AnnotationTypeInit       = "init"
	AnnotationTypeLogger     = "logger"
	
	// Flag constants
	FlagInit        = "-Init"
	FlagManual      = "-Manual"
	FlagPassContext = "-PassContext"
	FlagMiddleware  = "-Middleware"
	
	// Parameter constants
	ParamMethod = "method"
	ParamPath   = "path"
	ParamName   = "name"
	
	// Default module name for manual services
	DefaultModuleName = "Module"
)