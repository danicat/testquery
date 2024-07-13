package main

import (
	"bufio"
	"context"
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"

	_ "embed"

	_ "github.com/mattn/go-sqlite3"
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

	// Create a slice of interfaces to represent each column and a slice of pointers to each item in the interface slice
	values := make([]interface{}, len(columns))
	valuePtrs := make([]interface{}, len(columns))
	for i := range values {
		valuePtrs[i] = &values[i]
	}

	// Iterate through the result set
	var results []map[string]interface{}
	for rows.Next() {
		// Scan the result into the value pointers
		if err := rows.Scan(valuePtrs...); err != nil {
			log.Fatal(err)
		}

		// Create a map to store the row data
		rowMap := make(map[string]interface{})
		for i, col := range columns {
			var v interface{}
			val := values[i]
			b, ok := val.([]byte)
			if ok {
				v = string(b)
			} else {
				v = val
			}
			rowMap[col] = v
		}
		results = append(results, rowMap)
	}

	// Print the results as JSON
	jsonResults, err := json.MarshalIndent(results, "", "  ")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(string(jsonResults))

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
		if err != nil {
			return fmt.Errorf("failed to read input: %w", err)
		}

		err = executeQuery(db, text)
		if err != nil {
			log.Println(err)
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
		insert := `INSERT INTO all_coverage (package, file, from_line, from_col, to_line, to_col, stmt_num, count, function_name) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?);`
		_, err := db.ExecContext(ctx, insert, result.Package, result.File, result.FromLine, result.FromColumn, result.ToLine, result.ToColumn, result.StatementNumber, result.Count, result.FunctionName)
		if err != nil {
			return fmt.Errorf("failed to insert coverage results: %w", err)
		}
	}

	testCoverageResults, err := collectTestCoverageResults(pkgDir, testResults)
	if err != nil {
		return fmt.Errorf("failed to collect coverage results by test: %w", err)
	}

	for _, result := range testCoverageResults {
		insertSQL := `INSERT INTO test_coverage (test_name, package, file, from_line, from_col, to_line, to_col, stmt_num, count, function_name) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?);`
		_, err := db.Exec(insertSQL, result.TestName, result.Package, result.File, result.FromLine, result.FromColumn, result.ToLine, result.ToColumn, result.StatementNumber, result.Count, result.FunctionName)
		if err != nil {
			return err
		}
	}

	return nil
}
