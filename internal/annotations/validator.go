package annotations

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
)

// SchemaValidator defines the interface for validating annotations against their schemas
type SchemaValidator interface {
	// Validate annotation against its schema
	Validate(annotation *ParsedAnnotation, schema AnnotationSchema) error

	// ApplyDefaults applies default values for missing optional parameters
	ApplyDefaults(annotation *ParsedAnnotation, schema AnnotationSchema) error

	// TransformParameters transforms parameter values to correct types
	TransformParameters(annotation *ParsedAnnotation, schema AnnotationSchema) error
}

// validator is the concrete implementation of SchemaValidator
type validator struct{}

// NewValidator creates a new schema validator
func NewValidator() SchemaValidator {
	return &validator{}
}

// Validate validates an annotation against its schema
func (v *validator) Validate(annotation *ParsedAnnotation, schema AnnotationSchema) error {
	var errors []error

	// Validate required parameters are present
	for paramName, paramSpec := range schema.Parameters {
		if paramSpec.Required {
			if _, exists := annotation.Parameters[paramName]; !exists {
				errors = append(errors, &ValidationError{
					Parameter: paramName,
					Expected:  fmt.Sprintf("required parameter of type %s", paramSpec.Type.String()),
					Actual:    "missing",
					Loc:       annotation.Location,
					Hint:      fmt.Sprintf("Add -%s=<value> to the annotation", paramName),
				})
			}
		}
	}

	// Validate parameter types and values
	for paramName, paramValue := range annotation.Parameters {
		paramSpec, exists := schema.Parameters[paramName]
		if !exists {
			errors = append(errors, &ValidationError{
				Parameter: paramName,
				Expected:  "known parameter",
				Actual:    fmt.Sprintf("unknown parameter '%s'", paramName),
				Loc:       annotation.Location,
				Hint:      fmt.Sprintf("Remove -%s or check parameter name spelling", paramName),
			})
			continue
		}

		// Validate parameter type
		if err := v.validateParameterType(paramName, paramSpec.Type, paramValue, annotation.Location); err != nil {
			errors = append(errors, err)
			continue
		}

		// Run custom validator if present
		if paramSpec.Validator != nil {
			if err := paramSpec.Validator(paramValue); err != nil {
				errors = append(errors, &ValidationError{
					Parameter: paramName,
					Expected:  "valid value",
					Actual:    fmt.Sprintf("%v", paramValue),
					Loc:       annotation.Location,
					Hint:      err.Error(),
				})
			}
		}
	}

	// Run custom annotation validators
	for _, customValidator := range schema.Validators {
		if err := customValidator(annotation); err != nil {
			errors = append(errors, &SchemaError{
				Msg:  err.Error(),
				Loc:  annotation.Location,
				Hint: "Check annotation parameters and their combinations",
			})
		}
	}

	// Return combined errors if any
	if len(errors) > 0 {
		return &MultipleValidationErrors{Errors: errors}
	}

	return nil
}

// ApplyDefaults applies default values for missing optional parameters
func (v *validator) ApplyDefaults(annotation *ParsedAnnotation, schema AnnotationSchema) error {
	if annotation.Parameters == nil {
		annotation.Parameters = make(map[string]interface{})
	}

	for paramName, paramSpec := range schema.Parameters {
		// Apply default value if parameter is missing and has a default
		if _, exists := annotation.Parameters[paramName]; !exists && paramSpec.DefaultValue != nil {
			annotation.Parameters[paramName] = paramSpec.DefaultValue
		}
	}

	return nil
}

// TransformParameters transforms parameter values to correct types
func (v *validator) TransformParameters(annotation *ParsedAnnotation, schema AnnotationSchema) error {
	for paramName, paramValue := range annotation.Parameters {
		paramSpec, exists := schema.Parameters[paramName]
		if !exists {
			continue // Skip unknown parameters, they'll be caught in validation
		}

		// Transform the parameter value to the correct type
		transformedValue, err := v.transformParameterValue(paramValue, paramSpec.Type)
		if err != nil {
			return &ValidationError{
				Parameter: paramName,
				Expected:  fmt.Sprintf("value convertible to %s", paramSpec.Type.String()),
				Actual:    fmt.Sprintf("%v (%T)", paramValue, paramValue),
				Loc:       annotation.Location,
				Hint:      fmt.Sprintf("Ensure the value can be converted to %s", paramSpec.Type.String()),
			}
		}

		annotation.Parameters[paramName] = transformedValue
	}

	return nil
}

