package errors

import "fmt"

// ValidationError represents a validation error with detailed context
type ValidationError struct {
	*BaseError
	Field      string      // field that failed validation
	Value      interface{} // the value that failed validation
	Expected   string      // what was expected
	Actual     string      // what was provided
	Constraint string      // the validation constraint that failed
}

// NewValidationError creates a new validation error
func NewValidationError(field, expected, actual string) *ValidationError {
	message := fmt.Sprintf("validation failed for field '%s': expected %s, got %s", field, expected, actual)

	return &ValidationError{
		BaseError: New(ValidationErrorCode, message),
		Field:     field,
		Expected:  expected,
		Actual:    actual,
	}
}

// NewValidationErrorWithValue creates a validation error with the actual value
func NewValidationErrorWithValue(field string, value interface{}, constraint string) *ValidationError {
	message := fmt.Sprintf("validation failed for field '%s': %s", field, constraint)

	return &ValidationError{
		BaseError:  New(ValidationErrorCode, message),
		Field:      field,
		Value:      value,
		Constraint: constraint,
	}
}

// WithField sets the field name for the validation error
func (e *ValidationError) WithField(field string) *ValidationError {
	e.Field = field
	return e
}

// WithValue sets the value that failed validation
func (e *ValidationError) WithValue(value interface{}) *ValidationError {
	e.Value = value
	return e
}

// WithExpected sets what was expected
func (e *ValidationError) WithExpected(expected string) *ValidationError {
	e.Expected = expected
	return e
}

// WithActual sets what was actually provided
func (e *ValidationError) WithActual(actual string) *ValidationError {
	e.Actual = actual
	return e
}

// WithConstraint sets the constraint that failed
func (e *ValidationError) WithConstraint(constraint string) *ValidationError {
	e.Constraint = constraint
	return e
}

// WithLocation adds location information to the error
func (e *ValidationError) WithLocation(loc SourceLocation) *ValidationError {
	e.BaseError.WithLocation(loc)
	return e
}

// WithContext adds context data to the error
func (e *ValidationError) WithContext(key string, value interface{}) *ValidationError {
	e.BaseError.WithContext(key, value)
	return e
}

// WithSuggestion adds a helpful suggestion
func (e *ValidationError) WithSuggestion(suggestion string) *ValidationError {
	e.BaseError.WithSuggestion(suggestion)
	return e
}

// SyntaxError represents a syntax parsing error
type SyntaxError struct {
	*BaseError
	Token    string // the token that caused the error
	Position int    // position in the input where error occurred
}

// NewSyntaxError creates a new syntax error
func NewSyntaxError(message string) *SyntaxError {
	return &SyntaxError{
		BaseError: New(SyntaxErrorCode, message),
	}
}

// NewSyntaxErrorWithToken creates a syntax error with token information
func NewSyntaxErrorWithToken(message, token string, position int) *SyntaxError {
	if token != "" {
		message = fmt.Sprintf("%s (near token '%s')", message, token)
	}

	return &SyntaxError{
		BaseError: New(SyntaxErrorCode, message),
		Token:     token,
		Position:  position,
	}
}

// WithToken sets the problematic token
func (e *SyntaxError) WithToken(token string) *SyntaxError {
	e.Token = token
	return e
}

// WithPosition sets the position where the error occurred
func (e *SyntaxError) WithPosition(position int) *SyntaxError {
	e.Position = position
	return e
}

// WithLocation adds location information to the error
func (e *SyntaxError) WithLocation(loc SourceLocation) *SyntaxError {
	e.BaseError.WithLocation(loc)
	return e
}

// WithContext adds context data to the error
func (e *SyntaxError) WithContext(key string, value interface{}) *SyntaxError {
	e.BaseError.WithContext(key, value)
	return e
}

// WithSuggestion adds a helpful suggestion
func (e *SyntaxError) WithSuggestion(suggestion string) *SyntaxError {
	e.BaseError.WithSuggestion(suggestion)
	return e
}

// RegistrationError represents an error during component registration
type RegistrationError struct {
	*BaseError
	ComponentType string // type of component being registered
	ComponentName string // name of the component
	Reason        string // reason for registration failure
}

// NewRegistrationError creates a new registration error
func NewRegistrationError(componentType, componentName, reason string) *RegistrationError {
	message := fmt.Sprintf("failed to register %s '%s': %s", componentType, componentName, reason)

	return &RegistrationError{
		BaseError:     New(RegistrationErrorCode, message),
		ComponentType: componentType,
		ComponentName: componentName,
		Reason:        reason,
	}
}

// WithComponentType sets the component type
func (e *RegistrationError) WithComponentType(componentType string) *RegistrationError {
	e.ComponentType = componentType
	return e
}

// WithComponentName sets the component name
func (e *RegistrationError) WithComponentName(componentName string) *RegistrationError {
	e.ComponentName = componentName
	return e
}

// WithReason sets the reason for the registration failure
func (e *RegistrationError) WithReason(reason string) *RegistrationError {
	e.Reason = reason
	return e
}

// WithLocation adds location information to the error
func (e *RegistrationError) WithLocation(loc SourceLocation) *RegistrationError {
	e.BaseError.WithLocation(loc)
	return e
}

