package lexer

import (
	"fmt"
	"strings"
	"unicode"
)

type Lexer struct {
	src     []rune
	pos     int
	line    int
	col     int
	startOf bool
}

func New(src string) *Lexer {
	return &Lexer{
		src:     []rune(src),
		line:    1,
		col:     1,
		startOf: true,
	}
}

func (l *Lexer) Next() Token {
	for l.pos < len(l.src) {
		ch := l.src[l.pos]

		if unicode.IsSpace(ch) && ch != '\n' {
			l.advance()
			continue
		}

		if l.startOf && ch == '#' && l.pos+1 < len(l.src) && l.src[l.pos+1] == '!' {
			return l.readShebang()
		}

		l.startOf = false

		if ch == '#' && (l.pos+1 >= len(l.src) || l.src[l.pos+1] != '!') {
			return l.readComment()
		}

		if ch == '\n' {
			l.startOf = true
			tok := l.makeToken(TkNewline, "\\n")
			l.advance()
			return tok
		}

		if ch == '`' {
			return l.readTextBlock()
		}

		if ch == '"' {
			return l.readString()
		}

		if ch == '$' {
			return l.readDollar()
		}

		if ch == '@' {
			tok := l.makeToken(TkAt, "@")
			l.advance()
			return tok
		}

		if ch == '&' {
			tok := l.makeToken(TkAmpersand, "&")
			l.advance()
			return tok
		}

		if ch == '(' {
			tok := l.makeToken(TkLParen, "(")
			l.advance()
			return tok
		}
		if ch == ')' {
			tok := l.makeToken(TkRParen, ")")
			l.advance()
			return tok
		}
		if ch == '{' {
			tok := l.makeToken(TkLBrace, "{")
			l.advance()
			return tok
		}
		if ch == '}' {
			tok := l.makeToken(TkRBrace, "}")
			l.advance()
			return tok
		}
		if ch == '[' {
			tok := l.makeToken(TkLBracket, "[")
			l.advance()
			return tok
		}
		if ch == ']' {
			tok := l.makeToken(TkRBracket, "]")
			l.advance()
			return tok
		}
		if ch == ',' {
			tok := l.makeToken(TkComma, ",")
			l.advance()
			return tok
		}
		if ch == ':' {
			tok := l.makeToken(TkColon, ":")
			l.advance()
			return tok
		}

		if ch == '=' || ch == '!' || ch == '>' || ch == '<' {
			return l.readComparison()
		}

		if ch == '+' || ch == '-' || ch == '*' || ch == '/' {
			return l.readOperator()
		}

		if ch == '0' && l.pos+1 < len(l.src) && l.src[l.pos+1] == 'x' {
			return l.readHex()
		}
		if unicode.IsDigit(ch) {
			return l.readNumber()
		}

		if unicode.IsLetter(ch) || ch == '_' {
			return l.readIdent()
		}

		tok := l.makeToken(TkIdent, string(ch))
		l.advance()
		return tok
	}

	return l.makeToken(TkEOF, "")
}

func (l *Lexer) advance() {
	if l.pos < len(l.src) {
		if l.src[l.pos] == '\n' {
			l.line++
			l.col = 1
		} else {
			l.col++
		}
		l.pos++
	}
}

func (l *Lexer) peek() rune {
	if l.pos+1 < len(l.src) {
		return l.src[l.pos+1]
	}
	return 0
}

func (l *Lexer) makeToken(kind TokenKind, literal string) Token {
	return Token{Kind: kind, Literal: literal, Line: l.line, Col: l.col}
}

func (l *Lexer) readShebang() Token {
	startPos := l.pos
	for l.pos < len(l.src) && l.src[l.pos] != '\n' {
		l.advance()
	}
	line := string(l.src[startPos:l.pos])
	return Token{Kind: TkShebang, Literal: line, Line: l.line, Col: l.col}
}

func (l *Lexer) readComment() Token {
	for l.pos < len(l.src) && l.src[l.pos] != '\n' {
		l.advance()
	}
	return l.Next()
}

