package main

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

func setupCoverageTestDB(t *testing.T) *sql.DB {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open in-memory DB: %v", err)
	}

	_, err = db.Exec(`CREATE TABLE all_coverage (
		package TEXT NOT NULL,
		file TEXT NOT NULL,
		start_line INTEGER NOT NULL,
		start_col INTEGER NOT NULL,
		end_line INTEGER NOT NULL,
		end_col INTEGER NOT NULL,
		stmt_num INTEGER NOT NULL,
		count INTEGER NOT NULL,
		function_name TEXT NOT NULL
	);`)
	if err != nil {
		t.Fatalf("Failed to create schema: %v", err)
	}
	return db
}

func TestCollectCoverageResults(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "testcoverage")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a dummy Go source file for getFunctionName to parse
	dummyGoFileContent := `package main

func CoveredFunc() {
	// covered line
}

func UncoveredFunc() {
	// uncovered line
}

func AnotherCoveredFunc() {
	// another covered line
}
`
	dummyGoFilePath := filepath.Join(tmpDir, "dummy.go")
	if err := os.WriteFile(dummyGoFilePath, []byte(dummyGoFileContent), 0644); err != nil {
		t.Fatalf("Failed to write dummy.go: %v", err)
	}

	// Create a dummy coverage.out file
	// Note: The file paths in coverage.out are relative to the package directory from where `go test` was run.
	// For this test, we'll assume it's run in tmpDir.
	coverageOutContent := `mode: set
example.com/mypkg/dummy.go:3.18,5.2 1 1
example.com/mypkg/dummy.go:7.20,9.2 1 0
example.com/mypkg/dummy.go:11.25,13.2 1 2
`
	// The actual file path for coverage.out should be at the root of where collectCoverageResults expects it.
	// collectCoverageResults expects it to be in the current working directory or a path it can find.
	// For this test, we'll create it in tmpDir and then change the working directory for the duration of the test.

	originalWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current working directory: %v", err)
	}
	// Create coverage.out in tmpDir, because collectCoverageResults will look for "coverage.out"
	// and getFunctionName will look for pkgDir + "/" + fileName
	coverageOutPath := filepath.Join(tmpDir, "coverage.out")
	if err := os.WriteFile(coverageOutPath, []byte(coverageOutContent), 0644); err != nil {
		t.Fatalf("Failed to write coverage.out: %v", err)
	}

	// The paths in coverage.out are like "example.com/mypkg/dummy.go"
	// We need to make sure that pkgDir + "/" + fileName resolves correctly.
	// So, pkgDir should be "example.com/mypkg" if the files are in tmpDir.
	// Or, more simply, we can adjust the paths in coverage.out to be relative to tmpDir.
	// Let's adjust the coverage.out paths to be just "dummy.go" and set pkgDir to tmpDir.

	adjustedCoverageOutContent := `mode: set
dummy.go:3.18,5.2 1 1
dummy.go:7.20,9.2 1 0
dummy.go:11.25,13.2 1 2
`
	if err := os.WriteFile(coverageOutPath, []byte(adjustedCoverageOutContent), 0644); err != nil {
		t.Fatalf("Failed to write adjusted coverage.out: %v", err)
	}


	// Change working directory to tmpDir so "coverage.out" is found by ParseProfiles
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("Failed to change working directory to %s: %v", tmpDir, err)
	}
	defer os.Chdir(originalWd) // Change back to original wd

	expected := []CoverageResult{
		{Package: ".", File: "dummy.go", StartLine: 3, StartColumn: 18, EndLine: 5, EndColumn: 2, StatementNumber: 1, Count: 1, FunctionName: "CoveredFunc"},
		{Package: ".", File: "dummy.go", StartLine: 7, StartColumn: 20, EndLine: 9, EndColumn: 2, StatementNumber: 1, Count: 0, FunctionName: "UncoveredFunc"},
		{Package: ".", File: "dummy.go", StartLine: 11, StartColumn: 25, EndLine: 13, EndColumn: 2, StatementNumber: 1, Count: 2, FunctionName: "AnotherCoveredFunc"},
	}


	// pkgDir for collectCoverageResults should be tmpDir, as dummy.go is there.
	// And coverage.out refers to "dummy.go" directly.
	actual, err := collectCoverageResults(tmpDir)
	if err != nil {
		t.Fatalf("collectCoverageResults() error = %v", err)
	}

	// Sort for stable comparison
	sort.Slice(actual, func(i, j int) bool {
		return actual[i].StartLine < actual[j].StartLine
	})
	sort.Slice(expected, func(i, j int) bool {
		return expected[i].StartLine < expected[j].StartLine
	})

	// Normalize file paths in actual results for comparison if necessary
	for i := range actual {
		actual[i].Package = filepath.ToSlash(actual[i].Package)
		actual[i].File = filepath.ToSlash(actual[i].File)
		// If Package is derived from profile.FileName which might have OS-specific separators
		// and we set it to ".", it's fine. If it was more complex, normalization would be key.
	}


	if !reflect.DeepEqual(actual, expected) {
		t.Errorf("collectCoverageResults() got = %v, want %v", actual, expected)
	}

	// Test with non-existent coverage.out (by removing it)
	if err := os.Remove(filepath.Join(tmpDir, "coverage.out")); err != nil {
		t.Logf("Could not remove coverage.out for error test: %v", err)
	}
	_, err = collectCoverageResults(tmpDir)
	if err == nil {
		t.Errorf("collectCoverageResults() expected error for missing coverage.out, got nil")
	} else if !strings.Contains(err.Error(), "no such file or directory") && !strings.Contains(err.Error(), "cannot find the file") {
		// Error message can vary by OS
		t.Errorf("collectCoverageResults() expected file not found error, got: %v", err)
	}
}

