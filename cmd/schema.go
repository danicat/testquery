package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var schemaCmd = &cobra.Command{
	Use:   "schema",
	Short: "Describe the test database schema and common queries.",
	Long:  `Provides a detailed overview of available tables, columns, and usage examples for querying the test database.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Print(`
# TestQuery (tq) Database Schema

The database consists of 4 main tables that allow you to correlate tests with the code they execute.

## Tables

### 1. all_tests
Records the results of every test executed during the build process.
- package: The Go package containing the test.
- test: The name of the test function.
- action: Result (run, pass, fail, skip).
- elapsed: Time taken to run the test.
- output: Raw console output from the test.

### 2. all_coverage
Aggregated coverage data for the entire project.
- file: Source file path.
- function_name: Name of the covered function.
- start_line, end_line: Range of the covered block.
- count: Number of times this block was executed across ALL tests.

### 3. test_coverage
Mapping of specific tests to the code blocks they cover.
- test_name: Name of the test.
- file: Source file being covered.
- start_line, end_line: The code block range.
- count: Number of times this specific test hit this block.

### 4. all_code
A searchable copy of your project's source code.
- file: Source file path.
- line_number: The line index.
- content: The raw code on that line.

## Common Queries

- Find tests covering a specific line:
  SELECT DISTINCT test_name FROM test_coverage WHERE file LIKE '%main.go' AND start_line <= 10 AND end_line >= 10 AND count > 0;

- Find all uncovered functions:
  SELECT DISTINCT file, function_name FROM all_coverage WHERE count = 0;

- Search source code for a string:
  SELECT file, line_number, content FROM all_code WHERE content LIKE '%TODO%';

- Find the slowest tests:
  SELECT test, elapsed FROM all_tests WHERE action = 'pass' ORDER BY elapsed DESC LIMIT 10;
`)
	},
}

func init() {
	rootCmd.AddCommand(schemaCmd)
}
