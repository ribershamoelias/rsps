package lexer

type TokenType string

const (
	Illegal TokenType = "ILLEGAL"
	EOF     TokenType = "EOF"

	Identifier TokenType = "IDENTIFIER"
	Type       TokenType = "TYPE"
	Number     TokenType = "NUMBER"
	String     TokenType = "STRING"

	App   TokenType = "APP"
	Ref   TokenType = "REF"
	True  TokenType = "TRUE"
	False TokenType = "FALSE"
	Now   TokenType = "NOW"

	LBrace   TokenType = "LBRACE"
	RBrace   TokenType = "RBRACE"
	Question TokenType = "QUESTION"
	Equal    TokenType = "EQUAL"

	Unique TokenType = "UNIQUE"
	Index  TokenType = "INDEX"
)

type Token struct {
	Type    TokenType
	Literal string
	Line    int
	Column  int
}

var keywords = map[string]TokenType{
	"app":      App,
	"ref":      Ref,
	"true":     True,
	"false":    False,
	"now":      Now,
	"string":   Type,
	"text":     Type,
	"int":      Type,
	"float":    Type,
	"bool":     Type,
	"date":     Type,
	"datetime": Type,
	"json":     Type,
}

func LookupIdent(literal string) TokenType {
	if tokenType, ok := keywords[literal]; ok {
		return tokenType
	}
	return Identifier
}