func TestPopulateCoverageResults(t *testing.T) {
	db := setupCoverageTestDB(t)
	defer db.Close()

	tmpDir, err := os.MkdirTemp("", "testpopulatecoverage")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	originalWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current working directory: %v", err)
	}

	dummyGoFileContent := `package main

func MyFunc() {
	// some code
}`
	dummyGoFilePath := filepath.Join(tmpDir, "test.go")
	if err := os.WriteFile(dummyGoFilePath, []byte(dummyGoFileContent), 0644); err != nil {
		t.Fatalf("Failed to write dummy.go: %v", err)
	}

	coverageOutContent := `mode: set
test.go:3.15,5.2 1 1
`
	coverageOutPath := filepath.Join(tmpDir, "coverage.out") // This path is relative to tmpDir
	if err := os.WriteFile(coverageOutPath, []byte(coverageOutContent), 0644); err != nil {
		t.Fatalf("Failed to write coverage.out: %v", err)
	}

	// Chdir so that ParseProfiles finds "coverage.out" and getFunctionName finds "test.go"
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("Failed to change working directory to %s: %v", tmpDir, err)
	}
	defer os.Chdir(originalWd)


	err = populateCoverageResults(context.Background(), db, tmpDir) // pkgDir is tmpDir
	if err != nil {
		t.Fatalf("populateCoverageResults() error = %v", err)
	}

	rows, err := db.Query("SELECT package, file, start_line, start_col, end_line, end_col, stmt_num, count, function_name FROM all_coverage")
	if err != nil {
		t.Fatalf("Failed to query all_coverage: %v", err)
	}
	defer rows.Close()

	var actual []CoverageResult
	for rows.Next() {
		var cr CoverageResult
		if err := rows.Scan(&cr.Package, &cr.File, &cr.StartLine, &cr.StartColumn, &cr.EndLine, &cr.EndColumn, &cr.StatementNumber, &cr.Count, &cr.FunctionName); err != nil {
			t.Fatalf("Failed to scan row: %v", err)
		}
		actual = append(actual, cr)
	}

	expected := []CoverageResult{
		{Package: ".", File: "test.go", StartLine: 3, StartColumn: 15, EndLine: 5, EndColumn: 2, StatementNumber: 1, Count: 1, FunctionName: "MyFunc"},
	}

	// Normalize paths if necessary
	for i := range actual {
		actual[i].Package = filepath.ToSlash(actual[i].Package)
		actual[i].File = filepath.ToSlash(actual[i].File)
	}


	if !reflect.DeepEqual(actual, expected) {
		t.Errorf("populateCoverageResults() data in DB = %+v, want %+v", actual, expected)
	}
}
