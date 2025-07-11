package main

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

// MockDB is a mock for sql.DB
type MockDB struct {
	ExecContextFunc func(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
}

func (m *MockDB) ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	if m.ExecContextFunc != nil {
		return m.ExecContextFunc(ctx, query, args...)
	}
	return nil, nil
}

// MockSQLResult is a mock for sql.Result
type MockSQLResult struct {
	LastInsertIdFunc func() (int64, error)
	RowsAffectedFunc func() (int64, error)
}

func (m *MockSQLResult) LastInsertId() (int64, error) {
	if m.LastInsertIdFunc != nil {
		return m.LastInsertIdFunc()
	}
	return 0, nil
}

func (m *MockSQLResult) RowsAffected() (int64, error) {
	if m.RowsAffectedFunc != nil {
		return m.RowsAffectedFunc()
	}
	return 0, nil
}

func TestCollectCodeLines(t *testing.T) {
	// Create a temporary directory for test files
	tmpDir, err := os.MkdirTemp("", "test_collect")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create dummy Go files
	file1Content := "package main

func hello() {}"
	file1Path := filepath.Join(tmpDir, "file1.go")
	if err := os.WriteFile(file1Path, []byte(file1Content), 0644); err != nil {
		t.Fatalf("Failed to write dummy file1: %v", err)
	}

	file2Content := "package other

// A comment
var x = 10"
	subDir := filepath.Join(tmpDir, "subdir")
	if err := os.Mkdir(subDir, 0755); err != nil {
		t.Fatalf("Failed to create sub dir: %v", err)
	}
	file2Path := filepath.Join(subDir, "file2.go")
	if err := os.WriteFile(file2Path, []byte(file2Content), 0644); err != nil {
		t.Fatalf("Failed to write dummy file2: %v", err)
	}

	// Create an empty go file
	emptyFilePath := filepath.Join(tmpDir, "empty.go")
	if err := os.WriteFile(emptyFilePath, []byte(""), 0644); err != nil {
		t.Fatalf("Failed to write empty file: %v", err)
	}

	// Create a non-go file
	nonGoFilePath := filepath.Join(tmpDir, "text.txt")
	if err := os.WriteFile(nonGoFilePath, []byte("hello world"), 0644); err != nil {
		t.Fatalf("Failed to write non-go file: %v", err)
	}


	expected := []CodeLine{
		{Package: tmpDir, File: "empty.go", LineNumber: 1, Content: ""},
		{Package: tmpDir, File: "file1.go", LineNumber: 1, Content: "package main"},
		{Package: tmpDir, File: "file1.go", LineNumber: 2, Content: ""},
		{Package: tmpDir, File: "file1.go", LineNumber: 3, Content: "func hello() {}"},
		{Package: subDir, File: "file2.go", LineNumber: 1, Content: "package other"},
		{Package: subDir, File: "file2.go", LineNumber: 2, Content: ""},
		{Package: subDir, File: "file2.go", LineNumber: 3, Content: "// A comment"},
		{Package: subDir, File: "file2.go", LineNumber: 4, Content: "var x = 10"},
	}

	results, err := collectCodeLines(tmpDir)
	if err != nil {
		t.Fatalf("collectCodeLines failed: %v", err)
	}

	// Normalize file paths in results for comparison
	var normalizedResults []CodeLine
	for _, r := range results {
		// We only care about the base directory for package comparison in this test.
		// In real scenarios, it would be the go package name.
		// For files directly in tmpDir, Package will be tmpDir.
		// For files in subDir, Package will be subDir.
		var pkgPath string
		absResultPkg, _ := filepath.Abs(r.Package)
		absTmpDir, _ := filepath.Abs(tmpDir)
		absSubDir, _ := filepath.Abs(subDir)

		if absResultPkg == absTmpDir {
			pkgPath = tmpDir
		} else if absResultPkg == absSubDir {
			pkgPath = subDir
		} else {
			t.Errorf("Unexpected package path: %s", r.Package)
		}

		normalizedResults = append(normalizedResults, CodeLine{
			Package:    pkgPath,
			File:       r.File,
			LineNumber: r.LineNumber,
			Content:    r.Content,
		})
	}

	// Sort both slices for consistent comparison as order from filepath.Walk is not guaranteed
	sortCodeLines := func(lines []CodeLine) {
		for i := 0; i < len(lines); i++ {
			for j := i + 1; j < len(lines); j++ {
				if lines[i].Package > lines[j].Package ||
					(lines[i].Package == lines[j].Package && lines[i].File > lines[j].File) ||
					(lines[i].Package == lines[j].Package && lines[i].File == lines[j].File && lines[i].LineNumber > lines[j].LineNumber) {
					lines[i], lines[j] = lines[j], lines[i]
				}
			}
		}
	}

	sortCodeLines(expected)
	sortCodeLines(normalizedResults)

	if !reflect.DeepEqual(normalizedResults, expected) {
		t.Errorf("Expected:\n%v\nGot:\n%v", expected, normalizedResults)
	}

	// Test with a non-existent directory
	_, err = collectCodeLines(filepath.Join(tmpDir, "nonexistent"))
	if err == nil {
		t.Errorf("Expected error for non-existent directory, got nil")
	}
}

