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
