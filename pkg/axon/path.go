package axon

import (
	"strings"
)

// AxonPathPartType represents the type of path part
type AxonPathPartType int

const (
	StaticPart AxonPathPartType = iota
	ParameterPart
	WildcardPart
)

// AxonPathPart represents a single part of an Axon path
type AxonPathPart struct {
	Type      AxonPathPartType
	Value     string // For static parts: the literal text, for parameters: the parameter name
	ParamType string // For parameters: the type (e.g., "int", "string"), empty for untyped
}


// AxonPath represents a path in Axon format and provides parsed parts
type AxonPath string

// Raw returns the original Axon path format
func (p AxonPath) Raw() string {
	return string(p)
}

// Parts parses the Axon path and returns the individual parts
func (p AxonPath) Parts() []AxonPathPart {
	path := string(p)
	var parts []AxonPathPart

	i := 0
	for i < len(path) {
		if path[i] == '{' {
			// Find the closing brace
			j := i + 1
			for j < len(path) && path[j] != '}' {
				j++
			}
			if j < len(path) {
				paramContent := path[i+1 : j]

				// Check for wildcard
				if paramContent == "*" {
					parts = append(parts, AxonPathPart{
						Type:  WildcardPart,
						Value: "*",
					})
				} else {
					// Extract parameter name and type
					paramName := paramContent
					paramType := ""
					if colonIndex := strings.Index(paramContent, ":"); colonIndex != -1 {
						paramName = paramContent[:colonIndex]
						paramType = paramContent[colonIndex+1:]
					}

					parts = append(parts, AxonPathPart{
						Type:      ParameterPart,
						Value:     paramName,
						ParamType: paramType,
					})
				}
				i = j + 1
			} else {
				// Malformed, treat as static
				parts = append(parts, AxonPathPart{
					Type:  StaticPart,
					Value: string(path[i]),
				})
				i++
			}
		} else {
			// Static part - collect consecutive static characters
			start := i
			for i < len(path) && path[i] != '{' {
				i++
			}
			parts = append(parts, AxonPathPart{
				Type:  StaticPart,
				Value: path[start:i],
			})
		}
	}

	return parts
}


// NewAxonPath creates a new AxonPath from a string
func NewAxonPath(path string) AxonPath {
	return AxonPath(path)
}