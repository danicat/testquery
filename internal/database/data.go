// Copyright 2025 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package database

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/danicat/testquery/internal/collector"
	_ "embed"
)

//go:embed sql/schema.sql
var DDL string

func CreateTables(db *sql.DB) error {
	return CreateTablesFromDDL(db, DDL)
}

func CreateTablesFromDDL(db *sql.DB, ddl string) error {
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	statements := strings.Split(ddl, ";")
	for _, stmt := range statements {
		stmt = strings.TrimSpace(stmt)
		if stmt != "" {
			if _, err := tx.Exec(stmt); err != nil {
				tx.Rollback()
				return fmt.Errorf("failed to execute statement %q: %w", stmt, err)
			}
		}
	}
	return tx.Commit()
}

func PopulateTables(db *sql.DB, pkgDirs []string) error {
	testResults, err := collector.PopulateTestResults(context.Background(), db, pkgDirs)
	if err != nil {
		return fmt.Errorf("failed to populate test results: %w", err)
	}

	if err := collector.PopulateCoverageResults(context.Background(), db, pkgDirs); err != nil {
		return fmt.Errorf("failed to populate coverage results: %w", err)
	}

	if err := collector.PopulateTestCoverageResults(context.Background(), db, pkgDirs, testResults); err != nil {
		return fmt.Errorf("failed to populate test coverage results: %w", err)
	}

	if err := collector.PopulateCode(context.Background(), db, pkgDirs); err != nil {
		return fmt.Errorf("failed to populate code: %w", err)
	}

	return nil
}

func PersistDatabase(db *sql.DB, dbFile string) error {
	_, err := db.Exec("VACUUM INTO ?", dbFile)
	if err != nil {
		return fmt.Errorf("failed to save database file: %w", err)
	}

	return nil
}
