package main

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

func setupDataTestDB(t *testing.T) *sql.DB {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open in-memory DB: %v", err)
	}
	return db
}

func TestCreateTables(t *testing.T) {
	db := setupDataTestDB(t)
	defer db.Close()

	err := createTables(context.Background(), db)
	if err != nil {
		t.Fatalf("createTables() error = %v", err)
	}

	expectedTables := []string{"all_tests", "all_coverage", "test_coverage", "all_code"}
	expectedViews := []string{"failed_tests", "passed_tests", "missing_coverage", "code_coverage"}

	rows, err := db.Query("SELECT name, type FROM sqlite_master WHERE type='table' OR type='view' ORDER BY name")
	if err != nil {
		t.Fatalf("Failed to query sqlite_master: %v", err)
	}
	defer rows.Close()

	var foundTables []string
	var foundViews []string
	for rows.Next() {
		var name, typeName string
		if err := rows.Scan(&name, &typeName); err != nil {
			t.Fatalf("Failed to scan row: %v", err)
		}
		// SQLite may prefix temp tables/views, but these are not temp.
		// Filter out sqlite internal tables.
		if strings.HasPrefix(name, "sqlite_") {
			continue
		}
		if typeName == "table" {
			foundTables = append(foundTables, name)
		} else if typeName == "view" {
			foundViews = append(foundViews, name)
		}
	}

	sort.Strings(foundTables)
	sort.Strings(foundViews)
	sort.Strings(expectedTables)
	sort.Strings(expectedViews)

	if !reflect.DeepEqual(foundTables, expectedTables) {
		t.Errorf("createTables() tables got = %v, want %v", foundTables, expectedTables)
	}
	if !reflect.DeepEqual(foundViews, expectedViews) {
		t.Errorf("createTables() views got = %v, want %v", foundViews, expectedViews)
	}
}


func TestPersistDatabase(t *testing.T) {
	db := setupDataTestDB(t) // Use an in-memory DB first
	defer db.Close()

	// Add some data to make the DB non-empty
	_, err := db.Exec("CREATE TABLE test_persist (id INT); INSERT INTO test_persist (id) VALUES (1);")
	if err != nil {
		t.Fatalf("Failed to create test table for persist: %v", err)
	}

	tmpDir, err := os.MkdirTemp("", "testpersist")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	dbFilePath := filepath.Join(tmpDir, "persisted.db")

	err = persistDatabase(db, dbFilePath)
	if err != nil {
		t.Fatalf("persistDatabase() error = %v", err)
	}

	if _, errStat := os.Stat(dbFilePath); os.IsNotExist(errStat) {
		t.Errorf("persistDatabase() did not create database file '%s'", dbFilePath)
	}

	// Try to open the persisted database and check content
	persistedDB, err := sql.Open("sqlite3", dbFilePath)
	if err != nil {
		t.Fatalf("Failed to open persisted DB file: %v", err)
	}
	defer persistedDB.Close()

	var count int
	err = persistedDB.QueryRow("SELECT COUNT(*) FROM test_persist WHERE id = 1").Scan(&count)
	if err != nil {
		t.Fatalf("Failed to query persisted DB: %v", err)
	}
	if count != 1 {
		t.Errorf("Data not correctly persisted. Expected count 1, got %d", count)
	}
}


// Mocking setup for populateTables tests
var (
	mockCollectTestResultsData    []TestEvent
	mockCollectTestResultsError   error
	mockCollectCoverageData       []CoverageResult
	mockCollectCoverageError      error
	mockCollectCodeLinesData      []CodeLine
	mockCollectCodeLinesError     error
	mockCollectTestCoverageData   []TestCoverageResult
	mockCollectTestCoverageError  error

	originalCollectTestResults    func(pkgDir string) ([]TestEvent, error)
	originalCollectCoverageResults func(pkgDir string) ([]CoverageResult, error)
	originalCollectCodeLines      func(pkgDir string) ([]CodeLine, error)
	originalCollectTestCoverageResults func(pkgDir string, testResults []TestEvent) ([]TestCoverageResult, error)
)

func setupPopulateMocks() {
	originalCollectTestResults = collectTestResults
	collectTestResults = func(pkgDir string) ([]TestEvent, error) {
		return mockCollectTestResultsData, mockCollectTestResultsError
	}

	originalCollectCoverageResults = collectCoverageResults
	collectCoverageResults = func(pkgDir string) ([]CoverageResult, error) {
		return mockCollectCoverageData, mockCollectCoverageError
	}

	originalCollectCodeLines = collectCodeLines
	collectCodeLines = func(pkgDir string) ([]CodeLine, error) {
		return mockCollectCodeLinesData, mockCollectCodeLinesError
	}

	originalCollectTestCoverageResults = collectTestCoverageResults
	collectTestCoverageResults = func(pkgDir string, tr []TestEvent) ([]TestCoverageResult, error) {
		return mockCollectTestCoverageData, mockCollectTestCoverageError
	}
}

