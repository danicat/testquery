package main

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"os/exec"
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

func setupAllTestsDB(t *testing.T) *sql.DB {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open in-memory DB: %v", err)
	}

	// Use the actual schema structure
	_, err = db.Exec(`CREATE TABLE all_tests (
		"time" TIMESTAMP NOT NULL,
		"action" TEXT NOT NULL,
		package TEXT NOT NULL,
        test TEXT NOT NULL,
        elapsed NUMERIC NULL,
        "output" TEXT NULL
	);`)
	if err != nil {
		t.Fatalf("Failed to create schema: %v", err)
	}
	return db
}

func TestParseTestOutput(t *testing.T) {
	// Sample valid JSON output from `go test -json`
	// Note: Each JSON object is on a new line.
	jsonData := `{"Time":"2023-10-27T10:00:00Z","Action":"run","Package":"example.com/mypkg","Test":"TestExample"}
{"Time":"2023-10-27T10:00:01Z","Action":"pass","Package":"example.com/mypkg","Test":"TestExample","Elapsed":0.123}
{"Time":"2023-10-27T10:00:02Z","Action":"run","Package":"example.com/mypkg","Test":"TestAnother"}
{"Time":"2023-10-27T10:00:03Z","Action":"fail","Package":"example.com/mypkg","Test":"TestAnother","Elapsed":0.456,"Output":"--- FAIL: TestAnother (0.46s)\n"}
{"Time":"2023-10-27T10:00:04Z","Action":"skip","Package":"example.com/mypkg","Test":"TestSkipped","Elapsed":0.001}
{"Time":"2023-10-27T10:00:05Z","Action":"output","Package":"example.com/mypkg","Output":"coverage: 75.0% of statements\n"}
{"Time":"2023-10-27T10:00:06Z","Action":"summary","Package":"example.com/mypkg","Test":"TestNotARealTestName"}
`
	// Malformed JSON
	malformedJsonData := `{"Time":"2023-10-27T10:00:00Z","Action":"run","Package":"example.com/mypkg","Test":"TestExample"}
{"Time":"2023-10-27T10:00:01Z",Action:"pass","Package":"example.com/mypkg","Test":"TestExample","Elapsed":0.123}
` // Missing quotes around Action key in second line

	passTime, _ := time.Parse(time.RFC3339Nano, "2023-10-27T10:00:01Z")
	failTime, _ := time.Parse(time.RFC3339Nano, "2023-10-27T10:00:03Z")
	skipTime, _ := time.Parse(time.RFC3339Nano, "2023-10-27T10:00:04Z")
	passElapsed := 0.123
	failElapsed := 0.456
	skipElapsed := 0.001
	failOutput := "--- FAIL: TestAnother (0.46s)\n"


	expected := []TestEvent{
		{Time: passTime, Action: "pass", Package: "example.com/mypkg", Test: "TestExample", Elapsed: &passElapsed},
		{Time: failTime, Action: "fail", Package: "example.com/mypkg", Test: "TestAnother", Elapsed: &failElapsed, Output: &failOutput},
		{Time: skipTime, Action: "skip", Package: "example.com/mypkg", Test: "TestSkipped", Elapsed: &skipElapsed},
	}

	// Sort expected for consistent comparison if order isn't guaranteed (though it should be here)
	sort.Slice(expected, func(i, j int) bool { return expected[i].Time.Before(expected[j].Time) })


	t.Run("valid json", func(t *testing.T) {
		actual, err := parseTestOutput([]byte(jsonData))
		if err != nil {
			t.Fatalf("parseTestOutput() with valid JSON error = %v", err)
		}
		sort.Slice(actual, func(i, j int) bool { return actual[i].Time.Before(actual[j].Time) })

		if !reflect.DeepEqual(actual, expected) {
			t.Errorf("parseTestOutput() got = %+v, want %+v", actual, expected)
			// For more detailed diff:
			for i := 0; i < len(actual) || i < len(expected); i++ {
				if i < len(actual) && i < len(expected) {
					if !reflect.DeepEqual(actual[i], expected[i]) {
						t.Errorf("Mismatch at index %d: got %+v, want %+v", i, actual[i], expected[i])
						if actual[i].Elapsed != nil && expected[i].Elapsed != nil && *actual[i].Elapsed != *expected[i].Elapsed {
							t.Errorf("  Elapsed diff: got %f, want %f", *actual[i].Elapsed, *expected[i].Elapsed)
						}
						if actual[i].Output != nil && expected[i].Output != nil && *actual[i].Output != *expected[i].Output {
							t.Errorf("  Output diff: got %s, want %s", *actual[i].Output, *expected[i].Output)
						}
					}
				} else if i < len(actual) {
					t.Errorf("Extra actual element at index %d: %+v", i, actual[i])
				} else {
					t.Errorf("Missing actual element, expected at index %d: %+v", i, expected[i])
				}
			}
		}
	})

	t.Run("malformed json", func(t *testing.T) {
		_, err := parseTestOutput([]byte(malformedJsonData))
		if err == nil {
			t.Errorf("parseTestOutput() with malformed JSON expected error, got nil")
		}
	})

	t.Run("empty input", func(t *testing.T) {
		actual, err := parseTestOutput([]byte(""))
		if err != nil {
			t.Errorf("parseTestOutput() with empty input error = %v", err)
		}
		if len(actual) != 0 {
			t.Errorf("parseTestOutput() with empty input expected empty slice, got %v", actual)
		}
	})
}

