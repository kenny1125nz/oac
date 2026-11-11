package parser

import (
	"fmt"
	"strconv"

	"github.com/agentic-coding/ash/ast"
	"github.com/agentic-coding/ash/lexer"
)

type Parser struct {
	tokens []lexer.Token
	pos    int
}

func New(tokens []lexer.Token) *Parser {
	return &Parser{tokens: tokens, pos: 0}
}

func (p *Parser) current() lexer.Token {
	if p.pos < len(p.tokens) {
		return p.tokens[p.pos]
	}
	return lexer.Token{Kind: lexer.TkEOF, Line: -1, Col: -1}
}

func (p *Parser) peek() lexer.Token {
	if p.pos+1 < len(p.tokens) {
		return p.tokens[p.pos+1]
	}
	return lexer.Token{Kind: lexer.TkEOF, Line: -1, Col: -1}
}

func (p *Parser) advance() lexer.Token {
	tok := p.current()
	if p.pos < len(p.tokens) {
		p.pos++
	}
	return tok
}

func (p *Parser) skipNewlines() {
	for p.current().Kind == lexer.TkNewline {
		p.pos++
	}
}

func (p *Parser) expect(kind lexer.TokenKind) (lexer.Token, error) {
	tok := p.current()
	if tok.Kind != kind {
		return tok, fmt.Errorf("expected %s at %d:%d, got %s(%q)", kind, tok.Line, tok.Col, tok.Kind, tok.Literal)
	}
	p.pos++
	return tok, nil
}

func (p *Parser) ParseScript() (*ast.Script, error) {
	script := &ast.Script{
		Position: ast.Pos{Line: 1, Col: 1},
	}
	script.Statements = []ast.Node{}
	p.skipNewlines()

	if p.current().Kind == lexer.TkShebang {
		sh, err := p.parseShebang()
		if err != nil {
			return nil, err
		}
		script.Shebang = sh
		p.skipNewlines()
	}

	if p.current().Kind == lexer.TkCompactConfig {
		tok := p.advance()
		parsed := lexer.ParseCompactLine(tok.Literal)
		cc := ast.CompactConfig{
			Position: ast.Pos{Line: tok.Line, Col: tok.Col},
			Mode:     parsed.Mode,
			Window:   parsed.Window,
			Strategy: parsed.Strategy,
		}
		if parsed != nil {
			cc.Raw = parsed.Raw
		}
		script.Compact = &cc
		p.skipNewlines()
	}

	for p.current().Kind != lexer.TkEOF {
		stmt, err := p.parseStatement()
		if err != nil {
			return nil, err
		}
		if stmt != nil {
			if p.current().Kind == lexer.TkAmpersand {
				p.advance()
				stmt = &ast.Background{
					Position: stmt.Pos(),
					Stmt:     stmt,
				}
			}
			script.Statements = append(script.Statements, stmt)
		}
		p.skipNewlines()

		if p.current().Kind == lexer.TkNewline {
			p.pos++
			p.skipNewlines()
		}
	}

	return script, nil
}

func (p *Parser) parseShebang() (*ast.ShebangDecl, error) {
	tok := p.advance()
	sh, err := lexer.ParseShebang(tok.Literal)
	if err != nil {
		return nil, fmt.Errorf("at %d:%d: %w", tok.Line, tok.Col, err)
	}
	return &ast.ShebangDecl{
		Position: ast.Pos{Line: tok.Line, Col: tok.Col},
		Engine:   sh.Engine,
		Version:  sh.Version,
		Model:    sh.Model,
	}, nil
}

