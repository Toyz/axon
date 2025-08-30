package utils

import (
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/fatih/color"
)

// DiagnosticLevel represents the level of diagnostic output
type DiagnosticLevel int

const (
	DiagnosticSilent DiagnosticLevel = iota
	DiagnosticError
	DiagnosticWarn
	DiagnosticInfo
	DiagnosticVerbose
	DiagnosticDebug
)

// DiagnosticSystem provides structured, user-friendly output
type DiagnosticSystem struct {
	level     DiagnosticLevel
	useColors bool
	showTime  bool
	output    io.Writer
	errorOut  io.Writer
	indent    int
}

// NewDiagnosticSystem creates a new diagnostic system
func NewDiagnosticSystem(level DiagnosticLevel) *DiagnosticSystem {
	return &DiagnosticSystem{
		level:     level,
		useColors: shouldUseColors(),
		showTime:  level >= DiagnosticVerbose,
		output:    os.Stdout,
		errorOut:  os.Stderr,
		indent:    0,
	}
}

// NewQuietDiagnostics creates a diagnostic system that only shows errors
func NewQuietDiagnostics() *DiagnosticSystem {
	return NewDiagnosticSystem(DiagnosticError)
}

// NewVerboseDiagnostics creates a diagnostic system with full output
func NewVerboseDiagnostics() *DiagnosticSystem {
	return NewDiagnosticSystem(DiagnosticVerbose)
}

// Color constants for terminal output
const (
	ColorReset   = "\033[0m"
	ColorBold    = "\033[1m"
	ColorDim     = "\033[2m"
	ColorRed     = "\033[31m"
	ColorGreen   = "\033[32m"
	ColorYellow  = "\033[33m"
	ColorBlue    = "\033[34m"
	ColorMagenta = "\033[35m"
	ColorCyan    = "\033[36m"
	ColorGray    = "\033[90m"
)

// Error outputs error messages (always shown unless silent)
func (d *DiagnosticSystem) Error(format string, args ...interface{}) {
	if d.level >= DiagnosticError {
		d.writeMessage(d.errorOut, "ERROR", ColorRed, format, args...)
	}
}

// Warn outputs warning messages
func (d *DiagnosticSystem) Warn(format string, args ...interface{}) {
	if d.level >= DiagnosticWarn {
		d.writeMessage(d.output, "WARN", ColorYellow, format, args...)
	}
}

// Info outputs informational messages
func (d *DiagnosticSystem) Info(format string, args ...interface{}) {
	if d.level >= DiagnosticInfo {
		d.writeMessage(d.output, "INFO", ColorBlue, format, args...)
	}
}

// Success outputs success messages with emphasis
func (d *DiagnosticSystem) Success(format string, args ...interface{}) {
	if d.level >= DiagnosticInfo {
		d.writeMessage(d.output, "SUCCESS", ColorGreen, format, args...)
	}
}

// Verbose outputs detailed messages (verbose mode only)
func (d *DiagnosticSystem) Verbose(format string, args ...interface{}) {
	if d.level >= DiagnosticVerbose {
		d.writeMessage(d.output, "VERBOSE", ColorGray, format, args...)
	}
}

// Debug outputs debug messages (highest verbosity)
func (d *DiagnosticSystem) Debug(format string, args ...interface{}) {
	if d.level >= DiagnosticDebug {
		d.writeMessage(d.output, "DEBUG", ColorMagenta, format, args...)
	}
}

// Progress shows progress without a level prefix
func (d *DiagnosticSystem) Progress(format string, args ...interface{}) {
	if d.level >= DiagnosticInfo {
		message := fmt.Sprintf(format, args...)
		fmt.Fprintf(d.output, "✓ %s\n", message)
	}
}



// Section creates a prominent section header
func (d *DiagnosticSystem) Section(title string) {
	if d.level >= DiagnosticInfo {
		fmt.Fprintf(d.output, "%s\n", title)
	}
}

// Subsection creates a subsection header
func (d *DiagnosticSystem) Subsection(title string) {
	if d.level >= DiagnosticInfo {
		fmt.Fprintf(d.output, "\n%s:\n", title)
	}
}

