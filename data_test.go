package main

import (
	"context"
	"database/sql"
	"errors"
	"testing"
)

// --- Mocks from all_code_test.go (consider refactoring to a common test helper if used more) ---
type MockDB struct {
	ExecContextFunc func(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
	ExecFunc        func(query string, args ...interface{}) (sql.Result, error) // For persistDatabase
	CloseFunc       func() error
	QueryFunc       func(query string, args ...interface{}) (*sql.Rows, error)
}

func (m *MockDB) ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	if m.ExecContextFunc != nil {
		return m.ExecContextFunc(ctx, query, args...)
	}
	return &MockSQLResult{}, nil
}

func (m *MockDB) Exec(query string, args ...interface{}) (sql.Result, error) {
	if m.ExecFunc != nil {
		return m.ExecFunc(query, args...)
	}
	return &MockSQLResult{}, nil
}

func (m *MockDB) Close() error {
	if m.CloseFunc != nil {
		return m.CloseFunc()
	}
	return nil
}

func (m *MockDB) Query(query string, args ...interface{}) (*sql.Rows, error) {
	if m.QueryFunc != nil {
		return m.QueryFunc(query, args...)
	}
	// Needs a mock sql.Rows, returning empty for now or error if not set
	return nil, errors.New("QueryFunc not implemented in mock")
}


type MockSQLResult struct {
	LastInsertIdFunc func() (int64, error)
	RowsAffectedFunc func() (int64, error)
}

func (m *MockSQLResult) LastInsertId() (int64, error) {
	if m.LastInsertIdFunc != nil {
		return m.LastInsertIdFunc()
	}
	return 0, nil
}

func (m *MockSQLResult) RowsAffected() (int64, error) {
	if m.RowsAffectedFunc != nil {
		return m.RowsAffectedFunc()
	}
	return 1, nil // Default to 1 row affected for successful exec
}
// --- End Mocks ---

// Mock implementations for populate functions (to be overridden in tests)
var (
	mockPopulateTestResultsFunc       func(ctx context.Context, db *sql.DB, pkgDir string) ([]TestEvent, error)
	mockPopulateCoverageResultsFunc   func(ctx context.Context, db *sql.DB, pkgDir string) error
	mockPopulateTestCoverageResultsFunc func(ctx context.Context, db *sql.DB, pkgDir string, testResults []TestEvent) error
	mockPopulateCodeFunc              func(ctx context.Context, db *sql.DB, pkgDir string) error
)

// Override original functions with mocks for testing populateTables
func setupPopulateMocks() {
	origPopulateTestResults := populateTestResults
	origPopulateCoverageResults := populateCoverageResults
	origPopulateTestCoverageResults := populateTestCoverageResults
	origPopulateCode := populateCode

	populateTestResults = func(ctx context.Context, db *sql.DB, pkgDir string) ([]TestEvent, error) {
		if mockPopulateTestResultsFunc != nil {
			return mockPopulateTestResultsFunc(ctx, db, pkgDir)
		}
		return []TestEvent{}, nil // Default mock behavior
	}
	populateCoverageResults = func(ctx context.Context, db *sql.DB, pkgDir string) error {
		if mockPopulateCoverageResultsFunc != nil {
			return mockPopulateCoverageResultsFunc(ctx, db, pkgDir)
		}
		return nil // Default mock behavior
	}
	populateTestCoverageResults = func(ctx context.Context, db *sql.DB, pkgDir string, testResults []TestEvent) error {
		if mockPopulateTestCoverageResultsFunc != nil {
			return mockPopulateTestCoverageResultsFunc(ctx, db, pkgDir, testResults)
		}
		return nil // Default mock behavior
	}
	populateCode = func(ctx context.Context, db *sql.DB, pkgDir string) error {
		if mockPopulateCodeFunc != nil {
			return mockPopulateCodeFunc(ctx, db, pkgDir)
		}
		return nil // Default mock behavior
	}

	// Teardown function to restore original implementations
	teardownPopulateMocks = func() {
		populateTestResults = origPopulateTestResults
		populateCoverageResults = origPopulateCoverageResults
		populateTestCoverageResults = origPopulateTestCoverageResults
		populateCode = origPopulateCode
		resetPopulateMockFuncs()
	}
}