func (p *Parser) parseStatement() (ast.Node, error) {
	tok := p.current()

	switch tok.Kind {
	case lexer.TkIdent:
		switch tok.Literal {
		case "if":
			return p.parseIf()
		case "for":
			return p.parseFor()
		case "while":
			return p.parseWhile()
		case "fn":
			return p.parseFnDecl()
		case "do":
			return p.parseDo()
		case "try":
			return p.parseTry()
		case "within":
			return p.parseWithin()
		case "wait":
			return p.parseWait()
		case "exec":
			return p.parseExec()
		case "print":
			return p.parsePrint()
		case "exit":
			return p.parseExit()
		case "env":
			return p.parseEnv()
		case "include":
			return p.parseInclude()
		case "compact":
			return p.parseCompact()
		case "return":
			return p.parseReturn()
		case "break":
			return p.parseBreak()
		case "continue":
			return p.parseContinue()
		default:
			if p.peek().Kind == lexer.TkAssign {
				return p.parseVarAssign()
			}
			if p.peek().Kind == lexer.TkLParen {
				return p.parseFnCall()
			}
			return p.parseExpr()
		}
	case lexer.TkDollarLBrace, lexer.TkDollarLParen, lexer.TkDollar,
		lexer.TkString, lexer.TkTextBlock, lexer.TkInt, lexer.TkFloat,
		lexer.TkLParen:
		return p.parseExpr()
	case lexer.TkLBrace:
		return p.parseBlock()
	default:
		return p.parseExpr()
	}
}

func (p *Parser) parseExpr() (ast.Node, error) {
	left, err := p.parseBinaryExpr(0)
	if err != nil {
		return nil, err
	}

	// Handle postfix index access: arr[0], fn()[i]
	for p.current().Kind == lexer.TkLBracket {
		p.advance()
		idx, idxErr := p.parseBinaryExpr(0)
		if idxErr != nil {
			return nil, idxErr
		}
		if _, err := p.expect(lexer.TkRBracket); err != nil {
			return nil, err
		}
		left = &ast.IndexExpr{
			Position: left.Pos(),
			Object:   left,
			Index:    idx,
		}
	}

	return left, nil
}

var precedence = map[string]int{
	"or":  1,
	"and": 2,
	"==":  3, "!=": 3, ">": 3, "<": 3, ">=": 3, "<=": 3,
	"+": 4, "-": 4,
	"*": 5, "/": 5,
}

func (p *Parser) parseBinaryExpr(minPrec int) (ast.Node, error) {
	left, err := p.parsePrimary()
	if err != nil {
		return nil, err
	}

	for {
		op := ""
		tok := p.current()
		switch tok.Kind {
		case lexer.TkEq:
			op = "=="
			p.pos++
		case lexer.TkNeq:
			op = "!="
			p.pos++
		case lexer.TkGt:
			op = ">"
			p.pos++
		case lexer.TkLt:
			op = "<"
			p.pos++
		case lexer.TkGte:
			op = ">="
			p.pos++
		case lexer.TkLte:
			op = "<="
			p.pos++
		case lexer.TkPlus:
			op = "+"
			p.pos++
		case lexer.TkMinus:
			op = "-"
			p.pos++
		case lexer.TkStar:
			op = "*"
			p.pos++
		case lexer.TkSlash:
			op = "/"
			p.pos++
		case lexer.TkIdent:
			if tok.Literal == "and" || tok.Literal == "or" {
				op = tok.Literal
				p.pos++
			}
		}

		if op == "" {
			break
		}

		prec := precedence[op]
		if prec < minPrec {
			p.pos--
			break
		}

		right, err := p.parseBinaryExpr(prec + 1)
		if err != nil {
			return nil, err
		}
		left = &ast.BinaryExpr{
			Position: left.Pos(),
			Left:     left,
			Right:    right,
			Op:       op,
		}
	}

	return left, nil
}

