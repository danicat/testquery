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

package database

import (
	"database/sql"
	"os"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

func TestCreateTables(t *testing.T) {
	// Create an in-memory SQLite database
	db, err := sql.Open("sqlite3", "file:test?mode=memory&cache=shared")
	if err != nil {
		t.Fatalf("Failed to open in-memory database: %v", err)
	}
	defer db.Close()

	// Call the function we are testing
	if err := CreateTables(db); err != nil {
		t.Fatalf("CreateTables failed: %v", err)
	}

	// Check that the tables were created
	tables := []string{"all_tests", "all_coverage", "test_coverage", "all_code"}
	for _, table := range tables {
		rows, err := db.Query("SELECT name FROM sqlite_master WHERE type='table' AND name=?;", table)
		if err != nil {
			t.Fatalf("Failed to query for table %s: %v", table, err)
		}
		defer rows.Close()
		if !rows.Next() {
			t.Errorf("Table %s was not created", table)
		}
	}

	views := []string{"passed_tests", "failed_tests"}
	for _, view := range views {
		rows, err := db.Query("SELECT name FROM sqlite_master WHERE type='view' AND name=?;", view)
		if err != nil {
			t.Fatalf("Failed to query for view %s: %v", view, err)
		}
		defer rows.Close()
		if !rows.Next() {
			t.Errorf("View %s was not created", view)
		}
	}
}

func TestCreateTablesFromDDL_Error(t *testing.T) {
	// Create an in-memory SQLite database
	db, err := sql.Open("sqlite3", "file:test?mode=memory&cache=shared")
	if err != nil {
		t.Fatalf("Failed to open in-memory database: %v", err)
	}
	defer db.Close()

	// Call the function with a malformed DDL
	ddl := "CREATE TABLE all_tests (id INTEGER malformed); CREATE TABLE b (id INTEGER);"
	CreateTablesFromDDL(db, ddl)
	rows, err := db.Query("SELECT name FROM sqlite_master WHERE type='table' AND name='all_tests';")
	if err != nil {
		t.Fatalf("Failed to query for table all_tests: %v", err)
	}
	defer rows.Close()
	if rows.Next() {
		t.Error("Table all_tests was created, but it should not have been")
	}
}


func TestPersistDatabase(t *testing.T) {
	// Create an in-memory SQLite database
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open in-memory database: %v", err)
	}
	defer db.Close()
	// Create a temporary file for the database
	tmpfile, err := os.CreateTemp("", "test.db")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpfile.Name())

	// Call the function we are testing
	if err := PersistDatabase(db, tmpfile.Name()); err != nil {
		t.Fatalf("PersistDatabase failed: %v", err)
	}

	// Check that the file was created and is not empty
	info, err := os.Stat(tmpfile.Name())
	if err != nil {
		t.Fatalf("Failed to stat database file: %v", err)
	}
	if info.Size() == 0 {
		t.Error("Database file is empty")
	}
}
