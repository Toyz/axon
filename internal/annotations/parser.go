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
	parts := p.splitRespectingQuotes(input)
	
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

// splitRespectingQuotes splits input on whitespace but respects quoted strings and flag values
func (p *parser) splitRespectingQuotes(input string) []string {
	var parts []string
	var current strings.Builder
	var inQuotes bool
	var quoteChar rune
	var inFlagValue bool
	
	for i, r := range input {
		switch {
		case !inQuotes && !inFlagValue && unicode.IsSpace(r):
			// Outside quotes and flag values, hit whitespace - end current token
			if current.Len() > 0 {
				parts = append(parts, current.String())
				current.Reset()
			}
		case !inQuotes && !inFlagValue && r == '-' && (i == 0 || unicode.IsSpace(rune(input[i-1]))):
			// Start of a flag (dash at beginning or after whitespace)
			if current.Len() > 0 {
				parts = append(parts, current.String())
				current.Reset()
			}
			current.WriteRune(r)
		case !inQuotes && r == '=' && strings.HasPrefix(current.String(), "-"):
			// Start of flag value
			inFlagValue = true
			current.WriteRune(r)
		case inFlagValue && !inQuotes && unicode.IsSpace(r):
			// In flag value, hit space - check if this ends the flag value
			// Look ahead to see if next non-space character starts a new flag
			nextFlagStart := false
			for j := i + 1; j < len(input); j++ {
				if !unicode.IsSpace(rune(input[j])) {
					if input[j] == '-' {
						nextFlagStart = true
					}
					break
				}
			}
			if nextFlagStart {
				// This space ends the flag value
				inFlagValue = false
				parts = append(parts, current.String())
				current.Reset()
			} else {
				// This space is part of the flag value
				current.WriteRune(r)
			}
		case !inQuotes && (r == '"' || r == '\''):
			// Start of quoted string
			inQuotes = true
			quoteChar = r
			current.WriteRune(r)
		case inQuotes && r == quoteChar:
			// Check if this quote is escaped by looking at the previous character
			isEscaped := false
			if i > 0 && input[i-1] == '\\' {
				// Count consecutive backslashes to determine if this quote is actually escaped
				backslashCount := 0
				for j := i - 1; j >= 0 && input[j] == '\\'; j-- {
					backslashCount++
				}
				// If odd number of backslashes, the quote is escaped
				isEscaped = (backslashCount%2 == 1)
			}
			
			if isEscaped {
				// This is an escaped quote, already handled by escape sequence processing
				// Don't add it again, just continue
			} else {
				// End of quoted string
				inQuotes = false
				current.WriteRune(r)
			}
		case inQuotes && r == '\\' && i+1 < len(input):
			// Escape sequence in quoted string
			current.WriteRune(r)
			i++ // Skip next character
			if i < len(input) {
				current.WriteRune(rune(input[i]))
			}
		default:
			// Regular character
			current.WriteRune(r)
		}
	}
	
	// Add final token if any
	if current.Len() > 0 {
		parts = append(parts, current.String())
	}
	
	return parts
}

