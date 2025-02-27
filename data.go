package main

import (
	"context"
	"database/sql"
	"fmt"

	_ "embed"
)

//go:embed sql/schema.sql
var ddl string

func createTables(ctx context.Context, db *sql.DB) error {
	_, err := db.ExecContext(ctx, ddl)
	return err
}

func populateTables(ctx context.Context, db *sql.DB, pkgDir string) error {
	testResults, err := populateTestResults(ctx, db, pkgDir)
	if err != nil {
		return fmt.Errorf("failed to populate test results: %w", err)
	}

	err = populateCoverageResults(ctx, db, pkgDir)
	if err != nil {
		return fmt.Errorf("failed to populate coverage results: %w", err)
	}

	err = populateTestCoverageResults(ctx, db, pkgDir, testResults)
	if err != nil {
		return fmt.Errorf("failed to populate coverage results: %w", err)
	}

	err = populateCode(ctx, db, pkgDir)
	if err != nil {
		return fmt.Errorf("failed to populate code: %w", err)
	}

	return nil
}

func persistDatabase(db *sql.DB, dbFile string) error {
	_, err := db.Exec("VACUUM INTO ?", dbFile)
	if err != nil {
		return fmt.Errorf("failed to save database file: %w", err)
	}

	return nil
}
