package cmd

import (
	"database/sql"
	"fmt"
	"io"
	"os"

	"github.com/danicat/testquery/internal/database"
	"github.com/danicat/testquery/internal/pkgpattern"
	"github.com/danicat/testquery/internal/query"
	_ "github.com/mattn/go-sqlite3"
	"github.com/spf13/cobra"
)

var queryCmd = &cobra.Command{
	Use:   "query [sql]",
	Short: "Execute a single SQL query.",
	Long: `Executes a SQL query against test and coverage data.

You can query an existing database (Offline Mode):
  tq query "SELECT * FROM all_tests" --db testquery.db

Or run tests and query the results immediately (Live Mode):
  tq query "SELECT * FROM all_tests" --pkg ./...`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		dbFile, _ := cmd.Flags().GetString("db")
		pkg, _ := cmd.Flags().GetString("pkg")
		format, _ := cmd.Flags().GetString("format")

		if dbFile != "" && cmd.Flags().Changed("pkg") {
			return fmt.Errorf("cannot use --db (offline mode) and --pkg (live mode) together")
		}

		if dbFile != "" {
			return runQuery(args[0], dbFile, format)
		}

		return RunQueryInMemory(os.Stdout, args[0], pkg, format)
	},
}

func init() {
	rootCmd.AddCommand(queryCmd)
	queryCmd.Flags().StringVar(&dbFile, "db", "", "Path to a persistent SQLite database (Offline Mode)")
	queryCmd.Flags().String("pkg", "./...", "Go package pattern to analyze immediately (Live Mode)")
	queryCmd.Flags().String("format", "table", "Output format: 'table' or 'json'")
}

func runQuery(q, dbFile, format string) error {
	db, err := sql.Open("sqlite3", dbFile)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer func() {
		if err := db.Close(); err != nil {
			fmt.Printf("failed to close database: %v\n", err)
		}
	}()

	return query.Execute(os.Stdout, db, q, format)
}

func RunQueryInMemory(w io.Writer, q, pkg, format string) error {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		return fmt.Errorf("failed to open in-memory database: %w", err)
	}
	defer db.Close()

	if err := RunCollect(db, pkg); err != nil {
		return fmt.Errorf("failed to collect data: %w", err)
	}

	return query.Execute(w, db, q, format)
}

func RunCollect(db *sql.DB, pkgSpecifier string) error {
	pkgDirs, err := pkgpattern.ListPackages(pkgSpecifier)
	if err != nil {
		return fmt.Errorf("failed to list packages: %w", err)
	}

	if err := database.CreateTables(db); err != nil {
		return fmt.Errorf("failed to create tables: %w", err)
	}

	if err := database.PopulateTables(db, pkgDirs); err != nil {
		return fmt.Errorf("failed to populate tables: %w", err)
	}

	return nil
}
