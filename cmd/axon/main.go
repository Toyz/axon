package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/toyz/axon/internal/cli"
	"github.com/toyz/axon/internal/utils"
)

func main() {
	// Define command-line flags
	var (
		moduleFlag  = flag.String("module", "", "Custom module name for imports (defaults to go.mod module)")
		verboseFlag = flag.Bool("verbose", false, "Enable verbose output and detailed error reporting")
		quietFlag   = flag.Bool("quiet", false, "Only show errors and final results")
		cleanFlag   = flag.Bool("clean", false, "Delete all autogen_module.go files from the specified directories")
		helpFlag    = flag.Bool("help", false, "Show help information")
	)

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options] <directory-paths...>\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Axon Framework Code Generator\n")
		fmt.Fprintf(os.Stderr, "Recursively scans directories for Go files with axon:: annotations and generates FX modules.\n\n")
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
		fmt.Fprintf(os.Stderr, "  %s --quiet ./...                              # Minimal output\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s --clean ./...                              # Delete all autogen_module.go files\n", os.Args[0])
	}

	flag.Parse()

	// Show help if requested
	if *helpFlag {
		flag.Usage()
		os.Exit(0)
	}

	// Validate arguments
	args := flag.Args()
	if len(args) == 0 {
		fmt.Fprintf(os.Stderr, "Error: At least one directory path is required\n\n")
		flag.Usage()
		os.Exit(1)
	}

	// Create diagnostic system based on flags
	var diagnostics *utils.DiagnosticSystem
	if *quietFlag {
		diagnostics = utils.NewQuietDiagnostics()
	} else if *verboseFlag {
		diagnostics = utils.NewVerboseDiagnostics()
	} else {
		diagnostics = utils.NewDiagnosticSystem(utils.DiagnosticInfo)
	}

	// Clean startup - no banner needed, generator will handle output
	
	// Handle clean operation
	if *cleanFlag {
		diagnostics.Info("Starting cleanup operation...")
		
		cleaner := cli.NewCleaner()
		err := cleaner.CleanGeneratedFiles(args)
		if err != nil {
			diagnostics.Error("Clean operation failed: %v", err)
			os.Exit(1)
		}
		
		diagnostics.Success("All autogen_module.go files have been removed")
		return
	}

	// Create and configure generator (no extra output)
	generator := cli.NewGeneratorWithDiagnostics(*verboseFlag, diagnostics)
	
	if *moduleFlag != "" {
		generator.SetCustomModule(*moduleFlag)
	}
	
	// Run the generation process
	err := generator.Generate(args)
	if err != nil {
		diagnostics.Error("Generation failed: %v", err)
		os.Exit(1)
	}

	// Generator handles its own completion message
}