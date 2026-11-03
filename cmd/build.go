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
	"os"

	"github.com/danicat/testquery/internal/database"
	"github.com/danicat/testquery/internal/pkgpattern"
	_ "github.com/mattn/go-sqlite3"
	"github.com/spf13/cobra"
)

var buildCmd = &cobra.Command{
	Use:   "build",
	Short: "Build a persistent test database.",
	Long: `Executes Go tests with coverage and stores all results in a persistent SQLite database.
This database can be queried later using the --db flag in 'query', 'shell', or 'explain' commands.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		pkg, _ := cmd.Flags().GetString("pkg")
		output, _ := cmd.Flags().GetString("output")
		force, _ := cmd.Flags().GetBool("force")
		perTestCov, _ := cmd.Flags().GetBool("per-test-coverage")
		return runBuild(output, pkg, force, perTestCov)
	},
}

func init() {
	rootCmd.AddCommand(buildCmd)
	buildCmd.Flags().String("pkg", "./...", "Go package pattern to test and analyze")
	buildCmd.Flags().String("output", "testquery.db", "Path to the output SQLite database file")
	buildCmd.Flags().BoolP("force", "f", false, "Force overwrite of existing SQLite test database")
	buildCmd.Flags().Bool("per-test-coverage", true, "Collect per-test statement coverage to populate test_coverage")
}

func runBuild(dbFile, pkgSpecifier string, force, perTestCoverage bool) error {
	if _, err := os.Stat(dbFile); err == nil {
		if !force {
			return fmt.Errorf("database file %q already exists (use --force or -f to overwrite)", dbFile)
		}
		for _, path := range []string{dbFile, dbFile + "-wal", dbFile + "-shm"} {
			if rmErr := os.Remove(path); rmErr != nil && !os.IsNotExist(rmErr) {
				return fmt.Errorf("failed to remove existing database file %q: %w", path, rmErr)
			}
		}
	}

	pkgDirs, err := pkgpattern.ListPackages(pkgSpecifier)
	if err != nil {
		return fmt.Errorf("failed to list packages: %w", err)
	}

	db, err := sql.Open("sqlite3", dbFile)
	if err != nil {
		return fmt.Errorf("failed to instantiate sqlite: %w", err)
	}
	defer func() { _ = db.Close() }()

	if err := database.CreateTables(db); err != nil {
		return fmt.Errorf("failed to create tables: %w", err)
	}

	if err := database.PopulateTablesWithOptions(db, pkgDirs, database.PopulateOptions{
		PerTestCoverage: perTestCoverage,
	}); err != nil {
		return fmt.Errorf("failed to populate tables: %w", err)
	}

	_, err = db.Exec("INSERT OR REPLACE INTO metadata (key, value) VALUES (?, ?)", "pkg", pkgSpecifier)
	if err != nil {
		return fmt.Errorf("failed to insert metadata: %w", err)
	}

	return nil
}
