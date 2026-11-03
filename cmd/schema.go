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

	"github.com/spf13/cobra"
)

var schemaCmd = &cobra.Command{
	Use:   "schema",
	Short: "Describe the test database schema and common queries.",
	Long:  `Provides a detailed overview of available tables, views, columns, and usage examples for querying the test database.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Print(`
# TestQuery (tq) Database Schema

The database consists of 5 core tables and 10 built-in views that correlate tests with the code they execute.
All tables normalize "package" to the full Go module import path and "file" to the workspace-relative file path.

## Tables

### 1. all_tests
Records the results of every test and subtest executed during the build process.
- time: Timestamp of the test event.
- action / status: Result ('pass' or 'fail').
- package: Full Go import path of the package containing the test.
- test: Name of the test or subtest (e.g., 'TestFoo' or 'TestFoo/sub').
- parent_test: Parent test name for subtests (NULL for top-level tests).
- is_subtest: 1 if the test is a t.Run subtest (contains '/'), 0 otherwise.
- is_parent: 1 if the test is a parent container with child subtests, 0 for executable leaf tests.
- elapsed: Time taken to run the test in seconds.
- output: Console output captured for the test.

### 2. all_coverage
Aggregated statement coverage blocks across the entire project.
- package: Full Go import path of the covered file's package.
- file: Workspace-relative path to the source file.
- function_name: Name of the enclosing function.
- start_line, start_col, end_line, end_col: Source range of the basic block.
- stmt_num: Number of statements in the block.
- count: Execution count across all tests.

### 3. test_coverage
Mapping of individual leaf tests to the statement blocks they cover (populated when --per-test-coverage is enabled).
- test_package: Full Go import path of the test package.
- test_name: Name of the leaf test or subtest.
- package: Full Go import path of the covered file's package.
- file: Workspace-relative path to the source file.
- function_name: Name of the enclosing function.
- start_line, start_col, end_line, end_col: Source range of the basic block.
- stmt_num: Number of statements in the block.
- count: Execution count for this specific test.

### 4. all_code
Searchable lines of your project's Go source files.
- package: Full Go import path of the package.
- file: Workspace-relative path to the source file.
- line_number: 1-based line number.
- content: Raw source line text.

### 5. metadata
Key-value build metadata (including 'pkg' and 'per_test_coverage').

## Built-in Views

- leaf_tests: Executable leaf tests only (WHERE is_parent = 0), preventing double-counting of parent containers.
- passed_tests: Passing leaf tests (package, test, elapsed).
- failed_tests: Failing leaf tests (package, test, elapsed, output).
- slow_tests: Leaf tests ordered by elapsed time descending.
- test_summary: Per-package counts of leaf_tests, subtests, total_events, passed, failed, and total_elapsed.
- missing_coverage: Uncovered statement blocks (count = 0).
- code_coverage: Line-by-line source code joined with coverage counts (covered > 0 when covered).
- file_coverage: Per-file covered statements, total statements, and coverage_pct.
- package_coverage: Per-package covered statements, total statements, and coverage_pct.
- uncovered_lines: Source lines belonging to uncovered statement blocks.

## Common Queries

- Find tests covering a specific line:
  SELECT DISTINCT test_name FROM test_coverage WHERE file LIKE '%main.go' AND start_line <= 10 AND end_line >= 10 AND count > 0;

- Find all uncovered functions:
  SELECT DISTINCT file, function_name FROM all_coverage WHERE count = 0;

- Check line-level code coverage:
  SELECT file, line_number, content, covered FROM code_coverage WHERE covered > 0;

- Find the slowest leaf tests:
  SELECT package, test, elapsed FROM slow_tests LIMIT 10;
`)
	},
}

func init() {
	rootCmd.AddCommand(schemaCmd)
}
