package eval

import (
	"context"
	"fmt"
	"io"
	"math"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"github.com/agentic-coding/ash/ast"
	"github.com/agentic-coding/ash/compact"
	"github.com/agentic-coding/ash/engine"
	"github.com/agentic-coding/ash/executor"
	"github.com/agentic-coding/ash/lexer"
	"github.com/agentic-coding/ash/parser"
)

type ExitError struct {
	Code   int
	Stdout string
	Stderr string
}

func (e *ExitError) Error() string {
	return fmt.Sprintf("exit code %d", e.Code)
}

type SignalKind int

const (
	SignalReturn SignalKind = iota
	SignalBreak
	SignalContinue
)

type FlowSignal struct {
	Kind  SignalKind
	Value Value
}

type Evaluator struct {
	globals  *Scope
	current  *Scope
	stdout   io.Writer
	stderr   io.Writer
	executor *executor.Executor
	ctx      context.Context
	signal   *FlowSignal

	shebangEngine string
	compactCfg    *compact.Config
	agentDir      string
	engines       map[string]engine.Engine
	bgWg          sync.WaitGroup
	bgMu          sync.Mutex
}

func New() *Evaluator {
	return &Evaluator{
		globals:    NewGlobalScope(),
		current:    NewGlobalScope(),
		stdout:     os.Stdout,
		stderr:     os.Stderr,
		executor:   executor.New(),
		ctx:        context.Background(),
		compactCfg: compact.NewConfig(),
		engines:    make(map[string]engine.Engine),
	}
}

func (e *Evaluator) EvalScript(script *ast.Script) error {
	if script == nil {
		return nil
	}

	// Apply global compact header config
	if script.Compact != nil {
		raw := script.Compact.Raw
		if raw != nil {
			cfg := compact.FromHeader(raw)
			e.compactCfg = cfg
		}
	}

	// Log shebang engine declaration
	if script.Shebang != nil {
		sh := script.Shebang
		e.shebangEngine = sh.Engine
		eng, ok := engine.Get(sh.Engine)
		if !ok {
			fmt.Fprintf(e.stderr, "warning: engine %q not registered\n", sh.Engine)
		} else if sh.Version != "" {
			// Best-effort version check
			r := eng.Run(e.ctx, "--version", &engine.Options{})
			if r.ExitCode == 0 && r.Stdout != "" {
				if !strings.Contains(r.Stdout, sh.Version) {
					fmt.Fprintf(e.stderr, "warning: engine %q version mismatch (declared %q, installed %s)\n",
						sh.Engine, sh.Version, strings.TrimSpace(r.Stdout))
				}
			}
		}
	}

	for _, stmt := range script.Statements {
		_, err := e.evalStatement(stmt)
		if err != nil {
			if exitErr, ok := err.(*ExitError); ok {
				return exitErr
			}
			return err
		}
		if e.signal != nil {
			break
		}
	}
	return nil
}

func (e *Evaluator) EvalStatements(stmts []ast.Node) error {
	for _, stmt := range stmts {
		_, err := e.evalStatement(stmt)
		if err != nil {
			return err
		}
		if e.signal != nil {
			break
		}
	}
	return nil
}

func (e *Evaluator) evalStatement(stmt ast.Node) (Value, error) {
	switch s := stmt.(type) {
	case *ast.VarAssign:
		return e.evalVarAssign(s)
	case *ast.Exec:
		return e.evalExec(s)
	case *ast.Print:
		return e.evalPrint(s)
	case *ast.Exit:
		return e.evalExit(s)
	case *ast.Include:
		return e.evalInclude(s)
	case *ast.Env:
		return e.evalEnv(s)
	case *ast.IfStmt:
		return e.evalIf(s)
	case *ast.ForStmt:
		return e.evalFor(s)
	case *ast.WhileStmt:
		return e.evalWhile(s)
	case *ast.FnDecl:
		return e.evalFnDecl(s)
	case *ast.FnCall:
		return e.evalFnCall(s)
	case *ast.Return:
		return e.evalReturn(s)
	case *ast.Break:
		return nil, e.evalBreak(s)
	case *ast.Continue:
		return nil, e.evalContinue(s)
	case *ast.AgentCall:
		return e.evalDo(s)
	case *ast.DirBlock:
		return e.evalDirBlock(s)
	case *ast.CompactStmt:
		return e.evalCompact(s)
	case *ast.BinaryTry:
		return e.evalBinaryTry(s)
	case *ast.EvalTry:
		return e.evalEvalTry(s)
	case *ast.WaitBlock:
		return e.evalWait(s)
	case *ast.Background:
		return e.evalBackground(s)
	case *ast.Block:
		return e.evalBlock(s)
	default:
		return e.evalExpr(stmt)
	}
}