func restorePopulateMocks() {
	collectTestResults = originalCollectTestResults
	collectCoverageResults = originalCollectCoverageResults
	collectCodeLines = originalCollectCodeLines
	collectTestCoverageResults = originalCollectTestCoverageResults

	mockCollectTestResultsData = nil
	mockCollectTestResultsError = nil
	mockCollectCoverageData = nil
	mockCollectCoverageError = nil
	mockCollectCodeLinesData = nil
	mockCollectCodeLinesError = nil
	mockCollectTestCoverageData = nil
	mockCollectTestCoverageError = nil
}


func TestPopulateTables(t *testing.T) {
	db := setupDataTestDB(t) // In-memory DB
	defer db.Close()

	// Need to create tables first
	if err := createTables(context.Background(), db); err != nil {
		t.Fatalf("Failed to create tables for TestPopulateTables: %v", err)
	}

	setupPopulateMocks()
	defer restorePopulateMocks()

	// Prepare mock data
	tm, _ := time.Parse(time.RFC3339, "2023-01-01T12:00:00Z")
	elapsed := 0.1
	mockCollectTestResultsData = []TestEvent{
		{Time: tm, Action: "pass", Package: "mypkg", Test: "TestOne", Elapsed: &elapsed},
	}
	mockCollectCoverageData = []CoverageResult{
		{Package: "mypkg", File: "file.go", StartLine: 1, EndLine: 2, Count: 1, FunctionName: "FuncA"},
	}
	mockCollectCodeLinesData = []CodeLine{
		{Package: "mypkg", File: "file.go", LineNumber: 1, Content: "line1"},
	}
	mockCollectTestCoverageData = []TestCoverageResult{
		{TestName: "TestOne", Package: "mypkg", File: "file.go", StartLine: 1, EndLine: 2, Count: 1, FunctionName: "FuncA"},
	}


	err := populateTables(context.Background(), db, "./fakepkgdir")
	if err != nil {
		t.Fatalf("populateTables() error = %v", err)
	}

	// Verify data in all_tests
	var testCount int
	err = db.QueryRow("SELECT COUNT(*) FROM all_tests WHERE package='mypkg' AND test='TestOne'").Scan(&testCount)
	if err != nil {
		t.Fatalf("Query all_tests failed: %v", err)
	}
	if testCount != 1 {
		t.Errorf("Expected 1 test result for TestOne, got %d", testCount)
	}

	// Verify data in all_coverage
	var coverageCount int
	err = db.QueryRow("SELECT COUNT(*) FROM all_coverage WHERE package='mypkg' AND file='file.go' AND function_name='FuncA'").Scan(&coverageCount)
	if err != nil {
		t.Fatalf("Query all_coverage failed: %v", err)
	}
	if coverageCount != 1 {
		t.Errorf("Expected 1 coverage result for FuncA, got %d", coverageCount)
	}

	// Verify data in all_code
	var codeCount int
	err = db.QueryRow("SELECT COUNT(*) FROM all_code WHERE package='mypkg' AND file='file.go' AND line_number=1").Scan(&codeCount)
	if err != nil {
		t.Fatalf("Query all_code failed: %v", err)
	}
	if codeCount != 1 {
		t.Errorf("Expected 1 code line, got %d", codeCount)
	}

	// Verify data in test_coverage
	var testCoverageCount int
	err = db.QueryRow("SELECT COUNT(*) FROM test_coverage WHERE test_name='TestOne' AND package='mypkg' AND file='file.go' AND function_name='FuncA'").Scan(&testCoverageCount)
	if err != nil {
		t.Fatalf("Query test_coverage failed: %v", err)
	}
	if testCoverageCount != 1 {
		t.Errorf("Expected 1 test_coverage result for TestOne/FuncA, got %d", testCoverageCount)
	}


	// Test error propagation from collectors
	restorePopulateMocks() // Clear previous mocks
	setupPopulateMocks()   // Re-init for error test

	mockCollectTestResultsError = fmt.Errorf("test results collection error")
	err = populateTables(context.Background(), db, "./fakepkgdir")
	if err == nil || !strings.Contains(err.Error(), "test results collection error") {
		t.Errorf("populateTables() did not propagate error from collectTestResults. Got err: %v", err)
	}
	mockCollectTestResultsError = nil // Reset for next error test

	mockCollectCoverageError = fmt.Errorf("coverage collection error")
	err = populateTables(context.Background(), db, "./fakepkgdir")
	if err == nil || !strings.Contains(err.Error(), "coverage collection error") {
		t.Errorf("populateTables() did not propagate error from collectCoverageResults. Got err: %v", err)
	}
	mockCollectCoverageError = nil

	mockCollectCodeLinesError = fmt.Errorf("code lines collection error")
	err = populateTables(context.Background(), db, "./fakepkgdir")
	if err == nil || !strings.Contains(err.Error(), "code lines collection error") {
		t.Errorf("populateTables() did not propagate error from collectCodeLines. Got err: %v", err)
	}
	mockCollectCodeLinesError = nil

	mockCollectTestCoverageError = fmt.Errorf("test coverage collection error")
	err = populateTables(context.Background(), db, "./fakepkgdir") // Will use mockCollectTestResultsData (empty if not set)
	if err == nil || !strings.Contains(err.Error(), "test coverage collection error") {
		t.Errorf("populateTables() did not propagate error from collectTestCoverageResults. Got err: %v", err)
	}
	mockCollectTestCoverageError = nil

	restorePopulateMocks()
}
