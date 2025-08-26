package templates

import (
	"strings"
	"testing"
)

func TestNewImportManager(t *testing.T) {
	im := NewImportManager()

	if im == nil {
		t.Fatal("NewImportManager returned nil")
	}

	if im.sourceImports == nil {
		t.Error("sourceImports map not initialized")
	}

	if im.knownTypes == nil {
		t.Error("knownTypes map not initialized")
	}

	if im.packageResolver == nil {
		t.Error("packageResolver not initialized")
	}
}

func TestDetectTypeImports(t *testing.T) {
	im := NewImportManager()

	testCases := []struct {
		name     string
		code     string
		expected []string // expected import paths
	}{
		{
			name:     "context import",
			code:     "func Start(ctx context.Context) error { return nil }",
			expected: []string{"context"},
		},
		{
			name:     "multiple imports",
			code:     "func Handler(ctx context.Context) error { fmt.Println(\"test\"); return nil }",
			expected: []string{"context", "fmt"},
		},
		{
			name:     "slog import",
			code:     "logger := slog.New(handler)",
			expected: []string{"log/slog"},
		},
		{
			name:     "no imports needed",
			code:     "func simple() { return }",
			expected: []string{},
		},
		{
			name:     "http server",
			code:     "server := &http.Server{Addr: \":8080\"}",
			expected: []string{"net/http"},
		},
		{
			name:     "json marshal",
			code:     "data, err := json.Marshal(obj)",
			expected: []string{"encoding/json"},
		},
		{
			name:     "time operations",
			code:     "now := time.Now(); duration := time.Second",
			expected: []string{"time"},
		},
		{
			name:     "sync mutex",
			code:     "var mu sync.Mutex; mu.Lock()",
			expected: []string{"sync"},
		},
		{
			name:     "bytes buffer",
			code:     "buf := &bytes.Buffer{}",
			expected: []string{"bytes"},
		},
		{
			name:     "regexp compile",
			code:     "re := regexp.MustCompile(`\\d+`)",
			expected: []string{"regexp"},
		},
		{
			name:     "sql database",
			code:     "db, err := sql.Open(\"mysql\", dsn)",
			expected: []string{"database/sql"},
		},
		{
			name:     "crypto operations",
			code:     "hash := sha256.Sum256(data); key, _ := rsa.GenerateKey(rand.Reader, 2048)",
			expected: []string{"crypto/sha256", "crypto/rsa", "crypto/rand", "math/rand"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			imports := im.detectTypeImports(tc.code)

			// Convert to map for easier comparison
			importPaths := make(map[string]bool)
			for _, imp := range imports {
				importPaths[imp.Path] = true
			}

			// Check expected imports are present
			for _, expectedPath := range tc.expected {
				if !importPaths[expectedPath] {
					t.Errorf("Expected import %s not found in detected imports", expectedPath)
				}
			}

			// Check no unexpected imports
			if len(imports) != len(tc.expected) {
				t.Errorf("Expected %d imports, got %d. Got: %v", len(tc.expected), len(imports), importPaths)
			}
		})
	}
}

func TestDetectFrameworkImports(t *testing.T) {
	im := NewImportManager()

	testCases := []struct {
		name     string
		code     string
		expected []string
	}{
		{
			name:     "fx import",
			code:     "fx.Provide(NewService)",
			expected: []string{"go.uber.org/fx"},
		},
		{
			name:     "fxevent import",
			code:     "func (l *logger) LogEvent(event fxevent.Event) {}",
			expected: []string{"go.uber.org/fx/fxevent"},
		},
		{
			name:     "axon import",
			code:     "axon.DefaultRouteRegistry.RegisterRoute(info)",
			expected: []string{"github.com/toyz/axon/pkg/axon"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			imports := im.detectFrameworkImports(tc.code)

			importPaths := make(map[string]bool)
			for _, imp := range imports {
				importPaths[imp.Path] = true
			}

			for _, expectedPath := range tc.expected {
				if !importPaths[expectedPath] {
					t.Errorf("Expected import %s not found", expectedPath)
				}
			}
		})
	}
}

