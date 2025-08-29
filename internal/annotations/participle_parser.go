package annotations

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/alecthomas/participle/v2"
	"github.com/alecthomas/participle/v2/lexer"
)

// ParticipleParser represents a parser using alecthomas/participle
type ParticipleParser struct {
	parser   *participle.Parser[Annotation]
	registry AnnotationRegistry
}

// Annotation represents the root of an axon annotation
type Annotation struct {
	Comment   string `parser:"@Comment"`
	Axon      string `parser:"@Axon"`
	Separator string `parser:"@Separator"`
	Type      string `parser:"@Ident"`
	Target    string `parser:"@Ident?"`
}

// Parameter represents a key-value parameter
// type Parameter struct {
// 	Key   string `parser:"'-' @Ident"`
// 	Value string `parser:"(@Equals (@String | @Path | @Ident | @Number))?"`
// }

// Flag represents a boolean flag
// type Flag struct {
// 	Name string `parser:"'-' @Ident"`
// }

// Value represents a parameter value
// type Value struct {
// 	String *string `parser:"@String"`
// 	Path   *string `parser:"| @Path"`
// 	Ident  *string `parser:"| @Ident"`
// 	Number *float64 `parser:"| @Number"`
// }

// NewParticipleParser creates a new parser using participle
func NewParticipleParser(registry AnnotationRegistry) *ParticipleParser {
	// Define the lexer
	lex := lexer.MustSimple([]lexer.SimpleRule{
		{Name: "Comment", Pattern: `//`},
		{Name: "Axon", Pattern: `axon`},
		{Name: "Separator", Pattern: `::`},
		{Name: "String", Pattern: `"(\\"|[^"])*"`},
		{Name: "Path", Pattern: `/[^\s]*`}, // Handle paths starting with /
		{Name: "Ident", Pattern: `[a-zA-Z_][a-zA-Z0-9_]*`},
		{Name: "Number", Pattern: `[0-9]+(\.[0-9]+)?`},
		{Name: "Equals", Pattern: `=`},
		{Name: "Punct", Pattern: `[-()]`},
		{Name: "Whitespace", Pattern: `\s+`},
	})

	parser := participle.MustBuild[Annotation](
		participle.Lexer(lex),
		participle.Elide("Whitespace"),
		participle.UseLookahead(2),
	)

	return &ParticipleParser{
		parser:   parser,
		registry: registry,
	}
}

// ParseAnnotation parses an annotation string
func (p *ParticipleParser) ParseAnnotation(comment string, location SourceLocation) (*ParsedAnnotation, error) {
	// First, manually parse the basic annotation structure
	annotationType, target, remaining, err := p.parseBasicStructure(comment)
	if err != nil {
		return nil, fmt.Errorf("failed to parse basic structure: %w", err)
	}

	// Convert type string to AnnotationType
	parsedType, err := p.parseAnnotationType(annotationType)
	if err != nil {
		return nil, fmt.Errorf("invalid annotation type '%s': %w", annotationType, err)
	}

	// Create the result structure
	parsed := &ParsedAnnotation{
		Type:       parsedType,
		Target:     target,
		Parameters: make(map[string]interface{}),
		Location:   location,
		Raw:        comment,
	}

	// If there are no parameters or flags, skip parameter parsing but still validate
	if remaining == "" {
		// No parameters to parse
	} else {

		// Parse positional parameters and named parameters/flags
		positionalParams, namedPart, err := p.parsePositionalAndNamed(remaining)
		if err != nil {
			return nil, fmt.Errorf("failed to parse positional and named parts: %w", err)
		}

	// Handle positional parameters based on annotation type
	p.handlePositionalParameters(parsed, positionalParams)

	// Parse named parameters and flags if any
	if namedPart != "" {
		paramsAndFlags, err := p.parseParamsAndFlags(namedPart)
		if err != nil {
			return nil, fmt.Errorf("failed to parse parameters and flags: %w", err)
		}

		// Process named parameters and flags
		for _, item := range paramsAndFlags.Items {
			if item.HasValue && item.Value != nil {
				// This is a parameter with an explicit value
				key := item.Flag
				
				// Extract the raw value from ParamValue
				rawValue := item.Value.Raw
				
				// Convert value to appropriate type based on schema
				convertedValue := p.convertParameterValue(key, rawValue, parsed.Type)
				parsed.Parameters[key] = convertedValue
			} else {
				// This could be either a boolean flag or a parameter with default value
				key := item.Flag
				
				// Check if this is a boolean parameter
				if p.isBooleanParameter(key, parsed.Type) {
					// For boolean parameters, -Flag means true
					parsed.Parameters[key] = true
				} else if p.isParameterWithDefault(key, parsed.Type) {
					// For parameters with defaults, -Flag means use the default value
					defaultValue := p.getParameterDefault(key, parsed.Type)
					parsed.Parameters[key] = defaultValue
				} else {
					// Unknown parameter - this should be caught by schema validation
					// For now, treat as a boolean flag
					parsed.Parameters[key] = true
				}
			}
		}
	}
	} // End of parameter parsing else block

	// Don't apply default values - only use what the user explicitly specified

	// Validate against schema
	if p.registry != nil {
		err = p.validateAgainstSchema(parsed)
		if err != nil {
			return nil, fmt.Errorf("schema validation failed: %w", err)
		}
	}

	return parsed, nil
}

