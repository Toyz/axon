package templates

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// Import represents a Go import statement
type Import struct {
	Path  string // import path (e.g., "context", "github.com/user/project/services")
	Alias string // import alias (empty if no alias)
}

// ImportManager manages import detection and generation for templates
type ImportManager struct {
	sourceImports   map[string][]Import  // imports from source files (filename -> imports)
	knownTypes      map[string]string    // type -> import path mapping
	packageResolver *PackageResolver     // for dynamic package path resolution
}

// PackageResolver resolves package paths dynamically based on project structure
type PackageResolver struct {
	ModuleRoot string            // Root directory of the Go module
	ModulePath string            // Module path from go.mod
	PackageMap map[string]string // package name -> full import path cache
}

// NewImportManager creates a new ImportManager
func NewImportManager() *ImportManager {
	return &ImportManager{
		sourceImports: make(map[string][]Import),
		knownTypes:    make(map[string]string),
		packageResolver: &PackageResolver{
			PackageMap: make(map[string]string),
		},
	}
}

// NewImportManagerWithResolver creates a new ImportManager with a PackageResolver
func NewImportManagerWithResolver(resolver *PackageResolver) *ImportManager {
	return &ImportManager{
		sourceImports:   make(map[string][]Import),
		knownTypes:      make(map[string]string),
		packageResolver: resolver,
	}
}

// NewPackageResolver creates a new PackageResolver by detecting module information
func NewPackageResolver(projectRoot string) (*PackageResolver, error) {
	resolver := &PackageResolver{
		PackageMap: make(map[string]string),
	}
	
	// Find go.mod file to determine module root and path
	moduleRoot, modulePath, err := findModuleInfo(projectRoot)
	if err != nil {
		return nil, fmt.Errorf("failed to find module info: %w", err)
	}
	
	resolver.ModuleRoot = moduleRoot
	resolver.ModulePath = modulePath
	
	return resolver, nil
}

// AddSourceImports adds imports from a source file to the manager
func (im *ImportManager) AddSourceImports(filename string, imports []Import) {
	im.sourceImports[filename] = imports
}

// ExtractImportsFromAST extracts import statements from an AST file
func (im *ImportManager) ExtractImportsFromAST(file *ast.File) []Import {
	var imports []Import
	
	for _, imp := range file.Imports {
		importPath := strings.Trim(imp.Path.Value, `"`)
		alias := ""
		
		if imp.Name != nil {
			alias = imp.Name.Name
		}
		
		imports = append(imports, Import{
			Path:  importPath,
			Alias: alias,
		})
	}
	
	return imports
}

// GetRequiredImports analyzes generated code and returns required imports
func (im *ImportManager) GetRequiredImports(generatedCode string) []Import {
	var requiredImports []Import
	importSet := make(map[string]Import) // use map to avoid duplicates
	
	// Add imports based on known type patterns in the code
	typeImports := im.detectTypeImports(generatedCode)
	for _, imp := range typeImports {
		key := imp.Path + "|" + imp.Alias
		importSet[key] = imp
	}
	
	// Add framework imports if needed
	frameworkImports := im.detectFrameworkImports(generatedCode)
	for _, imp := range frameworkImports {
		key := imp.Path + "|" + imp.Alias
		importSet[key] = imp
	}
	
	// Convert map back to slice
	for _, imp := range importSet {
		requiredImports = append(requiredImports, imp)
	}
	
	return requiredImports
}

