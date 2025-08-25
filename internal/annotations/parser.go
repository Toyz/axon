package annotations

import (
	"fmt"
	"strings"
	"unicode"
)

// ParserEngine interface defines the core parsing functionality
type ParserEngine interface {
	ParseAnnotation(comment string, location SourceLocation) (*ParsedAnnotation, error)
	ParseFile(filePath string) ([]*ParsedAnnotation, error)
	ValidateAnnotation(annotation *ParsedAnnotation) error
}

type parser struct {
	registry  AnnotationRegistry
	validator SchemaValidator
}

func NewParser(registry AnnotationRegistry) ParserEngine {
	return &parser{
		registry:  registry,
		validator: NewValidator(),
	}
}

type TokenType int

const (
	ParameterToken TokenType = iota
	FlagToken
	QuotedStringToken
	CommaToken
)

type AnnotationToken struct {
	Type     TokenType
	Value    string
	Position int
}

type ParseContext struct {
	Input  string
	Line   int
	Column int
}

type ParseError struct {
	Message    string
	Location   SourceLocation
	Suggestion string
}

func (e ParseError) Error() string {
	return fmt.Sprintf("%s:%d:%d: %s. %s",
		e.Location.File, e.Location.Line, e.Location.Column,
		e.Message, e.Suggestion)
}

func (p *parser) ParseAnnotation(comment string, location SourceLocation) (*ParsedAnnotation, error) {
	// Step 1: Normalize comment prefix
	normalized, err := p.normalizeCommentPrefix(comment, location)
	if err != nil {
		return nil, err
	}

	// Step 2: Simple tokenization
	tokens := p.simpleTokenize(normalized)

	// Step 3: Parse tokens
	annotation, err := p.parseTokens(tokens, location)
	if err != nil {
		return nil, err
	}

	// Step 4: Validate
	if p.registry != nil {
		if err := p.ValidateAnnotation(annotation); err != nil {
			return nil, err
		}
	}

	return annotation, nil
}

func (p *parser) normalizeCommentPrefix(comment string, location SourceLocation) (string, error) {
	input := strings.TrimSpace(comment)
	
	if !strings.HasPrefix(input, "//") {
		return "", ParseError{
			Message:    "annotation must start with '//'",
			Location:   location,
			Suggestion: "Use format: //axon::type parameters",
		}
	}

	withoutSlashes := input[2:]
	withoutSlashes = strings.TrimLeftFunc(withoutSlashes, unicode.IsSpace)
	
	if !strings.HasPrefix(withoutSlashes, "axon::") {
		return "", ParseError{
			Message:    "annotation must contain 'axon::' prefix",
			Location:   location,
			Suggestion: "Use format: //axon::type parameters",
		}
	}

	return strings.TrimSpace(withoutSlashes[6:]), nil
}

func (p *parser) simpleTokenize(input string) []AnnotationToken {
	var tokens []AnnotationToken
	parts := strings.Fields(input)
	
	for _, part := range parts {
		if strings.HasPrefix(part, "-") {
			tokens = append(tokens, AnnotationToken{
				Type:  FlagToken,
				Value: part,
			})
		} else {
			tokens = append(tokens, AnnotationToken{
				Type:  ParameterToken,
				Value: part,
			})
		}
	}
	
	return tokens
}

func (p *parser) parseTokens(tokens []AnnotationToken, location SourceLocation) (*ParsedAnnotation, error) {
	if len(tokens) == 0 {
		return nil, ParseError{
			Message:    "empty annotation",
			Location:   location,
			Suggestion: "Provide annotation type after 'axon::'",
		}
	}

	annotationType, err := ParseAnnotationType(tokens[0].Value)
	if err != nil {
		return nil, ParseError{
			Message:    fmt.Sprintf("unknown annotation type: %s", tokens[0].Value),
			Location:   location,
			Suggestion: "Use one of: core, route, controller, middleware, interface",
		}
	}

	annotation := &ParsedAnnotation{
		Type:       annotationType,
		Parameters: make(map[string]interface{}),
		Flags:      make([]string, 0),
		Location:   location,
	}

	// Process remaining tokens
	var positionalParams []string
	for i := 1; i < len(tokens); i++ {
		token := tokens[i]
		
		if token.Type == FlagToken {
			err := p.processFlagToken(token, annotation, location)
			if err != nil {
				return nil, err
			}
		} else {
			positionalParams = append(positionalParams, token.Value)
		}
	}

	// Handle positional parameters for different annotation types
	if annotation.Type == RouteAnnotation {
		if len(positionalParams) >= 1 {
			annotation.Parameters["method"] = positionalParams[0]
		}
		if len(positionalParams) >= 2 {
			annotation.Parameters["path"] = positionalParams[1]
		}
	} else if annotation.Type == MiddlewareAnnotation {
		if len(positionalParams) >= 1 {
			annotation.Parameters["name"] = positionalParams[0]
		}
	}

	// Validate the annotation
	err = p.ValidateAnnotation(annotation)
	if err != nil {
		return nil, err
	}

	return annotation, nil
}

