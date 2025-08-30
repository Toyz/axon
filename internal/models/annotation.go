package models

import "github.com/toyz/axon/internal/annotations"

// Annotation represents a parsed annotation from source code comments
// This is now a wrapper around the new annotations.ParsedAnnotation
type Annotation struct {
	*annotations.ParsedAnnotation              // Embed the new annotation type
	Dependencies                  []Dependency // dependencies extracted from fx.In fields
	FileName                      string       // name of the file containing this annotation
	Line                          int          // line number of the annotation
}
