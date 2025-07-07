package main

import (
	"bytes"
	"context"
	"database/sql"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/chzyer/readline"
	_ "github.com/mattn/go-sqlite3"
	// "github.com/jedib0t/go-pretty/v6/table" // Was unused
)

func setupMainTestDB(t *testing.T) *sql.DB {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open in-memory DB: %v", err)
	}

	// Create a simple table for testing queries
	_, err = db.Exec(`
		CREATE TABLE items (id INTEGER PRIMARY KEY, name TEXT, price REAL);
		INSERT INTO items (id, name, price) VALUES (1, 'Apple', 0.5);
		INSERT INTO items (id, name, price) VALUES (2, 'Banana', 0.75);
		INSERT INTO items (id, name, price) VALUES (3, 'Cherry', 1.25);
	`)
	if err != nil {
		t.Fatalf("Failed to create schema and insert data: %v", err)
	}
	return db
}

func TestExecuteQuery(t *testing.T) {
	db := setupMainTestDB(t)
	defer db.Close()

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	query := "SELECT id, name, price FROM items WHERE price < 1.0 ORDER BY id;"
	err := executeQuery(db, query)

	w.Close()
	os.Stdout = oldStdout // Restore stdout

	if err != nil {
		t.Fatalf("executeQuery() error = %v", err)
	}

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	// Construct expected table output manually for simple cases
	// Using go-pretty/table's default style
	expectedOutputLines := []string{
		"+----+--------+-------+",
		"| ID | NAME   | PRICE |",
		"+----+--------+-------+",
		"|  1 | Apple  |   0.5 |",
		"|  2 | Banana |  0.75 |",
		"+----+--------+-------+",
	}
	expectedOutput := strings.Join(expectedOutputLines, "\n") + "\n" // table.Render adds a newline

	// Normalize line endings for comparison, just in case
	normalizedOutput := strings.ReplaceAll(output, "\r\n", "\n")
	normalizedExpectedOutput := strings.ReplaceAll(expectedOutput, "\r\n", "\n")

	// Trim whitespace from each line and filter out empty lines for more robust comparison
	cleanActualLines := []string{}
	for _, line := range strings.Split(normalizedOutput, "\n") {
		trimmedLine := strings.TrimSpace(line)
		if trimmedLine != "" {
			cleanActualLines = append(cleanActualLines, trimmedLine)
		}
	}

	cleanExpectedLines := []string{}
	for _, line := range strings.Split(normalizedExpectedOutput, "\n") {
		trimmedLine := strings.TrimSpace(line)
		if trimmedLine != "" {
			cleanExpectedLines = append(cleanExpectedLines, trimmedLine)
		}
	}

	if !reflect.DeepEqual(cleanActualLines, cleanExpectedLines) {
		t.Errorf("executeQuery() output mismatch:\nGot:\n%s\nWant:\n%s", normalizedOutput, normalizedExpectedOutput)
	}


	t.Run("invalid query", func(t *testing.T) {
		err := executeQuery(db, "SELECT * FROM non_existent_table;")
		if err == nil {
			t.Errorf("executeQuery() with invalid query expected error, got nil")
		} else if !strings.Contains(err.Error(), "no such table") {
			t.Errorf("executeQuery() with invalid query, wrong error message, got: %v", err)
		}
	})

	t.Run("query with no results", func(t *testing.T) {
		oldStdout := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		err := executeQuery(db, "SELECT id, name FROM items WHERE id > 100;")

		w.Close()
		os.Stdout = oldStdout // Restore stdout

		if err != nil {
			t.Fatalf("executeQuery() for no results, error = %v", err)
		}

		var bufEmpty bytes.Buffer
		bufEmpty.ReadFrom(r)
		outputEmpty := bufEmpty.String()

		expectedEmptyOutputLines := []string{
			"+----+------+",
			"| ID | NAME |",
			"+----+------+",
			"+----+------+",
		}
		expectedEmptyOutput := strings.Join(expectedEmptyOutputLines, "\n") + "\n"

		normalizedOutputEmpty := strings.ReplaceAll(outputEmpty, "\r\n", "\n")
		normalizedExpectedEmptyOutput := strings.ReplaceAll(expectedEmptyOutput, "\r\n", "\n")

		cleanActualEmptyLines := []string{}
		for _, line := range strings.Split(normalizedOutputEmpty, "\n") {
			trimmedLine := strings.TrimSpace(line)
			if trimmedLine != "" {
				cleanActualEmptyLines = append(cleanActualEmptyLines, trimmedLine)
			}
		}

		cleanExpectedEmptyLines := []string{}
		for _, line := range strings.Split(normalizedExpectedEmptyOutput, "\n") {
			trimmedLine := strings.TrimSpace(line)
			if trimmedLine != "" {
				cleanExpectedEmptyLines = append(cleanExpectedEmptyLines, trimmedLine)
			}
		}
		if !reflect.DeepEqual(cleanActualEmptyLines, cleanExpectedEmptyLines) {
			t.Errorf("executeQuery() for no results, output mismatch:\nGot:\n%s\nWant:\n%s", normalizedOutputEmpty, normalizedExpectedEmptyOutput)
		}
	})
}