// validateParameterType validates that a parameter value matches the expected type
func (v *validator) validateParameterType(paramName string, expectedType ParameterType, value interface{}, location SourceLocation) error {
	switch expectedType {
	case StringType:
		if _, ok := value.(string); !ok {
			return &ValidationError{
				Parameter: paramName,
				Expected:  "string",
				Actual:    fmt.Sprintf("%T", value),
				Loc:       location,
				Hint:      "Provide a string value",
			}
		}
	case BoolType:
		if _, ok := value.(bool); !ok {
			return &ValidationError{
				Parameter: paramName,
				Expected:  "bool",
				Actual:    fmt.Sprintf("%T", value),
				Loc:       location,
				Hint:      "Use true/false or provide as a flag",
			}
		}
	case IntType:
		if _, ok := value.(int); !ok {
			return &ValidationError{
				Parameter: paramName,
				Expected:  "int",
				Actual:    fmt.Sprintf("%T", value),
				Loc:       location,
				Hint:      "Provide an integer value",
			}
		}
	case StringSliceType:
		if _, ok := value.([]string); !ok {
			return &ValidationError{
				Parameter: paramName,
				Expected:  "[]string",
				Actual:    fmt.Sprintf("%T", value),
				Loc:       location,
				Hint:      "Provide comma-separated string values",
			}
		}
	default:
		return &ValidationError{
			Parameter: paramName,
			Expected:  "known type",
			Actual:    fmt.Sprintf("unknown type %d", expectedType),
			Loc:       location,
			Hint:      "Contact the developer - this is a schema definition error",
		}
	}

	return nil
}

// transformParameterValue attempts to transform a value to the target type
func (v *validator) transformParameterValue(value interface{}, targetType ParameterType) (interface{}, error) {
	// If already the correct type, return as-is
	if v.isCorrectType(value, targetType) {
		return value, nil
	}

	// Try to convert from string (common case from parsing)
	if strValue, ok := value.(string); ok {
		return v.convertFromString(strValue, targetType)
	}

	// Try to convert using reflection for other cases
	return v.convertUsingReflection(value, targetType)
}

// isCorrectType checks if a value is already the correct type
func (v *validator) isCorrectType(value interface{}, targetType ParameterType) bool {
	switch targetType {
	case StringType:
		_, ok := value.(string)
		return ok
	case BoolType:
		_, ok := value.(bool)
		return ok
	case IntType:
		_, ok := value.(int)
		return ok
	case StringSliceType:
		_, ok := value.([]string)
		return ok
	default:
		return false
	}
}

// convertFromString converts a string value to the target type
func (v *validator) convertFromString(strValue string, targetType ParameterType) (interface{}, error) {
	switch targetType {
	case StringType:
		return strValue, nil
	case BoolType:
		return strconv.ParseBool(strValue)
	case IntType:
		return strconv.Atoi(strValue)
	case StringSliceType:
		// Split by comma and trim whitespace
		if strValue == "" {
			return []string{}, nil
		}
		parts := strings.Split(strValue, ",")
		result := make([]string, len(parts))
		for i, part := range parts {
			result[i] = strings.TrimSpace(part)
		}
		return result, nil
	default:
		return nil, fmt.Errorf("unsupported target type: %d", targetType)
	}
}

// convertUsingReflection attempts conversion using reflection
func (v *validator) convertUsingReflection(value interface{}, targetType ParameterType) (interface{}, error) {
	valueReflect := reflect.ValueOf(value)

	switch targetType {
	case StringType:
		return fmt.Sprintf("%v", value), nil
	case BoolType:
		// Try to convert various types to bool
		switch valueReflect.Kind() {
		case reflect.Bool:
			return valueReflect.Bool(), nil
		case reflect.String:
			return strconv.ParseBool(valueReflect.String())
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			return valueReflect.Int() != 0, nil
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			return valueReflect.Uint() != 0, nil
		default:
			return nil, fmt.Errorf("cannot convert %T to bool", value)
		}
	case IntType:
		// Try to convert various types to int
		switch valueReflect.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			return int(valueReflect.Int()), nil
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			return int(valueReflect.Uint()), nil
		case reflect.String:
			return strconv.Atoi(valueReflect.String())
		default:
			return nil, fmt.Errorf("cannot convert %T to int", value)
		}
	case StringSliceType:
		// Try to convert slice types to []string
		if valueReflect.Kind() == reflect.Slice {
			length := valueReflect.Len()
			result := make([]string, length)
			for i := 0; i < length; i++ {
				result[i] = fmt.Sprintf("%v", valueReflect.Index(i).Interface())
			}
			return result, nil
		}
		// Convert single value to slice
		return []string{fmt.Sprintf("%v", value)}, nil
	default:
		return nil, fmt.Errorf("unsupported target type: %d", targetType)
	}
}
