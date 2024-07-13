package main

import (
	"bufio"
	"context"
	"database/sql"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"os"

	_ "embed"

	_ "github.com/mattn/go-sqlite3"

	"github.com/jedib0t/go-pretty/v6/table"
)

//go:embed sql/schema.sql
var ddl string

func main() {
	pkgDir := flag.String("pkg", ".", "directory of the package to test")
	flag.Parse()

	ctx := context.Background()
	err := run(ctx, *pkgDir)
	if err != nil {
		log.Fatalln(err)
	}
}

func run(ctx context.Context, pkgDir string) error {
	// Initialize the in-memory SQLite database
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		return fmt.Errorf("failed to instantiate sqlite: %w", err)
	}
	defer db.Close()

	err = createTables(ctx, db)
	if err != nil {
		return fmt.Errorf("failed to apply ddl: %w", err)
	}

	err = populateTables(ctx, db, pkgDir)
	if err != nil {
		return fmt.Errorf("failed to populate tables: %w", err)
	}

	return prompt(ctx, db)
}

func executeQuery(db *sql.DB, query string) error {
	rows, err := db.Query(query)
	if err != nil {
		return fmt.Errorf("failed to run query: %w", err)
	}
	defer rows.Close()

	// Get column names
	columns, err := rows.Columns()
	if err != nil {
		log.Fatal(err)
	}

	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)

	var header = make(table.Row, len(columns))
	for i := range columns {
		header[i] = columns[i]
	}
	t.AppendHeader(header)

	for rows.Next() {
		var values = make(table.Row, len(columns))
		var valuesPtr = make([]any, len(columns))
		for i := range values {
			valuesPtr[i] = &values[i]
		}

		if err := rows.Scan(valuesPtr...); err != nil {
			return fmt.Errorf("failed to read row: %w", err)
		}

		t.AppendRow(values)
	}

	t.Render()
	return nil
}

func prompt(ctx context.Context, db *sql.DB) error {
	reader := bufio.NewReader(os.Stdin)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		fmt.Print("\n> ")
		text, err := reader.ReadString('\n')
		if errors.Is(err, io.EOF) {
			return nil
		} else if err != nil {
			return fmt.Errorf("failed to read input: %w", err)
		}

		err = executeQuery(db, text)
		if err != nil {
			fmt.Println("ERROR: ", err)
		}
	}
}

func createTables(ctx context.Context, db *sql.DB) error {
	_, err := db.ExecContext(ctx, ddl)
	return err
}

func populateTables(ctx context.Context, db *sql.DB, pkgDir string) error {
	testResults, err := collectTestResults(pkgDir)
	if err != nil {
		return fmt.Errorf("failed to collect test results: %w", err)
	}

	for _, test := range testResults {
		insert := "INSERT INTO all_tests (\"time\", \"action\", package, test, elapsed, \"output\") VALUES (?, ?, ?, ?, ?, ?);"
		_, err = db.ExecContext(ctx, insert, test.Time, test.Action, test.Package, test.Test, test.Elapsed, test.Output)
		if err != nil {
			return fmt.Errorf("failed to insert test results: %w", err)
		}
	}

	coverageResults, err := collectCoverageResults(pkgDir)
	if err != nil {
		return fmt.Errorf("failed to collect coverage results: %w", err)
	}

	for _, result := range coverageResults {
		insert := `INSERT INTO all_coverage (package, file, start_line, start_col, end_line, end_col, stmt_num, count, function_name) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?);`
		_, err := db.ExecContext(ctx, insert, result.Package, result.File, result.StartLine, result.StartColumn, result.EndLine, result.EndColumn, result.StatementNumber, result.Count, result.FunctionName)
		if err != nil {
			return fmt.Errorf("failed to insert coverage results: %w", err)
		}
	}

	testCoverageResults, err := collectTestCoverageResults(pkgDir, testResults)
	if err != nil {
		return fmt.Errorf("failed to collect coverage results by test: %w", err)
	}

	for _, result := range testCoverageResults {
		insertSQL := `INSERT INTO test_coverage (test_name, package, file, start_line, start_col, end_line, end_col, stmt_num, count, function_name) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?);`
		_, err := db.Exec(insertSQL, result.TestName, result.Package, result.File, result.StartLine, result.StartColumn, result.EndLine, result.EndColumn, result.StatementNumber, result.Count, result.FunctionName)
		if err != nil {
			return err
		}
	}

	return nil
}
