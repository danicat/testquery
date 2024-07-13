package main

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

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
func collectTestResults(pkgDir string) ([]TestEvent, error) {
	cmd := exec.Command("go", "test", pkgDir, "-json", "-coverprofile=coverage.out")
	output, _ := cmd.Output()
	tests, err := parseTestOutput(output)
	if err != nil {
		return nil, fmt.Errorf("failed to parse test output: %w", err)
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
	list := "[" + strings.ReplaceAll(string(output[:len(output)-1]), "\n", ",") + "]"
	err := json.Unmarshal([]byte(list), &result)
	if err != nil {
		return nil, err
	}
	return result, nil
}