func TestPopulateCode(t *testing.T) {
	ctx := context.Background()
	mockDB := &MockDB{}
	var execCalls [][6]interface{} // To store call arguments

	mockDB.ExecContextFunc = func(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
		call := [6]interface{}{query, args[0], args[1], args[2], args[3], args[4]}
		execCalls = append(execCalls, call)
		return &MockSQLResult{}, nil
	}

	// Create a temporary directory for test files
	tmpDir, err := os.MkdirTemp("", "test_populate")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	file1Content := "package main
func main(){}"
	file1Path := filepath.Join(tmpDir, "main.go")
	if err := os.WriteFile(file1Path, []byte(file1Content), 0644); err != nil {
		t.Fatalf("Failed to write dummy file: %v", err)
	}

	//Expected data based on file1Content
	// Note: Package path normalization for the mock is simplified here.
	// collectCodeLines will return absolute paths for Package.
	// We'll check the number of calls and structure rather than exact package path string.
	expectedPkgPath, _ := filepath.Abs(tmpDir)

	expectedCalls := []struct {
		QueryContent string // Just the query string for simplicity
		Package      string
		File         string
		LineNumber   int
		Content      string
	}{
		{"INSERT INTO all_code (package, file, line_number, content) VALUES (?, ?, ?, ?);", expectedPkgPath, "main.go", 1, "package main"},
		{"INSERT INTO all_code (package, file, line_number, content) VALUES (?, ?, ?, ?);", expectedPkgPath, "main.go", 2, "func main(){}"},
	}


	err = populateCode(ctx, mockDB, tmpDir)
	if err != nil {
		t.Fatalf("populateCode failed: %v", err)
	}

	if len(execCalls) != len(expectedCalls) {
		t.Fatalf("Expected %d ExecContext calls, got %d", len(expectedCalls), len(execCalls))
	}

	for i, expectedCall := range expectedCalls {
		call := execCalls[i]
		if call[0] != expectedCall.QueryContent {
			t.Errorf("Call %d: Expected query %q, got %q", i, expectedCall.QueryContent, call[0])
		}
		// args[0] is package, args[1] is file, args[2] is line_number, args[3] is content
		if call[2] != expectedCall.File { // call[2] is args[1] which is file name
			t.Errorf("Call %d: Expected file %q, got %q", i, expectedCall.File, call[2])
		}
		if call[3] != expectedCall.LineNumber { // call[3] is args[2] which is line number
			t.Errorf("Call %d: Expected line_number %d, got %v", i, expectedCall.LineNumber, call[3])
		}
		if call[4] != expectedCall.Content { // call[4] is args[3] which is content
			t.Errorf("Call %d: Expected content %q, got %q", i, expectedCall.Content, call[4])
		}
		// Check package path - it should be an absolute path to tmpDir
		if call[1] != expectedPkgPath {
             t.Errorf("Call %d: Expected package path %q, got %q", i, expectedPkgPath, call[1])
		}
	}
}