func (p *Parser) parsePrimary() (ast.Node, error) {
	tok := p.current()

	if tok.Kind == lexer.TkLBracket {
		return p.parseArrayLiteral()
	}

	if tok.Kind == lexer.TkLParen {
		p.pos++
		expr, err := p.parseBinaryExpr(0)
		if err != nil {
			return nil, err
		}
		if _, err := p.expect(lexer.TkRParen); err != nil {
			return nil, err
		}
		return &ast.GroupExpr{Position: ast.Pos{Line: tok.Line, Col: tok.Col}, Inner: expr}, nil
	}

	if tok.Kind == lexer.TkIdent && tok.Literal == "not" {
		p.pos++
		right, err := p.parsePrimary()
		if err != nil {
			return nil, err
		}
		return &ast.UnaryExpr{Position: ast.Pos{Line: tok.Line, Col: tok.Col}, Op: "not", Right: right}, nil
	}

	if tok.Kind == lexer.TkMinus {
		p.pos++
		right, err := p.parsePrimary()
		if err != nil {
			return nil, err
		}
		return &ast.UnaryExpr{Position: ast.Pos{Line: tok.Line, Col: tok.Col}, Op: "-", Right: right}, nil
	}

	if tok.Kind == lexer.TkString {
		p.pos++
		return &ast.StringLiteral{
			Position: ast.Pos{Line: tok.Line, Col: tok.Col},
			Value:    tok.Literal,
		}, nil
	}

	if tok.Kind == lexer.TkTextBlock {
		p.pos++
		return &ast.TextBlock{
			Position: ast.Pos{Line: tok.Line, Col: tok.Col},
			Value:    tok.Literal,
		}, nil
	}

	if tok.Kind == lexer.TkInt {
		p.pos++
		val, _ := strconv.ParseInt(tok.Literal, 0, 64)
		return &ast.IntLiteral{
			Position: ast.Pos{Line: tok.Line, Col: tok.Col},
			Value:    val,
			Text:     tok.Literal,
		}, nil
	}

	if tok.Kind == lexer.TkFloat {
		p.pos++
		val, _ := strconv.ParseFloat(tok.Literal, 64)
		return &ast.FloatLiteral{
			Position: ast.Pos{Line: tok.Line, Col: tok.Col},
			Value:    val,
			Text:     tok.Literal,
		}, nil
	}

	if tok.Kind == lexer.TkDollarLBrace {
		p.pos++
		return &ast.VarRef{
			Position: ast.Pos{Line: tok.Line, Col: tok.Col},
			Name:     tok.Literal,
		}, nil
	}

	if tok.Kind == lexer.TkDollarLParen {
		p.pos++
		return &ast.CommandSubst{
			Position: ast.Pos{Line: tok.Line, Col: tok.Col},
			Cmd:      tok.Literal,
		}, nil
	}

	if tok.Kind == lexer.TkAt {
		p.advance()
		path, err := p.parsePrimary()
		if err != nil {
			return nil, err
		}
		return &ast.FilePath{Position: ast.Pos{Line: tok.Line, Col: tok.Col}, Path: path}, nil
	}

	if tok.Kind == lexer.TkIdent && tok.Literal == "true" {
		p.pos++
		return &ast.BoolLiteral{Position: ast.Pos{Line: tok.Line, Col: tok.Col}, Value: true}, nil
	}
	if tok.Kind == lexer.TkIdent && tok.Literal == "false" {
		p.pos++
		return &ast.BoolLiteral{Position: ast.Pos{Line: tok.Line, Col: tok.Col}, Value: false}, nil
	}

	if tok.Kind == lexer.TkIdent && p.peek().Kind == lexer.TkLParen {
		return p.parseFnCall()
	}

	if tok.Kind == lexer.TkIdent {
		if tok.Literal == "and" || tok.Literal == "or" {
			return nil, fmt.Errorf("unexpected keyword %q in expression at %d:%d", tok.Literal, tok.Line, tok.Col)
		}
		p.pos++
		return &ast.VarRef{
			Position: ast.Pos{Line: tok.Line, Col: tok.Col},
			Name:     tok.Literal,
		}, nil
	}

	return nil, fmt.Errorf("unexpected token %s(%q) at %d:%d", tok.Kind, tok.Literal, tok.Line, tok.Col)
}

func (p *Parser) parseBlock() (*ast.Block, error) {
	openTok, err := p.expect(lexer.TkLBrace)
	if err != nil {
		return nil, err
	}
	block := &ast.Block{Position: ast.Pos{Line: openTok.Line, Col: openTok.Col}}
	p.skipNewlines()

	for p.current().Kind != lexer.TkRBrace && p.current().Kind != lexer.TkEOF {
		stmt, err := p.parseStatement()
		if err != nil {
			return nil, err
		}
		if stmt != nil {
			block.Statements = append(block.Statements, stmt)
		}
		p.skipNewlines()
	}

	if _, err := p.expect(lexer.TkRBrace); err != nil {
		return nil, err
	}
	return block, nil
}

