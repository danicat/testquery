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

package shell

import (
	"bytes"
	"context"
	"database/sql"
	"io"
	"strings"
	"sync"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

func TestPrompt(t *testing.T) {
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
	insert := `INSERT INTO test (id, name) VALUES (1, 'foo');`
	if _, err := db.Exec(insert); err != nil {
		t.Fatalf("Failed to insert data: %v", err)
	}

	// Use a pipe to simulate user input
	r, w := io.Pipe()
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		defer w.Close()
		io.WriteString(w, "SELECT * FROM test;\n")
	}()

	// Capture the output in a buffer
	var outBuf bytes.Buffer
	err = Prompt(context.Background(), db, r, &outBuf)
	if err != nil && err.Error() != "failed to read line: EOF" {
		t.Fatalf("Prompt failed: %v", err)
	}

	wg.Wait()

	// Define the expected output
	expected := `
+----+------+
| ID | NAME |
+----+------+
|  1 | foo  |
+----+------+
`
	// Trim whitespace for a more robust comparison
	got := strings.TrimSpace(outBuf.String())
	want := strings.TrimSpace(expected)

	if !strings.Contains(got, want) {
		t.Errorf("Prompt() got = %v, want %v", got, want)
	}
}
