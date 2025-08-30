package utils

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFileProcessor_DefaultFilters(t *testing.T) {
	// Create temporary directory structure
	tmpDir := t.TempDir()
	
	// Create test files
	files := map[string]string{
		"main.go":           "package main",
		"main_test.go":      "package main",
		"autogen_module.go": "package main",
		"service.go":        "package service",
		"README.md":         "# README",
	}
	
	for filename, content := range files {
		filePath := filepath.Join(tmpDir, filename)
		err := os.WriteFile(filePath, []byte(content), 0644)
		if err != nil {
			t.Fatalf("Failed to create test file %s: %v", filename, err)
		}
	}
	
	// Test DefaultGoFileFilter
	goFilter := DefaultGoFileFilter()
	
	entries, err := os.ReadDir(tmpDir)
	if err != nil {
		t.Fatalf("Failed to read test directory: %v", err)
	}
	
	var goFiles []string
	for _, entry := range entries {
		if goFilter(filepath.Join(tmpDir, entry.Name()), entry) {
			goFiles = append(goFiles, entry.Name())
		}
	}
	
	expectedGoFiles := []string{"main.go", "service.go"}
	if len(goFiles) != len(expectedGoFiles) {
		t.Errorf("Expected %d Go files, got %d: %v", len(expectedGoFiles), len(goFiles), goFiles)
	}
	
	// Test TestGoFileFilter
	testFilter := TestGoFileFilter()
	
	var testFiles []string
	for _, entry := range entries {
		if testFilter(filepath.Join(tmpDir, entry.Name()), entry) {
			testFiles = append(testFiles, entry.Name())
		}
	}
	
	expectedTestFiles := []string{"main_test.go"}
	if len(testFiles) != len(expectedTestFiles) {
		t.Errorf("Expected %d test files, got %d: %v", len(expectedTestFiles), len(testFiles), testFiles)
	}
	
	// Test AutogenFileFilter
	autogenFilter := AutogenFileFilter()
	
	var autogenFiles []string
	for _, entry := range entries {
		if autogenFilter(filepath.Join(tmpDir, entry.Name()), entry) {
			autogenFiles = append(autogenFiles, entry.Name())
		}
	}
	
	expectedAutogenFiles := []string{"autogen_module.go"}
	if len(autogenFiles) != len(expectedAutogenFiles) {
		t.Errorf("Expected %d autogen files, got %d: %v", len(expectedAutogenFiles), len(autogenFiles), autogenFiles)
	}
}

