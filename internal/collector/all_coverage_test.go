package collector

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestCollectCoverageResults(t *testing.T) {
	// Create a temporary directory and a dummy coverage.out file
	tmpDir, err := os.MkdirTemp("", "test-collect-coverage-")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	coverageFile := filepath.Join(tmpDir, "coverage.out")
	coverageData := `mode: set
github.com/danicat/testquery/testdata/div.go:7.52,10.6 2 1
github.com/danicat/testquery/testdata/div.go:12.2,12.31 1 1
`
	if err := os.WriteFile(coverageFile, []byte(coverageData), 0644); err != nil {
		t.Fatalf("Failed to write coverage.out: %v", err)
	}

	// Temporarily change the working directory to the temp dir
	// so that cover.ParseProfiles can find the file.
	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current working directory: %v", err)
	}
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("Failed to change directory: %v", err)
	}
	defer os.Chdir(oldWd)

	// Call the function we are testing
	coverageResults, err := collectCoverageResults([]string{"./..."})
	if err != nil {
		t.Fatalf("collectCoverageResults failed: %v", err)
	}

	// Define the expected result
	expected := []CoverageResult{
		{
			Package:         "github.com/danicat/testquery/testdata/div.go",
			File:            "github.com/danicat/testquery/testdata/div.go",
			StartLine:       7,
			StartColumn:     52,
			EndLine:         10,
			EndColumn:       6,
			StatementNumber: 2,
			Count:           1,
			FunctionName:    "",
		},
		{
			Package:         "github.com/danicat/testquery/testdata/div.go",
			File:            "github.com/danicat/testquery/testdata/div.go",
			StartLine:       12,
			StartColumn:     2,
			EndLine:         12,
			EndColumn:       31,
			StatementNumber: 1,
			Count:           1,
			FunctionName:    "",
		},
	}

	// Check if the result matches the expectation
	if !reflect.DeepEqual(coverageResults, expected) {
		t.Errorf("collectCoverageResults() got = %v, want %v", coverageResults, expected)
	}
}
