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

package collector

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestCollectCoverageResults(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "test-collect-coverage-")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	goMod := "module example.com/testmod\n\ngo 1.24\n"
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goMod), 0644); err != nil {
		t.Fatalf("Failed to write go.mod: %v", err)
	}

	divGo := `package testdata

import "fmt"

// Div divides two integers.
// It returns an error if the divisor is zero.
func Div(a, b int) (int, error) {
	if b == 0 {
		return 0, fmt.Errorf("division by zero")
	}
	return a / b, nil
}
`
	divGoPath := filepath.Join(tmpDir, "div.go")
	if err := os.WriteFile(divGoPath, []byte(divGo), 0644); err != nil {
		t.Fatalf("Failed to write div.go: %v", err)
	}

	coverageFile := filepath.Join(tmpDir, "coverage.out")
	coverageData := fmt.Sprintf(`mode: set
%s:7.52,10.6 2 1
%s:12.2,12.31 1 1
`, "example.com/testmod/div.go", "example.com/testmod/div.go")
	if err := os.WriteFile(coverageFile, []byte(coverageData), 0644); err != nil {
		t.Fatalf("Failed to write coverage.out: %v", err)
	}

	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current working directory: %v", err)
	}
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("Failed to change directory: %v", err)
	}
	defer func() { _ = os.Chdir(oldWd) }()

	coverageResults, err := collectCoverageResults([]string{"./..."})
	if err != nil {
		t.Fatalf("collectCoverageResults failed: %v", err)
	}

	expected := []CoverageResult{
		{
			Package:         "example.com/testmod",
			File:            "div.go",
			StartLine:       7,
			StartColumn:     52,
			EndLine:         10,
			EndColumn:       6,
			StatementNumber: 2,
			Count:           1,
			FunctionName:    "Div",
		},
		{
			Package:         "example.com/testmod",
			File:            "div.go",
			StartLine:       12,
			StartColumn:     2,
			EndLine:         12,
			EndColumn:       31,
			StatementNumber: 1,
			Count:           1,
			FunctionName:    "Div",
		},
	}

	if !reflect.DeepEqual(coverageResults, expected) {
		t.Errorf("collectCoverageResults() got = %+v, want %+v", coverageResults, expected)
	}
}