// detectTypeImports detects imports needed for specific types in the code
func (im *ImportManager) detectTypeImports(code string) []Import {
	var imports []Import
	importSet := make(map[string]bool) // use set to avoid duplicates
	
	// Comprehensive Go standard library patterns (ordered by specificity)
	typePatterns := []struct {
		pattern string
		importPath string
	}{
		// Specific types first (most specific patterns)
		{`\bcontext\.Context\b`, "context"},
		{`\bcontext\.CancelFunc\b`, "context"},
		{`\btime\.Time\b`, "time"},
		{`\btime\.Duration\b`, "time"},
		{`\btime\.Location\b`, "time"},
		{`\btime\.Timer\b`, "time"},
		{`\btime\.Ticker\b`, "time"},
		{`\burl\.URL\b`, "net/url"},
		{`\burl\.Values\b`, "net/url"},
		{`\bhttp\.Request\b`, "net/http"},
		{`\bhttp\.Response\b`, "net/http"},
		{`\bhttp\.ResponseWriter\b`, "net/http"},
		{`\bhttp\.Handler\b`, "net/http"},
		{`\bhttp\.HandlerFunc\b`, "net/http"},
		{`\bhttp\.Client\b`, "net/http"},
		{`\bhttp\.Server\b`, "net/http"},
		{`\bsql\.DB\b`, "database/sql"},
		{`\bsql\.Tx\b`, "database/sql"},
		{`\bsql\.Rows\b`, "database/sql"},
		{`\bsql\.Row\b`, "database/sql"},
		{`\bsql\.Result\b`, "database/sql"},
		{`\bsql\.Stmt\b`, "database/sql"},
		{`\bjson\.Encoder\b`, "encoding/json"},
		{`\bjson\.Decoder\b`, "encoding/json"},
		{`\bjson\.RawMessage\b`, "encoding/json"},
		{`\bxml\.Encoder\b`, "encoding/xml"},
		{`\bxml\.Decoder\b`, "encoding/xml"},
		{`\bregexp\.Regexp\b`, "regexp"},
		{`\bsync\.Mutex\b`, "sync"},
		{`\bsync\.RWMutex\b`, "sync"},
		{`\bsync\.WaitGroup\b`, "sync"},
		{`\bsync\.Once\b`, "sync"},
		{`\bsync\.Cond\b`, "sync"},
		{`\bsync\.Pool\b`, "sync"},
		{`\batomic\.Value\b`, "sync/atomic"},
		{`\bbytes\.Buffer\b`, "bytes"},
		{`\bbytes\.Reader\b`, "bytes"},
		{`\bstrings\.Builder\b`, "strings"},
		{`\bstrings\.Reader\b`, "strings"},
		{`\bstrings\.Replacer\b`, "strings"},
		{`\bio\.Reader\b`, "io"},
		{`\bio\.Writer\b`, "io"},
		{`\bio\.ReadWriter\b`, "io"},
		{`\bio\.ReadCloser\b`, "io"},
		{`\bio\.WriteCloser\b`, "io"},
		{`\bio\.ReadWriteCloser\b`, "io"},
		{`\bbufio\.Reader\b`, "bufio"},
		{`\bbufio\.Writer\b`, "bufio"},
		{`\bbufio\.Scanner\b`, "bufio"},
		{`\bos\.File\b`, "os"},
		{`\bos\.FileInfo\b`, "os"},
		{`\bos\.FileMode\b`, "os"},
		{`\bos\.Process\b`, "os"},
		{`\bos\.ProcessState\b`, "os"},
		{`\bpath\.filepath\b`, "path/filepath"},
		{`\bfilepath\.WalkFunc\b`, "path/filepath"},
		{`\bnet\.Conn\b`, "net"},
		{`\bnet\.Listener\b`, "net"},
		{`\bnet\.Addr\b`, "net"},
		{`\bnet\.IP\b`, "net"},
		{`\bnet\.IPNet\b`, "net"},
		{`\btcp\.Conn\b`, "net"},
		{`\budp\.Conn\b`, "net"},
		{`\btls\.Config\b`, "crypto/tls"},
		{`\btls\.Conn\b`, "crypto/tls"},
		{`\brsa\.PrivateKey\b`, "crypto/rsa"},
		{`\brsa\.PublicKey\b`, "crypto/rsa"},
		{`\becdsa\.PrivateKey\b`, "crypto/ecdsa"},
		{`\becdsa\.PublicKey\b`, "crypto/ecdsa"},
		{`\bx509\.Certificate\b`, "crypto/x509"},
		{`\bmd5\.Sum\b`, "crypto/md5"},
		{`\bsha1\.Sum\b`, "crypto/sha1"},
		{`\bsha256\.Sum256\b`, "crypto/sha256"},
		{`\bsha512\.Sum512\b`, "crypto/sha512"},
		{`\bhmac\.New\b`, "crypto/hmac"},
		{`\brand\.Reader\b`, "crypto/rand"},
		{`\brand\.Rand\b`, "math/rand"},
		{`\brand\.Source\b`, "math/rand"},
		{`\bbig\.Int\b`, "math/big"},
		{`\bbig\.Float\b`, "math/big"},
		{`\bbig\.Rat\b`, "math/big"},
		{`\bslog\.Logger\b`, "log/slog"},
		{`\bslog\.Handler\b`, "log/slog"},
		{`\bslog\.Record\b`, "log/slog"},
		{`\btemplate\.Template\b`, "text/template"},
		{`\bhtml\.template\.Template\b`, "html/template"},
		{`\bflag\.FlagSet\b`, "flag"},
		{`\btesting\.T\b`, "testing"},
		{`\btesting\.B\b`, "testing"},
		{`\btesting\.M\b`, "testing"},
		{`\breflect\.Type\b`, "reflect"},
		{`\breflect\.Value\b`, "reflect"},
		{`\breflect\.Kind\b`, "reflect"},
		{`\btar\.Header\b`, "archive/tar"},
		{`\btar\.Reader\b`, "archive/tar"},
		{`\btar\.Writer\b`, "archive/tar"},
		{`\bzip\.Reader\b`, "archive/zip"},
		{`\bzip\.Writer\b`, "archive/zip"},
		{`\bgzip\.Reader\b`, "compress/gzip"},
		{`\bgzip\.Writer\b`, "compress/gzip"},
		{`\bzlib\.Reader\b`, "compress/zlib"},
		{`\bzlib\.Writer\b`, "compress/zlib"},
		{`\bflate\.Reader\b`, "compress/flate"},
		{`\bflate\.Writer\b`, "compress/flate"},
		{`\bimage\.Image\b`, "image"},
		{`\bimage\.Rectangle\b`, "image"},
		{`\bimage\.Point\b`, "image"},
		{`\bcolor\.Color\b`, "image/color"},
		{`\bcolor\.RGBA\b`, "image/color"},
		{`\bjpeg\.Options\b`, "image/jpeg"},
		{`\bpng\.Encoder\b`, "image/png"},
		{`\bgif\.GIF\b`, "image/gif"},
		
		// Third-party common types (not stdlib but commonly used)
		{`\buuid\.UUID\b`, "github.com/google/uuid"},
		
		// Package-level patterns (less specific, checked after specific types)
		{`\bcontext\.`, "context"},
		{`\btime\.`, "time"},
		{`\bfmt\.`, "fmt"},
		{`\bos\.`, "os"},
		{`\bio\.`, "io"},
		{`\bnet\.`, "net"},
		{`\bhttp\.`, "net/http"},
		{`\burl\.`, "net/url"},
		{`\bsql\.`, "database/sql"},
		{`\bjson\.`, "encoding/json"},
		{`\bxml\.`, "encoding/xml"},
		{`\bbase64\.`, "encoding/base64"},
		{`\bhex\.`, "encoding/hex"},
		{`\bregexp\.`, "regexp"},
		{`\bsync\.`, "sync"},
		{`\batomic\.`, "sync/atomic"},
		{`\bbytes\.`, "bytes"},
		{`\bstrings\.`, "strings"},
		{`\bbufio\.`, "bufio"},
		{`\bpath\.`, "path"},
		{`\bfilepath\.`, "path/filepath"},
		{`\btcp\.`, "net"},
		{`\budp\.`, "net"},
		{`\btls\.`, "crypto/tls"},
		{`\brsa\.`, "crypto/rsa"},
		{`\becdsa\.`, "crypto/ecdsa"},
		{`\bx509\.`, "crypto/x509"},
		{`\bmd5\.`, "crypto/md5"},
		{`\bsha1\.`, "crypto/sha1"},
		{`\bsha256\.`, "crypto/sha256"},
		{`\bsha512\.`, "crypto/sha512"},
		{`\bhmac\.`, "crypto/hmac"},
		{`\brand\.`, "math/rand"},
		{`\bmath\.`, "math"},
		{`\bbig\.`, "math/big"},
		{`\bslog\.`, "log/slog"},
		{`\blog\.`, "log"},
		{`\btemplate\.`, "text/template"},
		{`\bflag\.`, "flag"},
		{`\btesting\.`, "testing"},
		{`\breflect\.`, "reflect"},
		{`\bsort\.`, "sort"},
		{`\bstrconv\.`, "strconv"},
		{`\berrors\.`, "errors"},
		{`\btar\.`, "archive/tar"},
		{`\bzip\.`, "archive/zip"},
		{`\bgzip\.`, "compress/gzip"},
		{`\bzlib\.`, "compress/zlib"},
		{`\bflate\.`, "compress/flate"},
		{`\bimage\.`, "image"},
		{`\bcolor\.`, "image/color"},
		{`\bjpeg\.`, "image/jpeg"},
		{`\bpng\.`, "image/png"},
		{`\bgif\.`, "image/gif"},
		{`\bcsv\.`, "encoding/csv"},
		{`\bgob\.`, "encoding/gob"},
		{`\bpem\.`, "encoding/pem"},
		{`\basn1\.`, "encoding/asn1"},
		{`\bmime\.`, "mime"},
		{`\bmultipart\.`, "mime/multipart"},
		{`\bquotedprintable\.`, "mime/quotedprintable"},
		{`\bsmtp\.`, "net/smtp"},
		{`\btextproto\.`, "net/textproto"},
		{`\brpc\.`, "net/rpc"},
		{`\bjsonrpc\.`, "net/rpc/jsonrpc"},
		{`\bexec\.`, "os/exec"},
		{`\bsignal\.`, "os/signal"},
		{`\buser\.`, "os/user"},
		{`\bruntime\.`, "runtime"},
		{`\bdebug\.`, "runtime/debug"},
		{`\bpprof\.`, "runtime/pprof"},
		{`\btrace\.`, "runtime/trace"},
		{`\bsyscall\.`, "syscall"},
		{`\bunsafe\.`, "unsafe"},
	}
	
	for _, tp := range typePatterns {
		re := regexp.MustCompile(tp.pattern)
		if re.MatchString(code) {
			importSet[tp.importPath] = true
		}
	}
	
	// Convert set back to slice
	for importPath := range importSet {
		imports = append(imports, Import{Path: importPath})
	}
	
	return imports
}

