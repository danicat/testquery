package cmd

import (
	"github.com/spf13/cobra"
)

var (
	dbFile string
)

var rootCmd = &cobra.Command{
	Use:   "tq",
	Short: "TestQuery (tq) is a tool for querying Go test and coverage data using SQL.",
	Long: `TestQuery (tq) allows you to analyze Go test results and coverage data using a relational database.

You can use it in two modes:
1. Live Mode: Run tests and query the results immediately in-memory (using the --pkg flag).
2. Offline Mode: Build a persistent SQLite database (using the 'build' command) and query it later (using the --db flag).

It is designed to help developers identify testing gaps, understand code execution paths, and analyze test performance in large codebases.`,
}

func Execute() error {
	return rootCmd.Execute()
}
