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

package database

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"

	_ "embed"

	"github.com/danicat/testquery/internal/collector"
)

//go:embed sql/schema.sql
var DDL string

type PopulateOptions struct {
	PerTestCoverage bool
}

func CreateTables(db *sql.DB) error {
	_, err := db.Exec(DDL)
	return err
}

func PopulateTablesWithOptions(db *sql.DB, pkgDirs []string, opts PopulateOptions) error {
	testResults, err := collector.PopulateTestResults(context.Background(), db, pkgDirs)
	if err != nil {
		return fmt.Errorf("failed to populate test results: %w", err)
	}

	if err := collector.PopulateCoverageResults(context.Background(), db, pkgDirs); err != nil {
		return fmt.Errorf("failed to populate coverage results: %w", err)
	}

	if opts.PerTestCoverage {
		if err := collector.PopulateTestCoverageResults(context.Background(), db, pkgDirs, testResults); err != nil {
			return fmt.Errorf("failed to populate test coverage results: %w", err)
		}
	}

	if _, err := db.Exec("INSERT OR REPLACE INTO metadata (key, value) VALUES (?, ?)", "per_test_coverage", strconv.FormatBool(opts.PerTestCoverage)); err != nil {
		return fmt.Errorf("failed to insert per_test_coverage metadata: %w", err)
	}

	if err := collector.PopulateCode(context.Background(), db, pkgDirs); err != nil {
		return fmt.Errorf("failed to populate code: %w", err)
	}

	return nil
}
