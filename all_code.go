package main

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type CodeLine struct {
	Package    string `json:"package"`
	File       string `json:"file"`
	LineNumber int    `json:"line_number"`
	Content    string `json:"content"`
}

// collectCodeLines collects all lines of code from Go files
func collectCodeLines(pkgDir string) ([]CodeLine, error) {
	var results []CodeLine

	err := filepath.Walk(pkgDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && strings.HasSuffix(info.Name(), ".go") {
			packageName := filepath.Dir(path)
			fileName := filepath.Base(path)

			data, err := os.ReadFile(path)
			if err != nil {
				return err
			}

			lines := strings.Split(string(data), "\n")
			for i, line := range lines {
				results = append(results, CodeLine{
					Package:    packageName,
					File:       fileName,
					LineNumber: i + 1,
					Content:    line,
				})
			}
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to extract lines of code: %w", err)
	}

	return results, nil
}

func populateCode(ctx context.Context, db *sql.DB, pkgDir string) error {
	allCode, err := collectCodeLines(pkgDir)
	if err != nil {
		return fmt.Errorf("failed to collect coverage results: %w", err)
	}

	for _, result := range allCode {
		insertSQL := `INSERT INTO all_code (package, file, line_number, content) VALUES (?, ?, ?, ?);`
		_, err := db.ExecContext(ctx, insertSQL, result.Package, result.File, result.LineNumber, result.Content)
		if err != nil {
			return fmt.Errorf("failed to insert code lines: %w", err)
		}
	}
	return nil
}