func (p *Parser) parseVarAssign() (ast.Node, error) {
	name := p.advance()
	if _, err := p.expect(lexer.TkAssign); err != nil {
		return nil, err
	}
	val, err := p.parseExpr()
	if err != nil {
		return nil, err
	}
	return &ast.VarAssign{
		Position: ast.Pos{Line: name.Line, Col: name.Col},
		Name:     name.Literal,
		Value:    val,
	}, nil
}

func (p *Parser) parseIf() (ast.Node, error) {
	tok := p.advance()
	p.skipNewlines()

	cond, err := p.parseBinaryExpr(0)
	if err != nil {
		return nil, err
	}

	p.skipNewlines()
	body, err := p.parseBlock()
	if err != nil {
		return nil, err
	}

	n := &ast.IfStmt{
		Position: ast.Pos{Line: tok.Line, Col: tok.Col},
		Cond:     cond,
		Body:     body,
	}

	for p.current().Kind == lexer.TkIdent && p.current().Literal == "else" {
		p.advance()
		if p.current().Kind == lexer.TkIdent && p.current().Literal == "if" {
			p.advance()
			p.skipNewlines()
			eiCond, err := p.parseBinaryExpr(0)
			if err != nil {
				return nil, err
			}
			p.skipNewlines()
			eiBody, err := p.parseBlock()
			if err != nil {
				return nil, err
			}
			n.ElseIfs = append(n.ElseIfs, ast.ElseIf{
				Position: ast.Pos{Line: tok.Line, Col: tok.Col},
				Cond:     eiCond,
				Body:     eiBody,
			})
		} else {
			p.skipNewlines()
			elseBody, err := p.parseBlock()
			if err != nil {
				return nil, err
			}
			n.Else = elseBody
			break
		}
	}

	return n, nil
}

func (p *Parser) parseFor() (ast.Node, error) {
	tok := p.advance()
	p.skipNewlines()

	varTok, err := p.expect(lexer.TkIdent)
	if err != nil {
		return nil, err
	}

	inTok, err := p.expect(lexer.TkIdent)
	if err != nil {
		return nil, err
	}
	if inTok.Literal != "in" {
		return nil, fmt.Errorf("expected 'in' at %d:%d", varTok.Line, varTok.Col)
	}

	list, err := p.parseBinaryExpr(0)
	if err != nil {
		return nil, err
	}

	p.skipNewlines()
	body, err := p.parseBlock()
	if err != nil {
		return nil, err
	}

	return &ast.ForStmt{
		Position: ast.Pos{Line: tok.Line, Col: tok.Col},
		Var:      varTok.Literal,
		List:     list,
		Body:     body,
	}, nil
}

func (p *Parser) parseWhile() (ast.Node, error) {
	tok := p.advance()
	p.skipNewlines()

	cond, err := p.parseBinaryExpr(0)
	if err != nil {
		return nil, err
	}

	p.skipNewlines()
	body, err := p.parseBlock()
	if err != nil {
		return nil, err
	}

	return &ast.WhileStmt{
		Position: ast.Pos{Line: tok.Line, Col: tok.Col},
		Cond:     cond,
		Body:     body,
	}, nil
}

func (p *Parser) parseFnDecl() (ast.Node, error) {
	tok := p.advance()
	p.skipNewlines()

	nameTok, err := p.expect(lexer.TkIdent)
	if err != nil {
		return nil, err
	}

	if _, err := p.expect(lexer.TkLParen); err != nil {
		return nil, err
	}

	var params []string
	if p.current().Kind == lexer.TkIdent {
		params = append(params, p.advance().Literal)
		for p.current().Kind == lexer.TkComma {
			p.advance()
			paramTok, err := p.expect(lexer.TkIdent)
			if err != nil {
				return nil, err
			}
			params = append(params, paramTok.Literal)
		}
	}

	if _, err := p.expect(lexer.TkRParen); err != nil {
		return nil, err
	}

	p.skipNewlines()
	body, err := p.parseBlock()
	if err != nil {
		return nil, err
	}

	return &ast.FnDecl{
		Position: ast.Pos{Line: tok.Line, Col: tok.Col},
		Name:     nameTok.Literal,
		Params:   params,
		Body:     body,
	}, nil
}