// List outputs a bulleted list item
func (d *DiagnosticSystem) List(format string, args ...interface{}) {
	if d.level >= DiagnosticInfo {
		message := fmt.Sprintf(format, args...)
		fmt.Fprintf(d.output, "- %s\n", message)
	}
}

// Indent increases the indentation level
func (d *DiagnosticSystem) Indent() {
	d.indent++
}

// Unindent decreases the indentation level
func (d *DiagnosticSystem) Unindent() {
	if d.indent > 0 {
		d.indent--
	}
}

// Summary outputs a final summary with statistics
func (d *DiagnosticSystem) Summary(title string, stats map[string]interface{}) {
	if d.level >= DiagnosticInfo {
		fmt.Fprintf(d.output, "\n%s\n", title)
		
		for key, value := range stats {
			fmt.Fprintf(d.output, "   %s: %v\n", key, value)
		}
		fmt.Fprintln(d.output)
	}
}

// Category outputs a category header like [Controllers]
func (d *DiagnosticSystem) Category(title string) {
	if d.level >= DiagnosticInfo {
		fmt.Fprintf(d.output, "\n[%s]\n", title)
	}
}

// AxonHeader outputs the main Axon header
func (d *DiagnosticSystem) AxonHeader(message string) {
	if d.level >= DiagnosticInfo {
		cyan := color.New(color.FgCyan)
		cyan.Printf("Axon: %s\n", message)
	}
}

// SourcePath outputs the source path
func (d *DiagnosticSystem) SourcePath(path string) {
	if d.level >= DiagnosticInfo {
		fmt.Printf("Source Path: %s\n\n", path)
	}
}

// PhaseHeader outputs a phase header
func (d *DiagnosticSystem) PhaseHeader(phase string) {
	if d.level >= DiagnosticInfo {
		blue := color.New(color.FgBlue)
		blue.Printf("%s:\n", phase)
	}
}

// PhaseItem outputs a phase item with checkmark
func (d *DiagnosticSystem) PhaseItem(message string) {
	if d.level >= DiagnosticInfo {
		green := color.New(color.FgGreen)
		green.Print("✓ ")
		fmt.Printf("%s\n", message)
	}
}

// PhaseProgress outputs a phase progress item
func (d *DiagnosticSystem) PhaseProgress(message string) {
	if d.level >= DiagnosticInfo {
		// Special formatting for writing operations
		if strings.Contains(message, "Writing") {
			magenta := color.New(color.FgMagenta)
			magenta.Print("✏ ")
			fmt.Printf("%s\n", message)
		} else {
			fmt.Printf("- %s\n", message)
		}
	}
}

// GenerationComplete outputs the completion message
func (d *DiagnosticSystem) GenerationComplete() {
	if d.level >= DiagnosticInfo {
		fmt.Println()
		green := color.New(color.FgGreen)
		green.Println("Axon: Generation complete!")
	}
}

// writeMessage is the internal message writing function
func (d *DiagnosticSystem) writeMessage(writer io.Writer, level, color, format string, args ...interface{}) {
	message := fmt.Sprintf(format, args...)
	
	var output strings.Builder
	output.WriteString(d.getIndent())
	
	// Add timestamp if enabled
	if d.showTime {
		output.WriteString(time.Now().Format("15:04:05 "))
	}
	
	// Add colored level if colors are enabled
	if d.useColors {
		output.WriteString(fmt.Sprintf("%s[%s]%s ", color, level, ColorReset))
	} else {
		output.WriteString(fmt.Sprintf("[%s] ", level))
	}
	
	// Add the message
	output.WriteString(message)
	output.WriteString("\n")
	
	fmt.Fprint(writer, output.String())
}



// getIndent returns the current indentation string
func (d *DiagnosticSystem) getIndent() string {
	return strings.Repeat("  ", d.indent)
}

// shouldUseColors determines if colors should be used
func shouldUseColors() bool {
	// Check if NO_COLOR is set (standard)
	if os.Getenv("NO_COLOR") != "" {
		return false
	}
	
	// Check if FORCE_COLOR is set
	if os.Getenv("FORCE_COLOR") != "" {
		return true
	}
	
	// Check if we have a terminal
	term := os.Getenv("TERM")
	return term != "" && term != "dumb"
}