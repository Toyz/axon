package utils

import (
	"bytes"
	"fmt"
	"go/format"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
)

// FormatGoCode formats Go source code using the same logic as gofmt
func FormatGoCode(source []byte) ([]byte, error) {
	return format.Source(source)
}

// FormatGoCodeString formats Go source code from a string and returns a string
func FormatGoCodeString(source string) (string, error) {
	formatted, err := format.Source([]byte(source))
	if err != nil {
		// If formatting fails, try to parse to see if it's valid Go
		fset := token.NewFileSet()
		_, parseErr := parser.ParseFile(fset, "", source, parser.ParseComments)
		if parseErr != nil {
			return source, fmt.Errorf("invalid Go syntax: %w (format error: %v)", parseErr, err)
		}
		// If parsing works but formatting doesn't, return the original
		return source, err
	}
	return string(formatted), nil
}

// FormatGoFileWithGofmt formats a Go file using the gofmt command as fallback
func FormatGoFileWithGofmt(filename string) error {
	cmd := exec.Command("gofmt", "-w", filename)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("gofmt failed for %s: %w (stderr: %s)", filename, err, stderr.String())
	}

	return nil
}

// FormatAndWriteGoFile formats Go code and writes it to a file with fallback options
func FormatAndWriteGoFile(filename string, code string) error {
	// Try to format first
	formatted, err := FormatGoCodeString(code)
	if err != nil {
		// If formatting fails, write the unformatted code first
		if writeErr := os.WriteFile(filename, []byte(code), 0644); writeErr != nil {
			return fmt.Errorf("failed to write unformatted code to %s: %w (format error: %v)", filename, writeErr, err)
		}

		// Try to format with gofmt as fallback
		if gofmtErr := FormatGoFileWithGofmt(filename); gofmtErr != nil {
			return fmt.Errorf("failed to format %s with gofmt: %w (original format error: %v)", filename, gofmtErr, err)
		}

		return nil
	}

	// Write the formatted code
	return os.WriteFile(filename, []byte(formatted), 0644)
}

// ValidateGoCode checks if the provided code is valid Go syntax
func ValidateGoCode(code string) error {
	fset := token.NewFileSet()
	_, err := parser.ParseFile(fset, "", code, parser.ParseComments)
	return err
}