func (p *Parser) parseFnCall() (ast.Node, error) {
	nameTok := p.advance()
	return p.parseFnCallArgs(nameTok)
}

func (p *Parser) parseFnCallArgs(nameTok lexer.Token) (ast.Node, error) {
	if _, err := p.expect(lexer.TkLParen); err != nil {
		return nil, err
	}

	var args []ast.Node
	if p.current().Kind != lexer.TkRParen {
		arg, err := p.parseBinaryExpr(0)
		if err != nil {
			return nil, err
		}
		args = append(args, arg)
		for p.current().Kind == lexer.TkComma {
			p.advance()
			arg, err := p.parseBinaryExpr(0)
			if err != nil {
				return nil, err
			}
			args = append(args, arg)
		}
	}

	if _, err := p.expect(lexer.TkRParen); err != nil {
		return nil, err
	}

	return &ast.FnCall{
		Position: ast.Pos{Line: nameTok.Line, Col: nameTok.Col},
		Name:     nameTok.Literal,
		Args:     args,
	}, nil
}

func (p *Parser) parseDo() (ast.Node, error) {
	tok := p.advance()
	p.skipNewlines()

	prompt, err := p.parseExpr()
	if err != nil {
		return nil, err
	}

	n := &ast.AgentCall{
		Position: ast.Pos{Line: tok.Line, Col: tok.Col},
		Prompt:   prompt,
		Subagent: "default",
	}

	// "with subagent <name>" is optional — defaults to "default"
	if p.current().Kind == lexer.TkIdent && p.current().Literal == "with" {
		p.advance()

		subTokCheck, err := p.expect(lexer.TkIdent)
		if err != nil {
			return nil, err
		}
		if subTokCheck.Literal != "subagent" {
			return nil, fmt.Errorf("expected 'subagent' after 'with' at %d:%d", tok.Line, tok.Col)
		}

		subName := ""
		for {
			subTok, err := p.expect(lexer.TkIdent)
			if err != nil {
				return nil, err
			}
			subName += subTok.Literal
			if p.current().Kind == lexer.TkMinus && p.peek().Kind == lexer.TkIdent {
				subName += "-"
				p.advance()
			} else {
				break
			}
		}
		n.Subagent = subName
	}

	if p.current().Kind == lexer.TkIdent && p.current().Literal == "using" {
		p.advance()
		model, err := p.parsePrimary()
		if err != nil {
			return nil, err
		}
		n.Model = model
	}

	if p.current().Kind == lexer.TkIdent && p.current().Literal == "in" {
		p.advance()
		dir, err := p.parseExpr()
		if err != nil {
			return nil, err
		}
		n.Dir = dir
	}

	if p.current().Kind == lexer.TkIdent && p.current().Literal == "compact" {
		p.advance()
		compactArg, err := p.parsePrimary()
		if err != nil {
			return nil, err
		}
		n.Compact = compactArg
	}

	return n, nil
}