// parseBasicStructure manually parses the basic annotation structure
func (p *ParticipleParser) parseBasicStructure(comment string) (annotationType, target, remaining string, err error) {
	// Trim leading whitespace first
	comment = strings.TrimSpace(comment)
	
	// Remove comment prefix
	if !strings.HasPrefix(comment, "//") {
		return "", "", "", fmt.Errorf("annotation must start with '//'")
	}
	content := strings.TrimPrefix(comment, "//")
	content = strings.TrimSpace(content)

	// Check for axon prefix
	if !strings.HasPrefix(content, "axon::") {
		return "", "", "", fmt.Errorf("annotation must contain 'axon::' prefix")
	}
	content = strings.TrimPrefix(content, "axon::")

	// Split by spaces to get type and target
	parts := strings.Fields(content)
	if len(parts) == 0 {
		return "", "", "", fmt.Errorf("empty annotation")
	}

	annotationType = parts[0]

	// Everything after the type is remaining (could be target, positional params, or named params)
	remaining = strings.TrimPrefix(content, annotationType)
	remaining = strings.TrimSpace(remaining)

	// For now, don't try to distinguish targets from positional parameters
	// Let the caller handle this based on annotation type
	target = ""

	return annotationType, target, remaining, nil
}

// parseParamsAndFlags parses just the parameters and flags part using a simple manual parser
func (p *ParticipleParser) parseParamsAndFlags(input string) (*ParamsAndFlags, error) {
	if input == "" {
		return &ParamsAndFlags{}, nil
	}

	result := &ParamsAndFlags{}
	
	// Split by spaces to get individual parameters/flags
	parts := strings.Fields(input)
	
	for _, part := range parts {
		if !strings.HasPrefix(part, "-") {
			continue // Skip non-flag parts
		}
		
		// Remove the leading dash
		part = strings.TrimPrefix(part, "-")
		
		// Check if it has an equals sign (parameter) or not (flag)
		if strings.Contains(part, "=") {
			// This is a parameter with a value
			keyValue := strings.SplitN(part, "=", 2)
			if len(keyValue) == 2 {
				result.Items = append(result.Items, ParamItem{
					Flag:     keyValue[0],
					HasValue: true,
					Value:    &ParamValue{Raw: keyValue[1]},
				})
			}
		} else {
			// This is a boolean flag
			result.Items = append(result.Items, ParamItem{
				Flag:     part,
				HasValue: false,
				Value:    nil,
			})
		}
	}
	
	return result, nil
}

// ParamsAndFlags represents just the parameters and flags part
type ParamsAndFlags struct {
	Items []ParamItem `parser:"@@*"`
}

// ParamItem represents a single parameter or flag
type ParamItem struct {
	Flag      string      `parser:"@Dash @Ident"`
	HasValue  bool        `parser:"(@Equals"`
	Value     *ParamValue `parser:"  @@"`
	ClosePar  bool        `parser:")?"`
}

