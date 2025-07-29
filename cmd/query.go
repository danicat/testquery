package cmd

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	"github.com/danicat/testquery/internal/database"
	"github.com/danicat/testquery/internal/pkgpattern"
	"github.com/danicat/testquery/internal/query"
	_ "github.com/mattn/go-sqlite3"
	"github.com/spf13/cobra"
)

var queryCmd = &cobra.Command{
	Use:   "query [query]",
	Short: "Execute a single query.",
	Long:  `Executes a single SQL query against the test database.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		force, _ := cmd.Flags().GetBool("force")
		pkg, _ := cmd.Flags().GetString("pkg")
		return runQuery(args[0], dbFile, pkg, force)
	},
}

func init() {
	rootCmd.AddCommand(queryCmd)
	queryCmd.Flags().StringVar(&dbFile, "db", "testquery.db", "database file name")
	queryCmd.Flags().Bool("force", false, "force recreation of the database")
	queryCmd.Flags().String("pkg", "./...", "package specifier")
}

func runQuery(q, dbFile, pkg string, force bool) error {
	if force {
		log.Println("Forcing database recreation...")
		if err := os.Remove(dbFile); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("failed to remove existing database: %w", err)
		}
	}

	_, err := os.Stat(dbFile)
	if os.IsNotExist(err) {
		log.Printf("Database %q not found, creating a new one...", dbFile)
		if err := runCollect(dbFile, pkg); err != nil {
			return fmt.Errorf("failed to create database: %w", err)
		}
	} else {
		log.Printf("Using existing database %q", dbFile)
	}

	db, err := sql.Open("sqlite3", dbFile)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer func() {
		if err := db.Close(); err != nil {
			fmt.Printf("failed to close database: %v\n", err)
		}
	}()

	return query.Execute(os.Stdout, db, q)
}

func runCollect(dbFile, pkgSpecifier string) error {
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

	return nil
}