func TestFileProcessor_HasGoFiles(t *testing.T) {
	fp := NewFileProcessor()
	
	// Test with directory containing Go files
	tmpDir := t.TempDir()
	
	// Create a Go file
	goFile := filepath.Join(tmpDir, "main.go")
	err := os.WriteFile(goFile, []byte("package main"), 0644)
	if err != nil {
		t.Fatalf("Failed to create Go file: %v", err)
	}
	
	hasGo, err := fp.HasGoFiles(tmpDir)
	if err != nil {
		t.Fatalf("HasGoFiles failed: %v", err)
	}
	
	if !hasGo {
		t.Error("Expected directory to have Go files")
	}
	
	// Test with directory containing only test files
	testDir := t.TempDir()
	testFile := filepath.Join(testDir, "main_test.go")
	err = os.WriteFile(testFile, []byte("package main"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	
	hasGo, err = fp.HasGoFiles(testDir)
	if err != nil {
		t.Fatalf("HasGoFiles failed: %v", err)
	}
	
	if hasGo {
		t.Error("Expected directory to not have Go files (only test files)")
	}
	
	// Test with empty directory
	emptyDir := t.TempDir()
	
	hasGo, err = fp.HasGoFiles(emptyDir)
	if err != nil {
		t.Fatalf("HasGoFiles failed: %v", err)
	}
	
	if hasGo {
		t.Error("Expected empty directory to not have Go files")
	}
}

func TestFileProcessor_WalkFiles(t *testing.T) {
	fp := NewFileProcessor()
	
	// Create test directory structure
	tmpDir := t.TempDir()
	
	// Create subdirectories
	subDir1 := filepath.Join(tmpDir, "pkg1")
	subDir2 := filepath.Join(tmpDir, "pkg2")
	vendorDir := filepath.Join(tmpDir, "vendor")
	
	err := os.MkdirAll(subDir1, 0755)
	if err != nil {
		t.Fatalf("Failed to create subdirectory: %v", err)
	}
	
	err = os.MkdirAll(subDir2, 0755)
	if err != nil {
		t.Fatalf("Failed to create subdirectory: %v", err)
	}
	
	err = os.MkdirAll(vendorDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create vendor directory: %v", err)
	}
	
	// Create test files
	files := map[string]string{
		filepath.Join(tmpDir, "main.go"):           "package main",
		filepath.Join(subDir1, "service.go"):      "package pkg1",
		filepath.Join(subDir1, "service_test.go"): "package pkg1",
		filepath.Join(subDir2, "handler.go"):      "package pkg2",
		filepath.Join(vendorDir, "vendor.go"):     "package vendor",
	}
	
	for filePath, content := range files {
		err := os.WriteFile(filePath, []byte(content), 0644)
		if err != nil {
			t.Fatalf("Failed to create test file %s: %v", filePath, err)
		}
	}
	
	// Test walking with Go file filter and directory filter
	options := FileWalkOptions{
		FileFilter:      DefaultGoFileFilter(),
		DirectoryFilter: DefaultDirectoryFilter(),
		SkipErrors:      true,
	}
	
	matchedFiles, err := fp.WalkFiles(tmpDir, options)
	if err != nil {
		t.Fatalf("WalkFiles failed: %v", err)
	}
	
	// Should find main.go, pkg1/service.go, pkg2/handler.go
	// Should NOT find service_test.go (test file) or vendor.go (vendor directory)
	expectedCount := 3
	if len(matchedFiles) != expectedCount {
		t.Errorf("Expected %d files, got %d: %v", expectedCount, len(matchedFiles), matchedFiles)
	}
	
	// Verify vendor directory was skipped
	for _, file := range matchedFiles {
		if filepath.Dir(file) == vendorDir {
			t.Errorf("Vendor directory should have been skipped, but found: %s", file)
		}
	}
}

func TestFileProcessor_ScanDirectoriesWithGoFiles(t *testing.T) {
	fp := NewFileProcessor()
	
	// Create test directory structure
	tmpDir := t.TempDir()
	
	// Create subdirectories
	subDir1 := filepath.Join(tmpDir, "pkg1")
	subDir2 := filepath.Join(tmpDir, "pkg2")
	emptyDir := filepath.Join(tmpDir, "empty")
	
	err := os.MkdirAll(subDir1, 0755)
	if err != nil {
		t.Fatalf("Failed to create subdirectory: %v", err)
	}
	
	err = os.MkdirAll(subDir2, 0755)
	if err != nil {
		t.Fatalf("Failed to create subdirectory: %v", err)
	}
	
	err = os.MkdirAll(emptyDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create empty directory: %v", err)
	}
	
	// Create Go files in some directories
	files := map[string]string{
		filepath.Join(tmpDir, "main.go"):      "package main",
		filepath.Join(subDir1, "service.go"): "package pkg1",
		filepath.Join(subDir2, "handler.go"): "package pkg2",
	}
	
	for filePath, content := range files {
		err := os.WriteFile(filePath, []byte(content), 0644)
		if err != nil {
			t.Fatalf("Failed to create test file %s: %v", filePath, err)
		}
	}
	
	// Scan for directories with Go files
	packageDirs, err := fp.ScanDirectoriesWithGoFiles([]string{tmpDir})
	if err != nil {
		t.Fatalf("ScanDirectoriesWithGoFiles failed: %v", err)
	}
	
	// Should find tmpDir, subDir1, subDir2
	// Should NOT find emptyDir
	expectedCount := 3
	if len(packageDirs) != expectedCount {
		t.Errorf("Expected %d package directories, got %d: %v", expectedCount, len(packageDirs), packageDirs)
	}
	
	// Verify empty directory was not included
	for _, dir := range packageDirs {
		if dir == emptyDir {
			t.Errorf("Empty directory should not have been included: %s", dir)
		}
	}
}

func TestFileProcessor_CleanDirectories(t *testing.T) {
	fp := NewFileProcessor()
	
	// Create test directory structure
	tmpDir := t.TempDir()
	
	subDir := filepath.Join(tmpDir, "pkg1")
	err := os.MkdirAll(subDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create subdirectory: %v", err)
	}
	
	// Create autogen files
	autogenFiles := []string{
		filepath.Join(tmpDir, "autogen_module.go"),
		filepath.Join(subDir, "autogen_module.go"),
	}
	
	for _, filePath := range autogenFiles {
		err := os.WriteFile(filePath, []byte("package main"), 0644)
		if err != nil {
			t.Fatalf("Failed to create autogen file %s: %v", filePath, err)
		}
	}
	
	// Create regular Go file (should not be removed)
	regularFile := filepath.Join(tmpDir, "main.go")
	err = os.WriteFile(regularFile, []byte("package main"), 0644)
	if err != nil {
		t.Fatalf("Failed to create regular file: %v", err)
	}
	
	// Clean directories
	removedFiles, err := fp.CleanDirectories([]string{tmpDir})
	if err != nil {
		t.Fatalf("CleanDirectories failed: %v", err)
	}
	
	// Should have removed 2 autogen files
	expectedCount := 2
	if len(removedFiles) != expectedCount {
		t.Errorf("Expected %d removed files, got %d: %v", expectedCount, len(removedFiles), removedFiles)
	}
	
	// Verify autogen files were removed
	for _, autogenFile := range autogenFiles {
		if _, err := os.Stat(autogenFile); !os.IsNotExist(err) {
			t.Errorf("Autogen file should have been removed: %s", autogenFile)
		}
	}
	
	// Verify regular file was not removed
	if _, err := os.Stat(regularFile); os.IsNotExist(err) {
		t.Error("Regular file should not have been removed")
	}
}

func TestFileProcessor_ParseDirectoryFiles(t *testing.T) {
	fp := NewFileProcessor()
	
	// Create test directory
	tmpDir := t.TempDir()
	
	// Create Go files in the same package
	files := map[string]string{
		"main.go":    "package main\n\nfunc main() {}",
		"service.go": "package main\n\ntype Service struct {}",
	}
	
	for filename, content := range files {
		filePath := filepath.Join(tmpDir, filename)
		err := os.WriteFile(filePath, []byte(content), 0644)
		if err != nil {
			t.Fatalf("Failed to create test file %s: %v", filename, err)
		}
	}
	
	// Parse directory files
	parsedFiles, packageName, err := fp.ParseDirectoryFiles(tmpDir)
	if err != nil {
		t.Fatalf("ParseDirectoryFiles failed: %v", err)
	}
	
	// Should have parsed 2 files
	expectedCount := 2
	if len(parsedFiles) != expectedCount {
		t.Errorf("Expected %d parsed files, got %d", expectedCount, len(parsedFiles))
	}
	
	// Should have detected package name
	expectedPackage := "main"
	if packageName != expectedPackage {
		t.Errorf("Expected package name %s, got %s", expectedPackage, packageName)
	}
	
	// Verify files were parsed correctly
	for filePath, astFile := range parsedFiles {
		if astFile == nil {
			t.Errorf("AST file should not be nil for %s", filePath)
		}
		
		if astFile.Name.Name != expectedPackage {
			t.Errorf("Expected package %s in AST, got %s", expectedPackage, astFile.Name.Name)
		}
	}
}