var teardownPopulateMocks func()

func resetPopulateMockFuncs() {
	mockPopulateTestResultsFunc = nil
	mockPopulateCoverageResultsFunc = nil
	mockPopulateTestCoverageResultsFunc = nil
	mockPopulateCodeFunc = nil
}


func TestCreateTables(t *testing.T) {
	ctx := context.Background()
	mockDB := &MockDB{}
	var execContextCalled bool
	var receivedQuery string

	mockDB.ExecContextFunc = func(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
		execContextCalled = true
		receivedQuery = query
		return &MockSQLResult{}, nil
	}

	err := createTables(ctx, mockDB)
	if err != nil {
		t.Fatalf("createTables failed: %v", err)
	}

	if !execContextCalled {
		t.Errorf("Expected db.ExecContext to be called, but it wasn't")
	}
	if receivedQuery != ddl {
		t.Errorf("Expected DDL query %q, got %q", ddl, receivedQuery)
	}
}

func TestPopulateTables(t *testing.T) {
	setupPopulateMocks()
	defer teardownPopulateMocks()

	ctx := context.Background()
	mockDB := &MockDB{} // Not strictly needed if sub-functions are fully mocked
	pkgDir := "./testpkg"

	var populateTestResultsCalled, populateCoverageResultsCalled, populateTestCoverageResultsCalled, populateCodeCalled bool

	mockPopulateTestResultsFunc = func(ctx context.Context, db *sql.DB, dir string) ([]TestEvent, error) {
		populateTestResultsCalled = true
		if dir != pkgDir {
			t.Errorf("Expected pkgDir %s for populateTestResults, got %s", pkgDir, dir)
		}
		return []TestEvent{{Test: "TestSample"}}, nil // Return some data for TestCoverage
	}
	mockPopulateCoverageResultsFunc = func(ctx context.Context, db *sql.DB, dir string) error {
		populateCoverageResultsCalled = true
		if dir != pkgDir {
			t.Errorf("Expected pkgDir %s for populateCoverageResults, got %s", pkgDir, dir)
		}
		return nil
	}
	mockPopulateTestCoverageResultsFunc = func(ctx context.Context, db *sql.DB, dir string, testResults []TestEvent) error {
		populateTestCoverageResultsCalled = true
		if dir != pkgDir {
			t.Errorf("Expected pkgDir %s for populateTestCoverageResults, got %s", pkgDir, dir)
		}
		if len(testResults) == 0 || testResults[0].Test != "TestSample" {
			t.Errorf("Expected testResults to be passed to populateTestCoverageResults, got %v", testResults)
		}
		return nil
	}
	mockPopulateCodeFunc = func(ctx context.Context, db *sql.DB, dir string) error {
		populateCodeCalled = true
		if dir != pkgDir {
			t.Errorf("Expected pkgDir %s for populateCode, got %s", pkgDir, dir)
		}
		return nil
	}

	err := populateTables(ctx, mockDB, pkgDir)
	if err != nil {
		t.Fatalf("populateTables failed: %v", err)
	}

	if !populateTestResultsCalled {
		t.Errorf("Expected populateTestResults to be called")
	}
	if !populateCoverageResultsCalled {
		t.Errorf("Expected populateCoverageResults to be called")
	}
	if !populateTestCoverageResultsCalled {
		t.Errorf("Expected populateTestCoverageResults to be called")
	}
	if !populateCodeCalled {
		t.Errorf("Expected populateCode to be called")
	}

	// Test error propagation from populateTestResults
	expectedErr := errors.New("populateTestResults error")
	mockPopulateTestResultsFunc = func(ctx context.Context, db *sql.DB, dir string) ([]TestEvent, error) {
		return nil, expectedErr
	}
	err = populateTables(ctx, mockDB, pkgDir)
	if !errors.Is(err, expectedErr) {
		t.Errorf("Expected error from populateTestResults to propagate, got %v", err)
	}
	resetPopulateMockFuncs() // Reset for next error check

	// Test error propagation from populateCoverageResults
	expectedErr = errors.New("populateCoverageResults error")
	mockPopulateCoverageResultsFunc = func(ctx context.Context, db *sql.DB, dir string) error { return expectedErr }
	err = populateTables(ctx, mockDB, pkgDir)
	if !errors.Is(err, expectedErr) {
		t.Errorf("Expected error from populateCoverageResults to propagate, got %v", err)
	}
	resetPopulateMockFuncs()

	// ... similar error propagation tests for populateTestCoverageResults and populateCode ...
	expectedErr = errors.New("populateTestCoverageResults error")
	mockPopulateTestCoverageResultsFunc = func(ctx context.Context, db *sql.DB, dir string, testResults []TestEvent) error { return expectedErr }
	err = populateTables(ctx, mockDB, pkgDir)
	if !errors.Is(err, expectedErr) {
		t.Errorf("Expected error from populateTestCoverageResults to propagate, got %v", err)
	}
	resetPopulateMockFuncs()

	expectedErr = errors.New("populateCode error")
	mockPopulateCodeFunc = func(ctx context.Context, db *sql.DB, dir string) error { return expectedErr }
	err = populateTables(ctx, mockDB, pkgDir)
	if !errors.Is(err, expectedErr) {
		t.Errorf("Expected error from populateCode to propagate, got %v", err)
	}
	resetPopulateMockFuncs()
}

