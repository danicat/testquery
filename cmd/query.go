// Copyright (c) 2026 Google LLC
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

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
		noTruncate, _ := cmd.Flags().GetBool("no-truncate")
		full, _ := cmd.Flags().GetBool("full")
		perTestCov, _ := cmd.Flags().GetBool("per-test-coverage")

		if dbFile != "" && cmd.Flags().Changed("pkg") {
			return fmt.Errorf("cannot use --db (offline mode) and --pkg (live mode) together")
		}

		var opts []query.Option
		if noTruncate || full {
			opts = append(opts, query.WithNoTruncate(true))
		}

		if dbFile != "" {
			return runQuery(args[0], dbFile, format, opts...)
		}

		return RunQueryInMemoryWithOptions(os.Stdout, args[0], pkg, format, perTestCov, opts...)
	},
}

func init() {
	rootCmd.AddCommand(queryCmd)
	queryCmd.Flags().StringVar(&dbFile, "db", "", "Path to a persistent SQLite database (Offline Mode)")
	queryCmd.Flags().String("pkg", "./...", "Go package pattern to analyze immediately (Live Mode)")
	queryCmd.Flags().String("format", "table", "Output format: 'table' or 'json'")
	queryCmd.Flags().Bool("no-truncate", false, "Do not truncate cells longer than 80 characters in table output")
	queryCmd.Flags().Bool("full", false, "Display full cell values without truncation (alias for --no-truncate)")
	queryCmd.Flags().Bool("per-test-coverage", true, "Collect per-test statement coverage in live mode")
}

func runQuery(q, dbFile, format string, opts ...query.Option) error {
	db, err := sql.Open("sqlite3", dbFile)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer func() {
		if err := db.Close(); err != nil {
			fmt.Printf("failed to close database: %v\n", err)
		}
	}()

	return query.Execute(os.Stdout, db, q, format, opts...)
}

func RunQueryInMemory(w io.Writer, q, pkg, format string, opts ...query.Option) error {
	return RunQueryInMemoryWithOptions(w, q, pkg, format, true, opts...)
}

func RunQueryInMemoryWithOptions(w io.Writer, q, pkg, format string, perTestCoverage bool, opts ...query.Option) error {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		return fmt.Errorf("failed to open in-memory database: %w", err)
	}
	defer func() { _ = db.Close() }()

	if err := RunCollectWithOptions(db, pkg, perTestCoverage); err != nil {
		return fmt.Errorf("failed to collect data: %w", err)
	}

	return query.Execute(w, db, q, format, opts...)
}

func RunCollect(db *sql.DB, pkgSpecifier string) error {
	return RunCollectWithOptions(db, pkgSpecifier, true)
}

func RunCollectWithOptions(db *sql.DB, pkgSpecifier string, perTestCoverage bool) error {
	pkgDirs, err := pkgpattern.ListPackages(pkgSpecifier)
	if err != nil {
		return fmt.Errorf("failed to list packages: %w", err)
	}

	if err := database.CreateTables(db); err != nil {
		return fmt.Errorf("failed to create tables: %w", err)
	}

	if err := database.PopulateTablesWithOptions(db, pkgDirs, database.PopulateOptions{
		PerTestCoverage: perTestCoverage,
	}); err != nil {
		return fmt.Errorf("failed to populate tables: %w", err)
	}

	return nil
}