// Mock exec.Command for collectTestResults
// This is a common pattern for testing functions that call external commands.
var mockExecCommand func(command string, args ...string) *exec.Cmd
var mockExitError = false // Simulate exit error from `go test` (e.g. on test failure)

func TestMain(m *testing.M) {
	// Save original execCommand
	origExecCommand := execCommand
	execCommand = func(command string, args ...string) *exec.Cmd {
		if mockExecCommand != nil {
			return mockExecCommand(command, args...)
		}
		// If no mock is set, or if the mock wants to call the original:
		return origExecCommand(command, args...)
	}

	retCode := m.Run()

	// Restore original execCommand
	execCommand = origExecCommand
	os.Exit(retCode)
}

func TestCollectTestResults(t *testing.T) {
	// Setup mock for exec.Command
	// The actual main.execCommand is manipulated in TestMain
	mockExecCommand = func(command string, args ...string) *exec.Cmd {
		// Check if the command and args are what we expect
		// e.g., command == "go" and args contains "test", "-json"
		cs := []string{"-c", ""}
		if command == "go" && len(args) > 1 && args[0] == "test" && args[2] == "-json" {
			// Create a script that outputs our predefined JSON
			jsonData := `{"Time":"2023-10-27T10:00:01Z","Action":"pass","Package":"example.com/mypkg","Test":"TestExample","Elapsed":0.123}
{"Time":"2023-10-27T10:00:03Z","Action":"fail","Package":"example.com/mypkg","Test":"TestAnother","Elapsed":0.456,"Output":"failure"}`

			var script string
			if mockExitError {
				// Simulate `go test` exiting with non-zero status on failure
				script = fmt.Sprintf("echo '%s'; exit 1", jsonData)
			} else {
				script = fmt.Sprintf("echo '%s'; exit 0", jsonData)
			}
			cs = append(cs, script)

		} else {
			// Fallback for unexpected commands: try to run it, or fail test
			t.Errorf("Unexpected command: %s %v", command, args)
			cs = append(cs, "exit 1") // cause an error
		}
		cmd := exec.Command("sh", cs...) // Use "sh -c" to echo the data
		return cmd
	}
	defer func() { mockExecCommand = nil; mockExitError = false }() // Cleanup

	passTime, _ := time.Parse(time.RFC3339Nano, "2023-10-27T10:00:01Z")
	failTime, _ := time.Parse(time.RFC3339Nano, "2023-10-27T10:00:03Z")
	passElapsed := 0.123
	failElapsed := 0.456
	failOutput := "failure"

	expected := []TestEvent{
		{Time: passTime, Action: "pass", Package: "example.com/mypkg", Test: "TestExample", Elapsed: &passElapsed},
		{Time: failTime, Action: "fail", Package: "example.com/mypkg", Test: "TestAnother", Elapsed: &failElapsed, Output: &failOutput},
	}
	sort.Slice(expected, func(i, j int) bool { return expected[i].Time.Before(expected[j].Time) })

	t.Run("no exit error", func(t *testing.T) {
		mockExitError = false
		actual, err := collectTestResults("./fakepkg")
		if err != nil {
			t.Fatalf("collectTestResults() error = %v", err)
		}
		sort.Slice(actual, func(i, j int) bool { return actual[i].Time.Before(actual[j].Time) })
		if !reflect.DeepEqual(actual, expected) {
			t.Errorf("collectTestResults() got = %+v, want %+v", actual, expected)
		}
	})

	t.Run("with exit error", func(t *testing.T) {
		mockExitError = true
		actual, err := collectTestResults("./fakepkg_with_failure") // Different pkgDir to ensure mock runs
		// We expect no error from collectTestResults itself, as exit status 1 from `go test` is normal on failure.
		if err != nil && !strings.Contains(err.Error(), "failed to execute go test command") {
			// The refactored code in all_tests.go now returns an error if cmd.Output() returns an error
			// that is *not* an exit status error. So, if `go test` itself fails to run (e.g. bad command),
			// it will error. If `go test` runs but tests fail (exit 1), it should *not* error here.
			// The current mock always makes cmd.Output() succeed or fail based on `exit 0` or `exit 1`
			// from the `sh -c` script.
			// The check `!strings.Contains(err.Error(), "exit status")` in `all_tests.go` handles this.
			// So, an error here means the mock command failed in an unexpected way, or the logic is wrong.
			// Let's adjust the expectation: `collectTestResults` should parse output even if `go test` exits with 1.
			// The current `cmd.Output()` in `collectTestResults` will return an `*exec.ExitError` if the script exits 1.
			// This error IS caught by the `if err != nil && !strings.Contains(err.Error(), "exit status")` check.
			// So, `collectTestResults` should NOT return an error here.
			t.Fatalf("collectTestResults() with exit error, unexpected error from function: %v", err)
		}
		if err == nil { // If it didn't error (which is expected now)
			sort.Slice(actual, func(i, j int) bool { return actual[i].Time.Before(actual[j].Time) })
			if !reflect.DeepEqual(actual, expected) {
				t.Errorf("collectTestResults() with exit error, got = %+v, want %+v", actual, expected)
			}
		} else { // If it did error
			// This case should not be hit if the logic in all_tests.go is correct for handling ExitError
			t.Errorf("collectTestResults() returned an unexpected error when go test fails: %v", err)
		}
	})


	// Test case where the command itself fails (not just test failures)
	mockExecCommand = func(command string, args ...string) *exec.Cmd {
		return exec.Command("commandthatdoesnotexist")
	}
	_, err := collectTestResults("./anotherfakepkg")
	if err == nil {
		t.Errorf("collectTestResults() expected error for command failure, got nil")
	} else if !strings.Contains(err.Error(), "failed to execute go test command") {
		t.Errorf("collectTestResults() expected specific error for command failure, got: %v", err)
	}
	mockExecCommand = nil // reset
}


