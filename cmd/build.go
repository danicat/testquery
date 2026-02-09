package cmd

import (
	"database/sql"
	"fmt"

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
		return runBuild(output, pkg)
	},
}

func init() {
	rootCmd.AddCommand(buildCmd)
	buildCmd.Flags().String("pkg", "./...", "Go package pattern to test and analyze")
	buildCmd.Flags().String("output", "testquery.db", "Path to the output SQLite database file")
}

func runBuild(dbFile, pkgSpecifier string) error {
	pkgDirs, err := pkgpattern.ListPackages(pkgSpecifier)
	if err != nil {
		return fmt.Errorf("failed to list packages: %w", err)
	}

	db, err := sql.Open("sqlite3", dbFile)
	if err != nil {
		return fmt.Errorf("failed to instantiate sqlite: %w", err)
	}
	defer db.Close()

	if err := database.CreateTables(db); err != nil {
		return fmt.Errorf("failed to create tables: %w", err)
	}

	if err := database.PopulateTables(db, pkgDirs); err != nil {
		return fmt.Errorf("failed to populate tables: %w", err)
	}

	_, err = db.Exec("INSERT INTO metadata (key, value) VALUES (?, ?)", "pkg", pkgSpecifier)
	if err != nil {
		return fmt.Errorf("failed to insert metadata: %w", err)
	}

	return nil
}