func (l *Lexer) readTextBlock() Token {
	startLine, startCol := l.line, l.col
	l.advance()

	if l.pos+1 < len(l.src) && l.src[l.pos] == '`' && l.src[l.pos+1] == '`' {
		l.advance()
		l.advance()
	} else {
		return Token{Kind: TkIdent, Literal: "`", Line: startLine, Col: startCol}
	}

	var buf strings.Builder
	for l.pos < len(l.src) {
		if l.src[l.pos] == '`' {
			if l.pos+2 < len(l.src) && l.src[l.pos+1] == '`' && l.src[l.pos+2] == '`' {
				l.advance()
				l.advance()
				l.advance()
				break
			}
		}
		buf.WriteRune(l.src[l.pos])
		l.advance()
	}
	return Token{Kind: TkTextBlock, Literal: buf.String(), Line: startLine, Col: startCol}
}

func (l *Lexer) readString() Token {
	startLine, startCol := l.line, l.col
	l.advance()
	var buf strings.Builder
	for l.pos < len(l.src) {
		ch := l.src[l.pos]
		if ch == '"' {
			l.advance()
			break
		}
		if ch == '\\' {
			l.advance()
			if l.pos < len(l.src) {
				next := l.src[l.pos]
				switch next {
				case '"':
					buf.WriteRune('"')
				case '$':
					buf.WriteRune('$')
				case '\\':
					buf.WriteRune('\\')
				case 'n':
					buf.WriteRune('\n')
				case 't':
					buf.WriteRune('\t')
				default:
					buf.WriteRune('\\')
					buf.WriteRune(next)
				}
				l.advance()
			}
			continue
		}
		buf.WriteRune(ch)
		l.advance()
	}
	return Token{Kind: TkString, Literal: buf.String(), Line: startLine, Col: startCol}
}

func (l *Lexer) readDollar() Token {
	startLine, startCol := l.line, l.col
	l.advance()

	if l.pos < len(l.src) && l.src[l.pos] == '{' {
		l.advance()
		var name strings.Builder
		for l.pos < len(l.src) && l.src[l.pos] != '}' {
			name.WriteRune(l.src[l.pos])
			l.advance()
		}
		if l.pos < len(l.src) {
			l.advance()
		}
		return Token{Kind: TkDollarLBrace, Literal: strings.TrimSpace(name.String()), Line: startLine, Col: startCol}
	}

	if l.pos < len(l.src) && l.src[l.pos] == '(' {
		l.advance()
		depth := 1
		var cmd strings.Builder
		for l.pos < len(l.src) && depth > 0 {
			ch := l.src[l.pos]
			if ch == '(' {
				depth++
			} else if ch == ')' {
				depth--
				if depth == 0 {
					l.advance()
					break
				}
			}
			cmd.WriteRune(ch)
			l.advance()
		}
		return Token{Kind: TkDollarLParen, Literal: strings.TrimSpace(cmd.String()), Line: startLine, Col: startCol}
	}

	if l.pos < len(l.src) && l.src[l.pos] == '?' {
		tok := Token{Kind: TkDollarLBrace, Literal: "?", Line: startLine, Col: startCol}
		l.advance()
		return tok
	}

	var name strings.Builder
	for l.pos < len(l.src) && (unicode.IsLetter(l.src[l.pos]) || unicode.IsDigit(l.src[l.pos]) || l.src[l.pos] == '_') {
		name.WriteRune(l.src[l.pos])
		l.advance()
	}
	n := name.String()
	if n == "" {
		return Token{Kind: TkDollar, Literal: "$", Line: startLine, Col: startCol}
	}
	return Token{Kind: TkDollarLBrace, Literal: n, Line: startLine, Col: startCol}
}

func (l *Lexer) readComparison() Token {
	ch := l.src[l.pos]
	startLine, startCol := l.line, l.col
	l.advance()

	if ch == '=' {
		if l.pos < len(l.src) && l.src[l.pos] == '=' {
			l.advance()
			return Token{Kind: TkEq, Literal: "==", Line: startLine, Col: startCol}
		}
		return Token{Kind: TkAssign, Literal: "=", Line: startLine, Col: startCol}
	}

	if ch == '!' {
		if l.pos < len(l.src) && l.src[l.pos] == '=' {
			l.advance()
			return Token{Kind: TkNeq, Literal: "!=", Line: startLine, Col: startCol}
		}
		return Token{Kind: TkIdent, Literal: "!", Line: startLine, Col: startCol}
	}

	if ch == '>' {
		if l.pos < len(l.src) && l.src[l.pos] == '=' {
			l.advance()
			return Token{Kind: TkGte, Literal: ">=", Line: startLine, Col: startCol}
		}
		return Token{Kind: TkGt, Literal: ">", Line: startLine, Col: startCol}
	}

	if ch == '<' {
		if l.pos < len(l.src) && l.src[l.pos] == '=' {
			l.advance()
			return Token{Kind: TkLte, Literal: "<=", Line: startLine, Col: startCol}
		}
		return Token{Kind: TkLt, Literal: "<", Line: startLine, Col: startCol}
	}

	return Token{Kind: TkIdent, Literal: string(ch), Line: startLine, Col: startCol}
}

