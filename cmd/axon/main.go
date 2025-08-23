package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/toyz/axon/internal/cli"
	"github.com/toyz/axon/internal/parser"
)

func main() {
	// Define command-line flags
	var (
		moduleFlag = flag.String("module", "", "Custom module name for imports (defaults to go.mod module)")
		helpFlag   = flag.Bool("help", false, "Show help information")
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
		fmt.Fprintf(os.Stderr, "\nExamples:\n")
		fmt.Fprintf(os.Stderr, "  %s ./internal/controllers ./internal/services\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s ./...  # Scan all subdirectories recursively\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s ./internal/controllers ./internal/services\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s --module github.com/myorg/myapp ./internal\n", os.Args[0])
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

	// Validate directory paths
	for _, dir := range directories {
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			fmt.Fprintf(os.Stderr, "Error: Directory does not exist: %s\n", dir)
			os.Exit(1)
		}
	}

	// Create CLI configuration
	config := cli.Config{
		Directories: directories,
		ModuleName:  *moduleFlag,
	}

	// Run the generator
	generator := cli.NewGenerator()
	if err := generator.Run(config); err != nil {
		log.Fatalf("Generation failed: %v", err)
	}

	fmt.Println("Code generation completed successfully!")
}