func (p *Parser) parseTry() (ast.Node, error) {
	tok := p.advance()
	p.skipNewlines()

	body, err := p.parseBlock()
	if err != nil {
		return nil, err
	}
	p.skipNewlines()

	// Check for "evaluate with"
	evalMode := false
	if p.current().Kind == lexer.TkIdent && p.current().Literal == "evaluate" {
		p.advance()
		withTok, err := p.expect(lexer.TkIdent)
		if err != nil {
			return nil, err
		}
		if withTok.Literal != "with" {
			return nil, fmt.Errorf("expected 'with' at %d:%d", tok.Line, tok.Col)
		}
		p.skipNewlines()
		evalMode = true
	}

	if evalMode {
		evalBody, err := p.parseBlock()
		if err != nil {
			return nil, err
		}
		p.skipNewlines()

		var acceptBlock, partialBlock, failBlock *ast.Block

		if p.current().Kind == lexer.TkIdent && p.current().Literal == "accept" {
			p.advance()
			p.skipNewlines()
			ab, err := p.parseBlock()
			if err != nil {
				return nil, err
			}
			acceptBlock = ab
			p.skipNewlines()
		}

		if p.current().Kind == lexer.TkIdent && p.current().Literal == "partial" {
			p.advance()
			p.skipNewlines()
			pb, err := p.parseBlock()
			if err != nil {
				return nil, err
			}
			partialBlock = pb
			p.skipNewlines()
		}

		if p.current().Kind == lexer.TkIdent && p.current().Literal == "fail" {
			p.advance()
			p.skipNewlines()
			fb, err := p.parseBlock()
			if err != nil {
				return nil, err
			}
			failBlock = fb
			p.skipNewlines()
		}

		uptoTok, err := p.expect(lexer.TkIdent)
		if err != nil {
			return nil, err
		}
		if uptoTok.Literal != "upto" {
			return nil, fmt.Errorf("expected 'upto' at %d:%d", tok.Line, tok.Col+1)
		}

		maxTok, err := p.expect(lexer.TkInt)
		if err != nil {
			return nil, err
		}

		maxVal := 1
		fmt.Sscanf(maxTok.Literal, "%d", &maxVal)

		return &ast.EvalTry{
			Position: ast.Pos{Line: tok.Line, Col: tok.Col},
			Body:     body,
			Eval:     evalBody,
			Accept:   acceptBlock,
			Partial:  partialBlock,
			Fail:     failBlock,
			Max:      maxVal,
		}, nil
	}

	// Binary try: try { } fail { } upto N
	var failBlock *ast.Block
	if p.current().Kind == lexer.TkIdent && p.current().Literal == "fail" {
		p.advance()
		p.skipNewlines()
		fb, err := p.parseBlock()
		if err != nil {
			return nil, err
		}
		failBlock = fb
		p.skipNewlines()
	}

	uptoTok, err := p.expect(lexer.TkIdent)
	if err != nil {
		return nil, err
	}
	if uptoTok.Literal != "upto" {
		return nil, fmt.Errorf("expected 'upto' at %d:%d", tok.Line, tok.Col+1)
	}

	maxTok, err := p.expect(lexer.TkInt)
	if err != nil {
		return nil, err
	}

	maxVal := 1
	fmt.Sscanf(maxTok.Literal, "%d", &maxVal)

	return &ast.BinaryTry{
		Position: ast.Pos{Line: tok.Line, Col: tok.Col},
		Body:     body,
		Fail:     failBlock,
		Max:      maxVal,
	}, nil
}

func (p *Parser) parseWithin() (ast.Node, error) {
	tok := p.advance()
	p.skipNewlines()

	dir, err := p.parseExpr()
	if err != nil {
		return nil, err
	}

	p.skipNewlines()
	body, err := p.parseBlock()
	if err != nil {
		return nil, err
	}

	return &ast.DirBlock{
		Position: ast.Pos{Line: tok.Line, Col: tok.Col},
		Dir:      dir,
		Body:     body,
	}, nil
}

func (p *Parser) parseWait() (ast.Node, error) {
	tok := p.advance()
	p.skipNewlines()

	var body *ast.Block
	if p.current().Kind == lexer.TkLBrace {
		b, err := p.parseBlock()
		if err != nil {
			return nil, err
		}
		body = b
	}

	return &ast.WaitBlock{
		Position: ast.Pos{Line: tok.Line, Col: tok.Col},
		Body:     body,
	}, nil
}

