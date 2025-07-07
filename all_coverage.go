package main

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"

	"golang.org/x/tools/cover"
)

// CoverageResult represents the structure of a coverage result
type CoverageResult struct {
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

var collectCoverageResults = func(pkgDir string) ([]CoverageResult, error) {
	profiles, err := cover.ParseProfiles("coverage.out")
	if err != nil {
		return nil, err
	}

	var results []CoverageResult
	for _, profile := range profiles {
		packageName := filepath.Dir(profile.FileName)
		fileName := filepath.Base(profile.FileName)
		for _, block := range profile.Blocks {
			functionName, err := getFunctionName(pkgDir+"/"+fileName, block.StartLine)
			if err != nil {
				return nil, fmt.Errorf("failed to retrieve function name: %w", err)
			}

			results = append(results, CoverageResult{
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

	return results, nil
}

func populateCoverageResults(ctx context.Context, db *sql.DB, pkgDir string) error {
	coverageResults, err := collectCoverageResults(pkgDir)
	if err != nil {
		return fmt.Errorf("failed to collect coverage results: %w", err)
	}

	for _, result := range coverageResults {
		insertSQL := `INSERT INTO all_coverage (package, file, start_line, start_col, end_line, end_col, stmt_num, count, function_name) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?);`
		_, err := db.ExecContext(ctx, insertSQL, result.Package, result.File, result.StartLine, result.StartColumn, result.EndLine, result.EndColumn, result.StatementNumber, result.Count, result.FunctionName)
		if err != nil {
			return fmt.Errorf("failed to insert coverage results: %w", err)
		}
	}
	return nil
}
