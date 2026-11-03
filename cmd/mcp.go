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

package cmd

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/danicat/testquery/internal/database"
	_ "github.com/mattn/go-sqlite3"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/spf13/cobra"
)

var mcpCmd = &cobra.Command{
	Use:   "mcp",
	Short: "Start an MCP server.",
	Long:  `Exposes TestQuery (tq) as a Model Context Protocol (MCP) server, allowing AI agents to query your test and coverage data.`,
	RunE: func(cmd *cobra.Command, args []string) error {

		s := mcp.NewServer(&mcp.Implementation{}, nil)
		mcp.AddTool(s, &mcp.Tool{
			Name:        "query",
			Description: "Query the test database",
		}, func(ctx context.Context, ss *mcp.ServerSession, ctp *mcp.CallToolParamsFor[map[string]any]) (*mcp.CallToolResultFor[any], error) {
			q, ok := ctp.Arguments["query"].(string)
			if !ok {
				return &mcp.CallToolResultFor[any]{
					Content: []mcp.Content{
						&mcp.TextContent{
							Text: "Invalid query argument",
						},
					},
					IsError: true,
				}, nil
			}

			var b bytes.Buffer
			// Default to table format for now to maintain existing behavior
			if err := RunQueryInMemory(&b, q, "./...", "table"); err != nil {
				return &mcp.CallToolResultFor[any]{

					Content: []mcp.Content{
						&mcp.TextContent{
							Text: err.Error(),
						},
					},
					IsError: true,
				}, nil
			}

			return &mcp.CallToolResultFor[any]{
				Content: []mcp.Content{
					&mcp.TextContent{
						Text: b.String(),
					},
				},
			}, nil
		})

		mcp.AddTool(s, &mcp.Tool{
			Name:        "discover",
			Description: "Discover all available tables and their schemas",
		}, func(ctx context.Context, ss *mcp.ServerSession, ctp *mcp.CallToolParamsFor[map[string]any]) (*mcp.CallToolResultFor[any], error) {
			schema, err := getSchema()
			if err != nil {
				return &mcp.CallToolResultFor[any]{
					Content: []mcp.Content{
						&mcp.TextContent{
							Text: err.Error(),
						},
					},
					IsError: true,
				}, nil
			}

			b, err := json.Marshal(schema)
			if err != nil {
				return &mcp.CallToolResultFor[any]{
					Content: []mcp.Content{
						&mcp.TextContent{
							Text: err.Error(),
						},
					},
					IsError: true,
				}, nil
			}

			return &mcp.CallToolResultFor[any]{
				Content: []mcp.Content{
					&mcp.TextContent{
						Text: string(b),
					},
				},
			}, nil
		})

		transport := mcp.NewStdioTransport()
		return s.Run(context.Background(), transport)
	},
}

func getSchema() (map[string]string, error) {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		return nil, fmt.Errorf("failed to open in-memory database: %w", err)
	}
	defer func() { _ = db.Close() }()

	if err := database.CreateTables(db); err != nil {
		return nil, fmt.Errorf("failed to create tables: %w", err)
	}

	rows, err := db.Query("SELECT name, sql FROM sqlite_master WHERE type IN ('table', 'view')")
	if err != nil {
		return nil, fmt.Errorf("failed to query schema: %w", err)
	}
	defer func() { _ = rows.Close() }()

	schema := make(map[string]string)
	for rows.Next() {
		var name, sql string
		if err := rows.Scan(&name, &sql); err != nil {
			return nil, fmt.Errorf("failed to scan schema row: %w", err)
		}
		schema[name] = sql
	}

	return schema, nil
}

func init() {
	rootCmd.AddCommand(mcpCmd)
}
