package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
)

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
