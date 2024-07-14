package main

import (
	"context"
	"database/sql"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"strings"

	_ "github.com/mattn/go-sqlite3"

	"github.com/chzyer/readline"
	"github.com/jedib0t/go-pretty/v6/table"
)

func main() {
	pkgDir := flag.String("pkg", ".", "directory of the package to test")
	flag.Parse()

	ctx := context.Background()

	rl, err := readline.NewEx(&readline.Config{
		Prompt:                 "> ",
		HistoryFile:            "/tmp/readline-multiline",
		DisableAutoSaveHistory: true,
	})
	if err != nil {
		log.Fatalln(err)
	}
	defer rl.Close()

	err = run(ctx, *pkgDir, rl)
	if err != nil && !errors.Is(err, io.EOF) && !errors.Is(err, context.Canceled) {
		log.Fatalln(err)
	}
}

func run(ctx context.Context, pkgDir string, rl *readline.Instance) error {
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

	return prompt(ctx, db, rl)
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

func prompt(ctx context.Context, db *sql.DB, rl *readline.Instance) error {
	var cmds []string
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		line, err := rl.Readline()
		if err != nil {
			return fmt.Errorf("failed to read line: %w", err)
		}

		line = strings.TrimSpace(line)
		if len(line) == 0 {
			continue
		}

		cmds = append(cmds, line)
		if !strings.HasSuffix(line, ";") {
			rl.SetPrompt(">>> ")
			continue
		}

		cmd := strings.Join(cmds, " ")
		cmds = cmds[:0]
		rl.SetPrompt("> ")
		rl.SaveHistory(cmd)

		err = executeQuery(db, cmd)
		if err != nil {
			fmt.Println("ERROR: ", err)
		}
	}
}
