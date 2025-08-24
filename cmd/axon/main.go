package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/toyz/axon/internal/cli"
	"github.com/toyz/axon/internal/parser"
)

func main() {
	// Define command-line flags
	var (
		moduleFlag  = flag.String("module", "", "Custom module name for imports (defaults to go.mod module)")
		verboseFlag = flag.Bool("verbose", false, "Enable verbose output and detailed error reporting")
		helpFlag    = flag.Bool("help", false, "Show help information")
	)

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options] <directory-paths...>\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Axon Framework Code Generator\n")
		fmt.Fprintf(os.Stderr, "Recursively scans directories for Go files with %s annotations and generates FX modules.\n\n", parser.AnnotationPrefix)
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nArguments:\n")
		fmt.Fprintf(os.Stderr, "  directory-paths    One or more directories to scan for annotated Go files\n")
		fmt.Fprintf(os.Stderr, "                     Supports Go-style patterns like './...' for recursive scanning\n")
		fmt.Fprintf(os.Stderr, "\nDirectory Patterns:\n")
		fmt.Fprintf(os.Stderr, "  ./...              Scan current directory and all subdirectories recursively\n")
		fmt.Fprintf(os.Stderr, "  ./internal/...     Scan internal directory and all its subdirectories\n")
		fmt.Fprintf(os.Stderr, "  ./pkg/controllers  Scan only the specific directory (no recursion)\n")
		fmt.Fprintf(os.Stderr, "\nExamples:\n")
		fmt.Fprintf(os.Stderr, "  %s ./...                                       # Scan everything recursively\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s ./internal/...                             # Scan internal directory recursively\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s ./internal/controllers ./internal/services # Scan specific directories\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s --module github.com/myorg/myapp ./...      # Specify custom module name\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s --verbose ./internal/...                   # Enable detailed output\n", os.Args[0])
	}

	flag.Parse()

	// Show help if requested
	if *helpFlag {
		flag.Usage()
		os.Exit(0)
	}

	// Get directory paths from remaining arguments
	directories := flag.Args()
	if len(directories) == 0 {
		fmt.Fprintf(os.Stderr, "Error: At least one directory path is required\n\n")
		flag.Usage()
		os.Exit(1)
	}

	// Validate directory paths (handle Go-style patterns like ./...)
	for _, dir := range directories {
		// Handle Go-style recursive patterns
		if strings.HasSuffix(dir, "/...") {
			baseDir := strings.TrimSuffix(dir, "/...")
			if baseDir == "" {
				baseDir = "."
			}
			// Validate the base directory exists
			if _, err := os.Stat(baseDir); os.IsNotExist(err) {
				fmt.Fprintf(os.Stderr, "Error: Base directory does not exist: %s (from pattern %s)\n", baseDir, dir)
				os.Exit(1)
			}
		} else {
			// Validate regular directory paths
			if _, err := os.Stat(dir); os.IsNotExist(err) {
				fmt.Fprintf(os.Stderr, "Error: Directory does not exist: %s\n", dir)
				os.Exit(1)
			}
		}
	}

	// Create CLI configuration
	config := cli.Config{
		Directories: directories,
		ModuleName:  *moduleFlag,
		Verbose:     *verboseFlag,
	}

	// Run the generator
	generator := cli.NewGenerator(*verboseFlag)
	if err := generator.Run(config); err != nil {
		// Use diagnostic reporter for better error output
		reporter := cli.NewDiagnosticReporter(*verboseFlag)
		reporter.ReportError(err)
		os.Exit(1)
	}

	// Report success with summary
	generator.ReportSuccess()
}