package cmd

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"

	"github.com/danicat/testquery/internal/shell"
	_ "github.com/mattn/go-sqlite3"
	"github.com/spf13/cobra"
)

var shellCmd = &cobra.Command{
	Use:   "shell",
	Short: "Start an interactive SQL shell.",
	Long:  `Starts an interactive SQL shell to query the test database.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		force, _ := cmd.Flags().GetBool("force")
		pkg, _ := cmd.Flags().GetString("pkg")
		return runShell(dbFile, pkg, force)
	},
}

func init() {
	rootCmd.AddCommand(shellCmd)
	shellCmd.Flags().StringVar(&dbFile, "db", "testquery.db", "database file name")
	shellCmd.Flags().Bool("force", false, "force recreation of the database")
	shellCmd.Flags().String("pkg", "./...", "package specifier")
}

func runShell(dbFile, pkg string, force bool) error {
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

	return shell.Prompt(context.Background(), db)
}
