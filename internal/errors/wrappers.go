package errors

import "fmt"

// Common error wrapping patterns used throughout the codebase
// These replace the duplicated wrapping functions in utils/errors.go

// WrapWithOperation wraps an error with an operation context
func WrapWithOperation(operation, item string, cause error) *BaseError {
	message := fmt.Sprintf("failed to %s %s", operation, item)
	return Wrap(UnknownErrorCode, message, cause)
}

// WrapRegisterError wraps an error with a "failed to register" message
func WrapRegisterError(componentType, name string, cause error) *RegistrationError {
	err := NewRegistrationError(componentType, name, cause.Error())
	err.WithCause(cause)
	return err
}

// WrapParseError wraps an error with a "failed to parse" message
func WrapParseError(item string, cause error) *SyntaxError {
	message := fmt.Sprintf("failed to parse %s", item)
	return &SyntaxError{
		BaseError: Wrap(SyntaxErrorCode, message, cause),
	}
}

// WrapGenerateError wraps an error with a "failed to generate" message
func WrapGenerateError(generationType, item string, cause error) *GenerationError {
	message := fmt.Sprintf("failed to generate %s", item)
	return &GenerationError{
		BaseError:      Wrap(GenerationErrorCode, message, cause),
		GenerationType: generationType,
		TargetFile:     item,
	}
}

// WrapValidationError wraps an error with a "failed to validate" message
func WrapValidationError(field string, cause error) *ValidationError {
	return &ValidationError{
		BaseError: Wrap(ValidationErrorCode, fmt.Sprintf("failed to validate %s", field), cause),
		Field:     field,
	}
}

// WrapFileSystemError wraps file system related errors
func WrapFileSystemError(operation, path string, cause error) *BaseError {
	message := fmt.Sprintf("failed to %s file '%s'", operation, path)
	return Wrap(FileSystemErrorCode, message, cause).
		WithContext("operation", operation).
		WithContext("path", path)
}

// WrapTemplateError wraps template processing errors
func WrapTemplateError(templateName, operation string, cause error) *GenerationError {
	message := fmt.Sprintf("failed to %s template '%s'", operation, templateName)
	return &GenerationError{
		BaseError:      Wrap(TemplateErrorCode, message, cause),
		GenerationType: "template",
		TargetFile:     templateName,
		Stage:          operation,
	}
}

// WrapConfigurationError wraps configuration-related errors
func WrapConfigurationError(configType, operation string, cause error) *BaseError {
	message := fmt.Sprintf("failed to %s configuration '%s'", operation, configType)
	return Wrap(ConfigurationErrorCode, message, cause).
		WithContext("config_type", configType).
		WithContext("operation", operation)
}

// WrapDependencyError wraps dependency injection errors
func WrapDependencyError(dependencyType, dependencyName string, cause error) *BaseError {
	message := fmt.Sprintf("failed to resolve dependency '%s' of type '%s'", dependencyName, dependencyType)
	return Wrap(DependencyErrorCode, message, cause).
		WithContext("dependency_type", dependencyType).
		WithContext("dependency_name", dependencyName)
}

// Convenience functions for common operations

// RegisterError creates a registration error without wrapping
func RegisterError(componentType, name, reason string) *RegistrationError {
	return NewRegistrationError(componentType, name, reason)
}

// ParseError creates a syntax error without wrapping
func ParseError(message string) *SyntaxError {
	return NewSyntaxError(message)
}

// ValidateError creates a validation error without wrapping
func ValidateError(field, expected, actual string) *ValidationError {
	return NewValidationError(field, expected, actual)
}

// GenerateError creates a generation error without wrapping
func GenerateError(message string) *GenerationError {
	return NewGenerationError(message)
}

// CreateSchemaError creates a schema error without wrapping
func CreateSchemaError(message string) *SchemaError {
	return NewSchemaError(message)
}

// FileSystemError creates a file system error
func FileSystemError(operation, path, message string) *BaseError {
	fullMessage := fmt.Sprintf("failed to %s file '%s': %s", operation, path, message)
	return New(FileSystemErrorCode, fullMessage).
		WithContext("operation", operation).
		WithContext("path", path)
}

// TemplateError creates a template error
func TemplateError(templateName, operation, message string) *GenerationError {
	fullMessage := fmt.Sprintf("template error in '%s' during %s: %s", templateName, operation, message)
	return &GenerationError{
		BaseError:      New(TemplateErrorCode, fullMessage),
		GenerationType: "template",
		TargetFile:     templateName,
		Stage:          operation,
	}
}

// ConfigurationError creates a configuration error
func ConfigurationError(configType, message string) *BaseError {
	fullMessage := fmt.Sprintf("configuration error in '%s': %s", configType, message)
	return New(ConfigurationErrorCode, fullMessage).
		WithContext("config_type", configType)
}

// DependencyError creates a dependency error
func DependencyError(dependencyType, dependencyName, message string) *BaseError {
	fullMessage := fmt.Sprintf("dependency error for '%s' of type '%s': %s", dependencyName, dependencyType, message)
	return New(DependencyErrorCode, fullMessage).
		WithContext("dependency_type", dependencyType).
		WithContext("dependency_name", dependencyName)
}

// Error collection helpers

// CollectValidationErrors creates a MultipleErrors from validation errors
func CollectValidationErrors(errors ...*ValidationError) *MultipleErrors {
	axonErrors := make([]AxonError, len(errors))
	for i, err := range errors {
		axonErrors[i] = err
	}
	return CollectErrors(axonErrors...)
}

// AddToMultiple adds an error to a MultipleErrors, creating it if nil
func AddToMultiple(multiple **MultipleErrors, err AxonError) {
	if *multiple == nil {
		*multiple = NewMultipleErrors()
	}
	(*multiple).Add(err)
}

// AddValidationError adds a validation error to a collection
func AddValidationError(multiple **MultipleErrors, field, expected, actual string) {
	AddToMultiple(multiple, NewValidationError(field, expected, actual))
}

// AddSyntaxError adds a syntax error to a collection
func AddSyntaxError(multiple **MultipleErrors, message string) {
	AddToMultiple(multiple, NewSyntaxError(message))
}

// AddRegistrationError adds a registration error to a collection
func AddRegistrationError(multiple **MultipleErrors, componentType, name, reason string) {
	AddToMultiple(multiple, NewRegistrationError(componentType, name, reason))
}