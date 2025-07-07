package main

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

func setupTestDB(t *testing.T) *sql.DB {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open in-memory DB: %v", err)
	}

	// Create schema (simplified for this test, or use actual schema)
	_, err = db.Exec(`CREATE TABLE all_code (
		package TEXT NOT NULL,
		file TEXT NOT NULL,
		line_number INTEGER NOT NULL,
		content TEXT NOT NULL
	);`)
	if err != nil {
		t.Fatalf("Failed to create schema: %v", err)
	}
	return db
}

func TestCollectCodeLines(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "testcollectcode")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Structure:
	// tmpDir/
	//   - file1.go
	//   - subpkg/
	//     - file2.go
	//   - file3.txt (should be ignored)

	filePath1 := filepath.Join(tmpDir, "file1.go")
	content1 := "package main\n\nfunc main() {}"
	if err := os.WriteFile(filePath1, []byte(content1), 0644); err != nil {
		t.Fatalf("Failed to write file1.go: %v", err)
	}

	subPkgDir := filepath.Join(tmpDir, "subpkg")
	if err := os.Mkdir(subPkgDir, 0755); err != nil {
		t.Fatalf("Failed to create subpkg dir: %v", err)
	}
	filePath2 := filepath.Join(subPkgDir, "file2.go")
	content2 := "package subpkg\n\nvar X = 10"
	if err := os.WriteFile(filePath2, []byte(content2), 0644); err != nil {
		t.Fatalf("Failed to write file2.go: %v", err)
	}

	filePath3 := filepath.Join(tmpDir, "file3.txt")
	content3 := "This is not a go file."
	if err := os.WriteFile(filePath3, []byte(content3), 0644); err != nil {
		t.Fatalf("Failed to write file3.txt: %v", err)
	}

	// Create an empty .go file
	emptyFilePath := filepath.Join(tmpDir, "empty.go")
	if _, err := os.Create(emptyFilePath); err != nil {
		t.Fatalf("Failed to create empty.go: %v", err)
	}


	expected := []CodeLine{
		{Package: filepath.Base(tmpDir), File: "file1.go", LineNumber: 1, Content: "package main"},
		{Package: filepath.Base(tmpDir), File: "file1.go", LineNumber: 2, Content: ""},
		{Package: filepath.Base(tmpDir), File: "file1.go", LineNumber: 3, Content: "func main() {}"},
		{Package: filepath.Join(filepath.Base(tmpDir), "subpkg"), File: "file2.go", LineNumber: 1, Content: "package subpkg"},
		{Package: filepath.Join(filepath.Base(tmpDir), "subpkg"), File: "file2.go", LineNumber: 2, Content: ""},
		{Package: filepath.Join(filepath.Base(tmpDir), "subpkg"), File: "file2.go", LineNumber: 3, Content: "var X = 10"},
		{Package: filepath.Base(tmpDir), File: "empty.go", LineNumber: 1, Content: ""}, // Empty file has one line with empty content by default from strings.Split
	}

	// Normalize package names in expected results
	for i := range expected {
		expected[i].Package = filepath.ToSlash(expected[i].Package)
	}


	actual, err := collectCodeLines(tmpDir)
	if err != nil {
		t.Fatalf("collectCodeLines() error = %v", err)
	}

	// Normalize package names in actual results
	for i := range actual {
		actual[i].Package = filepath.ToSlash(actual[i].Package)
	}

	// Sort both slices for stable comparison
	sort.Slice(actual, func(i, j int) bool {
		if actual[i].Package != actual[j].Package {
			return actual[i].Package < actual[j].Package
		}
		if actual[i].File != actual[j].File {
			return actual[i].File < actual[j].File
		}
		return actual[i].LineNumber < actual[j].LineNumber
	})
	sort.Slice(expected, func(i, j int) bool {
		if expected[i].Package != expected[j].Package {
			return expected[i].Package < expected[j].Package
		}
		if expected[i].File != expected[j].File {
			return expected[i].File < expected[j].File
		}
		return expected[i].LineNumber < expected[j].LineNumber
	})

	if !reflect.DeepEqual(actual, expected) {
		t.Errorf("collectCodeLines() got = %v, want %v", actual, expected)
	}

	// Test with non-existent directory
	_, err = collectCodeLines(filepath.Join(tmpDir, "nonexistent"))
	if err == nil {
		t.Errorf("collectCodeLines() expected error for non-existent dir, got nil")
	}
}

func TestPopulateCode(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	tmpDir, err := os.MkdirTemp("", "testpopulatecode")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	filePath1 := filepath.Join(tmpDir, "file1.go")
	content1 := "package main\nfunc main() {}"
	if err := os.WriteFile(filePath1, []byte(content1), 0644); err != nil {
		t.Fatalf("Failed to write file1.go: %v", err)
	}

	baseTmpDir := filepath.Base(tmpDir)

	err = populateCode(context.Background(), db, tmpDir)
	if err != nil {
		t.Fatalf("populateCode() error = %v", err)
	}

	rows, err := db.Query("SELECT package, file, line_number, content FROM all_code ORDER BY package, file, line_number")
	if err != nil {
		t.Fatalf("Failed to query all_code: %v", err)
	}
	defer rows.Close()

	var actual []CodeLine
	for rows.Next() {
		var cl CodeLine
		if err := rows.Scan(&cl.Package, &cl.File, &cl.LineNumber, &cl.Content); err != nil {
			t.Fatalf("Failed to scan row: %v", err)
		}
		actual = append(actual, cl)
	}

	expected := []CodeLine{
		{Package: baseTmpDir, File: "file1.go", LineNumber: 1, Content: "package main"},
		{Package: baseTmpDir, File: "file1.go", LineNumber: 2, Content: "func main() {}"},
	}

	// Normalize package names
	for i := range expected {
		expected[i].Package = filepath.ToSlash(expected[i].Package)
	}
	for i := range actual {
		actual[i].Package = filepath.ToSlash(actual[i].Package)
	}


	if !reflect.DeepEqual(actual, expected) {
		t.Errorf("populateCode() data in DB = %v, want %v", actual, expected)
	}
}
