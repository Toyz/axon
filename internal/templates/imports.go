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

// Pre-compiled regex patterns for performance
var (
	// Type patterns for import detection (compiled once at package init)
	typePatternRegexes = []struct {
		regex      *regexp.Regexp
		importPath string
	}{
		// Specific types first (most specific patterns)
		{regexp.MustCompile(`\bcontext\.Context\b`), "context"},
		{regexp.MustCompile(`\bcontext\.CancelFunc\b`), "context"},
		{regexp.MustCompile(`\btime\.Time\b`), "time"},
		{regexp.MustCompile(`\btime\.Duration\b`), "time"},
		{regexp.MustCompile(`\btime\.Location\b`), "time"},
		{regexp.MustCompile(`\btime\.Timer\b`), "time"},
		{regexp.MustCompile(`\btime\.Ticker\b`), "time"},
		{regexp.MustCompile(`\burl\.URL\b`), "net/url"},
		{regexp.MustCompile(`\burl\.Values\b`), "net/url"},
		{regexp.MustCompile(`\bhttp\.Request\b`), "net/http"},
		{regexp.MustCompile(`\bhttp\.Response\b`), "net/http"},
		{regexp.MustCompile(`\bhttp\.ResponseWriter\b`), "net/http"},
		{regexp.MustCompile(`\bhttp\.Handler\b`), "net/http"},
		{regexp.MustCompile(`\bhttp\.HandlerFunc\b`), "net/http"},
		{regexp.MustCompile(`\bhttp\.Client\b`), "net/http"},
		{regexp.MustCompile(`\bhttp\.Server\b`), "net/http"},
		{regexp.MustCompile(`\bsql\.DB\b`), "database/sql"},
		{regexp.MustCompile(`\bsql\.Tx\b`), "database/sql"},
		{regexp.MustCompile(`\bsql\.Rows\b`), "database/sql"},
		{regexp.MustCompile(`\bsql\.Row\b`), "database/sql"},
		{regexp.MustCompile(`\bsql\.Result\b`), "database/sql"},
		{regexp.MustCompile(`\bsql\.Stmt\b`), "database/sql"},
		{regexp.MustCompile(`\bjson\.Encoder\b`), "encoding/json"},
		{regexp.MustCompile(`\bjson\.Decoder\b`), "encoding/json"},
		{regexp.MustCompile(`\bjson\.RawMessage\b`), "encoding/json"},
		{regexp.MustCompile(`\bxml\.Encoder\b`), "encoding/xml"},
		{regexp.MustCompile(`\bxml\.Decoder\b`), "encoding/xml"},
		{regexp.MustCompile(`\bregexp\.Regexp\b`), "regexp"},
		{regexp.MustCompile(`\bsync\.Mutex\b`), "sync"},
		{regexp.MustCompile(`\bsync\.RWMutex\b`), "sync"},
		{regexp.MustCompile(`\bsync\.WaitGroup\b`), "sync"},
		{regexp.MustCompile(`\bsync\.Once\b`), "sync"},
		{regexp.MustCompile(`\bsync\.Cond\b`), "sync"},
		{regexp.MustCompile(`\bsync\.Pool\b`), "sync"},
		{regexp.MustCompile(`\batomic\.Value\b`), "sync/atomic"},
		{regexp.MustCompile(`\bbytes\.Buffer\b`), "bytes"},
		{regexp.MustCompile(`\bbytes\.Reader\b`), "bytes"},
		{regexp.MustCompile(`\bstrings\.Builder\b`), "strings"},
		{regexp.MustCompile(`\bstrings\.Reader\b`), "strings"},
		{regexp.MustCompile(`\bstrings\.Replacer\b`), "strings"},
		{regexp.MustCompile(`\bio\.Reader\b`), "io"},
		{regexp.MustCompile(`\bio\.Writer\b`), "io"},
		{regexp.MustCompile(`\bio\.ReadWriter\b`), "io"},
		{regexp.MustCompile(`\bio\.ReadCloser\b`), "io"},
		{regexp.MustCompile(`\bio\.WriteCloser\b`), "io"},
		{regexp.MustCompile(`\bio\.ReadWriteCloser\b`), "io"},
		{regexp.MustCompile(`\bbufio\.Reader\b`), "bufio"},
		{regexp.MustCompile(`\bbufio\.Writer\b`), "bufio"},
		{regexp.MustCompile(`\bbufio\.Scanner\b`), "bufio"},
		{regexp.MustCompile(`\bos\.File\b`), "os"},
		{regexp.MustCompile(`\bos\.FileInfo\b`), "os"},
		{regexp.MustCompile(`\bos\.FileMode\b`), "os"},
		{regexp.MustCompile(`\bos\.Process\b`), "os"},
		{regexp.MustCompile(`\bos\.ProcessState\b`), "os"},
		{regexp.MustCompile(`\bpath\.filepath\b`), "path/filepath"},
		{regexp.MustCompile(`\bfilepath\.WalkFunc\b`), "path/filepath"},
		{regexp.MustCompile(`\bnet\.Conn\b`), "net"},
		{regexp.MustCompile(`\bnet\.Listener\b`), "net"},
		{regexp.MustCompile(`\bnet\.Addr\b`), "net"},
		{regexp.MustCompile(`\bnet\.IP\b`), "net"},
		{regexp.MustCompile(`\bnet\.IPNet\b`), "net"},
		{regexp.MustCompile(`\btcp\.Conn\b`), "net"},
		{regexp.MustCompile(`\budp\.Conn\b`), "net"},
		{regexp.MustCompile(`\btls\.Config\b`), "crypto/tls"},
		{regexp.MustCompile(`\btls\.Conn\b`), "crypto/tls"},
		{regexp.MustCompile(`\brsa\.PrivateKey\b`), "crypto/rsa"},
		{regexp.MustCompile(`\brsa\.PublicKey\b`), "crypto/rsa"},
		{regexp.MustCompile(`\becdsa\.PrivateKey\b`), "crypto/ecdsa"},
		{regexp.MustCompile(`\becdsa\.PublicKey\b`), "crypto/ecdsa"},
		{regexp.MustCompile(`\bx509\.Certificate\b`), "crypto/x509"},
		{regexp.MustCompile(`\bmd5\.Sum\b`), "crypto/md5"},
		{regexp.MustCompile(`\bsha1\.Sum\b`), "crypto/sha1"},
		{regexp.MustCompile(`\bsha256\.Sum256\b`), "crypto/sha256"},
		{regexp.MustCompile(`\bsha512\.Sum512\b`), "crypto/sha512"},
		{regexp.MustCompile(`\bhmac\.New\b`), "crypto/hmac"},
		{regexp.MustCompile(`\brand\.Reader\b`), "crypto/rand"},
		{regexp.MustCompile(`\brand\.Rand\b`), "math/rand"},
		{regexp.MustCompile(`\brand\.Source\b`), "math/rand"},
		{regexp.MustCompile(`\bbig\.Int\b`), "math/big"},
		{regexp.MustCompile(`\bbig\.Float\b`), "math/big"},
		{regexp.MustCompile(`\bbig\.Rat\b`), "math/big"},
		{regexp.MustCompile(`\bslog\.Logger\b`), "log/slog"},
		{regexp.MustCompile(`\bslog\.Handler\b`), "log/slog"},
		{regexp.MustCompile(`\bslog\.Record\b`), "log/slog"},
		{regexp.MustCompile(`\btemplate\.Template\b`), "text/template"},
		{regexp.MustCompile(`\bhtml\.template\.Template\b`), "html/template"},
		{regexp.MustCompile(`\bflag\.FlagSet\b`), "flag"},
		{regexp.MustCompile(`\btesting\.T\b`), "testing"},
		{regexp.MustCompile(`\btesting\.B\b`), "testing"},
		{regexp.MustCompile(`\btesting\.M\b`), "testing"},
		{regexp.MustCompile(`\breflect\.Type\b`), "reflect"},
		{regexp.MustCompile(`\breflect\.Value\b`), "reflect"},
		{regexp.MustCompile(`\breflect\.Kind\b`), "reflect"},
		{regexp.MustCompile(`\btar\.Header\b`), "archive/tar"},
		{regexp.MustCompile(`\btar\.Reader\b`), "archive/tar"},
		{regexp.MustCompile(`\btar\.Writer\b`), "archive/tar"},
		{regexp.MustCompile(`\bzip\.Reader\b`), "archive/zip"},
		{regexp.MustCompile(`\bzip\.Writer\b`), "archive/zip"},
		{regexp.MustCompile(`\bgzip\.Reader\b`), "compress/gzip"},
		{regexp.MustCompile(`\bgzip\.Writer\b`), "compress/gzip"},
		{regexp.MustCompile(`\bzlib\.Reader\b`), "compress/zlib"},
		{regexp.MustCompile(`\bzlib\.Writer\b`), "compress/zlib"},
		{regexp.MustCompile(`\bflate\.Reader\b`), "compress/flate"},
		{regexp.MustCompile(`\bflate\.Writer\b`), "compress/flate"},
		{regexp.MustCompile(`\bimage\.Image\b`), "image"},
		{regexp.MustCompile(`\bimage\.Rectangle\b`), "image"},
		{regexp.MustCompile(`\bimage\.Point\b`), "image"},
		{regexp.MustCompile(`\bcolor\.Color\b`), "image/color"},
		{regexp.MustCompile(`\bcolor\.RGBA\b`), "image/color"},
		{regexp.MustCompile(`\bjpeg\.Options\b`), "image/jpeg"},
		{regexp.MustCompile(`\bpng\.Encoder\b`), "image/png"},
		{regexp.MustCompile(`\bgif\.GIF\b`), "image/gif"},
		
		// Third-party common types (not stdlib but commonly used)
		{regexp.MustCompile(`\buuid\.UUID\b`), "github.com/google/uuid"},
		
		// Package-level patterns (less specific, checked after specific types)
		{regexp.MustCompile(`\bcontext\.`), "context"},
		{regexp.MustCompile(`\btime\.`), "time"},
		{regexp.MustCompile(`\bfmt\.`), "fmt"},
		{regexp.MustCompile(`\bos\.`), "os"},
		{regexp.MustCompile(`\bio\.`), "io"},
		{regexp.MustCompile(`\bnet\.`), "net"},
		{regexp.MustCompile(`\bhttp\.`), "net/http"},
		{regexp.MustCompile(`\burl\.`), "net/url"},
		{regexp.MustCompile(`\bsql\.`), "database/sql"},
		{regexp.MustCompile(`\bjson\.`), "encoding/json"},
		{regexp.MustCompile(`\bxml\.`), "encoding/xml"},
		{regexp.MustCompile(`\bbase64\.`), "encoding/base64"},
		{regexp.MustCompile(`\bhex\.`), "encoding/hex"},
		{regexp.MustCompile(`\bregexp\.`), "regexp"},
		{regexp.MustCompile(`\bsync\.`), "sync"},
		{regexp.MustCompile(`\batomic\.`), "sync/atomic"},
		{regexp.MustCompile(`\bbytes\.`), "bytes"},
		{regexp.MustCompile(`\bstrings\.`), "strings"},
		{regexp.MustCompile(`\bbufio\.`), "bufio"},
		{regexp.MustCompile(`\bpath\.`), "path"},
		{regexp.MustCompile(`\bfilepath\.`), "path/filepath"},
		{regexp.MustCompile(`\btcp\.`), "net"},
		{regexp.MustCompile(`\budp\.`), "net"},
		{regexp.MustCompile(`\btls\.`), "crypto/tls"},
		{regexp.MustCompile(`\brsa\.`), "crypto/rsa"},
		{regexp.MustCompile(`\becdsa\.`), "crypto/ecdsa"},
		{regexp.MustCompile(`\bx509\.`), "crypto/x509"},
		{regexp.MustCompile(`\bmd5\.`), "crypto/md5"},
		{regexp.MustCompile(`\bsha1\.`), "crypto/sha1"},
		{regexp.MustCompile(`\bsha256\.`), "crypto/sha256"},
		{regexp.MustCompile(`\bsha512\.`), "crypto/sha512"},
		{regexp.MustCompile(`\bhmac\.`), "crypto/hmac"},
		{regexp.MustCompile(`\brand\.`), "math/rand"},
		{regexp.MustCompile(`\bmath\.`), "math"},
		{regexp.MustCompile(`\bbig\.`), "math/big"},
		{regexp.MustCompile(`\bslog\.`), "log/slog"},
		{regexp.MustCompile(`\blog\.`), "log"},
		{regexp.MustCompile(`\btemplate\.`), "text/template"},
		{regexp.MustCompile(`\bflag\.`), "flag"},
		{regexp.MustCompile(`\btesting\.`), "testing"},
		{regexp.MustCompile(`\breflect\.`), "reflect"},
		{regexp.MustCompile(`\bsort\.`), "sort"},
		{regexp.MustCompile(`\bstrconv\.`), "strconv"},
		{regexp.MustCompile(`\berrors\.`), "errors"},
		{regexp.MustCompile(`\btar\.`), "archive/tar"},
		{regexp.MustCompile(`\bzip\.`), "archive/zip"},
		{regexp.MustCompile(`\bgzip\.`), "compress/gzip"},
		{regexp.MustCompile(`\bzlib\.`), "compress/zlib"},
		{regexp.MustCompile(`\bflate\.`), "compress/flate"},
		{regexp.MustCompile(`\bimage\.`), "image"},
		{regexp.MustCompile(`\bcolor\.`), "image/color"},
		{regexp.MustCompile(`\bjpeg\.`), "image/jpeg"},
		{regexp.MustCompile(`\bpng\.`), "image/png"},
		{regexp.MustCompile(`\bgif\.`), "image/gif"},
		{regexp.MustCompile(`\bcsv\.`), "encoding/csv"},
		{regexp.MustCompile(`\bgob\.`), "encoding/gob"},
		{regexp.MustCompile(`\bpem\.`), "encoding/pem"},
		{regexp.MustCompile(`\basn1\.`), "encoding/asn1"},
		{regexp.MustCompile(`\bmime\.`), "mime"},
		{regexp.MustCompile(`\bmultipart\.`), "mime/multipart"},
		{regexp.MustCompile(`\bquotedprintable\.`), "mime/quotedprintable"},
		{regexp.MustCompile(`\bsmtp\.`), "net/smtp"},
		{regexp.MustCompile(`\btextproto\.`), "net/textproto"},
		{regexp.MustCompile(`\brpc\.`), "net/rpc"},
		{regexp.MustCompile(`\bjsonrpc\.`), "net/rpc/jsonrpc"},
		{regexp.MustCompile(`\bexec\.`), "os/exec"},
		{regexp.MustCompile(`\bsignal\.`), "os/signal"},
		{regexp.MustCompile(`\buser\.`), "os/user"},
		{regexp.MustCompile(`\bruntime\.`), "runtime"},
		{regexp.MustCompile(`\bdebug\.`), "runtime/debug"},
		{regexp.MustCompile(`\bpprof\.`), "runtime/pprof"},
		{regexp.MustCompile(`\btrace\.`), "runtime/trace"},
		{regexp.MustCompile(`\bsyscall\.`), "syscall"},
		{regexp.MustCompile(`\bunsafe\.`), "unsafe"},
	}

	// Framework patterns (compiled once at package init)
	frameworkPatternRegexes = map[*regexp.Regexp]string{
		regexp.MustCompile(`\bfx\.`):      "go.uber.org/fx",
		regexp.MustCompile(`\bfxevent\.`): "go.uber.org/fx/fxevent",
		regexp.MustCompile(`\becho\.`):    "github.com/labstack/echo/v4",
		regexp.MustCompile(`\baxon\.`):    "github.com/toyz/axon/pkg/axon",
	}
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
	
	// Validate and sanitize the project root path
	cleanProjectRoot := filepath.Clean(projectRoot)
	if !filepath.IsAbs(cleanProjectRoot) {
		var err error
		cleanProjectRoot, err = filepath.Abs(cleanProjectRoot)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve absolute path: %w", err)
		}
	}
	
	// Find go.mod file to determine module root and path
	moduleRoot, modulePath, err := findModuleInfo(cleanProjectRoot)
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
	
	// Use pre-compiled regex patterns for performance
	for _, tp := range typePatternRegexes {
		if tp.regex.MatchString(code) {
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
	
	// Use pre-compiled regex patterns for performance
	for regex, importPath := range frameworkPatternRegexes {
		if regex.MatchString(code) {
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
	
	// Use simple string matching for better performance instead of regex
	// Look for package usage patterns: "packageName." and "*packageName."
	quotedPackageName := regexp.QuoteMeta(packageName)
	patterns := []string{
		quotedPackageName + ".",
		"*" + quotedPackageName + ".",
	}
	
	for _, pattern := range patterns {
		if strings.Contains(code, pattern) {
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
	// Validate and clean the start directory
	dir := filepath.Clean(startDir)
	if !filepath.IsAbs(dir) {
		return "", "", fmt.Errorf("start directory must be absolute path")
	}
	
	for {
		// Safely construct the go.mod path using only the filename
		goModPath := filepath.Join(dir, filepath.Base("go.mod"))
		if _, err := os.Stat(goModPath); err == nil {
			// Found go.mod, extract module path
			modulePath, err := extractModulePath(goModPath)
			if err != nil {
				return "", "", fmt.Errorf("failed to extract module path from %s: %w", filepath.Base(goModPath), err)
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
	
	return "", "", fmt.Errorf("go.mod not found in %s or any parent directory", filepath.Base(startDir))
}

// extractModulePath extracts the module path from a go.mod file
func extractModulePath(goModPath string) (string, error) {
	// Validate the file path and ensure it's only the go.mod filename
	cleanPath := filepath.Clean(goModPath)
	if filepath.Base(cleanPath) != "go.mod" {
		return "", fmt.Errorf("invalid file: must be go.mod")
	}
	
	// Validate that the path doesn't contain directory traversal attempts
	if strings.Contains(cleanPath, "..") {
		return "", fmt.Errorf("invalid path: directory traversal not allowed")
	}
	
	content, err := os.ReadFile(cleanPath)
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
	// Validate and sanitize the filename
	cleanFilename := filepath.Clean(filename)
	
	// Validate that the path doesn't contain directory traversal attempts
	if strings.Contains(cleanFilename, "..") {
		return nil, fmt.Errorf("invalid filename: directory traversal not allowed")
	}
	
	// Validate that it's a Go file
	if filepath.Ext(cleanFilename) != ".go" {
		return nil, fmt.Errorf("invalid file: must be a .go file")
	}
	
	// Use only the base filename for error reporting to avoid path disclosure
	baseFilename := filepath.Base(cleanFilename)
	
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, cleanFilename, nil, parser.ParseComments)
	if err != nil {
		return nil, fmt.Errorf("failed to parse file %s: %w", baseFilename, err)
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