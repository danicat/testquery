package collector

import (
	"context"
	"database/sql"
	"fmt"

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

func collectCoverageResults(pkgDirs []string) ([]CoverageResult, error) {
	profiles, err := cover.ParseProfiles("coverage.out")
	if err != nil {
		return nil, fmt.Errorf("failed to parse coverage profiles: %w", err)
	}

	var results []CoverageResult
	for _, profile := range profiles {
		for _, block := range profile.Blocks {
			results = append(results, CoverageResult{
				Package:         profile.FileName,
				File:            profile.FileName,
				StartLine:       block.StartLine,
				StartColumn:     block.StartCol,
				EndLine:         block.EndLine,
				EndColumn:       block.EndCol,
				StatementNumber: block.NumStmt,
				Count:           block.Count,
				FunctionName:    "",
			})
		}
	}

	return results, nil
}

func PopulateCoverageResults(ctx context.Context, db *sql.DB, pkgDirs []string) error {
	coverageResults, err := collectCoverageResults(pkgDirs)
	if err != nil {
		return fmt.Errorf("failed to collect coverage results: %w", err)
	}

	stmt, err := db.PrepareContext(ctx, `INSERT INTO all_coverage (package, file, start_line, start_col, end_line, end_col, stmt_num, count, function_name) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?);`)
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()

	for _, result := range coverageResults {
		_, err := stmt.ExecContext(ctx, result.Package, result.File, result.StartLine, result.StartColumn, result.EndLine, result.EndColumn, result.StatementNumber, result.Count, result.FunctionName)
		if err != nil {
			return fmt.Errorf("failed to insert coverage results: %w", err)
		}
	}
	return nil
}
