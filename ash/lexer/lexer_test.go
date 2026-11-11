package lexer

import (
	"testing"
)

func lexStr(t *testing.T, src string) []Token {
	tokens, err := Tokenize(src)
	if err != nil {
		t.Fatalf("tokenize error: %v", err)
	}
	return tokens
}

func ident(s string) Token { return Token{Kind: TkIdent, Literal: s} }

func TestIdentifiers(t *testing.T) {
	tokens := lexStr(t, "hello world _foo bar123")
	expected := []Token{ident("hello"), ident("world"), ident("_foo"), ident("bar123")}
	assertTokens(t, tokens, expected)
}

func TestKeywords(t *testing.T) {
	src := "if else for while fn return do with subagent using try fail evaluate accept partial upto wait compact include exec print exit env break continue not and or"
	tokens := lexStr(t, src)
	expected := []string{
		"if", "else", "for", "while", "fn", "return",
		"do", "with", "subagent", "using",
		"try", "fail", "evaluate", "accept", "partial", "upto",
		"wait", "compact", "include", "exec", "print", "exit", "env",
		"break", "continue", "not", "and", "or",
	}
	var got []Token
	for _, tok := range tokens {
		if tok.Kind == TkNewline {
			continue
		}
		got = append(got, tok)
	}
	if len(got) != len(expected) {
		t.Fatalf("expected %d tokens, got %d: %v", len(expected), len(got), got)
	}
	for i, exp := range expected {
		if got[i].Kind != TkIdent || got[i].Literal != exp {
			t.Errorf("token %d: expected ident(%q), got %s(%q)", i, exp, got[i].Kind, got[i].Literal)
		}
	}
}

func TestNumbers(t *testing.T) {
	tests := []struct {
		src  string
		kind TokenKind
		lit  string
	}{
		{"123", TkInt, "123"},
		{"3.14", TkFloat, "3.14"},
		{"0xDEAD", TkInt, "0xDEAD"},
		{"0xbeef", TkInt, "0xbeef"},
	}
	for _, tt := range tests {
		tokens := lexStr(t, tt.src)
		if len(tokens) != 1 {
			t.Errorf("%q: expected 1 token, got %d", tt.src, len(tokens))
			continue
		}
		if tokens[0].Kind != tt.kind || tokens[0].Literal != tt.lit {
			t.Errorf("%q: expected %s(%q), got %s(%q)", tt.src, tt.kind, tt.lit, tokens[0].Kind, tokens[0].Literal)
		}
	}
}

func TestOperatorsAndDelimiters(t *testing.T) {
	tests := []struct {
		src  string
		kind TokenKind
		lit  string
	}{
		{"=", TkAssign, "="},
		{"==", TkEq, "=="},
		{"!=", TkNeq, "!="},
		{">", TkGt, ">"},
		{"<", TkLt, "<"},
		{">=", TkGte, ">="},
		{"<=", TkLte, "<="},
		{"+", TkPlus, "+"},
		{"-", TkMinus, "-"},
		{"*", TkStar, "*"},
		{"/", TkSlash, "/"},
		{"(", TkLParen, "("},
		{")", TkRParen, ")"},
		{"{", TkLBrace, "{"},
		{"}", TkRBrace, "}"},
		{",", TkComma, ","},
		{"&", TkAmpersand, "&"},
		{"@", TkAt, "@"},
		{":", TkColon, ":"},
	}
	for _, tt := range tests {
		tokens := lexStr(t, tt.src)
		if len(tokens) != 1 {
			t.Errorf("%q: expected 1 token, got %d", tt.src, len(tokens))
			continue
		}
		if tokens[0].Kind != tt.kind || tokens[0].Literal != tt.lit {
			t.Errorf("%q: expected %s(%q), got %s(%q)", tt.src, tt.kind, tt.lit, tokens[0].Kind, tokens[0].Literal)
		}
	}
}

func TestString(t *testing.T) {
	tokens := lexStr(t, `"hello world"`)
	if len(tokens) != 1 || tokens[0].Kind != TkString || tokens[0].Literal != "hello world" {
		t.Fatalf("expected string(%q), got %v", "hello world", tokens)
	}
}

func TestStringEscapes(t *testing.T) {
	tokens := lexStr(t, `"hello \"world\" \\ \$"`)
	if len(tokens) != 1 || tokens[0].Kind != TkString {
		t.Fatalf("expected string, got %v", tokens)
	}
	expected := `hello "world" \ $`
	if tokens[0].Literal != expected {
		t.Errorf("expected %q, got %q", expected, tokens[0].Literal)
	}
}

