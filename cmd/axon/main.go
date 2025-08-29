package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

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

	// Show startup banner
	diagnostics.Section("Axon Code Generator")
	
	// Handle clean operation
	if *cleanFlag {
		diagnostics.Info("Starting cleanup operation...")
		diagnostics.StartProgress("Cleaning generated files")
		
		cleaner := cli.NewCleaner()
		err := cleaner.CleanGeneratedFiles(args)
		if err != nil {
			diagnostics.EndProgress(false, "")
			diagnostics.Error("Clean operation failed: %v", err)
			os.Exit(1)
		}
		
		diagnostics.EndProgress(true, "")
		diagnostics.Success("All autogen_module.go files have been removed")
		return
	}

	// Show configuration
	if *verboseFlag {
		diagnostics.Subsection("Configuration")
		diagnostics.List("Target directories: %s", strings.Join(args, ", "))
		if *moduleFlag != "" {
			diagnostics.List("Custom module: %s", *moduleFlag)
		}
		diagnostics.List("Verbose mode: enabled")
	}

	// Create and configure generator
	diagnostics.StartProgress("Initializing generator")
	generator := cli.NewGeneratorWithDiagnostics(*verboseFlag, diagnostics)
	
	if *moduleFlag != "" {
		generator.SetCustomModule(*moduleFlag)
		diagnostics.Debug("Using custom module: %s", *moduleFlag)
	}
	diagnostics.EndProgress(true, "")

	// Run the generation process
	diagnostics.Subsection("Code Generation")
	diagnostics.StartProgress("Processing directories")
	
	err := generator.Generate(args)
	if err != nil {
		diagnostics.EndProgress(false, "")
		diagnostics.Error("Generation failed: %v", err)
		os.Exit(1)
	}
	
	diagnostics.EndProgress(true, "")

	// Show final summary
	summary := generator.GetSummary()
	stats := map[string]interface{}{
		"Packages processed":   summary.PackagesProcessed,
		"Modules generated":    len(summary.GeneratedFiles),
		"Controllers found":    summary.ControllersFound,
		"Services found":       summary.ServicesFound,
		"Middlewares found":    summary.MiddlewaresFound,
		"Custom parsers found": summary.ParsersDiscovered,
	}
	
	diagnostics.Summary("Generation Complete!", stats)
	
	// Show generated files in verbose mode
	if *verboseFlag && len(summary.GeneratedFiles) > 0 {
		diagnostics.Subsection("Generated Files")
		for _, file := range summary.GeneratedFiles {
			diagnostics.List("%s", file)
		}
	}
	
	diagnostics.Success("Your Axon application is ready to use!")
}