// ParamValue represents a parameter value - we'll parse it as a raw string and handle comma-separation later
type ParamValue struct {
	Raw string `parser:"@Value"`
}

// convertParameterValue converts a value to the appropriate type based on the schema
func (p *ParticipleParser) convertParameterValue(key string, value interface{}, annotationType AnnotationType) interface{} {
	if p.registry == nil {
		return value // Return as-is if no schema available
	}

	schema, err := p.registry.GetSchema(annotationType)
	if err != nil {
		return value
	}

	paramSpec, exists := schema.Parameters[key]
	if !exists {
		return value
	}

	switch paramSpec.Type {
	case IntType:
		if strVal, ok := value.(string); ok {
			if intVal, err := strconv.Atoi(strVal); err == nil {
				return intVal
			}
		}
		return value
	case BoolType:
		if strVal, ok := value.(string); ok {
			if boolVal, err := strconv.ParseBool(strVal); err == nil {
				return boolVal
			}
		}
		return value
	case StringSliceType:
		// If it's already a slice, return it
		if sliceVal, ok := value.([]string); ok {
			return sliceVal
		}
		// If it's a string, split by comma
		if strVal, ok := value.(string); ok {
			return strings.Split(strVal, ",")
		}
		return value
	case StringType:
		// For string types, remove surrounding quotes if present
		if strVal, ok := value.(string); ok {
			// Remove surrounding quotes (both single and double)
			if len(strVal) >= 2 {
				if (strVal[0] == '"' && strVal[len(strVal)-1] == '"') ||
					(strVal[0] == '\'' && strVal[len(strVal)-1] == '\'') {
					return strVal[1 : len(strVal)-1]
				}
			}
			return strVal
		}
		return value
	default:
		return value
	}
}

// parseAnnotationType converts string to AnnotationType
func (p *ParticipleParser) parseAnnotationType(typeStr string) (AnnotationType, error) {
	// Use the existing ParseAnnotationType function
	annotationType, err := ParseAnnotationType(typeStr)
	if err != nil {
		return CoreAnnotation, err
	}

	// Validate that this annotation type is registered in our schema registry
	if p.registry != nil && !p.registry.IsRegistered(annotationType) {
		return CoreAnnotation, fmt.Errorf("annotation type '%s' is not registered in schema registry", typeStr)
	}

	return annotationType, nil
}

// parsePositionalAndNamed separates positional parameters from named parameters/flags
func (p *ParticipleParser) parsePositionalAndNamed(input string) ([]string, string, error) {
	input = strings.TrimSpace(input)
	if input == "" {
		return nil, "", nil
	}

	// Split by spaces, but be careful about quoted strings and equals signs
	var positional []string
	var namedParts []string
	inNamed := false

	parts := strings.Fields(input)
	for _, part := range parts {
		if strings.HasPrefix(part, "-") {
			// This is a named parameter or flag
			inNamed = true
			namedParts = append(namedParts, part)
		} else if !inNamed {
			// This is a positional parameter
			positional = append(positional, part)
		} else {
			// This is part of named parameters (could be a value)
			namedParts = append(namedParts, part)
		}
	}

	namedStr := strings.Join(namedParts, " ")
	return positional, namedStr, nil
}

// handlePositionalParameters assigns positional parameters based on annotation type
func (p *ParticipleParser) handlePositionalParameters(annotation *ParsedAnnotation, positional []string) {
	switch annotation.Type {
	case RouteAnnotation:
		if len(positional) >= 1 {
			annotation.Parameters["method"] = positional[0]
		}
		if len(positional) >= 2 {
			annotation.Parameters["path"] = positional[1]
		}
	case MiddlewareAnnotation:
		if len(positional) >= 1 {
			annotation.Parameters["Name"] = positional[0]
		}
	case RouteParserAnnotation:
		if len(positional) >= 1 {
			annotation.Target = positional[0]
			annotation.Parameters["name"] = positional[0]
		}
	}
}