func TestPopulateTestResults(t *testing.T) {
	db := setupAllTestsDB(t)
	defer db.Close()

	// Mock collectTestResults to avoid running actual commands
	originalCollectTestResults := collectTestResults
	defer func() { collectTestResults = originalCollectTestResults }()

	mockTime1, _ := time.Parse(time.RFC3339Nano, "2023-01-01T12:00:00Z")
	mockTime2, _ := time.Parse(time.RFC3339Nano, "2023-01-01T12:01:00Z")
	mockElapsed1 := 0.5
	mockOutput2 := "FAIL: TestTwo"

	mockData := []TestEvent{
		{Time: mockTime1, Action: "pass", Package: "pkg1", Test: "TestOne", Elapsed: &mockElapsed1},
		{Time: mockTime2, Action: "fail", Package: "pkg1", Test: "TestTwo", Output: &mockOutput2},
	}

	collectTestResults = func(pkgDir string) ([]TestEvent, error) {
		if pkgDir == "errorpkg" {
			return nil, fmt.Errorf("mock collect error")
		}
		return mockData, nil
	}

	_, err := populateTestResults(context.Background(), db, "fakepkg")
	if err != nil {
		t.Fatalf("populateTestResults() error = %v", err)
	}

	rows, err := db.Query("SELECT time, action, package, test, elapsed, output FROM all_tests ORDER BY time")
	if err != nil {
		t.Fatalf("Failed to query all_tests: %v", err)
	}
	defer rows.Close()

	var actual []TestEvent
	for rows.Next() {
		var te TestEvent
		var sqlTime string // Read time as string then parse, SQLite time handling can be tricky
		var elapsed sql.NullFloat64
		var output sql.NullString

		if err := rows.Scan(&sqlTime, &te.Action, &te.Package, &te.Test, &elapsed, &output); err != nil {
			t.Fatalf("Failed to scan row: %v", err)
		}
		// Parse time from string. Assuming UTC for simplicity, as stored by go test json output.
		// The format from SQLite might vary, so robust parsing or matching the stored format is needed.
		// For this test, we'll assume RFC3339 or similar that time.Parse can handle.
		// The time from go test json is RFC3339Nano. SQLite stores it as TEXT in "YYYY-MM-DD HH:MM:SS.SSS" format by default.
		// Let's re-query and format the expected time to match typical DB storage if not RFC3339.
		// Or, ensure TestEvent.Time is stored and retrieved in a consistent format.
		// For now, assume direct parse works if DB stores it close to RFC3339.
		// This will likely fail if DB format is different.
		parsedTime, err := time.Parse("2006-01-02 15:04:05.999999999-07:00", sqlTime) // Common SQLite format if includes TZ
		if err != nil {
			parsedTime, err = time.Parse("2006-01-02 15:04:05", strings.Split(sqlTime, ".")[0]) // Try without subseconds if parse fails
			if err != nil {
				t.Logf("Attempting to parse time string from DB: %s", sqlTime)
				// Fallback to parsing assuming UTC and possibly different precision
				// This part is tricky because SQLite date strings might not perfectly match Go's time.Time string output
				// or RFC3339Nano.
				// The `go-sqlite3` driver handles time.Time correctly, usually storing as RFC3339 format.
				parsedTime, err = time.Parse(time.RFC3339Nano, sqlTime)
				if err != nil {
					t.Fatalf("Failed to parse time '%s' from DB: %v", sqlTime, err)
				}
			}
		}
		te.Time = parsedTime.UTC() // Ensure UTC for comparison

		if elapsed.Valid {
			te.Elapsed = &elapsed.Float64
		}
		if output.Valid {
			te.Output = &output.String
		}
		actual = append(actual, te)
	}

	// Ensure mockData times are UTC for comparison
	for i := range mockData {
		mockData[i].Time = mockData[i].Time.UTC()
	}

	// Sort for comparison
	sort.Slice(actual, func(i, j int) bool { return actual[i].Time.Before(actual[j].Time) })
	sort.Slice(mockData, func(i, j int) bool { return mockData[i].Time.Before(mockData[j].Time) })


	if !reflect.DeepEqual(actual, mockData) {
		t.Errorf("populateTestResults() data in DB = \n%+v, want \n%+v", actual, mockData)
		for i := 0; i < len(actual) || i < len(mockData); i++ {
			if i < len(actual) && i < len(mockData) {
				if !reflect.DeepEqual(actual[i], mockData[i]) {
					t.Logf("Diff at index %d:", i)
					t.Logf("  Actual: Time=%v, Action=%s, Pkg=%s, Test=%s, Elapsed=%v, Output=%v", actual[i].Time, actual[i].Action, actual[i].Package, actual[i].Test, actual[i].Elapsed, actual[i].Output)
					t.Logf("  Expected: Time=%v, Action=%s, Pkg=%s, Test=%s, Elapsed=%v, Output=%v", mockData[i].Time, mockData[i].Action, mockData[i].Package, mockData[i].Test, mockData[i].Elapsed, mockData[i].Output)
					if !actual[i].Time.Equal(mockData[i].Time) {
						t.Logf("    Time diff: Actual.Time.Location: %s, Expected.Time.Location: %s", actual[i].Time.Location(), mockData[i].Time.Location())
					}
				}
			}
		}
	}

	// Test error case from collectTestResults
	_, err = populateTestResults(context.Background(), db, "errorpkg")
	if err == nil {
		t.Errorf("populateTestResults() with collect error expected error, got nil")
	} else if !strings.Contains(err.Error(), "mock collect error") {
		t.Errorf("populateTestResults() with collect error, wrong error message: got %v", err)
	}
}


// Helper to convert TestEvent to a comparable string for easier diffing in logs
func testEventToString(te TestEvent) string {
	var el, op string
	if te.Elapsed != nil {
		el = fmt.Sprintf("%.3f", *te.Elapsed)
	} else {
		el = "<nil>"
	}
	if te.Output != nil {
		op = *te.Output
	} else {
		op = "<nil>"
	}
	return fmt.Sprintf("Time:%s Action:%s Pkg:%s Test:%s Elapsed:%s Output:'%s'",
		te.Time.Format(time.RFC3339Nano), te.Action, te.Package, te.Test, el, op)
}

func printEventSlice(events []TestEvent) string {
	var sb strings.Builder
	sb.WriteString("[\n")
	for _, e := range events {
		sb.WriteString("  " + testEventToString(e) + "\n")
	}
	sb.WriteString("]")
	return sb.String()
}