func (e *Evaluator) evalExpr(node ast.Node) (Value, error) {
	switch n := node.(type) {
	case *ast.BinaryExpr:
		return e.evalBinaryExpr(n)
	case *ast.UnaryExpr:
		return e.evalUnaryExpr(n)
	case *ast.VarRef:
		return e.evalVarRef(n)
	case *ast.StringLiteral:
		return e.evalString(n)
	case *ast.TextBlock:
		return e.evalTextBlock(n)
	case *ast.IntLiteral:
		return n.Value, nil
	case *ast.FloatLiteral:
		return n.Value, nil
	case *ast.BoolLiteral:
		return n.Value, nil
	case *ast.CommandSubst:
		return e.evalCommandSubst(n)
	case *ast.FnCall:
		return e.evalFnCall(n)
	case *ast.FilePath:
		return e.evalFilePath(n)
	case *ast.ArrayLiteral:
		return e.evalArrayLiteral(n)
	case *ast.IndexExpr:
		return e.evalIndexExpr(n)
	case *ast.GroupExpr:
		return e.evalExpr(n.Inner)
	default:
		return nil, fmt.Errorf("unhandled expression type: %T at %s", n, node.Pos())
	}
}

var interpRe = regexp.MustCompile(`\$\{([^}]+)\}|\$\(([^)]+)\)`)

func (e *Evaluator) resolveInterpolations(s string) (string, error) {
	idx := strings.Index(s, "\\$")
	for idx >= 0 {
		s = s[:idx] + "\x00" + s[idx+2:]
		idx = strings.Index(s, "\\$")
	}

	result := interpRe.ReplaceAllStringFunc(s, func(match string) string {
		parts := interpRe.FindStringSubmatch(match)
		if parts == nil {
			return match
		}
		if parts[1] != "" {
			val, ok := e.current.Get(parts[1])
			if !ok {
				return match
			}
			return fmt.Sprintf("%v", val)
		}
		if parts[2] != "" {
			res, err := e.executor.Run(e.ctx, parts[2])
			if err != nil {
				return match
			}
			return res.Stdout
		}
		return match
	})

	result = strings.ReplaceAll(result, "\x00", "$")
	return result, nil
}

func (e *Evaluator) evalString(n *ast.StringLiteral) (Value, error) {
	if len(n.Interps) > 0 {
		var parts []string
		for _, sp := range n.Interps {
			if sp.Var != "" {
				val, ok := e.current.Get(sp.Var)
				if !ok {
					parts = append(parts, sp.Text)
				} else {
					parts = append(parts, fmt.Sprintf("%v", val))
				}
			} else if sp.Cmd != "" {
				res, err := e.executor.Run(e.ctx, sp.Cmd)
				if err != nil {
					parts = append(parts, "")
				} else {
					parts = append(parts, res.Stdout)
				}
			} else {
				parts = append(parts, sp.Text)
			}
		}
		return strings.Join(parts, ""), nil
	}
	return e.resolveInterpolations(n.Value)
}

func (e *Evaluator) evalTextBlock(n *ast.TextBlock) (Value, error) {
	return e.resolveInterpolations(n.Value)
}

func (e *Evaluator) evalFilePath(n *ast.FilePath) (Value, error) {
	pathVal, err := e.evalExpr(n.Path)
	if err != nil {
		return nil, err
	}
	pathStr := fmt.Sprintf("%v", pathVal)
	data, err := os.ReadFile(pathStr)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", pathStr, err)
	}
	resolved, err := e.resolveInterpolations(string(data))
	if err != nil {
		return nil, err
	}
	return resolved, nil
}

func (e *Evaluator) evalBlock(block *ast.Block) (Value, error) {
	if block == nil {
		return nil, nil
	}
	for _, stmt := range block.Statements {
		val, err := e.evalStatement(stmt)
		if err != nil {
			if _, ok := err.(*ExitError); ok {
				return val, err
			}
			return nil, err
		}
		if e.signal != nil {
			return val, nil
		}
	}
	return nil, nil
}

func (e *Evaluator) evalBlockInPlace(b *ast.Block) (Value, error) {
	var result Value
	for _, stmt := range b.Statements {
		val, err := e.evalStatement(stmt)
		if err != nil {
			return val, err
		}
		if val != nil {
			result = val
		}
		if e.signal != nil {
			break
		}
	}
	return result, nil
}