func TestPersistDatabase(t *testing.T) {
	mockDB := &MockDB{}
	var execCalled bool
	var receivedQuery string
	var receivedArg string
	dbFile := "test.db"

	mockDB.ExecFunc = func(query string, args ...interface{}) (sql.Result, error) {
		execCalled = true
		receivedQuery = query
		if len(args) > 0 {
			receivedArg = args[0].(string)
		}
		return &MockSQLResult{}, nil
	}

	err := persistDatabase(mockDB, dbFile)
	if err != nil {
		t.Fatalf("persistDatabase failed: %v", err)
	}

	if !execCalled {
		t.Errorf("Expected db.Exec to be called, but it wasn't")
	}
	expectedQuery := "VACUUM INTO ?"
	if receivedQuery != expectedQuery {
		t.Errorf("Expected query %q, got %q", expectedQuery, receivedQuery)
	}
	if receivedArg != dbFile {
		t.Errorf("Expected dbFile argument %q, got %q", dbFile, receivedArg)
	}

	// Test error propagation
	expectedErr := errors.New("db exec error")
	mockDB.ExecFunc = func(query string, args ...interface{}) (sql.Result, error) {
		return nil, expectedErr
	}
	err = persistDatabase(mockDB, dbFile)
	if !errors.Is(err, expectedErr) {
		t.Errorf("Expected error from db.Exec to propagate, got %v", err)
	}
}

// This setup function needs to be called by tests that modify global variables (like our function overrides)
// to ensure that tests can run in parallel and don't interfere with each other if t.Parallel() is used.
// However, for simplicity in this example, we are not using t.Parallel() and rely on sequential execution.
// A more robust solution might involve using interfaces for dependencies and injecting mocks,
// rather than overriding package-level functions.
func TestMain(m *testing.M) {
	// setupPopulateMocks() // Setup once if not tearing down per test
	// code := m.Run()
	// teardownPopulateMocks() // Teardown once
	// os.Exit(code)
	// For now, let each test manage its own setup/teardown of mocks as they are simple.
	m.Run()
}
