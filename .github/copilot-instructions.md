# Axon Framework

Axon is a Go annotation-driven web framework that uses code generation to create dependency injection modules, HTTP route handlers, and middleware chains. It leverages Uber FX for dependency injection and Echo for HTTP routing.

**CRITICAL**: Always reference these instructions first and fallback to search or bash commands only when you encounter unexpected information that does not match the info here.

## Working Effectively

### Bootstrap, Build and Test
- `go mod tidy` - Download and verify dependencies (5-10 seconds on first run)
- `go build -v ./cmd/axon` - Build the Axon CLI (first build: ~15s, subsequent: under 1s). NEVER CANCEL. Set timeout to 30+ seconds.
- `./axon --help` - Verify CLI works correctly
- `go test -v ./cmd/... ./pkg/... ./internal/...` - Run full test suite (~4 seconds). NEVER CANCEL. Set timeout to 30+ seconds.

### Code Generation Workflow
- `./axon ./internal/...` - Generate autogen_module.go files for all packages recursively (under 1 second)
- `./axon ./examples/complete-app/internal/...` - Generate code for example application
- `./axon --clean ./...` - Clean all generated autogen_module.go files
- `./axon --verbose ./internal/...` - Enable detailed output for debugging

### Building Applications
- After code generation: `go build -v .` in application directory (under 1 second)
- Example app build: `cd examples/complete-app && go build -v .` (first build: ~1.6s, subsequent: under 1s) 

### Running Applications
- Example app: `cd examples/complete-app && PORT=8080 ./complete-app`
- Health check: `curl http://localhost:8080/health`
- User endpoints: `curl http://localhost:8080/users`

## Validation

### ALWAYS Run These Commands Before Completing Changes:
- `go fmt ./...` - Format code (should show no output when code is properly formatted)
- `go vet ./...` - Static analysis (should show no errors)
- `~/go/bin/goimports -w .` - Fix imports (install with `go install golang.org/x/tools/cmd/goimports@latest`)
- `go test -v ./cmd/... ./pkg/... ./internal/...` - Ensure all tests pass. NEVER CANCEL. 30+ second timeout.
- `go build -v ./cmd/axon && ./axon ./examples/complete-app/internal/... && cd examples/complete-app && go build -v .` - Full end-to-end validation

### Manual Testing Scenarios
After making changes, ALWAYS validate with these complete scenarios:
1. **Code Generation Flow**: Generate code for example app, build it, run it, test HTTP endpoints
2. **CLI Testing**: Build CLI, test all flags (--help, --clean, --verbose), verify error handling
3. **Application Testing**: Start example app and test at least 3 endpoints: /health, /users, /users/1

### Build Timing Expectations
- **NEVER CANCEL builds or tests** - they may appear to hang but are working normally
- CLI build: First build ~15s, subsequent builds under 1s (set 30+ second timeout)
- Test suite: ~4 seconds (set 30+ second timeout) 
- Code generation: under 0.1 seconds
- Application build: First build ~1.6s, subsequent builds under 1s
- Dependency download: 5-10 seconds on first run

## Common Tasks

### Project Structure
```
/home/runner/work/axon/axon/
├── cmd/axon/           # CLI application entry point
├── internal/           # Internal framework packages
│   ├── annotations/    # Annotation parsing and validation
│   ├── cli/           # CLI implementation
│   ├── generator/     # Code generation logic
│   ├── models/        # Data structures
│   ├── parser/        # AST parsing
│   ├── registry/      # Component registries
│   └── templates/     # Code generation templates
├── pkg/axon/          # Public API packages
├── examples/          # Example applications
│   ├── complete-app/  # Full-featured example
│   └── simple-app/    # Minimal example
├── .github/           # GitHub workflows and config
├── go.mod             # Go module definition
└── README.md          # Primary documentation
```

### Key Files and Their Purpose
- `cmd/axon/main.go` - CLI entry point with flag parsing
- `internal/cli/generator.go` - Main code generation orchestration
- `internal/templates/templates.go` - Code generation templates
- `pkg/axon/response.go` - Public response helpers
- `examples/complete-app/` - Comprehensive example showing all features
- `.github/workflows/test.yml` - CI/CD pipeline (tests, build, integration)

### Annotation Types
The framework supports these annotation types in Go comments:
- `//axon::controller` - Marks HTTP controllers
- `//axon::route METHOD /path` - Defines HTTP routes
- `//axon::core` - Marks core services
- `//axon::inject` - Dependency injection field
- `//axon::init` - Initialization field
- `//axon::middleware Name` - Named middleware
- `//axon::interface` - Interface generation
- `//axon::parser Type` - Custom parameter parsers

### Development Workflow
1. Write Go code with axon:: annotations
2. Run `./axon ./internal/...` to generate autogen_module.go files
3. Build application with `go build -v .`
4. Run and test application functionality
5. Format code with `go fmt ./...`
6. Validate with `go vet ./...` and test suite

### CI/CD Pipeline Understanding
The `.github/workflows/test.yml` includes:
- Unit tests across all packages
- CLI build validation
- Example app code generation and build
- Integration tests with running application
- End-to-end API testing with curl

### Troubleshooting Common Issues
- **Build fails**: Run `go mod tidy` first, check Go version (requires 1.25+)
- **Code generation fails**: Use `--verbose` flag for detailed error output
- **Tests fail**: Ensure all autogen_module.go files are cleaned with `--clean` first
- **Import errors**: Run `~/go/bin/goimports -w .` to fix imports
- **Application won't start**: Check that code generation completed successfully

### Repository Health Indicators
When repository is healthy, you should see:
- All tests pass (100% success rate)
- CLI builds without warnings
- Example app generates, builds, and runs successfully
- HTTP endpoints respond correctly (health, users, products)
- No linting errors from go vet or goimports

ALWAYS build and exercise your changes through the complete workflow before marking work complete.