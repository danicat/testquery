// Copyright 2025 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package query

import (
	"bytes"
	"database/sql"
	"strings"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

func TestExecute(t *testing.T) {
	// Create an in-memory SQLite database
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open in-memory database: %v", err)
	}
	defer db.Close()

	// Create a table and insert some data
	schema := `CREATE TABLE test (id INTEGER, name TEXT);`
	if _, err := db.Exec(schema); err != nil {
		t.Fatalf("Failed to create schema: %v", err)
	}

	insert := `INSERT INTO test (id, name) VALUES (1, 'foo'), (2, 'bar');`
	if _, err := db.Exec(insert); err != nil {
		t.Fatalf("Failed to insert data: %v", err)
	}

	// Execute a query and capture the output
	var buf bytes.Buffer
	if err := Execute(&buf, db, "SELECT * FROM test"); err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	// Define the expected output
	expected := `
+----+------+
| ID | NAME |
+----+------+
|  1 | foo  |
|  2 | bar  |
+----+------+
`
	// Trim whitespace for a more robust comparison
	got := strings.TrimSpace(buf.String())
	want := strings.TrimSpace(expected)

	if got != want {
		t.Errorf("Execute() got = %v, want %v", got, want)
	}
}