func (p *Parser) parseArrayLiteral() (ast.Node, error) {
	tok := p.advance()
	var elements []ast.Node
	p.skipNewlines()

	if p.current().Kind != lexer.TkRBracket {
		expr, err := p.parseBinaryExpr(0)
		if err != nil {
			return nil, err
		}
		elements = append(elements, expr)
		p.skipNewlines()

		for p.current().Kind == lexer.TkComma {
			p.advance()
			p.skipNewlines()
			if p.current().Kind == lexer.TkRBracket {
				break
			}
			expr, err := p.parseBinaryExpr(0)
			if err != nil {
				return nil, err
			}
			elements = append(elements, expr)
			p.skipNewlines()
		}
	}

	if _, err := p.expect(lexer.TkRBracket); err != nil {
		return nil, err
	}
	return &ast.ArrayLiteral{
		Position: ast.Pos{Line: tok.Line, Col: tok.Col},
		Elements: elements,
	}, nil
}

func (p *Parser) parseBackground() (ast.Node, error) {
	tok := p.advance()
	p.skipNewlines()

	stmt, err := p.parseStatement()
	if err != nil {
		return nil, err
	}

	return &ast.Background{
		Position: ast.Pos{Line: tok.Line, Col: tok.Col},
		Stmt:     stmt,
	}, nil
}

func (p *Parser) parseExec() (ast.Node, error) {
	tok := p.advance()
	p.skipNewlines()

	cmd, err := p.parseExpr()
	if err != nil {
		return nil, err
	}

	return &ast.Exec{
		Position: ast.Pos{Line: tok.Line, Col: tok.Col},
		Cmd:      cmd,
	}, nil
}

func (p *Parser) parsePrint() (ast.Node, error) {
	tok := p.advance()
	p.skipNewlines()

	msg, err := p.parseExpr()
	if err != nil {
		return nil, err
	}

	return &ast.Print{
		Position: ast.Pos{Line: tok.Line, Col: tok.Col},
		Message:  msg,
	}, nil
}

func (p *Parser) parseExit() (ast.Node, error) {
	tok := p.advance()
	p.skipNewlines()

	code, err := p.parseExpr()
	if err != nil {
		return nil, err
	}

	return &ast.Exit{
		Position: ast.Pos{Line: tok.Line, Col: tok.Col},
		Code:     code,
	}, nil
}

func (p *Parser) parseInclude() (ast.Node, error) {
	tok := p.advance()
	p.skipNewlines()

	path, err := p.parseExpr()
	if err != nil {
		return nil, err
	}

	return &ast.Include{
		Position: ast.Pos{Line: tok.Line, Col: tok.Col},
		Path:     path,
	}, nil
}

func (p *Parser) parseCompact() (ast.Node, error) {
	tok := p.advance()
	p.skipNewlines()

	arg, err := p.parseExpr()
	if err != nil {
		return nil, err
	}

	return &ast.CompactStmt{
		Position: ast.Pos{Line: tok.Line, Col: tok.Col},
		Arg:      arg,
	}, nil
}

func (p *Parser) parseEnv() (ast.Node, error) {
	tok := p.advance()
	keyTok, err := p.expect(lexer.TkIdent)
	if err != nil {
		return nil, err
	}
	return &ast.Env{
		Position: ast.Pos{Line: tok.Line, Col: tok.Col},
		Key:      keyTok.Literal,
	}, nil
}

func (p *Parser) parseReturn() (ast.Node, error) {
	tok := p.advance()
	p.skipNewlines()

	if p.current().Kind == lexer.TkNewline || p.current().Kind == lexer.TkRBrace || p.current().Kind == lexer.TkEOF {
		return &ast.Return{
			Position: ast.Pos{Line: tok.Line, Col: tok.Col},
		}, nil
	}

	val, err := p.parseExpr()
	if err != nil {
		return nil, err
	}

	return &ast.Return{
		Position: ast.Pos{Line: tok.Line, Col: tok.Col},
		Value:    val,
	}, nil
}

func (p *Parser) parseBreak() (ast.Node, error) {
	tok := p.advance()
	return &ast.Break{Position: ast.Pos{Line: tok.Line, Col: tok.Col}}, nil
}

func (p *Parser) parseContinue() (ast.Node, error) {
	tok := p.advance()
	return &ast.Continue{Position: ast.Pos{Line: tok.Line, Col: tok.Col}}, nil
}

func Parse(tokens []lexer.Token) (*ast.Script, error) {
	return New(tokens).ParseScript()
}
