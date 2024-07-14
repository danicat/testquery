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

	_ "github.com/mattn/go-sqlite3"

	"github.com/jedib0t/go-pretty/v6/table"
)

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
		return fmt.Errorf("failed to retrieve column names: %w", err)
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
			fmt.Println()
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