func (e *Evaluator) evalVarAssign(n *ast.VarAssign) (Value, error) {
	val, err := e.evalExpr(n.Value)
	if err != nil {
		return nil, err
	}
	e.current.Set(n.Name, val)
	e.setExitCode(0)
	return val, nil
}

func (e *Evaluator) evalVarRef(n *ast.VarRef) (Value, error) {
	val, ok := e.current.Get(n.Name)
	if !ok {
		return nil, fmt.Errorf("undefined variable: %s at %s", n.Name, n.Pos())
	}
	return val, nil
}

func toFloat(v Value) (float64, bool) {
	switch x := v.(type) {
	case int64:
		return float64(x), true
	case float64:
		return x, true
	default:
		return 0, false
	}
}

func resultIsTruthy(v Value) bool {
	b, ok := toBool(v)
	if !ok {
		return true
	}
	return b
}

func toInt(v Value) (int64, bool) {
	switch x := v.(type) {
	case int64:
		return x, true
	case float64:
		if x == math.Trunc(x) {
			return int64(x), true
		}
		return int64(x), false
	default:
		return 0, false
	}
}

func toString(v Value) (string, bool) {
	switch x := v.(type) {
	case string:
		return x, true
	case int64:
		return strconv.FormatInt(x, 10), true
	case float64:
		return strconv.FormatFloat(x, 'f', -1, 64), true
	case bool:
		if x {
			return "true", true
		}
		return "false", true
	case []Value:
		var elems []string
		for _, e := range x {
			s, _ := toString(e)
			elems = append(elems, s)
		}
		return "[" + strings.Join(elems, ", ") + "]", true
	default:
		return fmt.Sprintf("%v", x), true
	}
}

func toBool(v Value) (bool, bool) {
	switch x := v.(type) {
	case bool:
		return x, true
	case int64:
		return x != 0, true
	case float64:
		return x != 0, true
	case string:
		return x != "" && x != "0" && x != "false", true
	case []Value:
		return len(x) > 0, true
	default:
		return false, false
	}
}

func (e *Evaluator) evalBinaryExpr(n *ast.BinaryExpr) (Value, error) {
	left, err := e.evalExpr(n.Left)
	if err != nil {
		return nil, err
	}
	right, err := e.evalExpr(n.Right)
	if err != nil {
		return nil, err
	}

	switch n.Op {
	case "==":
		return valuesEqual(left, right), nil
	case "!=":
		return !valuesEqual(left, right), nil
	case ">", "<", ">=", "<=":
		return e.evalComparison(n.Op, left, right)
	case "+":
		return e.evalAdd(left, right)
	case "-":
		return e.evalSub(left, right)
	case "*":
		return e.evalMul(left, right)
	case "/":
		return e.evalDiv(left, right)
	case "and":
		lv, lok := toBool(left)
		rv, rok := toBool(right)
		if !lok || !rok {
			return nil, fmt.Errorf("and requires boolean operands at %s", n.Pos())
		}
		return lv && rv, nil
	case "or":
		lv, lok := toBool(left)
		rv, rok := toBool(right)
		if !lok || !rok {
			return nil, fmt.Errorf("or requires boolean operands at %s", n.Pos())
		}
		return lv || rv, nil
	default:
		return nil, fmt.Errorf("unknown operator %q at %s", n.Op, n.Pos())
	}
}

func valuesEqual(a, b Value) bool {
	// Array comparison
	aa, aok := a.([]Value)
	ba, bok := b.([]Value)
	if aok && bok {
		if len(aa) != len(ba) {
			return false
		}
		for i := range aa {
			if !valuesEqual(aa[i], ba[i]) {
				return false
			}
		}
		return true
	}
	if aok || bok {
		return false
	}

	af, aok := toFloat(a)
	bf, bok := toFloat(b)
	if aok && bok {
		return af == bf
	}
	as, aok2 := toString(a)
	bs, bok2 := toString(b)
	if aok2 && bok2 {
		return as == bs
	}
	return fmt.Sprintf("%v", a) == fmt.Sprintf("%v", b)
}

