package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"time"
)

// TestResult represents the structure of a test result
type TestEvent struct {
	Time         time.Time `json:"time"`
	Action       string    `json:"action"`
	Package      string    `json:"package"`
	Test         string    `json:"test"`
	Elapsed      *float64  `json:"elapsed,omitempty"`
	Output       *string   `json:"output,omitempty"`
	FailedBuild  *string   `json:"FailedBuild,omitempty"`
}

// collectTestResults runs `go test -json` and parses the output
func collectTestResults(pkgDir string) ([]TestEvent, error) {
	cmd := exec.Command("go", "test", pkgDir, "-json", "-coverprofile=coverage.out")
	output, err := cmd.CombinedOutput()
	if err != nil {
		if _, ok := err.(*exec.ExitError); !ok {
			// This is an error running the command, not a test failure.
			return nil, fmt.Errorf("failed to run go test: %w: %s", err, string(output))
		}
	}

	tests, err := parseTestOutput(output)
	if err != nil {
		return nil, fmt.Errorf("failed to parse test output: %w. Output: %s", err, string(output))
	}

	// Check for a build failure event, which indicates the package could not be tested.
	for _, event := range tests {
		if event.Action == "fail" && event.FailedBuild != nil {
			return nil, fmt.Errorf("build failed for package %s", *event.FailedBuild)
		}
	}

	var results []TestEvent
	for _, test := range tests {
		if test.Test == "" || (test.Action != "pass" && test.Action != "fail") {
			continue
		}
		results = append(results, test)
	}
	return results, nil
}

func parseTestOutput(output []byte) ([]TestEvent, error) {
	var result []TestEvent
	decoder := json.NewDecoder(strings.NewReader(string(output)))
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

func populateTestResults(ctx context.Context, db *sql.DB, pkgDir string) ([]TestEvent, error) {
	testResults, err := collectTestResults(pkgDir)
	if err != nil {
		return nil, fmt.Errorf("failed to collect test results: %w", err)
	}

	for _, test := range testResults {
		insertSQL := "INSERT INTO all_tests (\"time\", \"action\", package, test, elapsed, \"output\") VALUES (?, ?, ?, ?, ?, ?);"
		_, err = db.ExecContext(ctx, insertSQL, test.Time, test.Action, test.Package, test.Test, test.Elapsed, test.Output)
		if err != nil {
			return nil, fmt.Errorf("failed to insert test results: %w", err)
		}
	}

	return testResults, nil
}