// detectFrameworkImports detects framework-specific imports needed
func (im *ImportManager) detectFrameworkImports(code string) []Import {
	var imports []Import
	
	// Framework patterns and their required imports
	frameworkPatterns := map[string]string{
		`\bfx\.`:                   "go.uber.org/fx",
		`\bfxevent\.`:              "go.uber.org/fx/fxevent",
		`\becho\.`:                 "github.com/labstack/echo/v4",
		`\baxon\.`:                 "github.com/toyz/axon/pkg/axon",
	}
	
	for pattern, importPath := range frameworkPatterns {
		re := regexp.MustCompile(pattern)
		if re.MatchString(code) {
			imports = append(imports, Import{Path: importPath})
		}
	}
	
	return imports
}

// GenerateImportBlock generates a properly formatted import block
func (im *ImportManager) GenerateImportBlock(requiredImports []Import) string {
	if len(requiredImports) == 0 {
		return ""
	}
	
	// Group imports by type
	stdLib := []Import{}
	thirdParty := []Import{}
	local := []Import{}
	
	for _, imp := range requiredImports {
		if im.isStandardLibrary(imp.Path) {
			stdLib = append(stdLib, imp)
		} else if im.isLocalPackage(imp.Path) {
			local = append(local, imp)
		} else {
			thirdParty = append(thirdParty, imp)
		}
	}
	
	// Sort each group
	sort.Slice(stdLib, func(i, j int) bool { return stdLib[i].Path < stdLib[j].Path })
	sort.Slice(thirdParty, func(i, j int) bool { return thirdParty[i].Path < thirdParty[j].Path })
	sort.Slice(local, func(i, j int) bool { return local[i].Path < local[j].Path })
	
	var importBlock strings.Builder
	importBlock.WriteString("import (\n")
	
	// Add standard library imports
	for _, imp := range stdLib {
		if imp.Alias != "" {
			importBlock.WriteString(fmt.Sprintf("\t%s \"%s\"\n", imp.Alias, imp.Path))
		} else {
			importBlock.WriteString(fmt.Sprintf("\t\"%s\"\n", imp.Path))
		}
	}
	
	// Add blank line between groups if needed
	if len(stdLib) > 0 && (len(thirdParty) > 0 || len(local) > 0) {
		importBlock.WriteString("\n")
	}
	
	// Add third-party imports
	for _, imp := range thirdParty {
		if imp.Alias != "" {
			importBlock.WriteString(fmt.Sprintf("\t%s \"%s\"\n", imp.Alias, imp.Path))
		} else {
			importBlock.WriteString(fmt.Sprintf("\t\"%s\"\n", imp.Path))
		}
	}
	
	// Add blank line between groups if needed
	if len(thirdParty) > 0 && len(local) > 0 {
		importBlock.WriteString("\n")
	}
	
	// Add local imports
	for _, imp := range local {
		if imp.Alias != "" {
			importBlock.WriteString(fmt.Sprintf("\t%s \"%s\"\n", imp.Alias, imp.Path))
		} else {
			importBlock.WriteString(fmt.Sprintf("\t\"%s\"\n", imp.Path))
		}
	}
	
	importBlock.WriteString(")")
	return importBlock.String()
}

