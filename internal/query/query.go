// Copyright (c) 2026 Google LLC
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

package query

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"regexp"
	"strings"

	"github.com/jedib0t/go-pretty/v6/table"
)

const DefaultMaxCellWidth = 80

type config struct {
	noTruncate   bool
	maxCellWidth int
}

type Option func(*config)

func WithNoTruncate(noTruncate bool) Option {
	return func(c *config) {
		c.noTruncate = noTruncate
	}
}

func Execute(w io.Writer, db *sql.DB, query string, format string, opts ...Option) error {
	cfg := config{
		maxCellWidth: DefaultMaxCellWidth,
	}
	for _, opt := range opts {
		opt(&cfg)
	}

	rows, err := db.Query(query)
	if err != nil {
		return formatQueryError(db, query, err)
	}
	defer func() {
		_ = rows.Close()
	}()

	columns, err := rows.Columns()
	if err != nil {
		return err
	}

	if format == "json" {
		return executeJSON(w, rows, columns)
	}

	t := table.NewWriter()
	t.SetOutputMirror(w)

	header := make(table.Row, len(columns))
	for i := range columns {
		header[i] = columns[i]
	}
	t.AppendHeader(header)

	for rows.Next() {
		values := make(table.Row, len(columns))
		valuesPtr := make([]any, len(columns))
		for i := range values {
			valuesPtr[i] = &values[i]
		}

		if err := rows.Scan(valuesPtr...); err != nil {
			return err
		}

		for i, val := range values {
			if b, ok := val.([]byte); ok {
				val = string(b)
				values[i] = val
			}
			if !cfg.noTruncate && cfg.maxCellWidth > 0 {
				if s, ok := val.(string); ok {
					values[i] = truncateCell(s, cfg.maxCellWidth)
				}
			}
		}

		t.AppendRow(values)
	}

	if err = rows.Err(); err != nil {
		return err
	}

	if err = rows.Close(); err != nil {
		return err
	}

	t.Render()
	return nil
}

func truncateCell(s string, maxWidth int) string {
	runes := []rune(s)
	if len(runes) <= maxWidth {
		return s
	}
	remaining := len(runes) - maxWidth
	return fmt.Sprintf("%s... (+%d chars)", string(runes[:maxWidth]), remaining)
}

func formatQueryError(db *sql.DB, failedQuery string, queryErr error) error {
	metaRows, err := db.Query(`SELECT name, type FROM sqlite_master WHERE type IN ('table', 'view') AND name NOT LIKE 'sqlite_%' ORDER BY name ASC`)
	if err != nil {
		return queryErr
	}
	defer func() {
		_ = metaRows.Close()
	}()

	var tables []string
	var views []string
	var allObjects []string

	for metaRows.Next() {
		var name, objType string
		if scanErr := metaRows.Scan(&name, &objType); scanErr != nil {
			continue
		}
		allObjects = append(allObjects, name)
		if objType == "view" {
			views = append(views, name)
		} else {
			tables = append(tables, name)
		}
	}
	_ = metaRows.Close()

	var b strings.Builder
	b.WriteString(queryErr.Error())

	for _, name := range allObjects {
		pattern := `(?i)\b` + regexp.QuoteMeta(name) + `\b`
		if matched, _ := regexp.MatchString(pattern, failedQuery); matched {
			cols := getTableColumns(db, name)
			if len(cols) > 0 {
				_, _ = fmt.Fprintf(&b, "\nColumns in %s: %s", name, strings.Join(cols, ", "))
			}
		}
	}

	if len(tables) > 0 {
		_, _ = fmt.Fprintf(&b, "\nAvailable tables: %s", strings.Join(tables, ", "))
	}
	if len(views) > 0 {
		_, _ = fmt.Fprintf(&b, "\nAvailable views: %s", strings.Join(views, ", "))
	}

	return fmt.Errorf("%s", b.String())
}

func getTableColumns(db *sql.DB, tableName string) []string {
	escaped := strings.ReplaceAll(tableName, `"`, `""`)
	rows, err := db.Query(fmt.Sprintf(`PRAGMA table_info("%s")`, escaped))
	if err != nil {
		return nil
	}
	defer func() {
		_ = rows.Close()
	}()

	var cols []string
	for rows.Next() {
		var cid int
		var name, colType string
		var notNull int
		var dfltValue any
		var pk int
		if err := rows.Scan(&cid, &name, &colType, &notNull, &dfltValue, &pk); err != nil {
			continue
		}
		cols = append(cols, name)
	}
	return cols
}

func executeJSON(w io.Writer, rows *sql.Rows, columns []string) error {
	var results []map[string]any

	for rows.Next() {
		values := make([]any, len(columns))
		valuesPtr := make([]any, len(columns))
		for i := range values {
			valuesPtr[i] = &values[i]
		}

		if err := rows.Scan(valuesPtr...); err != nil {
			return err
		}

		rowMap := make(map[string]any)
		for i, col := range columns {
			val := values[i]
			if b, ok := val.([]byte); ok {
				rowMap[col] = string(b)
			} else {
				rowMap[col] = val
			}
		}
		results = append(results, rowMap)
	}

	if err := rows.Err(); err != nil {
		return err
	}

	if results == nil {
		results = []map[string]any{}
	}

	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(results)
}
