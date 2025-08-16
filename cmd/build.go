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

package cmd

import (
	"database/sql"
	"fmt"

	"github.com/danicat/testquery/internal/database"
	"github.com/danicat/testquery/internal/pkgpattern"
	_ "github.com/mattn/go-sqlite3"
	"github.com/spf13/cobra"
)

var buildCmd = &cobra.Command{
	Use:   "build",
	Short: "Build the test database from a package.",
	Long:  `Builds the test database by collecting test and coverage data from a specified Go package.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		pkg, _ := cmd.Flags().GetString("pkg")
		output, _ := cmd.Flags().GetString("output")
		return runBuild(output, pkg)
	},
}

func init() {
	rootCmd.AddCommand(buildCmd)
	buildCmd.Flags().String("pkg", "./...", "Go package specifier")
	buildCmd.Flags().String("output", "testquery.db", "Output database file")
}

func runBuild(dbFile, pkgSpecifier string) error {
	pkgDirs, err := pkgpattern.ListPackages(pkgSpecifier)
	if err != nil {
		return fmt.Errorf("failed to list packages: %w", err)
	}

	db, err := sql.Open("sqlite3", dbFile)
	if err != nil {
		return fmt.Errorf("failed to instantiate sqlite: %w", err)
	}
	defer db.Close()

	if err := database.CreateTables(db); err != nil {
		return fmt.Errorf("failed to create tables: %w", err)
	}

	if err := database.PopulateTables(db, pkgDirs); err != nil {
		return fmt.Errorf("failed to populate tables: %w", err)
	}

	_, err = db.Exec("INSERT INTO metadata (key, value) VALUES (?, ?)", "pkg", pkgSpecifier)
	if err != nil {
		return fmt.Errorf("failed to insert metadata: %w", err)
	}

	return nil
}