// FilterUnusedImports removes imports that aren't actually used in the code
func (im *ImportManager) FilterUnusedImports(imports []Import, generatedCode string) []Import {
	var usedImports []Import
	
	for _, imp := range imports {
		if im.isImportUsed(imp, generatedCode) {
			usedImports = append(usedImports, imp)
		}
	}
	
	return usedImports
}

// isImportUsed checks if an import is actually used in the generated code
func (im *ImportManager) isImportUsed(imp Import, code string) bool {
	// Get the package name to look for
	packageName := imp.Alias
	if packageName == "" {
		// Extract package name from import path
		parts := strings.Split(imp.Path, "/")
		packageName = parts[len(parts)-1]
		
		// Handle special cases like "log/slog" -> "slog"
		if strings.Contains(packageName, ".") {
			packageName = strings.Split(packageName, ".")[0]
		}
	}
	
	// Look for usage patterns
	patterns := []string{
		fmt.Sprintf(`\b%s\.`, regexp.QuoteMeta(packageName)),
		fmt.Sprintf(`\*%s\.`, regexp.QuoteMeta(packageName)),
	}
	
	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		if re.MatchString(code) {
			return true
		}
	}
	
	return false
}

// isStandardLibrary checks if an import path is from the Go standard library
func (im *ImportManager) isStandardLibrary(importPath string) bool {
	// Standard library packages don't contain dots (except for some special cases)
	if !strings.Contains(importPath, ".") {
		return true
	}
	
	// Handle special standard library cases and extended standard library
	stdLibPrefixes := []string{
		"golang.org/x/",
	}
	
	for _, prefix := range stdLibPrefixes {
		if strings.HasPrefix(importPath, prefix) {
			return true
		}
	}
	
	// Comprehensive list of all Go standard library packages
	stdLibPackages := map[string]bool{
		// Core packages
		"archive/tar": true, "archive/zip": true,
		"bufio": true, "builtin": true, "bytes": true,
		"compress/bzip2": true, "compress/flate": true, "compress/gzip": true, "compress/lzw": true, "compress/zlib": true,
		"container/heap": true, "container/list": true, "container/ring": true,
		"context": true,
		"crypto": true, "crypto/aes": true, "crypto/cipher": true, "crypto/des": true, "crypto/dsa": true, "crypto/ecdsa": true, "crypto/ed25519": true, "crypto/elliptic": true, "crypto/hmac": true, "crypto/md5": true, "crypto/rand": true, "crypto/rc4": true, "crypto/rsa": true, "crypto/sha1": true, "crypto/sha256": true, "crypto/sha512": true, "crypto/subtle": true, "crypto/tls": true, "crypto/x509": true, "crypto/x509/pkix": true,
		"database/sql": true, "database/sql/driver": true,
		"debug/buildinfo": true, "debug/dwarf": true, "debug/elf": true, "debug/gosym": true, "debug/macho": true, "debug/pe": true, "debug/plan9obj": true,
		"embed": true,
		"encoding": true, "encoding/ascii85": true, "encoding/asn1": true, "encoding/base32": true, "encoding/base64": true, "encoding/binary": true, "encoding/csv": true, "encoding/gob": true, "encoding/hex": true, "encoding/json": true, "encoding/pem": true, "encoding/xml": true,
		"errors": true,
		"expvar": true,
		"flag": true,
		"fmt": true,
		"go/ast": true, "go/build": true, "go/build/constraint": true, "go/constant": true, "go/doc": true, "go/format": true, "go/importer": true, "go/parser": true, "go/printer": true, "go/scanner": true, "go/token": true, "go/types": true,
		"hash": true, "hash/adler32": true, "hash/crc32": true, "hash/crc64": true, "hash/fnv": true, "hash/maphash": true,
		"html": true, "html/template": true,
		"image": true, "image/color": true, "image/color/palette": true, "image/draw": true, "image/gif": true, "image/jpeg": true, "image/png": true,
		"index/suffixarray": true,
		"io": true, "io/fs": true, "io/ioutil": true,
		"log": true, "log/slog": true, "log/syslog": true,
		"math": true, "math/big": true, "math/bits": true, "math/cmplx": true, "math/rand": true,
		"mime": true, "mime/multipart": true, "mime/quotedprintable": true,
		"net": true, "net/http": true, "net/http/cgi": true, "net/http/cookiejar": true, "net/http/fcgi": true, "net/http/httptest": true, "net/http/httptrace": true, "net/http/httputil": true, "net/http/pprof": true, "net/mail": true, "net/netip": true, "net/rpc": true, "net/rpc/jsonrpc": true, "net/smtp": true, "net/textproto": true, "net/url": true,
		"os": true, "os/exec": true, "os/signal": true, "os/user": true,
		"path": true, "path/filepath": true,
		"plugin": true,
		"reflect": true,
		"regexp": true, "regexp/syntax": true,
		"runtime": true, "runtime/cgo": true, "runtime/debug": true, "runtime/metrics": true, "runtime/pprof": true, "runtime/race": true, "runtime/trace": true,
		"sort": true,
		"strconv": true,
		"strings": true,
		"sync": true, "sync/atomic": true,
		"syscall": true, "syscall/js": true,
		"testing": true, "testing/fstest": true, "testing/iotest": true, "testing/quick": true,
		"text/scanner": true, "text/tabwriter": true, "text/template": true, "text/template/parse": true,
		"time": true, "time/tzdata": true,
		"unicode": true, "unicode/utf16": true, "unicode/utf8": true,
		"unsafe": true,
	}
	
	return stdLibPackages[importPath]
}

