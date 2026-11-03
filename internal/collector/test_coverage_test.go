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
	"os"
	"path/filepath"
	"testing"
)

func TestGetFunctionName(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "test-get-function-name-")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	sourceFile := filepath.Join(tmpDir, "main.go")
	sourceCode := `package main

func main() {
	// A comment
}

func anotherFunction() {
	// Another comment
}
`
	if err := os.WriteFile(sourceFile, []byte(sourceCode), 0644); err != nil {
		t.Fatalf("Failed to write source file: %v", err)
	}

	finder, err := newFunctionFinderWithContext(sourceFile, nil)
	if err != nil {
		t.Fatalf("newFunctionFinderWithContext() failed: %v", err)
	}

	testCases := []struct {
		name         string
		lineNumber   int
		expectedFunc string
	}{
		{"Inside main", 4, "main"},
		{"Inside anotherFunction", 8, "anotherFunction"},
		{"Outside any function", 1, ""},
		{"On a blank line", 2, ""},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			funcName := finder.findFunction(tc.lineNumber)
			if funcName != tc.expectedFunc {
				t.Errorf("findFunction() = %v, want %v", funcName, tc.expectedFunc)
			}
		})
	}
}

func TestBuildTestRunRegex(t *testing.T) {
	got := buildTestRunRegex("TestCountSyllables/beautiful/case_(1)")
	want := `^TestCountSyllables$/^beautiful$/^case_\(1\)$`
	if got != want {
		t.Errorf("buildTestRunRegex() = %q, want %q", got, want)
	}
}