func (p *parser) processFlagToken(token AnnotationToken, annotation *ParsedAnnotation, location SourceLocation) error {
	flagValue := token.Value
	
	// Remove leading dash
	if strings.HasPrefix(flagValue, "-") {
		flagValue = flagValue[1:]
	}

	// Check if it has explicit value (-Mode=Transient)
	if strings.Contains(flagValue, "=") {
		parts := strings.SplitN(flagValue, "=", 2)
		paramName := parts[0]
		paramValue := parts[1]
		
		// Strip quotes from parameter value
		paramValue = p.stripQuotes(paramValue)
		
		// Handle comma-separated values
		if strings.Contains(paramValue, ",") {
			values := strings.Split(paramValue, ",")
			for i, v := range values {
				values[i] = strings.TrimSpace(p.stripQuotes(v))
			}
			annotation.Parameters[paramName] = values
		} else {
			annotation.Parameters[paramName] = paramValue
		}
		return nil
	}

	// This is a flag without explicit value (-Init or -PassContext)
	paramName := flagValue
	
	// Special handling for backward compatibility with old parser
	// Certain flags should remain as flags even if they have schema definitions
	legacyFlags := map[string]bool{
		"Init":   true,
		"Manual": true,
	}
	
	if legacyFlags[paramName] {
		// Add to flags array for backward compatibility
		annotation.Flags = append(annotation.Flags, "-"+paramName)
		return nil
	}
	
	if p.registry != nil {
		schema, err := p.registry.GetSchema(annotation.Type)
		if err == nil {
			if paramSpec, exists := schema.Parameters[paramName]; exists {
				// For boolean parameters, flag without value means true
				if paramSpec.Type == BoolType {
					annotation.Parameters[paramName] = true
					return nil
				}
				// For non-boolean parameters, use schema default: -Init becomes Init: "Same"
				annotation.Parameters[paramName] = paramSpec.DefaultValue
				return nil
			}
		}
	}

	// If no schema found, assume it's a boolean flag
	annotation.Parameters[paramName] = true
	return nil
}

// stripQuotes removes surrounding quotes from a string
func (p *parser) stripQuotes(s string) string {
	s = strings.TrimSpace(s)
	if len(s) >= 2 {
		if (s[0] == '"' && s[len(s)-1] == '"') || (s[0] == '\'' && s[len(s)-1] == '\'') {
			return s[1 : len(s)-1]
		}
	}
	return s
}

func (p *parser) ParseFile(filePath string) ([]*ParsedAnnotation, error) {
	return nil, fmt.Errorf("ParseFile not yet implemented")
}

func (p *parser) ValidateAnnotation(annotation *ParsedAnnotation) error {
	if p.registry == nil || p.validator == nil {
		return nil
	}

	schema, err := p.registry.GetSchema(annotation.Type)
	if err != nil {
		return ParseError{
			Message:    fmt.Sprintf("no schema found for annotation type: %s", annotation.Type),
			Location:   annotation.Location,
			Suggestion: "Check if annotation type is registered",
		}
	}

	// Apply default values first
	err = p.validator.ApplyDefaults(annotation, schema)
	if err != nil {
		return err
	}

	// Transform parameters to correct types
	err = p.validator.TransformParameters(annotation, schema)
	if err != nil {
		return err
	}

	// Validate the annotation
	err = p.validator.Validate(annotation, schema)
	if err != nil {
		return err
	}

	return nil
}