// WithCause adds an underlying error cause
func (e *RegistrationError) WithCause(cause error) *RegistrationError {
	e.BaseError.WithCause(cause)
	return e
}

// SchemaError represents a schema-related error
type SchemaError struct {
	*BaseError
	SchemaType   string // type of schema
	SchemaName   string // name of the schema
	ParameterName string // parameter that caused the error (if applicable)
}

// NewSchemaError creates a new schema error
func NewSchemaError(message string) *SchemaError {
	return &SchemaError{
		BaseError: New(SchemaErrorCode, message),
	}
}

// NewSchemaErrorWithDetails creates a schema error with detailed information
func NewSchemaErrorWithDetails(schemaType, schemaName, message string) *SchemaError {
	fullMessage := fmt.Sprintf("schema error in %s '%s': %s", schemaType, schemaName, message)

	return &SchemaError{
		BaseError:  New(SchemaErrorCode, fullMessage),
		SchemaType: schemaType,
		SchemaName: schemaName,
	}
}

// WithSchemaType sets the schema type
func (e *SchemaError) WithSchemaType(schemaType string) *SchemaError {
	e.SchemaType = schemaType
	return e
}

// WithSchemaName sets the schema name
func (e *SchemaError) WithSchemaName(schemaName string) *SchemaError {
	e.SchemaName = schemaName
	return e
}

// WithParameterName sets the parameter that caused the error
func (e *SchemaError) WithParameterName(paramName string) *SchemaError {
	e.ParameterName = paramName
	return e
}

// WithLocation adds location information to the error
func (e *SchemaError) WithLocation(loc SourceLocation) *SchemaError {
	e.BaseError.WithLocation(loc)
	return e
}

// GenerationError represents an error during code generation
type GenerationError struct {
	*BaseError
	GenerationType string // type of generation (template, module, etc.)
	TargetFile     string // target file being generated
	Stage          string // stage of generation where error occurred
}

// NewGenerationError creates a new generation error
func NewGenerationError(message string) *GenerationError {
	return &GenerationError{
		BaseError: New(GenerationErrorCode, message),
	}
}

// NewGenerationErrorWithDetails creates a generation error with details
func NewGenerationErrorWithDetails(generationType, targetFile, stage, message string) *GenerationError {
	fullMessage := fmt.Sprintf("generation error in %s for '%s' at stage '%s': %s",
		generationType, targetFile, stage, message)

	return &GenerationError{
		BaseError:      New(GenerationErrorCode, fullMessage),
		GenerationType: generationType,
		TargetFile:     targetFile,
		Stage:          stage,
	}
}

// WithGenerationType sets the generation type
func (e *GenerationError) WithGenerationType(generationType string) *GenerationError {
	e.GenerationType = generationType
	return e
}

// WithTargetFile sets the target file
func (e *GenerationError) WithTargetFile(targetFile string) *GenerationError {
	e.TargetFile = targetFile
	return e
}

// WithStage sets the generation stage
func (e *GenerationError) WithStage(stage string) *GenerationError {
	e.Stage = stage
	return e
}

// ParserError represents parser-related errors
type ParserError struct {
	*BaseError
	ParserType   string // type of parser
	TypeName     string // type name being parsed
	FunctionName string // parser function name (if applicable)
}

// NewParserError creates a new parser error
func NewParserError(code ErrorCode, message string) *ParserError {
	return &ParserError{
		BaseError: New(code, message),
	}
}

// NewParserRegistrationError creates a parser registration error
func NewParserRegistrationError(typeName, fileName string, line int, existingFile string, existingLine int) *ParserError {
	message := fmt.Sprintf("parser for type '%s' already registered", typeName)

	return &ParserError{
		BaseError: New(ParserRegistrationErrorCode, message).
			WithLocation(SourceLocation{File: fileName, Line: line}).
			WithSuggestions(
				"Choose a different type name for your parser",
				"Remove the duplicate parser registration",
				fmt.Sprintf("Check existing parser at %s:%d", existingFile, existingLine),
			),
		ParserType: "route_parser",
		TypeName:   typeName,
	}
}

// NewParserValidationError creates a parser validation error
func NewParserValidationError(functionName, fileName string, line int, expectedSignature, actualIssue string) *ParserError {
	message := fmt.Sprintf("parser function '%s' has invalid signature: %s", functionName, actualIssue)

	return &ParserError{
		BaseError: New(ParserValidationErrorCode, message).
			WithLocation(SourceLocation{File: fileName, Line: line}).
			WithSuggestions(
				fmt.Sprintf("Expected signature: %s", expectedSignature),
				"Ensure the first parameter is echo.Context",
				"Ensure the second parameter is string",
				"Ensure the function returns (T, error)",
			),
		ParserType:   "route_parser",
		FunctionName: functionName,
	}
}

// WithParserType sets the parser type
func (e *ParserError) WithParserType(parserType string) *ParserError {
	e.ParserType = parserType
	return e
}

// WithTypeName sets the type name
func (e *ParserError) WithTypeName(typeName string) *ParserError {
	e.TypeName = typeName
	return e
}

// WithFunctionName sets the function name
func (e *ParserError) WithFunctionName(functionName string) *ParserError {
	e.FunctionName = functionName
	return e
}