func (e *Evaluator) evalComparison(op string, left, right Value) (Value, error) {
	af, aok := toFloat(left)
	bf, bok := toFloat(right)
	if aok && bok {
		switch op {
		case ">":
			return af > bf, nil
		case "<":
			return af < bf, nil
		case ">=":
			return af >= bf, nil
		case "<=":
			return af <= bf, nil
		}
	}

	as, aok2 := toString(left)
	bs, bok2 := toString(right)
	if aok2 && bok2 {
		switch op {
		case ">":
			return as > bs, nil
		case "<":
			return as < bs, nil
		case ">=":
			return as >= bs, nil
		case "<=":
			return as <= bs, nil
		}
	}

	return nil, fmt.Errorf("comparison %s not supported between %T and %T", op, left, right)
}

func (e *Evaluator) evalAdd(left, right Value) (Value, error) {
	// Array concatenation: [1, 2] + [3, 4]
	la, lok := left.([]Value)
	ra, rok := right.([]Value)
	if lok && rok {
		return append(la, ra...), nil
	}
	if lok {
		return nil, fmt.Errorf("+ not supported between array and %T", right)
	}
	if rok {
		return nil, fmt.Errorf("+ not supported between %T and array", left)
	}

	ai, aok := toInt(left)
	bi, bok := toInt(right)
	if aok && bok {
		return ai + bi, nil
	}

	af, aokf := toFloat(left)
	bf, bokf := toFloat(right)
	if aokf && bokf {
		return af + bf, nil
	}

	as, aok2 := toString(left)
	bs, bok2 := toString(right)
	if aok2 && bok2 {
		return as + bs, nil
	}

	return nil, fmt.Errorf("+ not supported between %T and %T", left, right)
}

func (e *Evaluator) evalSub(left, right Value) (Value, error) {
	ai, aok := toInt(left)
	bi, bok := toInt(right)
	if aok && bok {
		return ai - bi, nil
	}

	af, aokf := toFloat(left)
	bf, bokf := toFloat(right)
	if aokf && bokf {
		return af - bf, nil
	}
	return nil, fmt.Errorf("- not supported between %T and %T", left, right)
}

func (e *Evaluator) evalMul(left, right Value) (Value, error) {
	ai, aok := toInt(left)
	bi, bok := toInt(right)
	if aok && bok {
		return ai * bi, nil
	}

	af, aokf := toFloat(left)
	bf, bokf := toFloat(right)
	if aokf && bokf {
		return af * bf, nil
	}
	return nil, fmt.Errorf("* not supported between %T and %T", left, right)
}

func (e *Evaluator) evalDiv(left, right Value) (Value, error) {
	af, aokf := toFloat(left)
	bf, bokf := toFloat(right)
	if aokf && bokf {
		if bf == 0 {
			return nil, fmt.Errorf("division by zero")
		}
		return af / bf, nil
	}
	return nil, fmt.Errorf("/ not supported between %T and %T", left, right)
}

func (e *Evaluator) evalUnaryExpr(n *ast.UnaryExpr) (Value, error) {
	right, err := e.evalExpr(n.Right)
	if err != nil {
		return nil, err
	}
	switch n.Op {
	case "not":
		bv, ok := toBool(right)
		if !ok {
			return nil, fmt.Errorf("not requires boolean operand at %s", n.Pos())
		}
		return !bv, nil
	case "-":
		fv, ok := toFloat(right)
		if !ok {
			return nil, fmt.Errorf("- requires numeric operand at %s", n.Pos())
		}
		return -fv, nil
	default:
		return nil, fmt.Errorf("unknown unary operator %q at %s", n.Op, n.Pos())
	}
}

func (e *Evaluator) evalArrayLiteral(n *ast.ArrayLiteral) (Value, error) {
	var result []Value
	for _, elem := range n.Elements {
		val, err := e.evalExpr(elem)
		if err != nil {
			return nil, err
		}
		result = append(result, val)
	}
	return result, nil
}

func (e *Evaluator) evalIndexExpr(n *ast.IndexExpr) (Value, error) {
	obj, err := e.evalExpr(n.Object)
	if err != nil {
		return nil, err
	}
	idxVal, err := e.evalExpr(n.Index)
	if err != nil {
		return nil, err
	}

	arr, ok := obj.([]Value)
	if !ok {
		return nil, fmt.Errorf("cannot index non-array value at %s", n.Pos())
	}

	idx, ok := toInt(idxVal)
	if !ok {
		return nil, fmt.Errorf("array index must be an integer at %s", n.Pos())
	}

	if idx < 0 || int(idx) >= len(arr) {
		return nil, fmt.Errorf("array index %d out of bounds (length %d) at %s", idx, len(arr), n.Pos())
	}

	return arr[idx], nil
}

