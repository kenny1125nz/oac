package lexer

import "fmt"

type TokenKind int

const (
	TkEOF TokenKind = iota
	TkNewline
	TkIdent
	TkString
	TkTextBlock
	TkInt
	TkFloat
	TkAssign
	TkEq
	TkNeq
	TkGt
	TkLt
	TkGte
	TkLte
	TkPlus
	TkMinus
	TkStar
	TkSlash
	TkLParen
	TkRParen
	TkLBrace
	TkRBrace
	TkComma
	TkAmpersand
	TkAt
	TkDollar
	TkDollarLParen
	TkDollarLBrace
	TkLBracket
	TkRBracket
	TkColon
	TkHash
	TkShebang
	TkCompactConfig
)

var tokenNames = map[TokenKind]string{
	TkEOF:           "EOF",
	TkNewline:       "newline",
	TkIdent:         "identifier",
	TkString:        "string",
	TkTextBlock:     "text block",
	TkInt:           "integer",
	TkFloat:         "float",
	TkAssign:        "=",
	TkEq:            "==",
	TkNeq:           "!=",
	TkGt:            ">",
	TkLt:            "<",
	TkGte:           ">=",
	TkLte:           "<=",
	TkPlus:          "+",
	TkMinus:         "-",
	TkStar:          "*",
	TkSlash:         "/",
	TkLParen:        "(",
	TkRParen:        ")",
	TkLBrace:        "{",
	TkRBrace:        "}",
	TkComma:         ",",
	TkAmpersand:     "&",
	TkAt:            "@",
	TkLBracket:      "[",
	TkRBracket:      "]",
	TkDollar:        "$",
	TkDollarLParen:  "$(",
	TkDollarLBrace:  "${",
	TkColon:         ":",
	TkHash:          "#",
	TkShebang:       "shebang",
	TkCompactConfig: "compact config",
}

func (k TokenKind) String() string {
	if n, ok := tokenNames[k]; ok {
		return n
	}
	return fmt.Sprintf("token(%d)", k)
}

type Token struct {
	Kind    TokenKind
	Literal string
	Line    int
	Col     int
}

func (t Token) String() string {
	if t.Literal != "" {
		return fmt.Sprintf("%s(%q)@%d:%d", t.Kind, t.Literal, t.Line, t.Col)
	}
	return fmt.Sprintf("%s@%d:%d", t.Kind, t.Line, t.Col)
}

var keywords = map[string]struct{}{}

func init() {
	kw := []string{
		"if", "else", "for", "while", "fn", "return",
		"do", "with", "within", "subagent", "using",
		"try", "fail", "evaluate", "accept", "partial", "upto",
		"wait", "compact", "include", "exec", "print", "exit", "env",
		"break", "continue", "not", "and", "or",
	}
	for _, w := range kw {
		keywords[w] = struct{}{}
	}
}

func IsKeyword(s string) bool {
	_, ok := keywords[s]
	return ok
}
