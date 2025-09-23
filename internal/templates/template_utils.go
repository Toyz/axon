package templates

import (
	"strings"

	"github.com/toyz/axon/internal/models"
)

// TemplateUtils provides common utilities for template generation
type TemplateUtils struct{}

// NewTemplateUtils creates a new template utilities instance
func NewTemplateUtils() *TemplateUtils {
	return &TemplateUtils{}
}

// ConvertDependencies converts model dependencies to template dependencies
func (tu *TemplateUtils) ConvertDependencies(deps []models.Dependency) []DependencyData {
	var result []DependencyData
	for _, dep := range deps {
		result = append(result, DependencyData{
			Name:      tu.ToCamelCase(dep.Name),
			FieldName: dep.Name,
			Type:      dep.Type,
			IsInit:    dep.IsInit,
		})
	}
	return result
}

// FilterInjectedDependencies filters out init dependencies
func (tu *TemplateUtils) FilterInjectedDependencies(deps []DependencyData) []DependencyData {
	var result []DependencyData
	for _, dep := range deps {
		if !dep.IsInit {
			result = append(result, dep)
		}
	}
	return result
}

// ToCamelCase converts a string to camelCase
func (tu *TemplateUtils) ToCamelCase(s string) string {
	if s == "" {
		return s
	}
	return strings.ToLower(s[:1]) + s[1:]
}

// IsConfigLikeType checks if a type appears to be a configuration type
func (tu *TemplateUtils) IsConfigLikeType(typeName string) bool {
	lower := strings.ToLower(typeName)
	return strings.Contains(lower, "config") ||
		   strings.Contains(lower, "settings") ||
		   strings.Contains(lower, "options")
}

// FindConfigParam finds a config parameter from dependencies
func (tu *TemplateUtils) FindConfigParam(deps []DependencyData) string {
	for _, dep := range deps {
		if tu.IsConfigLikeType(dep.Type) {
			return dep.Name
		}
	}
	return ""
}

// BuildProviderName creates a provider function name from a struct name
func (tu *TemplateUtils) BuildProviderName(structName string) string {
	return "New" + structName + "Provider"
}

// BuildInterfaceName creates an interface name from a struct name
func (tu *TemplateUtils) BuildInterfaceName(structName string) string {
	return structName + "Interface"
}

// ShouldGenerateProvider determines if a provider should be generated
func (tu *TemplateUtils) ShouldGenerateProvider(service models.CoreServiceMetadata) bool {
	return !service.IsManualModule() && service.GetConstructor() == ""
}

// ShouldGenerateInitInvoke determines if init invoke should be generated
func (tu *TemplateUtils) ShouldGenerateInitInvoke(service models.CoreServiceMetadata) bool {
	return service.IsLifecycleEnabled()
}

// ExtractTypeName extracts the type name from a potentially qualified type
func (tu *TemplateUtils) ExtractTypeName(qualifiedType string) string {
	if strings.Contains(qualifiedType, ".") {
		parts := strings.Split(qualifiedType, ".")
		return parts[len(parts)-1]
	}
	return qualifiedType
}

// QuoteString wraps a string in quotes for code generation
func (tu *TemplateUtils) QuoteString(s string) string {
	return `"` + s + `"`
}

// JoinQuoted joins strings with quotes and commas for array literals
func (tu *TemplateUtils) JoinQuoted(items []string) string {
	if len(items) == 0 {
		return ""
	}

	var quoted []string
	for _, item := range items {
		quoted = append(quoted, tu.QuoteString(item))
	}

	return strings.Join(quoted, ", ")
}

// IsLoggerType checks if a type represents a logger dependency
func (tu *TemplateUtils) IsLoggerType(typeName string) bool {
	// Remove pointer prefix for analysis
	baseType := strings.TrimPrefix(typeName, "*")

	// Check for common logger patterns
	loggerPatterns := []string{
		"slog.Logger",
		"log.Logger",
		"Logger",
		"log.Entry",
		"logrus.Logger",
		"zap.Logger",
	}

	for _, pattern := range loggerPatterns {
		if strings.Contains(baseType, pattern) {
			return true
		}
	}

	return false
}

// BuildMiddlewareInstancesArray builds a Go array literal for middleware instances
func (tu *TemplateUtils) BuildMiddlewareInstancesArray(middlewares []string) string {
	if len(middlewares) == 0 {
		return "[]axon.MiddlewareInstance{}"
	}

	var instances []string
	for _, mw := range middlewares {
		varName := strings.ToLower(mw)
		instance := `{
		Name:     "` + mw + `",
		Handler:  ` + varName + `.Handle,
		Instance: ` + varName + `,
	}`
		instances = append(instances, instance)
	}

	return "[]axon.MiddlewareInstance{" + strings.Join(instances, ", ") + "}"
}

// BuildParameterInstancesArray builds a Go array literal for parameter instances
func (tu *TemplateUtils) BuildParameterInstancesArray(paramTypes map[string]string) string {
	if len(paramTypes) == 0 {
		return "[]axon.ParameterInstance{}"
	}

	var instances []string
	for name, typ := range paramTypes {
		instance := `{
		Name: "` + name + `",
		Type: "` + typ + `",
	}`
		instances = append(instances, instance)
	}

	return "[]axon.ParameterInstance{" + strings.Join(instances, ", ") + "}"
}

// BuildMiddlewareList builds a comma-separated list of middleware handlers
func (tu *TemplateUtils) BuildMiddlewareList(middlewares []string) string {
	if len(middlewares) == 0 {
		return ""
	}

	var handlers []string
	for _, mw := range middlewares {
		handlers = append(handlers, strings.ToLower(mw)+".Handle")
	}

	return strings.Join(handlers, ", ")
}

// BuildMiddlewaresArray builds a Go string array literal for middleware names
func (tu *TemplateUtils) BuildMiddlewaresArray(middlewares []string) string {
	if len(middlewares) == 0 {
		return "[]string{}"
	}

	quoted := tu.JoinQuoted(middlewares)
	return "[]string{" + quoted + "}"
}

// DefaultTemplateUtils provides a global instance for convenience
var DefaultTemplateUtils = NewTemplateUtils()