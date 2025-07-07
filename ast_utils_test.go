package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGetFunctionName(t *testing.T) {
	// Create a temporary directory for test files
	tmpDir, err := os.MkdirTemp("", "testgetfuncname")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Test case 1: Simple function
	content1 := `package main

func MyFunction() {
	// some code
}
`
	filePath1 := filepath.Join(tmpDir, "file1.go")
	if err := os.WriteFile(filePath1, []byte(content1), 0644); err != nil {
		t.Fatalf("Failed to write test file1: %v", err)
	}

	// Test case 2: Multiple functions and comments
	content2 := `package main

import "fmt"

// Some comments
func AnotherFunction() {
	fmt.Println("hello")
}

/* More comments */
func YetAnotherFunction(param int) string {
	if param > 0 {
		return "positive"
	}
	return "non-positive"
}
`
	filePath2 := filepath.Join(tmpDir, "file2.go")
	if err := os.WriteFile(filePath2, []byte(content2), 0644); err != nil {
		t.Fatalf("Failed to write test file2: %v", err)
	}

	// Test case 3: File with no functions
	content3 := `package main

var GlobalVar = 10
const MyConst = "hello"
`
	filePath3 := filepath.Join(tmpDir, "file3.go")
	if err := os.WriteFile(filePath3, []byte(content3), 0644); err != nil {
		t.Fatalf("Failed to write test file3: %v", err)
	}


	tests := []struct {
		name         string
		filePath     string
		lineNumber   int
		expectedFunc string
		expectError  bool
	}{
		{"simple function line 3", filePath1, 3, "MyFunction", false},
		{"simple function line 4", filePath1, 4, "MyFunction", false},
		{"simple function line 5", filePath1, 5, "MyFunction", false},
		{"line before function", filePath1, 2, "", false}, // Expected empty as it's not within a func
		{"line after function", filePath1, 6, "", false},  // Expected empty
		{"another function line 7", filePath2, 7, "AnotherFunction", false},
		{"yet another function line 12", filePath2, 12, "YetAnotherFunction", false},
		{"yet another function line 15", filePath2, 15, "YetAnotherFunction", false},
		{"line between functions", filePath2, 9, "", false},
		{"non-existent file", "nonexistent.go", 1, "", true},
		{"file with no functions", filePath3, 3, "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			funcName, err := getFunctionName(tt.filePath, tt.lineNumber)
			if (err != nil) != tt.expectError {
				t.Errorf("getFunctionName() error = %v, expectError %v", err, tt.expectError)
				return
			}
			if funcName != tt.expectedFunc {
				t.Errorf("getFunctionName() = %v, want %v", funcName, tt.expectedFunc)
			}
		})
	}
}
