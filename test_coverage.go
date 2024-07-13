package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os/exec"
	"path/filepath"

	"golang.org/x/tools/cover"
)

// TestCoverageResult represents the structure of a test-specific coverage result
type TestCoverageResult struct {
	TestName        string `json:"test_name"`
	Package         string `json:"package"`
	File            string `json:"file"`
	FromLine        int    `json:"from_line"`
	FromColumn      int    `json:"from_col"`
	ToLine          int    `json:"to_line"`
	ToColumn        int    `json:"to_col"`
	StatementNumber int    `json:"stmt_num"`
	Count           int    `json:"count"`
	Function        string `json:"function"`
}

func collectTestCoverageResults(pkgDir string, testResults []TestEvent) ([]TestCoverageResult, error) {
	var results []TestCoverageResult

	for _, test := range testResults {
		cmd := exec.Command("go", "test", pkgDir, "-run", "^"+test.Test+"$", "-coverprofile="+test.Test+".out")
		cmd.Run()

		profiles, err := cover.ParseProfiles(test.Test + ".out")
		if err != nil {
			return nil, err
		}

		for _, profile := range profiles {
			packageName := filepath.Dir(profile.FileName)
			fileName := filepath.Base(profile.FileName)
			for _, block := range profile.Blocks {
				functionName, err := getFunctionName(pkgDir+"/"+fileName, block.StartLine)
				if err != nil {
					return nil, fmt.Errorf("failed to retrieve function name: %w", err)
				}

				results = append(results, TestCoverageResult{
					TestName:        test.Test,
					Package:         packageName,
					File:            fileName,
					FromLine:        block.StartLine,
					FromColumn:      block.StartCol,
					ToLine:          block.EndLine,
					ToColumn:        block.EndCol,
					StatementNumber: block.NumStmt,
					Count:           block.Count,
					Function:        functionName,
				})
			}
		}
	}

	return results, nil
}

// getFunctionName returns the name of the function at the given line number
func getFunctionName(fileName string, lineNumber int) (string, error) {
	fs := token.NewFileSet()
	node, err := parser.ParseFile(fs, fileName, nil, 0)
	if err != nil {
		return "", fmt.Errorf("failed to parse file: %w", err)
	}

	for _, decl := range node.Decls {
		if funcDecl, ok := decl.(*ast.FuncDecl); ok {
			start := fs.Position(funcDecl.Pos()).Line
			end := fs.Position(funcDecl.End()).Line
			if start <= lineNumber && lineNumber <= end {
				return funcDecl.Name.Name, nil
			}
		}
	}

	return "", nil
}