func (e *Evaluator) evalCommandSubst(n *ast.CommandSubst) (Value, error) {
	cmd, err := e.resolveInterpolations(n.Cmd)
	if err != nil {
		return "", nil
	}
	res, err := e.executor.Run(e.ctx, cmd)
	if err != nil {
		return "", nil
	}
	return strings.TrimSpace(res.Stdout), nil
}

func (e *Evaluator) evalExec(n *ast.Exec) (Value, error) {
	cmdVal, err := e.evalExpr(n.Cmd)
	if err != nil {
		return nil, err
	}
	cmdStr := fmt.Sprintf("%v", cmdVal)

	res, err := e.executor.Run(e.ctx, cmdStr)
	e.setRuntimeVars(res)
	if res.ExitCode != 0 {
		return "", &ExitError{Code: res.ExitCode, Stdout: res.Stdout, Stderr: res.Stderr}
	}
	return res.Stdout, nil
}

func (e *Evaluator) setRuntimeVars(res executor.ExecResult) {
	e.current.SetLocal("?", int64(res.ExitCode))
	e.current.SetLocal("stdout", res.Stdout)
	e.current.SetLocal("stderr", res.Stderr)
}

func (e *Evaluator) evalPrint(n *ast.Print) (Value, error) {
	msg, err := e.evalExpr(n.Message)
	if err != nil {
		return nil, err
	}
	_, printErr := fmt.Fprintln(e.stdout, msg)
	if printErr != nil {
		return nil, printErr
	}
	e.setExitCode(0)
	return msg, nil
}

func (e *Evaluator) evalExit(n *ast.Exit) (Value, error) {
	codeVal, err := e.evalExpr(n.Code)
	if err != nil {
		return nil, err
	}
	code := int64(0)
	switch v := codeVal.(type) {
	case int64:
		code = v
	case float64:
		code = int64(v)
	case string:
		c, parseErr := strconv.ParseInt(v, 10, 64)
		if parseErr == nil {
			code = c
		}
	}
	e.setExitCode(int(code))
	return nil, &ExitError{Code: int(code)}
}

func (e *Evaluator) evalInclude(n *ast.Include) (Value, error) {
	pathVal, err := e.evalExpr(n.Path)
	if err != nil {
		return nil, err
	}
	path := fmt.Sprintf("%v", pathVal)

	src, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("include: %w at %s", err, n.Pos())
	}

	tokens, tokenErr := lexer.Tokenize(string(src))
	if tokenErr != nil {
		return nil, fmt.Errorf("include %q: %w at %s", path, tokenErr, n.Pos())
	}

	included, parseErr := parser.Parse(tokens)
	if parseErr != nil {
		return nil, fmt.Errorf("include %q: %w at %s", path, parseErr, n.Pos())
	}

	parentScope := e.current
	for _, stmt := range included.Statements {
		if _, stmtErr := e.evalStatement(stmt); stmtErr != nil {
			e.current = parentScope
			return nil, stmtErr
		}
	}

	e.setExitCode(0)
	return nil, nil
}

func (e *Evaluator) evalEnv(n *ast.Env) (Value, error) {
	val := os.Getenv(n.Key)
	e.setExitCode(0)
	return val, nil
}

func (e *Evaluator) evalIf(n *ast.IfStmt) (Value, error) {
	cond, err := e.evalExpr(n.Cond)
	if err != nil {
		return nil, err
	}

	condBool, ok := toBool(cond)
	if !ok {
		condBool = isTruthy(cond)
	}

	if condBool {
		return e.evalBlock(n.Body)
	}

	for _, elseif := range n.ElseIfs {
		eiCond, eiErr := e.evalExpr(elseif.Cond)
		if eiErr != nil {
			return nil, eiErr
		}
		eiBool, ok := toBool(eiCond)
		if !ok {
			eiBool = isTruthy(eiCond)
		}
		if eiBool {
			return e.evalBlock(elseif.Body)
		}
	}

	if n.Else != nil {
		return e.evalBlock(n.Else)
	}

	return nil, nil
}

func isTruthy(v Value) bool {
	switch x := v.(type) {
	case bool:
		return x
	case int64:
		return x != 0
	case float64:
		return x != 0
	case string:
		return x != "" && x != "0" && x != "false"
	case []Value:
		return len(x) > 0
	case nil:
		return false
	default:
		return true
	}
}

