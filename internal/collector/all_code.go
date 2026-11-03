// Copyright (c) 2026 Google LLC
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

package collector

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

// collectCodeLines collects all lines of code from Go files.
func collectCodeLines(pkgDirs []string) ([]CodeLine, error) {
	mc, err := resolveModuleContext(pkgDirs)
	if err != nil {
		return nil, err
	}

	var results []CodeLine
	seenFiles := make(map[string]bool)

	for _, dir := range pkgDirs {
		if dir == "" {
			continue
		}
		walkDir := strings.TrimSuffix(strings.TrimSuffix(dir, "/..."), "...")
		if walkDir == "" {
			walkDir = "."
		}
		err := filepath.Walk(walkDir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if info.IsDir() {
				if info.Name() != "." && strings.HasPrefix(info.Name(), ".") {
					return filepath.SkipDir
				}
				return nil
			}
			if strings.HasSuffix(info.Name(), ".go") {
				normFile, normPkg := mc.normalizeFileAndPackage(path, filepath.Dir(path))
				if seenFiles[normFile] {
					return nil
				}
				seenFiles[normFile] = true

				data, err := os.ReadFile(path)
				if err != nil {
					return fmt.Errorf("failed to read file %s: %w", path, err)
				}

				lines := strings.Split(string(data), "\n")
				for i, line := range lines {
					results = append(results, CodeLine{
						Package:    normPkg,
						File:       normFile,
						LineNumber: i + 1,
						Content:    line,
					})
				}
			}
			return nil
		})
		if err != nil {
			return nil, fmt.Errorf("failed to extract lines of code from %s: %w", dir, err)
		}
	}

	return results, nil
}

func PopulateCode(ctx context.Context, db *sql.DB, pkgDirs []string) error {
	allCode, err := collectCodeLines(pkgDirs)
	if err != nil {
		return fmt.Errorf("failed to collect code lines: %w", err)
	}

	stmt, err := db.PrepareContext(ctx, `INSERT INTO all_code (package, file, line_number, content) VALUES (?, ?, ?, ?);`)
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer func() {
		_ = stmt.Close()
	}()

	for _, result := range allCode {
		_, err := stmt.ExecContext(ctx, result.Package, result.File, result.LineNumber, result.Content)
		if err != nil {
			return fmt.Errorf("failed to insert code line for %s/%s:%d: %w", result.Package, result.File, result.LineNumber, err)
		}
	}

	return nil
}
