package parser

import (
	"fmt"
	"os"

	"rsps/internal/ast"
	"rsps/internal/lexer"
)

func ParseString(source string) (*ast.Application, error) {
	lex := lexer.New(source)
	prs := New(lex)
	app, err := prs.ParseApplication()
	if err != nil {
		return nil, err
	}
	if len(prs.Errors()) > 0 {
		return nil, prs.wrapErrors()
	}
	return app, nil
}

func ParseFile(path string) (*ast.Application, error) {
	bytes, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read rsps file '%s': %w", path, err)
	}
	return ParseString(string(bytes))
}