func (e *Evaluator) evalFor(n *ast.ForStmt) (Value, error) {
	listVal, err := e.evalExpr(n.List)
	if err != nil {
		return nil, err
	}

	var items []string

	switch list := listVal.(type) {
	case []Value:
		for _, item := range list {
			s, _ := toString(item)
			items = append(items, s)
		}
	default:
		listStr := fmt.Sprintf("%v", listVal)
		items = strings.Split(listStr, "\n")
		if len(items) == 1 && items[0] == "" {
			items = nil
		}
	}

	for _, item := range items {
		e.current.Set(n.Var, item)
		_, loopErr := e.evalBlock(n.Body)
		if loopErr != nil {
			if exitErr, ok := loopErr.(*ExitError); ok {
				return nil, exitErr
			}
			return nil, loopErr
		}
		if e.signal != nil {
			if e.signal.Kind == SignalBreak {
				e.signal = nil
				break
			}
			if e.signal.Kind == SignalContinue {
				e.signal = nil
				continue
			}
			break
		}
	}

	return nil, nil
}

func (e *Evaluator) evalWhile(n *ast.WhileStmt) (Value, error) {
	for {
		cond, err := e.evalExpr(n.Cond)
		if err != nil {
			return nil, err
		}
		condBool, ok := toBool(cond)
		if !ok {
			condBool = isTruthy(cond)
		}
		if !condBool {
			break
		}

		_, loopErr := e.evalBlock(n.Body)
		if loopErr != nil {
			if exitErr, ok := loopErr.(*ExitError); ok {
				return nil, exitErr
			}
			return nil, loopErr
		}
		if e.signal != nil {
			if e.signal.Kind == SignalBreak {
				e.signal = nil
				break
			}
			if e.signal.Kind == SignalContinue {
				e.signal = nil
				continue
			}
			break
		}
	}
	return nil, nil
}

func (e *Evaluator) evalFnDecl(n *ast.FnDecl) (Value, error) {
	e.current.Set(n.Name, n)
	e.setExitCode(0)
	return nil, nil
}

func (e *Evaluator) evalFnCall(n *ast.FnCall) (Value, error) {
	// Built-in functions
	if n.Name == "len" {
		if len(n.Args) != 1 {
			return nil, fmt.Errorf("len() requires 1 argument, got %d at %s", len(n.Args), n.Pos())
		}
		arg, err := e.evalExpr(n.Args[0])
		if err != nil {
			return nil, err
		}
		switch v := arg.(type) {
		case []Value:
			return int64(len(v)), nil
		case string:
			return int64(len(v)), nil
		default:
			return nil, fmt.Errorf("len() not supported for %T at %s", arg, n.Pos())
		}
	}

	val, ok := e.current.Get(n.Name)
	if !ok {
		return nil, fmt.Errorf("undefined function: %s at %s", n.Name, n.Pos())
	}

	fn, ok := val.(*ast.FnDecl)
	if !ok {
		return nil, fmt.Errorf("%s is not a function at %s", n.Name, n.Pos())
	}

	e.pushScope()
	if len(n.Args) != len(fn.Params) {
		e.popScope()
		return nil, fmt.Errorf("function %s expects %d args, got %d at %s", n.Name, len(fn.Params), len(n.Args), n.Pos())
	}

	for i, param := range fn.Params {
		argVal, argErr := e.evalExpr(n.Args[i])
		if argErr != nil {
			e.popScope()
			return nil, argErr
		}
		e.current.SetLocal(param, argVal)
	}

	var result Value
	_, bodyErr := e.evalBlock(fn.Body)
	if bodyErr != nil {
		e.popScope()
		if exitErr, ok := bodyErr.(*ExitError); ok && exitErr.Code == 0 {
			return nil, nil
		}
		return nil, bodyErr
	}

	if e.signal != nil && e.signal.Kind == SignalReturn {
		result = e.signal.Value
		e.signal = nil
	}

	e.popScope()
	e.setExitCode(0)
	return result, nil
}

func (e *Evaluator) evalReturn(n *ast.Return) (Value, error) {
	if n.Value != nil {
		val, err := e.evalExpr(n.Value)
		if err != nil {
			return nil, err
		}
		e.current.SetLocal("?", int64(0))
		e.signal = &FlowSignal{Kind: SignalReturn, Value: val}
		return val, nil
	}
	e.current.SetLocal("?", int64(0))
	e.signal = &FlowSignal{Kind: SignalReturn}
	return nil, nil
}

func (e *Evaluator) evalBreak(n *ast.Break) error {
	e.signal = &FlowSignal{Kind: SignalBreak}
	return nil
}

func (e *Evaluator) evalContinue(n *ast.Continue) error {
	e.signal = &FlowSignal{Kind: SignalContinue}
	return nil
}

