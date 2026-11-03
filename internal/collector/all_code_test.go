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
	"reflect"
	"testing"
)

func TestCollectCodeLines(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "test-collect-code-lines-")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	goMod := "module example.com/testmod\n\ngo 1.24\n"
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goMod), 0644); err != nil {
		t.Fatalf("Failed to write go.mod: %v", err)
	}

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

	// Pass both root tmpDir and subDir to also verify deduplication.
	codeLines, err := collectCodeLines([]string{tmpDir, subDir})
	if err != nil {
		t.Fatalf("collectCodeLines failed: %v", err)
	}

	expected := []CodeLine{
		{Package: "example.com/testmod", File: "main.go", LineNumber: 1, Content: "package main"},
		{Package: "example.com/testmod", File: "main.go", LineNumber: 2, Content: ""},
		{Package: "example.com/testmod", File: "main.go", LineNumber: 3, Content: "func main() {}"},
		{Package: "example.com/testmod/sub", File: "sub/helper.go", LineNumber: 1, Content: "package sub"},
		{Package: "example.com/testmod/sub", File: "sub/helper.go", LineNumber: 2, Content: ""},
		{Package: "example.com/testmod/sub", File: "sub/helper.go", LineNumber: 3, Content: "func helper() {}"},
	}

	if !reflect.DeepEqual(codeLines, expected) {
		t.Errorf("collectCodeLines() got = %+v, want %+v", codeLines, expected)
	}
}
