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
	"os"
	"path/filepath"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

const testSchemaSQL = `
CREATE TABLE all_tests (
	"time" TIMESTAMP NOT NULL,
	"action" TEXT NOT NULL,
	status TEXT NOT NULL,
	package TEXT NOT NULL,
	test TEXT NOT NULL,
	parent_test TEXT NULL,
	is_subtest INTEGER NOT NULL DEFAULT 0,
	is_parent INTEGER NOT NULL DEFAULT 0,
	elapsed NUMERIC NULL,
	"output" TEXT NULL,
	PRIMARY KEY (package, test)
);

CREATE TABLE all_coverage (
	package TEXT NOT NULL,
	file TEXT NOT NULL,
	start_line INTEGER NOT NULL,
	start_col INTEGER NOT NULL,
	end_line INTEGER NOT NULL,
	end_col INTEGER NOT NULL,
	stmt_num INTEGER NOT NULL,
	count INTEGER NOT NULL,
	function_name TEXT NOT NULL
);

CREATE TABLE test_coverage (
	test_package TEXT NOT NULL DEFAULT '',
	test_name TEXT NOT NULL,
	package TEXT NOT NULL,
	file TEXT NOT NULL,
	start_line INTEGER NOT NULL,
	start_col INTEGER NOT NULL,
	end_line INTEGER NOT NULL,
	end_col INTEGER NOT NULL,
	stmt_num INTEGER NOT NULL,
	count INTEGER NOT NULL,
	function_name TEXT NULL
);

CREATE TABLE all_code (
	package TEXT NOT NULL,
	file TEXT NOT NULL,
	line_number INTEGER NOT NULL,
	content TEXT NOT NULL
);

CREATE VIEW leaf_tests AS
SELECT * FROM all_tests WHERE is_parent = 0;

CREATE VIEW failed_tests AS
SELECT package, test, elapsed, "output"
FROM all_tests
WHERE action = 'fail' AND is_parent = 0;

CREATE VIEW passed_tests AS
SELECT package, test, elapsed
FROM all_tests
WHERE action = 'pass' AND is_parent = 0;

CREATE VIEW code_coverage AS
SELECT
	ac.package,
	ac.file,
	ac.line_number,
	ac.content,
	MAX(IFNULL(cov.count, 0)) AS covered
FROM all_code ac
LEFT JOIN all_coverage cov
  ON ac.file = cov.file
 AND ac.line_number BETWEEN cov.start_line AND cov.end_line
WHERE ac.file NOT LIKE '%_test.go'
GROUP BY ac.package, ac.file, ac.line_number, ac.content;
`

func setupIntegrationTest(t *testing.T) (string, func()) {
	tmpDir, err := os.MkdirTemp("", "integration-test-")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	goModContent := "module example.com/test-project\n\ngo 1.24\n"
	goModPath := filepath.Join(tmpDir, "go.mod")
	if err := os.WriteFile(goModPath, []byte(goModContent), 0644); err != nil {
		t.Fatalf("Failed to write go.mod: %v", err)
	}

	mainContent := `package main

func Add(a, b int) int {
	return a + b
}

func Sub(a, b int) int {
	return a - b
}
`
	mainPath := filepath.Join(tmpDir, "main.go")
	if err := os.WriteFile(mainPath, []byte(mainContent), 0644); err != nil {
		t.Fatalf("Failed to write main.go: %v", err)
	}

	mainTestContent := `package main

import "testing"

func TestAdd(t *testing.T) {
	t.Run("positive", func(t *testing.T) {
		if Add(1, 2) != 3 {
			t.Error("1 + 2 should be 3")
		}
	})
	t.Run("zero", func(t *testing.T) {
		if Add(0, 0) != 0 {
			t.Error("0 + 0 should be 0")
		}
	})
}

func TestFail(t *testing.T) {
	t.Error("this test is designed to fail")
}
`
	mainTestPath := filepath.Join(tmpDir, "main_test.go")
	if err := os.WriteFile(mainTestPath, []byte(mainTestContent), 0644); err != nil {
		t.Fatalf("Failed to write main_test.go: %v", err)
	}

	return tmpDir, func() {
		_ = os.RemoveAll(tmpDir)
	}
}

func TestPopulateTestResults(t *testing.T) {
	tmpDir, cleanup := setupIntegrationTest(t)
	defer cleanup()

	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current working directory: %v", err)
	}
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("Failed to change directory: %v", err)
	}
	defer func() { _ = os.Chdir(oldWd) }()

	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open in-memory database: %v", err)
	}
	defer func() { _ = db.Close() }()

	if _, err := db.Exec(testSchemaSQL); err != nil {
		t.Fatalf("Failed to create schema: %v", err)
	}

	testEvents, err := PopulateTestResults(context.Background(), db, []string{"./..."})
	if err != nil {
		t.Fatalf("PopulateTestResults failed: %v", err)
	}

	if err := PopulateCoverageResults(context.Background(), db, []string{"./..."}); err != nil {
		t.Fatalf("PopulateCoverageResults failed: %v", err)
	}

	if err := PopulateTestCoverageResults(context.Background(), db, []string{"./..."}, testEvents); err != nil {
		t.Fatalf("PopulateTestCoverageResults failed: %v", err)
	}

	if err := PopulateCode(context.Background(), db, []string{"."}); err != nil {
		t.Fatalf("PopulateCode failed: %v", err)
	}

	// 1. Verify leaf_tests excludes the parent container TestAdd
	var leafCount, totalEvents int
	if err := db.QueryRow("SELECT COUNT(*) FROM leaf_tests").Scan(&leafCount); err != nil {
		t.Fatalf("Failed to count leaf_tests: %v", err)
	}
	if err := db.QueryRow("SELECT COUNT(*) FROM all_tests").Scan(&totalEvents); err != nil {
		t.Fatalf("Failed to count all_tests: %v", err)
	}
	if totalEvents != 4 || leafCount != 3 {
		t.Errorf("Expected 4 total events and 3 leaf tests, got total=%d leaf=%d", totalEvents, leafCount)
	}

	// 2. Regression check: code_coverage WHERE covered > 0 must be non-zero
	var coveredLines int
	if err := db.QueryRow("SELECT COUNT(*) FROM code_coverage WHERE covered > 0").Scan(&coveredLines); err != nil {
		t.Fatalf("Failed to query code_coverage: %v", err)
	}
	if coveredLines == 0 {
		t.Errorf("Regression: expected code_coverage WHERE covered > 0 to be non-zero, got 0")
	}

	// 3. Regression check: test_coverage must be populated for passing leaf subtests
	var testCovRows int
	if err := db.QueryRow("SELECT COUNT(*) FROM test_coverage WHERE count > 0").Scan(&testCovRows); err != nil {
		t.Fatalf("Failed to query test_coverage: %v", err)
	}
	if testCovRows == 0 {
		t.Errorf("Regression: expected test_coverage WHERE count > 0 to be non-zero, got 0")
	}
}
