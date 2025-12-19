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

package collector

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFindFunction(t *testing.T) {
	// Create a temporary directory and a dummy Go file
	tmpDir, err := os.MkdirTemp("", "test-find-function-")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	goFile := `package test

import "fmt"

func main() {
	fmt.Println("Hello, World!")
}

func anotherFunction() {
	fmt.Println("This is another function.")
}
`
	goFilePath := filepath.Join(tmpDir, "test.go")
	if err := os.WriteFile(goFilePath, []byte(goFile), 0644); err != nil {
		t.Fatalf("Failed to write test.go: %v", err)
	}

	// Create a new function finder
	ff, err := newFunctionFinder(goFilePath)
	if err != nil {
		t.Fatalf("newFunctionFinder failed: %v", err)
	}

	testCases := []struct {
		line int
		want string
	}{
		{5, "main"},
		{6, "main"},
		{9, "anotherFunction"},
		{10, "anotherFunction"},
		{1, ""},
		{12, ""},
	}

	for _, tc := range testCases {
		got := ff.findFunction(tc.line)
		if got != tc.want {
			t.Errorf("findFunction(%d) = %q, want %q", tc.line, got, tc.want)
		}
	}
}
