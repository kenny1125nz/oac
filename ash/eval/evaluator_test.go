package eval

import (
	"bytes"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/agentic-coding/ash/ast"
	"github.com/agentic-coding/ash/lexer"
	"github.com/agentic-coding/ash/parser"
)

func parseAndEval(t *testing.T, src string) (*Evaluator, error) {
	tokens, err := lexer.Tokenize(src)
	if err != nil {
		t.Fatalf("tokenize error: %v", err)
	}
	script, err := parser.Parse(tokens)
	if err != nil {
		return nil, err
	}
	e := New()
	output := &bytes.Buffer{}
	e.stdout = output
	err = e.EvalScript(script)
	return e, err
}

func evalExpr(t *testing.T, src string) (Value, error) {
	tokens, err := lexer.Tokenize(src)
	if err != nil {
		t.Fatalf("tokenize error: %v", err)
	}
	script, err := parser.Parse(tokens)
	if err != nil {
		return nil, err
	}
	e := New()
	var result Value
	for _, stmt := range script.Statements {
		val, err := e.evalStatement(stmt)
		if err != nil {
			return nil, err
		}
		result = val
	}
	return result, nil
}

func TestExprArithmetic(t *testing.T) {
	tests := []struct {
		src  string
		want Value
	}{
		{`1 + 2`, int64(3)},
		{`3 - 1`, int64(2)},
		{`2 * 3`, int64(6)},
		{`6 / 2`, float64(3)},
		{`1 + 2 * 3`, int64(7)},
		{`(1 + 2) * 3`, int64(9)},
		{`1.5 + 2.5`, float64(4)},
	}
	for _, tt := range tests {
		result, err := evalExpr(t, tt.src)
		if err != nil {
			t.Errorf("%q: unexpected error: %v", tt.src, err)
			continue
		}
		if result != tt.want {
			t.Errorf("%q: got %v (%T), want %v (%T)", tt.src, result, result, tt.want, tt.want)
		}
	}
}

func TestExprComparison(t *testing.T) {
	tests := []struct {
		src  string
		want Value
	}{
		{`1 == 1`, true},
		{`1 != 2`, true},
		{`2 > 1`, true},
		{`1 < 2`, true},
		{`2 >= 2`, true},
		{`2 <= 2`, true},
		{`1 == 2`, false},
		{`1 > 2`, false},
	}
	for _, tt := range tests {
		result, err := evalExpr(t, tt.src)
		if err != nil {
			t.Errorf("%q: unexpected error: %v", tt.src, err)
			continue
		}
		if result != tt.want {
			t.Errorf("%q: got %v, want %v", tt.src, result, tt.want)
		}
	}
}

func TestExprBoolean(t *testing.T) {
	tests := []struct {
		src  string
		want Value
	}{
		{`true and true`, true},
		{`true and false`, false},
		{`true or false`, true},
		{`false or false`, false},
		{`not true`, false},
		{`not false`, true},
	}
	for _, tt := range tests {
		result, err := evalExpr(t, tt.src)
		if err != nil {
			t.Errorf("%q: unexpected error: %v", tt.src, err)
			continue
		}
		if result != tt.want {
			t.Errorf("%q: got %v, want %v", tt.src, result, tt.want)
		}
	}
}

func TestExprFull(t *testing.T) {
	tests := []struct {
		src  string
		want Value
	}{
		{`$? == 0`, true},
		{`5 > 3`, true},
		{`true and false`, false},
		{`not $?`, true},
		{`($? == 0 and 5 > 3)`, true},
	}
	for _, tt := range tests {
		result, err := evalExpr(t, tt.src)
		if err != nil {
			t.Errorf("%q: unexpected error: %v", tt.src, err)
			continue
		}
		if result != tt.want {
			t.Errorf("%q: got %v, want %v", tt.src, result, tt.want)
		}
	}
}

