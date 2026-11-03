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

package collector

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"time"
)

// TestEvent represents the structure of a test result event from `go test -json`.
type TestEvent struct {
	Time       time.Time `json:"time"`
	Action     string    `json:"action"`
	Package    string    `json:"package"`
	Test       string    `json:"test"`
	ParentTest *string   `json:"parent_test,omitempty"`
	IsSubtest  int       `json:"is_subtest"`
	IsParent   int       `json:"is_parent"`
	Elapsed    *float64  `json:"elapsed,omitempty"`
	Output     *string   `json:"output,omitempty"`
}

// collectTestResults runs `go test -json` and parses the output.
func collectTestResults(pkgDirs []string) ([]TestEvent, error) {
	mc, err := resolveModuleContext(pkgDirs)
	if err != nil {
		return nil, err
	}

	args := []string{"test"}
	args = append(args, pkgDirs...)
	args = append(args, "-json", "-coverprofile=coverage.out")

	cmd := exec.Command("go", args...)
	cmd.Dir = "." // Run from project root
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	_ = cmd.Run()

	tests, err := parseTestOutput(stdout.Bytes())
	if err != nil {
		return nil, fmt.Errorf("failed to parse test output: %w. Output: %s", err, stderr.String())
	}

	return filterAndEnrichTestEvents(tests, mc), nil
}

func filterAndEnrichTestEvents(tests []TestEvent, mc *moduleContext) []TestEvent {
	outputByTest := make(map[string]strings.Builder)
	var results []TestEvent

	for _, test := range tests {
		if test.Test == "" {
			continue
		}
		if mc != nil && test.Package != "" {
			test.Package = mc.normalizePackage(test.Package)
		}
		key := test.Package + "\x00" + test.Test
		if test.Action == "output" && test.Output != nil {
			b := outputByTest[key]
			b.WriteString(*test.Output)
			outputByTest[key] = b
			continue
		}
		if test.Action != "pass" && test.Action != "fail" {
			continue
		}
		if test.Output == nil {
			if b, ok := outputByTest[key]; ok && b.Len() > 0 {
				out := b.String()
				test.Output = &out
			}
		}
		if idx := strings.LastIndex(test.Test, "/"); idx != -1 {
			parent := test.Test[:idx]
			test.ParentTest = &parent
			test.IsSubtest = 1
		} else {
			test.ParentTest = nil
			test.IsSubtest = 0
		}
		results = append(results, test)
	}

	for i := range results {
		prefix := results[i].Test + "/"
		for j := range results {
			if i != j && results[i].Package == results[j].Package && strings.HasPrefix(results[j].Test, prefix) {
				results[i].IsParent = 1
				break
			}
		}
	}

	return results
}

func parseTestOutput(output []byte) ([]TestEvent, error) {
	var result []TestEvent
	decoder := json.NewDecoder(bytes.NewReader(output))
	for {
		var event TestEvent
		if err := decoder.Decode(&event); err == io.EOF {
			break
		} else if err != nil {
			return nil, err
		}
		result = append(result, event)
	}
	return result, nil
}

func PopulateTestResults(ctx context.Context, db *sql.DB, pkgDirs []string) ([]TestEvent, error) {
	testResults, err := collectTestResults(pkgDirs)
	if err != nil {
		return nil, fmt.Errorf("failed to collect test results: %w", err)
	}

	const insertSQL = `INSERT OR REPLACE INTO all_tests ("time", "action", status, package, test, parent_test, is_subtest, is_parent, elapsed, "output") VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?);`
	for _, test := range testResults {
		_, err = db.ExecContext(ctx, insertSQL, test.Time, test.Action, test.Action, test.Package, test.Test, test.ParentTest, test.IsSubtest, test.IsParent, test.Elapsed, test.Output)
		if err != nil {
			return nil, fmt.Errorf("failed to insert test results: %w", err)
		}
	}

	return testResults, nil
}