// isLocalPackage checks if an import path is a local package
func (im *ImportManager) isLocalPackage(importPath string) bool {
	if im.packageResolver == nil || im.packageResolver.ModulePath == "" {
		return false
	}
	
	return strings.HasPrefix(importPath, im.packageResolver.ModulePath)
}

// ResolveLocalPackage resolves a local package name to its full import path
func (im *ImportManager) ResolveLocalPackage(packageName string) (string, error) {
	if im.packageResolver == nil {
		return "", fmt.Errorf("no package resolver configured")
	}
	
	return im.packageResolver.ResolvePackagePath(packageName)
}

// ResolvePackagePath converts a package directory to its full import path
func (pr *PackageResolver) ResolvePackagePath(packageDir string) (string, error) {
	if pr.ModulePath == "" {
		return "", fmt.Errorf("module path not set")
	}
	
	// Check cache first
	if cachedPath, exists := pr.PackageMap[packageDir]; exists {
		return cachedPath, nil
	}
	
	// Clean the package directory path
	cleanDir := filepath.Clean(packageDir)
	
	// Remove leading "./" if present
	cleanDir = strings.TrimPrefix(cleanDir, "./")
	
	// Construct full import path
	fullPath := pr.ModulePath
	if cleanDir != "." && cleanDir != "" {
		fullPath = pr.ModulePath + "/" + cleanDir
	}
	
	// Cache the result
	pr.PackageMap[packageDir] = fullPath
	
	return fullPath, nil
}

