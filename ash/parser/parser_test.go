package parser

import (
	"testing"

	"github.com/agentic-coding/ash/ast"
	"github.com/agentic-coding/ash/lexer"
)

func parseStr(t *testing.T, src string) *ast.Script {
	tokens, err := lexer.Tokenize(src)
	if err != nil {
		t.Fatalf("tokenize error: %v", err)
	}
	script, err := Parse(tokens)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	return script
}

func TestEmptyScript(t *testing.T) {
	script := parseStr(t, "")
	if script == nil {
		t.Fatal("expected non-nil script")
	}
	if len(script.Statements) != 0 {
		t.Errorf("expected 0 statements, got %d", len(script.Statements))
	}
}

func TestShebang(t *testing.T) {
	script := parseStr(t, "#!opencode:1.2.0:sonnet")
	if script.Shebang == nil {
		t.Fatal("expected shebang")
	}
	if script.Shebang.Engine != "opencode" || script.Shebang.Version != "1.2.0" || script.Shebang.Model != "sonnet" {
		t.Errorf("shebang mismatch: %+v", script.Shebang)
	}
}

func TestCompactConfig(t *testing.T) {
	script := parseStr(t, "#!compact mode=on window=64000\nprint \"hello\"")
	if script.Compact == nil {
		t.Fatal("expected compact config")
	}
	if script.Compact.Mode != "on" || script.Compact.Window != "64000" {
		t.Errorf("compact config: mode=%q window=%q", script.Compact.Mode, script.Compact.Window)
	}
}

func TestVarAssignString(t *testing.T) {
	script := parseStr(t, `MSG = "hello"`)
	if len(script.Statements) != 1 {
		t.Fatalf("expected 1 statement, got %d", len(script.Statements))
	}
	va, ok := script.Statements[0].(*ast.VarAssign)
	if !ok {
		t.Fatalf("expected VarAssign, got %T", script.Statements[0])
	}
	if va.Name != "MSG" {
		t.Errorf("var name = %q, want %q", va.Name, "MSG")
	}
	str, ok := va.Value.(*ast.StringLiteral)
	if !ok {
		t.Fatalf("expected StringLiteral, got %T", va.Value)
	}
	if str.Value != "hello" {
		t.Errorf("string value = %q, want %q", str.Value, "hello")
	}
}

func TestVarAssignVarRef(t *testing.T) {
	script := parseStr(t, `MSG = $NAME`)
	if len(script.Statements) != 1 {
		t.Fatalf("expected 1 statement, got %d", len(script.Statements))
	}
	va := script.Statements[0].(*ast.VarAssign)
	vr, ok := va.Value.(*ast.VarRef)
	if !ok {
		t.Fatalf("expected VarRef, got %T", va.Value)
	}
	if vr.Name != "NAME" {
		t.Errorf("var ref name = %q, want %q", vr.Name, "NAME")
	}
}

func TestVarAssignBareIdent(t *testing.T) {
	script := parseStr(t, `MSG = NAME`)
	if len(script.Statements) != 1 {
		t.Fatalf("expected 1 statement, got %d", len(script.Statements))
	}
	va := script.Statements[0].(*ast.VarAssign)
	vr, ok := va.Value.(*ast.VarRef)
	if !ok {
		t.Fatalf("expected VarRef, got %T", va.Value)
	}
	if vr.Name != "NAME" {
		t.Errorf("var ref name = %q, want %q", vr.Name, "NAME")
	}
}

func TestExpr(t *testing.T) {
	tests := []struct {
		src string
	}{
		{`$? == 0`},
		{`$COUNT > 5`},
		{`$OK and $READY`},
		{`not $FAILED`},
		{`$TOTAL + 1`},
		{`$N * 2`},
		{`($X > 0 and $Y < 10)`},
	}
	for _, tt := range tests {
		script := parseStr(t, tt.src)
		if len(script.Statements) != 1 {
			t.Errorf("%q: expected 1 statement, got %d", tt.src, len(script.Statements))
		}
	}
}

func TestExprBareIdent(t *testing.T) {
	tests := []struct {
		src string
	}{
		{`? == 0`},
		{`COUNT > 5`},
		{`OK and READY`},
		{`not FAILED`},
		{`TOTAL + 1`},
		{`N * 2`},
		{`(X > 0 and Y < 10)`},
	}
	for _, tt := range tests {
		script := parseStr(t, tt.src)
		if len(script.Statements) != 1 {
			t.Errorf("%q: expected 1 statement, got %d", tt.src, len(script.Statements))
		}
	}
}

func TestFullExample(t *testing.T) {
	src := `#!opencode:1.2.0
#!compact mode=on strategy=truncate window=64000

MSG = "hello"
print MSG
`
	script := parseStr(t, src)
	if script.Shebang == nil {
		t.Errorf("expected shebang")
	}
	if script.Compact == nil {
		t.Errorf("expected compact config")
	}
	if len(script.Statements) != 2 {
		t.Errorf("expected 2 statements, got %d", len(script.Statements))
	}
}

func TestParseErrors(t *testing.T) {
	tests := []string{
		`if { }`,          // missing condition
		`for { }`,         // missing var + in
		`do "prompt" with`, // missing subagent
		`try { } upto`,    // missing number
	}
	for _, src := range tests {
		tokens, err := lexer.Tokenize(src)
		if err != nil {
			t.Fatalf("tokenize error for %q: %v", src, err)
		}
		_, err = Parse(tokens)
		if err == nil {
			t.Errorf("expected parse error for %q", src)
		}
	}
}
