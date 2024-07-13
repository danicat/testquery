package main

import (
	"fmt"
	"path/filepath"

	"golang.org/x/tools/cover"
)

// CoverageResult represents the structure of a coverage result
type CoverageResult struct {
	Package         string `json:"package"`
	File            string `json:"file"`
	FromLine        int    `json:"from_line"`
	FromColumn      int    `json:"from_col"`
	ToLine          int    `json:"to_line"`
	ToColumn        int    `json:"to_col"`
	StatementNumber int    `json:"stmt_num"`
	Count           int    `json:"count"`
	FunctionName    string `json:"function_name"`
}

func collectCoverageResults(pkgDir string) ([]CoverageResult, error) {
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
				FromLine:        block.StartLine,
				FromColumn:      block.StartCol,
				ToLine:          block.EndLine,
				ToColumn:        block.EndCol,
				StatementNumber: block.NumStmt,
				Count:           block.Count,
				FunctionName:    functionName,
			})
		}
	}

	return results, nil
}
