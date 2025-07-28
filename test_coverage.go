package main

import (
	"context"
	"database/sql"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"

	"golang.org/x/tools/cover"
)

// TestCoverageResult represents the structure of a test-specific coverage result
type TestCoverageResult struct {
	TestName        string `json:"test_name"`
	Package         string `json:"package"`
	File            string `json:"file"`
	StartLine       int    `json:"start_line"`
	StartColumn     int    `json:"start_col"`
	EndLine         int    `json:"end_line"`
	EndColumn       int    `json:"end_col"`
	StatementNumber int    `json:"stmt_num"`
	Count           int    `json:"count"`
	FunctionName    string `json:"function_name"`
}

var nonAlphanumeric = regexp.MustCompile(`[^a-zA-Z0-9_]+`)

func sanitizeTestName(testName string) string {
	return nonAlphanumeric.ReplaceAllString(testName, "_")
}

func collectTestCoverageResults(pkgDir string, testResults []TestEvent) ([]TestCoverageResult, error) {
	var results []TestCoverageResult

	for _, test := range testResults {
		sanitizedTestName := sanitizeTestName(test.Test)
		coverageFile := sanitizedTestName + ".out"

		cmd := exec.Command("go", "test", pkgDir, "-run", "^"+test.Test+"$", "-coverprofile="+coverageFile)
		if err := cmd.Run(); err != nil {
			os.Remove(coverageFile) // Clean up even on failure
			if _, ok := err.(*exec.ExitError); ok {
				// Test failed, which is expected. Log and continue.
				log.Printf("test failed, skipping coverage for %s", test.Test)
				continue
			} else {
				// For other errors (e.g., command not found), return the error.
				return nil, fmt.Errorf("failed to run go test for coverage: %w", err)
			}
		}

		profiles, err := cover.ParseProfiles(coverageFile)
		os.Remove(coverageFile) // Clean up after parsing
		if err != nil {
			log.Printf("failed to parse coverage profile for %s: %v", test.Test, err)
			continue
		}

		for _, profile := range profiles {
			packageName := filepath.Dir(profile.FileName)
			fileName := filepath.Base(profile.FileName)
			for _, block := range profile.Blocks {
				functionName, err := getFunctionName(pkgDir+"/"+fileName, block.StartLine)
				if err != nil {
					log.Printf("failed to get function name for %s: %v", test.Test, err)
					continue
				}

				results = append(results, TestCoverageResult{
					TestName:        test.Test,
					Package:         packageName,
					File:            fileName,
					StartLine:       block.StartLine,
					StartColumn:     block.StartCol,
					EndLine:         block.EndLine,
					EndColumn:       block.EndCol,
					StatementNumber: block.NumStmt,
					Count:           block.Count,
					FunctionName:    functionName,
				})
			}
		}
	}

	return results, nil
}

// getFunctionName returns the name of the function at the given line number
func getFunctionName(fileName string, lineNumber int) (string, error) {
	fs := token.NewFileSet()
	node, err := parser.ParseFile(fs, fileName, nil, 0)
	if err != nil {
		return "", fmt.Errorf("failed to parse file: %w", err)
	}

	for _, decl := range node.Decls {
		if funcDecl, ok := decl.(*ast.FuncDecl); ok {
			start := fs.Position(funcDecl.Pos()).Line
			end := fs.Position(funcDecl.End()).Line
			if start <= lineNumber && lineNumber <= end {
				return funcDecl.Name.Name, nil
			}
		}
	}

	return "", fmt.Errorf("function not found at line %d in %s", lineNumber, fileName)
}

func populateTestCoverageResults(ctx context.Context, db *sql.DB, pkgDir string, testResults []TestEvent) error {
	testCoverageResults, err := collectTestCoverageResults(pkgDir, testResults)
	if err != nil {
		return fmt.Errorf("failed to collect coverage results by test: %w", err)
	}

	for _, result := range testCoverageResults {
		insertSQL := `INSERT INTO test_coverage (test_name, package, file, start_line, start_col, end_line, end_col, stmt_num, count, function_name) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?);`
		_, err := db.ExecContext(ctx, insertSQL, result.TestName, result.Package, result.File, result.StartLine, result.StartColumn, result.EndLine, result.EndColumn, result.StatementNumber, result.Count, result.FunctionName)
		if err != nil {
			return fmt.Errorf("failed to insert test coverage results: %w", err)
		}
	}

	return nil
}
