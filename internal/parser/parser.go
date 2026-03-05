package parser

import (
	"fmt"

	"rsps/internal/ast"
	"rsps/internal/lexer"
)

type Parser struct {
	lexer     *lexer.Lexer
	curToken  lexer.Token
	peekToken lexer.Token
	errors    []error
}

func New(lex *lexer.Lexer) *Parser {
	parser := &Parser{lexer: lex}
	parser.nextToken()
	parser.nextToken()
	return parser
}

func (p *Parser) Errors() []error {
	return p.errors
}

func (p *Parser) ParseApplication() (*ast.Application, error) {
	if p.curToken.Type != lexer.App {
		return nil, p.newError(p.curToken, "expected 'app' keyword")
	}

	if !p.expectPeek(lexer.Identifier) {
		return nil, p.wrapErrors()
	}

	app := &ast.Application{
		Name: p.curToken.Literal,
		Pos: ast.Position{
			Line:   p.curToken.Line,
			Column: p.curToken.Column,
		},
	}

	if !p.expectPeek(lexer.LBrace) {
		return nil, p.wrapErrors()
	}

	p.nextToken()
	for p.curToken.Type != lexer.RBrace && p.curToken.Type != lexer.EOF {
		entity, err := p.parseEntity()
		if err != nil {
			p.errors = append(p.errors, err)
			return nil, p.wrapErrors()
		}
		app.Entities = append(app.Entities, entity)
		p.nextToken()
	}

	if p.curToken.Type != lexer.RBrace {
		return nil, p.newError(p.curToken, "missing closing '}' for app block")
	}

	if len(p.errors) > 0 {
		return nil, p.wrapErrors()
	}

	return app, nil
}

func (p *Parser) parseEntity() (*ast.Entity, error) {
	if p.curToken.Type != lexer.Identifier {
		return nil, p.newError(p.curToken, "expected entity name")
	}

	entity := &ast.Entity{
		Name: p.curToken.Literal,
		Pos: ast.Position{
			Line:   p.curToken.Line,
			Column: p.curToken.Column,
		},
	}

	if !p.expectPeek(lexer.LBrace) {
		return nil, p.wrapErrors()
	}

	p.nextToken()
	for p.curToken.Type != lexer.RBrace && p.curToken.Type != lexer.EOF {
		field, err := p.parseField()
		if err != nil {
			return nil, err
		}
		entity.Fields = append(entity.Fields, field)
		p.nextToken()
	}

	if p.curToken.Type != lexer.RBrace {
		return nil, p.newError(p.curToken, "missing closing '}' for entity block")
	}

	return entity, nil
}

func (p *Parser) parseField() (*ast.Field, error) {
	if p.curToken.Type != lexer.Identifier {
		return nil, p.newError(p.curToken, "expected field name")
	}

	field := &ast.Field{
		Name: p.curToken.Literal,
		Pos: ast.Position{
			Line:   p.curToken.Line,
			Column: p.curToken.Column,
		},
	}

	if !p.nextTokenIfType(lexer.Type, lexer.Identifier, lexer.Ref) {
		return nil, p.newError(p.peekToken, "expected field type or 'ref <entity>'")
	}

	if p.curToken.Type == lexer.Ref {
		field.Type = ast.TypeRef
		if !p.expectPeek(lexer.Identifier) {
			return nil, p.wrapErrors()
		}
		field.Reference = &ast.Reference{
			Entity: p.curToken.Literal,
			Pos: ast.Position{
				Line:   p.curToken.Line,
				Column: p.curToken.Column,
			},
		}
	} else {
		field.Type = ast.FieldType(p.curToken.Literal)
	}

	if p.peekToken.Type == lexer.Question {
		p.nextToken()
		field.Nullable = true
	}

	for p.peekToken.Type == lexer.Unique || p.peekToken.Type == lexer.Index {
		p.nextToken()
		switch p.curToken.Type {
		case lexer.Unique:
			field.Attributes = append(field.Attributes, ast.AttrUnique)
		case lexer.Index:
			field.Attributes = append(field.Attributes, ast.AttrIndex)
		}
	}

	if p.peekToken.Type == lexer.Equal {
		p.nextToken()
		p.nextToken()
		literal, err := p.parseLiteral()
		if err != nil {
			return nil, err
		}
		field.Default = literal
	}

	return field, nil
}

func (p *Parser) parseLiteral() (*ast.Literal, error) {
	literal := &ast.Literal{
		Value: p.curToken.Literal,
		Pos: ast.Position{
			Line:   p.curToken.Line,
			Column: p.curToken.Column,
		},
	}

	switch p.curToken.Type {
	case lexer.String:
		literal.Kind = "string"
	case lexer.Number:
		literal.Kind = "number"
	case lexer.True, lexer.False:
		literal.Kind = "bool"
	case lexer.Now:
		literal.Kind = "now"
	case lexer.Identifier, lexer.Type:
		literal.Kind = "identifier"
	default:
		return nil, p.newError(p.curToken, "invalid literal")
	}

	return literal, nil
}

func (p *Parser) nextToken() {
	p.curToken = p.peekToken
	p.peekToken = p.lexer.NextToken()
}

func (p *Parser) expectPeek(tokenType lexer.TokenType) bool {
	if p.peekToken.Type == tokenType {
		p.nextToken()
		return true
	}
	p.errors = append(p.errors, p.newError(p.peekToken, fmt.Sprintf("expected %s, got %s", tokenType, p.peekToken.Type)))
	return false
}

func (p *Parser) nextTokenIfType(allowed ...lexer.TokenType) bool {
	for _, tokenType := range allowed {
		if p.peekToken.Type == tokenType {
			p.nextToken()
			return true
		}
	}
	return false
}

func (p *Parser) newError(token lexer.Token, message string) error {
	return fmt.Errorf("parse error at %d:%d: %s", token.Line, token.Column, message)
}

func (p *Parser) wrapErrors() error {
	if len(p.errors) == 0 {
		return nil
	}
	if len(p.errors) == 1 {
		return p.errors[0]
	}

	message := "parse errors:\n"
	for _, err := range p.errors {
		message += " - " + err.Error() + "\n"
	}
	return fmt.Errorf("%s", message)
}