// convertValue converts a parsed Value to interface{}
// func (p *ParticipleParser) convertValue(value *Value) interface{} {
// 	if value.String != nil {
// 		// Remove quotes
// 		str := *value.String
// 		if len(str) >= 2 && str[0] == '"' && str[len(str)-1] == '"' {
// 			return str[1 : len(str)-1]
// 		}
// 		return str
// 	}
// 	if value.Path != nil {
// 		return *value.Path
// 	}
// 	if value.Ident != nil {
// 		return *value.Ident
// 	}
// 	if value.Number != nil {
// 		return *value.Number
// 	}
// 	return nil
// }

// validateAgainstSchema validates the parsed annotation against its schema
// but only checks that provided parameters/flags are valid - doesn't apply defaults
func (p *ParticipleParser) validateAgainstSchema(annotation *ParsedAnnotation) error {
	schema, err := p.registry.GetSchema(annotation.Type)
	if err != nil {
		return fmt.Errorf("no schema found for annotation type: %s", annotation.Type)
	}

	// Validate parameters (only those with explicit values)
	for paramName, paramValue := range annotation.Parameters {
		paramSpec, exists := schema.Parameters[paramName]
		if !exists {
			return fmt.Errorf("unknown parameter '%s' for annotation type %s", paramName, annotation.Type)
		}

		// Validate parameter value if validator is provided
		if paramSpec.Validator != nil {
			if err := paramSpec.Validator(paramValue); err != nil {
				return fmt.Errorf("parameter '%s' validation failed: %w", paramName, err)
			}
		}
	}

	// Check for missing required parameters
	for paramName, paramSpec := range schema.Parameters {
		if paramSpec.Required {
			if _, exists := annotation.Parameters[paramName]; !exists {
				// Special case for route annotations to provide better error messages
				if annotation.Type == RouteAnnotation {
					if paramName == "method" {
						return fmt.Errorf("route annotation requires method parameter (e.g., GET, POST)")
					}
					if paramName == "path" {
						return fmt.Errorf("route annotation requires path parameter (e.g., /users)")
					}
				}
				return fmt.Errorf("missing required parameter '%s' for annotation type %s", paramName, annotation.Type)
			}
		}
	}

	return nil
}

// ParseFile parses annotations from a file (placeholder for now)
func (p *ParticipleParser) ParseFile(filePath string) ([]*ParsedAnnotation, error) {
	// TODO: Implement file parsing
	return nil, fmt.Errorf("ParseFile not implemented yet")
}

// ValidateAnnotation validates an annotation (placeholder for now)
func (p *ParticipleParser) ValidateAnnotation(annotation *ParsedAnnotation) error {
	// TODO: Implement validation
	return nil
}
// isParameterWithDefault checks if a parameter supports being used without a value (with default)
func (p *ParticipleParser) isParameterWithDefault(paramName string, annotationType AnnotationType) bool {
	if p.registry == nil {
		return false
	}

	schema, err := p.registry.GetSchema(annotationType)
	if err != nil {
		return false
	}

	paramSpec, exists := schema.Parameters[paramName]
	if !exists {
		return false
	}

	// Parameters with default values can be used without explicit values
	return paramSpec.DefaultValue != nil
}

// getParameterDefault gets the default value for a parameter
func (p *ParticipleParser) getParameterDefault(paramName string, annotationType AnnotationType) interface{} {
	if p.registry == nil {
		return nil
	}

	schema, err := p.registry.GetSchema(annotationType)
	if err != nil {
		return nil
	}

	paramSpec, exists := schema.Parameters[paramName]
	if !exists {
		return nil
	}

	return paramSpec.DefaultValue
}

// isBooleanParameter checks if a parameter is of boolean type
func (p *ParticipleParser) isBooleanParameter(paramName string, annotationType AnnotationType) bool {
	if p.registry == nil {
		return false
	}

	schema, err := p.registry.GetSchema(annotationType)
	if err != nil {
		return false
	}

	paramSpec, exists := schema.Parameters[paramName]
	if !exists {
		return false
	}

	return paramSpec.Type == BoolType
}