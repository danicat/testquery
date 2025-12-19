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
	"reflect"
	"testing"
)

func TestCollectCodeLines(t *testing.T) {
	// Create a temporary directory for our test files
	tmpDir, err := os.MkdirTemp("", "test-collect-code-lines-")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create dummy Go files
	file1Content := "package main\n\nfunc main() {}"
	file1Path := filepath.Join(tmpDir, "main.go")
	if err := os.WriteFile(file1Path, []byte(file1Content), 0644); err != nil {
		t.Fatalf("Failed to write file1: %v", err)
	}

	subDir := filepath.Join(tmpDir, "sub")
	if err := os.Mkdir(subDir, 0755); err != nil {
		t.Fatalf("Failed to create subdir: %v", err)
	}
	file2Content := "package sub\n\nfunc helper() {}"
	file2Path := filepath.Join(subDir, "helper.go")
	if err := os.WriteFile(file2Path, []byte(file2Content), 0644); err != nil {
		t.Fatalf("Failed to write file2: %v", err)
	}

	// Call the function we are testing
	codeLines, err := collectCodeLines([]string{tmpDir})
	if err != nil {
		t.Fatalf("collectCodeLines failed: %v", err)
	}

	// Define the expected result
	expected := []CodeLine{
		{Package: tmpDir, File: "main.go", LineNumber: 1, Content: "package main"},
		{Package: tmpDir, File: "main.go", LineNumber: 2, Content: ""},
		{Package: tmpDir, File: "main.go", LineNumber: 3, Content: "func main() {}"},
		{Package: subDir, File: "helper.go", LineNumber: 1, Content: "package sub"},
		{Package: subDir, File: "helper.go", LineNumber: 2, Content: ""},
		{Package: subDir, File: "helper.go", LineNumber: 3, Content: "func helper() {}"},
	}

	// Check if the result matches the expectation
	if !reflect.DeepEqual(codeLines, expected) {
		t.Errorf("collectCodeLines() got = %v, want %v", codeLines, expected)
	}
}

func TestCollectCodeLines_ReadFileError(t *testing.T) {
	// Create a temporary directory for our test files
	tmpDir, err := os.MkdirTemp("", "test-collect-code-lines-read-error-")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a dummy Go file
	fileContent := "package main"
	filePath := filepath.Join(tmpDir, "unreadable.go")
	if err := os.WriteFile(filePath, []byte(fileContent), 0644); err != nil {
		t.Fatalf("Failed to write file: %v", err)
	}

	// Make the file unreadable
	if err := os.Chmod(filePath, 0000); err != nil {
		t.Fatalf("Failed to make file unreadable: %v", err)
	}
	defer os.Chmod(filePath, 0644) // Clean up

	// Call the function we are testing
	_, err = collectCodeLines([]string{tmpDir})

	// Check that we got an error
	if err == nil {
		t.Error("collectCodeLines() did not return an error, but one was expected")
	}
}