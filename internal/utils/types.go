package utils

import "strings"

// ExtractPackageFromType extracts the package name from a type string like "*config.Config"
func ExtractPackageFromType(typeStr string) string {
	// Remove pointer prefix
	typeStr = strings.TrimPrefix(typeStr, "*")

	// Handle complex types like maps, slices, channels
	if strings.HasPrefix(typeStr, "map[") {
		// For maps, extract package from the value type
		// Find the closing bracket of the key type
		bracketCount := 0
		valueStart := -1
		for i, char := range typeStr {
			if char == '[' {
				bracketCount++
			} else if char == ']' {
				bracketCount--
				if bracketCount == 0 {
					valueStart = i + 1
					break
				}
			}
		}
		if valueStart > 0 && valueStart < len(typeStr) {
			valueType := typeStr[valueStart:]
			return ExtractPackageFromType(valueType) // Recursive call for value type
		}
	} else if strings.HasPrefix(typeStr, "[]") {
		// For slices, extract package from the element type
		elementType := typeStr[2:]
		return ExtractPackageFromType(elementType) // Recursive call for element type
	} else if strings.HasPrefix(typeStr, "chan ") {
		// For channels, extract package from the element type
		elementType := typeStr[5:]
		return ExtractPackageFromType(elementType) // Recursive call for element type
	} else if strings.HasPrefix(typeStr, "func(") {
		// For function types, extract package from return types
		// Example: "func() *services.SessionService" -> "services"
		// Find the return type after the closing parenthesis
		parenCount := 0
		returnStart := -1
		for i, char := range typeStr {
			if char == '(' {
				parenCount++
			} else if char == ')' {
				parenCount--
				if parenCount == 0 && i+1 < len(typeStr) {
					returnStart = i + 1
					break
				}
			}
		}
		if returnStart > 0 && returnStart < len(typeStr) {
			returnType := strings.TrimSpace(typeStr[returnStart:])
			return ExtractPackageFromType(returnType) // Recursive call for return type
		}
	}

	// For simple types, check if it contains a package qualifier
	if dotIndex := strings.Index(typeStr, "."); dotIndex != -1 {
		return typeStr[:dotIndex]
	}

	return ""
}

// ExtractDependencyName extracts a variable name from a dependency type
func ExtractDependencyName(depType string) string {
	// Remove pointer prefix
	name := strings.TrimPrefix(depType, "*")

	// Handle package-qualified types (e.g., "pkg.Type" -> "type")
	if dotIndex := strings.LastIndex(name, "."); dotIndex != -1 {
		name = name[dotIndex+1:]
	}

	// Keep the original case for field names - Go struct fields are exported (PascalCase)
	return name
}