func (e *Evaluator) evalDo(n *ast.AgentCall) (Value, error) {
	prompt, err := e.evalExpr(n.Prompt)
	if err != nil {
		return nil, err
	}

	promptStr := fmt.Sprintf("%v", prompt)
	promptStr, err = e.resolveInterpolations(promptStr)
	if err != nil {
		return nil, err
	}

	modelStr := ""
	if n.Model != nil {
		modelVal, err := e.evalExpr(n.Model)
		if err == nil {
			modelStr = fmt.Sprintf("%v", modelVal)
		}
	}

	dirStr := ""
	if n.Dir != nil {
		dirVal, err := e.evalExpr(n.Dir)
		if err != nil {
			return nil, err
		}
		dirStr = fmt.Sprintf("%v", dirVal)
		if fi, statErr := os.Stat(dirStr); statErr != nil {
			return nil, fmt.Errorf("working directory does not exist: %s at %s", dirStr, n.Pos())
		} else if !fi.IsDir() {
			return nil, fmt.Errorf("not a directory: %s at %s", dirStr, n.Pos())
		}
	} else if e.agentDir != "" {
		dirStr = e.agentDir
	}

	// Engine chosen by shebang header; subagent is a capability passed to the engine
	engineName := e.shebangEngine
	if engineName == "" {
		engineName = n.Subagent
	}

	eng, ok := engine.Get(engineName)
	var r *engine.Result
	if ok {
		r = eng.Run(e.ctx, promptStr, &engine.Options{Model: modelStr, Subagent: n.Subagent, Dir: dirStr})
	} else {
		args := []string{engineName, promptStr}
		if modelStr != "" {
			args = append(args, "--model", modelStr)
		}
		cmd := exec.CommandContext(e.ctx, args[0], args[1:]...)
		if dirStr != "" {
			cmd.Dir = dirStr
		}
		var stdout, stderr strings.Builder
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr
		execErr := cmd.Run()
		r = &engine.Result{
			Stdout: strings.TrimSuffix(stdout.String(), "\n"),
			Stderr: strings.TrimSuffix(stderr.String(), "\n"),
		}
		if execErr != nil {
			if exitErr, ok := execErr.(*exec.ExitError); ok {
				r.ExitCode = exitErr.ExitCode()
			} else {
				r.ExitCode = 1
			}
		}
	}

	e.current.Set("stdout", r.Stdout)
	e.current.Set("stderr", r.Stderr)
	e.current.Set("?", int64(r.ExitCode))

	// Per-agent compact subclause — overrides global config
	if n.Compact != nil {
		compactVal, err := e.evalExpr(n.Compact)
		if err == nil {
			if s, ok := toString(compactVal); ok {
				if d := compact.ParseDirective(s); d != nil {
					msg := e.compactCfg.ApplyDirective(d)
					fmt.Fprintln(e.stderr, msg)
				}
			}
		}
	}

	if r.ExitCode != 0 {
		return r.Stdout, &ExitError{Code: r.ExitCode, Stdout: r.Stdout, Stderr: r.Stderr}
	}

	return r.Stdout, nil
}

func (e *Evaluator) evalDirBlock(n *ast.DirBlock) (Value, error) {
	dirVal, err := e.evalExpr(n.Dir)
	if err != nil {
		return nil, err
	}
	dirStr := fmt.Sprintf("%v", dirVal)

	if fi, statErr := os.Stat(dirStr); statErr != nil {
		return nil, fmt.Errorf("working directory does not exist: %s at %s", dirStr, n.Pos())
	} else if !fi.IsDir() {
		return nil, fmt.Errorf("not a directory: %s at %s", dirStr, n.Pos())
	}

	saved := e.agentDir
	e.agentDir = dirStr
	_, bodyErr := e.evalBlock(n.Body)
	e.agentDir = saved

	return nil, bodyErr
}

func (e *Evaluator) evalCompact(n *ast.CompactStmt) (Value, error) {
	if n.Arg == nil {
		return int64(0), nil
	}

	val, err := e.evalExpr(n.Arg)
	if err != nil {
		return nil, err
	}

	if s, ok := toString(val); ok {
		if d := compact.ParseDirective(s); d != nil {
			msg := e.compactCfg.ApplyDirective(d)
			fmt.Fprintln(e.stderr, msg)
			return int64(0), nil
		}
	}

	if b, ok := toBool(val); ok {
		if b {
			e.compactCfg.Mode = compact.ModeOn
		} else {
			e.compactCfg.Mode = compact.ModeOff
		}
	}

	return int64(0), nil
}

