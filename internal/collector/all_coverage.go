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
	"go/ast"
	"go/parser"
	"go/token"
	"os"

	"golang.org/x/tools/cover"
)

// CoverageResult represents the structure of a coverage result.
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
	if _, err := os.Stat("coverage.out"); os.IsNotExist(err) {
		return nil, nil
	}

	mc, err := resolveModuleContext(pkgDirs)
	if err != nil {
		return nil, err
	}

	profiles, err := cover.ParseProfiles("coverage.out")
	if err != nil {
		return nil, fmt.Errorf("failed to parse coverage profiles: %w", err)
	}

	var results []CoverageResult
	for _, profile := range profiles {
		functionFinder, err := newFunctionFinderWithContext(profile.FileName, mc)
		if err != nil {
			return nil, fmt.Errorf("failed to create function finder: %w", err)
		}

		normFile, normPkg := mc.normalizeFileAndPackage(profile.FileName, "")

		for _, block := range profile.Blocks {
			results = append(results, CoverageResult{
				Package:         normPkg,
				File:            normFile,
				StartLine:       block.StartLine,
				StartColumn:     block.StartCol,
				EndLine:         block.EndLine,
				EndColumn:       block.EndCol,
				StatementNumber: block.NumStmt,
				Count:           block.Count,
				FunctionName:    functionFinder.findFunction(block.StartLine),
			})
		}
	}

	return results, nil
}

func PopulateCoverageResults(ctx context.Context, db *sql.DB, pkgDirs []string) error {
	defer func() {
		_ = os.Remove("coverage.out")
	}()

	coverageResults, err := collectCoverageResults(pkgDirs)
	if err != nil {
		return fmt.Errorf("failed to collect coverage results: %w", err)
	}

	stmt, err := db.PrepareContext(ctx, `INSERT INTO all_coverage (package, file, start_line, start_col, end_line, end_col, stmt_num, count, function_name) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?);`)
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer func() {
		_ = stmt.Close()
	}()

	for _, result := range coverageResults {
		_, err := stmt.ExecContext(ctx, result.Package, result.File, result.StartLine, result.StartColumn, result.EndLine, result.EndColumn, result.StatementNumber, result.Count, result.FunctionName)
		if err != nil {
			return fmt.Errorf("failed to insert coverage results: %w", err)
		}
	}
	return nil
}

type functionFinder struct {
	fset         *token.FileSet
	file         *ast.File
	lineMap      map[int]string
	fullFilePath string
}

func newFunctionFinderWithContext(fileName string, mc *moduleContext) (*functionFinder, error) {
	if mc == nil {
		var err error
		mc, err = resolveModuleContext(nil)
		if err != nil {
			return nil, err
		}
	}

	fullFilePath, err := mc.resolveFileOnDisk(fileName)
	if err != nil {
		return nil, err
	}

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, fullFilePath, nil, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to parse file %s: %w", fullFilePath, err)
	}

	return &functionFinder{
		fset:         fset,
		file:         file,
		lineMap:      make(map[int]string),
		fullFilePath: fullFilePath,
	}, nil
}

func (f *functionFinder) findFunction(line int) string {
	if fn, ok := f.lineMap[line]; ok {
		return fn
	}

	var funcName string
	ast.Inspect(f.file, func(n ast.Node) bool {
		if fn, ok := n.(*ast.FuncDecl); ok {
			startLine := f.fset.Position(fn.Pos()).Line
			endLine := f.fset.Position(fn.End()).Line
			if line >= startLine && line <= endLine {
				funcName = fn.Name.Name
				for i := startLine; i <= endLine; i++ {
					f.lineMap[i] = funcName
				}
				return false
			}
		}
		return true
	})

	return funcName
}
