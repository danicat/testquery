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

package query

import (
	"bytes"
	"database/sql"
	"strings"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

func TestExecute(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open in-memory database: %v", err)
	}
	defer func() { _ = db.Close() }()

	schema := `CREATE TABLE test (id INTEGER, name TEXT);`
	if _, err := db.Exec(schema); err != nil {
		t.Fatalf("Failed to create schema: %v", err)
	}

	insert := `INSERT INTO test (id, name) VALUES (1, 'foo'), (2, 'bar');`
	if _, err := db.Exec(insert); err != nil {
		t.Fatalf("Failed to insert data: %v", err)
	}

	var buf bytes.Buffer
	if err := Execute(&buf, db, "SELECT * FROM test", "table"); err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	expected := `
+----+------+
| ID | NAME |
+----+------+
|  1 | foo  |
|  2 | bar  |
+----+------+
`
	got := strings.TrimSpace(buf.String())
	want := strings.TrimSpace(expected)

	if got != want {
		t.Errorf("Execute() got = %v, want %v", got, want)
	}
}

func TestExecuteTableTruncation(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open in-memory database: %v", err)
	}
	defer func() { _ = db.Close() }()

	if _, err := db.Exec(`CREATE TABLE selene_tests (test_name TEXT, killed_mutant_ids TEXT);`); err != nil {
		t.Fatalf("Failed to create schema: %v", err)
	}

	longValue := strings.Repeat("a", 100)
	if _, err := db.Exec(`INSERT INTO selene_tests VALUES ('TestFoo', ?);`, longValue); err != nil {
		t.Fatalf("Failed to insert data: %v", err)
	}

	var truncatedBuf bytes.Buffer
	if err := Execute(&truncatedBuf, db, "SELECT * FROM selene_tests", "table"); err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	if !strings.Contains(truncatedBuf.String(), "... (+20 chars)") {
		t.Errorf("expected truncated output to contain '... (+20 chars)', got:\n%s", truncatedBuf.String())
	}

	var fullBuf bytes.Buffer
	if err := Execute(&fullBuf, db, "SELECT * FROM selene_tests", "table", WithNoTruncate(true)); err != nil {
		t.Fatalf("Execute with WithNoTruncate failed: %v", err)
	}
	if !strings.Contains(fullBuf.String(), longValue) {
		t.Errorf("expected full output to contain full 100-char value, got:\n%s", fullBuf.String())
	}
}

func TestExecuteDynamicErrorDiagnostics(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open in-memory database: %v", err)
	}
	defer func() { _ = db.Close() }()

	_, err = db.Exec(`
		CREATE TABLE all_tests (package TEXT, test TEXT, action TEXT);
		CREATE TABLE selene_tests (test_name TEXT, package TEXT, status TEXT, mutations_killed INTEGER);
		CREATE VIEW passed_tests AS SELECT package, test FROM all_tests WHERE action = 'pass';
	`)
	if err != nil {
		t.Fatalf("Failed to create schema: %v", err)
	}

	var buf bytes.Buffer
	queryErr := Execute(&buf, db, "SELECT nonexistent_col FROM selene_tests", "table")
	if queryErr == nil {
		t.Fatalf("expected error for invalid column, got nil")
	}

	msg := queryErr.Error()
	for _, expectedSubstr := range []string{
		"Columns in selene_tests: test_name, package, status, mutations_killed",
		"Available tables: all_tests, selene_tests",
		"Available views: passed_tests",
	} {
		if !strings.Contains(msg, expectedSubstr) {
			t.Errorf("expected error diagnostic to contain %q, got:\n%s", expectedSubstr, msg)
		}
	}
}
