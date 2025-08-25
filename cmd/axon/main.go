package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/toyz/axon/internal/cli"
	"github.com/toyz/axon/internal/parser"
)

func main() {
	// Define command-line flags
	var (
		moduleFlag  = flag.String("module", "", "Custom module name for imports (defaults to go.mod module)")
		verboseFlag = flag.Bool("verbose", false, "Enable verbose output and detailed error reporting")
		cleanFlag   = flag.Bool("clean", false, "Delete all autogen_module.go files from the specified directories")
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
		fmt.Fprintf(os.Stderr, "  %s --clean ./...                              # Delete all autogen_module.go files\n", os.Args[0])
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

	// Handle clean command
	if *cleanFlag {
		if err := cleanAutogenFiles(directories, *verboseFlag); err != nil {
			fmt.Fprintf(os.Stderr, "Error during cleanup: %v\n", err)
			os.Exit(1)
		}
		return
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

// cleanAutogenFiles removes all autogen_module.go files from the specified directories
func cleanAutogenFiles(directories []string, verbose bool) error {
	var deletedFiles []string
	var errors []error

	for _, dir := range directories {
		if strings.HasSuffix(dir, "/...") {
			// Handle recursive patterns
			baseDir := strings.TrimSuffix(dir, "/...")
			if baseDir == "" {
				baseDir = "."
			}
			
			files, err := findAutogenFilesRecursive(baseDir)
			if err != nil {
				errors = append(errors, fmt.Errorf("failed to scan directory %s: %w", baseDir, err))
				continue
			}
			
			for _, file := range files {
				if err := os.Remove(file); err != nil {
					errors = append(errors, fmt.Errorf("failed to delete %s: %w", file, err))
				} else {
					deletedFiles = append(deletedFiles, file)
					if verbose {
						fmt.Printf("Deleted: %s\n", file)
					}
				}
			}
		} else {
			// Handle specific directory
			autogenFile := filepath.Join(dir, "autogen_module.go")
			if _, err := os.Stat(autogenFile); err == nil {
				if err := os.Remove(autogenFile); err != nil {
					errors = append(errors, fmt.Errorf("failed to delete %s: %w", autogenFile, err))
				} else {
					deletedFiles = append(deletedFiles, autogenFile)
					if verbose {
						fmt.Printf("Deleted: %s\n", autogenFile)
					}
				}
			}
		}
	}

	// Report results
	if len(deletedFiles) > 0 {
		fmt.Printf("Successfully deleted %d autogen_module.go file(s):\n", len(deletedFiles))
		if !verbose {
			for _, file := range deletedFiles {
				fmt.Printf("  - %s\n", file)
			}
		}
	} else {
		fmt.Println("No autogen_module.go files found to delete.")
	}

	if len(errors) > 0 {
		fmt.Fprintf(os.Stderr, "\nEncountered %d error(s) during cleanup:\n", len(errors))
		for _, err := range errors {
			fmt.Fprintf(os.Stderr, "  - %v\n", err)
		}
		return fmt.Errorf("cleanup completed with %d error(s)", len(errors))
	}

	return nil
}

// findAutogenFilesRecursive recursively finds all autogen_module.go files in a directory
func findAutogenFilesRecursive(rootDir string) ([]string, error) {
	var autogenFiles []string
	
	// Convert to absolute path for consistency
	absRootDir, err := filepath.Abs(rootDir)
	if err != nil {
		return nil, fmt.Errorf("failed to get absolute path for %s: %w", rootDir, err)
	}
	
	err = filepath.Walk(absRootDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		
		// Skip hidden directories and files
		if strings.HasPrefix(info.Name(), ".") {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		
		// Skip vendor directory
		if info.IsDir() && info.Name() == "vendor" {
			return filepath.SkipDir
		}
		
		// Check if this is an autogen_module.go file
		if !info.IsDir() && info.Name() == "autogen_module.go" {
			autogenFiles = append(autogenFiles, path)
		}
		
		return nil
	})
	
	return autogenFiles, err
}