package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// execCommand is a variable to enable mocking of exec.Command for tests
var execCommand = exec.Command // Uses the imported "os/exec" package

// TestResult represents the structure of a test result
type TestEvent struct {
	Time    time.Time `json:"time"`
	Action  string    `json:"action"`
	Package string    `json:"package"`
	Test    string    `json:"test"`
	Elapsed *float64  `json:"elapsed,omitempty"`
	Output  *string   `json:"output,omitempty"`
}

// collectTestResults runs `go test -json` and parses the output
var collectTestResults = func(pkgDir string) ([]TestEvent, error) {
	cmd := execCommand("go", "test", pkgDir, "-json", "-coverprofile=coverage.out")
	output, err := cmd.Output()
	// Ignore exit errors, as go test returns non-zero on test failures
	if err != nil && !strings.Contains(err.Error(), "exit status") {
		return nil, fmt.Errorf("failed to execute go test command: %w", err)
	}
	return parseTestOutput(output)
}

func parseTestOutput(output []byte) ([]TestEvent, error) {
	var events []TestEvent
	decoder := json.NewDecoder(strings.NewReader(string(output)))
	for {
		var event TestEvent
		if err := decoder.Decode(&event); err != nil {
			if err.Error() == "EOF" { // Correctly check for EOF
				break
			}
			return nil, fmt.Errorf("failed to decode test event: %w", err)
		}
		// Filter for relevant events (actual test results)
		if event.Test != "" && (event.Action == "pass" || event.Action == "fail" || event.Action == "skip") {
			events = append(events, event)
		}
	}
	return events, nil
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
