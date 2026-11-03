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
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"golang.org/x/tools/cover"
)

// TestCoverageResult represents the structure of a test-specific coverage result.
type TestCoverageResult struct {
	TestPackage     string `json:"test_package"`
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

// buildTestRunRegex builds a `go test -run` pattern that anchors each `/`-separated
// subtest segment independently so subtests and special regex characters match accurately.
func buildTestRunRegex(testName string) string {
	parts := strings.Split(testName, "/")
	for i, part := range parts {
		parts[i] = "^" + regexp.QuoteMeta(part) + "$"
	}
	return strings.Join(parts, "/")
}

func collectTestCoverageResults(pkgDirs []string, testResults []TestEvent) ([]TestCoverageResult, error) {
	mc, err := resolveModuleContext(pkgDirs)
	if err != nil {
		return nil, err
	}

	tmpDir, err := os.MkdirTemp("", "testquery-cov-*")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp coverage dir: %w", err)
	}
	defer func() {
		_ = os.RemoveAll(tmpDir)
	}()

	// Group passing leaf tests by package so we compile each package's test binary once.
	testsByPkg := make(map[string][]TestEvent)
	var pkgOrder []string
	for _, test := range testResults {
		if test.IsParent != 0 {
			continue
		}
		if test.Action != "pass" {
			log.Printf("test failed, skipping coverage for %s", test.Test)
			continue
		}
		if _, exists := testsByPkg[test.Package]; !exists {
			pkgOrder = append(pkgOrder, test.Package)
		}
		testsByPkg[test.Package] = append(testsByPkg[test.Package], test)
	}

	finderCache := make(map[string]*functionFinder)
	var results []TestCoverageResult

	for pkgIdx, pkgName := range pkgOrder {
		pkgTests := testsByPkg[pkgName]
		pkgDir := mc.resolvePackageDir(pkgName)
		testPkgNorm := mc.normalizePackage(pkgName)

		// Precompile coverage-instrumented test binary once for this package.
		testBin := filepath.Join(tmpDir, fmt.Sprintf("pkg_%d_%s.test", pkgIdx, sanitizeTestName(pkgName)))
		buildCmd := exec.Command("go", "test", "-c", "-cover", "-o", testBin, pkgDir)
		if mc.projectRoot != "" {
			buildCmd.Dir = mc.projectRoot
		}
		hasTestBin := buildCmd.Run() == nil

		for testIdx, test := range pkgTests {
			coverageFile := filepath.Join(tmpDir, fmt.Sprintf("cov_%d_%d_%s.out", pkgIdx, testIdx, sanitizeTestName(test.Test)))
			runRegex := buildTestRunRegex(test.Test)

			var cmd *exec.Cmd
			if hasTestBin {
				cmd = exec.Command(testBin, "-test.run", runRegex, "-test.coverprofile="+coverageFile)
				cmd.Dir = pkgDir
			} else {
				cmd = exec.Command("go", "test", pkgDir, "-run", runRegex, "-coverprofile="+coverageFile)
				if mc.projectRoot != "" {
					cmd.Dir = mc.projectRoot
				}
			}

			if runErr := cmd.Run(); runErr != nil {
				_ = os.Remove(coverageFile)
				if _, ok := runErr.(*exec.ExitError); ok {
					log.Printf("test failed, skipping coverage for %s", test.Test)
					continue
				}
				return nil, fmt.Errorf("failed to run go test for coverage: %w", runErr)
			}

			profiles, parseErr := cover.ParseProfiles(coverageFile)
			_ = os.Remove(coverageFile)
			if parseErr != nil {
				log.Printf("failed to parse coverage profile for %s: %v", test.Test, parseErr)
				continue
			}

			for _, profile := range profiles {
				finder, ok := finderCache[profile.FileName]
				if !ok {
					f, fErr := newFunctionFinderWithContext(profile.FileName, mc)
					if fErr != nil {
						// Also try resolving relative to pkgDir
						relCandidate := filepath.Join(pkgDir, filepath.Base(profile.FileName))
						f, fErr = newFunctionFinderWithContext(relCandidate, mc)
					}
					if fErr == nil {
						finder = f
						finderCache[profile.FileName] = finder
					}
				}

				normFile, normPkg := mc.normalizeFileAndPackage(profile.FileName, testPkgNorm)

				for _, block := range profile.Blocks {
					var functionName string
					if finder != nil {
						functionName = finder.findFunction(block.StartLine)
					}

					results = append(results, TestCoverageResult{
						TestPackage:     testPkgNorm,
						TestName:        test.Test,
						Package:         normPkg,
						File:            normFile,
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
	}

	return results, nil
}

func PopulateTestCoverageResults(ctx context.Context, db *sql.DB, pkgDirs []string, testResults []TestEvent) error {
	testCoverageResults, err := collectTestCoverageResults(pkgDirs, testResults)
	if err != nil {
		return fmt.Errorf("failed to collect coverage results by test: %w", err)
	}

	stmt, err := db.PrepareContext(ctx, `INSERT INTO test_coverage (test_package, test_name, package, file, start_line, start_col, end_line, end_col, stmt_num, count, function_name) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);`)
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer func() {
		_ = stmt.Close()
	}()

	for _, result := range testCoverageResults {
		_, err := stmt.ExecContext(ctx, result.TestPackage, result.TestName, result.Package, result.File, result.StartLine, result.StartColumn, result.EndLine, result.EndColumn, result.StatementNumber, result.Count, result.FunctionName)
		if err != nil {
			return fmt.Errorf("failed to insert test coverage results for test %q: %w", result.TestName, err)
		}
	}

	return nil
}