func TestVarAssignAndRef(t *testing.T) {
	_, err := parseAndEval(t, `X = 42
Y = X`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestPrint(t *testing.T) {
	var buf bytes.Buffer
	e := New()
	e.stdout = &buf

	tokens, _ := lexer.Tokenize(`print "hello"`)
	script, _ := parser.Parse(tokens)
	err := e.EvalScript(script)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := strings.TrimSpace(buf.String())
	if output != "hello" {
		t.Errorf("print output = %q, want %q", output, "hello")
	}
}

func TestSetExitCode(t *testing.T) {
	e := New()
	e.setExitCode(42)
	val, _ := e.current.Get("?")
	if val != int64(42) {
		t.Errorf("$? = %v, want 42", val)
	}
}

func TestExecEcho(t *testing.T) {
	if _, err := parseAndEval(t, `exec echo hello`); err != nil {
		t.Skipf("exec not available: %v", err)
	}
}

func TestStringInterpolation(t *testing.T) {
	e := New()
	e.current.Set("NAME", "world")

	lit := &ast.StringLiteral{Value: "hello ${NAME}"}
	val, err := e.evalExpr(lit)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if val != "hello world" {
		t.Errorf("interpolated = %q, want %q", val, "hello world")
	}
}

func TestStringInterpolationEscape(t *testing.T) {
	e := New()
	e.current.Set("NAME", "world")

	lit := &ast.StringLiteral{Value: "hello \x00{NAME}"}
	val, err := e.evalExpr(lit)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if val != "hello ${NAME}" {
		t.Errorf("interpolated = %q, want %q", val, "hello ${NAME}")
	}
}

func TestEnv(t *testing.T) {
	os.Setenv("ASH_TEST_VAR", "test_value")
	defer os.Unsetenv("ASH_TEST_VAR")

	tokens, _ := lexer.Tokenize(`env ASH_TEST_VAR`)
	script, _ := parser.Parse(tokens)
	e := New()
	err := e.EvalScript(script)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGlobalScope(t *testing.T) {
	e := New()
	val, ok := e.current.Get("?")
	if !ok {
		t.Fatal("expected $? to exist")
	}
	if val != int64(0) {
		t.Errorf("$? = %v, want 0", val)
	}
}

func TestScopeSetGet(t *testing.T) {
	e := New()
	e.pushScope()
	e.current.Set("X", int64(42))
	val, ok := e.current.Get("X")
	if !ok || val != int64(42) {
		t.Errorf("Get(X) = %v, %v, want 42, true", val, ok)
	}
}

func TestScopeShadow(t *testing.T) {
	e := New()
	e.current.Set("X", int64(1))

	e.pushScope()
	e.current.SetLocal("X", int64(2))
	val, _ := e.current.Get("X")
	if val != int64(2) {
		t.Errorf("inner X = %v, want 2", val)
	}
	e.popScope()

	val, _ = e.current.Get("X")
	if val != int64(1) {
		t.Errorf("outer X = %v, want 1", val)
	}
}

func TestScopeSetNearest(t *testing.T) {
	e := New()
	e.pushScope()

	e.current.Set("X", int64(5))
	val, _ := e.current.Get("X")
	if val != int64(5) {
		t.Errorf("X = %v, want 5", val)
	}

	if !e.current.HasLocal("X") {
		t.Errorf("X should be local")
	}
}

func TestReassignNearest(t *testing.T) {
	e := New()
	e.current.Set("Y", int64(10))

	e.pushScope()
	e.current.Set("Y", int64(20))

	val, _ := e.current.Get("Y")
	if val != int64(20) {
		t.Errorf("Y in inner scope = %v, want 20", val)
	}

	e.popScope()
	val, _ = e.current.Get("Y")
	if val != int64(20) {
		t.Errorf("Y in outer scope = %v, want 20 (reassign nearest)", val)
	}
}

func TestReassignLocal(t *testing.T) {
	e := New()
	e.current.Set("Y", int64(10))

	e.pushScope()
	e.current.SetLocal("Y", int64(20))
	e.popScope()

	val, _ := e.current.Get("Y")
	if val != int64(10) {
		t.Errorf("outer Y = %v, want 10 (local set does not touch parent)", val)
	}
}

func TestIfTrue(t *testing.T) {
	var buf bytes.Buffer
	e := New()
	e.stdout = &buf

	tokens, _ := lexer.Tokenize("if true {\nprint \"yes\"\n}")
	script, _ := parser.Parse(tokens)
	err := e.EvalScript(script)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := strings.TrimSpace(buf.String())
	if output != "yes" {
		t.Errorf("if output = %q, want %q", output, "yes")
	}
}

func TestIfElse(t *testing.T) {
	var buf bytes.Buffer
	e := New()
	e.stdout = &buf

	tokens, _ := lexer.Tokenize("if false {\nprint \"a\"\n} else {\nprint \"b\"\n}")
	script, _ := parser.Parse(tokens)
	err := e.EvalScript(script)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := strings.TrimSpace(buf.String())
	if output != "b" {
		t.Errorf("if-else output = %q, want %q", output, "b")
	}
}

func TestIfElseIf(t *testing.T) {
	var buf bytes.Buffer
	e := New()
	e.stdout = &buf

	tokens, _ := lexer.Tokenize("X = 2\nif X == 1 {\n} else if X == 2 {\nprint \"two\"\n} else {\n}")
	script, _ := parser.Parse(tokens)
	err := e.EvalScript(script)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := strings.TrimSpace(buf.String())
	if output != "two" {
		t.Errorf("if-else if output = %q, want %q", output, "two")
	}
}

func TestForLoop(t *testing.T) {
	var buf bytes.Buffer
	e := New()
	e.stdout = &buf

	tokens, _ := lexer.Tokenize("for X in \"a\nb\nc\" {\nprint X\n}")
	script, _ := parser.Parse(tokens)
	err := e.EvalScript(script)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := strings.TrimSpace(buf.String())
	if output != "a\nb\nc" {
		t.Errorf("for output = %q, want %q", output, "a\nb\nc")
	}
}

func TestWhileLoop(t *testing.T) {
	var buf bytes.Buffer
	e := New()
	e.stdout = &buf
	e.current.Set("COUNT", int64(0))

	tokens, _ := lexer.Tokenize("while COUNT < 3 {\nprint COUNT\nCOUNT = COUNT + 1\n}")
	script, _ := parser.Parse(tokens)
	err := e.EvalScript(script)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := strings.TrimSpace(buf.String())
	if output != "0\n1\n2" {
		t.Errorf("while output = %q, want %q", output, "0\n1\n2")
	}
}

func TestBreak(t *testing.T) {
	var buf bytes.Buffer
	e := New()
	e.stdout = &buf

	tokens, _ := lexer.Tokenize("for X in \"a\nb\nc\" {\nif X == \"b\" {\nbreak\n}\nprint X\n}")
	script, _ := parser.Parse(tokens)
	err := e.EvalScript(script)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := strings.TrimSpace(buf.String())
	if output != "a" {
		t.Errorf("break output = %q, want %q", output, "a")
	}
}

func TestContinue(t *testing.T) {
	var buf bytes.Buffer
	e := New()
	e.stdout = &buf

	tokens, _ := lexer.Tokenize("for X in \"a\nb\nc\" {\nif X == \"b\" {\ncontinue\n}\nprint X\n}")
	script, _ := parser.Parse(tokens)
	err := e.EvalScript(script)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := strings.TrimSpace(buf.String())
	if output != "a\nc" {
		t.Errorf("continue output = %q, want %q", output, "a\nc")
	}
}

func TestFnDeclAndCall(t *testing.T) {
	var buf bytes.Buffer
	e := New()
	e.stdout = &buf

	tokens, _ := lexer.Tokenize("fn greet(NAME) {\nprint \"hello \" + NAME\n}\ngreet(\"world\")")
	script, _ := parser.Parse(tokens)
	err := e.EvalScript(script)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := strings.TrimSpace(buf.String())
	if output != "hello world" {
		t.Errorf("fn output = %q, want %q", output, "hello world")
	}
}

func TestFnReturn(t *testing.T) {
	tokens, _ := lexer.Tokenize(`fn add(A, B) { return A + B }
R = add(3, 4)`)
	script, _ := parser.Parse(tokens)
	e := New()
	err := e.EvalScript(script)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	val, ok := e.current.Get("R")
	if !ok {
		t.Fatal("R not set")
	}
	expected := int64(7)
	if val != expected {
		t.Errorf("R = %v, want %v", val, expected)
	}
}

func TestCompact(t *testing.T) {
	var stderr bytes.Buffer
	e := New()
	e.stderr = &stderr
	tokens, _ := lexer.Tokenize(`compact "truncate 32000"`)
	script, _ := parser.Parse(tokens)
	err := e.EvalScript(script)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	out := stderr.String()
	if !strings.Contains(out, "truncate") {
		t.Errorf("compact output = %q, want truncate directive", out)
	}
}

func TestCompactSummarize(t *testing.T) {
	var stderr bytes.Buffer
	e := New()
	e.stderr = &stderr
	tokens, _ := lexer.Tokenize(`compact "summarize"`)
	script, _ := parser.Parse(tokens)
	err := e.EvalScript(script)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	out := stderr.String()
	if !strings.Contains(out, "summarize") {
		t.Errorf("compact output = %q, want summarize directive", out)
	}
}

func TestBackgroundAndWait(t *testing.T) {
	var buf bytes.Buffer
	e := New()
	e.stdout = &buf

	tokens, _ := lexer.Tokenize("{\nprint \"bg\"\n} &\nwait")
	script, _ := parser.Parse(tokens)
	err := e.EvalScript(script)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := strings.TrimSpace(buf.String())
	if output != "bg" {
		t.Errorf("bg+wait output = %q, want %q", output, "bg")
	}
}

func TestBinaryTrySuccess(t *testing.T) {
	var buf bytes.Buffer
	e := New()
	e.stdout = &buf

	tokens, _ := lexer.Tokenize("try {\nprint \"ok\"\n} upto 1")
	script, _ := parser.Parse(tokens)
	err := e.EvalScript(script)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := strings.TrimSpace(buf.String())
	if output != "ok" {
		t.Errorf("try output = %q, want %q", output, "ok")
	}
}

func TestBinaryTryFallback(t *testing.T) {
	var buf bytes.Buffer
	e := New()
	e.stdout = &buf

	tokens, _ := lexer.Tokenize("try {\nexit 1\n} fail {\nprint \"fallback\"\n} upto 1")
	script, _ := parser.Parse(tokens)
	err := e.EvalScript(script)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := strings.TrimSpace(buf.String())
	if output != "fallback" {
		t.Errorf("try fallback output = %q, want %q", output, "fallback")
	}
}

func TestEvalTryAccept(t *testing.T) {
	var buf bytes.Buffer
	e := New()
	e.stdout = &buf

	tokens, _ := lexer.Tokenize("try {\nX = 1\n} evaluate with {\nX == 1\n} accept {\nprint \"accepted\"\n} upto 1")
	script, _ := parser.Parse(tokens)
	err := e.EvalScript(script)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := strings.TrimSpace(buf.String())
	if output != "accepted" {
		t.Errorf("eval try accept output = %q, want %q", output, "accepted")
	}
}

func TestEvalTryReject(t *testing.T) {
	var buf bytes.Buffer
	e := New()
	e.stdout = &buf

	tokens, _ := lexer.Tokenize("try {\nX = 0\n} evaluate with {\nX == 1\n} accept {\nprint \"accepted\"\n} fail {\nprint \"rejected\"\n} upto 1")
	script, _ := parser.Parse(tokens)
	err := e.EvalScript(script)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := strings.TrimSpace(buf.String())
	if output != "rejected" {
		t.Errorf("eval try reject output = %q, want %q", output, "rejected")
	}
}

func TestCommandSubst(t *testing.T) {
	result, err := evalExpr(t, `$(echo hello)`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "hello" {
		t.Errorf("$(echo hello) = %q, want %q", result, "hello")
	}
}

func TestCommandSubstWithVarInterp(t *testing.T) {
	tokens, err := lexer.Tokenize(`NAME = "world"
GREETING = $(echo "hello ${NAME}")`)
	if err != nil {
		t.Fatalf("tokenize error: %v", err)
	}
	script, err := parser.Parse(tokens)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	e := New()
	err = e.EvalScript(script)
	if err != nil {
		t.Fatalf("eval error: %v", err)
	}

	greeting, ok := e.current.Get("GREETING")
	if !ok {
		t.Fatal("GREETING not set")
	}
	if greeting != "hello world" {
		t.Errorf("GREETING = %q, want %q", greeting, "hello world")
	}
}

func TestDoAgentCall(t *testing.T) {
	if _, err := exec.LookPath("echo"); err != nil {
		t.Skip("echo not found")
	}
	e := New()

	tokens, _ := lexer.Tokenize(`do "hello" with subagent echo`)
	script, _ := parser.Parse(tokens)
	err := e.EvalScript(script)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	stdout := e.GetVar("stdout")
	if stdout == nil {
		t.Fatal("stdout should be set")
	}
}