func (e *Evaluator) evalBinaryTry(n *ast.BinaryTry) (Value, error) {
	max := n.Max
	if max <= 0 {
		max = 1
	}

	e.pushScope()
	defer e.popScope()

	for attempt := 0; attempt < max; attempt++ {
		var result Value
		var bodyErr error

		if n.Body != nil {
			for _, stmt := range n.Body.Statements {
				result, bodyErr = e.evalStatement(stmt)
				if bodyErr != nil {
					break
				}
			}
		}

		if bodyErr == nil {
			b, _ := toBool(result)
			if b {
				e.current.Set("?", int64(0))
				return result, nil
			}
		}

		e.current.Set("?", int64(1))
	}

	if n.Fail != nil {
		return e.evalBlockInPlace(n.Fail)
	}
	return int64(0), nil
}

func (e *Evaluator) evalEvalTry(n *ast.EvalTry) (Value, error) {
	max := n.Max
	if max <= 0 {
		max = 1
	}

	e.pushScope()
	defer e.popScope()

	for attempt := 0; attempt < max; attempt++ {
		if n.Body != nil {
			for _, stmt := range n.Body.Statements {
				_, err := e.evalStatement(stmt)
				if err != nil {
					e.current.Set("error", err.Error())
					e.current.Set("?", int64(1))
					goto fail
				}
			}
		}

		if n.Eval == nil {
			return int64(0), nil
		}

		prevExit, _ := e.current.Get("?")

		// Capture evaluator stdout for ${report}
		savedStdout := e.stdout
		var reportBuf strings.Builder
		e.stdout = io.MultiWriter(savedStdout, &reportBuf)

		val, evalErr := e.evalBlockInPlace(n.Eval)

		e.stdout = savedStdout
		e.current.Set("report", reportBuf.String())

		afterExit, _ := e.current.Get("?")
		exitCode := int64(0)
		if code, ok := afterExit.(int64); ok {
			exitCode = code
		}

		if evalErr != nil {
			goto fail
		}

		// Determine routing: use exit code if changed, else use value truthiness
		isAccept := false
		if prevExit != afterExit {
			isAccept = exitCode == 0
		} else {
			b, _ := toBool(val)
			isAccept = b
		}

		if isAccept {
			if n.Partial != nil {
				e.evalBlockInPlace(n.Partial)
			}
			if n.Accept != nil {
				return e.evalBlockInPlace(n.Accept)
			}
			return int64(0), nil
		}

		if n.Partial != nil {
			e.evalBlockInPlace(n.Partial)
		}
	}

fail:
	if n.Fail != nil {
		return e.evalBlockInPlace(n.Fail)
	}
	return int64(0), nil
}

func (e *Evaluator) evalWait(n *ast.WaitBlock) (Value, error) {
	if n.Body != nil {
		var wg sync.WaitGroup
		for _, stmt := range n.Body.Statements {
			wg.Add(1)
			s := stmt
			go func() {
				defer wg.Done()
				e.pushScope()
				defer e.popScope()
				_, _ = e.evalStatement(s)
			}()
		}
		wg.Wait()
	}
	e.bgWg.Wait()
	return int64(0), nil
}

func (e *Evaluator) evalBackground(n *ast.Background) (Value, error) {
	e.bgWg.Add(1)
	go func() {
		defer e.bgWg.Done()
		_, _ = e.evalStatement(n.Stmt)
	}()
	return int64(0), nil
}

func (e *Evaluator) pushScope() {
	e.current = NewScope(e.current)
}

func (e *Evaluator) popScope() {
	if e.current.parent != nil {
		e.current = e.current.parent
	}
}

func (e *Evaluator) setExitCode(code int) {
	e.current.SetLocal("?", int64(code))
}

func (e *Evaluator) SetOutput(w io.Writer) {
	e.stdout = w
}

func (e *Evaluator) SetError(w io.Writer) {
	e.stderr = w
}

func (e *Evaluator) GetVar(name string) Value {
	val, _ := e.current.Get(name)
	return val
}

func (e *Evaluator) EvalStmt(stmt ast.Node) (Value, error) {
	return e.evalStatement(stmt)
}

func (e *Evaluator) EvalScriptWithExit(s *ast.Script) int {
	err := e.EvalScript(s)
	if err != nil {
		if exit, ok := err.(*ExitError); ok {
			return exit.Code
		}
		fmt.Fprintf(e.stderr, "error: %v\n", err)
		return 3
	}
	return 0
}
