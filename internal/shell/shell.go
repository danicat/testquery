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

package shell

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"strings"

	"github.com/chzyer/readline"
	"github.com/danicat/testquery/internal/query"
)

func Prompt(ctx context.Context, db *sql.DB, r io.Reader, w io.Writer) error {
	rl, err := readline.NewEx(&readline.Config{
		Prompt:                 "> ",
		HistoryFile:            "/tmp/testquery-history",
		DisableAutoSaveHistory: true,
		Stdin:                  io.NopCloser(r),
		Stdout:                 w,
	})
	if err != nil {
		return fmt.Errorf("failed to create readline instance: %w", err)
	}
	defer rl.Close()

	var cmds []string
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		line, err := rl.Readline()
		if err != nil {
			if err == readline.ErrInterrupt {
				if len(cmds) == 0 {
					return nil
				}
				cmds = cmds[:0]
				continue
			} else if err == io.EOF {
				return nil
			}
			return fmt.Errorf("failed to read line: %w", err)
		}

		line = strings.TrimSpace(line)
		if len(line) == 0 {
			continue
		}

		cmds = append(cmds, line)
		if !strings.HasSuffix(line, ";") {
			rl.SetPrompt(">>> ")
			continue
		}

		cmd := strings.Join(cmds, " ")
		cmds = cmds[:0]
		rl.SetPrompt("> ")
		rl.SaveHistory(cmd)

		if err = query.Execute(w, db, cmd); err != nil {
			fmt.Fprintf(w, "ERROR: %v\n", err)
		}
	}
}
