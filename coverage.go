package main

import (
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
}

func collectCoverageResults() ([]CoverageResult, error) {
	profiles, err := cover.ParseProfiles("coverage.out")
	if err != nil {
		return nil, err
	}

	var results []CoverageResult
	for _, profile := range profiles {
		packageName := filepath.Dir(profile.FileName)
		fileName := filepath.Base(profile.FileName)
		for _, block := range profile.Blocks {
			results = append(results, CoverageResult{
				Package:         packageName,
				File:            fileName,
				FromLine:        block.StartLine,
				FromColumn:      block.StartCol,
				ToLine:          block.EndLine,
				ToColumn:        block.EndCol,
				StatementNumber: block.NumStmt,
				Count:           block.Count,
			})
		}
	}

	return results, nil
}
