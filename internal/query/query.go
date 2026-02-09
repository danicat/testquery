package query

import (
	"database/sql"
	"encoding/json"
	"io"

	"github.com/jedib0t/go-pretty/v6/table"
)

func Execute(w io.Writer, db *sql.DB, query string, format string) error {
	rows, err := db.Query(query)
	if err != nil {
		return err
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return err
	}

	if format == "json" {
		return executeJSON(w, rows, columns)
	}

	t := table.NewWriter()

	t.SetOutputMirror(w)

	var header = make(table.Row, len(columns))
	for i := range columns {
		header[i] = columns[i]
	}
	t.AppendHeader(header)

	for rows.Next() {
		var values = make(table.Row, len(columns))
		var valuesPtr = make([]any, len(columns))
		for i := range values {
			valuesPtr[i] = &values[i]
		}

		if err := rows.Scan(valuesPtr...); err != nil {
			return err
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

func executeJSON(w io.Writer, rows *sql.Rows, columns []string) error {
	var results []map[string]any

	for rows.Next() {
		var values = make([]any, len(columns))
		var valuesPtr = make([]any, len(columns))
		for i := range values {
			valuesPtr[i] = &values[i]
		}

		if err := rows.Scan(valuesPtr...); err != nil {
			return err
		}

		rowMap := make(map[string]any)
		for i, col := range columns {
			// Handle []byte specifically for SQLite results if necessary,
			// though typical scan into 'any' works.
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

	// If no results, output empty array
	if results == nil {
		results = []map[string]any{}
	}

	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(results)
}
