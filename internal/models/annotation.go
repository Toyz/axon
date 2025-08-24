package models

// Annotation represents a parsed annotation from source code comments
type Annotation struct {
	Type         AnnotationType    // controller, route, middleware, core, interface
	Target       string            // struct or method name
	Parameters   map[string]string // key-value parameters from annotation
	Flags        []string          // flags like -Init, -Manual, etc.
	Dependencies []Dependency      // dependencies extracted from fx.In fields
	FileName     string            // name of the file containing this annotation
	Line         int               // line number of the annotation
}