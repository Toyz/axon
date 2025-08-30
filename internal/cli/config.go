package cli

// Config holds the configuration for the CLI generator
type Config struct {
	// Directories is the list of directories to scan for annotated Go files
	Directories []string

	// ModuleName is the custom module name for imports
	// If empty, will be determined from go.mod file
	ModuleName string

	// Verbose enables detailed logging and error reporting
	Verbose bool
}
