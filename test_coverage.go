package main

import (
	"context"
	"database/sql"
	"fmt"
	"os/exec"
	"path/filepath"

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

var collectTestCoverageResults = func(pkgDir string, testResults []TestEvent) ([]TestCoverageResult, error) {
	var results []TestCoverageResult

	for _, test := range testResults {
		cmd := exec.Command("go", "test", pkgDir, "-run", "^"+test.Test+"$", "-coverprofile="+test.Test+".out")
		cmd.Run()

		profiles, err := cover.ParseProfiles(test.Test + ".out")
		if err != nil {
			return nil, err
		}

		for _, profile := range profiles {
			packageName := filepath.Dir(profile.FileName)
			fileName := filepath.Base(profile.FileName)
			for _, block := range profile.Blocks {
				functionName, err := getFunctionName(pkgDir+"/"+fileName, block.StartLine)
				if err != nil {
					return nil, fmt.Errorf("failed to retrieve function name: %w", err)
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