func TestTextBlock(t *testing.T) {
	src := "```\nhello\nworld\n```"
	tokens := lexStr(t, src)
	if len(tokens) != 1 || tokens[0].Kind != TkTextBlock {
		t.Fatalf("expected text block, got %v", tokens)
	}
}

func TestVarRefBare(t *testing.T) {
	tokens := lexStr(t, `$NAME $? ${stderr}`)
	if len(tokens) != 3 {
		t.Fatalf("expected 3 tokens, got %d: %v", len(tokens), tokens)
	}
	if tokens[0].Kind != TkDollarLBrace || tokens[0].Literal != "NAME" {
		t.Errorf("expected ${NAME}, got %s(%q)", tokens[0].Kind, tokens[0].Literal)
	}
	if tokens[1].Kind != TkDollarLBrace || tokens[1].Literal != "?" {
		t.Errorf("expected ${?}, got %s(%q)", tokens[1].Kind, tokens[1].Literal)
	}
	if tokens[2].Kind != TkDollarLBrace || tokens[2].Literal != "stderr" {
		t.Errorf("expected ${stderr}, got %s(%q)", tokens[2].Kind, tokens[2].Literal)
	}
}

func TestCommandSubst(t *testing.T) {
	tokens := lexStr(t, `$(echo hello)`)
	if len(tokens) != 1 || tokens[0].Kind != TkDollarLParen {
		t.Fatalf("expected $(cmd), got %v", tokens)
	}
	if tokens[0].Literal != "echo hello" {
		t.Errorf("expected 'echo hello', got %q", tokens[0].Literal)
	}
}

func TestShebang(t *testing.T) {
	src := "#!opencode:1.2.0\nprint hello"
	tokens, err := Tokenize(src)
	if err != nil {
		t.Fatal(err)
	}
	if len(tokens) < 3 || tokens[0].Kind != TkShebang {
		t.Fatalf("expected shebang first, got %v", tokens[0])
	}
	if tokens[0].Literal != "#!opencode:1.2.0" {
		t.Errorf("expected shebang %q, got %q", "#!opencode:1.2.0", tokens[0].Literal)
	}
}

func TestCompactConfig(t *testing.T) {
	src := `#!compact mode=on window=64000 strategy=truncate`
	tokens := lexStr(t, src)
	if len(tokens) != 1 || tokens[0].Kind != TkCompactConfig {
		t.Fatalf("expected compact config, got %v", tokens)
	}
}

func TestComment(t *testing.T) {
	tokens := lexStr(t, `hello # this is a comment
world`)
	if len(tokens) != 3 {
		t.Fatalf("expected 3 tokens (hello, newline, world), got %d: %v", len(tokens), tokens)
	}
	if tokens[0].Literal != "hello" || tokens[2].Literal != "world" {
		t.Errorf("expected [hello, newline, world], got %v", tokens)
	}
}

func TestNewlines(t *testing.T) {
	tokens := lexStr(t, "a\nb\nc")
	if len(tokens) != 5 {
		t.Fatalf("expected 5 tokens (a, nl, b, nl, c), got %d: %v", len(tokens), tokens)
	}
	if tokens[0].Literal != "a" || tokens[2].Literal != "b" || tokens[4].Literal != "c" {
		t.Errorf("unexpected token sequence: %v", tokens)
	}
	for _, i := range []int{1, 3} {
		if tokens[i].Kind != TkNewline {
			t.Errorf("token %d: expected newline, got %v", i, tokens[i])
		}
	}
}

func TestComplexExpr(t *testing.T) {
	tokens := lexStr(t, `$? == 0 and not $FAILED`)
	if len(tokens) != 6 {
		t.Fatalf("expected 6 tokens, got %d: %v", len(tokens), tokens)
	}
	expected := []TokenKind{TkDollarLBrace, TkEq, TkInt, TkIdent, TkIdent, TkDollarLBrace}
	for i, exp := range expected {
		if tokens[i].Kind != exp {
			t.Errorf("token %d: expected %s, got %s(%q)", i, exp, tokens[i].Kind, tokens[i].Literal)
		}
	}
}

func assertTokens(t *testing.T, got, expected []Token) {
	t.Helper()
	if len(got) != len(expected) {
		t.Fatalf("token count: expected %d, got %d\n  expected: %v\n  got:      %v", len(expected), len(got), expected, got)
	}
	for i := range got {
		if got[i].Kind != expected[i].Kind || got[i].Literal != expected[i].Literal {
			t.Errorf("token %d:\n  expected %s(%q)\n  got      %s(%q)", i, expected[i].Kind, expected[i].Literal, got[i].Kind, got[i].Literal)
		}
	}
}