// splitCommaSeparatedValues splits comma-separated values respecting quotes
func (p *parser) splitCommaSeparatedValues(input string) []string {
	var values []string
	var current strings.Builder
	var inQuotes bool
	var quoteChar rune
	var lastWasComma bool
	
	runes := []rune(input)
	for i := 0; i < len(runes); i++ {
		r := runes[i]
		
		switch {
		case !inQuotes && r == ',':
			// Outside quotes and hit comma - end current value
			value := strings.TrimSpace(current.String())
			values = append(values, p.stripQuotes(value))
			current.Reset()
			lastWasComma = true
		case !inQuotes && (r == '"' || r == '\''):
			// Start of quoted string
			inQuotes = true
			quoteChar = r
			current.WriteRune(r)
			lastWasComma = false
		case inQuotes && r == quoteChar:
			// End of quoted string
			inQuotes = false
			current.WriteRune(r)
			lastWasComma = false
		case inQuotes && r == '\\' && i+1 < len(runes):
			// Escape sequence in quoted string
			current.WriteRune(r)
			i++ // Skip next character
			if i < len(runes) {
				current.WriteRune(runes[i])
			}
			lastWasComma = false
		default:
			// Regular character
			current.WriteRune(r)
			lastWasComma = false
		}
	}
	
	// Add final value - always add if we have content or if last character was comma
	if current.Len() > 0 || lastWasComma {
		value := strings.TrimSpace(current.String())
		values = append(values, p.stripQuotes(value))
	}
	
	return values
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
			annotation.Parameters["Name"] = positionalParams[0]
		}
	} else if annotation.Type == RouteParserAnnotation {
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
		
		// Check if the value is quoted
		isQuoted := false
		if len(paramValue) >= 2 {
			if (paramValue[0] == '"' && paramValue[len(paramValue)-1] == '"') ||
			   (paramValue[0] == '\'' && paramValue[len(paramValue)-1] == '\'') {
				isQuoted = true
			}
		}
		
		// Check if the parameter expects a slice type from the schema
		expectsSlice := false
		if p.registry != nil {
			if schema, err := p.registry.GetSchema(annotation.Type); err == nil {
				if paramSpec, exists := schema.Parameters[paramName]; exists {
					expectsSlice = paramSpec.Type == StringSliceType
				}
			}
		}
		
		if isQuoted {
			// Quoted value - check if it should be treated as comma-separated
			unquotedValue := p.stripQuotes(paramValue)
			if expectsSlice && strings.Contains(unquotedValue, ",") {
				// Check if this looks like multiple quoted values: "val1","val2"
				// If the unquoted value contains quotes, it's likely multiple quoted values
				if strings.Contains(unquotedValue, "\"") || strings.Contains(unquotedValue, "'") {
					// Parse the original paramValue (with outer quotes) as comma-separated
					values := p.splitCommaSeparatedValues(paramValue)
					annotation.Parameters[paramName] = values
				} else {
					// Single quoted value with commas inside - split the unquoted content
					values := p.splitCommaSeparatedValues(unquotedValue)
					annotation.Parameters[paramName] = values
				}
			} else {
				// Treat as single string
				annotation.Parameters[paramName] = unquotedValue
			}
		} else if strings.Contains(paramValue, ",") {
			// Unquoted value with commas - treat as comma-separated list
			values := p.splitCommaSeparatedValues(paramValue)
			annotation.Parameters[paramName] = values
		} else {
			// Single unquoted value
			annotation.Parameters[paramName] = p.stripQuotes(paramValue)
		}
		return nil
	}

	// This is a flag without explicit value (-Init or -PassContext)
	paramName := flagValue
	
	// Special handling for backward compatibility with old parser
	// Certain flags should remain as flags even if they have schema definitions
	legacyFlags := map[string]bool{
		"Init": true,
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
				// For non-boolean parameters, use schema default or appropriate zero value
				if paramSpec.DefaultValue != nil {
					annotation.Parameters[paramName] = paramSpec.DefaultValue
				} else {
					// Use appropriate zero value for the type
					switch paramSpec.Type {
					case StringType:
						annotation.Parameters[paramName] = ""
					case IntType:
						annotation.Parameters[paramName] = 0
					case StringSliceType:
						annotation.Parameters[paramName] = []string{}
					default:
						annotation.Parameters[paramName] = ""
					}
				}
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
			unquoted := s[1 : len(s)-1]
			
			// Process escape sequences manually for more robust handling
			result := make([]rune, 0, len(unquoted))
			runes := []rune(unquoted)
			
			for i := 0; i < len(runes); i++ {
				if runes[i] == '\\' && i+1 < len(runes) {
					// Handle escape sequence
					switch runes[i+1] {
					case '"':
						result = append(result, '"')
						i++ // Skip the escaped character
					case '\'':
						result = append(result, '\'')
						i++ // Skip the escaped character
					case '\\':
						result = append(result, '\\')
						i++ // Skip the escaped character
					case 'n':
						result = append(result, '\n')
						i++ // Skip the escaped character
					case 't':
						result = append(result, '\t')
						i++ // Skip the escaped character
					case 'r':
						result = append(result, '\r')
						i++ // Skip the escaped character
					default:
						// Unknown escape sequence, keep the backslash
						result = append(result, runes[i])
					}
				} else {
					result = append(result, runes[i])
				}
			}
			
			return string(result)
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