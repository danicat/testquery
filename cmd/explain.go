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

		return RunQueryInMemoryWithOptions(os.Stdout, q, pkg, "table", true)
	},
}

func init() {
	rootCmd.AddCommand(explainCmd)
	explainCmd.Flags().StringVar(&dbFile, "db", "", "database file name")
	explainCmd.Flags().String("pkg", "./...", "package specifier")
}