func (l *Lexer) readOperator() Token {
	ch := l.src[l.pos]
	startLine, startCol := l.line, l.col
	l.advance()
	switch ch {
	case '+':
		return Token{Kind: TkPlus, Literal: "+", Line: startLine, Col: startCol}
	case '-':
		return Token{Kind: TkMinus, Literal: "-", Line: startLine, Col: startCol}
	case '*':
		return Token{Kind: TkStar, Literal: "*", Line: startLine, Col: startCol}
	case '/':
		return Token{Kind: TkSlash, Literal: "/", Line: startLine, Col: startCol}
	}
	return Token{Kind: TkIdent, Literal: string(ch), Line: startLine, Col: startCol}
}

func (l *Lexer) readHex() Token {
	startLine, startCol := l.line, l.col
	l.advance()
	l.advance()
	var buf strings.Builder
	buf.WriteString("0x")
	for l.pos < len(l.src) && (unicode.IsDigit(l.src[l.pos]) || (l.src[l.pos] >= 'a' && l.src[l.pos] <= 'f') || (l.src[l.pos] >= 'A' && l.src[l.pos] <= 'F')) {
		buf.WriteRune(l.src[l.pos])
		l.advance()
	}
	return Token{Kind: TkInt, Literal: buf.String(), Line: startLine, Col: startCol}
}

func (l *Lexer) readNumber() Token {
	startLine, startCol := l.line, l.col
	var buf strings.Builder
	for l.pos < len(l.src) && unicode.IsDigit(l.src[l.pos]) {
		buf.WriteRune(l.src[l.pos])
		l.advance()
	}
	if l.pos < len(l.src) && l.src[l.pos] == '.' {
		buf.WriteRune('.')
		l.advance()
		for l.pos < len(l.src) && unicode.IsDigit(l.src[l.pos]) {
			buf.WriteRune(l.src[l.pos])
			l.advance()
		}
		return Token{Kind: TkFloat, Literal: buf.String(), Line: startLine, Col: startCol}
	}
	return Token{Kind: TkInt, Literal: buf.String(), Line: startLine, Col: startCol}
}

func (l *Lexer) readIdent() Token {
	startLine, startCol := l.line, l.col
	var buf strings.Builder
	for l.pos < len(l.src) && (unicode.IsLetter(l.src[l.pos]) || unicode.IsDigit(l.src[l.pos]) || l.src[l.pos] == '_') {
		buf.WriteRune(l.src[l.pos])
		l.advance()
	}
	word := buf.String()
	return Token{Kind: TkIdent, Literal: word, Line: startLine, Col: startCol}
}

func (l *Lexer) Tokenize() ([]Token, error) {
	var tokens []Token
	prevNewline := false

	for {
		tok := l.Next()
		if tok.Kind == TkEOF {
			break
		}
		if tok.Kind == TkNewline {
			if prevNewline {
				continue
			}
			prevNewline = true
			tokens = append(tokens, tok)
			continue
		}
		prevNewline = false

		if tok.Kind == TkShebang && len(tok.Literal) >= 10 && tok.Literal[:10] == "#!compact " {
			tok.Kind = TkCompactConfig
		}

		tokens = append(tokens, tok)
	}

	return tokens, nil
}

func Tokenize(src string) ([]Token, error) {
	return New(src).Tokenize()
}

var ErrUnexpectedEOF = fmt.Errorf("unexpected end of file")
var ErrUnexpectedToken = func(tok Token, expected string) error {
	return fmt.Errorf("unexpected token %s at %d:%d, expected %s", tok.Kind, tok.Line, tok.Col, expected)
}
