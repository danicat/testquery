package main

import (
	"fmt"
	"testing"
	"time"
)

// Duplicating mock setup logic here for main_test.go to control collectors during run() call.
// Ideally, this would be shared or refactored if used in many places.

var (
	// Store original functions from the main package
	originalMainCollectTestResultsFunc    func(pkgDir string) ([]TestEvent, error)
	originalMainCollectCoverageResultsFunc func(pkgDir string) ([]CoverageResult, error)
	originalMainCollectCodeLinesFunc      func(pkgDir string) ([]CodeLine, error)
	originalMainCollectTestCoverageResultsFunc func(pkgDir string, testResults []TestEvent) ([]TestCoverageResult, error)
)

func setupMainTestPopulateMocks(t *testing.T) {
	// Save original functions
	originalMainCollectTestResultsFunc = collectTestResults
	originalMainCollectCoverageResultsFunc = collectCoverageResults
	originalMainCollectCodeLinesFunc = collectCodeLines
	originalMainCollectTestCoverageResultsFunc = collectTestCoverageResults

	// Apply mocks
	collectTestResults = func(pkgDir string) ([]TestEvent, error) {
		// Return minimal valid data or error if specific test paths need it
		// For TestRunFunctionality_QueryFlag, we mostly care that it doesn't hang.
		// So, empty data is fine.
		tm, _ := time.Parse(time.RFC3339, "2023-01-01T00:00:00Z")
		el := 0.0
		return []TestEvent{{Time: tm, Action: "pass", Package: "mock", Test: "MockTest", Elapsed: &el}}, nil
	}

	collectCoverageResults = func(pkgDir string) ([]CoverageResult, error) {
		return []CoverageResult{{Package: "mock", File: "mock.go", StartLine: 1, EndLine: 1, Count: 1, FunctionName: "MockFunc"}}, nil
	}

	collectCodeLines = func(pkgDir string) ([]CodeLine, error) {
		return []CodeLine{{Package: "mock", File: "mock.go", LineNumber: 1, Content: "mock line"}}, nil
	}

	collectTestCoverageResults = func(pkgDir string, testResults []TestEvent) ([]TestCoverageResult, error) {
		if len(testResults) > 0 && testResults[0].Test == "ErrorCollectorTest" {
			return nil, fmt.Errorf("mock test coverage collector error")
		}
		return []TestCoverageResult{{TestName: "MockTest", Package: "mock", File: "mock.go", StartLine: 1, EndLine: 1, Count: 1, FunctionName: "MockFunc"}}, nil
	}
}

func restoreMainTestPopulateMocks() {
	if originalMainCollectTestResultsFunc != nil {
		collectTestResults = originalMainCollectTestResultsFunc
	}
	if originalMainCollectCoverageResultsFunc != nil {
		collectCoverageResults = originalMainCollectCoverageResultsFunc
	}
	if originalMainCollectCodeLinesFunc != nil {
		collectCodeLines = originalMainCollectCodeLinesFunc
	}
	if originalMainCollectTestCoverageResultsFunc != nil {
		collectTestCoverageResults = originalMainCollectTestCoverageResultsFunc
	}
}
