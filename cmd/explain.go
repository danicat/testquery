package cmd

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
)

var explainCmd = &cobra.Command{
	Use:   "explain [file:line]",
	Short: "Find tests covering a specific line of code.",
	Long: `Identifies all tests that execute the specified file and line.

Example:
  tq explain main.go:50 --db testquery.db`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		dbFile, _ := cmd.Flags().GetString("db")
		pkg, _ := cmd.Flags().GetString("pkg")

		if dbFile != "" && cmd.Flags().Changed("pkg") {
			return fmt.Errorf("cannot use --db (offline mode) and --pkg (live mode) together")
		}

		target := args[0]

		parts := strings.Split(target, ":")
		if len(parts) != 2 {
			return fmt.Errorf("invalid format, expected file:line")
		}

		filename := parts[0]
		line, err := strconv.Atoi(parts[1])
		if err != nil {
			return fmt.Errorf("invalid line number: %v", err)
		}

		// Construct query to find covering tests
		q := fmt.Sprintf(`
			SELECT DISTINCT test_name 
			FROM test_coverage 
			WHERE (file = '%s' OR file LIKE '%%/%s')
			AND start_line <= %d 
			AND end_line >= %d 
			AND count > 0
			ORDER BY test_name ASC`,
			filename, filename, line, line)

		if dbFile != "" {
			return runQuery(q, dbFile, "table")
		}

		return RunQueryInMemory(os.Stdout, q, pkg, "table")
	},
}

func init() {
	rootCmd.AddCommand(explainCmd)
	explainCmd.Flags().StringVar(&dbFile, "db", "", "database file name")
	explainCmd.Flags().String("pkg", "./...", "package specifier")
}
