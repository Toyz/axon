package templates

import (
	"fmt"
	"sort"
	"strings"
)

// ImportManager handles import generation and deduplication
type ImportManager struct {
	moduleImport   string
	standardImports map[string]bool
	packageImports  map[string]string // alias -> path
	userPackages   []string
}

// NewImportManager creates a new import manager
func NewImportManager() *ImportManager {
	return &ImportManager{
		standardImports: make(map[string]bool),
		packageImports:  make(map[string]string),
		userPackages:    make([]string, 0),
	}
}

// SetModuleImport sets the module import path
func (im *ImportManager) SetModuleImport(moduleImport string) {
	im.moduleImport = moduleImport
}

// AddImport adds a standard import
func (im *ImportManager) AddImport(importPath string) {
	if importPath != "" {
		im.standardImports[importPath] = true
	}
}

// AddPackageImport adds a package import with alias
func (im *ImportManager) AddPackageImport(alias, path string) {
	if alias != "" && path != "" {
		im.packageImports[alias] = path
	}
}

// AddUserPackages adds user-defined packages
func (im *ImportManager) AddUserPackages(packages ...string) {
	for _, pkg := range packages {
		if pkg != "" && !im.containsUserPackage(pkg) {
			im.userPackages = append(im.userPackages, pkg)
		}
	}
}

// containsUserPackage checks if a user package is already added
func (im *ImportManager) containsUserPackage(pkg string) bool {
	for _, existing := range im.userPackages {
		if existing == pkg {
			return true
		}
	}
	return false
}

// GenerateImports generates the import section
func (im *ImportManager) GenerateImports() string {
	if im.isEmpty() {
		return ""
	}

	var imports []string

	// Add standard imports (sorted)
	if len(im.standardImports) > 0 {
		var stdImports []string
		for imp := range im.standardImports {
			stdImports = append(stdImports, fmt.Sprintf(`"%s"`, imp))
		}
		sort.Strings(stdImports)
		imports = append(imports, stdImports...)
	}

	// Add package imports with aliases (sorted by alias)
	if len(im.packageImports) > 0 {
		var aliases []string
		for alias := range im.packageImports {
			aliases = append(aliases, alias)
		}
		sort.Strings(aliases)

		for _, alias := range aliases {
			path := im.packageImports[alias]
			imports = append(imports, fmt.Sprintf(`%s "%s"`, alias, path))
		}
	}

	// Add user packages (maintain order)
	for _, pkg := range im.userPackages {
		imports = append(imports, fmt.Sprintf(`"%s"`, pkg))
	}

	if len(imports) == 0 {
		return ""
	}

	// Format as import block
	if len(imports) == 1 {
		return fmt.Sprintf("import %s\n", imports[0])
	}

	var result strings.Builder
	result.WriteString("import (\n")
	for _, imp := range imports {
		result.WriteString(fmt.Sprintf("\t%s\n", imp))
	}
	result.WriteString(")\n")

	return result.String()
}

// GenerateMinimalImports generates imports with module import and user packages
func (im *ImportManager) GenerateMinimalImports() string {
	if im.moduleImport == "" && len(im.userPackages) == 0 {
		return ""
	}

	var imports []string

	// Add module import
	if im.moduleImport != "" {
		imports = append(imports, fmt.Sprintf(`"%s"`, im.moduleImport))
	}

	// Add user packages
	for _, pkg := range im.userPackages {
		imports = append(imports, fmt.Sprintf(`"%s"`, pkg))
	}

	if len(imports) == 0 {
		return ""
	}

	if len(imports) == 1 {
		return fmt.Sprintf("import %s\n", imports[0])
	}

	var result strings.Builder
	result.WriteString("import (\n")
	for _, imp := range imports {
		result.WriteString(fmt.Sprintf("\t%s\n", imp))
	}
	result.WriteString(")\n")

	return result.String()
}

// isEmpty checks if there are any imports to generate
func (im *ImportManager) isEmpty() bool {
	return len(im.standardImports) == 0 &&
		   len(im.packageImports) == 0 &&
		   len(im.userPackages) == 0 &&
		   im.moduleImport == ""
}

// Clone creates a copy of the import manager
func (im *ImportManager) Clone() *ImportManager {
	clone := NewImportManager()
	clone.moduleImport = im.moduleImport

	// Copy standard imports
	for imp := range im.standardImports {
		clone.standardImports[imp] = true
	}

	// Copy package imports
	for alias, path := range im.packageImports {
		clone.packageImports[alias] = path
	}

	// Copy user packages
	clone.userPackages = make([]string, len(im.userPackages))
	copy(clone.userPackages, im.userPackages)

	return clone
}

// Merge merges another import manager into this one
func (im *ImportManager) Merge(other *ImportManager) {
	// Merge module import (other takes precedence)
	if other.moduleImport != "" {
		im.moduleImport = other.moduleImport
	}

	// Merge standard imports
	for imp := range other.standardImports {
		im.standardImports[imp] = true
	}

	// Merge package imports
	for alias, path := range other.packageImports {
		im.packageImports[alias] = path
	}

	// Merge user packages
	im.AddUserPackages(other.userPackages...)
}