// Note: Testing the `run` function itself is more complex due to readline and main loop.
// It would typically involve more extensive mocking or refactoring `run` into smaller testable units.
// For now, focusing on `executeQuery` which is a key part of `run`'s functionality.
// Testing flag parsing could be done by manipulating os.Args and calling flag.Parse().
// Example:
// func TestMainFlags(t *testing.T) {
//	 oldArgs := os.Args
//	 defer func() { os.Args = oldArgs }()
//	 os.Args = []string{"cmd", "-pkg", "/test/path", "-query", "SELECT 1"}
//	 // Call your main function or parts of it that use these flags
//	 // This is highly dependent on how main is structured.
// }
// The `prompt` function is also hard to test without a mock readline or by simulating input.
// These are generally tested via integration tests or more specialized UI testing tools.

func TestRunFunctionality_QueryFlag(t *testing.T) {
    // This is a limited test for the -query flag path in run()
    // It relies on executeQuery being tested separately for output correctness.
    ctx := context.Background()

    // Setup in-memory DB for the run function
    db, err := sql.Open("sqlite3", ":memory:")
    if err != nil {
        t.Fatalf("Failed to instantiate sqlite: %v", err)
    }
    defer db.Close()

    err = createTables(ctx, db) // Use actual createTables
    if err != nil {
        t.Fatalf("Failed to apply ddl: %v", err)
    }
    // For this test, we don't need to populate tables, just test query execution path.

    // Override the global db variable used by run if it's not passed (it is in the current run)
    // Or, refactor run to accept a *sql.DB for better testability.
    // The current run function creates its own DB or opens one.

    // To test the query flag, we need to simulate the part of `run` that handles it.
    // The `run` function itself is complex. A simpler approach for this part:
    // Create a temporary DB, then call executeQuery directly as `run` would.

    // Redirect stdout to capture output
    oldStdout := os.Stdout
    r, w, pipeErr := os.Pipe()
    if pipeErr != nil {
        t.Fatalf("Failed to create pipe: %v", pipeErr)
    }
    os.Stdout = w

    // Use a dummy readline instance for functions that require it, though not directly for -query
    dummyRL, rlErr := readline.NewEx(&readline.Config{Prompt: ">", HistoryFile: "/dev/null"})
    if rlErr != nil {
        t.Fatalf("Failed to create dummy readline: %v", rlErr)
    }
    defer dummyRL.Close()

    // Test the -query path of the run function
    // We use a temporary db file name, but since persist is false and open is false,
    // it should use :memory: and then try to persist if persist=true
    // For -query, persist is not the primary concern, but the query execution.
    // Let's ensure it uses an in-memory DB for this test path.
    // The `run` function will create its own in-memory DB if open=false.
    // We'll use a simple query against one of the tables created by `createTables`.
    // e.g., query the structure of all_code table
    // Setup mocks for populateTables data collectors to prevent actual command execution
    setupMainTestPopulateMocks(t)
    defer restoreMainTestPopulateMocks()

    err = run(ctx, ".", dummyRL, false, false, "test_query_flag.db", "SELECT name FROM sqlite_master WHERE type='table' AND name='all_code';")

    w.Close()
    os.Stdout = oldStdout // Restore

    if err != nil {
        t.Errorf("run() with -query flag returned error: %v", err)
    }

    var buf bytes.Buffer
    buf.ReadFrom(r)
    output := buf.String()

    // Expected output for "SELECT name FROM sqlite_master WHERE type='table' AND name='all_code';"
    expectedOutputLines := []string{
        "+----------+",
        "| NAME     |",
        "+----------+",
        "| all_code |",
        "+----------+",
    }
    expectedOutput := strings.Join(expectedOutputLines, "\n") + "\n"

    normalizedOutput := strings.ReplaceAll(output, "\r\n", "\n")
    normalizedExpectedOutput := strings.ReplaceAll(expectedOutput, "\r\n", "\n")

    cleanActualLines := []string{}
    for _, line := range strings.Split(normalizedOutput, "\n") {
        trimmedLine := strings.TrimSpace(line)
        if trimmedLine != "" {
            cleanActualLines = append(cleanActualLines, trimmedLine)
        }
    }

    cleanExpectedLines := []string{}
    for _, line := range strings.Split(normalizedExpectedOutput, "\n") {
        trimmedLine := strings.TrimSpace(line)
        if trimmedLine != "" {
            cleanExpectedLines = append(cleanExpectedLines, trimmedLine)
        }
    }

    if !reflect.DeepEqual(cleanActualLines, cleanExpectedLines) {
        t.Errorf("run() with -query flag, output mismatch:\nGot:\n%s\nWant:\n%s", normalizedOutput, normalizedExpectedOutput)
    }

    // Test the -persist and -open flags (basic check)
    // This test is more of an integration test for these flags.
    // It checks if files are created and can be reopened.
    dbFileName := "test_persist.db"
    defer os.Remove(dbFileName) // Clean up

    // Run with persist to create the DB file
    err = run(ctx, ".", dummyRL, true, false, dbFileName, "SELECT 1;") // A simple query to ensure execution
    if err != nil {
        t.Fatalf("run() with persist=true failed: %v", err)
    }
    if _, errStat := os.Stat(dbFileName); os.IsNotExist(errStat) {
        t.Errorf("run() with persist=true did not create database file '%s'", dbFileName)
    }

    // Run with open=true to open the created DB file
    // Redirect output as it will print query results
    _, wOpen, _ := os.Pipe() // rOpen was unused
    os.Stdout = wOpen

    err = run(ctx, ".", dummyRL, false, true, dbFileName, "SELECT 1;")
    wOpen.Close()
    os.Stdout = oldStdout // Restore

    if err != nil {
        t.Errorf("run() with open=true failed: %v", err)
    }
}

// Mock readline for testing the prompt function (if it were to be tested directly)
// type mockReadline struct {
// 	linesToRead []string
// 	idx         int
//  prompt      string
// }
// func (m *mockReadline) Readline() (string, error) {
// 	if m.idx < len(m.linesToRead) {
// 		line := m.linesToRead[m.idx]
// 		m.idx++
// 		return line, nil
// 	}
// 	return "", io.EOF // Simulate end of input
// }
// func (m *mockReadline) SetPrompt(p string) { m.prompt = p }
// func (m *mockReadline) SaveHistory(line string) bool { return true }
// func (m *mockReadline) Close() error { return nil }

// func TestPrompt(t *testing.T) {
// This would require a mock readline and a way to feed input and check output/db state.
// }