func TestGenerateImportBlock(t *testing.T) {
	im := NewImportManager()

	imports := []Import{
		{Path: "context"},
		{Path: "fmt"},
		{Path: "go.uber.org/fx"},
		{Path: "github.com/toyz/axon/pkg/axon"},
		{Path: "github.com/user/project/services"},
	}

	// Set up a mock resolver to identify local packages
	im.packageResolver = &PackageResolver{
		ModulePath: "github.com/user/project",
	}

	result := im.GenerateImportBlock(imports)

	// Check that import block is properly formatted
	if !strings.Contains(result, "import (") {
		t.Error("Import block should start with 'import ('")
	}

	if !strings.Contains(result, "\"context\"") {
		t.Error("Should contain context import")
	}

	if !strings.Contains(result, "\"fmt\"") {
		t.Error("Should contain fmt import")
	}

	if !strings.Contains(result, "\"go.uber.org/fx\"") {
		t.Error("Should contain fx import")
	}

	// Check grouping - standard library should come first
	contextPos := strings.Index(result, "\"context\"")
	fxPos := strings.Index(result, "\"go.uber.org/fx\"")

	if contextPos > fxPos {
		t.Error("Standard library imports should come before third-party imports")
	}
}

func TestFilterUnusedImports(t *testing.T) {
	im := NewImportManager()

	imports := []Import{
		{Path: "context"},
		{Path: "fmt"},
		{Path: "unused/package"},
	}

	code := "func Handler(ctx context.Context) { fmt.Println(\"test\") }"

	filtered := im.FilterUnusedImports(imports, code)

	// Should keep context and fmt, remove unused/package
	expectedPaths := map[string]bool{
		"context": true,
		"fmt":     true,
	}

	if len(filtered) != 2 {
		t.Errorf("Expected 2 imports after filtering, got %d", len(filtered))
	}

	for _, imp := range filtered {
		if !expectedPaths[imp.Path] {
			t.Errorf("Unexpected import %s in filtered results", imp.Path)
		}
	}
}

func TestIsStandardLibrary(t *testing.T) {
	im := NewImportManager()

	testCases := []struct {
		path     string
		expected bool
	}{
		// Core standard library
		{"context", true},
		{"fmt", true},
		{"log/slog", true},
		{"net/http", true},
		{"encoding/json", true},
		{"database/sql", true},
		{"crypto/sha256", true},
		{"crypto/rsa", true},
		{"sync/atomic", true},
		{"path/filepath", true},
		{"archive/tar", true},
		{"compress/gzip", true},
		{"image/jpeg", true},
		{"text/template", true},
		{"html/template", true},
		{"go/ast", true},
		{"go/parser", true},
		{"testing/quick", true},
		{"runtime/debug", true},
		{"syscall/js", true},

		// Extended standard library
		{"golang.org/x/crypto", true},
		{"golang.org/x/net", true},

		// Third-party packages
		{"github.com/user/project", false},
		{"go.uber.org/fx", false},
		{"github.com/google/uuid", false},
		{"github.com/labstack/echo/v4", false},
		{"gopkg.in/yaml.v3", false},
	}

	for _, tc := range testCases {
		result := im.isStandardLibrary(tc.path)
		if result != tc.expected {
			t.Errorf("isStandardLibrary(%s) = %v, expected %v", tc.path, result, tc.expected)
		}
	}
}

func TestPackageResolver(t *testing.T) {
	resolver := &PackageResolver{
		ModulePath: "github.com/user/project",
		PackageMap: make(map[string]string),
	}

	testCases := []struct {
		packageDir string
		expected   string
	}{
		{"services", "github.com/user/project/services"},
		{"internal/config", "github.com/user/project/internal/config"},
		{"./controllers", "github.com/user/project/controllers"},
		{".", "github.com/user/project"},
	}

	for _, tc := range testCases {
		result, err := resolver.ResolvePackagePath(tc.packageDir)
		if err != nil {
			t.Errorf("ResolvePackagePath(%s) returned error: %v", tc.packageDir, err)
			continue
		}

		if result != tc.expected {
			t.Errorf("ResolvePackagePath(%s) = %s, expected %s", tc.packageDir, result, tc.expected)
		}
	}
}
