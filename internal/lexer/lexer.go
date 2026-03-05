package lexer

import "fmt"

type Lexer struct {
	input        []rune
	position     int
	readPosition int
	ch           rune
	line         int
	column       int
}

func New(input string) *Lexer {
	lexer := &Lexer{
		input:  []rune(input),
		line:   1,
		column: 0,
	}
	lexer.readChar()
	return lexer
}

func (l *Lexer) NextToken() Token {
	l.skipWhitespace()

	token := Token{Line: l.line, Column: l.column}

	switch l.ch {
	case '{':
		token.Type = LBrace
		token.Literal = "{"
		l.readChar()
	case '}':
		token.Type = RBrace
		token.Literal = "}"
		l.readChar()
	case '?':
		token.Type = Question
		token.Literal = "?"
		l.readChar()
	case '=':
		token.Type = Equal
		token.Literal = "="
		l.readChar()
	case '@':
		startLine := l.line
		startColumn := l.column
		l.readChar()
		ident := l.readIdentifier()
		token.Line = startLine
		token.Column = startColumn
		switch ident {
		case "unique":
			token.Type = Unique
			token.Literal = "@unique"
		case "index":
			token.Type = Index
			token.Literal = "@index"
		default:
			token.Type = Illegal
			token.Literal = fmt.Sprintf("@%s", ident)
		}
	case '"':
		token.Type = String
		token.Literal = l.readString()
	case 0:
		token.Type = EOF
		token.Literal = ""
	default:
		if isLetter(l.ch) {
			literal := l.readIdentifier()
			token.Type = LookupIdent(literal)
			token.Literal = literal
			return token
		}
		if isDigit(l.ch) || (l.ch == '-' && isDigit(l.peekChar())) {
			token.Type = Number
			token.Literal = l.readNumber()
			return token
		}
		token.Type = Illegal
		token.Literal = string(l.ch)
		l.readChar()
	}

	return token
}

func (l *Lexer) readChar() {
	if l.readPosition >= len(l.input) {
		l.position = l.readPosition
		l.ch = 0
		return
	}
	l.position = l.readPosition
	l.ch = l.input[l.readPosition]
	l.readPosition++

	if l.ch == '\n' {
		l.line++
		l.column = 0
		return
	}
	l.column++
}

func (l *Lexer) skipWhitespace() {
	for l.ch == ' ' || l.ch == '\t' || l.ch == '\n' || l.ch == '\r' {
		l.readChar()
	}
}

func (l *Lexer) readIdentifier() string {
	start := l.position
	for isLetter(l.ch) || isDigit(l.ch) {
		l.readChar()
	}
	return string(l.input[start:l.position])
}

func (l *Lexer) readNumber() string {
	start := l.position
	if l.ch == '-' {
		l.readChar()
	}
	hasDot := false
	for isDigit(l.ch) || (!hasDot && l.ch == '.') {
		if l.ch == '.' {
			hasDot = true
		}
		l.readChar()
	}
	return string(l.input[start:l.position])
}

func (l *Lexer) readString() string {
	startLine := l.line
	startColumn := l.column
	l.readChar()
	start := l.position

	for l.ch != '"' && l.ch != 0 {
		if l.ch == '\\' && l.peekChar() == '"' {
			l.readChar()
		}
		l.readChar()
	}

	if l.ch == 0 {
		return fmt.Sprintf("unterminated string at %d:%d", startLine, startColumn)
	}

	value := string(l.input[start:l.position])
	l.readChar()
	return value
}

func (l *Lexer) peekChar() rune {
	if l.readPosition >= len(l.input) {
		return 0
	}
	return l.input[l.readPosition]
}

func isLetter(ch rune) bool {
	return (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || ch == '_'
}

func isDigit(ch rune) bool {
	return ch >= '0' && ch <= '9'
}