// findModuleInfo finds the go.mod file and extracts module information
func findModuleInfo(startDir string) (moduleRoot, modulePath string, err error) {
	dir := startDir
	
	for {
		goModPath := filepath.Join(dir, "go.mod")
		if _, err := os.Stat(goModPath); err == nil {
			// Found go.mod, extract module path
			modulePath, err := extractModulePath(goModPath)
			if err != nil {
				return "", "", fmt.Errorf("failed to extract module path from %s: %w", goModPath, err)
			}
			return dir, modulePath, nil
		}
		
		parent := filepath.Dir(dir)
		if parent == dir {
			// Reached root directory
			break
		}
		dir = parent
	}
	
	return "", "", fmt.Errorf("go.mod not found in %s or any parent directory", startDir)
}

// extractModulePath extracts the module path from a go.mod file
func extractModulePath(goModPath string) (string, error) {
	content, err := os.ReadFile(goModPath)
	if err != nil {
		return "", fmt.Errorf("failed to read go.mod: %w", err)
	}
	
	// Parse go.mod to extract module path
	lines := strings.Split(string(content), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "module ") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				return parts[1], nil
			}
		}
	}
	
	return "", fmt.Errorf("module declaration not found in go.mod")
}

// ExtractImportsFromFile extracts imports from a Go source file
func ExtractImportsFromFile(filename string) ([]Import, error) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, filename, nil, parser.ParseComments)
	if err != nil {
		return nil, fmt.Errorf("failed to parse file %s: %w", filename, err)
	}
	
	var imports []Import
	for _, imp := range file.Imports {
		importPath := strings.Trim(imp.Path.Value, `"`)
		alias := ""
		
		if imp.Name != nil {
			alias = imp.Name.Name
		}
		
		imports = append(imports, Import{
			Path:  importPath,
			Alias: alias,
		})
	}
	
	return imports, nil
}