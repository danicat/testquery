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

var Version = "dev"

func main() {
	pkgDir := flag.String("pkg", ".", "directory of the package to test")
	persist := flag.Bool("persist", false, "persist database between runs")
	dbFile := flag.String("dbfile", "testquery.db", "database file name for use with --persist and --open")
	openDB := flag.Bool("open", false, "open a database from a previous run")
	query := flag.String("query", "", "runs a single query and returns the result")
	version := flag.Bool("version", false, "shows version information")
	flag.Parse()

	if *version {
		fmt.Println("tq", Version)
		return
	}

	ctx := context.Background()

	rl, err := readline.NewEx(&readline.Config{
		Prompt:                 "> ",
		HistoryFile:            "/tmp/testquery-history",
		DisableAutoSaveHistory: true,
	})
	if err != nil {
		log.Fatalln(err)
	}
	defer rl.Close()

	err = run(ctx, *pkgDir, rl, *persist, *openDB, *dbFile, *query)
	if err != nil && !errors.Is(err, io.EOF) && !errors.Is(err, context.Canceled) {
		log.Fatalln(err)
	}
}

func run(ctx context.Context, pkgDir string, rl *readline.Instance, persist, open bool, dbFile string, query string) error {
	var db *sql.DB
	var err error

	if open {
		db, err = sql.Open("sqlite3", dbFile)
		if err != nil {
			return fmt.Errorf("failed to open database: %w", err)
		}
		defer db.Close()
	} else {
		db, err = sql.Open("sqlite3", ":memory:")
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
	}

	if persist {
		defer persistDatabase(db, dbFile)
	}

	if query != "" {
		return executeQuery(db, query)
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
